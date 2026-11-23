package routers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/middleware"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type directLinkScopeConfig struct {
	conf.ConfigProvider
}

func (directLinkScopeConfig) System() *conf.System {
	return &conf.System{Mode: conf.MasterMode, SessionSecret: "direct-link-scope-test"}
}

func (directLinkScopeConfig) Cors() *conf.Cors {
	return &conf.Cors{AllowOrigins: []string{"UNSET"}}
}

type directLinkScopeTokenAuth struct {
	auth.TokenAuth
	scopes []string
}

func (a *directLinkScopeTokenAuth) VerifyAndRetrieveUser(c *gin.Context) (bool, error) {
	util.WithValue(c, inventory.UserIDCtx{}, 1)
	if a.scopes != nil {
		util.WithValue(c, auth.ScopeContextKey{}, a.scopes)
	}
	return false, nil
}

type directLinkScopeUsers struct {
	inventory.UserClient
}

func (directLinkScopeUsers) GetLoginUserByID(context.Context, int) (*ent.User, error) {
	return &ent.User{ID: 1}, nil
}

type directLinkScopeDep struct {
	dependency.Dep
	tokenAuth *directLinkScopeTokenAuth
	encoder   hashid.Encoder
}

func (*directLinkScopeDep) ConfigProvider() conf.ConfigProvider { return directLinkScopeConfig{} }
func (*directLinkScopeDep) Logger() logging.Logger {
	return logging.NewConsoleLogger(logging.LevelError)
}
func (*directLinkScopeDep) ServerStaticFS() static.ServeFileSystem { return nil }
func (*directLinkScopeDep) KV() cache.Driver                       { return nil }
func (*directLinkScopeDep) GeneralAuth() auth.Auth                 { return nil }
func (*directLinkScopeDep) SettingProvider() setting.Provider      { return nil }
func (*directLinkScopeDep) UserClient() inventory.UserClient       { return directLinkScopeUsers{} }
func (d *directLinkScopeDep) HashIDEncoder() hashid.Encoder        { return d.encoder }
func (d *directLinkScopeDep) TokenAuth() auth.TokenAuth            { return d.tokenAuth }
func (d *directLinkScopeDep) ForkWithLogger(ctx context.Context, _ logging.Logger) context.Context {
	return context.WithValue(ctx, dependency.DepCtx{}, d)
}

func TestDirectLinkRoutesRequireWriteScope(t *testing.T) {
	oldMode := gin.Mode()
	oldStore := middleware.Store
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() {
		gin.SetMode(oldMode)
		middleware.Store = oldStore
	})

	tokenAuth := &directLinkScopeTokenAuth{}
	encoder, err := hashid.New("direct-link-scope-test")
	require.NoError(t, err)
	router := initMasterRouter(&directLinkScopeDep{tokenAuth: tokenAuth, encoder: encoder})
	tests := []struct {
		name          string
		scopes        []string
		wantCode      int
		requiredScope string
	}{
		{
			name:          "read only",
			scopes:        []string{"openid", types.ScopeFilesRead},
			wantCode:      serializer.CodeInsufficientScope,
			requiredScope: types.ScopeFilesWrite,
		},
		{
			name:          "no file scope",
			scopes:        []string{"openid"},
			wantCode:      serializer.CodeInsufficientScope,
			requiredScope: types.ScopeFilesRead,
		},
		{
			name:     "write implies read",
			scopes:   []string{"openid", types.ScopeFilesWrite},
			wantCode: serializer.CodeParamErr,
		},
		{
			name:     "read and write",
			scopes:   []string{"openid", types.ScopeFilesRead, types.ScopeFilesWrite},
			wantCode: serializer.CodeParamErr,
		},
		{
			name:     "session without OAuth scopes",
			wantCode: serializer.CodeParamErr,
		},
	}
	for _, route := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPut, path: "/api/v4/file/source"},
		{method: http.MethodDelete, path: "/api/v4/file/source/!"},
	} {
		for _, test := range tests {
			t.Run(route.method+"/"+test.name, func(t *testing.T) {
				tokenAuth.scopes = test.scopes
				// Authorized requests reach input validation without mutating files.
				req := httptest.NewRequest(route.method, route.path, strings.NewReader("{"))
				req.Header.Set("Content-Type", "application/json")
				recorder := httptest.NewRecorder()
				router.ServeHTTP(recorder, req)

				require.Equal(t, http.StatusOK, recorder.Code)
				var response struct {
					Code int    `json:"code"`
					Msg  string `json:"msg"`
				}
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
				require.Equal(t, test.wantCode, response.Code, recorder.Body.String())
				if test.requiredScope != "" {
					require.Contains(t, response.Msg, test.requiredScope)
				}
			})
		}
	}
}
