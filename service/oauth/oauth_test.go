package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestValidateClientAuth(t *testing.T) {
	confidential := &ent.OAuthClient{Secret: "s3cret"}
	public := &ent.OAuthClient{Secret: ""}

	// Confidential clients: secret required and must match.
	require.Error(t, validateClientAuth(confidential, "", "challenge"))
	require.Error(t, validateClientAuth(confidential, "wrong", "challenge"))
	require.NoError(t, validateClientAuth(confidential, "s3cret", ""))

	// Public clients: secret ignored, PKCE challenge mandatory.
	require.Error(t, validateClientAuth(public, "", ""))
	require.Error(t, validateClientAuth(public, "anything", ""))
	require.NoError(t, validateClientAuth(public, "", "challenge"))
	require.NoError(t, validateClientAuth(public, "stale-known-secret", "challenge"))
}

type oauthTestClient struct {
	inventory.OAuthClientClient
	app           *ent.OAuthClient
	grants, reads int
}

func (c *oauthTestClient) GetByGUIDWithGrants(context.Context, string, int) (*ent.OAuthClient, error) {
	if c.app == nil {
		return nil, errors.New("unknown client")
	}
	return c.app, nil
}
func (c *oauthTestClient) UpsertGrant(context.Context, int, int, []string) error {
	c.grants++
	return nil
}
func (c *oauthTestClient) GetByGUID(context.Context, string) (*ent.OAuthClient, error) {
	c.reads++
	return nil, errors.New("stop after grant verification")
}

type oauthTestCache struct {
	cache.Driver
	code   *AuthorizationCode
	writes int
}

func (c *oauthTestCache) Get(string) (any, bool)         { return c.code, c.code != nil }
func (c *oauthTestCache) Set(string, any, int) error     { c.writes++; return nil }
func (c *oauthTestCache) Delete(string, ...string) error { return nil }

type oauthTestDep struct {
	dependency.Dep
	client *oauthTestClient
	kv     *oauthTestCache
}

func (d oauthTestDep) OAuthClientClient() inventory.OAuthClientClient { return d.client }
func (d oauthTestDep) KV() cache.Driver                               { return d.kv }
func (d oauthTestDep) UserClient() inventory.UserClient               { return nil }
func (d oauthTestDep) TokenAuth() auth.TokenAuth                      { return nil }
func (d oauthTestDep) Logger() logging.Logger {
	return logging.NewConsoleLogger(logging.LevelError)
}

func oauthTestContext(client *oauthTestClient, kv *oauthTestCache) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	engine.ContextWithFallback = true
	ctx := context.WithValue(context.Background(), dependency.DepCtx{}, oauthTestDep{client: client, kv: kv})
	ctx = context.WithValue(ctx, inventory.UserCtx{}, &ent.User{ID: 1})
	c.Request = httptest.NewRequest("POST", "/", nil).WithContext(ctx)
	return c
}

func TestConsentDenialValidatesRedirectWithoutIssuingGrant(t *testing.T) {
	client := &oauthTestClient{app: &ent.OAuthClient{RedirectUris: []string{"http://127.0.0.1/callback"}}}
	kv := &oauthTestCache{}
	c := oauthTestContext(client, kv)
	s := GrantService{ClientID: "client", RedirectURI: "http://127.0.0.1:49152/callback", State: "state", Deny: true}
	response, err := s.Issue(c)
	require.NoError(t, err)
	require.Equal(t, "access_denied", response.Error)
	require.Equal(t, "state", response.State)
	require.Empty(t, response.Code)
	require.Zero(t, client.grants)
	require.Zero(t, kv.writes)

	s.RedirectURI = "http://evil.example/callback"
	response, err = s.Issue(c)
	require.ErrorContains(t, err, "Invalid redirect URI")
	require.Nil(t, response)
	client.app = nil
	s.RedirectURI = "http://127.0.0.1:49152/callback"
	response, err = s.Issue(c)
	require.ErrorContains(t, err, "App not found")
	require.Nil(t, response)
	require.Zero(t, client.grants)
	require.Zero(t, kv.writes)
}

func TestTokenExchangeBindsProvidedRedirectAndRejectsPKCEDowngrade(t *testing.T) {
	uri := "http://127.0.0.1:49152/callback"
	other := "http://127.0.0.1:49153/callback"
	verifier := "verifier"
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])
	for _, test := range []struct {
		name                string
		redirect            string
		challenge, verifier string
		valid               bool
	}{
		{"legacy omission", "", "", "", true},
		{"exact redirect", uri, challenge, verifier, true},
		{"other port", other, challenge, verifier, false},
		{"PKCE downgrade", uri, "", verifier, false},
		{"missing verifier", uri, challenge, "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := &oauthTestClient{}
			kv := &oauthTestCache{code: &AuthorizationCode{ClientID: "client", RedirectURI: uri, CodeChallenge: test.challenge}}
			s := ExchangeTokenService{ClientID: "client", RedirectURI: test.redirect, CodeVerifier: test.verifier}
			_, err := s.Exchange(oauthTestContext(client, kv))
			require.Error(t, err)
			if test.valid {
				require.Equal(t, 1, client.reads, "valid grant reaches client credential validation")
			} else {
				require.Zero(t, client.reads, "invalid grant must fail before client credential validation")
			}
		})
	}
}

func TestConsentRequestCannotSetInternalDenial(t *testing.T) {
	var service GrantService
	require.NoError(t, json.Unmarshal([]byte(`{"deny":true}`), &service))
	require.False(t, service.Deny)
}
