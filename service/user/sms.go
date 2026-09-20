package user

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/activity"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/sms"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

type (
	// SmsSendCodeService dispatches a verification code to a phone number.
	SmsSendCodeService struct {
		Phone string `json:"phone" binding:"required,max=20"`
		Scene string `json:"scene" binding:"required,oneof=login bind reset"`
	}
	SmsSendCodeParameterCtx struct{}

	// SmsLoginService signs in (or provisions) a user by phone + code.
	SmsLoginService struct {
		Phone string `json:"phone" binding:"required,max=20"`
		Code  string `json:"code" binding:"required,len=6"`
	}
	SmsLoginParameterCtx struct{}

	// SmsResetService resets the password of the phone-bound account.
	SmsResetService struct {
		Phone    string `json:"phone" binding:"required,max=20"`
		Code     string `json:"code" binding:"required,len=6"`
		Password string `json:"password" binding:"required,min=6,max=128"`
	}
	SmsResetParameterCtx struct{}

	// SmsBindService binds a verified phone to the signed-in user.
	SmsBindService struct {
		Phone string `json:"phone" binding:"required,max=20"`
		Code  string `json:"code" binding:"required,len=6"`
	}
	SmsBindParameterCtx struct{}

	// SmsUnbindService clears the signed-in user's phone binding.
	SmsUnbindService      struct{}
	SmsUnbindParameterCtx struct{}
)

const (
	smsCodeTTLSeconds  = 300
	smsSendThrottleSec = 60
	smsCodePrefix      = "sms_code_"
	smsThrottlePrefix  = "sms_rl_"
	// smsMailDomain hosts synthetic addresses for phone-provisioned
	// accounts, mirroring the connect.qq/wechat convention.
	smsMailDomain = "sms.local"
)

// Permissive E.164-ish shape: optional country-code plus 5–15 digits.
var phonePattern = regexp.MustCompile(`^\+?[0-9]{5,15}$`)

func normalizePhone(raw string) (string, error) {
	p := strings.TrimSpace(strings.ReplaceAll(raw, " ", ""))
	if !phonePattern.MatchString(p) {
		return "", serializer.NewError(serializer.CodeParamErr, "Invalid phone number", nil)
	}
	return p, nil
}

func smsCodeKey(scene, phone string) string { return smsCodePrefix + scene + "_" + phone }

func smsCode() string {
	// 6-digit numeric code from crypto/rand.
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "000000"
	}
	return fmt.Sprintf("%06d", n.Int64())
}

// verifySmsCode consumes a stored code; single-use on success.
func verifySmsCode(c *gin.Context, dep dependency.Dep, scene, phone, code string) error {
	raw, ok := dep.KV().Get(smsCodeKey(scene, phone))
	stored, isStr := raw.(string)
	if !ok || !isStr || stored != code {
		return serializer.NewError(serializer.CodeSmsCodeErr, "Incorrect or expired verification code", nil)
	}
	if err := dep.KV().Delete(smsCodePrefix, scene+"_"+phone); err != nil {
		return serializer.NewError(serializer.CodeInternalSetting, "Failed to consume verification code", err)
	}
	return nil
}

