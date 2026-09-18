package auth

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"

	"github.com/golang-jwt/jwt/v5"
)

type OIDCIDTokenClaims struct {
	jwt.RegisteredClaims
	Nonce             string `json:"nonce,omitempty"`
	Name              string `json:"name,omitempty"`
	PreferredUsername string `json:"preferred_username,omitempty"`
	Picture           string `json:"picture,omitempty"`
	UpdatedAt         int64  `json:"updated_at,omitempty"`
	Email             string `json:"email,omitempty"`
	EmailVerified     bool   `json:"email_verified,omitempty"`
	// AD FS and similar providers emit these instead of the standard
	// profile claims; their userinfo endpoints often return only sub.
	UPN        string `json:"upn,omitempty"`
	UniqueName string `json:"unique_name,omitempty"`
	GivenName  string `json:"given_name,omitempty"`
	FamilyName string `json:"family_name,omitempty"`
}

type JWKSet struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func SignOIDCIDToken(privateKeyRaw string, claims *OIDCIDTokenClaims) (string, error) {
	key, err := parseRSAPrivateKey(privateKeyRaw)
	if err != nil {
		return "", err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = oidcSigningKeyID(&key.PublicKey)
	return token.SignedString(key)
}

func OIDCJWKSet(privateKeyRaw string) (*JWKSet, error) {
	key, err := parseRSAPrivateKey(privateKeyRaw)
	if err != nil {
		return nil, err
	}

	kid := oidcSigningKeyID(&key.PublicKey)
	return &JWKSet{Keys: []JWK{buildJWK(&key.PublicKey, kid)}}, nil
}

func parseRSAPrivateKey(privateKeyRaw string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privateKeyRaw))
	if block == nil {
		return nil, fmt.Errorf("invalid OIDC signing key PEM")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("invalid OIDC signing key: %w", err)
	}

	return key, nil
}

func oidcSigningKeyID(key *rsa.PublicKey) string {
	der, _ := x509.MarshalPKIXPublicKey(key)
	sum := sha256.Sum256(der)
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func buildJWK(key *rsa.PublicKey, kid string) JWK {
	return JWK{
		Kty: "RSA",
		Use: "sig",
		Alg: "RS256",
		Kid: kid,
		N:   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}
}
