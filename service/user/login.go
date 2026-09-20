package user

import (
	"context"
	"fmt"
	"net/netip"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/activity"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
	"github.com/cloudreve/Cloudreve/v4/pkg/email"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/pquerna/otp/totp"
)

// LoginParameterCtx define key fore UserLoginService
type LoginParameterCtx struct{}

// UserLoginService 管理用户登录的服务
type UserLoginService struct {
	UserName string `form:"email" json:"email" binding:"required,email"`
	Password string `form:"password" json:"password" binding:"required,min=4,max=128"`
}

type (
	// UserResetService 密码重设服务
	UserResetService struct {
		Password string `form:"password" json:"password" binding:"required,min=6,max=128"`
		Secret   string `json:"secret" binding:"required"`
	}
	UserResetParameterCtx struct{}
)

// Reset 重设密码
func (service *UserResetService) Reset(c *gin.Context) (*User, error) {
	dep := dependency.FromContext(c)
	userClient := dep.UserClient()
	kv := dep.KV()
	uid := hashid.FromContext(c)

	resetSession, ok := kv.Get(fmt.Sprintf("user_reset_%d", uid))
	if !ok || resetSession.(string) != service.Secret {
		return nil, serializer.NewError(serializer.CodeTempLinkExpired, "Link is expired", nil)
	}

	if err := kv.Delete(fmt.Sprintf("user_reset_%d", uid)); err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to delete reset session", err)
	}

	u, err := userClient.GetActiveByID(c, uid)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeUserNotFound, "User not found", err)
	}

	u, err = userClient.UpdatePassword(c, u, service.Password)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to update password", err)
	}

	userRes := BuildUser(u, dep.HashIDEncoder())
	return &userRes, nil
}

type (
	// UserResetEmailService 发送密码重设邮件服务
	UserResetEmailService struct {
		UserName string `form:"email" json:"email" binding:"required,email"`
		Language string `form:"language" json:"language"`
	}
	UserResetEmailParameterCtx struct{}
)

const userResetPrefix = "user_reset_"

// Reset 发送密码重设邮件
func (service *UserResetEmailService) Reset(c *gin.Context) error {
	dep := dependency.FromContext(c)
	userClient := dep.UserClient()

	u, err := userClient.GetByEmail(c, service.UserName)
	if err != nil {
		return serializer.NewError(serializer.CodeUserNotFound, "User not found", err)
	}

	if u, err = userClient.LiftExpiredBan(c, u); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to lift expired ban", err)
	}

	if u.Status == user.StatusManualBanned || u.Status == user.StatusSysBanned {
		return banError(u, "This user is banned")
	}

	if u.Status == user.StatusInactive {
		return serializer.NewError(serializer.CodeUserNotActivated, "This user is not activated", nil)
	}

	secret := util.RandStringRunesCrypto(32)
	if err := dep.KV().Set(fmt.Sprintf("%s%d", userResetPrefix, u.ID), secret, 3600); err != nil {
		return serializer.NewError(serializer.CodeInternalSetting, "Failed to create reset session", err)
	}

	base := dep.SettingProvider().SiteURL(c)
	resetUrl := routes.MasterUserResetUrl(base)
	queries := resetUrl.Query()
	queries.Add("id", hashid.EncodeUserID(dep.HashIDEncoder(), u.ID))
	queries.Add("secret", secret)
	resetUrl.RawQuery = queries.Encode()

	title, body, err := email.NewResetEmail(c, dep.SettingProvider(), u, resetUrl.String())
	if err != nil {
		return serializer.NewError(serializer.CodeFailedSendEmail, "Failed to send activation email", err)
	}

	if err := dep.EmailClient(c).Send(c, u.Email, title, body); err != nil {
		return serializer.NewError(serializer.CodeFailedSendEmail, "Failed to send activation email", err)
	}
	activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventEmailSent,
		activity.Actor(u.ID), activity.Extra(map[string]any{"kind": "password_reset"}))

	return nil
}

// Login 用户登录函数
func (service *UserLoginService) Login(c *gin.Context) (*ent.User, string, error) {
	dep := dependency.FromContext(c)
	userClient := dep.UserClient()

	ctx := context.WithValue(c, inventory.LoadUserGroup{}, true)
	expectedUser, err := userClient.GetByEmail(ctx, service.UserName)
	if err == nil {
		expectedUser, err = userClient.LiftExpiredBan(ctx, expectedUser)
	}

	// 一系列校验
	if err != nil {
		err = serializer.NewError(serializer.CodeInvalidPassword, "Incorrect password or email address", err)
	} else if checkErr := inventory.CheckPassword(expectedUser, service.Password); checkErr != nil {
		err = serializer.NewError(serializer.CodeInvalidPassword, "Incorrect password or email address", err)
	} else if expectedUser.Status == user.StatusManualBanned || expectedUser.Status == user.StatusSysBanned {
		err = banError(expectedUser, "This account has been blocked")
	} else if expectedUser.Status == user.StatusInactive {
		err = serializer.NewError(serializer.CodeUserNotActivated, "This account is not activated", nil)
	} else if ipErr := checkLoginIPWhitelist(c.ClientIP(), expectedUser.Edges.Group); ipErr != nil {
		err = ipErr
	}

	if err != nil {
		activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventUserLoginFailed,
			activity.Extra(map[string]any{"email": service.UserName}))
		return nil, "", err
	}

	if expectedUser.TwoFactorSecret != "" {
		twoFaSessionID := uuid.Must(uuid.NewV4())
		dep.KV().Set(fmt.Sprintf("user_2fa_%s", twoFaSessionID), expectedUser.ID, 600)
		return expectedUser, twoFaSessionID.String(), nil
	}

	return expectedUser, "", nil
}

