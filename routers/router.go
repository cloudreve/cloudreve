package routers

import (
	"net/http"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/middleware"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/downloader/slave"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/webdav"
	"github.com/cloudreve/Cloudreve/v4/routers/controllers"
	abusesvc "github.com/cloudreve/Cloudreve/v4/service/abuse"
	adminsvc "github.com/cloudreve/Cloudreve/v4/service/admin"
	"github.com/cloudreve/Cloudreve/v4/service/basic"
	"github.com/cloudreve/Cloudreve/v4/service/explorer"
	"github.com/cloudreve/Cloudreve/v4/service/node"
	"github.com/cloudreve/Cloudreve/v4/service/oauth"
	"github.com/cloudreve/Cloudreve/v4/service/setting"
	sharesvc "github.com/cloudreve/Cloudreve/v4/service/share"
	usersvc "github.com/cloudreve/Cloudreve/v4/service/user"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

// InitRouter 初始化路由
func InitRouter(dep dependency.Dep) *gin.Engine {
	l := dep.Logger()
	if dep.ConfigProvider().System().Mode == conf.MasterMode {
		l.Info("Current running mode: Master.")
		return initMasterRouter(dep)
	}

	l.Info("Current running mode: Slave.")
	return initSlaveRouter(dep)
}

func newGinEngine(dep dependency.Dep) *gin.Engine {
	r := gin.New()
	r.ContextWithFallback = true
	r.Use(gin.Recovery())
	r.Use(middleware.InitializeHandling(dep))
	if dep.ConfigProvider().System().Mode == conf.SlaveMode {
		r.Use(middleware.InitializeHandlingSlave())
	}
	r.Use(middleware.Logging())
	return r
}

func initSlaveFileRouter(v4 *gin.RouterGroup) {
	// Upload related, no signature required under this router group
	upload := v4.Group("upload")
	{
		// 上传分片
		upload.POST(":sessionId",
			controllers.FromUri[explorer.UploadService](explorer.UploadParameterCtx{}),
			controllers.SlaveUpload,
		)
		// 创建上传会话上传
		upload.PUT("",
			controllers.FromJSON[explorer.SlaveCreateUploadSessionService](explorer.SlaveCreateUploadSessionParamCtx{}),
			controllers.SlaveGetUploadSession,
		)
		// 删除上传会话
		upload.DELETE(":sessionId",
			controllers.FromUri[explorer.SlaveDeleteUploadSessionService](explorer.SlaveDeleteUploadSessionParamCtx{}),
			controllers.SlaveDeleteUploadSession)
	}
	file := v4.Group("file")
	{
		// Get entity content for preview/download
		file.GET("content/:nodeId/:src/:speed/:name",
			middleware.Sandbox(),
			controllers.FromUri[explorer.EntityDownloadService](explorer.EntityDownloadParameterCtx{}),
			controllers.SlaveServeEntity,
		)
		file.HEAD("content/:nodeId/:src/:speed/:name",
			controllers.FromUri[explorer.EntityDownloadService](explorer.EntityDownloadParameterCtx{}),
			controllers.SlaveServeEntity,
		)
		// Get media metadata
		file.GET("meta/:src/:ext",
			controllers.FromUri[explorer.SlaveMetaService](explorer.SlaveMetaParamCtx{}),
			controllers.SlaveMeta,
		)
		// Get thumbnail
		file.GET("thumb/:src/:ext",
			controllers.FromUri[explorer.SlaveThumbService](explorer.SlaveThumbParamCtx{}),
			controllers.SlaveThumb,
		)
		// 删除文件
		file.DELETE("",
			controllers.FromJSON[explorer.SlaveDeleteFileService](explorer.SlaveDeleteFileParamCtx{}),
			controllers.SlaveDelete)
		// 列出文件
		file.GET("list",
			controllers.FromQuery[explorer.SlaveListService](explorer.SlaveListParamCtx{}),
			controllers.SlaveList,
		)
	}
}

// initSlaveRouter 初始化从机模式路由
func initSlaveRouter(dep dependency.Dep) *gin.Engine {
	r := newGinEngine(dep)
	// 跨域相关
	initCORS(dep.Logger(), dep.ConfigProvider(), r)
	v4 := r.Group(constants.APIPrefix + "/slave")
	// 鉴权中间件
	v4.Use(middleware.SignRequired(dep.GeneralAuth()))
	// 禁止缓存
	v4.Use(middleware.CacheControl())

	/*
		路由
	*/
	{
		// Ping
		v4.POST("ping",
			controllers.FromJSON[adminsvc.SlavePingService](adminsvc.SlavePingParameterCtx{}),
			controllers.SlavePing,
		)
		// // 测试 Aria2 RPC 连接
		// v4.POST("ping/aria2", controllers.AdminTestAria2)
		initSlaveFileRouter(v4)

		// 离线下载
		download := v4.Group("download")
		{
			// 创建离线下载任务
			download.POST("task",
				controllers.FromJSON[slave.CreateSlaveDownload](node.CreateSlaveDownloadTaskParamCtx{}),
				middleware.PrepareSlaveDownloader(dep, node.CreateSlaveDownloadTaskParamCtx{}),
				controllers.SlaveDownloadTaskCreate)
			// 获取任务状态
			download.POST("status",
				controllers.FromJSON[slave.GetSlaveDownload](node.GetSlaveDownloadTaskParamCtx{}),
				middleware.PrepareSlaveDownloader(dep, node.GetSlaveDownloadTaskParamCtx{}),
				controllers.SlaveDownloadTaskStatus)
			// 取消离线下载任务
			download.POST("cancel",
				controllers.FromJSON[slave.CancelSlaveDownload](node.CancelSlaveDownloadTaskParamCtx{}),
				middleware.PrepareSlaveDownloader(dep, node.CancelSlaveDownloadTaskParamCtx{}),
				controllers.SlaveCancelDownloadTask)
			// 选取任务文件
			download.POST("select",
				controllers.FromJSON[slave.SetSlaveFilesToDownload](node.SelectSlaveDownloadFilesParamCtx{}),
				middleware.PrepareSlaveDownloader(dep, node.SelectSlaveDownloadFilesParamCtx{}),
				controllers.SlaveSelectFilesToDownload)
			// 测试下载器连接
			download.POST("test",
				controllers.FromJSON[slave.TestSlaveDownload](node.TestSlaveDownloadParamCtx{}),
				middleware.PrepareSlaveDownloader(dep, node.TestSlaveDownloadParamCtx{}),
				controllers.SlaveTestDownloader,
			)
		}

		// 异步任务
		task := v4.Group("task")
		{
			task.PUT("",
				controllers.FromJSON[cluster.CreateSlaveTask](node.CreateSlaveTaskParamCtx{}),
				controllers.SlaveCreateTask)
			task.GET(":id",
				controllers.FromUri[node.GetSlaveTaskService](node.GetSlaveTaskParamCtx{}),
				controllers.SlaveGetTask)
			task.POST("cleanup",
				controllers.FromJSON[cluster.FolderCleanup](node.FolderCleanupParamCtx{}),
				controllers.SlaveCleanupFolder)
		}
	}
	return r
}

// initCORS 初始化跨域配置
func initCORS(l logging.Logger, config conf.ConfigProvider, router *gin.Engine) {
	c := config.Cors()
	if c.AllowOrigins[0] != "UNSET" {
		router.Use(cors.New(cors.Config{
			AllowOrigins:     c.AllowOrigins,
			AllowMethods:     c.AllowMethods,
			AllowHeaders:     c.AllowHeaders,
			AllowCredentials: c.AllowCredentials,
			ExposeHeaders:    c.ExposeHeaders,
		}))
		return
	}

	// slave模式下未启动跨域的警告
	if config.System().Mode == conf.SlaveMode {
		l.Warning("You are running Cloudreve as slave node, if you are using slave storage policy, please enable CORS feature in config file, otherwise file cannot be uploaded from Master basic.")
	}
}

// initMasterRouter 初始化主机模式路由
func initMasterRouter(dep dependency.Dep) *gin.Engine {
	r := newGinEngine(dep)
	// 跨域相关
	initCORS(dep.Logger(), dep.ConfigProvider(), r) // Done

	/*
		静态资源
	*/
	r.Use(gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedPaths([]string{"/api/"})))
	r.Use(middleware.SharePreview(dep))
	r.GET(".well-known/openid-configuration", controllers.OpenIDConfiguration)
	r.Use(middleware.FrontendFileHandler(dep))
	r.GET("manifest.json", controllers.Manifest)

	noAuth := r.Group(constants.APIPrefix)
	wopi := noAuth.Group("file/wopi", middleware.HashID(hashid.FileID), middleware.ViewerSessionValidation())
	{
		// 获取文件信息
		wopi.GET(":id", controllers.CheckFileInfo)
		// 获取文件内容
		wopi.GET(":id/contents", controllers.GetFile)
		// 更新文件内容
		wopi.POST(":id/contents", controllers.PutFile)
		// 通用文件操作
		wopi.POST(":id", controllers.ModifyFile)
	}

	v4 := r.Group(constants.APIPrefix)

	/*
		中间件
	*/
	v4.Use(middleware.Session(dep)) // Done

	// 用户会话
	v4.Use(middleware.CurrentUser())

	// 禁止缓存
	v4.Use(middleware.CacheControl()) // Done

	/*
		路由
	*/
	{
		// Redirect file source link
		source := r.Group("f")
		source.Use(middleware.ContentCORS())
		{
			source.OPTIONS("*option", middleware.ContentCORS())
			source.GET(":id/:name",
				middleware.HashID(hashid.SourceLinkID),
				controllers.AnonymousPermLink(false))
			source.GET("d/:id/:name",
				middleware.HashID(hashid.SourceLinkID),
				controllers.AnonymousPermLink(true))
		}

		shareShort := r.Group("s")
		{
			shareShort.GET(":id",
				controllers.FromUri[sharesvc.ShortLinkRedirectService](sharesvc.ShortLinkRedirectParamCtx{}),
				controllers.ShareRedirect,
			)
			shareShort.GET(":id/:password",
				controllers.FromUri[sharesvc.ShortLinkRedirectService](sharesvc.ShortLinkRedirectParamCtx{}),
				controllers.ShareRedirect,
			)
		}

		// 全局设置相关
		site := v4.Group("site")
		{
			// 测试用路由
			site.GET("ping", controllers.Ping)
			// 验证码
			site.GET("captcha", controllers.Captcha)
			// 站点全局配置
			site.GET("config/:section",
				controllers.FromUri[basic.GetSettingService](basic.GetSettingParamCtx{}),
				controllers.SiteConfig,
			)
		}

		// User authentication
		session := v4.Group("session")
		{
			token := session.Group("token")
			// Token based authentication
			{
				// 用户登录
				token.POST("",
					middleware.RateLimitByIP("login", 10, time.Minute),
					middleware.CaptchaRequired(func(c *gin.Context) bool {
						return dep.SettingProvider().LoginCaptchaEnabled(c)
					}),
					controllers.FromJSON[usersvc.UserLoginService](usersvc.LoginParameterCtx{}),
					controllers.UserLoginValidation,
					controllers.UserIssueToken,
				)
				// 2-factor authentication
				token.POST("2fa",
					middleware.RateLimitByIP("login_2fa", 10, time.Minute),
					controllers.FromJSON[usersvc.OtpValidationService](usersvc.OtpValidationParameterCtx{}),
					controllers.UserLogin2FAValidation,
					controllers.UserIssueToken,
				)
				token.POST("refresh",
					middleware.RateLimitByIP("token_refresh", 30, time.Minute),
					middleware.RequiredScopes(types.ScopeOfflineAccess),
					controllers.FromJSON[usersvc.RefreshTokenService](usersvc.RefreshTokenParameterCtx{}),
					controllers.UserRefreshToken,
				)
				token.DELETE("",
					controllers.FromJSON[usersvc.RefreshTokenService](usersvc.RefreshTokenParameterCtx{}),
					controllers.UserSignOut,
				)
			}

			// Prepare login
			session.GET("prepare",
				controllers.FromQuery[usersvc.PrepareLoginService](usersvc.PrepareLoginParameterCtx{}),
				controllers.UserPrepareLogin,
			)

			// Inbound OIDC single sign-on
			ssoRouter := session.Group("sso")
			{
				ssoRouter.GET("",
					controllers.FromQuery[usersvc.SSOLoginService](usersvc.SSOLoginParameterCtx{}),
					controllers.UserSSOLogin,
				)
				ssoRouter.GET("callback",
					controllers.FromQuery[usersvc.SSOCallbackService](usersvc.SSOCallbackParameterCtx{}),
					controllers.UserSSOCallback,
				)
				ssoRouter.POST("exchange",
					middleware.RateLimitByIP("sso_exchange", 20, time.Minute),
					controllers.FromJSON[usersvc.SSOExchangeService](usersvc.SSOExchangeParameterCtx{}),
					controllers.UserSSOExchange,
				)
			}

			// SMS verification-code sign-in
			smsRouter := session.Group("sms")
			{
				// Send a verification code (login/bind/reset scenes)
				smsRouter.POST("send",
					middleware.RateLimitByIP("sms_send", 5, time.Minute),
					middleware.CaptchaRequired(func(c *gin.Context) bool {
						return dep.SettingProvider().LoginCaptchaEnabled(c)
					}),
					controllers.FromJSON[usersvc.SmsSendCodeService](usersvc.SmsSendCodeParameterCtx{}),
					controllers.UserSendSmsCode,
				)
				// Sign in with phone + code
				smsRouter.POST("login",
					middleware.RateLimitByIP("sms_login", 10, time.Minute),
					controllers.FromJSON[usersvc.SmsLoginService](usersvc.SmsLoginParameterCtx{}),
					controllers.UserSmsLogin,
					controllers.UserIssueToken,
				)
			}

			// QQ Connect (non-OIDC OAuth2 provider)
			qqRouter := session.Group("qq")
			{
				qqRouter.GET("login",
					controllers.FromQuery[usersvc.QQLoginService](usersvc.QQLoginParameterCtx{}),
					controllers.UserQQLogin,
				)
				qqRouter.GET("callback",
					controllers.FromQuery[usersvc.QQCallbackService](usersvc.QQCallbackParameterCtx{}),
					controllers.UserQQCallback,
				)
			}

			// WeChat Open Platform scan login (non-OIDC OAuth2 provider)
			wechatRouter := session.Group("wechat")
			{
				wechatRouter.GET("login",
					controllers.FromQuery[usersvc.WeChatLoginService](usersvc.WeChatLoginParameterCtx{}),
					controllers.UserWeChatLogin,
				)
				wechatRouter.GET("callback",
					controllers.FromQuery[usersvc.WeChatCallbackService](usersvc.WeChatCallbackParameterCtx{}),
					controllers.UserWeChatCallback,
				)
			}

			oauthRouter := session.Group("oauth")
			{
				oauthRouter.GET("app/:app_id",
					controllers.FromUri[oauth.GetAppRegistrationService](oauth.GetAppRegistrationParamCtx{}),
					controllers.GetAppRegistration,
				)
				oauthRouter.POST("consent",
					middleware.LoginRequired(),
					controllers.FromJSON[oauth.GrantService](oauth.GrantParamCtx{}),
					controllers.GrantAppConsent,
				)
				oauthRouter.POST("consent/deny",
					middleware.LoginRequired(),
					controllers.FromJSON[oauth.GrantService](oauth.GrantParamCtx{}),
					controllers.DenyAppConsent,
				)
				oauthRouter.POST("token",
					middleware.RateLimitByIP("oauth_token", 20, time.Minute),
					controllers.FromForm[oauth.ExchangeTokenService](oauth.ExchangeTokenParamCtx{}),
					controllers.ExchangeToken,
				)
				oauthRouter.GET("jwks", controllers.OpenIDJWKS)
				oauthRouter.GET("userinfo",
					middleware.LoginRequired(),
					controllers.FromQuery[oauth.UserInfoService](oauth.UserInfoParamCtx{}),
					controllers.OpenIDUserInfo,
				)
				oauthRouter.DELETE("grant/:app_id",
					middleware.LoginRequired(),
					middleware.RequiredScopes(types.ScopeUserSecurityInfoWrite),
					controllers.FromUri[oauth.DeleteOAuthGrantService](oauth.DeleteOAuthGrantParamCtx{}),
					controllers.DeleteOAuthGrant,
				)
			}

			authn := session.Group("authn")
			{
				// WebAuthn login prepare
				authn.PUT("",
					middleware.RateLimitByIP("authn", 20, time.Minute),
					middleware.IsFunctionEnabled(func(c *gin.Context) bool {
						return dep.SettingProvider().AuthnEnabled(c)
					}),
					controllers.StartLoginAuthn,
				)
				// WebAuthn finish login
				authn.POST("",
					middleware.RateLimitByIP("authn", 20, time.Minute),
					middleware.IsFunctionEnabled(func(c *gin.Context) bool {
						return dep.SettingProvider().AuthnEnabled(c)
					}),
					controllers.FromJSON[usersvc.FinishPasskeyLoginService](usersvc.FinishPasskeyLoginParameterCtx{}),
					controllers.FinishLoginAuthn,
					controllers.UserIssueToken,
				)
			}
		}

		// 用户相关路由
		user := v4.Group("user")
		{
			// 用户注册 Done
			user.POST("",
				middleware.RateLimitByIP("register", 5, time.Minute),
				middleware.IsFunctionEnabled(func(c *gin.Context) bool {
					return dep.SettingProvider().RegisterEnabled(c)
				}),
				middleware.CaptchaRequired(func(c *gin.Context) bool {
					return dep.SettingProvider().RegCaptchaEnabled(c)
				}),
				controllers.FromJSON[usersvc.UserRegisterService](usersvc.RegisterParameterCtx{}),
				controllers.UserRegister,
			)
			// 通过邮件里的链接重设密码
			user.PATCH("reset/:id",
				middleware.RateLimitByIP("reset_apply", 10, time.Minute),
				middleware.HashID(hashid.UserID),
				controllers.FromJSON[usersvc.UserResetService](usersvc.UserResetParameterCtx{}),
				controllers.UserReset,
			)
			// 发送密码重设邮件
			user.POST("reset",
				middleware.RateLimitByIP("reset_mail", 5, time.Minute),
				middleware.CaptchaRequired(func(c *gin.Context) bool {
					return dep.SettingProvider().ForgotPasswordCaptchaEnabled(c)
				}),
				controllers.FromJSON[usersvc.UserResetEmailService](usersvc.UserResetEmailParameterCtx{}),
				controllers.UserSendReset,
			)
			// 通过短信验证码重设密码
			user.POST("reset_sms",
				middleware.RateLimitByIP("reset_sms", 10, time.Minute),
				controllers.FromJSON[usersvc.SmsResetService](usersvc.SmsResetParameterCtx{}),
				controllers.UserSmsReset,
			)
			// 邮件激活 Done
			user.GET("activate/:id",
				middleware.SignRequired(dep.GeneralAuth()),
				middleware.HashID(hashid.UserID),
				controllers.UserActivate,
			)
			// 邮箱更换确认 Done
			user.GET("activate_email/:id",
				middleware.SignRequired(dep.GeneralAuth()),
				middleware.HashID(hashid.UserID),
				controllers.UserActivateEmailChange,
			)
			// 获取用户头像
			user.GET("avatar/:id",
				middleware.HashID(hashid.UserID),
				controllers.FromQuery[usersvc.GetAvatarService](usersvc.GetAvatarServiceParamsCtx{}),
				controllers.GetUserAvatar,
			)
			// User info
			user.GET("info/:id", middleware.HashID(hashid.UserID), controllers.UserGet)
			// List user shares
			user.GET("shares/:id",
				middleware.HashID(hashid.UserID),
				controllers.FromQuery[sharesvc.ListShareService](sharesvc.ListShareParamCtx{}),
				controllers.ListPublicShare,
			)
		}

		// 需要携带签名验证的
		sign := v4.Group("")
		sign.Use(middleware.SignRequired(dep.GeneralAuth()))
		{
			file := sign.Group("file")
			{
				file.GET("archive/:sessionID/archive.zip",
					controllers.FromUri[explorer.ArchiveService](explorer.ArchiveParamCtx{}),
					controllers.DownloadArchive,
				)
			}

			// Copy user session
			sign.GET(
				"user/session/copy/:id",
				middleware.MobileRequestOnly(),
				controllers.UserPerformCopySession,
			)
		}

		// Receive calls from slave node
		slave := v4.Group("slave")
		slave.Use(
			middleware.SlaveRPCSignRequired(),
		)
		{
			initSlaveFileRouter(slave)
			// Get credential
			slave.GET("credential/:id",
				controllers.FromUri[node.OauthCredentialService](node.OauthCredentialParamCtx{}),
				controllers.SlaveGetCredential)
			statelessUpload := slave.Group("statelessUpload")
			{
				// Prepare upload
				statelessUpload.PUT("prepare",
					controllers.FromJSON[fs.StatelessPrepareUploadService](node.StatelessPrepareUploadParamCtx{}),
					controllers.StatelessPrepareUpload)
				// Complete upload
				statelessUpload.POST("complete",
					controllers.FromJSON[fs.StatelessCompleteUploadService](node.StatelessCompleteUploadParamCtx{}),
					controllers.StatelessCompleteUpload)
				// On upload failed
				statelessUpload.POST("failed",
					controllers.FromJSON[fs.StatelessOnUploadFailedService](node.StatelessOnUploadFailedParamCtx{}),
					controllers.StatelessOnUploadFailed)
				// Create file
				statelessUpload.POST("create",
					controllers.FromJSON[fs.StatelessCreateFileService](node.StatelessCreateFileParamCtx{}),
					controllers.StatelessCreateFile)
			}
		}

		// 回调接口
		callback := v4.Group("callback")
		{
			// 远程策略上传回调
			callback.POST(
				"remote/:sessionID/:key",
				middleware.UseUploadSession(types.PolicyTypeRemote),
				middleware.RemoteCallbackAuth(),
				controllers.ProcessCallback(http.StatusOK, false),
			)
			// OSS callback
			callback.POST(
				"oss/:sessionID/:key",
				middleware.UseUploadSession(types.PolicyTypeOss),
				middleware.OSSCallbackAuth(),
				controllers.OSSCallbackValidate,
				controllers.ProcessCallback(http.StatusBadRequest, false),
			)
			// 又拍云策略上传回调
			callback.POST(
				"upyun/:sessionID/:key",
				middleware.UseUploadSession(types.PolicyTypeUpyun),
				controllers.UpyunCallbackAuth,
				controllers.ProcessCallback(http.StatusBadRequest, false),
			)
			onedrive := callback.Group("onedrive")
			{
				// 文件上传完成
				onedrive.POST(
					":sessionID/:key",
					middleware.UseUploadSession(types.PolicyTypeOd),
					controllers.ProcessCallback(http.StatusOK, false),
				)
			}
			// Google Drive related
			gdrive := callback.Group("googledrive")
			{
				// OAuth 完成
				gdrive.GET(
					"auth",
					controllers.GoogleDriveOAuth,
				)
			}
			// 腾讯云COS策略上传回调
			callback.GET(
				"cos/:sessionID/:key",
				middleware.UseUploadSession(types.PolicyTypeCos),
				controllers.ProcessCallback(http.StatusBadRequest, false),
			)
			// AWS S3策略上传回调
			callback.GET(
				"s3/:sessionID/:key",
				middleware.UseUploadSession(types.PolicyTypeS3),
				controllers.ProcessCallback(http.StatusBadRequest, false),
			)
			// 金山 ks3策略上传回调
			callback.GET(
				"ks3/:sessionID/:key",
				middleware.UseUploadSession(types.PolicyTypeKs3),
				controllers.ProcessCallback(http.StatusBadRequest, false),
			)
			// Huawei OBS upload callback
			callback.POST(
				"obs/:sessionID/:key",
				middleware.UseUploadSession(types.PolicyTypeObs),
				controllers.ProcessCallback(http.StatusBadRequest, false),
			)
			// Qiniu callback
			callback.POST(
				"qiniu/:sessionID/:key",
				middleware.UseUploadSession(types.PolicyTypeQiniu),
				controllers.QiniuCallbackValidate,
				controllers.ProcessCallback(http.StatusBadRequest, true),
			)
		}

		// Workflows
		wf := v4.Group("workflow")
		wf.Use(middleware.LoginRequired())
		wf.Use(middleware.RequiredScopes(types.ScopeWorkflowRead))
		{
			// List
			wf.GET("",
				controllers.FromQuery[explorer.ListTaskService](explorer.ListTaskParamCtx{}),
				controllers.ListTasks,
			)
			// GetTaskProgress
			wf.GET("progress/:id",
				middleware.HashID(hashid.TaskID),
				controllers.GetTaskPhaseProgress,
			)
			// Retry a failed task with its original args
			wf.POST(":id/retry",
				middleware.RequiredScopes(types.ScopeWorkflowWrite),
				middleware.HashID(hashid.TaskID),
				controllers.RetryTask,
			)
			// Cancel a queued or suspending task
			wf.POST(":id/cancel",
				middleware.RequiredScopes(types.ScopeWorkflowWrite),
				middleware.HashID(hashid.TaskID),
				controllers.CancelTask,
			)
			// Delete (hide) a finished task record
			wf.DELETE(":id",
				middleware.RequiredScopes(types.ScopeWorkflowWrite),
				middleware.HashID(hashid.TaskID),
				controllers.DeleteTask,
			)
			// Create task to create an archive file
			wf.POST("archive",
				middleware.RequiredScopes(types.ScopeWorkflowWrite),
				controllers.FromJSON[explorer.ArchiveWorkflowService](explorer.CreateArchiveParamCtx{}),
				controllers.CreateArchive,
			)
			// Create task to extract an archive file
			wf.POST("extract",
				middleware.RequiredScopes(types.ScopeWorkflowWrite),
				controllers.FromJSON[explorer.ArchiveWorkflowService](explorer.CreateArchiveParamCtx{}),
				controllers.ExtractArchive,
			)

			remoteDownload := wf.Group("download")
			{
				// Create task to download a file
				remoteDownload.POST("",
					middleware.RequiredScopes(types.ScopeWorkflowWrite),
					controllers.FromJSON[explorer.DownloadWorkflowService](explorer.CreateDownloadParamCtx{}),
					controllers.CreateRemoteDownload,
				)
				// Set download target
				remoteDownload.PATCH(":id",
					middleware.RequiredScopes(types.ScopeWorkflowWrite),
					middleware.HashID(hashid.TaskID),
					controllers.FromJSON[explorer.SetDownloadFilesService](explorer.SetDownloadFilesParamCtx{}),
					controllers.SetDownloadTaskTarget,
				)
				remoteDownload.DELETE(":id",
					middleware.RequiredScopes(types.ScopeWorkflowWrite),
					middleware.HashID(hashid.TaskID),
					controllers.CancelDownloadTask,
				)
			}
		}

		// 文件
		file := v4.Group("file")
		file.Use(middleware.RequiredScopes(types.ScopeFilesRead))
		{
			// List files
			file.GET("",
				controllers.FromQuery[explorer.ListFileService](explorer.ListFileParameterCtx{}),
				controllers.ListDirectory,
			)
			file.GET("archive",
				controllers.FromQuery[explorer.ArchiveListFilesService](explorer.ArchiveListFilesParamCtx{}),
				controllers.ListArchiveFiles,
			)
			// Create file
			file.POST("create",
				middleware.RequiredScopes(types.ScopeFilesWrite),
				controllers.FromJSON[explorer.CreateFileService](explorer.CreateFileParameterCtx{}),
				controllers.CreateFile,
			)
			// Rename file
			file.POST("rename",
				middleware.RequiredScopes(types.ScopeFilesWrite),
				controllers.FromJSON[explorer.RenameFileService](explorer.RenameFileParameterCtx{}),
				controllers.RenameFile,
			)
			// Move or copy files
			file.POST("move",
				middleware.RequiredScopes(types.ScopeFilesWrite),
				controllers.FromJSON[explorer.MoveFileService](explorer.MoveFileParameterCtx{}),
				middleware.ValidateBatchFileCount(dep, explorer.MoveFileParameterCtx{}),
				controllers.MoveFile)
			// Get URL of the file for preview/download
			file.POST("url",
				middleware.ContextHint(),
				controllers.FromJSON[explorer.FileURLService](explorer.FileURLParameterCtx{}),
				middleware.ValidateBatchFileCount(dep, explorer.FileURLParameterCtx{}),
				controllers.FileURL,
			)
			// Update file content
			file.PUT("content",
				middleware.RequiredScopes(types.ScopeFilesWrite),
				controllers.FromQuery[explorer.FileUpdateService](explorer.FileUpdateParameterCtx{}),
				controllers.PutContent)
			// Get entity content for preview/download
			content := file.Group("content")
			content.Use(middleware.ContentCORS())
			{
				content.OPTIONS("*option", middleware.ContentCORS())
				content.GET(":id/:speed/:name",
					middleware.SignRequired(dep.GeneralAuth()),
					middleware.HashID(hashid.EntityID),
					middleware.Sandbox(),
					controllers.FromUri[explorer.EntityDownloadService](explorer.EntityDownloadParameterCtx{}),
					controllers.ServeEntity,
				)
				content.HEAD(":id/:speed/:name",
					middleware.SignRequired(dep.GeneralAuth()),
					middleware.HashID(hashid.EntityID),
					controllers.FromUri[explorer.EntityDownloadService](explorer.EntityDownloadParameterCtx{}),
					controllers.ServeEntity,
				)
			}
			// get thumb
			file.GET("thumb",
				middleware.ContextHint(),
				controllers.FromQuery[explorer.FileThumbService](explorer.FileThumbParameterCtx{}),
				controllers.Thumb,
			)
			// Delete files
			file.DELETE("",
				middleware.RequiredScopes(types.ScopeFilesWrite),
				controllers.FromJSON[explorer.DeleteFileService](explorer.DeleteFileParameterCtx{}),
				middleware.ValidateBatchFileCount(dep, explorer.DeleteFileParameterCtx{}),
				controllers.Delete,
			)
			// Empty trash bin
			file.DELETE("trash",
				middleware.RequiredScopes(types.ScopeFilesWrite),
				controllers.EmptyTrash,
			)
			// Force unlock
			file.DELETE("lock",
				middleware.RequiredScopes(types.ScopeFilesWrite),
				controllers.FromJSON[explorer.UnlockFileService](explorer.UnlockFileParameterCtx{}),
				controllers.Unlock,
			)
			// Restore files
			file.POST("restore",
				middleware.RequiredScopes(types.ScopeFilesWrite),
				controllers.FromJSON[explorer.DeleteFileService](explorer.DeleteFileParameterCtx{}),
				middleware.ValidateBatchFileCount(dep, explorer.DeleteFileParameterCtx{}),
				controllers.Restore,
			)
			// Patch metadata
			file.PATCH("metadata",
				middleware.RequiredScopes(types.ScopeFilesWrite),
				controllers.FromJSON[explorer.PatchMetadataService](explorer.PatchMetadataParameterCtx{}),
				middleware.ValidateBatchFileCount(dep, explorer.PatchMetadataParameterCtx{}),
				controllers.PatchMetadata,
			)
			// Tag management across all of the user's files
			file.GET("tag",
				controllers.FromQuery[explorer.ListTagsService](explorer.ListTagsParameterCtx{}),
				controllers.ListTags,
			)
			file.PATCH("tag",
				middleware.RequiredScopes(types.ScopeFilesWrite),
				controllers.FromJSON[explorer.PatchTagService](explorer.PatchTagParameterCtx{}),
				controllers.PatchTag,
			)
			file.DELETE("tag",
				middleware.RequiredScopes(types.ScopeFilesWrite),
				controllers.FromJSON[explorer.DeleteTagService](explorer.DeleteTagParameterCtx{}),
				controllers.DeleteTag,
			)
			// List storage policies available to the current group
			file.GET("policy",
				controllers.FromQuery[explorer.AllowedPolicyService](explorer.AllowedPolicyParamCtx{}),
				controllers.ListStoragePolicies,
			)
			// Set preferred storage policy for a directory
			file.PUT("policy",
				middleware.RequiredScopes(types.ScopeFilesWrite),
				controllers.FromJSON[explorer.PreferredPolicyService](explorer.PreferredPolicyParamCtx{}),
				controllers.UpdatePreferredPolicy,
			)
			// Relocate files to a different storage policy
			file.POST("relocate",
				middleware.RequiredScopes(types.ScopeFilesWrite),
				controllers.FromJSON[explorer.FileRelocateService](explorer.FileRelocateParamCtx{}),
				controllers.RelocatePolicy,
			)
			// Upload related
			upload := file.Group("upload", middleware.RequiredScopes(types.ScopeFilesWrite))
			{
				// Create upload session
				upload.PUT("",
					controllers.FromJSON[explorer.CreateUploadSessionService](explorer.CreateUploadSessionParameterCtx{}),
					controllers.CreateUploadSession,
				)
				// Upload file data
				upload.POST(":sessionId/:index",
					controllers.FromUri[explorer.UploadService](explorer.UploadParameterCtx{}),
					controllers.FileUpload,
				)
				upload.DELETE("",
					controllers.FromJSON[explorer.DeleteUploadSessionService](explorer.DeleteUploadSessionParameterCtx{}),
					controllers.DeleteUploadSession,
				)
			}
			// Pin file
			pin := file.Group("pin", middleware.RequiredScopes(types.ScopeFilesWrite))
			{
				// Pin file
				pin.PUT("",
					controllers.FromJSON[explorer.PinFileService](explorer.PinFileParameterCtx{}),
					controllers.Pin,
				)
				// Unpin file
				pin.DELETE("",
					controllers.FromJSON[explorer.PinFileService](explorer.PinFileParameterCtx{}),
					controllers.Unpin,
				)
			}
			// Get file info
			file.GET("info",
				controllers.FromQuery[explorer.GetFileInfoService](explorer.GetFileInfoParameterCtx{}),
				controllers.GetFileInfo,
			)
			// Per-file ACL entries (Permissions dialog)
			acl := file.Group("acl")
			{
				acl.GET("",
					controllers.FromQuery[explorer.AclListService](explorer.AclListParamCtx{}),
					controllers.ListAcl,
				)
				acl.GET("subjects",
					controllers.FromQuery[explorer.AclSubjectSearchService](explorer.AclSubjectSearchParamCtx{}),
					controllers.SearchAclSubjects,
				)
				acl.PUT("",
					middleware.RequiredScopes(types.ScopeFilesWrite),
					controllers.FromJSON[explorer.AclUpsertService](explorer.AclUpsertParamCtx{}),
					controllers.UpsertAcl,
				)
				acl.DELETE("",
					middleware.RequiredScopes(types.ScopeFilesWrite),
					controllers.FromQuery[explorer.AclDeleteService](explorer.AclDeleteParamCtx{}),
					controllers.DeleteAcl,
				)
			}
			// Per-file audit activity (Activity dialog)
			file.GET("activity",
				controllers.FromQuery[explorer.FileActivityService](explorer.FileActivityParamCtx{}),
				controllers.GetFileActivity,
			)
			// Version management
			version := file.Group("version", middleware.RequiredScopes(types.ScopeFilesWrite))
			{
				// Set current version
				version.POST("current",
					controllers.FromJSON[explorer.SetCurrentVersionService](explorer.SetCurrentVersionParamCtx{}),
					controllers.SetCurrentVersion,
				)
				// Delete a version from a file
				version.DELETE("",
					controllers.FromJSON[explorer.DeleteVersionService](explorer.DeleteVersionParamCtx{}),
					controllers.DeleteVersion,
				)
			}
			file.PUT("viewerSession",
				middleware.RequiredScopes(types.ScopeFilesWrite),
				controllers.FromJSON[explorer.CreateViewerSessionService](explorer.CreateViewerSessionParamCtx{}),
				controllers.CreateViewerSession,
			)
			// Create task to import files
			wf.POST("import",
				middleware.IsAdmin(),
				middleware.RequiredScopes(types.ScopeWorkflowWrite, types.ScopeAdminWrite),
				controllers.FromJSON[explorer.ImportWorkflowService](explorer.CreateImportParamCtx{}),
				controllers.ImportFiles,
			)
			// Create task to import files
			wf.POST("rebuildFtsIndex",
				middleware.IsAdmin(),
				middleware.RequiredScopes(types.ScopeWorkflowWrite, types.ScopeAdminWrite),
				controllers.FromJSON[explorer.RebuildFTSIndexWorkflowService](explorer.CreateRebuildFTSIndexParamCtx{}),
				controllers.RebuildFTSIndex,
			)
			// Create task to audit physical blobs against the entities table
			wf.POST("blobAudit",
				middleware.IsAdmin(),
				middleware.RequiredScopes(types.ScopeWorkflowWrite, types.ScopeAdminWrite),
				controllers.FromJSON[explorer.BlobAuditWorkflowService](explorer.BlobAuditParamCtx{}),
				controllers.BlobAudit,
			)

			// 取得文件外链
			source := file.Group("source")
			{
				source.PUT("",
					controllers.FromJSON[explorer.GetDirectLinkService](explorer.GetDirectLinkParamCtx{}),
					middleware.ValidateBatchFileCount(dep, explorer.GetDirectLinkParamCtx{}),
					controllers.GetSource,
				)
				source.DELETE(":id",
					middleware.HashID(hashid.SourceLinkID),
					controllers.DeleteDirectLink,
				)
			}
			// Patch view
			file.PATCH("view",
				middleware.RequiredScopes(types.ScopeFilesWrite),
				controllers.FromJSON[explorer.PatchViewService](explorer.PatchViewParameterCtx{}),
				controllers.PatchView,
			)

			// Server event push
			file.GET("events",
				middleware.LoginRequired(),
				middleware.IsFunctionEnabled(func(c *gin.Context) bool {
					return dep.SettingProvider().EventHubEnabled(c)
				}),
				controllers.FromQuery[explorer.ExplorerEventService](explorer.ExplorerEventParamCtx{}),
				controllers.HandleExplorerEventsPush,
			)

			// Full text search
			file.GET("search",
				middleware.LoginRequired(),
				middleware.IsFunctionEnabled(func(c *gin.Context) bool {
					return dep.SettingProvider().FTSEnabled(c)
				}),
				controllers.FromQuery[explorer.FulltextSearchService](explorer.FulltextSearchParamCtx{}),
				controllers.FulltextSearch,
			)
		}

		// 分享相关
		share := v4.Group("share")
		share.Use(middleware.RequiredScopes(types.ScopeSharesRead))
		{
			// Create share link
			share.PUT("",
				middleware.LoginRequired(),
				middleware.RequiredScopes(types.ScopeSharesWrite),
				controllers.FromJSON[sharesvc.ShareCreateService](sharesvc.ShareCreateParamCtx{}),
				controllers.CreateShare,
			)
			// Edit existing share link
			share.POST(":id",
				middleware.LoginRequired(),
				middleware.RequiredScopes(types.ScopeSharesWrite),
				middleware.HashID(hashid.ShareID),
				controllers.FromJSON[sharesvc.ShareCreateService](sharesvc.ShareCreateParamCtx{}),
				controllers.EditShare,
			)
			// Get share link info
			share.GET("info/:id",
				middleware.HashID(hashid.ShareID),
				controllers.FromQuery[sharesvc.ShareInfoService](sharesvc.ShareInfoParamCtx{}),
				controllers.GetShare,
			)
			// Public share directory listing (anonymous-accessible)
			share.GET("listed",
				middleware.RateLimitByIP("share_listed", 60, time.Minute),
				controllers.FromQuery[sharesvc.ListPublicShareService](sharesvc.ListPublicShareParamCtx{}),
				controllers.ListPublicShares,
			)
			// Purchase a paid share with credits
			share.POST("purchase/:id",
				middleware.LoginRequired(),
				middleware.HashID(hashid.ShareID),
				controllers.PurchaseShare,
			)
			// List my shares
			share.GET("",
				middleware.LoginRequired(),
				controllers.FromQuery[sharesvc.ListShareService](sharesvc.ListShareParamCtx{}),
				controllers.ListShare,
			)
			// 删除分享
			share.DELETE(":id",
				middleware.LoginRequired(),
				middleware.RequiredScopes(types.ScopeSharesWrite),
				middleware.HashID(hashid.ShareID),
				controllers.DeleteShare,
			)
			share.DELETE("",
				middleware.LoginRequired(),
				middleware.RequiredScopes(types.ScopeSharesWrite),
				controllers.FromJSON[sharesvc.BatchDeleteShareService](sharesvc.BatchDeleteParamCtx{}),
				controllers.BatchDeleteShare,
			)
			//// 获取README文本文件内容
			//share.GET("readme/:id",
			//	middleware.CheckShareUnlocked(),
			//	controllers.PreviewShareReadme,
			//)
		}

		// 举报滥用
		abuse := v4.Group("abuse")
		{
			abuse.POST("report",
				middleware.RateLimitByIP("abuse_report", 10, time.Hour),
				middleware.CaptchaRequired(func(c *gin.Context) bool {
					return dep.SettingProvider().AbuseCaptchaEnabled(c)
				}),
				controllers.FromJSON[abusesvc.ReportService](abusesvc.ReportParamCtx{}),
				controllers.ReportAbuse,
			)
		}

		// 需要登录保护的
		auth := v4.Group("")
		auth.Use(middleware.LoginRequired())
		{
			// 管理
			admin := auth.Group("admin", middleware.IsAdminOrDelegated())
			admin.Use(middleware.RequiredScopes(types.ScopeAdminRead))
			{
				admin.GET("summary",
					controllers.FromQuery[adminsvc.SummaryService](adminsvc.SummaryParamCtx{}),
					controllers.AdminSummary,
				)

				settings := admin.Group("settings", middleware.AdminSection(types.GroupPermissionAdminSettings))
				{
					// Get settings
					settings.POST("",
						controllers.FromJSON[adminsvc.GetSettingService](adminsvc.GetSettingParamCtx{}),
						controllers.AdminGetSettings,
					)
					// Patch settings
					settings.PATCH("",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.SetSettingService](adminsvc.SetSettingParamCtx{}),
						controllers.AdminSetSettings,
					)
				}

				// 用户组管理
				group := admin.Group("group", middleware.AdminSection(types.GroupPermissionAdminGroups))
				{
					// 列出用户组
					group.POST("",
						controllers.FromJSON[adminsvc.AdminListService](adminsvc.AdminListServiceParamsCtx{}),
						controllers.AdminListGroups,
					)
					// 获取用户组
					group.GET(":id",
						controllers.FromUri[adminsvc.SingleGroupService](adminsvc.SingleGroupParamCtx{}),
						controllers.AdminGetGroup,
					)
					// 创建用户组
					group.PUT("",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.UpsertGroupService](adminsvc.UpsertGroupParamCtx{}),
						controllers.AdminCreateGroup,
					)
					// 更新用户组
					group.PUT(":id",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.UpsertGroupService](adminsvc.UpsertGroupParamCtx{}),
						controllers.AdminUpdateGroup,
					)
					// 删除用户组
					group.DELETE(":id",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromUri[adminsvc.SingleGroupService](adminsvc.SingleGroupParamCtx{}),
						controllers.AdminDeleteGroup,
					)
				}

				tool := admin.Group("tool", middleware.AdminSection(types.GroupPermissionAdminSettings))
				{
					tool.GET("wopi",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromQuery[adminsvc.FetchWOPIDiscoveryService](adminsvc.FetchWOPIDiscoveryParamCtx{}),
						controllers.AdminFetchWopi,
					)
					tool.POST("thumbExecutable",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.ThumbGeneratorTestService](adminsvc.ThumbGeneratorTestParamCtx{}),
						controllers.AdminTestThumbGenerator)
					tool.POST("mail",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.TestSMTPService](adminsvc.TestSMTPParamCtx{}),
						controllers.AdminSendTestMail,
					)
					tool.DELETE("entityUrlCache",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.AdminClearEntityUrlCache,
					)
				}

				queue := admin.Group("queue", middleware.AdminSection(types.GroupPermissionAdminQueue))
				{
					queue.GET("metrics", controllers.AdminGetQueueMetrics)
					// List tasks
					queue.POST("",
						controllers.FromJSON[adminsvc.AdminListService](adminsvc.AdminListServiceParamsCtx{}),
						controllers.AdminListTasks,
					)
					// Get task
					queue.GET(":id",
						controllers.FromUri[adminsvc.SingleTaskService](adminsvc.SingleTaskParamCtx{}),
						controllers.AdminGetTask,
					)
					// Batch delete task
					queue.POST("batch/delete",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.BatchTaskService](adminsvc.BatchTaskParamCtx{}),
						controllers.AdminBatchDeleteTask,
					)
					// Cleanup tasks
					queue.POST("cleanup",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.CleanupTaskService](adminsvc.CleanupTaskParameterCtx{}),
						controllers.AdminCleanupTask,
					)
					// // 列出任务
					// queue.POST("list", controllers.AdminListTask)
					// // 新建文件导入任务
					// queue.POST("import", controllers.AdminCreateImportTask)
				}

				// 存储策略管理
				policy := admin.Group("policy", middleware.AdminSection(types.GroupPermissionAdminStorage))
				{
					// 列出存储策略
					policy.POST("",
						controllers.FromJSON[adminsvc.AdminListService](adminsvc.AdminListServiceParamsCtx{}),
						controllers.AdminListPolicies,
					)
					// 获取存储策略详情
					policy.GET(":id",
						controllers.FromUri[adminsvc.SingleStoragePolicyService](adminsvc.GetStoragePolicyParamCtx{}),
						controllers.AdminGetPolicy,
					)
					// 创建存储策略
					policy.PUT("",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.CreateStoragePolicyService](adminsvc.CreateStoragePolicyParamCtx{}),
						controllers.AdminCreatePolicy,
					)
					// 更新存储策略
					policy.PUT(":id",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.UpdateStoragePolicyService](adminsvc.UpdateStoragePolicyParamCtx{}),
						controllers.AdminUpdatePolicy,
					)
					// 创建跨域策略
					policy.POST("cors",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.CreateStoragePolicyCorsService](adminsvc.CreateStoragePolicyCorsParamCtx{}),
						controllers.AdminCreateStoragePolicyCors,
					)
					// // 获取 OneDrive OAuth URL
					oauth := policy.Group("oauth")
					{
						// 获取 OneDrive OAuth URL
						oauth.POST("signin",
							middleware.RequiredScopes(types.ScopeAdminWrite),
							controllers.FromJSON[adminsvc.GetOauthRedirectService](adminsvc.GetOauthRedirectParamCtx{}),
							controllers.AdminOdOAuthURL,
						)
						// 获取 OAuth 回调 URL
						oauth.GET("redirect", controllers.AdminGetPolicyOAuthCallbackURL)
						oauth.GET("status/:id",
							controllers.FromUri[adminsvc.SingleStoragePolicyService](adminsvc.GetStoragePolicyParamCtx{}),
							controllers.AdminGetPolicyOAuthStatus,
						)
						oauth.POST("callback",
							middleware.RequiredScopes(types.ScopeAdminWrite),
							controllers.FromJSON[adminsvc.FinishOauthCallbackService](adminsvc.FinishOauthCallbackParamCtx{}),
							controllers.AdminFinishOauthCallback,
						)
						oauth.GET("root/:id",
							controllers.FromUri[adminsvc.SingleStoragePolicyService](adminsvc.GetStoragePolicyParamCtx{}),
							controllers.AdminGetSharePointDriverRoot,
						)
					}

					// // 获取 存储策略
					// policy.GET(":id", controllers.AdminGetPolicy)
					// 删除 存储策略
					policy.DELETE(":id",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromUri[adminsvc.SingleStoragePolicyService](adminsvc.GetStoragePolicyParamCtx{}),
						controllers.AdminDeletePolicy,
					)
				}

				node := admin.Group("node", middleware.AdminSection(types.GroupPermissionAdminStorage))
				{
					node.POST("",
						controllers.FromJSON[adminsvc.AdminListService](adminsvc.AdminListServiceParamsCtx{}),
						controllers.AdminListNodes,
					)
					node.GET(":id",
						controllers.FromUri[adminsvc.SingleNodeService](adminsvc.SingleNodeParamCtx{}),
						controllers.AdminGetNode,
					)
					node.POST("test",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.TestNodeService](adminsvc.TestNodeParamCtx{}),
						controllers.AdminTestSlave,
					)
					node.POST("test/downloader",
						controllers.FromJSON[adminsvc.TestNodeDownloaderService](adminsvc.TestNodeDownloaderParamCtx{}),
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.AdminTestDownloader,
					)
					node.PUT("",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.UpsertNodeService](adminsvc.UpsertNodeParamCtx{}),
						controllers.AdminCreateNode,
					)
					node.PUT(":id",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.UpsertNodeService](adminsvc.UpsertNodeParamCtx{}),
						controllers.AdminUpdateNode,
					)
					node.DELETE(":id",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromUri[adminsvc.SingleNodeService](adminsvc.SingleNodeParamCtx{}),
						controllers.AdminDeleteNode,
					)
				}

				oauthClient := admin.Group("oauthClient", middleware.AdminSection(types.GroupPermissionAdminSettings))
				{
					// List OAuth clients
					oauthClient.POST("",
						controllers.FromJSON[adminsvc.AdminListService](adminsvc.AdminListServiceParamsCtx{}),
						controllers.AdminListOAuthClients,
					)
					// Get OAuth client
					oauthClient.GET(":id",
						controllers.FromUri[adminsvc.SingleOAuthClientService](adminsvc.SingleOAuthClientParamCtx{}),
						controllers.AdminGetOAuthClient,
					)
					// Create OAuth client
					oauthClient.PUT("",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.UpsertOAuthClientService](adminsvc.UpsertOAuthClientParamCtx{}),
						controllers.AdminCreateOAuthClient,
					)
					// Update OAuth client
					oauthClient.PUT(":id",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.UpsertOAuthClientService](adminsvc.UpsertOAuthClientParamCtx{}),
						controllers.AdminUpdateOAuthClient,
					)
					// Delete OAuth client
					oauthClient.DELETE(":id",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromUri[adminsvc.SingleOAuthClientService](adminsvc.SingleOAuthClientParamCtx{}),
						controllers.AdminDeleteOAuthClient,
					)
					// Batch delete OAuth clients
					oauthClient.POST("batch/delete",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.BatchOAuthClientService](adminsvc.BatchOAuthClientParamCtx{}),
						controllers.AdminBatchDeleteOAuthClient,
					)
				}

				user := admin.Group("user", middleware.AdminSection(types.GroupPermissionAdminUsers))
				{
					// 列出用户
					user.POST("",
						controllers.FromJSON[adminsvc.AdminListService](adminsvc.AdminListServiceParamsCtx{}),
						controllers.AdminListUsers,
					)
					// 获取用户
					user.GET(":id",
						controllers.FromUri[adminsvc.SingleUserService](adminsvc.SingleUserParamCtx{}),
						controllers.AdminGetUser,
					)
					// 更新用户
					user.PUT(":id",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.UpsertUserService](adminsvc.UpsertUserParamCtx{}),
						controllers.AdminUpdateUser,
					)
					// 创建用户
					user.PUT("",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.UpsertUserService](adminsvc.UpsertUserParamCtx{}),
						controllers.AdminCreateUser,
					)
					batch := user.Group("batch")
					{
						// 批量删除用户
						batch.POST("delete",
							middleware.RequiredScopes(types.ScopeAdminWrite),
							controllers.FromJSON[adminsvc.BatchUserService](adminsvc.BatchUserParamCtx{}),
							controllers.AdminDeleteUser,
						)
						// 批量更新用户
						batch.POST("update",
							middleware.RequiredScopes(types.ScopeAdminWrite),
							controllers.FromJSON[adminsvc.BatchUserUpdateService](adminsvc.BatchUserUpdateParamCtx{}),
							controllers.AdminBatchUpdateUser,
						)
					}
					user.POST(":id/calibrate",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromUri[adminsvc.SingleUserService](adminsvc.SingleUserParamCtx{}),
						controllers.AdminCalibrateStorage,
					)
					// 列出邀请码
					user.POST("invitation/list",
						controllers.FromJSON[adminsvc.InvitationCodeListService](adminsvc.InvitationCodeListParamCtx{}),
						controllers.AdminListInvitationCodes,
					)
					// 创建邀请码
					user.PUT("invitation",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.UpsertInvitationCodeService](adminsvc.UpsertInvitationCodeParamCtx{}),
						controllers.AdminCreateInvitationCode,
					)
					// 删除邀请码
					user.DELETE("invitation/:id",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromUri[adminsvc.SingleInvitationCodeService](adminsvc.SingleInvitationCodeParamCtx{}),
						controllers.AdminDeleteInvitationCode,
					)
				}

				vas := admin.Group("vas", middleware.AdminSection(types.GroupPermissionAdminPayment))
				{
					// 列出礼品码
					vas.GET("giftcode",
						controllers.FromQuery[adminsvc.GiftCodeListService](adminsvc.GiftCodeListParamCtx{}),
						controllers.AdminListGiftCodes,
					)
					// 生成礼品码
					vas.PUT("giftcode",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.CreateGiftCodeService](adminsvc.CreateGiftCodeParamCtx{}),
						controllers.AdminCreateGiftCode,
					)
					// 删除礼品码
					vas.DELETE("giftcode/:id",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromUri[adminsvc.SingleGiftCodeService](adminsvc.SingleGiftCodeParamCtx{}),
						controllers.AdminDeleteGiftCode,
					)
					// 手动调整用户积分
					vas.POST("credit",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.AdjustCreditService](adminsvc.AdjustCreditParamCtx{}),
						controllers.AdminAdjustCredit,
					)
					// 列出商品
					vas.GET("sku",
						controllers.FromQuery[adminsvc.SkuListService](adminsvc.SkuListParamCtx{}),
						controllers.AdminListSkus,
					)
					// 创建商品
					vas.PUT("sku",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.SkuUpsertService](adminsvc.SkuUpsertParamCtx{}),
						controllers.AdminCreateSku,
					)
					// 更新商品
					vas.PUT("sku/:id",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.SkuUpsertService](adminsvc.SkuUpsertParamCtx{}),
						controllers.AdminUpdateSku,
					)
					// 删除商品
					vas.DELETE("sku/:id",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromUri[adminsvc.SingleSkuService](adminsvc.SingleSkuParamCtx{}),
						controllers.AdminDeleteSku,
					)
				}

				event := admin.Group("event", middleware.AdminSection(types.GroupPermissionAdminEvents))
				{
					// 列出审计事件
					event.GET("",
						controllers.FromQuery[adminsvc.EventListService](adminsvc.EventListParamCtx{}),
						controllers.AdminListEvents,
					)
				}

				abuse := admin.Group("abuse", middleware.AdminSection(types.GroupPermissionAdminReports))
				{
					// 列出举报
					abuse.GET("",
						controllers.FromQuery[adminsvc.AbuseListService](adminsvc.AbuseListParamCtx{}),
						controllers.AdminListAbuseReports,
					)
					// 更新举报状态
					abuse.PATCH(":id",
						middleware.HashID(hashid.AuditLogID),
						controllers.FromJSON[adminsvc.AbuseUpdateService](adminsvc.AbuseUpdateParamCtx{}),
						controllers.AdminUpdateAbuseReport,
					)
				}

				file := admin.Group("file", middleware.AdminSection(types.GroupPermissionAdminFiles))
				{
					// 列出文件
					file.POST("",
						controllers.FromJSON[adminsvc.AdminListService](adminsvc.AdminListServiceParamsCtx{}),
						controllers.AdminListFiles,
					)
					// 获取文件
					file.GET(":id",
						controllers.FromUri[adminsvc.SingleFileService](adminsvc.SingleFileParamCtx{}),
						controllers.AdminGetFile,
					)
					// 更新文件
					file.PUT(":id",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.UpsertFileService](adminsvc.UpsertFileParamCtx{}),
						controllers.AdminUpdateFile,
					)
					// 获取文件 URL
					file.GET("url/:id",
						controllers.FromUri[adminsvc.SingleFileService](adminsvc.SingleFileParamCtx{}),
						controllers.AdminGetFileUrl,
					)
					// 批量删除文件
					file.POST("batch/delete",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.BatchFileService](adminsvc.BatchFileParamCtx{}),
						controllers.AdminBatchDeleteFile,
					)
				}

				entity := admin.Group("entity", middleware.AdminSection(types.GroupPermissionAdminFiles))
				{
					// List blobs
					entity.POST("",
						controllers.FromJSON[adminsvc.AdminListService](adminsvc.AdminListServiceParamsCtx{}),
						controllers.AdminListEntities,
					)
					// Get entity
					entity.GET(":id",
						controllers.FromUri[adminsvc.SingleEntityService](adminsvc.SingleEntityParamCtx{}),
						controllers.AdminGetEntity,
					)
					// Batch delete entity
					entity.POST("batch/delete",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.BatchEntityService](adminsvc.BatchEntityParamCtx{}),
						controllers.AdminBatchDeleteEntity,
					)
					// Relocate entities to another storage policy
					entity.POST("relocate",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.RelocateEntityService](adminsvc.RelocateEntityParamCtx{}),
						controllers.AdminRelocateEntity,
					)
					// Get entity url
					entity.GET("url/:id",
						controllers.FromUri[adminsvc.SingleEntityService](adminsvc.SingleEntityParamCtx{}),
						controllers.AdminGetEntityUrl,
					)
				}

				share := admin.Group("share", middleware.AdminSection(types.GroupPermissionAdminShares))
				{
					// List shares
					share.POST("",
						controllers.FromJSON[adminsvc.AdminListService](adminsvc.AdminListServiceParamsCtx{}),
						controllers.AdminListShares,
					)
					// Get share
					share.GET(":id",
						controllers.FromUri[adminsvc.SingleShareService](adminsvc.SingleShareParamCtx{}),
						controllers.AdminGetShare,
					)
					// Batch delete shares
					share.POST("batch/delete",
						middleware.RequiredScopes(types.ScopeAdminWrite),
						controllers.FromJSON[adminsvc.BatchShareService](adminsvc.BatchShareParamCtx{}),
						controllers.AdminBatchDeleteShare,
					)
				}
			}

			// 用户
			user := auth.Group("user")
			{
				// 当前登录用户信息
				user.GET("me", middleware.RequiredScopes(types.ScopeUserInfoRead), controllers.UserMe)
				// 存储信息
				user.GET("capacity", middleware.RequiredScopes(types.ScopeUserInfoRead), controllers.UserStorage)
				// Search user by keywords
				user.GET("search",
					middleware.RequiredScopes(types.ScopeUserInfoRead),
					controllers.FromQuery[usersvc.SearchUserService](usersvc.SearchUserParamCtx{}),
					controllers.UserSearch,
				)

				// WebAuthn 注册相关
				authn := user.Group("authn",
					middleware.RequiredScopes(types.ScopeUserSecurityInfoWrite),
					middleware.IsFunctionEnabled(func(c *gin.Context) bool {
						return dep.SettingProvider().AuthnEnabled(c)
					}))
				{
					authn.PUT("", controllers.StartRegAuthn)
					authn.POST("",
						controllers.FromJSON[usersvc.FinishPasskeyRegisterService](usersvc.FinishPasskeyRegisterParameterCtx{}),
						controllers.FinishRegAuthn,
					)
					authn.DELETE("",
						controllers.FromQuery[usersvc.DeletePasskeyService](usersvc.DeletePasskeyParameterCtx{}),
						controllers.UserDeletePasskey,
					)
				}

				// 用户设置
				setting := user.Group("setting")
				setting.Use(middleware.RequiredScopes(types.ScopeUserInfoRead))
				{
					// 获取当前用户设定
					setting.GET("", controllers.UserSetting)
					// 当前公告（已忽略时为空）
					setting.GET("announcement",
						controllers.FromQuery[usersvc.AnnouncementService](usersvc.AnnouncementParamCtx{}),
						controllers.UserAnnouncement,
					)
					// 从文件上传头像
					setting.PUT("avatar", middleware.RequiredScopes(types.ScopeUserInfoWrite), controllers.UploadAvatar)
					// 更改用户设定
					setting.PATCH("",
						middleware.RequiredScopes(types.ScopeUserInfoWrite),
						controllers.FromJSON[usersvc.PatchUserSetting](usersvc.PatchUserSettingParamsCtx{}),
						controllers.UpdateOption,
					)
					// 获得二步验证初始化信息
					setting.GET("2fa", controllers.UserInit2FA)
					// 重新生成二步验证备用代码
					setting.PUT("2fa/backup",
						middleware.RequiredScopes(types.ScopeUserSecurityInfoWrite),
						middleware.RateLimitByIP("backup_2fa", 5, time.Hour),
						controllers.FromJSON[usersvc.Backup2FAService](usersvc.Backup2FAParameterCtx{}),
						controllers.UserBackup2FA,
					)
					// 请求更换邮箱（向新地址发送确认链接）
					setting.POST("email",
						middleware.RequiredScopes(types.ScopeUserSecurityInfoWrite),
						middleware.RateLimitByIP("email_change", 5, time.Hour),
						controllers.FromJSON[usersvc.RequestEmailChangeService](usersvc.RequestEmailChangeParamCtx{}),
						controllers.UserRequestEmailChange,
					)
					// 绑定手机号（短信验证码校验）
					setting.PUT("phone",
						middleware.RequiredScopes(types.ScopeUserSecurityInfoWrite),
						middleware.RateLimitByIP("phone_bind", 10, time.Hour),
						controllers.FromJSON[usersvc.SmsBindService](usersvc.SmsBindParameterCtx{}),
						controllers.UserBindPhone,
					)
					// 解绑手机号
					setting.DELETE("phone",
						middleware.RequiredScopes(types.ScopeUserSecurityInfoWrite),
						controllers.FromJSON[usersvc.SmsUnbindService](usersvc.SmsUnbindParameterCtx{}),
						controllers.UserUnbindPhone,
					)
					// 解除外部账号绑定（QQ Connect 等）
					setting.DELETE("sso_binding/:provider",
						middleware.RequiredScopes(types.ScopeUserSecurityInfoWrite),
						controllers.FromUri[usersvc.SsoUnbindService](usersvc.SsoUnbindParameterCtx{}),
						controllers.UserUnbindSso,
					)
				}

				// 私密空间
				vault := user.Group("vault")
				{
					// 启用私密空间
					vault.POST("",
						middleware.RequiredScopes(types.ScopeUserSecurityInfoWrite),
						controllers.FromJSON[usersvc.VaultSetupService](usersvc.VaultSetupParameterCtx{}),
						controllers.UserVaultSetup,
					)
					// 解锁私密空间
					vault.PUT("unlock",
						middleware.RequiredScopes(types.ScopeUserSecurityInfoWrite),
						middleware.RateLimitByIP("vault_unlock", 10, time.Hour),
						controllers.FromJSON[usersvc.VaultUnlockService](usersvc.VaultUnlockParameterCtx{}),
						controllers.UserVaultUnlock,
					)
					// 立即锁定私密空间
					vault.DELETE("unlock",
						middleware.RequiredScopes(types.ScopeUserSecurityInfoWrite),
						controllers.FromJSON[usersvc.VaultUnlockService](usersvc.VaultUnlockParameterCtx{}),
						controllers.UserVaultLock,
					)
					// 关闭私密空间
					vault.DELETE("",
						middleware.RequiredScopes(types.ScopeUserSecurityInfoWrite),
						controllers.FromJSON[usersvc.VaultDisableService](usersvc.VaultDisableParameterCtx{}),
						controllers.UserVaultDisable,
					)
				}

				// 积分与兑换
				credit := user.Group("credit")
				{
					// 余额与权益
					credit.GET("",
						middleware.RequiredScopes(types.ScopeUserInfoRead),
						controllers.FromQuery[usersvc.CreditService](usersvc.CreditParamCtx{}),
						controllers.UserCredit,
					)
					// 积分流水
					credit.GET("txns",
						middleware.RequiredScopes(types.ScopeUserInfoRead),
						controllers.FromQuery[usersvc.CreditTxnListService](usersvc.CreditTxnListParamCtx{}),
						controllers.UserCreditTxns,
					)
					// 兑换礼品码
					credit.POST("redeem",
						middleware.RequiredScopes(types.ScopeUserInfoWrite),
						middleware.RateLimitByIP("redeem", 20, time.Hour),
						controllers.FromJSON[usersvc.RedeemGiftCodeService](usersvc.RedeemGiftCodeParamCtx{}),
						controllers.UserRedeemGiftCode,
					)
				}

				// 积分商城
				shop := user.Group("shop")
				{
					// 列出商品
					shop.GET("skus",
						middleware.RequiredScopes(types.ScopeUserInfoRead),
						controllers.FromQuery[usersvc.SkuListService](usersvc.SkuListParamCtx{}),
						controllers.UserListSkus,
					)
					// 积分购买商品
					shop.POST("purchase",
						middleware.RequiredScopes(types.ScopeUserInfoWrite),
						controllers.FromJSON[usersvc.PurchaseSkuService](usersvc.PurchaseSkuParamCtx{}),
						controllers.UserPurchaseSku,
					)
				}
			}

			// WebDAV and devices
			devices := auth.Group("devices")
			devices.Use(middleware.RequiredScopes(types.ScopeDavAccountRead))
			{
				dav := devices.Group("dav")
				{
					// List WebDAV accounts
					dav.GET("",
						controllers.FromQuery[setting.ListDavAccountsService](setting.ListDavAccountParamCtx{}),
						controllers.ListDavAccounts,
					)
					// Create WebDAV account
					dav.PUT("",
						middleware.RequiredScopes(types.ScopeDavAccountWrite),
						controllers.FromJSON[setting.CreateDavAccountService](setting.CreateDavAccountParamCtx{}),
						controllers.CreateDAVAccounts,
					)
					// Create WebDAV account
					dav.PATCH(":id",
						middleware.RequiredScopes(types.ScopeDavAccountWrite),
						middleware.HashID(hashid.DavAccountID),
						controllers.FromJSON[setting.CreateDavAccountService](setting.CreateDavAccountParamCtx{}),
						controllers.UpdateDAVAccounts,
					)
					// Delete WebDAV account
					dav.DELETE(":id",
						middleware.RequiredScopes(types.ScopeDavAccountWrite),
						middleware.HashID(hashid.DavAccountID),
						controllers.DeleteDAVAccounts,
					)
				}
				//// 获取账号信息
				//devices.GET("dav", controllers.GetWebDAVAccounts)
				//// 删除目录挂载
				//devices.DELETE("mount/:id",
				//	middleware.HashID(hashid.FolderID),
				//	controllers.DeleteWebDAVMounts,
				//)
				//// 创建目录挂载
				//devices.POST("mount", controllers.CreateWebDAVMounts)
				//// 更新账号可读性
				//devices.PATCH("accounts", controllers.UpdateWebDAVAccountsReadonly)
			}

		}

	}

	// 初始化WebDAV相关路由
	initWebDAV(r.Group("dav"))
	return r
}

// initWebDAV 初始化WebDAV相关路由
func initWebDAV(group *gin.RouterGroup) {
	{
		group.Use(middleware.CacheControl(), middleware.WebDAVAuth())
		group.Any("/*path", webdav.ServeHTTP)
		group.Any("", webdav.ServeHTTP)
		group.Handle("PROPFIND", "/*path", webdav.ServeHTTP)
		group.Handle("PROPFIND", "", webdav.ServeHTTP)
		group.Handle("MKCOL", "/*path", webdav.ServeHTTP)
		group.Handle("LOCK", "/*path", webdav.ServeHTTP)
		group.Handle("UNLOCK", "/*path", webdav.ServeHTTP)
		group.Handle("PROPPATCH", "/*path", webdav.ServeHTTP)
		group.Handle("COPY", "/*path", webdav.ServeHTTP)
		group.Handle("MOVE", "/*path", webdav.ServeHTTP)

	}
}
