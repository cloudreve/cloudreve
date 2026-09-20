package user

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/require"
)

func TestBackup2FAProcess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	logger := logging.NewConsoleLogger(logging.LevelError)
	cfg, err := conf.NewIniConfigProvider(t.TempDir()+"/conf.ini", logger)
	require.NoError(t, err)

	key, err := totp.Generate(totp.GenerateOpts{Issuer: "test", AccountName: "b@example.com"})
	require.NoError(t, err)

	group := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	u := client.User.Create().
		SetEmail("b@example.com").SetNick("b").SetGroup(group).
		SetTwoFactorSecret(key.Secret()).
		SaveX(ctx)
	no2fa := client.User.Create().
		SetEmail("n@example.com").SetNick("n").SetGroup(group).
		SaveX(ctx)

	dep := dependency.NewDependency(
		dependency.WithDbClient(client),
		dependency.WithConfigProvider(cfg),
		dependency.WithLogger(logger),
		dependency.WithSettingProvider(cronSettingProvider{}),
	)

	newCtx := func(u *ent.User) *gin.Context {
		engine := gin.New()
		engine.ContextWithFallback = true
		c := gin.CreateTestContextOnly(httptest.NewRecorder(), engine)
		c.Request = httptest.NewRequest("PUT", "/", nil)
		util.WithValue(c, dependency.DepCtx{}, dep)
		util.WithValue(c, inventory.UserCtx{}, u)
		return c
	}

	appErr := func(err error) serializer.AppError {
		t.Helper()
		var ae serializer.AppError
		require.ErrorAs(t, err, &ae)
		return ae
	}

	// 2FA not enabled -> rejected before TOTP validation.
	_, err = (&Backup2FAService{TwoFACode: "000000"}).Process(newCtx(no2fa))
	require.Error(t, err)
	require.Equal(t, serializer.CodeFeatureNotEnabled, appErr(err).Code)

	// Wrong TOTP -> rejected, no codes stored.
	_, err = (&Backup2FAService{TwoFACode: "000000"}).Process(newCtx(u))
	require.Error(t, err)
	require.Equal(t, serializer.Code2FACodeErr, appErr(err).Code)

	// Valid TOTP -> 10 hyphenated codes, digests persisted.
	valid, err := totp.GenerateCode(key.Secret(), time.Now())
	require.NoError(t, err)
	codes, err := (&Backup2FAService{TwoFACode: valid}).Process(newCtx(u))
	require.NoError(t, err)
	require.Len(t, codes, 10)
	for _, code := range codes {
		require.Len(t, code, 9)
		require.Equal(t, '-', rune(code[4]))
	}
	require.Len(t, client.User.GetX(ctx, u.ID).TwoFactorBackupCodes, 10)
}
