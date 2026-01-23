package share

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
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/service/explorer"
	"github.com/gin-gonic/gin"
)

type (
	ShortLinkRedirectService struct {
		ID       string `uri:"id" binding:"required"`
		Password string `uri:"password"`
	}
	ShortLinkRedirectParamCtx struct{}
)

const (
	ogStatusInvalidLink      = "Invalid Link"
	ogStatusNeedLogin        = "Need Login"
	ogStatusShareExpired     = "Share Expired"
	ogStatusPasswordRequired = "Password Required"
	ogDefaultFileName        = "Shared File"
	ogDefaultFolderName      = "Shared Folder"
)

func (s *ShortLinkRedirectService) RedirectTo(c *gin.Context) string {
	shareLongUrl := routes.MasterShareLongUrl(s.ID, s.Password)

	shortLinkQuery := c.Request.URL.Query() // Query in ShortLink, adapt to Cloudreve V3
	shareLongUrlQuery := shareLongUrl.Query()

	userSpecifiedPath := shortLinkQuery.Get("path")
	if userSpecifiedPath != "" {
		masterPath := shareLongUrlQuery.Get("path")
		masterPath += "/" + strings.TrimPrefix(userSpecifiedPath, "/")

		shareLongUrlQuery.Set("path", masterPath)
	}

	shortLinkQuery.Del("path") // 防止用户指定的 Path 就是空字符串
	for k, vals := range shortLinkQuery {
		shareLongUrlQuery[k] = append(shareLongUrlQuery[k], vals...)
	}

	shareLongUrl.RawQuery = shareLongUrlQuery.Encode()
	return shareLongUrl.String()
}

// RenderOGPage renders an Open Graph HTML page for social media previews
func (s *ShortLinkRedirectService) RenderOGPage(c *gin.Context) (string, error) {
	dep := dependency.FromContext(c)
	shareClient := dep.ShareClient()
	settings := dep.SettingProvider()
	u := inventory.UserFromContext(c)

	// Check if anonymous users have permission to access share links
	anonymousGroup, err := dep.GroupClient().AnonymousGroup(c)
	needLogin := err != nil || anonymousGroup.Permissions == nil || !anonymousGroup.Permissions.Enabled(int(types.GroupPermissionShareDownload))

	// Load share with user and file info
	ctx := context.WithValue(c, inventory.LoadShareUser{}, true)
	ctx = context.WithValue(ctx, inventory.LoadShareFile{}, true)
	share, err := shareClient.GetByHashID(ctx, s.ID)

	// Determine share status
	shareNotFound := err != nil
	shareExpired := !shareNotFound && inventory.IsValidShare(share) != nil

	// Get site settings
	siteBasic := settings.SiteBasic(c)
	pwa := settings.PWA(c)
	base := settings.SiteURL(c)

	// Build share URL
	shareURL := routes.MasterShareUrl(base, s.ID, "")

	// Build thumbnail URL - use PWA large icon (PNG), fallback to medium icon
	thumbnailURL := pwa.LargeIcon
	if thumbnailURL == "" {
		thumbnailURL = pwa.MediumIcon
	}
	if thumbnailURL != "" && !isAbsoluteURL(thumbnailURL) {
		thumbnailURL = base.ResolveReference(&url.URL{Path: thumbnailURL}).String()
	}

	// Prepare OG data
	ogData := &ShareOGData{
		SiteName:        siteBasic.Name,
		SiteDescription: siteBasic.Description,
		SiteURL:         base.String(),
		ShareURL:        shareURL.String(),
		ShareID:         s.ID,
		ThumbnailURL:    thumbnailURL,
		RedirectURL:     s.RedirectTo(c),
	}

	// Get file info (hide for invalid/expired/password-protected/login-required shares)
	switch {
	case shareNotFound:
		return renderStatusOG(ogData, siteBasic.Name, ogStatusInvalidLink)
	case needLogin:
		return renderStatusOG(ogData, siteBasic.Name, ogStatusNeedLogin)
	case shareExpired:
		return renderStatusOG(ogData, siteBasic.Name, ogStatusShareExpired)
	case share.Password != "" && !isShareUnlocked(share, s.Password, u):
		// Password required but not provided or incorrect.
		return renderStatusOG(ogData, siteBasic.Name, ogStatusPasswordRequired)
	case share.Edges.File == nil:
		return renderStatusOG(ogData, siteBasic.Name, ogStatusInvalidLink)
	}

	// Get owner name (don't expose email for privacy).
	if share.Edges.User != nil {
		ogData.OwnerName = share.Edges.User.Nick
	}

	shareFile := share.Edges.File
	if types.FileType(shareFile.Type) == types.FileTypeFolder {
		ogData.FolderName = defaultIfEmpty(shareFile.Name, ogDefaultFolderName)
		ogData.DisplayName = ogData.FolderName
		return RenderFolderOGHTML(ogData, ogFolderTitleTemplate, ogFolderDescTemplate)
	}

	ogData.FileName = defaultIfEmpty(shareFile.Name, ogDefaultFileName)
	ogData.FileSize = FormatFileSize(shareFile.Size)
	ogData.FileExt = strings.TrimPrefix(path.Ext(ogData.FileName), ".")
	ogData.DisplayName = ogData.FileName

	return RenderOGHTML(ogData, ogFileTitleTemplate, ogFileDescTemplate)
}

