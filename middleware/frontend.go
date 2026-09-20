package middleware

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"strings"
)

// FrontendFileHandler 前端静态文件处理
func FrontendFileHandler(dep dependency.Dep) gin.HandlerFunc {
	fs := dep.ServerStaticFS()
	l := dep.Logger()

	ignoreFunc := func(c *gin.Context) {
		c.Next()
	}

	if fs == nil {
		return ignoreFunc
	}

	// 读取index.html
	file, err := fs.Open("/index.html")
	if err != nil {
		l.Warning("Static file \"index.html\" does not exist, it might affect the display of the homepage.")
		return ignoreFunc
	}

	fileContentBytes, err := io.ReadAll(file)
	if err != nil {
		l.Warning("Cannot read static file \"index.html\", it might affect the display of the homepage.")
		return ignoreFunc
	}
	fileContent := string(fileContentBytes)

	fileServer := http.FileServer(fs)
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Skipping routers handled by backend
		if strings.HasPrefix(path, "/api") ||
			strings.HasPrefix(path, "/dav") ||
			strings.HasPrefix(path, "/f/") ||
			strings.HasPrefix(path, "/s/") ||
			path == "/manifest.json" {
			c.Next()
			return
		}

		// 不存在的路径和index.html均返回index.html
		if (path == "/index.html") || (path == "/") || !fs.Exists("/", path) {
			// 读取、替换站点设置
			settingClient := dep.SettingProvider()
			siteBasic := settingClient.SiteBasicLocalized(c, setting.ParseAcceptLanguage(c.GetHeader("Accept-Language")))
			pwaOpts := settingClient.PWA(c)
			theme := settingClient.Theme(c)
			finalHTML := util.Replace(map[string]string{
				"{siteName}":               siteBasic.Name,
				"{siteDes}":                siteBasic.Description,
				"{siteScript}":             siteBasic.Script,
				"{pwa_small_icon}":         pwaOpts.SmallIcon,
				"{pwa_medium_icon}":        pwaOpts.MediumIcon,
				"var(--defaultThemeColor)": theme.DefaultTheme,
			}, fileContent)

			c.Header("Content-Type", "text/html")
			c.Header("Cache-Control", "public, no-cache")
			c.String(200, finalHTML)
			c.Abort()
			return
		}

		if path == "/sw.js" || strings.HasPrefix(path, "/locales/") {
			c.Header("Cache-Control", "public, no-cache")
		} else if strings.HasPrefix(path, "/assets/") {
			c.Header("Cache-Control", "public, max-age=31536000")
		}

		// 存在的静态文件
		fileServer.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}
