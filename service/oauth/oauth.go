package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
)

type (
	GetAppRegistrationParamCtx struct{}
	GetAppRegistrationService  struct {
		AppID string `uri:"app_id" binding:"required"`
	}
)

func (s *GetAppRegistrationService) Get(c *gin.Context) (*AppRegistration, error) {
	dep := dependency.FromContext(c)
	oAuthClient := dep.OAuthClientClient()

	app, err := oAuthClient.GetByGUIDWithGrants(c, s.AppID, inventory.UserIDFromContext(c))
	if err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "App not found", err)
	}

	var grant *ent.OAuthGrant
	if len(app.Edges.Grants) == 1 {
		grant = app.Edges.Grants[0]
	}

	return BuildAppRegistration(app, grant), nil
}

type (
	GrantParamCtx struct{}
	GrantService  struct {
		ClientID            string `json:"client_id" binding:"required"`
		ResponseType        string `json:"response_type" binding:"required,eq=code"`
		RedirectURI         string `json:"redirect_uri" binding:"required"`
		State               string `json:"state" binding:"max=4096"`
		Scope               string `json:"scope" binding:"required"`
		CodeChallenge       string `json:"code_challenge" binding:"max=255"`
		CodeChallengeMethod string `json:"code_challenge_method" binding:"omitempty,eq=S256"`
	}
)

func (s *GrantService) Get(c *gin.Context) (*GrantResponse, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	kv := dep.KV()
	oAuthClient := dep.OAuthClientClient()
	if s.CodeChallenge != "" && s.CodeChallengeMethod == "" {
		s.CodeChallengeMethod = "S256"
	}

	// 1. Get app registration and grant
	app, err := oAuthClient.GetByGUIDWithGrants(c, s.ClientID, user.ID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "App not found", err)
	}

	// 2. Validate redirect URL: must match one of the registered redirect URIs
	redirectValid := false
	for _, uri := range app.RedirectUris {
		if uri == s.RedirectURI {
			redirectValid = true
			break
		}
	}
	if !redirectValid {
		return nil, serializer.NewError(serializer.CodeParamErr, "Invalid redirect URI", nil)
	}

	// Parse requested scopes (space-separated per OAuth 2.0 spec)
	requestedScopes := strings.Split(s.Scope, " ")

	// Validate requested scopes: must be a subset of registered app scopes
	if !auth.ValidateScopes(requestedScopes, app.Scopes) {
		return nil, serializer.NewError(serializer.CodeParamErr, "Invalid scope requested", nil)
	}

	// 3. Create/update grant
	if err := oAuthClient.UpsertGrant(c, user.ID, app.ID, requestedScopes); err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to create grant", err)
	}

	// 4. Generate code and save required state into KV for future token exchange request.
	code := util.RandStringRunesCrypto(128)
	authCode := &AuthorizationCode{
		ClientID:      s.ClientID,
		UserID:        user.ID,
		Scopes:        requestedScopes,
		RedirectURI:   s.RedirectURI,
		CodeChallenge: s.CodeChallenge,
	}

	// Store auth code in KV with 10 minute TTL
	if err := kv.Set(authCodeKey(code), authCode, 600); err != nil {
		return nil, serializer.NewError(serializer.CodeCacheOperation, "Failed to store authorization code", err)
	}

	return &GrantResponse{
		Code:  code,
		State: s.State,
	}, nil
}

type (
	ExchangeTokenParamCtx struct{}
	ExchangeTokenService  struct {
		ClientID     string `form:"client_id" binding:"required"`
		ClientSecret string `form:"client_secret" binding:"required"`
		GrantType    string `form:"grant_type" binding:"required,eq=authorization_code"`
		Code         string `form:"code" binding:"required"`
		CodeVerifier string `form:"code_verifier"`
	}
)