// Send issues a verification code for the requested scene.
func (service *SmsSendCodeService) Send(c *gin.Context) error {
	dep := dependency.FromContext(c)
	settings := dep.SettingProvider()
	gw := settings.SmsGateway(c)
	if !gw.Enabled || gw.Endpoint == "" {
		return serializer.NewError(serializer.CodeFeatureNotEnabled, "SMS sign-in is not enabled", nil)
	}

	phone, err := normalizePhone(service.Phone)
	if err != nil {
		return err
	}

	userClient := dep.UserClient()
	existing, lookupErr := userClient.GetByPhone(c, phone)

	switch service.Scene {
	case "reset":
		// Consistent with the email reset flow: surface "not found" so the
		// UI can tell the user the number is unbound.
		if lookupErr != nil {
			return serializer.NewError(serializer.CodeUserNotFound, "No account bound to this phone", lookupErr)
		}
	case "bind":
		// The number must be free before a code is worth sending.
		if lookupErr == nil && existing.ID != inventory.UserIDFromContext(c) {
			return serializer.NewError(serializer.CodeConflict, "Phone already bound to another account", nil)
		}
	case "login":
		// Unknown numbers may still proceed when auto-provisioning is on.
		if lookupErr != nil && !gw.RegisterEnabled {
			return serializer.NewError(serializer.CodeUserNotFound, "No account bound to this phone", lookupErr)
		}
	}

	throttleKey := smsThrottlePrefix + phone
	if _, ok := dep.KV().Get(throttleKey); ok {
		return serializer.NewError(serializer.CodeRateLimited, "Verification code already sent, please wait", nil)
	}

	code := smsCode()
	if err := dep.KV().Set(smsCodeKey(service.Scene, phone), code, smsCodeTTLSeconds); err != nil {
		return serializer.NewError(serializer.CodeInternalSetting, "Failed to store verification code", err)
	}

	if err := sms.SendCode(c, dep.RequestClient(), gw, phone, code); err != nil {
		_ = dep.KV().Delete(smsCodePrefix, service.Scene+"_"+phone)
		dep.Logger().Warning("SMS send failed: %s", err)
		return serializer.NewError(serializer.CodeFailedSendSms, "Failed to send verification code", err)
	}
	_ = dep.KV().Set(throttleKey, true, smsSendThrottleSec)

	return nil
}

// Login verifies the code, then signs in or provisions the account — same
// return contract as UserLoginService.Login so UserIssueToken applies
// (including the 2FA continuation).
func (service *SmsLoginService) Login(c *gin.Context) (*ent.User, string, error) {
	dep := dependency.FromContext(c)
	settings := dep.SettingProvider()
	gw := settings.SmsGateway(c)
	if !gw.Enabled || gw.Endpoint == "" {
		return nil, "", serializer.NewError(serializer.CodeFeatureNotEnabled, "SMS sign-in is not enabled", nil)
	}

	phone, err := normalizePhone(service.Phone)
	if err != nil {
		return nil, "", err
	}

	userClient := dep.UserClient()
	ctx := context.WithValue(c, inventory.LoadUserGroup{}, true)
	expectedUser, err := userClient.GetByPhone(ctx, phone)

	var loginFailed error
	switch {
	case err != nil && !ent.IsNotFound(err):
		loginFailed = serializer.NewError(serializer.CodeDBError, "Failed to query user", err)
	case ent.IsNotFound(err):
		if !gw.RegisterEnabled || !settings.RegisterEnabled(c) {
			loginFailed = serializer.NewError(serializer.CodeUserNotFound, "No account bound to this phone", nil)
		} else if expectedUser, err = smsProvisionUser(c, dep, phone); err != nil {
			loginFailed = serializer.NewError(serializer.CodeDBError, "Failed to create account", err)
		} else {
			expectedUser, err = userClient.GetByID(ctx, expectedUser.ID)
			if err != nil {
				loginFailed = serializer.NewError(serializer.CodeDBError, "Failed to load user", err)
			}
		}
	case err == nil:
		expectedUser, err = userClient.LiftExpiredBan(ctx, expectedUser)
		if err != nil {
			loginFailed = serializer.NewError(serializer.CodeDBError, "Failed to lift expired ban", err)
		} else if expectedUser.Status == user.StatusManualBanned || expectedUser.Status == user.StatusSysBanned {
			loginFailed = banError(expectedUser, "This account has been blocked")
		} else if expectedUser.Status == user.StatusInactive {
			loginFailed = serializer.NewError(serializer.CodeUserNotActivated, "This account is not activated", nil)
		} else if ipErr := checkLoginIPWhitelist(c.ClientIP(), inventory.EffectiveGroup(expectedUser)); ipErr != nil {
			loginFailed = ipErr
		}
	}

	if loginFailed != nil {
		activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventUserLoginFailed,
			activity.Extra(map[string]any{"phone": phone}))
		return nil, "", loginFailed
	}

	// Verify only after user resolution — a valid code on a banned account
	// must not be consumed.
	if err := verifySmsCode(c, dep, "login", phone, service.Code); err != nil {
		return nil, "", err
	}

	if expectedUser.TwoFactorSecret != "" {
		twoFaSessionID := uuid.Must(uuid.NewV4())
		dep.KV().Set(fmt.Sprintf("user_2fa_%s", twoFaSessionID), expectedUser.ID, 600)
		return expectedUser, twoFaSessionID.String(), nil
	}

	return expectedUser, "", nil
}

