package controllers

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/cloudreve/Cloudreve/v4/service/share"
	"github.com/cloudreve/Cloudreve/v4/service/user"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

// StartLoginAuthn 开始注册WebAuthn登录
func StartLoginAuthn(c *gin.Context) {
	res, err := user.PreparePasskeyLogin(c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

// FinishLoginAuthn 完成注册WebAuthn登录
func FinishLoginAuthn(c *gin.Context) {
	service := ParametersFromContext[*user.FinishPasskeyLoginService](c, user.FinishPasskeyLoginParameterCtx{})
	u, err := service.FinishPasskeyLogin(c)
	if respondErr(c, err) {
		return
	}

	util.WithValue(c, inventory.UserCtx{}, u)
}

// StartRegAuthn 开始注册WebAuthn信息
func StartRegAuthn(c *gin.Context) {
	res, err := user.PreparePasskeyRegister(c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

// FinishRegAuthn 完成注册WebAuthn信息
func FinishRegAuthn(c *gin.Context) {
	service := ParametersFromContext[*user.FinishPasskeyRegisterService](c, user.FinishPasskeyRegisterParameterCtx{})
	res, err := service.FinishPasskeyRegister(c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

// UserDeletePasskey deletes user passkey
func UserDeletePasskey(c *gin.Context) {
	service := ParametersFromContext[*user.DeletePasskeyService](c, user.DeletePasskeyParameterCtx{})
	err := service.DeletePasskey(c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{})
}

// UserLoginValidation validates user login request
func UserLoginValidation(c *gin.Context) {
	service := ParametersFromContext[*user.UserLoginService](c, user.LoginParameterCtx{})
	expectedUser, twoFaSession, err := service.Login(c)
	if respondErr(c, err) {
		return
	}

	if twoFaSession == "" {
		// No 2FA required, proceed
		util.WithValue(c, inventory.UserCtx{}, expectedUser)
		c.Next()
		return
	}

	c.JSON(200, serializer.Response{Code: serializer.CodeNotFullySuccess, Data: twoFaSession})
	c.Abort()
}

// UserLogin2FAValidation validates user OTP code
func UserLogin2FAValidation(c *gin.Context) {
	service := ParametersFromContext[*user.OtpValidationService](c, user.OtpValidationParameterCtx{})
	expectedUser, err := service.Verify2FA(c)
	if respondErr(c, err) {
		return
	}

	util.WithValue(c, inventory.UserCtx{}, expectedUser)
	c.Next()
}

// UserIssueToken generates new token pair for user
func UserIssueToken(c *gin.Context) {
	resp, err := user.IssueToken(c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{
		Data: resp,
	})
}

// UserRefreshToken refreshes token pair for user
func UserRefreshToken(c *gin.Context) {
	service := ParametersFromContext[*user.RefreshTokenService](c, user.RefreshTokenParameterCtx{})
	resp, err := service.Refresh(c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{
		Data: resp,
	})
}

// UserRegister 用户注册
func UserRegister(c *gin.Context) {
	service := ParametersFromContext[*user.UserRegisterService](c, user.RegisterParameterCtx{})
	c.JSON(200, service.Register(c))
}

// UserSendReset 发送密码重设邮件
func UserSendReset(c *gin.Context) {
	service := ParametersFromContext[*user.UserResetEmailService](c, user.UserResetEmailParameterCtx{})
	if err := service.Reset(c); respondErr(c, err) {
		return
	}
	c.JSON(200, serializer.Response{})
}

// UserReset 重设密码
func UserReset(c *gin.Context) {
	service := ParametersFromContext[*user.UserResetService](c, user.UserResetParameterCtx{})
	res, err := service.Reset(c)
	if respondErr(c, err) {
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

// UserActivate 用户激活
func UserActivate(c *gin.Context) {
	c.JSON(200, user.ActivateUser(c))
}

// UserRequestEmailChange starts the self-service email change flow.
func UserRequestEmailChange(c *gin.Context) {
	service := ParametersFromContext[*user.RequestEmailChangeService](c, user.RequestEmailChangeParamCtx{})
	respond(c, service.Request(c), serializer.Response{})
}

// UserActivateEmailChange applies the pending email change from the signed
// confirmation link.
func UserActivateEmailChange(c *gin.Context) {
	respond(c, user.ActivateEmailChange(c), serializer.Response{})
}

// UserSignOut 用户退出登录
func UserSignOut(c *gin.Context) {
	service := ParametersFromContext[*user.RefreshTokenService](c, user.RefreshTokenParameterCtx{})
	res, err := service.Delete(c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{
		Data: res,
	})
}

// UserMe 获取当前登录的用户
func UserMe(c *gin.Context) {
	dep := dependency.FromContext(c)
	c.JSON(200, serializer.Response{
		Data: user.BuildUser(inventory.UserFromContext(c), dep.HashIDEncoder()),
	})
}

// UserGet 获取用户信息
func UserGet(c *gin.Context) {
	u, err := user.GetUser(c)
	if respondErr(c, err) {
		return
	}

	isAnonymous := inventory.IsAnonymousUser(inventory.UserFromContext(c))
	redactLevel := user.RedactLevelUser
	if isAnonymous {
		redactLevel = user.RedactLevelAnonymous
	}
	c.JSON(200, serializer.Response{
		Data: user.BuildUserRedacted(c, u, redactLevel, dependency.FromContext(c).HashIDEncoder()),
	})
}

// UserStorage 获取用户的存储信息
func UserStorage(c *gin.Context) {
	res, err := user.GetUserCapacity(c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{
		Data: res,
	})
}

// UserSetting 获取用户设定
func UserSetting(c *gin.Context) {
	res, err := user.GetUserSettings(c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{
		Data: res,
	})
}

// UploadAvatar 从文件上传头像
func UploadAvatar(c *gin.Context) {
	respond(c, user.UpdateUserAvatar(c), serializer.Response{})
}

// GetUserAvatar 获取用户头像
func GetUserAvatar(c *gin.Context) {
	service := ParametersFromContext[*user.GetAvatarService](c, user.GetAvatarServiceParamsCtx{})
	err := service.Get(c)
	if respondErr(c, err) {
		return
	}
}

// UpdateOption 更改用户设定
func UpdateOption(c *gin.Context) {
	service := ParametersFromContext[*user.PatchUserSetting](c, user.PatchUserSettingParamsCtx{})
	err := service.Patch(c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{})
}

// UserAnnouncement returns the current site announcement unless dismissed.
func UserAnnouncement(c *gin.Context) {
	service := ParametersFromContext[*user.AnnouncementService](c, user.AnnouncementParamCtx{})
	res, err := service.Get(c)
	if respondErr(c, err) {
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

// UserCredit returns the caller's credit balance and active grants.
func UserCredit(c *gin.Context) {
	service := ParametersFromContext[*user.CreditService](c, user.CreditParamCtx{})
	res, err := service.Get(c)
	if respondErr(c, err) {
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

// UserCreditTxns lists the caller's credit ledger.
func UserCreditTxns(c *gin.Context) {
	service := ParametersFromContext[*user.CreditTxnListService](c, user.CreditTxnListParamCtx{})
	res, err := service.List(c)
	if respondErr(c, err) {
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

// UserRedeemGiftCode redeems a gift code for the caller.
func UserRedeemGiftCode(c *gin.Context) {
	service := ParametersFromContext[*user.RedeemGiftCodeService](c, user.RedeemGiftCodeParamCtx{})
	res, err := service.Create(c)
	if respondErr(c, err) {
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

// UserListSkus lists enabled products for the shop page.
func UserListSkus(c *gin.Context) {
	service := ParametersFromContext[*user.SkuListService](c, user.SkuListParamCtx{})
	res, err := service.List(c)
	if respondErr(c, err) {
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

// UserPurchaseSku buys a product with credit points.
func UserPurchaseSku(c *gin.Context) {
	service := ParametersFromContext[*user.PurchaseSkuService](c, user.PurchaseSkuParamCtx{})
	res, err := service.Create(c)
	if respondErr(c, err) {
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

// UserInit2FA 初始化二步验证
func UserInit2FA(c *gin.Context) {
	secret, err := user.Init2FA(c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{
		Data: secret,
	})
}

// UserBackup2FA regenerates one-time 2FA recovery codes. Plaintext codes are
// returned once; only digests are persisted.
func UserBackup2FA(c *gin.Context) {
	service := c.MustGet(user.Backup2FAParameterCtx{}).(*user.Backup2FAService)
	codes, err := service.Process(c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{
		Data: codes,
	})
}

// UserPerformCopySession copy to create new session or refresh current session
func UserPerformCopySession(c *gin.Context) {
	//var service user.CopySessionService
	//if err := c.ShouldBindUri(&service); err == nil {
	//	res := service.Copy(c)
	//	c.JSON(200, res)
	//} else {
	//	c.JSON(200, ErrorResponse(err))
	//}
}

// UserPrepareLogin validates precondition for login
func UserPrepareLogin(c *gin.Context) {
	service := ParametersFromContext[*user.PrepareLoginService](c, user.PrepareLoginParameterCtx{})
	res, err := service.Prepare(c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

// UserSSOLogin redirects the browser to the configured OIDC provider.
func UserSSOLogin(c *gin.Context) {
	service := ParametersFromContext[*user.SSOLoginService](c, user.SSOLoginParameterCtx{})
	service.SSOLogin(c)
}

// UserSSOCallback completes the OIDC flow and redirects to the SPA with a
// one-time ticket.
func UserSSOCallback(c *gin.Context) {
	service := ParametersFromContext[*user.SSOCallbackService](c, user.SSOCallbackParameterCtx{})
	service.SSOCallback(c)
}

// UserQQLogin redirects the browser to the QQ Connect authorization page.
func UserQQLogin(c *gin.Context) {
	service := ParametersFromContext[*user.QQLoginService](c, user.QQLoginParameterCtx{})
	service.Login(c)
}

// UserQQCallback completes the QQ Connect flow and redirects either to the
// SPA ticket handoff or, for link mode, back to the security settings tab.
func UserQQCallback(c *gin.Context) {
	service := ParametersFromContext[*user.QQCallbackService](c, user.QQCallbackParameterCtx{})
	service.Callback(c)
}

// UserWeChatLogin redirects the browser to the WeChat scan authorization page.
func UserWeChatLogin(c *gin.Context) {
	service := ParametersFromContext[*user.WeChatLoginService](c, user.WeChatLoginParameterCtx{})
	service.Login(c)
}

// UserWeChatCallback completes the WeChat flow and redirects either to the
// SPA ticket handoff or, for link mode, back to the security settings tab.
func UserWeChatCallback(c *gin.Context) {
	service := ParametersFromContext[*user.WeChatCallbackService](c, user.WeChatCallbackParameterCtx{})
	service.Callback(c)
}

// UserUnbindSso removes the caller's external-account binding at a provider.
func UserUnbindSso(c *gin.Context) {
	service := ParametersFromContext[*user.SsoUnbindService](c, user.SsoUnbindParameterCtx{})
	err := service.Delete(c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{})
}

// UserVaultSetup enables the caller's private space.
func UserVaultSetup(c *gin.Context) {
	service := ParametersFromContext[*user.VaultSetupService](c, user.VaultSetupParameterCtx{})
	err := service.Setup(c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{})
}

// UserVaultUnlock opens the caller's private-space unlock session.
func UserVaultUnlock(c *gin.Context) {
	service := ParametersFromContext[*user.VaultUnlockService](c, user.VaultUnlockParameterCtx{})
	err := service.Unlock(c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{})
}

// UserVaultLock closes the caller's private-space unlock session.
func UserVaultLock(c *gin.Context) {
	service := ParametersFromContext[*user.VaultUnlockService](c, user.VaultUnlockParameterCtx{})
	err := service.Lock(c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{})
}

// UserVaultDisable turns off the caller's private space.
func UserVaultDisable(c *gin.Context) {
	service := ParametersFromContext[*user.VaultDisableService](c, user.VaultDisableParameterCtx{})
	err := service.Disable(c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{})
}

// UserSSOExchange trades the one-time ticket for a session token pair.
func UserSSOExchange(c *gin.Context) {
	service := ParametersFromContext[*user.SSOExchangeService](c, user.SSOExchangeParameterCtx{})
	res, err := service.SSOExchange(c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

// UserSearch Search user by keyword
func UserSearch(c *gin.Context) {
	service := ParametersFromContext[*user.SearchUserService](c, user.SearchUserParamCtx{})
	u, err := service.Search(c)
	if respondErr(c, err) {
		return
	}

	hasher := dependency.FromContext(c).HashIDEncoder()
	c.JSON(200, serializer.Response{
		Data: lo.Map(u, func(item *ent.User, index int) user.User {
			return user.BuildUserRedacted(c, item, user.RedactLevelUser, hasher)
		}),
	})
}

// ListPublicShare lists all public shares for given user
func ListPublicShare(c *gin.Context) {
	service := ParametersFromContext[*share.ListShareService](c, share.ListShareParamCtx{})
	resp, err := service.ListInUserProfile(c, hashid.FromContext(c))
	if respondErr(c, err) {
		return
	}

	if resp != nil {
		c.JSON(200, serializer.Response{
			Data: resp,
		})
	}
}
