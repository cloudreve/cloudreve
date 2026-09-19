package auth

import (
	"context"
	"testing"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
)

type stubSettingProvider struct {
	setting.Provider
	tokenAuth *setting.TokenAuth
	siteBasic *setting.SiteBasic
}

func (s *stubSettingProvider) TokenAuth(ctx context.Context) *setting.TokenAuth {
	return s.tokenAuth
}

func (s *stubSettingProvider) SiteBasic(ctx context.Context) *setting.SiteBasic {
	return s.siteBasic
}

type stubUserClient struct {
	inventory.UserClient
	user *ent.User
	err  error
}

func (s *stubUserClient) GetActiveByID(ctx context.Context, id int) (*ent.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.user, nil
}

func newTestTokenAuth(t *testing.T, userClient inventory.UserClient, kv cache.Driver) *tokenAuth {
	t.Helper()
	encoder, err := hashid.New("test-salt")
	if err != nil {
		t.Fatalf("failed to create hashid encoder: %v", err)
	}

	return &tokenAuth{
		idEncoder: encoder,
		s: &stubSettingProvider{
			tokenAuth: &setting.TokenAuth{
				AccessTokenTTL:  time.Hour,
				RefreshTokenTTL: 24 * time.Hour,
			},
			siteBasic: &setting.SiteBasic{ID: "site-id"},
		},
		secret:     []byte("jwt-secret"),
		userClient: userClient,
		l:          logging.NewConsoleLogger(logging.LevelDebug),
		kv:         kv,
	}
}

func TestIssueAndClaimsRoundTrip(t *testing.T) {
	ta := newTestTokenAuth(t, &stubUserClient{}, cache.NewMemoStore("", nil))

	user := &ent.User{ID: 42, Email: "u@example.com", Password: "pw"}
	token, err := ta.Issue(context.Background(), &IssueTokenArgs{User: user})
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	claims, err := ta.Claims(context.Background(), token.AccessToken)
	if err != nil {
		t.Fatalf("Claims failed: %v", err)
	}
	if claims.TokenType != TokenTypeAccess {
		t.Fatalf("expected access token type, got %q", claims.TokenType)
	}

	uid, err := ta.idEncoder.Decode(claims.Subject, hashid.UserID)
	if err != nil {
		t.Fatalf("failed to decode subject: %v", err)
	}
	if uid != user.ID {
		t.Fatalf("expected uid %d, got %d", user.ID, uid)
	}

	refreshClaims, err := ta.Claims(context.Background(), token.RefreshToken)
	if err != nil {
		t.Fatalf("Claims on refresh token failed: %v", err)
	}
	if refreshClaims.TokenType != TokenTypeRefresh {
		t.Fatalf("expected refresh token type, got %q", refreshClaims.TokenType)
	}
	if refreshClaims.RootTokenID == nil {
		t.Fatal("refresh token missing RootTokenID")
	}
}

func TestClaimsRejectsTamperedToken(t *testing.T) {
	ta := newTestTokenAuth(t, &stubUserClient{}, cache.NewMemoStore("", nil))

	if _, err := ta.Claims(context.Background(), "not.a.jwt"); err == nil {
		t.Fatal("expected error for malformed token")
	}

	// Sign with a different secret must fail verification.
	other := newTestTokenAuth(t, &stubUserClient{}, cache.NewMemoStore("", nil))
	other.secret = []byte("other-secret")
	token, err := other.Issue(context.Background(), &IssueTokenArgs{User: &ent.User{ID: 1}})
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}
	if _, err := ta.Claims(context.Background(), token.AccessToken); err == nil {
		t.Fatal("expected signature mismatch error")
	}
}

func TestRefreshRotatesTokenPair(t *testing.T) {
	user := &ent.User{ID: 7, Email: "r@example.com", Password: "pw"}
	kv := cache.NewMemoStore("", nil)
	ta := newTestTokenAuth(t, &stubUserClient{user: user}, kv)

	token, err := ta.Issue(context.Background(), &IssueTokenArgs{User: user})
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	refreshed, err := ta.Refresh(context.Background(), token.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}
	if refreshed.AccessToken == "" || refreshed.RefreshToken == "" {
		t.Fatal("refresh returned empty token pair")
	}
}

func TestRefreshRejectsAccessToken(t *testing.T) {
	user := &ent.User{ID: 7, Email: "r@example.com", Password: "pw"}
	ta := newTestTokenAuth(t, &stubUserClient{user: user}, cache.NewMemoStore("", nil))

	token, err := ta.Issue(context.Background(), &IssueTokenArgs{User: user})
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	if _, err := ta.Refresh(context.Background(), token.AccessToken); err != ErrInvalidRefreshToken {
		t.Fatalf("expected ErrInvalidRefreshToken, got %v", err)
	}
}

func TestRefreshRejectsRevokedRootToken(t *testing.T) {
	user := &ent.User{ID: 7, Email: "r@example.com", Password: "pw"}
	kv := cache.NewMemoStore("", nil)
	ta := newTestTokenAuth(t, &stubUserClient{user: user}, kv)

	token, err := ta.Issue(context.Background(), &IssueTokenArgs{User: user})
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	claims, err := ta.Claims(context.Background(), token.RefreshToken)
	if err != nil {
		t.Fatalf("Claims failed: %v", err)
	}
	kv.Set(RevokeTokenPrefix+claims.RootTokenID.String(), true, 0)

	if _, err := ta.Refresh(context.Background(), token.RefreshToken); err != ErrInvalidRefreshToken {
		t.Fatalf("expected ErrInvalidRefreshToken for revoked root, got %v", err)
	}
}

func TestRefreshRejectsChangedUserState(t *testing.T) {
	user := &ent.User{ID: 7, Email: "r@example.com", Password: "pw"}
	kv := cache.NewMemoStore("", nil)
	ta := newTestTokenAuth(t, &stubUserClient{user: user}, kv)

	token, err := ta.Issue(context.Background(), &IssueTokenArgs{User: user})
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	// Password change invalidates the state hash in the refresh token.
	changed := &ent.User{ID: 7, Email: "r@example.com", Password: "new-pw"}
	ta.userClient = &stubUserClient{user: changed}

	if _, err := ta.Refresh(context.Background(), token.RefreshToken); err != ErrInvalidRefreshToken {
		t.Fatalf("expected ErrInvalidRefreshToken after password change, got %v", err)
	}
}

func TestValidateScopes(t *testing.T) {
	if !ValidateScopes([]string{"a", "b"}, []string{"a", "b", "c"}) {
		t.Fatal("subset should validate")
	}
	if ValidateScopes([]string{"a", "z"}, []string{"a", "b"}) {
		t.Fatal("superset scope should be rejected")
	}
	if !ValidateScopes(nil, []string{"a"}) {
		t.Fatal("empty request should validate")
	}
}
