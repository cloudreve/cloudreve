package manager

import (
	"context"
	"encoding/gob"
	"fmt"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/lock"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/samber/lo"
)

const (
	EntityUrlCacheKeyPrefix     = "entity_url_"
	DownloadSentinelCachePrefix = "download_sentinel_"
)

type (
	ListArgs struct {
		Page           int
		PageSize       int
		PageToken      string
		Order          string
		OrderDirection string
		// StreamResponseCallback is used for streamed list operation, e.g. searching files.
		// Whenever a new item is found, this callback will be called with the current item and the parent item.
		StreamResponseCallback func(fs.File, []fs.File)
	}

	EntityUrlCache struct {
		Url                        string
		BrowserDownloadDisplayName string
		ExpireAt                   *time.Time
	}
)

func init() {
	gob.Register(EntityUrlCache{})
}

func (m *manager) Get(ctx context.Context, path *fs.URI, opts ...fs.Option) (fs.File, error) {
	return m.fs.Get(ctx, path, opts...)
}

func (m *manager) List(ctx context.Context, path *fs.URI, args *ListArgs) (fs.File, *fs.ListFileResult, error) {
	dbfsSetting := m.settings.DBFS(ctx)
	opts := []fs.Option{
		fs.WithPageSize(args.PageSize),
		fs.WithOrderBy(args.Order),
		fs.WithOrderDirection(args.OrderDirection),
		dbfs.WithFilePublicMetadata(),
		dbfs.WithContextHint(),
		dbfs.WithFileShareIfOwned(),
	}

	searchParams := path.SearchParameters()
	if searchParams != nil {
		if dbfsSetting.UseSSEForSearch {
			opts = append(opts, dbfs.WithStreamListResponseCallback(args.StreamResponseCallback))
		}

		if searchParams.Category != "" {
			// Overwrite search query with predefined category
			category := fs.SearchCategoryFromString(searchParams.Category)
			if category == setting.CategoryUnknown {
				return nil, nil, fmt.Errorf("unknown category: %s", searchParams.Category)
			}

			path = path.SetQuery(m.settings.SearchCategoryQuery(ctx, category))
			searchParams = path.SearchParameters()
		}
	}

	if dbfsSetting.UseCursorPagination || searchParams != nil {
		opts = append(opts, dbfs.WithCursorPagination(args.PageToken))
	} else {
		opts = append(opts, fs.WithPage(args.Page))
	}

	return m.fs.List(ctx, path, opts...)
}

func (m *manager) SharedAddressTranslation(ctx context.Context, path *fs.URI, opts ...fs.Option) (fs.File, *fs.URI, error) {
	o := newOption()
	for _, opt := range opts {
		opt.Apply(o)
	}

	return m.fs.SharedAddressTranslation(ctx, path)
}

func (m *manager) Create(ctx context.Context, path *fs.URI, fileType types.FileType, opts ...fs.Option) (fs.File, error) {
	o := newOption()
	for _, opt := range opts {
		opt.Apply(o)
	}

	if m.stateless {
		return nil, o.Node.CreateFile(ctx, &fs.StatelessCreateFileService{
			Path:   path.String(),
			Type:   fileType,
			UserID: o.StatelessUserID,
		})
	}

	isSymbolic := false
	if o.Metadata != nil {
		_, err := m.validateMetadata(ctx, lo.MapToSlice(o.Metadata, func(key string, value string) fs.MetadataPatch {
			if key == shareRedirectMetadataKey {
				isSymbolic = true
			}

			return fs.MetadataPatch{
				Key:   key,
				Value: value,
			}
		})...)
		if err != nil {
			return nil, err
		}
	}

	if isSymbolic {
		opts = append(opts, dbfs.WithSymbolicLink())
	}

	return m.fs.Create(ctx, path, fileType, opts...)
}

func (m *manager) Rename(ctx context.Context, path *fs.URI, newName string) (fs.File, error) {
	return m.fs.Rename(ctx, path, newName)
}

func (m *manager) MoveOrCopy(ctx context.Context, src []*fs.URI, dst *fs.URI, isCopy bool) error {
	return m.fs.MoveOrCopy(ctx, src, dst, isCopy)
}

func (m *manager) SoftDelete(ctx context.Context, path ...*fs.URI) error {
	return m.fs.SoftDelete(ctx, path...)
}

func (m *manager) Delete(ctx context.Context, path []*fs.URI, opts ...fs.Option) error {
	o := newOption()
	for _, opt := range opts {
		opt.Apply(o)
	}

	if !o.SkipSoftDelete && !o.SysSkipSoftDelete {
		return m.SoftDelete(ctx, path...)
	}

	staleEntities, err := m.fs.Delete(ctx, path, fs.WithUnlinkOnly(o.UnlinkOnly), fs.WithSysSkipSoftDelete(o.SysSkipSoftDelete))
	if err != nil {
		return err
	}

	m.l.Debug("New stale entities: %v", staleEntities)

	// Delete stale entities
	if len(staleEntities) > 0 {
		t, err := newExplicitEntityRecycleTask(ctx, lo.Map(staleEntities, func(entity fs.Entity, index int) int {
			return entity.ID()
		}))
		if err != nil {
			return fmt.Errorf("failed to create explicit entity recycle task: %w", err)
		}

		if err := m.dep.EntityRecycleQueue(ctx).QueueTask(ctx, t); err != nil {
			return fmt.Errorf("failed to queue explicit entity recycle task: %w", err)
		}
	}
	return nil
}

