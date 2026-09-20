package share

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/activity"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/service/explorer"
	"github.com/gin-gonic/gin"
)

type (
	// ShareCreateService 创建新分享服务
	ShareCreateService struct {
		Uri             string   `json:"uri" binding:"required_without=Uris"`
		Uris            []string `json:"uris" binding:"omitempty,min=1,max=50,dive,required"`
		IsPrivate       bool     `json:"is_private"`
		Password        string   `json:"password" binding:"omitempty,max=32"`
		RemainDownloads int      `json:"downloads"`
		Expire          int      `json:"expire"`
		ShareView       bool     `json:"share_view"`
		ShowReadMe      bool     `json:"show_readme"`
		HideReadMe      bool     `json:"hide_readme"`
		AllowUpload     bool     `json:"allow_upload"`
		AllowEdit       bool     `json:"allow_edit"`
		PreviewOnly     bool     `json:"preview_only"`
		UploadOnly      bool     `json:"upload_only"`
		// Optional owner-defined note shown on My Shares (#3570).
		Note string `json:"note" binding:"omitempty,max=255"`
		// Points price visitors must pay before downloading. 0 = free share.
		PricePoints int `json:"price_points" binding:"omitempty,min=0"`
		// ListedPublicly opts the share into the public share directory.
		// Requires the group's public-listing permission and is rejected on
		// password-protected shares.
		ListedPublicly bool `json:"listed_publicly"`
	}
	ShareCreateParamCtx struct{}

	BatchDeleteShareService struct {
		ShareIDs []string `json:"ids" binding:"required"`
	}
	BatchDeleteParamCtx struct{}
)

func (service *BatchDeleteShareService) Delete(c *gin.Context) error {
	dep := dependency.FromContext(c)
	uid := inventory.UserIDFromContext(c)
	shareClient := dep.ShareClient()

	var ids []int

	for _, v := range service.ShareIDs {
		id, err := dep.HashIDEncoder().Decode(v, hashid.ShareID)
		if err != nil {
			return fmt.Errorf("failed to decode hash id %q: %w", v, err)
		}

		ids = append(ids, id)
	}

	if err := shareClient.DeleteBatchByUserID(c, uid, ids); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to delete shares", err)
	}

	for _, id := range ids {
		activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventDeleteShare, activity.Share(id))
	}
	return nil
}

// Upsert 创建或更新分享
func (service *ShareCreateService) Upsert(c *gin.Context, existed int) (string, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	m := manager.NewFileManager(dep, user)
	defer m.Recycle()

	// Check group permission for creating share link
	if !user.Edges.Group.Permissions.Enabled(int(types.GroupPermissionShare)) {
		return "", serializer.NewError(serializer.CodeGroupNotAllowed, "Group permission denied", nil)
	}

	if service.PricePoints > 0 && !user.Edges.Group.Permissions.Enabled(int(types.GroupPermissionShareSell)) {
		return "", serializer.NewError(serializer.CodeGroupNotAllowed, "Group permission denied for paid share", nil)
	}

	if service.ListedPublicly {
		if !user.Edges.Group.Permissions.Enabled(int(types.GroupPermissionSharePublicList)) {
			return "", serializer.NewError(serializer.CodeGroupNotAllowed, "Group permission denied for public share listing", nil)
		}
		if service.IsPrivate {
			return "", serializer.NewError(serializer.CodeParamErr, "password-protected shares cannot be publicly listed", nil)
		}
	}

	rawUris := service.Uris
	if len(rawUris) == 0 && service.Uri != "" {
		rawUris = []string{service.Uri}
	}
	if len(rawUris) > 1 && service.UploadOnly {
		return "", serializer.NewError(serializer.CodeParamErr, "upload-only shares cannot cover multiple files", nil)
	}

	uris := make([]*fs.URI, 0, len(rawUris))
	seenUris := make(map[string]struct{}, len(rawUris))
	for _, raw := range rawUris {
		if _, ok := seenUris[raw]; ok {
			continue
		}
		seenUris[raw] = struct{}{}
		uri, err := fs.NewUriFromString(raw)
		if err != nil {
			return "", serializer.NewError(serializer.CodeParamErr, "unknown uri", err)
		}
		uris = append(uris, uri)
	}
	if len(uris) == 0 {
		return "", serializer.NewError(serializer.CodeParamErr, "unknown uri", nil)
	}

	var expires *time.Time
	if service.Expire > 0 {
		expires = new(time.Time)
		*expires = time.Now().Add(time.Duration(service.Expire) * time.Second)
	}

	share, err := m.CreateOrUpdateShare(c, uris, &manager.CreateShareArgs{
		IsPrivate:       service.IsPrivate,
		Password:        service.Password,
		RemainDownloads: service.RemainDownloads,
		Expire:          expires,
		ExistedShareID:  existed,
		ShareView:       service.ShareView,
		ShowReadMe:      service.ShowReadMe,
		HideReadMe:      service.HideReadMe,
		AllowUpload:     service.AllowUpload,
		AllowEdit:       service.AllowEdit,
		PreviewOnly:     service.PreviewOnly,
		UploadOnly:      service.UploadOnly,
		Note:            service.Note,
		PricePoints:     service.PricePoints,
		ListedPublicly:  service.ListedPublicly,
	})
	if err != nil {
		return "", err
	}

	eventType := types.EventShare
	if existed > 0 {
		eventType = types.EventEditShare
	}
	opts := []activity.Opt{activity.Share(share.ID)}
	if share.Edges.File != nil {
		opts = append(opts, activity.File(share.Edges.File.ID))
	}
	activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), eventType, opts...)

	base := dep.SettingProvider().SiteURL(c)
	return explorer.BuildShareLink(share, dep.HashIDEncoder(), base, true), nil
}

func DeleteShare(c *gin.Context, shareId int) error {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	shareClient := dep.ShareClient()

	ctx := context.WithValue(c, inventory.LoadShareFile{}, true)
	var (
		share *ent.Share
		err   error
	)
	if user.Edges.Group.Permissions.Enabled(int(types.GroupPermissionIsAdmin)) {
		share, err = shareClient.GetByID(ctx, shareId)
	} else {
		share, err = shareClient.GetByIDUser(ctx, shareId, user.ID)
	}
	if err != nil {
		return serializer.NewError(serializer.CodeNotFound, "share not found", err)
	}

	if err := shareClient.Delete(c, share.ID); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to delete share", err)
	}

	activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventDeleteShare, activity.Share(share.ID))
	return nil
}
