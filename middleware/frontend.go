package middleware

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	filemanagerfs "github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/sharepreview"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
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

		if renderSharePreviewForBot(c) {
			return
		}

		// 不存在的路径和index.html均返回index.html
		if path == "/index.html" || path == "/" || !fs.Exists("/", path) {
			// 读取、替换站点设置
			settingClient := dep.SettingProvider()
			siteBasic := settingClient.SiteBasic(c)
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

func renderSharePreviewForBot(c *gin.Context) bool {
	path := c.Request.URL.Path
	if path != "/home" && path != "/home/" {
		return false
	}
	if !util.IsSocialMediaBot(c.GetHeader("User-Agent")) {
		return false
	}
	shareURI := parseShareURI(c.Query("path"))
	if shareURI == nil {
		return false
	}
	shareID := shareURI.ID("")
	if shareID == "" {
		return false
	}

	html, err := sharepreview.RenderOGPage(c, sharepreview.Options{
		ID:        shareID,
		Password:  shareURI.Password(),
		SharePath: shareURI.PathTrimmed(),
	})
	if err != nil {
		return false
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
	c.Abort()
	return true
}

func parseShareURI(raw string) *filemanagerfs.URI {
	if raw == "" {
		return nil
	}
	shareURI, err := filemanagerfs.NewUriFromString(raw)
	if err != nil {
		sanitized := sanitizeInvalidPercentEscapes(raw)
		if sanitized != raw {
			shareURI, err = filemanagerfs.NewUriFromString(sanitized)
		}
	}
	if err != nil && strings.Contains(raw, "%") {
		unescaped, err := url.QueryUnescape(raw)
		if err == nil {
			shareURI, err = filemanagerfs.NewUriFromString(unescaped)
			if err != nil {
				sanitized := sanitizeInvalidPercentEscapes(unescaped)
				if sanitized != unescaped {
					shareURI, err = filemanagerfs.NewUriFromString(sanitized)
				}
			}
		}
	}
	if err != nil {
		return nil
	}
	if shareURI.FileSystem() != constants.FileSystemShare {
		return nil
	}
	return shareURI
}

func sanitizeInvalidPercentEscapes(raw string) string {
	if !strings.Contains(raw, "%") {
		return raw
	}

	var b strings.Builder
	b.Grow(len(raw))
	for i := 0; i < len(raw); i++ {
		if raw[i] != '%' {
			b.WriteByte(raw[i])
			continue
		}
		if i+2 < len(raw) && isHexDigit(raw[i+1]) && isHexDigit(raw[i+2]) {
			b.WriteByte('%')
			b.WriteByte(raw[i+1])
			b.WriteByte(raw[i+2])
			i += 2
			continue
		}
		// Replace stray % to avoid url.Parse failures.
		b.WriteString("%25")
	}
	return b.String()
}

func isHexDigit(b byte) bool {
	return ('0' <= b && b <= '9') || ('a' <= b && b <= 'f') || ('A' <= b && b <= 'F')
}
