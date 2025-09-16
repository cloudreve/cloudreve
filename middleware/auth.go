package middleware

import (
	"net/http"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver/oss"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"

	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
)

const (
	CallbackFailedStatusCode = http.StatusUnauthorized
)

// SignRequired 验证请求签名
func SignRequired(authInstance auth.Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		var err error
		switch c.Request.Method {
		case http.MethodPut, http.MethodPost, http.MethodPatch:
			err = auth.CheckRequest(c, authInstance, c.Request)
		default:
			err = auth.CheckURI(c, authInstance, c.Request.URL)
		}

		if err != nil {
			c.JSON(200, serializer.ErrWithDetails(c, serializer.CodeCredentialInvalid, err.Error(), err))
			c.Abort()
			return
		}

		c.Next()
	}
}

// CurrentUser 获取登录用户
func CurrentUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		dep := dependency.FromContext(c)
		shouldContinue, err := dep.TokenAuth().VerifyAndRetrieveUser(c)
		if err != nil {
			c.JSON(200, serializer.Err(c, err))
			c.Abort()
			return
		}

		if shouldContinue {
			// TODO: Logto handler
		}

		uid := inventory.UserIDFromContext(c)
		if err := SetUserCtx(c, uid); err != nil {
			c.JSON(200, serializer.Err(c, err))
			c.Abort()
			return
		}

		c.Next()
	}
}

// SetUserCtx set the current login user via uid
func SetUserCtx(c *gin.Context, uid int) error {
	dep := dependency.FromContext(c)
	userClient := dep.UserClient()
	loginUser, err := userClient.GetLoginUserByID(c, uid)
	if err != nil {
		return serializer.NewError(serializer.CodeDBError, "failed to get login user", err)
	}

	SetUserCtxByUser(c, loginUser)
	return nil
}

func SetUserCtxByUser(c *gin.Context, user *ent.User) {
	util.WithValue(c, inventory.UserCtx{}, user)
}

// LoginRequired 需要登录
func LoginRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if u := inventory.UserFromContext(c); u != nil && !inventory.IsAnonymousUser(u) {
			c.Next()
			return
		}

		c.JSON(200, serializer.ErrWithDetails(c, serializer.CodeCheckLogin, "Login required", nil))
		c.Abort()
	}
}

// WebDAVAuth 验证WebDAV登录及权限
func WebDAVAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, password, ok := c.Request.BasicAuth()
		if !ok {
			// OPTIONS 请求不需要鉴权
			if c.Request.Method == http.MethodOptions {
				c.Next()
				return
			}
			c.Writer.Header()["WWW-Authenticate"] = []string{`Basic realm="cloudreve"`}
			c.Status(http.StatusUnauthorized)
			c.Abort()
			return
		}

		dep := dependency.FromContext(c)
		l := dep.Logger()
		userClient := dep.UserClient()
		expectedUser, err := userClient.GetActiveByDavAccount(c, username, password)
		if err != nil {
			if username == "" {
				if u, err := userClient.GetByEmail(c, username); err == nil {
					// Try login with known user but incorrect password, record audit log
					SetUserCtxByUser(c, u)
				}
			}

			l.Debug("WebDAVAuth: failed to get user %q with provided credential: %s", username, err)
			c.Status(http.StatusUnauthorized)
			c.Abort()
			return
		}

		// Validate dav account
		accounts, err := expectedUser.Edges.DavAccountsOrErr()
		if err != nil || len(accounts) == 0 {
			l.Debug("WebDAVAuth: failed to get user dav accounts %q with provided credential: %s", username, err)
			c.Status(http.StatusUnauthorized)
			c.Abort()
			return
		}

		// 用户组已启用WebDAV？
		if !expectedUser.EnforceGroupPermission(types.GroupPermissionWebDAV) {
			c.Status(http.StatusForbidden)
			l.Debug("WebDAVAuth: user %q does not have WebDAV permission.", expectedUser.Email)
			c.Abort()
			return
		}

		// 检查是否只读
		if expectedUser.Edges.DavAccounts[0].Options.Enabled(int(types.DavAccountReadOnly)) {
			switch c.Request.Method {
			case http.MethodDelete, http.MethodPut, "MKCOL", "COPY", "MOVE", "LOCK", "UNLOCK":
				c.Status(http.StatusForbidden)
				c.Abort()
				return
			}
		}

		SetUserCtxByUser(c, expectedUser)
		c.Next()
	}
}

// 对上传会话进行验证
func UseUploadSession(policyType types.PolicyType) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 验证key并查找用户
		err := uploadCallbackCheck(c, policyType)
		if err != nil {
			c.JSON(CallbackFailedStatusCode, serializer.Err(c, err))
			c.Abort()
			return
		}

		c.Next()
	}
}

// uploadCallbackCheck 对上传回调请求的 callback key 进行验证，如果成功则返回上传用户
func uploadCallbackCheck(c *gin.Context, policyType types.PolicyType) error {
	// 验证 Callback Key
	sessionID := c.Param("sessionID")
	if sessionID == "" {
		return serializer.NewError(serializer.CodeParamErr, "Session ID cannot be empty", nil)
	}

	dep := dependency.FromContext(c)
	callbackSessionRaw, exist := dep.KV().Get(manager.UploadSessionCachePrefix + sessionID)
	if !exist {
		return serializer.NewError(serializer.CodeUploadSessionExpired, "Upload session does not exist or expired", nil)
	}

	callbackSession := callbackSessionRaw.(fs.UploadSession)
	c.Set(manager.UploadSessionCtx, &callbackSession)
	if callbackSession.Policy.Type != string(policyType) {
		return serializer.NewError(serializer.CodePolicyNotAllowed, "", nil)
	}

	if err := SetUserCtx(c, callbackSession.UID); err != nil {
		return err
	}

	return nil
}

// RemoteCallbackAuth 远程回调签名验证
func RemoteCallbackAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 验证签名
		session := c.MustGet(manager.UploadSessionCtx).(*fs.UploadSession)
		if session.Policy.Edges.Node == nil {
			c.JSON(CallbackFailedStatusCode, serializer.ErrWithDetails(c, serializer.CodeCredentialInvalid, "Node not found", nil))
			c.Abort()
			return
		}

		authInstance := auth.HMACAuth{SecretKey: []byte(session.Policy.Edges.Node.SlaveKey)}
		if err := auth.CheckRequest(c, authInstance, c.Request); err != nil {
			c.JSON(CallbackFailedStatusCode, serializer.ErrWithDetails(c, serializer.CodeCredentialInvalid, err.Error(), err))
			c.Abort()
			return
		}

		c.Next()

	}
}

// OSSCallbackAuth 阿里云OSS回调签名验证
func OSSCallbackAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		dep := dependency.FromContext(c)
		err := oss.VerifyCallbackSignature(c.Request, dep.KV(), dep.RequestClient(
			request.WithContext(c),
			request.WithLogger(logging.FromContext(c)),
		))
		if err != nil {
			dep.Logger().Debug("Failed to verify callback request: %s", err)
			c.JSON(401, serializer.GeneralUploadCallbackFailed{Error: "Failed to verify callback request."})
			c.Abort()
			return
		}

		c.Next()
	}
}

// IsAdmin 必须为管理员用户组
func IsAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := inventory.UserFromContext(c)
		if !user.EnforceGroupPermission(types.GroupPermissionIsAdmin) {
			c.JSON(200, serializer.ErrWithDetails(c, serializer.CodeNoPermissionErr, "", nil))
			c.Abort()
			return
		}

		c.Next()
	}
}