// Reset verifies the code and updates the phone-bound user's password.
func (service *SmsResetService) Reset(c *gin.Context) (*User, error) {
	dep := dependency.FromContext(c)
	settings := dep.SettingProvider()
	if gw := settings.SmsGateway(c); !gw.Enabled || gw.Endpoint == "" {
		return nil, serializer.NewError(serializer.CodeFeatureNotEnabled, "SMS sign-in is not enabled", nil)
	}

	phone, err := normalizePhone(service.Phone)
	if err != nil {
		return nil, err
	}

	userClient := dep.UserClient()
	u, err := userClient.GetByPhone(c, phone)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeUserNotFound, "No account bound to this phone", err)
	}

	if u, err = userClient.LiftExpiredBan(c, u); err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to lift expired ban", err)
	}
	if u.Status == user.StatusManualBanned || u.Status == user.StatusSysBanned {
		return nil, banError(u, "This user is banned")
	}
	if u.Status == user.StatusInactive {
		return nil, serializer.NewError(serializer.CodeUserNotActivated, "This user is not activated", nil)
	}

	if err := verifySmsCode(c, dep, "reset", phone, service.Code); err != nil {
		return nil, err
	}

	u, err = userClient.UpdatePassword(c, u, service.Password)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to update password", err)
	}

	userRes := BuildUser(u, dep.HashIDEncoder())
	return &userRes, nil
}

// Bind attaches the verified phone to the signed-in user.
func (service *SmsBindService) Bind(c *gin.Context) error {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	if u == nil || inventory.IsAnonymousUser(u) {
		return serializer.NewError(serializer.CodeCheckLogin, "Please sign in", nil)
	}
	if gw := dep.SettingProvider().SmsGateway(c); !gw.Enabled {
		return serializer.NewError(serializer.CodeFeatureNotEnabled, "SMS sign-in is not enabled", nil)
	}

	phone, err := normalizePhone(service.Phone)
	if err != nil {
		return err
	}

	if err := verifySmsCode(c, dep, "bind", phone, service.Code); err != nil {
		return err
	}

	if _, err := dep.UserClient().SetPhone(c, u, phone); err != nil {
		return serializer.NewError(serializer.CodeConflict, "Failed to bind phone; it may already be in use", err)
	}
	recordUserEvent(c, dep, u.ID, types.EventLinkAccount, map[string]any{"provider": "sms"})
	return nil
}

// Unbind clears the signed-in user's phone binding.
func (service *SmsUnbindService) Unbind(c *gin.Context) error {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	if u == nil || inventory.IsAnonymousUser(u) {
		return serializer.NewError(serializer.CodeCheckLogin, "Please sign in", nil)
	}
	if _, err := dep.UserClient().SetPhone(c, u, ""); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to unbind phone", err)
	}
	recordUserEvent(c, dep, u.ID, types.EventUnlinkAccount, map[string]any{"provider": "sms"})
	return nil
}

// smsProvisionUser creates a local account for a first-time phone login.
// The phone has no email claim, so a synthetic address under sms.local is
// used and the phone doubles as the initial nickname.
func smsProvisionUser(c *gin.Context, dep dependency.Dep, phone string) (*ent.User, error) {
	return dep.UserClient().Create(c, &inventory.NewUserArgs{
		Email:   fmt.Sprintf("sms_%s@%s", strings.TrimPrefix(phone, "+"), smsMailDomain),
		Nick:    "Mobile user " + phone,
		Status:  user.StatusActive,
		GroupID: dep.SettingProvider().DefaultGroup(c),
		Phone:   phone,
	})
}
