package share

import (
	"bytes"
	"fmt"
	"html/template"
	"regexp"
	"strings"
)

// ShareOGData contains data for rendering Open Graph HTML page.
type ShareOGData struct {
	SiteName        string
	SiteDescription string
	SiteURL         string
	ShareURL        string
	ShareID         string
	FileName        string
	FileSize        string
	FileExt         string
	FolderName      string
	OwnerName       string
	Status          string
	Title           string
	Description     string
	DisplayName     string
	ThumbnailURL    string
	RedirectURL     string
}

const ogHTMLTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta property="og:title" content="{{.Title}}">
    <meta property="og:description" content="{{.Description}}">
    <meta property="og:image" content="{{.ThumbnailURL}}">
    <meta property="og:url" content="{{.ShareURL}}">
    <meta property="og:type" content="website">
    <meta property="og:site_name" content="{{.SiteName}}">
    <meta name="twitter:card" content="summary">
    <meta name="twitter:title" content="{{.Title}}">
    <meta name="twitter:description" content="{{.Description}}">
    <meta name="twitter:image" content="{{.ThumbnailURL}}">
    <title>{{.Title}} - {{.SiteName}}</title>
</head>
<body>
    <script>window.location.href = "{{.RedirectURL}}";</script>
    <noscript>
        <p><a href="{{.RedirectURL}}">{{.DisplayName}}</a></p>
    </noscript>
</body>
</html>`

var ogTemplate = template.Must(template.New("og").Parse(ogHTMLTemplate))
var ogMagicVarRe = regexp.MustCompile(`\{[^{}]+\}`)
var ogOwnerSeparatorRe = regexp.MustCompile(`\s*·\s*·+\s*`)

const (
	ogFileTitleTemplate   = "{file_name}"
	ogFileDescTemplate    = "{file_size} · {owner_name}"
	ogFolderTitleTemplate = "{folder_name}"
	ogFolderDescTemplate  = "Folder · {owner_name}"
	ogStatusTitleTemplate = "{site_name}"
	ogStatusDescTemplate  = "{status}"
)

// RenderOGHTML renders the Open Graph HTML page for file shares.
func RenderOGHTML(data *ShareOGData, titleTemplate, descTemplate string) (string, error) {
	title := replaceOGFileMagicVars(titleTemplate, data)
	description := replaceOGFileMagicVars(descTemplate, data)
	return renderOGHTML(data, title, description)
}

// RenderFolderOGHTML renders the Open Graph HTML page for folder shares.
func RenderFolderOGHTML(data *ShareOGData, titleTemplate, descTemplate string) (string, error) {
	title := replaceOGFolderMagicVars(titleTemplate, data)
	description := replaceOGFolderMagicVars(descTemplate, data)
	return renderOGHTML(data, title, description)
}

// RenderStatusOGHTML renders the Open Graph HTML page for share status scenarios.
func RenderStatusOGHTML(data *ShareOGData, titleTemplate, descTemplate string) (string, error) {
	title := replaceOGStatusMagicVars(titleTemplate, data)
	description := replaceOGStatusMagicVars(descTemplate, data)
	return renderOGHTML(data, title, description)
}

func renderOGHTML(data *ShareOGData, title, description string) (string, error) {
	renderData := *data
	renderData.Title = title
	renderData.Description = description

	var buf bytes.Buffer
	if err := ogTemplate.Execute(&buf, &renderData); err != nil {
		return "", fmt.Errorf("failed to render OG template: %w", err)
	}
	return buf.String(), nil
}

func replaceOGFileMagicVars(tmpl string, data *ShareOGData) string {
	vars := map[string]string{
		"{site_name}":        data.SiteName,
		"{site_description}": data.SiteDescription,
		"{site_url}":         data.SiteURL,
		"{file_name}":        data.FileName,
		"{file_size}":        data.FileSize,
		"{file_ext}":         data.FileExt,
		"{owner_name}":       data.OwnerName,
		"{share_url}":        data.ShareURL,
		"{share_id}":         data.ShareID,
	}
	return replaceOGMagicVars(tmpl, vars, data.OwnerName, true)
}

func replaceOGFolderMagicVars(tmpl string, data *ShareOGData) string {
	vars := map[string]string{
		"{site_name}":        data.SiteName,
		"{site_description}": data.SiteDescription,
		"{site_url}":         data.SiteURL,
		"{folder_name}":      data.FolderName,
		"{owner_name}":       data.OwnerName,
		"{share_url}":        data.ShareURL,
		"{share_id}":         data.ShareID,
	}
	return replaceOGMagicVars(tmpl, vars, data.OwnerName, true)
}

func replaceOGStatusMagicVars(tmpl string, data *ShareOGData) string {
	vars := map[string]string{
		"{site_name}":        data.SiteName,
		"{site_description}": data.SiteDescription,
		"{site_url}":         data.SiteURL,
		"{share_url}":        data.ShareURL,
		"{share_id}":         data.ShareID,
		"{status}":           data.Status,
	}
	return replaceOGMagicVars(tmpl, vars, "", false)
}

func replaceOGMagicVars(tmpl string, vars map[string]string, ownerName string, trimOwner bool) string {
	rendered := ogMagicVarRe.ReplaceAllStringFunc(tmpl, func(match string) string {
		if value, ok := vars[match]; ok {
			return value
		}
		return match
	})
	if trimOwner {
		return trimEmptyOwnerSeparator(tmpl, rendered, ownerName)
	}
	return rendered
}

func trimEmptyOwnerSeparator(tmpl, rendered, ownerName string) string {
	if ownerName != "" || !strings.Contains(tmpl, "{owner_name}") {
		return rendered
	}

	trimmed := strings.TrimSpace(rendered)
	trimmed = ogOwnerSeparatorRe.ReplaceAllString(trimmed, " · ")
	trimmed = strings.TrimSpace(trimmed)
	trimmed = strings.Trim(trimmed, "\u00b7")
	return strings.TrimSpace(trimmed)
}

// FormatFileSize formats file size to human readable format
func FormatFileSize(size int64) string {
	const (
		B  = 1
		KB = 1024 * B
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
