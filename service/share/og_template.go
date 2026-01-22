package share

import (
	"bytes"
	"fmt"
	"html/template"
)

// ShareOGData contains data for rendering Open Graph HTML page
type ShareOGData struct {
	SiteName     string
	FileName     string
	FileSize     string
	OwnerName    string
	ShareURL     string
	ThumbnailURL string
	RedirectURL  string
}

const ogHTMLTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta property="og:title" content="{{.FileName}}">
    <meta property="og:description" content="{{.FileSize}}{{if .OwnerName}} · {{.OwnerName}}{{end}}">
    <meta property="og:image" content="{{.ThumbnailURL}}">
    <meta property="og:url" content="{{.ShareURL}}">
    <meta property="og:type" content="website">
    <meta property="og:site_name" content="{{.SiteName}}">
    <meta name="twitter:card" content="summary">
    <meta name="twitter:title" content="{{.FileName}}">
    <meta name="twitter:description" content="{{.FileSize}}{{if .OwnerName}} · {{.OwnerName}}{{end}}">
    <meta name="twitter:image" content="{{.ThumbnailURL}}">
    <title>{{.FileName}} - {{.SiteName}}</title>
</head>
<body>
    <script>window.location.href = "{{.RedirectURL}}";</script>
    <noscript>
        <p><a href="{{.RedirectURL}}">{{.FileName}}</a></p>
    </noscript>
</body>
</html>`

var ogTemplate = template.Must(template.New("og").Parse(ogHTMLTemplate))

// RenderOGHTML renders the Open Graph HTML page
func RenderOGHTML(data *ShareOGData) (string, error) {
	var buf bytes.Buffer
	if err := ogTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render OG template: %w", err)
	}
	return buf.String(), nil
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
