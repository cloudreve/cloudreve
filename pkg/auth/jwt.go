package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"
)

type TokenAuth interface {
	// Issue issues a new pair of credentials for the given user.
	Issue(ctx context.Context, args *IssueTokenArgs) (*Token, error)
	// VerifyAndRetrieveUser verifies the given token and inject the user into current context.
	// Returns if upper caller should continue process other session provider.
	VerifyAndRetrieveUser(c *gin.Context) (bool, error)
	// Refresh refreshes the given refresh token and returns a new pair of credentials.
	Refresh(ctx context.Context, refreshToken string) (*Token, error)
	// Claims parses the given token string and returns the claims.
	Claims(ctx context.Context, tokenStr string) (*Claims, error)
}

type IssueTokenArgs struct {
	User               *ent.User
	RootTokenID        *uuid.UUID
	ClientID           string
	Scopes             []string
	RefreshTTLOverride time.Duration
}

// Token stores token pair for authentication
type Token struct {
	AccessToken    string    `json:"access_token"`
	RefreshToken   string    `json:"refresh_token"`
	AccessExpires  time.Time `json:"access_expires"`
	RefreshExpires time.Time `json:"refresh_expires"`

	UID int `json:"-"`
}

type (
	TokenType                        string
	TokenIDContextKey                struct{}
	ScopeContextKey                  struct{}
	verifiedAccessClientIDContextKey struct{}
)

var (
	TokenTypeAccess  = TokenType("access")
	TokenTypeRefresh = TokenType("refresh")

	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrUserNotFound        = errors.New("user not found")
)

const (
	AuthorizationHeader = "Authorization"
	TokenHeaderPrefix   = "Bearer "
	RevokeTokenPrefix   = "jwt_revoke_"
)

type Claims struct {
	TokenType TokenType `json:"token_type"`
	jwt.RegisteredClaims
	StateHash   []byte     `json:"state_hash,omitempty"`
	RootTokenID *uuid.UUID `json:"root_token_id,omitempty"`
	Scopes      []string   `json:"scopes,omitempty"`
	ClientID    string     `json:"client_id,omitempty"`
}

// NewTokenAuth creates a new token based auth provider.
func NewTokenAuth(idEncoder hashid.Encoder, s setting.Provider, secret []byte, userClient inventory.UserClient,
	l logging.Logger, kv cache.Driver, oAuthClient inventory.OAuthClientClient) TokenAuth {
	return &tokenAuth{
		idEncoder:   idEncoder,
		s:           s,
		secret:      secret,
		userClient:  userClient,
		l:           l,
		kv:          kv,
		oAuthClient: oAuthClient,
	}
}

type tokenAuth struct {
	l           logging.Logger
	idEncoder   hashid.Encoder
	s           setting.Provider
	secret      []byte
	userClient  inventory.UserClient
	oAuthClient inventory.OAuthClientClient
	kv          cache.Driver
}

