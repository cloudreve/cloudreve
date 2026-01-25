package middleware

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"net/url"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/cloudreve/Cloudreve/v4/service/explorer"
	"github.com/cloudreve/Cloudreve/v4/service/share"
	"github.com/gin-gonic/gin"
)

const (
	ogStatusInvalidLink = "Invalid Link"
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
		uri, err := fs.NewUriFromString(rawPath)
		if err != nil || uri.FileSystem() != constants.FileSystemShare {
			return "", ""
		}

		return uri.ID(""), uri.Password()
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
		ShareURL:    routes.MasterShareUrl(base, id, password).String(),
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

	shareInfo, err := loadShareForOG(c, shareID, password)
	if err != nil {
		var appErr serializer.AppError
		if errors.As(err, &appErr) {
			data.Description = appErr.Msg
		} else {
			data.Description = ogStatusInvalidLink
		}
		return renderOGHTML(data)
	}

	data.Title = shareInfo.Name
	if shareInfo.SourceType != nil && *shareInfo.SourceType == types.FileTypeFolder {
		data.Description = "Folder"
	} else {
		data.Description = formatFileSize(shareInfo.Size)
		thumbnail, err := loadShareThumbnail(c, id, password, shareInfo)
		if err == nil {
			data.ImageURL = thumbnail
		}
	}

	data.Description += " · " + shareInfo.Owner.Nickname
	return renderOGHTML(data)
}

func loadShareThumbnail(c *gin.Context, shareID, password string, shareInfo *explorer.Share) (string, error) {
	shareUri, err := fs.NewUriFromString(fs.NewShareUri(shareID, password))
	if err != nil {
		return "", fmt.Errorf("failed to construct share uri: %w", err)
	}

	subService := &explorer.FileThumbService{
		Uri: shareUri.Join(shareInfo.Name).String(),
	}

	if err := SetUserCtx(c, 0); err != nil {
		return "", err
	}

	res, err := subService.Get(c)
	if err != nil {
		return "", err
	}

	return res.Url, nil
}

func loadShareForOG(c *gin.Context, shareID int, password string) (*explorer.Share, error) {
	subService := &share.ShareInfoService{
		Password:   password,
		CountViews: false,
	}

	if err := SetUserCtx(c, 0); err != nil {
		return nil, err
	}

	util.WithValue(c, hashid.ObjectIDCtx{}, shareID)
	return subService.Get(c)
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