// isAbsoluteURL checks if the URL is absolute (starts with http:// or https://)
func isAbsoluteURL(u string) bool {
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}

func isShareUnlocked(share *ent.Share, password string, viewer *ent.User) bool {
	if share.Password == "" {
		return true
	}
	if password == share.Password {
		return true
	}
	if viewer != nil && share.Edges.User != nil && share.Edges.User.ID == viewer.ID {
		return true
	}
	return false
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

type (
	ShareInfoService struct {
		Password      string `form:"password"`
		CountViews    bool   `form:"count_views"`
		OwnerExtended bool   `form:"owner_extended"`
	}
	ShareInfoParamCtx struct{}
)

func (s *ShareInfoService) Get(c *gin.Context) (*explorer.Share, error) {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	shareClient := dep.ShareClient()

	ctx := context.WithValue(c, inventory.LoadShareUser{}, true)
	ctx = context.WithValue(ctx, inventory.LoadShareFile{}, true)
	share, err := shareClient.GetByID(ctx, hashid.FromContext(c))
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, serializer.NewError(serializer.CodeNotFound, "Share not found", nil)
		}
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get share", err)
	}

	if err := inventory.IsValidShare(share); err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "Share link expired", err)
	}

	if s.CountViews {
		_ = shareClient.Viewed(c, share)
	}

	unlocked := isShareUnlocked(share, s.Password, u)

	base := dep.SettingProvider().SiteURL(c)
	res := explorer.BuildShare(share, base, dep.HashIDEncoder(), u, share.Edges.User, share.Edges.File.Name,
		types.FileType(share.Edges.File.Type), unlocked, false)

	if s.OwnerExtended && share.Edges.User.ID == u.ID {
		// Add more information about the shared file
		m := manager.NewFileManager(dep, u)
		defer m.Recycle()

		shareUri, err := fs.NewUriFromString(fs.NewShareUri(res.ID, s.Password))
		if err != nil {
			return nil, serializer.NewError(serializer.CodeInternalSetting, "Invalid share url", err)
		}

		root, err := m.Get(c, shareUri)
		if err != nil {
			return nil, serializer.NewError(serializer.CodeNotFound, "File not found", err)
		}

		res.SourceUri = root.Uri(true).String()
	}

	return res, nil

}

type (
	ListShareService struct {
		PageSize       int    `form:"page_size" binding:"required,min=10,max=100"`
		OrderBy        string `uri:"order_by" form:"order_by" json:"order_by"`
		OrderDirection string `uri:"order_direction" form:"order_direction" json:"order_direction"`
		NextPageToken  string `form:"next_page_token"`
	}
	ListShareParamCtx struct{}
)

func (s *ListShareService) List(c *gin.Context) (*ListShareResponse, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	hasher := dep.HashIDEncoder()
	shareClient := dep.ShareClient()

	args := &inventory.ListShareArgs{
		PaginationArgs: &inventory.PaginationArgs{
			UseCursorPagination: true,
			PageToken:           s.NextPageToken,
			PageSize:            s.PageSize,
			Order:               inventory.OrderDirection(s.OrderDirection),
			OrderBy:             s.OrderBy,
		},
		UserID: user.ID,
	}

	ctx := context.WithValue(c, inventory.LoadShareUser{}, true)
	ctx = context.WithValue(ctx, inventory.LoadShareFile{}, true)
	ctx = context.WithValue(ctx, inventory.LoadFileMetadata{}, true)
	res, err := shareClient.List(ctx, args)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list shares", err)
	}

	base := dep.SettingProvider().SiteURL(ctx)
	return BuildListShareResponse(res, hasher, base, user, true), nil
}

func (s *ListShareService) ListInUserProfile(c *gin.Context, uid int) (*ListShareResponse, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	hasher := dep.HashIDEncoder()
	shareClient := dep.ShareClient()

	targetUser, err := dep.UserClient().GetActiveByID(c, uid)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get user", err)
	}

	if targetUser.Settings != nil && targetUser.Settings.ShareLinksInProfile == types.ProfileHideShare {
		return nil, serializer.NewError(serializer.CodeParamErr, "User has disabled share links in profile", nil)
	}

	publicOnly := targetUser.Settings == nil || targetUser.Settings.ShareLinksInProfile == types.ProfilePublicShareOnly
	args := &inventory.ListShareArgs{
		PaginationArgs: &inventory.PaginationArgs{
			UseCursorPagination: true,
			PageToken:           s.NextPageToken,
			PageSize:            s.PageSize,
			Order:               inventory.OrderDirection(s.OrderDirection),
			OrderBy:             s.OrderBy,
		},
		UserID:     uid,
		PublicOnly: publicOnly,
	}

	ctx := context.WithValue(c, inventory.LoadShareUser{}, true)
	ctx = context.WithValue(ctx, inventory.LoadShareFile{}, true)
	ctx = context.WithValue(ctx, inventory.LoadFileMetadata{}, true)
	res, err := shareClient.List(ctx, args)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list shares", err)
	}

	base := dep.SettingProvider().SiteURL(ctx)
	return BuildListShareResponse(res, hasher, base, user, false), nil
}
