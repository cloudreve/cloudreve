package share

import (
	"context"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/activity"
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

type (
	ShareInfoService struct {
		Password       string `form:"password"`
		CountViews     bool   `form:"count_views"`
		OwnerExtended  bool   `form:"owner_extended"`
		PurchaseTicket string `form:"purchase_ticket"`
	}
	ShareInfoParamCtx struct{}
)

func (s *ShareInfoService) Get(c *gin.Context) (*explorer.Share, error) {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	shareClient := dep.ShareClient()

	ctx := context.WithValue(c, inventory.LoadShareUser{}, true)
	ctx = context.WithValue(ctx, inventory.LoadShareFile{}, true)
	ctx = context.WithValue(ctx, inventory.LoadShareFiles{}, true)
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
		activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventShareLinkViewed,
			activity.Share(share.ID), activity.File(share.Edges.File.ID))
	}

	unlocked := true
	// Share requires password
	if share.Password != "" && s.Password != share.Password && share.Edges.User.ID != u.ID {
		unlocked = false
	}

	base := dep.SettingProvider().SiteURL(c)
	sourceType := types.FileType(share.Edges.File.Type)
	if len(share.Edges.Files) > 0 {
		// Multi-file shares render as a folder listing regardless of the
		// anchor file's type.
		sourceType = types.FileTypeFolder
	}
	res := explorer.BuildShare(c, share, base, dep.HashIDEncoder(), u, share.Edges.User, share.Edges.File.Name,
		sourceType, unlocked, false)
	res.FileCount = len(share.Edges.Files)

	// Priced shares resolve the requester's payment state: owner, existing
	// buyer, or bearer of a valid resume ticket.
	if share.PricePoints > 0 {
		paid := share.Edges.User.ID == u.ID ||
			(u.Edges.Group != nil && u.Edges.Group.Permissions.Enabled(int(types.GroupPermissionShareFree)))
		purchaseTicket := ""
		if !paid {
			vasClient := dep.VasClient()
			if s.PurchaseTicket != "" {
				if p, err := vasClient.SharePurchaseByTicket(ctx, share.ID, s.PurchaseTicket); err == nil {
					paid = true
					purchaseTicket = p.Ticket
				}
			}
			if !paid && !inventory.IsAnonymousUser(u) {
				if p, err := vasClient.SharePurchase(ctx, share.ID, u.ID); err == nil {
					paid = true
					purchaseTicket = p.Ticket
				}
			}
		}
		res.Paid = &paid
		res.PurchaseTicket = purchaseTicket
	}

	if s.OwnerExtended && share.Edges.User.ID == u.ID {
		// Add more information about the shared file
		m := manager.NewFileManager(dep, u)
		defer m.Recycle()

		shareUri, err := fs.NewUriFromString(fs.NewShareUri(res.ID, s.Password))
		if err != nil {
			return nil, serializer.NewError(serializer.CodeInternalSetting, "Invalid share url", err)
		}

		if len(share.Edges.Files) > 0 {
			// For multi-file shares point source_uri at the anchor's real
			// path so the owner edit dialog resolves a concrete file.
			anchor, err := m.Get(c, shareUri.Join(share.Edges.File.Name))
			if err != nil {
				return nil, serializer.NewError(serializer.CodeNotFound, "File not found", err)
			}
			res.SourceUri = anchor.Uri(true).String()
		} else {
			root, err := m.Get(c, shareUri)
			if err != nil {
				return nil, serializer.NewError(serializer.CodeNotFound, "File not found", err)
			}

			res.SourceUri = root.Uri(true).String()
		}
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
	ctx = context.WithValue(ctx, inventory.LoadShareFiles{}, true)
	ctx = context.WithValue(ctx, inventory.LoadFileMetadata{}, true)
	res, err := shareClient.List(ctx, args)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list shares", err)
	}

	base := dep.SettingProvider().SiteURL(ctx)
	return BuildListShareResponse(ctx, res, hasher, base, user, true), nil
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

	// An unset user preference ("") inherits the site-wide default; explicit
	// values (public_share/all_share/hide_share) always win.
	level := types.ProfilePublicShareOnly
	if targetUser.Settings != nil {
		level = targetUser.Settings.ShareLinksInProfile
	}
	if level == types.ProfilePublicShareOnly {
		level = dep.SettingProvider().ShareDefaults(c).LinksInProfile
	}
	if level == types.ProfileHideShare {
		return nil, serializer.NewError(serializer.CodeParamErr, "User has disabled share links in profile", nil)
	}

	publicOnly := level != types.ProfileAllShare
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
	ctx = context.WithValue(ctx, inventory.LoadShareFiles{}, true)
	ctx = context.WithValue(ctx, inventory.LoadFileMetadata{}, true)
	res, err := shareClient.List(ctx, args)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list shares", err)
	}

	base := dep.SettingProvider().SiteURL(ctx)
	return BuildListShareResponse(ctx, res, hasher, base, user, false), nil
}

type (
	// ListPublicShareService lists shares opted into the public directory.
	// Anonymous-accessible; only non-expired, non-password shares surface.
	ListPublicShareService struct {
		PageSize       int    `form:"page_size" binding:"required,min=10,max=100"`
		Query          string `form:"query" binding:"omitempty,max=100"`
		OrderBy        string `form:"order_by"`
		OrderDirection string `form:"order_direction"`
		NextPageToken  string `form:"next_page_token"`
	}
	ListPublicShareParamCtx struct{}
)

func (s *ListPublicShareService) ListPublic(c *gin.Context) (*ListShareResponse, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	shareClient := dep.ShareClient()

	args := &inventory.ListShareArgs{
		PaginationArgs: &inventory.PaginationArgs{
			UseCursorPagination: true,
			PageToken:           s.NextPageToken,
			PageSize:            s.PageSize,
			Order:               inventory.OrderDirection(s.OrderDirection),
			OrderBy:             s.OrderBy,
		},
		ListedOnly: true,
		Query:      strings.TrimSpace(s.Query),
	}

	ctx := context.WithValue(c, inventory.LoadShareUser{}, true)
	ctx = context.WithValue(ctx, inventory.LoadShareFile{}, true)
	ctx = context.WithValue(ctx, inventory.LoadShareFiles{}, true)
	res, err := shareClient.List(ctx, args)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list shares", err)
	}

	base := dep.SettingProvider().SiteURL(ctx)
	// unlocked=true: listed shares carry no password, and the anonymous
	// share/info endpoint already exposes the same fields to visitors.
	return BuildListShareResponse(ctx, res, dep.HashIDEncoder(), base, user, true), nil
}