func (s *ExchangeTokenService) Exchange(c *gin.Context) (*TokenResponse, error) {
	dep := dependency.FromContext(c)
	kv := dep.KV()
	oAuthClient := dep.OAuthClientClient()
	userClient := dep.UserClient()
	tokenAuth := dep.TokenAuth()

	// 1. Retrieve and validate authorization code from KV
	codeKey := authCodeKey(s.Code)
	authCodeRaw, ok := kv.Get(codeKey)
	if !ok {
		return nil, serializer.NewError(serializer.CodeCredentialInvalid, "Invalid or expired authorization code", nil)
	}

	authCode, ok := authCodeRaw.(*AuthorizationCode)
	if !ok {
		return nil, serializer.NewError(serializer.CodeCredentialInvalid, "Invalid authorization code", nil)
	}

	// Delete the code immediately to prevent replay attacks
	_ = kv.Delete("", codeKey)

	// 2. Validate client_id matches the one in authorization code
	if authCode.ClientID != s.ClientID {
		return nil, serializer.NewError(serializer.CodeCredentialInvalid, "Client ID mismatch", nil)
	}

	// 3. Verify PKCE: SHA256(code_verifier) should match code_challenge
	if authCode.CodeChallenge != "" {
		verifierHash := sha256.Sum256([]byte(s.CodeVerifier))
		expectedChallenge := base64.RawURLEncoding.EncodeToString(verifierHash[:])
		if expectedChallenge != authCode.CodeChallenge {
			return nil, serializer.NewError(serializer.CodeCredentialInvalid, "Invalid code verifier", nil)
		}
	}

	// 4. Validate client secret
	app, err := oAuthClient.GetByGUID(c, s.ClientID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "App not found", err)
	}

	if app.Secret != s.ClientSecret {
		return nil, serializer.NewError(serializer.CodeCredentialInvalid, "Invalid client secret", nil)
	}

	// 5. Validate scopes are still valid for this app
	if !auth.ValidateScopes(authCode.Scopes, app.Scopes) {
		return nil, serializer.NewError(serializer.CodeParamErr, "Invalid scope", nil)
	}

	// 6. Get user
	user, err := userClient.GetActiveByID(c, authCode.UserID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeUserNotFound, "User not found", err)
	}

	// 7. Determine refresh token TTL override from app settings
	var refreshTTLOverride time.Duration
	if app.Props != nil && app.Props.RefreshTokenTTL > 0 {
		refreshTTLOverride = time.Duration(app.Props.RefreshTokenTTL) * time.Second
	}

	// 8. Issue tokens
	token, err := tokenAuth.Issue(c, &auth.IssueTokenArgs{
		User:               user,
		ClientID:           s.ClientID,
		Scopes:             authCode.Scopes,
		RefreshTTLOverride: refreshTTLOverride,
	})
	if err != nil {
		return nil, serializer.NewError(serializer.CodeCredentialInvalid, "Failed to issue token", err)
	}

	// 9. Update grant last used at
	if err := oAuthClient.UpdateGrantLastUsedAt(c, user.ID, app.ID); err != nil {
		dep.Logger().Warning("Failed to update grant last used at: %s", err)
	}

	// 10.

	// 11. Build response, only include refresh token if offline_access scope is present
	resp := &TokenResponse{
		AccessToken:           token.AccessToken,
		TokenType:             "Bearer",
		ExpiresIn:             int64(time.Until(token.AccessExpires).Seconds()),
		RefreshTokenExpiresIn: int64(time.Until(token.RefreshExpires).Seconds()),
		Scope:                 strings.Join(authCode.Scopes, " "),
	}

	for _, scope := range authCode.Scopes {
		if scope == types.ScopeOfflineAccess {
			resp.RefreshToken = token.RefreshToken
			break
		}
	}

	return resp, nil
}

type (
	DeleteOAuthGrantParamCtx struct{}
	DeleteOAuthGrantService  struct {
		AppID string `uri:"app_id" binding:"required"`
	}
)

func (s *DeleteOAuthGrantService) Delete(c *gin.Context) error {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	oAuthClient := dep.OAuthClientClient()

	// Delete the grant - the method validates that the grant belongs to the current user
	deleted, err := oAuthClient.DeleteGrantByUserAndClientGUID(c, user.ID, s.AppID)
	if err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to delete OAuth grant", err)
	}

	if !deleted {
		return serializer.NewError(serializer.CodeNotFound, "OAuth grant not found", nil)
	}

	return nil
}

type (
	UserInfoParamCtx struct{}
	UserInfoService  struct{}
)

// GetUserInfo returns OpenID Connect userinfo based on the access token's scopes.
// The response fields are conditionally included based on granted scopes:
// - openid: sub (always required)
// - profile: name, preferred_username, picture, updated_at
// - email: email, email_verified
func (s *UserInfoService) GetUserInfo(c *gin.Context) (*UserInfoResponse, error) {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	hashIDEncoder := dep.HashIDEncoder()

	// 1. Get and parse the access token from Authorization header
	hasScopes, scopes := auth.GetScopesFromContext(c)
	if hasScopes {
		// 2. Verify openid scope is present (required for userinfo endpoint)
		hasOpenID := false
		for _, scope := range scopes {
			if scope == types.ScopeOpenID {
				hasOpenID = true
				break
			}
		}
		if !hasOpenID {
			return nil, serializer.NewError(serializer.CodeNoPermissionErr, "openid scope required", nil)
		}
	} else {
		scopes = []string{types.ScopeOpenID, types.ScopeProfile, types.ScopeEmail}
	}

	// 4. Build response based on scopes
	resp := &UserInfoResponse{
		Sub: hashid.EncodeUserID(hashIDEncoder, u.ID),
	}

	// Check scopes and populate fields accordingly
	for _, scope := range scopes {
		switch scope {
		case types.ScopeProfile:
			siteUrl := dep.SettingProvider().SiteURL(c)
			resp.Name = u.Nick
			resp.PreferredUsername = u.Nick
			resp.Picture = routes.MasterUserAvatarUrl(siteUrl, hashid.EncodeUserID(hashIDEncoder, u.ID)).String()
			resp.UpdatedAt = u.UpdatedAt.Unix()
		case types.ScopeEmail:
			resp.Email = u.Email
			resp.EmailVerified = true // Users in Cloudreve have verified emails
		}
	}

	return resp, nil
}
