package share

import (
	"context"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	sharepkg "github.com/cloudreve/Cloudreve/v4/pkg/share"
	"github.com/cloudreve/Cloudreve/v4/pkg/sharepreview"
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

func (s *ShortLinkRedirectService) RedirectTo(c *gin.Context) string {
	return sharepkg.BuildRedirectURL(s.ID, s.Password, c.Query("path"), c.Request.URL.Query())
}

// RenderOGPage renders an Open Graph HTML page for social media previews
func (s *ShortLinkRedirectService) RenderOGPage(c *gin.Context) (string, error) {
	return sharepreview.RenderOGPage(c, sharepreview.Options{
		ID:        s.ID,
		Password:  s.Password,
		SharePath: c.Query("path"),
	})
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

	shareID := hashid.FromContext(c)
	share, unlocked, status, err := sharepkg.LoadShareForInfo(c, shareClient, shareID, u, s.Password)
	switch status {
	case sharepkg.LoadNotFound:
		return nil, serializer.NewError(serializer.CodeNotFound, "Share not found", nil)
	case sharepkg.LoadExpired:
		return nil, serializer.NewError(serializer.CodeNotFound, "Share link expired", err)
	case sharepkg.LoadError:
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get share", err)
	}

	if s.CountViews {
		_ = shareClient.Viewed(c, share)
	}

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