func (t *tokenAuth) Claims(ctx context.Context, tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return t.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

func (t *tokenAuth) Refresh(ctx context.Context, refreshToken string) (*Token, error) {
	token, err := jwt.ParseWithClaims(refreshToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return t.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || claims.TokenType != TokenTypeRefresh {
		return nil, ErrInvalidRefreshToken
	}

	uid, err := t.idEncoder.Decode(claims.Subject, hashid.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	expectedUser, err := t.userClient.GetActiveByID(ctx, uid)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Check if user changed password or revoked session
	expectedHash := t.hashUserState(ctx, expectedUser)
	if !bytes.Equal(claims.StateHash, expectedHash[:]) {
		return nil, ErrInvalidRefreshToken
	}

	// Check if root token is revoked
	if claims.RootTokenID == nil {
		return nil, ErrInvalidRefreshToken
	}

	_, ok = t.kv.Get(fmt.Sprintf("%s%s", RevokeTokenPrefix, claims.RootTokenID.String()))
	if ok {
		return nil, ErrInvalidRefreshToken
	}

	// If token issued for an OAuth client, check if the client is still valid
	refreshTTLOverride := time.Duration(0)
	if claims.ClientID != "" {
		client, err := t.oAuthClient.GetByGUIDWithGrants(ctx, claims.ClientID, expectedUser.ID)
		if err != nil || len(client.Edges.Grants) == 0 {
			return nil, ErrInvalidRefreshToken
		}

		// Consented scopes must be a subset of the client's scopes
		if !ValidateScopes(claims.Scopes, client.Edges.Grants[0].Scopes) {
			return nil, ErrInvalidRefreshToken
		}

		// Update last used at for the grant
		if err := t.oAuthClient.UpdateGrantLastUsedAt(ctx, expectedUser.ID, client.ID); err != nil {
			return nil, ErrInvalidRefreshToken
		}

		if client.Props != nil {
			refreshTTLOverride = time.Duration(client.Props.RefreshTokenTTL) * time.Second
		}
	}

	return t.Issue(ctx, &IssueTokenArgs{
		User:               expectedUser,
		RootTokenID:        claims.RootTokenID,
		Scopes:             claims.Scopes,
		ClientID:           claims.ClientID,
		RefreshTTLOverride: refreshTTLOverride,
	})
}

func (t *tokenAuth) VerifyAndRetrieveUser(c *gin.Context) (bool, error) {
	headerVal := c.GetHeader(AuthorizationHeader)
	if strings.HasPrefix(headerVal, TokenHeaderPrefixCr) {
		// This is an HMAC auth header, skip JWT verification
		return false, nil
	}

	tokenString := strings.TrimPrefix(headerVal, TokenHeaderPrefix)
	if tokenString == "" {
		return true, nil
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return t.secret, nil
	})

	if err != nil {
		t.l.Warning("Failed to parse jwt token: %s", err)
		return false, nil
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || claims.TokenType != TokenTypeAccess {
		return false, serializer.NewError(serializer.CodeCredentialInvalid, "Invalid token type", nil)
	}

	uid, err := t.idEncoder.Decode(claims.Subject, hashid.UserID)
	if err != nil {
		// A subject that cannot be decoded means the token is not usable with
		// the current hash ID salt. Treat it like any other invalid token and
		// continue as anonymous; returning an error here makes every endpoint
		// (including public ones) fail with a "404" response.
		return false, nil
	}

	util.WithValue(c, inventory.UserIDCtx{}, uid)
	util.WithValue(c, verifiedAccessClientIDContextKey{}, claims.ClientID)

	if claims.ClientID != "" {
		util.WithValue(c, ScopeContextKey{}, claims.Scopes)
	}
	return false, nil
}

func (t *tokenAuth) Issue(ctx context.Context, args *IssueTokenArgs) (*Token, error) {
	u := args.User
	rootTokenID := args.RootTokenID

	uidEncoded := hashid.EncodeUserID(t.idEncoder, u.ID)
	tokenSettings := t.s.TokenAuth(ctx)
	issueDate := time.Now()
	accessTokenExpired := time.Now().Add(tokenSettings.AccessTokenTTL)
	refreshTokenExpired := time.Now().Add(tokenSettings.RefreshTokenTTL)
	if args.RefreshTTLOverride > 0 {
		refreshTokenExpired = time.Now().Add(args.RefreshTTLOverride)
	}
	if rootTokenID == nil {
		newRootTokenID := uuid.Must(uuid.NewV4())
		rootTokenID = &newRootTokenID
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uidEncoded,
			NotBefore: jwt.NewNumericDate(issueDate),
			ExpiresAt: jwt.NewNumericDate(accessTokenExpired),
		},
		ClientID: args.ClientID,
		Scopes:   args.Scopes,
	}).SignedString(t.secret)
	if err != nil {
		return nil, fmt.Errorf("faield to sign access token: %w", err)
	}

	userHash := t.hashUserState(ctx, u)
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		TokenType:   TokenTypeRefresh,
		RootTokenID: rootTokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uidEncoded,
			NotBefore: jwt.NewNumericDate(issueDate),
			ExpiresAt: jwt.NewNumericDate(refreshTokenExpired),
		},
		Scopes:    args.Scopes,
		ClientID:  args.ClientID,
		StateHash: userHash[:],
	}).SignedString(t.secret)
	if err != nil {
		return nil, fmt.Errorf("faield to sign refresh token: %w", err)
	}

	return &Token{
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
		AccessExpires:  accessTokenExpired,
		RefreshExpires: refreshTokenExpired,
		UID:            u.ID,
	}, nil
}

// hashUserState returns a hash string for user state for critical fields, it is used
// to detect refresh token revocation after user changed password.
func (t *tokenAuth) hashUserState(ctx context.Context, u *ent.User) [32]byte {
	return sha256.Sum256([]byte(fmt.Sprintf("%s/%s/%s", u.Email, u.Password, t.s.SiteBasic(ctx).ID)))
}

// ValidateScopes checks if all requested scopes are a subset of the allowed scopes.
// Returns true if all requested scopes are valid, false otherwise.
func ValidateScopes(requestedScopes, allowedScopes []string) bool {
	allowed := make(map[string]struct{}, len(allowedScopes))
	for _, scope := range allowedScopes {
		allowed[scope] = struct{}{}
	}
	for _, scope := range requestedScopes {
		if _, ok := allowed[scope]; !ok {
			return false
		}
	}
	return true
}

func GetScopesFromContext(ctx context.Context) (bool, []string) {
	scopes, ok := ctx.Value(ScopeContextKey{}).([]string)
	if !ok {
		return false, nil
	}
	return true, scopes
}

// CheckFirstParty requires a verified first-party access token, not an OAuth client token.
func CheckFirstParty(ctx context.Context) error {
	clientID, ok := ctx.Value(verifiedAccessClientIDContextKey{}).(string)
	if !ok || clientID != "" {
		return serializer.NewError(serializer.CodeNoPermissionErr, "First-party authentication required", nil)
	}
	return nil
}

func CheckScope(c *gin.Context, requiredScopes ...string) error {
	hasScopes, tokenScopes := GetScopesFromContext(c)
	if !hasScopes {
		return nil
	}

	// Build a set of token scopes including implicit read permissions from write scopes
	scopeSet := make(map[string]struct{}, len(tokenScopes)*2)
	for _, scope := range tokenScopes {
		scopeSet[scope] = struct{}{}
		// If scope is "xxx.Write", also grant "xxx.Read"
		if resource, ok := extractWriteResource(scope); ok {
			scopeSet[resource+".Read"] = struct{}{}
		}
	}

	// Check if all required scopes are present
	for _, required := range requiredScopes {
		if _, ok := scopeSet[required]; !ok {
			return serializer.NewError(serializer.CodeInsufficientScope,
				"Insufficient scope: "+required, nil)
		}
	}

	return nil
}

// extractWriteResource extracts the resource name from a write scope.
// For example, "File.Write" returns ("File", true), "File.Read" returns ("", false).
func extractWriteResource(scope string) (string, bool) {
	const writeSuffix = ".Write"
	if len(scope) > len(writeSuffix) && scope[len(scope)-len(writeSuffix):] == writeSuffix {
		return scope[:len(scope)-len(writeSuffix)], true
	}
	return "", false
}
