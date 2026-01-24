package middleware

import (
	"bytes"
	"crypto/subtle"
	"fmt"
	"html/template"
	"net/url"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/gin-gonic/gin"
)

const (
	ogStatusInvalidLink      = "Invalid Link"
	ogStatusShareUnavailable = "Share Unavailable"
	ogStatusPasswordRequired = "Password Required"
	ogStatusNeedLogin        = "Login Required"
)

type ogData struct {
	SiteName    string
	Title       string
	Description string
	ImageURL    string
	ShareURL    string
	RedirectURL string
}

const ogHTMLTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta property="og:title" content="{{.Title}}">
    <meta property="og:description" content="{{.Description}}">
    <meta property="og:image" content="{{.ImageURL}}">
    <meta property="og:url" content="{{.ShareURL}}">
    <meta property="og:type" content="website">
    <meta property="og:site_name" content="{{.SiteName}}">
    <meta name="twitter:card" content="summary">
    <meta name="twitter:title" content="{{.Title}}">
    <meta name="twitter:description" content="{{.Description}}">
    <meta name="twitter:image" content="{{.ImageURL}}">
    <title>{{.Title}} - {{.SiteName}}</title>
</head>
<body>
    <script>window.location.href = "{{.RedirectURL}}";</script>
    <noscript><a href="{{.RedirectURL}}">{{.Title}}</a></noscript>
</body>
</html>`

var ogTemplate = template.Must(template.New("og").Parse(ogHTMLTemplate))

var socialMediaBots = []string{
	"facebookexternalhit",
	"facebookcatalog",
	"facebot",
	"twitterbot",
	"linkedinbot",
	"discordbot",
	"telegrambot",
	"slackbot",
	"whatsapp",
}

func isSocialMediaBot(ua string) bool {
	ua = strings.ToLower(ua)
	for _, bot := range socialMediaBots {
		if strings.Contains(ua, bot) {
			return true
		}
	}
	return false
}

// SharePreview 为社交媒体爬虫渲染OG预览页面
func SharePreview(dep dependency.Dep) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isSocialMediaBot(c.GetHeader("User-Agent")) {
			c.Next()
			return
		}

		id, password := extractShareParams(c)
		if id == "" {
			c.Next()
			return
		}

		html := renderShareOGPage(c, dep, id, password)
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Header("Cache-Control", "public, no-cache")
		c.String(200, html)
		c.Abort()
	}
}

func extractShareParams(c *gin.Context) (id, password string) {
	urlPath := c.Request.URL.Path

	if strings.HasPrefix(urlPath, "/s/") {
		parts := strings.Split(strings.TrimPrefix(urlPath, "/s/"), "/")
		if len(parts) >= 1 && parts[0] != "" {
			id = parts[0]
			if len(parts) >= 2 {
				password = parts[1]
			}
		}
	} else if urlPath == "/home" || urlPath == "/home/" {
		rawPath := c.Query("path")
		if strings.HasPrefix(rawPath, "cloudreve://") {
			u, err := url.Parse(rawPath)
			if err != nil {
				return "", ""
			}
			if strings.ToLower(u.Host) != "share" || u.User == nil {
				return "", ""
			}
			id = u.User.Username()
			password, _ = u.User.Password()
		}
	}

	if len(id) > 32 || len(password) > 32 {
		return "", ""
	}
	return id, password
}

func renderShareOGPage(c *gin.Context, dep dependency.Dep, id, password string) string {
	settings := dep.SettingProvider()
	siteBasic := settings.SiteBasic(c)
	pwa := settings.PWA(c)
	base := settings.SiteURL(c)

	data := &ogData{
		SiteName:    siteBasic.Name,
		Title:       siteBasic.Name,
		Description: siteBasic.Description,
		ShareURL:    routes.MasterShareUrl(base, id, "").String(),
		RedirectURL: routes.MasterShareLongUrl(id, password).String(),
	}

	if pwa.LargeIcon != "" {
		data.ImageURL = resolveURL(base, pwa.LargeIcon)
	} else if pwa.MediumIcon != "" {
		data.ImageURL = resolveURL(base, pwa.MediumIcon)
	}

	shareID, err := dep.HashIDEncoder().Decode(id, hashid.ShareID)
	if err != nil {
		data.Description = ogStatusInvalidLink
		return renderOGHTML(data)
	}

	share, err := loadShareForOG(c, dep, shareID)
	if err != nil {
		if ent.IsNotFound(err) {
			data.Description = ogStatusInvalidLink
		} else {
			data.Description = ogStatusShareUnavailable
		}
		return renderOGHTML(data)
	}

	if err := inventory.IsValidShare(share); err != nil {
		data.Description = ogStatusShareUnavailable
		return renderOGHTML(data)
	}

	if !canAnonymousAccessShare(c, dep) {
		data.Description = ogStatusNeedLogin
		return renderOGHTML(data)
	}

	if share.Password != "" && subtle.ConstantTimeCompare([]byte(share.Password), []byte(password)) != 1 {
		data.Description = ogStatusPasswordRequired
		return renderOGHTML(data)
	}

	targetFile := share.Edges.File
	if targetFile != nil {
		fileName := targetFile.Name
		fileType := types.FileType(targetFile.Type)

		data.Title = fileName
		if fileType == types.FileTypeFolder {
			data.Description = "Folder"
		} else {
			data.Description = formatFileSize(targetFile.Size)
		}

		if share.Edges.User != nil && share.Edges.User.Nick != "" {
			data.Description += " · " + share.Edges.User.Nick
		}
	}

	return renderOGHTML(data)
}

func loadShareForOG(c *gin.Context, dep dependency.Dep, shareID int) (*ent.Share, error) {
	ctx := inventory.WithShareEagerLoad(c)
	return dep.ShareClient().GetByID(ctx, shareID)
}

func canAnonymousAccessShare(c *gin.Context, dep dependency.Dep) bool {
	anonymousGroup, err := dep.GroupClient().AnonymousGroup(c)
	if err != nil {
		return false
	}
	return anonymousGroup.Permissions.Enabled(int(types.GroupPermissionShareDownload))
}

func renderOGHTML(data *ogData) string {
	var buf bytes.Buffer
	if err := ogTemplate.Execute(&buf, data); err != nil {
		return ""
	}
	return buf.String()
}

func resolveURL(base *url.URL, path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return base.ResolveReference(&url.URL{Path: path}).String()
}

func formatFileSize(size int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case size >= TB:
		return fmt.Sprintf("%.2f TB", float64(size)/TB)
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d B", size)
	}
}
