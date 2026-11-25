package routers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/middleware"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/cloudreve/Cloudreve/v4/service/oauth"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

type oauthConsentTestClients struct {
	inventory.OAuthClientClient
	t       *testing.T
	app     *ent.OAuthClient
	upserts int
}

func (c *oauthConsentTestClients) GetByGUIDWithGrants(_ context.Context, guid string, userID int) (*ent.OAuthClient, error) {
	require.Equal(c.t, c.app.GUID, guid)
	require.Equal(c.t, 1, userID)
	return c.app, nil
}

func (c *oauthConsentTestClients) UpsertGrant(_ context.Context, userID, clientID int, scopes []string) error {
	require.Equal(c.t, 1, userID)
	require.Equal(c.t, c.app.ID, clientID)
	c.upserts++
	c.app.SetGrants([]*ent.OAuthGrant{{
		UserID: userID, ClientID: clientID, Scopes: append([]string(nil), scopes...),
	}})
	return nil
}

type oauthConsentTestDep struct {
	directLinkScopeDep
	tokens  auth.TokenAuth
	clients *oauthConsentTestClients
	kv      *cache.MemoStore
}

func (d *oauthConsentTestDep) TokenAuth() auth.TokenAuth                      { return d.tokens }
func (d *oauthConsentTestDep) OAuthClientClient() inventory.OAuthClientClient { return d.clients }
func (d *oauthConsentTestDep) KV() cache.Driver                               { return d.kv }
func (d *oauthConsentTestDep) ForkWithLogger(ctx context.Context, _ logging.Logger) context.Context {
	return context.WithValue(ctx, dependency.DepCtx{}, d)
}

func TestOAuthConsentRequiresFirstParty(t *testing.T) {
	oldMode := gin.Mode()
	oldStore := middleware.Store
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() {
		gin.SetMode(oldMode)
		middleware.Store = oldStore
	})

	encoder, err := hashid.New("oauth-consent-test")
	require.NoError(t, err)
	secret := []byte("oauth-consent-test-secret")
	logger := logging.NewConsoleLogger(logging.LevelError)
	tokenAuth := auth.NewTokenAuth(encoder, nil, secret, nil, logger, nil, nil)
	requestedScopes := []string{types.ScopeOpenID, types.ScopeFilesRead, types.ScopeFilesWrite}

	for _, entrypoint := range []string{"route", "service"} {
		for _, grantState := range []string{"new grant", "existing grant"} {
			for _, test := range []struct {
				name        string
				clientID    string
				scopes      []string
				allowed     bool
				noToken     bool
				invalidJSON bool
			}{
				{name: "first-party login", allowed: true},
				{name: "openid only", clientID: "target-client", scopes: []string{types.ScopeOpenID}},
				{name: "security write scope", clientID: "target-client", scopes: []string{types.ScopeOpenID, types.ScopeUserSecurityInfoWrite}},
				{name: "nil scopes", clientID: "target-client"},
				{name: "empty scopes", clientID: "target-client", scopes: []string{}},
				{name: "different client", clientID: "other-client", scopes: []string{types.ScopeOpenID}},
				{name: "missing verified identity", noToken: true},
				{name: "denied before binding", clientID: "target-client", scopes: []string{types.ScopeOpenID}, invalidJSON: true},
			} {
				if (entrypoint == "route" && test.noToken) || (entrypoint == "service" && test.invalidJSON) {
					continue
				}
				t.Run(entrypoint+"/"+grantState+"/"+test.name, func(t *testing.T) {
					app := &ent.OAuthClient{
						ID: 2, GUID: "target-client", IsEnabled: true,
						RedirectUris: []string{"https://client.example/callback"},
						Scopes:       requestedScopes,
					}
					if grantState == "existing grant" {
						app.SetGrants([]*ent.OAuthGrant{{UserID: 1, ClientID: app.ID, Scopes: []string{types.ScopeOpenID}}})
					}
					previousGrants := app.Edges.Grants
					dep := &oauthConsentTestDep{
						tokens:  tokenAuth,
						clients: &oauthConsentTestClients{t: t, app: app},
						kv:      cache.NewMemoStore("", logger),
					}
					service := oauth.GrantService{
						ClientID: app.GUID, ResponseType: "code", RedirectURI: app.RedirectUris[0],
						Scope: strings.Join(requestedScopes, " "), State: "consent-test-state",
					}
					body, err := json.Marshal(service)
					require.NoError(t, err)
					if test.invalidJSON {
						body = []byte("{")
					}
					req := httptest.NewRequest(http.MethodPost, "/api/v4/session/oauth/consent", strings.NewReader(string(body)))
					req.Header.Set("Content-Type", "application/json")
					if !test.noToken {
						token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
							"token_type": auth.TokenTypeAccess,
							"client_id":  test.clientID,
							"scopes":     test.scopes,
							"sub":        hashid.EncodeUserID(encoder, 1),
							"exp":        time.Now().Add(time.Hour).Unix(),
						}).SignedString(secret)
						require.NoError(t, err)
						req.Header.Set(auth.AuthorizationHeader, auth.TokenHeaderPrefix+token)
					}

					recorder := httptest.NewRecorder()
					var response struct {
						Code int                  `json:"code"`
						Data *oauth.GrantResponse `json:"data"`
					}
					if entrypoint == "route" {
						router := initMasterRouter(dep)
						router.ServeHTTP(recorder, req)
						require.Equal(t, http.StatusOK, recorder.Code)
						require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
					} else {
						c, engine := gin.CreateTestContext(recorder)
						engine.ContextWithFallback = true
						c.Request = req
						util.WithValue(c, dependency.DepCtx{}, dep)
						middleware.SetUserCtxByUser(c, &ent.User{ID: 1})
						if !test.noToken {
							_, err := tokenAuth.VerifyAndRetrieveUser(c)
							require.NoError(t, err)
						}
						result, err := service.Get(c)
						if err != nil {
							var appErr serializer.AppError
							require.ErrorAs(t, err, &appErr)
							response.Code = appErr.Code
						}
						response.Data = result
					}

					codeCount := 0
					dep.kv.Store.Range(func(_, _ any) bool {
						codeCount++
						return true
					})
					if !test.allowed {
						require.Equal(t, serializer.CodeNoPermissionErr, response.Code)
						require.Nil(t, response.Data)
						require.Zero(t, dep.clients.upserts)
						require.Equal(t, previousGrants, app.Edges.Grants)
						require.Zero(t, codeCount)
						return
					}

					require.Zero(t, response.Code)
					require.NotNil(t, response.Data)
					require.NotEmpty(t, response.Data.Code)
					require.Equal(t, service.State, response.Data.State)
					require.Equal(t, 1, dep.clients.upserts)
					require.Equal(t, requestedScopes, app.Edges.Grants[0].Scopes)
					require.Equal(t, 1, codeCount)
					stored, ok := dep.kv.Get("oauth_code_" + response.Data.Code)
					require.True(t, ok)
					require.Equal(t, &oauth.AuthorizationCode{
						ClientID: app.GUID, UserID: 1, Scopes: requestedScopes, RedirectURI: service.RedirectURI,
					}, stored)
				})
			}
		}
	}
}
