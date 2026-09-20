package user

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type smsSettingProvider struct {
	setting.Provider
	gw              *setting.SmsGateway
	registerEnabled bool
	defaultGroup    int
}

func (p smsSettingProvider) SmsGateway(context.Context) *setting.SmsGateway { return p.gw }
func (p smsSettingProvider) RegisterEnabled(context.Context) bool           { return p.registerEnabled }
func (p smsSettingProvider) DefaultGroup(context.Context) int               { return p.defaultGroup }
func (p smsSettingProvider) AuditLogEnabled(context.Context, int) bool      { return false }
func (p smsSettingProvider) HashIDSalt(context.Context) string              { return "" }

func smsDep(t *testing.T, client *ent.Client, kv cache.Driver, p setting.Provider) dependency.Dep {
	t.Helper()
	logger := logging.NewConsoleLogger(logging.LevelError)
	cfg, err := conf.NewIniConfigProvider(t.TempDir()+"/conf.ini", logger)
	require.NoError(t, err)
	return dependency.NewDependency(
		dependency.WithDbClient(client),
		dependency.WithUserClient(inventory.NewUserClient(client)),
		dependency.WithKV(kv),
		dependency.WithConfigProvider(cfg),
		dependency.WithLogger(logger),
		dependency.WithSettingProvider(p),
	)
}

func smsCtx(dep dependency.Dep, u *ent.User) *gin.Context {
	engine := gin.New()
	engine.ContextWithFallback = true
	c := gin.CreateTestContextOnly(httptest.NewRecorder(), engine)
	c.Request = httptest.NewRequest("POST", "/", nil)
	util.WithValue(c, dependency.DepCtx{}, dep)
	if u != nil {
		util.WithValue(c, inventory.UserCtx{}, u)
	}
	return c
}

func enabledSmsProvider(defaultGroup int) smsSettingProvider {
	return smsSettingProvider{
		gw: &setting.SmsGateway{
			Enabled:         true,
			Endpoint:        "https://sms.example.com/send?phone={phone}&code={code}",
			Method:          "GET",
			RegisterEnabled: true,
		},
		registerEnabled: true,
		defaultGroup:    defaultGroup,
	}
}

func TestNormalizePhone(t *testing.T) {
	for _, ok := range []string{"+8613912345678", "13912345678", "  +1 555 0100 "} {
		p, err := normalizePhone(ok)
		require.NoError(t, err)
		require.NotEmpty(t, p)
	}
	for _, bad := range []string{"", "+", "1234", "abc123456", "+86139123456789012", "phone", "+1-555-0100"} {
		_, err := normalizePhone(bad)
		require.Error(t, err, bad)
	}
}

func TestMaskPhone(t *testing.T) {
	require.Equal(t, "", maskPhone(nil))
	short := "+123"
	require.Equal(t, "", maskPhone(&short))
	full := "+8613912345678"
	require.Equal(t, "+86****78", maskPhone(&full))
}

// TestVerifySmsCode exercises the KV-backed single-use code contract.
func TestVerifySmsCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	kv := cache.NewMemoStore("", nil)
	dep := dependency.NewDependency(dependency.WithKV(kv))
	c := smsCtx(dep, nil)
	phone := "+8613912345678"

	require.NoError(t, kv.Set(smsCodeKey("login", phone), "123456", smsCodeTTLSeconds))

	// Wrong code rejected; stored code survives a failed attempt.
	require.Error(t, verifySmsCode(c, dep, "login", phone, "654321"))
	require.NoError(t, verifySmsCode(c, dep, "login", phone, "123456"))

	// Replay after success is rejected (consumed).
	require.Error(t, verifySmsCode(c, dep, "login", phone, "123456"))

	// Wrong scene does not match.
	require.NoError(t, kv.Set(smsCodeKey("bind", phone), "999999", smsCodeTTLSeconds))
	require.Error(t, verifySmsCode(c, dep, "login", phone, "999999"))
	require.NoError(t, verifySmsCode(c, dep, "bind", phone, "999999"))

	// Non-string KV values are rejected instead of panicking.
	require.NoError(t, kv.Set(smsCodeKey("reset", phone), 123456, smsCodeTTLSeconds))
	require.Error(t, verifySmsCode(c, dep, "reset", phone, "123456"))
}

// TestSmsSendDisabled ensures every entry point refuses when the gateway is off.
func TestSmsSendDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	group := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	u := client.User.Create().SetEmail("u@example.com").SetNick("u").SetGroup(group).
		SetStatus(user.StatusActive).SetSettings(&types.UserSetting{}).SaveX(ctx)

	kv := cache.NewMemoStore("", nil)
	dep := smsDep(t, client, kv, smsSettingProvider{
		gw:           &setting.SmsGateway{Enabled: false},
		defaultGroup: group.ID,
	})

	err := (&SmsSendCodeService{Phone: "+8613912345678", Scene: "login"}).Send(smsCtx(dep, nil))
	requireAppCode(t, err, serializer.CodeFeatureNotEnabled)

	_, _, err = (&SmsLoginService{Phone: "+8613912345678", Code: "123456"}).Login(smsCtx(dep, nil))
	requireAppCode(t, err, serializer.CodeFeatureNotEnabled)

	_, err = (&SmsResetService{Phone: "+8613912345678", Code: "123456", Password: "secret1"}).Reset(smsCtx(dep, nil))
	requireAppCode(t, err, serializer.CodeFeatureNotEnabled)

	err = (&SmsBindService{Phone: "+8613912345678", Code: "123456"}).Bind(smsCtx(dep, u))
	requireAppCode(t, err, serializer.CodeFeatureNotEnabled)
}