type (
	LoginLogCtx struct{}
)

func IssueToken(c *gin.Context) (*BuiltinLoginResponse, error) {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)

	// Best-effort last-login stamp; a failed update must not block sign-in.
	if err := dep.UserClient().UpdateLastLogin(c, u.ID); err != nil {
		dep.Logger().Warning("Failed to update last_login for user %d: %s", u.ID, err)
	}

	token, err := dep.TokenAuth().Issue(c, &auth.IssueTokenArgs{
		User:        u,
		RootTokenID: nil,
	})
	if err != nil {
		return nil, serializer.NewError(serializer.CodeEncryptError, "Failed to issue token pair", err)
	}

	activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventUserLogin)

	return &BuiltinLoginResponse{
		User:  BuildUser(u, dep.HashIDEncoder()),
		Token: *token,
	}, nil
}

// RefreshTokenParameterCtx define key fore RefreshTokenService
type RefreshTokenParameterCtx struct{}

// RefreshTokenService refresh token service
type RefreshTokenService struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (s *RefreshTokenService) Refresh(c *gin.Context) (*auth.Token, error) {
	dep := dependency.FromContext(c)
	claims, _ := dep.TokenAuth().Claims(c, s.RefreshToken)
	token, err := dep.TokenAuth().Refresh(c, s.RefreshToken)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeCredentialInvalid, "Failed to issue token pair", err)
	}

	opts := []activity.Opt{}
	if claims != nil {
		if uid, err := dep.HashIDEncoder().Decode(claims.Subject, hashid.UserID); err == nil {
			opts = append(opts, activity.Actor(uid))
		}
	}
	activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventUserTokenRefresh, opts...)

	return token, nil
}

func (s *RefreshTokenService) Delete(c *gin.Context) (string, error) {
	dep := dependency.FromContext(c)
	claims, err := dep.TokenAuth().Claims(c, s.RefreshToken)
	if err != nil {
		return "", serializer.NewError(serializer.CodeCredentialInvalid, "Failed to parse token", err)
	}

	// Block root token
	if claims.RootTokenID != nil {
		tokenSettings := dep.SettingProvider().TokenAuth(c)
		dep.KV().Set(fmt.Sprintf("%s%s", auth.RevokeTokenPrefix, claims.RootTokenID.String()), true, int(tokenSettings.RefreshTokenTTL.Seconds()+10))
	}

	return "", nil
}

type (
	OtpValidationParameterCtx struct{}
	OtpValidationService      struct {
		OTP       string `json:"otp" binding:"required"`
		SessionID string `json:"session_id" binding:"required"`
	}
)

// Login 用户登录函数
func (service *OtpValidationService) Verify2FA(c *gin.Context) (*ent.User, error) {
	dep := dependency.FromContext(c)
	kv := dep.KV()

	sessionRaw, ok := kv.Get(fmt.Sprintf("user_2fa_%s", service.SessionID))
	if !ok {
		return nil, serializer.NewError(serializer.CodeNotFound, "Session not found", nil)
	}

	uid := sessionRaw.(int)
	ctx := context.WithValue(c, inventory.LoadUserGroup{}, true)
	expectedUser, err := dep.UserClient().GetByID(ctx, uid)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "User not found", err)
	}

	if expectedUser.TwoFactorSecret != "" {
		if !totp.Validate(service.OTP, expectedUser.TwoFactorSecret) {
			// TOTP failed — fall back to one-time recovery codes.
			ok, err := dep.UserClient().ConsumeTwoFABackupCode(c, expectedUser, service.OTP)
			if err != nil {
				return nil, serializer.NewError(serializer.CodeDBError, "Failed to verify recovery code", err)
			}
			if !ok {
				return nil, serializer.NewError(serializer.Code2FACodeErr, "Incorrect 2FA code", nil)
			}
		}
	}

	kv.Delete("user_2fa_", service.SessionID)
	return expectedUser, nil
}

type (
	PrepareLoginParameterCtx struct{}
	PrepareLoginService      struct {
		Email string `form:"email" binding:"required,email"`
	}
)

func (service *PrepareLoginService) Prepare(c *gin.Context) (*PrepareLoginResponse, error) {
	dep := dependency.FromContext(c)
	ctx := context.WithValue(c, inventory.LoadUserPasskey{}, true)
	expectedUser, err := dep.UserClient().GetByEmail(ctx, service.Email)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "User not found", err)
	}

	return &PrepareLoginResponse{
		WebAuthnEnabled: len(expectedUser.Edges.Passkey) > 0,
		PasswordEnabled: expectedUser.Password != "",
	}, nil
}

// checkLoginIPWhitelist enforces a group's login IP whitelist. Entries are
// exact IPs or CIDR ranges; an empty or nil list allows all addresses.
// Malformed entries are ignored so a bad admin value cannot lock everyone out.
func checkLoginIPWhitelist(clientIP string, group *ent.Group) error {
	if group == nil || group.Settings == nil || len(group.Settings.LoginIPWhitelist) == 0 {
		return nil
	}

	addr, err := netip.ParseAddr(clientIP)
	if err != nil {
		return serializer.NewError(serializer.CodeNoPermissionErr, "Cannot determine client IP", err)
	}

	for _, entry := range group.Settings.LoginIPWhitelist {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if prefix, perr := netip.ParsePrefix(entry); perr == nil {
			if prefix.Contains(addr) {
				return nil
			}
			continue
		}
		if entryAddr, aerr := netip.ParseAddr(entry); aerr == nil && entryAddr == addr {
			return nil
		}
	}

	return serializer.NewError(serializer.CodeNoPermissionErr, "Login from this IP address is not allowed", nil)
}
