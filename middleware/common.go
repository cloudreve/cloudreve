package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth/requestinfo"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

// HashID 将给定对象的HashID转换为真实ID
func HashID(IDType int) gin.HandlerFunc {
	return func(c *gin.Context) {
		dep := dependency.FromContext(c)
		if c.Param("id") != "" {
			id, err := dep.HashIDEncoder().Decode(c.Param("id"), IDType)
			if err == nil {
				util.WithValue(c, hashid.ObjectIDCtx{}, id)
				c.Next()
				return
			}
			c.JSON(200, serializer.ParamErr(c, "Failed to parse object ID", err))
			c.Abort()
			return

		}
		c.Next()
	}
}

// IsFunctionEnabled 当功能未开启时阻止访问
func IsFunctionEnabled(check func(c *gin.Context) bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !check(c) {
			c.JSON(200, serializer.ErrWithDetails(c, serializer.CodeFeatureNotEnabled, "This feature is not enabled", nil))
			c.Abort()
			return
		}

		c.Next()
	}
}

// CacheControl 屏蔽客户端缓存
func CacheControl() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "private, no-cache")
	}
}

func Sandbox() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Security-Policy", "sandbox")
	}
}

// StaticResourceCache 使用静态资源缓存策略
func StaticResourceCache(dep dependency.Dep) gin.HandlerFunc {
	settings := dep.SettingProvider()
	return func(c *gin.Context) {
		c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", settings.PublicResourceMaxAge(c)))

	}
}

// MobileRequestOnly
func MobileRequestOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		dep := dependency.FromContext(c)
		if c.GetHeader(constants.CrHeaderPrefix+"ios") == "" {
			c.Redirect(http.StatusMovedPermanently, dep.SettingProvider().SiteURL(c).String())
			c.Abort()
			return
		}

		c.Next()
	}
}

// InitializeHandling is added at the beginning of handler chain, it did following setups:
// 1. Inject dependency manager into request context
// 2. Generate and inject correlation ID for diagnostic.
func InitializeHandling(dep dependency.Dep) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIp := c.ClientIP()
		if idx := strings.Index(clientIp, ","); idx > 0 {
			clientIp = clientIp[:idx]
		}

		reqInfo := &requestinfo.RequestInfo{
			IP:        clientIp,
			Host:      c.Request.Host,
			UserAgent: c.Request.UserAgent(),
		}
		cid := uuid.FromStringOrNil(c.GetHeader(request.CorrelationHeader))
		if cid == uuid.Nil {
			cid = uuid.Must(uuid.NewV4())
		}

		l := dep.Logger().CopyWithPrefix(fmt.Sprintf("[Cid: %s]", cid))
		ctx := dep.ForkWithLogger(c.Request.Context(), l)
		ctx = context.WithValue(ctx, logging.CorrelationIDCtx{}, cid)
		ctx = context.WithValue(ctx, requestinfo.RequestInfoCtx{}, reqInfo)
		ctx = context.WithValue(ctx, logging.LoggerCtx{}, l)
		if id := c.Param("nodeId"); id != "" {
			ctx = context.WithValue(ctx, cluster.SlaveNodeIDCtx{}, id)
		} else {
			ctx = context.WithValue(ctx, cluster.SlaveNodeIDCtx{}, c.GetHeader(request.SlaveNodeIDHeader))
		}
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// InitializeHandlingSlave retrieves coll correlation ID and other metadata from request header
func InitializeHandlingSlave() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), cluster.MasterSiteIDCtx{}, c.GetHeader(request.SiteIDHeader))
		ctx = context.WithValue(ctx, cluster.MasterSiteUrlCtx{}, c.GetHeader(request.SiteURLHeader))
		ctx = context.WithValue(ctx, cluster.MasterSiteVersionCtx{}, c.GetHeader(request.SiteVersionHeader))
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// Logging logs incoming request info
func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		if raw != "" {
			path = path + "?" + raw
		}

		l := logging.FromContext(c)
		logging.Request(l, true, c.Writer.Status(), c.Request.Method, c.ClientIP(), path,
			c.Errors.ByType(gin.ErrorTypePrivate).String(), start)
	}
}
