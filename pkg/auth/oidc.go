package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/golang-jwt/jwt/v5"
)

const OIDCSigningPrivateKeySetting = "oidc_signing_private_key"

type OIDCIDTokenClaims struct {
	jwt.RegisteredClaims
	Nonce             string `json:"nonce,omitempty"`
	Name              string `json:"name,omitempty"`
	PreferredUsername string `json:"preferred_username,omitempty"`
	Picture           string `json:"picture,omitempty"`
	UpdatedAt         int64  `json:"updated_at,omitempty"`
	Email             string `json:"email,omitempty"`
	EmailVerified     bool   `json:"email_verified,omitempty"`
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

func SignOIDCIDToken(ctx context.Context, settingClient inventory.SettingClient, claims *OIDCIDTokenClaims) (string, error) {
	key, kid, err := loadOrCreateOIDCSigningKey(ctx, settingClient)
	if err != nil {
		return "", err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	return token.SignedString(key)
}

func OIDCJWKSet(ctx context.Context, settingClient inventory.SettingClient) (*JWKSet, error) {
	key, kid, err := loadOrCreateOIDCSigningKey(ctx, settingClient)
	if err != nil {
		return nil, err
	}

	return &JWKSet{Keys: []JWK{buildJWK(&key.PublicKey, kid)}}, nil
}

func loadOrCreateOIDCSigningKey(ctx context.Context, settingClient inventory.SettingClient) (*rsa.PrivateKey, string, error) {
	privateKeyRaw, err := settingClient.Get(ctx, OIDCSigningPrivateKeySetting)
	if err != nil && !ent.IsNotFound(err) {
		return nil, "", fmt.Errorf("failed to load OIDC signing key: %w", err)
	}
	if privateKeyRaw != "" {
		key, err := parseRSAPrivateKey(privateKeyRaw)
		if err != nil {
			return nil, "", err
		}
		return key, oidcSigningKeyID(&key.PublicKey), nil
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate OIDC signing key: %w", err)
	}

	privateKeyRaw = string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}))
	if err := settingClient.Set(ctx, map[string]string{OIDCSigningPrivateKeySetting: privateKeyRaw}); err != nil {
		return nil, "", fmt.Errorf("failed to persist OIDC signing key: %w", err)
	}

	return key, oidcSigningKeyID(&key.PublicKey), nil
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
