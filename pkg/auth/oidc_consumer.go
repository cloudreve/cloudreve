package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrOIDCDiscoveryFailed = errors.New("OIDC discovery failed")
	ErrOIDCKeyNotFound     = errors.New("no matching key in JWKS")
	ErrOIDCInvalidToken    = errors.New("invalid OIDC token")
)

// OIDCDiscovery is the subset of the provider metadata document Cloudreve
// needs to consume an external OIDC identity provider.
type OIDCDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

// OIDCTokenResponse is the token endpoint payload.
type OIDCTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token"`
}

// OIDCUserInfo is the userinfo endpoint payload subset.
type OIDCUserInfo struct {
	Sub               string `json:"sub"`
	Name              string `json:"name,omitempty"`
	PreferredUsername string `json:"preferred_username,omitempty"`
	Picture           string `json:"picture,omitempty"`
	Email             string `json:"email,omitempty"`
	EmailVerified     *bool  `json:"email_verified,omitempty"`
	// AD FS-style alternates, mirrored from the ID-token claims.
	UPN        string `json:"upn,omitempty"`
	UniqueName string `json:"unique_name,omitempty"`
	GivenName  string `json:"given_name,omitempty"`
	FamilyName string `json:"family_name,omitempty"`
}

// Validate checks the discovery document has the endpoints the code flow requires.
func (d *OIDCDiscovery) Validate() error {
	if d.Issuer == "" || d.AuthorizationEndpoint == "" || d.TokenEndpoint == "" || d.JWKSURI == "" {
		return fmt.Errorf("incomplete discovery document: %w", ErrOIDCDiscoveryFailed)
	}
	return nil
}

// VerifyOIDCIDToken validates signature, issuer, audience, expiry and nonce of
// an ID token against the provider's JWKS. Returns ErrOIDCKeyNotFound when the
// signing key is absent so callers can refetch JWKS once for key rotation.
func VerifyOIDCIDToken(idToken, issuer, clientID, nonce string, jwks *JWKSet) (*OIDCIDTokenClaims, error) {
	parser := jwt.NewParser()
	unverified, _, err := parser.ParseUnverified(idToken, &OIDCIDTokenClaims{})
	if err != nil {
		return nil, fmt.Errorf("malformed id_token: %w", err)
	}

	kid, _ := unverified.Header["kid"].(string)
	key, err := findJWK(jwks, kid)
	if err != nil {
		return nil, err
	}

	claims := &OIDCIDTokenClaims{}
	_, err = jwt.ParseWithClaims(idToken, claims, func(t *jwt.Token) (any, error) {
		return key, nil
	},
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(issuer),
		jwt.WithAudience(clientID),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOIDCInvalidToken, err)
	}

	if claims.Nonce != nonce {
		return nil, fmt.Errorf("nonce mismatch: %w", ErrOIDCInvalidToken)
	}

	return claims, nil
}

// findJWK selects the signing key by kid. A token without kid is only matched
// when the provider publishes a single key.
func findJWK(jwks *JWKSet, kid string) (*rsa.PublicKey, error) {
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" {
			continue
		}
		if k.Kid == kid || (kid == "" && len(jwks.Keys) == 1) {
			return jwkToRSAPublicKey(k)
		}
	}

	return nil, ErrOIDCKeyNotFound
}

func jwkToRSAPublicKey(jwk JWK) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("invalid JWK modulus: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("invalid JWK exponent: %w", err)
	}

	e := new(big.Int).SetBytes(eBytes).Int64()
	if e < 3 || e > int64(1<<31-1) {
		return nil, fmt.Errorf("invalid JWK exponent value")
	}

	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(e),
	}, nil
}