// TestSmsSendFailures covers validation, unknown-phone and send-failure paths.
// The endpoint intentionally points at an unroutable host so the gateway call
// fails after the code is staged; the staged code must be cleaned up.
func TestSmsSendFailures(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	group := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	client.User.Create().SetEmail("u@example.com").SetNick("u").SetGroup(group).
		SetStatus(user.StatusActive).SetSettings(&types.UserSetting{}).
		SetPhone("+8613912345678").SaveX(ctx)

	kv := cache.NewMemoStore("", nil)
	dep := smsDep(t, client, kv, smsSettingProvider{
		gw: &setting.SmsGateway{
			Enabled:         true,
			Endpoint:        "http://127.0.0.1:1/sms?phone={phone}&code={code}",
			Method:          "GET",
			RegisterEnabled: true,
		},
		registerEnabled: true,
		defaultGroup:    group.ID,
	})

	// Malformed phone.
	err := (&SmsSendCodeService{Phone: "not-a-phone", Scene: "login"}).Send(smsCtx(dep, nil))
	requireAppCode(t, err, serializer.CodeParamErr)

	// Reset for an unbound number → user not found.
	err = (&SmsSendCodeService{Phone: "+8610000000000", Scene: "reset"}).Send(smsCtx(dep, nil))
	requireAppCode(t, err, serializer.CodeUserNotFound)

	// Gateway send fails; the staged code must not linger.
	err = (&SmsSendCodeService{Phone: "+8613912345678", Scene: "login"}).Send(smsCtx(dep, nil))
	requireAppCode(t, err, serializer.CodeFailedSendSms)
	_, ok := kv.Get(smsCodeKey("login", "+8613912345678"))
	require.False(t, ok)
	_, ok = kv.Get(smsThrottlePrefix + "+8613912345678")
	require.False(t, ok)
}

func TestSmsLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	group := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	u := client.User.Create().SetEmail("u@example.com").SetNick("u").SetGroup(group).
		SetStatus(user.StatusActive).SetSettings(&types.UserSetting{}).
		SetPhone("+8613912345678").SaveX(ctx)

	kv := cache.NewMemoStore("", nil)
	dep := smsDep(t, client, kv, enabledSmsProvider(group.ID))

	// Unknown code → error, user not returned.
	_, _, err := (&SmsLoginService{Phone: "+8613912345678", Code: "000000"}).Login(smsCtx(dep, nil))
	requireAppCode(t, err, serializer.CodeSmsCodeErr)

	// Correct code → signed in.
	require.NoError(t, kv.Set(smsCodeKey("login", "+8613912345678"), "123456", smsCodeTTLSeconds))
	got, twoFA, err := (&SmsLoginService{Phone: "+8613912345678", Code: "123456"}).Login(smsCtx(dep, nil))
	require.NoError(t, err)
	require.Equal(t, u.ID, got.ID)
	require.Empty(t, twoFA)

	// Auto-provision a new account for an unknown phone.
	require.NoError(t, kv.Set(smsCodeKey("login", "+15550199"), "654321", smsCodeTTLSeconds))
	got, _, err = (&SmsLoginService{Phone: "+15550199", Code: "654321"}).Login(smsCtx(dep, nil))
	require.NoError(t, err)
	require.Equal(t, "+15550199", *got.Phone)
	require.Equal(t, "sms_15550199@sms.local", got.Email)

	// Unknown phone with registration disabled → user not found, code not
	// consumed (verify runs after user resolution).
	disabled := smsSettingProvider{
		gw:              &setting.SmsGateway{Enabled: true, Endpoint: "https://x.test", RegisterEnabled: false},
		registerEnabled: true,
		defaultGroup:    group.ID,
	}
	dep2 := smsDep(t, client, kv, disabled)
	require.NoError(t, kv.Set(smsCodeKey("login", "+15550000"), "111111", smsCodeTTLSeconds))
	_, _, err = (&SmsLoginService{Phone: "+15550000", Code: "111111"}).Login(smsCtx(dep2, nil))
	requireAppCode(t, err, serializer.CodeUserNotFound)
	_, ok := kv.Get(smsCodeKey("login", "+15550000"))
	require.True(t, ok, "code must survive a failed login")

	// Banned user is rejected and the code is not consumed.
	banned := client.User.Create().SetEmail("b@example.com").SetNick("b").SetGroup(group).
		SetStatus(user.StatusManualBanned).SetSettings(&types.UserSetting{}).
		SetPhone("+8610000000001").SaveX(ctx)
	require.NoError(t, kv.Set(smsCodeKey("login", "+8610000000001"), "222222", smsCodeTTLSeconds))
	_, _, err = (&SmsLoginService{Phone: "+8610000000001", Code: "222222"}).Login(smsCtx(dep, nil))
	require.Error(t, err)
	_, ok = kv.Get(smsCodeKey("login", "+8610000000001"))
	require.True(t, ok)
	_ = banned
}

