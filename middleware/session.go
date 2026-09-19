package middleware

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/pkg/sessionstore"
	"net/http"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// Store session存储
var Store sessions.Store

const SessionName = "cloudreve-session"

// Session 初始化session
func Session(dep dependency.Dep) gin.HandlerFunc {
	Store = sessionstore.NewStore(dep.KV(), []byte(dep.ConfigProvider().System().SessionSecret))
	sameSiteMode := http.SameSiteDefaultMode
	switch strings.ToLower(dep.ConfigProvider().Cors().SameSite) {
	case "default":
		sameSiteMode = http.SameSiteDefaultMode
	case "none":
		sameSiteMode = http.SameSiteNoneMode
	case "strict":
		sameSiteMode = http.SameSiteStrictMode
	case "lax":
		sameSiteMode = http.SameSiteLaxMode
	}

	// Also set Secure: true if using SSL, you should though
	Store.Options(sessions.Options{
		HttpOnly: true,
		MaxAge:   60 * 86400,
		Path:     "/",
		SameSite: sameSiteMode,
		Secure:   dep.ConfigProvider().Cors().Secure,
	})

	return sessions.Sessions(SessionName, Store)
}

// CSRFInit 初始化CSRF标记
func CSRFInit() gin.HandlerFunc {
	return func(c *gin.Context) {
		util.SetSession(c, map[string]interface{}{"CSRF": true})
		c.Next()
	}
}

// CSRFCheck 检查CSRF标记
func CSRFCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		if check, ok := util.GetSession(c, "CSRF").(bool); ok && check {
			c.Next()
			return
		}

		c.JSON(200, serializer.ErrWithDetails(c, serializer.CodeNoPermissionErr, "Invalid origin", nil))
		c.Abort()
	}
}
