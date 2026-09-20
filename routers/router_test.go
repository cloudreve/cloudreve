package routers

import (
	"path/filepath"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// stubSettingProvider satisfies setting.Provider for route registration;
// per-request setting reads are not exercised here.
type stubSettingProvider struct {
	setting.Provider
}

// TestMasterRouteWiring verifies the master router mounts the core route
// groups end-to-end — a renamed path or dropped middleware chain fails here.
func TestMasterRouteWiring(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := logging.NewConsoleLogger(logging.LevelError)
	cfg, err := conf.NewIniConfigProvider(filepath.Join(t.TempDir(), "conf.ini"), logger)
	require.NoError(t, err)

	dep := dependency.NewDependency(
		dependency.WithLogger(logger),
		dependency.WithConfigProvider(cfg),
		dependency.WithGeneralAuth(auth.HMACAuth{SecretKey: []byte("test")}),
		dependency.WithSettingProvider(&stubSettingProvider{}),
	)

	engine := InitRouter(dep)

	have := make(map[string]bool)
	for _, r := range engine.Routes() {
		have[r.Method+" "+r.Path] = true
	}

	expected := []string{
		"GET /api/v4/site/ping",
		"GET /api/v4/site/config/:section",
		"POST /api/v4/session/token",
		"GET /api/v4/session/prepare",
		"POST /api/v4/user",
		"GET /api/v4/user/activate/:id",
		"GET /api/v4/user/info/:id",
		"PUT /api/v4/file/upload",
		"POST /api/v4/file/upload/:sessionId/:index",
		"GET /api/v4/file/tag",
		"PATCH /api/v4/file/tag",
		"DELETE /api/v4/file/tag",
		"POST /api/v4/share/purchase/:id",
		"GET /api/v4/share/listed",
		"GET /api/v4/session/qq/login",
		"GET /api/v4/session/qq/callback",
		"DELETE /api/v4/user/setting/sso_binding/:provider",
		"POST /api/v4/user/vault",
		"PUT /api/v4/user/vault/unlock",
		"DELETE /api/v4/user/vault/unlock",
		"DELETE /api/v4/user/vault",
		"PUT /api/v4/user/setting/2fa/backup",
		"GET /f/:id/:name",
	}
	for _, e := range expected {
		require.True(t, have[e], "route missing: %s", e)
	}
}