// TestSmsLogin2FA checks the 2FA continuation mirrors password login.
func TestSmsLogin2FA(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	group := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	u := client.User.Create().SetEmail("t@example.com").SetNick("t").SetGroup(group).
		SetStatus(user.StatusActive).SetSettings(&types.UserSetting{}).
		SetPhone("+8613912345678").SetTwoFactorSecret("JBSWY3DPEHPK3PXP").SaveX(ctx)

	kv := cache.NewMemoStore("", nil)
	dep := smsDep(t, client, kv, enabledSmsProvider(group.ID))

	require.NoError(t, kv.Set(smsCodeKey("login", "+8613912345678"), "123456", smsCodeTTLSeconds))
	got, session, err := (&SmsLoginService{Phone: "+8613912345678", Code: "123456"}).Login(smsCtx(dep, nil))
	require.NoError(t, err)
	require.Equal(t, u.ID, got.ID)
	require.NotEmpty(t, session)

	// The session ticket maps back to the user for the 2FA endpoint.
	raw, ok := kv.Get("user_2fa_" + session)
	require.True(t, ok)
	require.Equal(t, u.ID, raw.(int))
}

func TestSmsReset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	group := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	u := client.User.Create().SetEmail("r@example.com").SetNick("r").SetGroup(group).
		SetStatus(user.StatusActive).SetSettings(&types.UserSetting{}).
		SetPhone("+8613912345678").SaveX(ctx)

	kv := cache.NewMemoStore("", nil)
	dep := smsDep(t, client, kv, enabledSmsProvider(group.ID))

	// Unbound phone → not found.
	_, err := (&SmsResetService{Phone: "+8610000000099", Code: "123456", Password: "newpass1"}).Reset(smsCtx(dep, nil))
	requireAppCode(t, err, serializer.CodeUserNotFound)

	// Wrong code → rejected, password untouched.
	require.NoError(t, kv.Set(smsCodeKey("reset", "+8613912345678"), "123456", smsCodeTTLSeconds))
	_, err = (&SmsResetService{Phone: "+8613912345678", Code: "000000", Password: "newpass1"}).Reset(smsCtx(dep, nil))
	requireAppCode(t, err, serializer.CodeSmsCodeErr)

	// Correct code → password updated; old password no longer matches.
	res, err := (&SmsResetService{Phone: "+8613912345678", Code: "123456", Password: "newpass1"}).Reset(smsCtx(dep, nil))
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NoError(t, inventory.CheckPassword(client.User.GetX(ctx, u.ID), "newpass1"))
}

func TestSmsBindUnbind(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	group := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	u := client.User.Create().SetEmail("a@example.com").SetNick("a").SetGroup(group).
		SetStatus(user.StatusActive).SetSettings(&types.UserSetting{}).SaveX(ctx)
	other := client.User.Create().SetEmail("o@example.com").SetNick("o").SetGroup(group).
		SetStatus(user.StatusActive).SetSettings(&types.UserSetting{}).
		SetPhone("+8613912345678").SaveX(ctx)

	kv := cache.NewMemoStore("", nil)
	dep := smsDep(t, client, kv, enabledSmsProvider(group.ID))

	// Binding a phone already claimed by another account is rejected at the
	// uniqueness constraint even if a code was issued.
	require.NoError(t, kv.Set(smsCodeKey("bind", "+8613912345678"), "123456", smsCodeTTLSeconds))
	err := (&SmsBindService{Phone: "+8613912345678", Code: "123456"}).Bind(smsCtx(dep, u))
	require.Error(t, err)
	require.Nil(t, client.User.GetX(ctx, u.ID).Phone)

	// Successful bind.
	require.NoError(t, kv.Set(smsCodeKey("bind", "+15550077"), "999999", smsCodeTTLSeconds))
	require.NoError(t, (&SmsBindService{Phone: "+15550077", Code: "999999"}).Bind(smsCtx(dep, u)))
	require.Equal(t, "+15550077", *client.User.GetX(ctx, u.ID).Phone)

	// Unbind clears the phone.
	require.NoError(t, (&SmsUnbindService{}).Unbind(smsCtx(dep, u)))
	require.Nil(t, client.User.GetX(ctx, u.ID).Phone)
	require.Equal(t, "+8613912345678", *client.User.GetX(ctx, other.ID).Phone)

	// Anonymous bind is rejected.
	err = (&SmsBindService{Phone: "+15550077", Code: "999999"}).Bind(smsCtx(dep, nil))
	requireAppCode(t, err, serializer.CodeCheckLogin)
}

func requireAppCode(t *testing.T, err error, code int) {
	t.Helper()
	require.Error(t, err)
	var ae serializer.AppError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, code, ae.Code)
}
