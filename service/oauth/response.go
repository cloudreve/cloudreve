package oauth

import (
	"encoding/gob"

	"github.com/cloudreve/Cloudreve/v4/ent"
)

const (
	authCodeKeyPrefix = "oauth_code_"
)

type AppRegistration struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	HomepageURL      string   `json:"homepage_url,omitempty"`
	Icon             string   `json:"icon,omitempty"`
	Description      string   `json:"description,omitempty"`
	ConstentedScopes []string `json:"consented_scopes,omitempty"`
}

type DiscoveryResponse struct {
	Issuer                           string   `json:"issuer"`
	AuthorizationEndpoint            string   `json:"authorization_endpoint"`
	TokenEndpoint                    string   `json:"token_endpoint"`
	UserInfoEndpoint                 string   `json:"userinfo_endpoint"`
	JWKSURI                          string   `json:"jwks_uri"`
	ResponseTypesSupported           []string `json:"response_types_supported"`
	GrantTypesSupported              []string `json:"grant_types_supported"`
	SubjectTypesSupported            []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
	TokenEndpointAuthMethods         []string `json:"token_endpoint_auth_methods_supported"`
	CodeChallengeMethodsSupported    []string `json:"code_challenge_methods_supported"`
	ScopesSupported                  []string `json:"scopes_supported"`
	ClaimsSupported                  []string `json:"claims_supported"`
}

func BuildAppRegistration(app *ent.OAuthClient, grant *ent.OAuthGrant) *AppRegistration {
	res := &AppRegistration{
		ID:          app.GUID,
		Name:        app.Name,
		HomepageURL: app.HomepageURL,
	}

	if app.Props != nil {
		res.Description = app.Props.Description
		res.Icon = app.Props.Icon
	}

	if grant != nil {
		res.ConstentedScopes = grant.Scopes
	}

	return res
}

type GrantResponse struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

// TokenResponse represents the OAuth token response.
type TokenResponse struct {
	AccessToken           string `json:"access_token"`
	TokenType             string `json:"token_type"`
	ExpiresIn             int64  `json:"expires_in"`
	RefreshTokenExpiresIn int64  `json:"refresh_token_expires_in"`
	RefreshToken          string `json:"refresh_token,omitempty"`
	IDToken               string `json:"id_token,omitempty"`
	Scope                 string `json:"scope"`
}

// UserInfoResponse represents the OpenID Connect userinfo response.
// Fields are conditionally included based on granted scopes.
type UserInfoResponse struct {
	// Always included (openid scope)
	Sub string `json:"sub"`

	// Profile scope
	Name              string `json:"name,omitempty"`
	PreferredUsername string `json:"preferred_username,omitempty"`
	Picture           string `json:"picture,omitempty"`
	UpdatedAt         int64  `json:"updated_at,omitempty"`

	// Email scope
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
}

// AuthorizationCode represents the data stored in KV for an OAuth authorization code.
// Used for PKCE verification during token exchange.
type AuthorizationCode struct {
	ClientID      string   `json:"client_id"`
	UserID        int      `json:"user_id"`
	Scopes        []string `json:"scopes"`
	RedirectURI   string   `json:"redirect_uri"`
	Nonce         string   `json:"nonce"`
	CodeChallenge string   `json:"code_challenge"`
}

func authCodeKey(code string) string {
	return authCodeKeyPrefix + code
}

func init() {
	gob.Register(&AuthorizationCode{})
}
