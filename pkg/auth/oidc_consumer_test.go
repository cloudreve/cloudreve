package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func testRSAKeyPEM(t *testing.T) string {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
}

func signConsumerToken(t *testing.T, pemKey string, claims *OIDCIDTokenClaims) string {
	token, err := SignOIDCIDToken(pemKey, claims)
	require.NoError(t, err)
	return token
}

func validClaims() *OIDCIDTokenClaims {
	return &OIDCIDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://idp.example.com",
			Audience:  jwt.ClaimStrings{"cloudreve"},
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Nonce: "nonce-1",
		Email: "user@example.com",
	}
}

func TestVerifyOIDCIDToken(t *testing.T) {
	pemKey := testRSAKeyPEM(t)
	jwks, err := OIDCJWKSet(pemKey)
	require.NoError(t, err)

	t.Run("valid token", func(t *testing.T) {
		token := signConsumerToken(t, pemKey, validClaims())
		claims, err := VerifyOIDCIDToken(token, "https://idp.example.com", "cloudreve", "nonce-1", jwks)
		require.NoError(t, err)
		require.Equal(t, "user@example.com", claims.Email)
		require.Equal(t, "user-1", claims.Subject)
	})

	t.Run("wrong issuer", func(t *testing.T) {
		token := signConsumerToken(t, pemKey, validClaims())
		_, err := VerifyOIDCIDToken(token, "https://evil.example.com", "cloudreve", "nonce-1", jwks)
		require.ErrorIs(t, err, ErrOIDCInvalidToken)
	})

	t.Run("wrong audience", func(t *testing.T) {
		token := signConsumerToken(t, pemKey, validClaims())
		_, err := VerifyOIDCIDToken(token, "https://idp.example.com", "other-client", "nonce-1", jwks)
		require.ErrorIs(t, err, ErrOIDCInvalidToken)
	})

	t.Run("nonce mismatch", func(t *testing.T) {
		token := signConsumerToken(t, pemKey, validClaims())
		_, err := VerifyOIDCIDToken(token, "https://idp.example.com", "cloudreve", "other-nonce", jwks)
		require.ErrorIs(t, err, ErrOIDCInvalidToken)
	})

	t.Run("expired token", func(t *testing.T) {
		claims := validClaims()
		claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Hour))
		token := signConsumerToken(t, pemKey, claims)
		_, err := VerifyOIDCIDToken(token, "https://idp.example.com", "cloudreve", "nonce-1", jwks)
		require.ErrorIs(t, err, ErrOIDCInvalidToken)
	})

	t.Run("forged signature", func(t *testing.T) {
		otherPEM := testRSAKeyPEM(t)
		token := signConsumerToken(t, otherPEM, validClaims())
		_, err := VerifyOIDCIDToken(token, "https://idp.example.com", "cloudreve", "nonce-1", jwks)
		// Foreign key produces a different kid -> key not found in JWKS.
		require.ErrorIs(t, err, ErrOIDCKeyNotFound)
	})

	t.Run("malformed token", func(t *testing.T) {
		_, err := VerifyOIDCIDToken("not-a-jwt", "https://idp.example.com", "cloudreve", "nonce-1", jwks)
		require.Error(t, err)
	})

	t.Run("adfs profile claims", func(t *testing.T) {
		claims := validClaims()
		claims.Email = ""
		claims.UPN = "user@corp.example.com"
		claims.UniqueName = `CORP\user`
		claims.GivenName = "Jane"
		claims.FamilyName = "Doe"
		claims.Name = "Jane Doe"
		token := signConsumerToken(t, pemKey, claims)
		parsed, err := VerifyOIDCIDToken(token, "https://idp.example.com", "cloudreve", "nonce-1", jwks)
		require.NoError(t, err)
		require.Equal(t, "user@corp.example.com", parsed.UPN)
		require.Equal(t, `CORP\user`, parsed.UniqueName)
		require.Equal(t, "Jane", parsed.GivenName)
		require.Equal(t, "Doe", parsed.FamilyName)
	})
}

func TestOIDCDiscoveryValidate(t *testing.T) {
	require.NoError(t, (&OIDCDiscovery{
		Issuer:                "https://idp.example.com",
		AuthorizationEndpoint: "https://idp.example.com/authorize",
		TokenEndpoint:         "https://idp.example.com/token",
		JWKSURI:               "https://idp.example.com/jwks",
	}).Validate())

	require.ErrorIs(t, (&OIDCDiscovery{Issuer: "https://idp.example.com"}).Validate(), ErrOIDCDiscoveryFailed)
}