func (m *manager) Walk(ctx context.Context, path *fs.URI, depth int, f fs.WalkFunc, opts ...fs.Option) error {
	return m.fs.Walk(ctx, path, depth, f, opts...)
}

func (m *manager) Capacity(ctx context.Context) (*fs.Capacity, error) {
	res, err := m.fs.Capacity(ctx, m.user)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (m *manager) CheckIfCapacityExceeded(ctx context.Context) error {
	ctx = context.WithValue(ctx, inventory.LoadUserGroup{}, true)
	capacity, err := m.Capacity(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user capacity: %w", err)
	}

	if capacity.Used <= capacity.Total {
		return nil
	}

	return nil
}

func (l *manager) ConfirmLock(ctx context.Context, ancestor fs.File, uri *fs.URI, token ...string) (func(), fs.LockSession, error) {
	return l.fs.ConfirmLock(ctx, ancestor, uri, token...)
}

func (l *manager) Lock(ctx context.Context, d time.Duration, requester *ent.User, zeroDepth bool,
	application lock.Application, uri *fs.URI, token string) (fs.LockSession, error) {
	return l.fs.Lock(ctx, d, requester, zeroDepth, application, uri, token)
}

func (l *manager) Unlock(ctx context.Context, tokens ...string) error {
	return l.fs.Unlock(ctx, tokens...)
}

func (l *manager) Refresh(ctx context.Context, d time.Duration, token string) (lock.LockDetails, error) {
	return l.fs.Refresh(ctx, d, token)
}

func (l *manager) Restore(ctx context.Context, path ...*fs.URI) error {
	return l.fs.Restore(ctx, path...)
}

func (l *manager) CreateOrUpdateShare(ctx context.Context, path *fs.URI, args *CreateShareArgs) (*ent.Share, error) {
	file, err := l.fs.Get(ctx, path, dbfs.WithRequiredCapabilities(dbfs.NavigatorCapabilityShare), dbfs.WithNotRoot())
	if err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "src file not found", err)
	}

	// Only file owner can share file
	if file.OwnerID() != l.user.ID {
		return nil, serializer.NewError(serializer.CodeNoPermissionErr, "permission denied", nil)
	}

	if file.IsSymbolic() {
		return nil, serializer.NewError(serializer.CodeNoPermissionErr, "cannot share symbolic file", nil)
	}

	var existed *ent.Share
	shareClient := l.dep.ShareClient()
	if args.ExistedShareID != 0 {
		loadShareCtx := context.WithValue(ctx, inventory.LoadShareFile{}, true)
		existed, err = shareClient.GetByID(loadShareCtx, args.ExistedShareID)
		if err != nil {
			return nil, serializer.NewError(serializer.CodeNotFound, "failed to get existed share", err)
		}

		if existed.Edges.File.ID != file.ID() {
			return nil, serializer.NewError(serializer.CodeNotFound, "share link not found", nil)
		}
	}

	password := ""
	if args.IsPrivate {
		password = args.Password
		if strings.TrimSpace(password) == "" {
			password = util.RandString(8, util.RandomLowerCases)
		}
	}

	props := &types.ShareProps{
		ShareView:  args.ShareView,
		ShowReadMe: args.ShowReadMe,
	}

	share, err := shareClient.Upsert(ctx, &inventory.CreateShareParams{
		OwnerID:         file.OwnerID(),
		FileID:          file.ID(),
		Password:        password,
		Expires:         args.Expire,
		RemainDownloads: args.RemainDownloads,
		Existed:         existed,
		Props:           props,
	})

	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "failed to create share", err)
	}

	return share, nil
}

func (m *manager) TraverseFile(ctx context.Context, fileID int) (fs.File, error) {
	return m.fs.TraverseFile(ctx, fileID)
}

func (m *manager) PatchView(ctx context.Context, uri *fs.URI, view *types.ExplorerView) error {
	if uri.PathTrimmed() == "" && uri.FileSystem() != constants.FileSystemMy && uri.FileSystem() != constants.FileSystemShare {
		if m.user.Settings.FsViewMap == nil {
			m.user.Settings.FsViewMap = make(map[string]types.ExplorerView)
		}

		if view == nil {
			delete(m.user.Settings.FsViewMap, string(uri.FileSystem()))
		} else {
			m.user.Settings.FsViewMap[string(uri.FileSystem())] = *view
		}

		if err := m.dep.UserClient().SaveSettings(ctx, m.user); err != nil {
			return serializer.NewError(serializer.CodeDBError, "failed to save user settings", err)
		}

		return nil
	}

	patch := &types.FileProps{
		View: view,
	}
	isDelete := view == nil
	if isDelete {
		patch.View = &types.ExplorerView{}
	}
	if err := m.fs.PatchProps(ctx, uri, patch, isDelete); err != nil {
		return err
	}

	return nil
}

func getEntityDisplayName(f fs.File, e fs.Entity) string {
	switch e.Type() {
	case types.EntityTypeThumbnail:
		return fmt.Sprintf("%s_thumbnail", f.DisplayName())
	case types.EntityTypeLivePhoto:
		return fmt.Sprintf("%s_live_photo.mov", f.DisplayName())
	default:
		return f.Name()
	}
}

func expireTimeToTTL(expireAt *time.Time) int {
	if expireAt == nil {
		return -1
	}

	return int(time.Until(*expireAt).Seconds())
}
