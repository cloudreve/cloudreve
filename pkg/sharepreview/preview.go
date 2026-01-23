package sharepreview

import (
	"context"
	"net/url"
	"path"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	sharepkg "github.com/cloudreve/Cloudreve/v4/pkg/share"
	"github.com/gin-gonic/gin"
)

const (
	ogStatusInvalidLink      = "Invalid Link"
	ogStatusShareExpired     = "Share Expired"
	ogStatusPasswordRequired = "Password Required"
	ogDefaultFileName        = "Shared File"
	ogDefaultFolderName      = "Shared Folder"
)

type Options struct {
	ID        string
	Password  string
	SharePath string
}

// RenderOGPage renders an Open Graph HTML page for social media previews.
func RenderOGPage(c *gin.Context, opts Options) (string, error) {
	dep := dependency.FromContext(c)
	shareClient := dep.ShareClient()
	settings := dep.SettingProvider()
	u := inventory.UserFromContext(c)

	// Check if anonymous users have permission to access share links.
	anonymousGroup, err := dep.GroupClient().AnonymousGroup(c)
	needLogin := err != nil || anonymousGroup.Permissions == nil || !anonymousGroup.Permissions.Enabled(int(types.GroupPermissionShareDownload))

	// Get site settings.
	siteBasic := settings.SiteBasic(c)
	pwa := settings.PWA(c)
	base := settings.SiteURL(c)

	sharePath := sharepkg.SanitizeSharePath(opts.SharePath)

	// Build share URL.
	shareURL := routes.MasterShareUrl(base, opts.ID, "")
	if sharePath != "" {
		shareURLQuery := shareURL.Query()
		shareURLQuery.Set("path", sharePath)
		shareURL.RawQuery = shareURLQuery.Encode()
	}

	// Build thumbnail URL - use PWA large icon (PNG), fallback to medium icon.
	thumbnailURL := pwa.LargeIcon
	if thumbnailURL == "" {
		thumbnailURL = pwa.MediumIcon
	}
	if thumbnailURL != "" && !isAbsoluteURL(thumbnailURL) {
		thumbnailURL = base.ResolveReference(&url.URL{Path: thumbnailURL}).String()
	}

	// Prepare OG data.
	ogData := &ShareOGData{
		SiteName:        siteBasic.Name,
		SiteDescription: siteBasic.Description,
		SiteURL:         base.String(),
		ShareURL:        shareURL.String(),
		ShareID:         opts.ID,
		ThumbnailURL:    thumbnailURL,
		RedirectURL:     sharepkg.BuildRedirectURL(opts.ID, opts.Password, sharePath, c.Request.URL.Query()),
	}

	// Get file info (hide for invalid/expired/password-protected/login-required shares).
	shareID, err := dep.HashIDEncoder().Decode(opts.ID, hashid.ShareID)
	if err != nil {
		return renderStatusOG(ogData, siteBasic.Name, ogStatusInvalidLink)
	}

	share, unlocked, status, _ := sharepkg.LoadShareForInfo(c, shareClient, shareID, u, opts.Password)
	switch status {
	case sharepkg.LoadNotFound:
		return renderStatusOG(ogData, siteBasic.Name, ogStatusInvalidLink)
	case sharepkg.LoadExpired:
		return renderStatusOG(ogData, siteBasic.Name, ogStatusShareExpired)
	case sharepkg.LoadError:
		return renderStatusOG(ogData, siteBasic.Name, ogStatusInvalidLink)
	}

	if needLogin {
		return renderStatusOG(ogData, siteBasic.Name, ogStatusShareExpired)
	}

	if share.Password != "" && !unlocked {
		// Password required but not provided or incorrect.
		return renderStatusOG(ogData, siteBasic.Name, ogStatusPasswordRequired)
	}

	if share.Edges.File == nil {
		return renderStatusOG(ogData, siteBasic.Name, ogStatusInvalidLink)
	}

	// Get owner name (don't expose email for privacy).
	if share.Edges.User != nil {
		ogData.OwnerName = share.Edges.User.Nick
	}

	fileType := types.FileType(share.Edges.File.Type)
	fileName := share.Edges.File.Name
	fileSize := share.Edges.File.Size
	if sharePath != "" {
		if fileType != types.FileTypeFolder {
			return renderStatusOG(ogData, siteBasic.Name, ogStatusInvalidLink)
		}

		resolvedFile, err := resolveSharePathFile(c, dep, u, opts.ID, opts.Password, sharePath)
		if err != nil {
			return renderStatusOG(ogData, siteBasic.Name, ogStatusInvalidLink)
		}

		fileType = resolvedFile.Type()
		fileName = resolvedFile.Name()
		if displayName := resolvedFile.DisplayName(); displayName != "" {
			fileName = displayName
		}
		fileSize = resolvedFile.Size()
	}

	if fileType == types.FileTypeFolder {
		ogData.FolderName = defaultIfEmpty(fileName, ogDefaultFolderName)
		ogData.DisplayName = ogData.FolderName
		return RenderFolderOGHTML(ogData, ogFolderTitleTemplate, ogFolderDescTemplate)
	}

	ogData.FileName = defaultIfEmpty(fileName, ogDefaultFileName)
	ogData.FileSize = FormatFileSize(fileSize)
	ogData.FileExt = strings.TrimPrefix(path.Ext(ogData.FileName), ".")
	ogData.DisplayName = ogData.FileName

	return RenderOGHTML(ogData, ogFileTitleTemplate, ogFileDescTemplate)
}

func renderStatusOG(data *ShareOGData, displayName, status string) (string, error) {
	data.Status = status
	data.DisplayName = displayName
	return RenderStatusOGHTML(data, ogStatusTitleTemplate, ogStatusDescTemplate)
}

func defaultIfEmpty(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func resolveSharePathFile(ctx context.Context, dep dependency.Dep, viewer *ent.User, id, password, sharePath string) (fs.File, error) {
	if viewer == nil {
		anonymous, err := dep.UserClient().AnonymousUser(ctx)
		if err != nil {
			return nil, err
		}
		viewer = anonymous
	}

	fm := manager.NewFileManager(dep, viewer)
	defer fm.Recycle()

	shareURI, err := fs.NewUriFromString(fs.NewShareUri(id, password))
	if err != nil {
		return nil, err
	}
	if sharePath != "" {
		shareURI = shareURI.JoinRaw(sharePath)
	}

	return fm.Get(ctx, shareURI)
}

// isAbsoluteURL checks if the URL is absolute (starts with http:// or https://).
func isAbsoluteURL(u string) bool {
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}
