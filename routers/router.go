package routers

import (
	"net/http"

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
	adminsvc "github.com/cloudreve/Cloudreve/v4/service/admin"
	"github.com/cloudreve/Cloudreve/v4/service/basic"
	"github.com/cloudreve/Cloudreve/v4/service/explorer"
	"github.com/cloudreve/Cloudreve/v4/service/node"
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
		{
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
					middleware.CaptchaRequired(func(c *gin.Context) bool {
						return dep.SettingProvider().LoginCaptchaEnabled(c)
					}),
					controllers.FromJSON[usersvc.UserLoginService](usersvc.LoginParameterCtx{}),
					controllers.UserLoginValidation,
					controllers.UserIssueToken,
				)
				// 2-factor authentication
				token.POST("2fa",
					controllers.FromJSON[usersvc.OtpValidationService](usersvc.OtpValidationParameterCtx{}),
					controllers.UserLogin2FAValidation,
					controllers.UserIssueToken,
				)
				token.POST("refresh",
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

			authn := session.Group("authn")
			{
				// WebAuthn login prepare
				authn.PUT("",
					middleware.IsFunctionEnabled(func(c *gin.Context) bool {
						return dep.SettingProvider().AuthnEnabled(c)
					}),
					controllers.StartLoginAuthn,
				)
				// WebAuthn finish login
				authn.POST("",
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
				middleware.HashID(hashid.UserID),
				controllers.FromJSON[usersvc.UserResetService](usersvc.UserResetParameterCtx{}),
				controllers.UserReset,
			)
			// 发送密码重设邮件
			user.POST("reset",
				middleware.CaptchaRequired(func(c *gin.Context) bool {
					return dep.SettingProvider().ForgotPasswordCaptchaEnabled(c)
				}),
				controllers.FromJSON[usersvc.UserResetEmailService](usersvc.UserResetEmailParameterCtx{}),
				controllers.UserSendReset,
			)
			// 邮件激活 Done
			user.GET("activate/:id",
				middleware.SignRequired(dep.GeneralAuth()),
				middleware.HashID(hashid.UserID),
				controllers.UserActivate,
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
			// Create task to create an archive file
			wf.POST("archive",
				controllers.FromJSON[explorer.ArchiveWorkflowService](explorer.CreateArchiveParamCtx{}),
				controllers.CreateArchive,
			)
			// Create task to extract an archive file
			wf.POST("extract",
				controllers.FromJSON[explorer.ArchiveWorkflowService](explorer.CreateArchiveParamCtx{}),
				controllers.ExtractArchive,
			)

			remoteDownload := wf.Group("download")
			{
				// Create task to download a file
				remoteDownload.POST("",
					controllers.FromJSON[explorer.DownloadWorkflowService](explorer.CreateDownloadParamCtx{}),
					controllers.CreateRemoteDownload,
				)
				// Set download target
				remoteDownload.PATCH(":id",
					middleware.HashID(hashid.TaskID),
					controllers.FromJSON[explorer.SetDownloadFilesService](explorer.SetDownloadFilesParamCtx{}),
					controllers.SetDownloadTaskTarget,
				)
				remoteDownload.DELETE(":id",
					middleware.HashID(hashid.TaskID),
					controllers.CancelDownloadTask,
				)
			}
		}

		// 文件
		file := v4.Group("file")
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
				controllers.FromJSON[explorer.CreateFileService](explorer.CreateFileParameterCtx{}),
				controllers.CreateFile,
			)
			// Rename file
			file.POST("rename",
				controllers.FromJSON[explorer.RenameFileService](explorer.RenameFileParameterCtx{}),
				controllers.RenameFile,
			)
			// Move or copy files
			file.POST("move",
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
				controllers.FromQuery[explorer.FileUpdateService](explorer.FileUpdateParameterCtx{}),
				controllers.PutContent)
			// Get entity content for preview/download
			content := file.Group("content")
			contentCors := cors.New(cors.Config{
				AllowOrigins: []string{"*"},
				AllowMethods: []string{"GET", "HEAD", "OPTIONS"},
				AllowHeaders: []string{"Range", "If-Range", "Authorization", "Content-Type"},
				ExposeHeaders: []string{"Content-Range", "Accept-Ranges", "Content-Length", "Content-Disposition"},
				AllowCredentials: false, // 使用 "*" 时必须为 false
			})
			content.Use(contentCors)
			{
				content.OPTIONS("*option", contentCors)
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
				controllers.FromJSON[explorer.DeleteFileService](explorer.DeleteFileParameterCtx{}),
				middleware.ValidateBatchFileCount(dep, explorer.DeleteFileParameterCtx{}),
				controllers.Delete,
			)
			// Force unlock
			file.DELETE("lock",
				controllers.FromJSON[explorer.UnlockFileService](explorer.UnlockFileParameterCtx{}),
				controllers.Unlock,
			)
			// Restore files
			file.POST("restore",
				controllers.FromJSON[explorer.DeleteFileService](explorer.DeleteFileParameterCtx{}),
				middleware.ValidateBatchFileCount(dep, explorer.DeleteFileParameterCtx{}),
				controllers.Restore,
			)
			// Patch metadata
			file.PATCH("metadata",
				controllers.FromJSON[explorer.PatchMetadataService](explorer.PatchMetadataParameterCtx{}),
				middleware.ValidateBatchFileCount(dep, explorer.PatchMetadataParameterCtx{}),
				controllers.PatchMetadata,
			)
			// Upload related
			upload := file.Group("upload")
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
			pin := file.Group("pin")
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
			// Version management
			version := file.Group("version")
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
				controllers.FromJSON[explorer.CreateViewerSessionService](explorer.CreateViewerSessionParamCtx{}),
				controllers.CreateViewerSession,
			)
			// Create task to import files
			wf.POST("import",
				middleware.IsAdmin(),
				controllers.FromJSON[explorer.ImportWorkflowService](explorer.CreateImportParamCtx{}),
				controllers.ImportFiles,
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
				controllers.FromJSON[explorer.PatchViewService](explorer.PatchViewParameterCtx{}),
				controllers.PatchView,
			)

			// Server event push
			file.GET("events",
				middleware.LoginRequired(),
				controllers.FromQuery[explorer.ExplorerEventService](explorer.ExplorerEventParamCtx{}),
				controllers.HandleExplorerEventsPush,
			)
		}

		// 分享相关
		share := v4.Group("share")
		{
			// Create share link
			share.PUT("",
				middleware.LoginRequired(),
				controllers.FromJSON[sharesvc.ShareCreateService](sharesvc.ShareCreateParamCtx{}),
				controllers.CreateShare,
			)
			// Edit existing share link
			share.POST(":id",
				middleware.LoginRequired(),
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
			// List my shares
			share.GET("",
				middleware.LoginRequired(),
				controllers.FromQuery[sharesvc.ListShareService](sharesvc.ListShareParamCtx{}),
				controllers.ListShare,
			)
			// 删除分享
			share.DELETE(":id",
				middleware.LoginRequired(),
				middleware.HashID(hashid.ShareID),
				controllers.DeleteShare,
			)
			//// 获取README文本文件内容
			//share.GET("readme/:id",
			//	middleware.CheckShareUnlocked(),
			//	controllers.PreviewShareReadme,
			//)
			//// 举报分享
			//share.POST("report/:id",
			//	middleware.CheckShareUnlocked(),
			//	controllers.ReportShare,
			//)
		}

		// 需要登录保护的
		auth := v4.Group("")
		auth.Use(middleware.LoginRequired())
		{
			// 管理
			admin := auth.Group("admin", middleware.IsAdmin())
			{
				admin.GET("summary",
					controllers.FromQuery[adminsvc.SummaryService](adminsvc.SummaryParamCtx{}),
					controllers.AdminSummary,
				)

				settings := admin.Group("settings")
				{
					// Get settings
					settings.POST("",
						controllers.FromJSON[adminsvc.GetSettingService](adminsvc.GetSettingParamCtx{}),
						controllers.AdminGetSettings,
					)
					// Patch settings
					settings.PATCH("",
						controllers.FromJSON[adminsvc.SetSettingService](adminsvc.SetSettingParamCtx{}),
						controllers.AdminSetSettings,
					)
				}

				// 用户组管理
				group := admin.Group("group")
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
						controllers.FromJSON[adminsvc.UpsertGroupService](adminsvc.UpsertGroupParamCtx{}),
						controllers.AdminCreateGroup,
					)
					// 更新用户组
					group.PUT(":id",
						controllers.FromJSON[adminsvc.UpsertGroupService](adminsvc.UpsertGroupParamCtx{}),
						controllers.AdminUpdateGroup,
					)
					// 删除用户组
					group.DELETE(":id",
						controllers.FromUri[adminsvc.SingleGroupService](adminsvc.SingleGroupParamCtx{}),
						controllers.AdminDeleteGroup,
					)
				}

				tool := admin.Group("tool")
				{
					tool.GET("wopi",
						controllers.FromQuery[adminsvc.FetchWOPIDiscoveryService](adminsvc.FetchWOPIDiscoveryParamCtx{}),
						controllers.AdminFetchWopi,
					)
					tool.POST("thumbExecutable",
						controllers.FromJSON[adminsvc.ThumbGeneratorTestService](adminsvc.ThumbGeneratorTestParamCtx{}),
						controllers.AdminTestThumbGenerator)
					tool.POST("mail",
						controllers.FromJSON[adminsvc.TestSMTPService](adminsvc.TestSMTPParamCtx{}),
						controllers.AdminSendTestMail,
					)
					tool.DELETE("entityUrlCache",
						controllers.AdminClearEntityUrlCache,
					)
				}

				queue := admin.Group("queue")
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
						controllers.FromJSON[adminsvc.BatchTaskService](adminsvc.BatchTaskParamCtx{}),
						controllers.AdminBatchDeleteTask,
					)
					// Cleanup tasks
					queue.POST("cleanup",
						controllers.FromJSON[adminsvc.CleanupTaskService](adminsvc.CleanupTaskParameterCtx{}),
						controllers.AdminCleanupTask,
					)
					// // 列出任务
					// queue.POST("list", controllers.AdminListTask)
					// // 新建文件导入任务
					// queue.POST("import", controllers.AdminCreateImportTask)
				}

				// 存储策略管理
				policy := admin.Group("policy")
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
						controllers.FromJSON[adminsvc.CreateStoragePolicyService](adminsvc.CreateStoragePolicyParamCtx{}),
						controllers.AdminCreatePolicy,
					)
					// 更新存储策略
					policy.PUT(":id",
						controllers.FromJSON[adminsvc.UpdateStoragePolicyService](adminsvc.UpdateStoragePolicyParamCtx{}),
						controllers.AdminUpdatePolicy,
					)
					// 创建跨域策略
					policy.POST("cors",
						controllers.FromJSON[adminsvc.CreateStoragePolicyCorsService](adminsvc.CreateStoragePolicyCorsParamCtx{}),
						controllers.AdminCreateStoragePolicyCors,
					)
					// // 获取 OneDrive OAuth URL
					oauth := policy.Group("oauth")
					{
						// 获取 OneDrive OAuth URL
						oauth.POST("signin",
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
						controllers.FromUri[adminsvc.SingleStoragePolicyService](adminsvc.GetStoragePolicyParamCtx{}),
						controllers.AdminDeletePolicy,
					)
				}

				node := admin.Group("node")
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
						controllers.FromJSON[adminsvc.TestNodeService](adminsvc.TestNodeParamCtx{}),
						controllers.AdminTestSlave,
					)
					node.POST("test/downloader",
						controllers.FromJSON[adminsvc.TestNodeDownloaderService](adminsvc.TestNodeDownloaderParamCtx{}),
						controllers.AdminTestDownloader,
					)
					node.PUT("",
						controllers.FromJSON[adminsvc.UpsertNodeService](adminsvc.UpsertNodeParamCtx{}),
						controllers.AdminCreateNode,
					)
					node.PUT(":id",
						controllers.FromJSON[adminsvc.UpsertNodeService](adminsvc.UpsertNodeParamCtx{}),
						controllers.AdminUpdateNode,
					)
					node.DELETE(":id",
						controllers.FromUri[adminsvc.SingleNodeService](adminsvc.SingleNodeParamCtx{}),
						controllers.AdminDeleteNode,
					)
				}

				user := admin.Group("user")
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
						controllers.FromJSON[adminsvc.UpsertUserService](adminsvc.UpsertUserParamCtx{}),
						controllers.AdminUpdateUser,
					)
					// 创建用户
					user.PUT("",
						controllers.FromJSON[adminsvc.UpsertUserService](adminsvc.UpsertUserParamCtx{}),
						controllers.AdminCreateUser,
					)
					batch := user.Group("batch")
					{
						// 批量删除用户
						batch.POST("delete",
							controllers.FromJSON[adminsvc.BatchUserService](adminsvc.BatchUserParamCtx{}),
							controllers.AdminDeleteUser,
						)
					}
					user.POST(":id/calibrate",
						controllers.FromUri[adminsvc.SingleUserService](adminsvc.SingleUserParamCtx{}),
						controllers.AdminCalibrateStorage,
					)
				}

				file := admin.Group("file")
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
						controllers.FromJSON[adminsvc.BatchFileService](adminsvc.BatchFileParamCtx{}),
						controllers.AdminBatchDeleteFile,
					)
				}

				entity := admin.Group("entity")
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
						controllers.FromJSON[adminsvc.BatchEntityService](adminsvc.BatchEntityParamCtx{}),
						controllers.AdminBatchDeleteEntity,
					)
					// Get entity url
					entity.GET("url/:id",
						controllers.FromUri[adminsvc.SingleEntityService](adminsvc.SingleEntityParamCtx{}),
						controllers.AdminGetEntityUrl,
					)
				}

				share := admin.Group("share")
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
						controllers.FromJSON[adminsvc.BatchShareService](adminsvc.BatchShareParamCtx{}),
						controllers.AdminBatchDeleteShare,
					)
				}
			}

			// 用户
			user := auth.Group("user")
			{
				// 当前登录用户信息
				user.GET("me", controllers.UserMe)
				// 存储信息
				user.GET("capacity", controllers.UserStorage)
				// Search user by keywords
				user.GET("search",
					controllers.FromQuery[usersvc.SearchUserService](usersvc.SearchUserParamCtx{}),
					controllers.UserSearch,
				)

				// WebAuthn 注册相关
				authn := user.Group("authn",
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
				{
					// 获取当前用户设定
					setting.GET("", controllers.UserSetting)
					// 从文件上传头像
					setting.PUT("avatar", controllers.UploadAvatar)
					// 更改用户设定
					setting.PATCH("",
						controllers.FromJSON[usersvc.PatchUserSetting](usersvc.PatchUserSettingParamsCtx{}),
						controllers.UpdateOption,
					)
					// 获得二步验证初始化信息
					setting.GET("2fa", controllers.UserInit2FA)
				}
			}

			// WebDAV and devices
			devices := auth.Group("devices")
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
						controllers.FromJSON[setting.CreateDavAccountService](setting.CreateDavAccountParamCtx{}),
						controllers.CreateDAVAccounts,
					)
					// Create WebDAV account
					dav.PATCH(":id",
						middleware.HashID(hashid.DavAccountID),
						controllers.FromJSON[setting.CreateDavAccountService](setting.CreateDavAccountParamCtx{}),
						controllers.UpdateDAVAccounts,
					)
					// Delete WebDAV account
					dav.DELETE(":id",
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
