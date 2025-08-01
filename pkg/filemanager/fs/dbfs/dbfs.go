package dbfs

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/lock"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gofrs/uuid"
	"github.com/samber/lo"
	"golang.org/x/tools/container/intsets"
)

const (
	ContextHintHeader         = constants.CrHeaderPrefix + "Context-Hint"
	NavigatorStateCachePrefix = "navigator_state_"
	ContextHintTTL            = 5 * 60 // 5 minutes

	folderSummaryCachePrefix = "folder_summary_"
	defaultPageSize          = 100
)

type (
	ContextHintCtxKey      struct{}
	ByPassOwnerCheckCtxKey struct{}
)

func NewDatabaseFS(u *ent.User, fileClient inventory.FileClient, shareClient inventory.ShareClient,
	l logging.Logger, ls lock.LockSystem, settingClient setting.Provider,
	storagePolicyClient inventory.StoragePolicyClient, hasher hashid.Encoder, userClient inventory.UserClient,
	cache, stateKv cache.Driver, directLinkClient inventory.DirectLinkClient) fs.FileSystem {
	return &DBFS{
		user:                u,
		navigators:          make(map[string]Navigator),
		fileClient:          fileClient,
		shareClient:         shareClient,
		l:                   l,
		ls:                  ls,
		settingClient:       settingClient,
		storagePolicyClient: storagePolicyClient,
		hasher:              hasher,
		userClient:          userClient,
		cache:               cache,
		stateKv:             stateKv,
		directLinkClient:    directLinkClient,
	}
}

type DBFS struct {
	user                *ent.User
	navigators          map[string]Navigator
	fileClient          inventory.FileClient
	userClient          inventory.UserClient
	storagePolicyClient inventory.StoragePolicyClient
	shareClient         inventory.ShareClient
	directLinkClient    inventory.DirectLinkClient
	l                   logging.Logger
	ls                  lock.LockSystem
	settingClient       setting.Provider
	hasher              hashid.Encoder
	cache               cache.Driver
	stateKv             cache.Driver
	mu                  sync.Mutex
}

func (f *DBFS) Recycle() {
	for _, navigator := range f.navigators {
		navigator.Recycle()
	}
}

func (f *DBFS) GetEntity(ctx context.Context, entityID int) (fs.Entity, error) {
	if entityID == 0 {
		return fs.NewEmptyEntity(f.user), nil
	}

	files, _, err := f.fileClient.GetEntitiesByIDs(ctx, []int{entityID}, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	if len(files) == 0 {
		return nil, fs.ErrEntityNotExist
	}

	return fs.NewEntity(files[0]), nil

}

func (f *DBFS) List(ctx context.Context, path *fs.URI, opts ...fs.Option) (fs.File, *fs.ListFileResult, error) {
	o := newDbfsOption()
	for _, opt := range opts {
		o.apply(opt)
	}

	// Get navigator
	navigator, err := f.getNavigator(ctx, path, NavigatorCapabilityListChildren)
	if err != nil {
		return nil, nil, err
	}

	searchParams := path.SearchParameters()
	isSearching := searchParams != nil

	parent, err := f.getFileByPath(ctx, navigator, path)
	if err != nil {
		return nil, nil, fmt.Errorf("Parent not exist: %w", err)
	}

	pageSize := 0
	orderDirection := ""
	orderBy := ""

	view := navigator.GetView(ctx, parent)
	if view != nil {
		pageSize = view.PageSize
		orderDirection = view.OrderDirection
		orderBy = view.Order
	}

	if o.PageSize > 0 {
		pageSize = o.PageSize
	}
	if o.OrderDirection != "" {
		orderDirection = o.OrderDirection
	}
	if o.OrderBy != "" {
		orderBy = o.OrderBy
	}

	// Validate pagination args
	props := navigator.Capabilities(isSearching)
	if pageSize > props.MaxPageSize {
		pageSize = props.MaxPageSize
	} else if pageSize == 0 {
		pageSize = defaultPageSize
	}

	if view != nil {
		view.PageSize = pageSize
		view.OrderDirection = orderDirection
		view.Order = orderBy
	}

	var hintId *uuid.UUID
	if o.generateContextHint {
		newHintId := uuid.Must(uuid.NewV4())
		hintId = &newHintId
	}

	if o.loadFilePublicMetadata {
		ctx = context.WithValue(ctx, inventory.LoadFilePublicMetadata{}, true)
	}
	if o.loadFileShareIfOwned && parent != nil && parent.OwnerID() == f.user.ID {
		ctx = context.WithValue(ctx, inventory.LoadFileShare{}, true)
	}

	var streamCallback func([]*File)
	if o.streamListResponseCallback != nil {
		streamCallback = func(files []*File) {
			o.streamListResponseCallback(parent, lo.Map(files, func(item *File, index int) fs.File {
				return item
			}))
		}
	}

	children, err := navigator.Children(ctx, parent, &ListArgs{
		Page: &inventory.PaginationArgs{
			Page:                o.FsOption.Page,
			PageSize:            pageSize,
			OrderBy:             orderBy,
			Order:               inventory.OrderDirection(orderDirection),
			UseCursorPagination: o.useCursorPagination,
			PageToken:           o.pageToken,
		},
		Search:         searchParams,
		StreamCallback: streamCallback,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get children: %w", err)
	}

	var storagePolicy *ent.StoragePolicy
	if parent != nil {
		storagePolicy, err = f.getPreferredPolicy(ctx, parent)
		if err != nil {
			f.l.Warning("Failed to get preferred policy: %v", err)
		}
	}

	return parent, &fs.ListFileResult{
		Files: lo.Map(children.Files, func(item *File, index int) fs.File {
			return item
		}),
		Props:                 props,
		Pagination:            children.Pagination,
		ContextHint:           hintId,
		RecursionLimitReached: children.RecursionLimitReached,
		MixedType:             children.MixedType,
		SingleFileView:        children.SingleFileView,
		Parent:                parent,
		StoragePolicy:         storagePolicy,
		View:                  view,
	}, nil
}

func (f *DBFS) Capacity(ctx context.Context, u *ent.User) (*fs.Capacity, error) {
	// First, get user's available storage packs
	var (
		res = &fs.Capacity{}
	)

	requesterGroup, err := u.Edges.GroupOrErr()
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get user's group", err)
	}

	res.Used = f.user.Storage
	res.Total = requesterGroup.MaxStorage
	return res, nil
}

func (f *DBFS) CreateEntity(ctx context.Context, file fs.File, policy *ent.StoragePolicy,
	entityType types.EntityType, req *fs.UploadRequest, opts ...fs.Option) (fs.Entity, error) {
	o := newDbfsOption()
	for _, opt := range opts {
		o.apply(opt)
	}

	// If uploader specified previous latest version ID (etag), we should check if it's still valid.
	if o.previousVersion != "" {
		entityId, err := f.hasher.Decode(o.previousVersion, hashid.EntityID)
		if err != nil {
			return nil, serializer.NewError(serializer.CodeParamErr, "Unknown version ID", err)
		}

		entities, err := file.(*File).Model.Edges.EntitiesOrErr()
		if err != nil || entities == nil {
			return nil, fmt.Errorf("create entity: previous entities not load")
		}

		// File is stale during edit if the latest entity is not the same as the one specified by uploader.
		if e := file.PrimaryEntity(); e == nil || e.ID() != entityId {
			return nil, fs.ErrStaleVersion
		}
	}

	fc, tx, ctx, err := inventory.WithTx(ctx, f.fileClient)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to start transaction", err)
	}

	fileModel := file.(*File).Model
	if o.removeStaleEntities {
		storageDiff, err := fc.RemoveStaleEntities(ctx, fileModel)
		if err != nil {
			_ = inventory.Rollback(tx)
			return nil, serializer.NewError(serializer.CodeDBError, "Failed to remove stale entities", err)
		}

		tx.AppendStorageDiff(storageDiff)
	}

	entity, storageDiff, err := fc.CreateEntity(ctx, fileModel, &inventory.EntityParameters{
		OwnerID:         file.(*File).Owner().ID,
		EntityType:      entityType,
		StoragePolicyID: policy.ID,
		Source:          req.Props.SavePath,
		Size:            req.Props.Size,
		UploadSessionID: uuid.FromStringOrNil(o.UploadRequest.Props.UploadSessionID),
	})
	if err != nil {
		_ = inventory.Rollback(tx)

		return nil, serializer.NewError(serializer.CodeDBError, "Failed to create entity", err)
	}
	tx.AppendStorageDiff(storageDiff)

	if err := inventory.CommitWithStorageDiff(ctx, tx, f.l, f.userClient); err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to commit create change", err)
	}

	return fs.NewEntity(entity), nil
}

func (f *DBFS) SharedAddressTranslation(ctx context.Context, path *fs.URI, opts ...fs.Option) (fs.File, *fs.URI, error) {
	o := newDbfsOption()
	for _, opt := range opts {
		o.apply(opt)
	}

	// Get navigator
	navigator, err := f.getNavigator(ctx, path, o.requiredCapabilities...)
	if err != nil {
		return nil, nil, err
	}

	ctx = context.WithValue(ctx, inventory.LoadFilePublicMetadata{}, true)
	if o.loadFileEntities {
		ctx = context.WithValue(ctx, inventory.LoadFileEntity{}, true)
	}

	uriTranslation := func(target *File, rebase bool) (fs.File, *fs.URI, error) {
		// Translate shared address to real address
		metadata := target.Metadata()
		if metadata == nil {
			if err := f.fileClient.QueryMetadata(ctx, target.Model); err != nil {
				return nil, nil, fmt.Errorf("failed to query metadata: %w", err)
			}
			metadata = target.Metadata()
		}
		redirect, ok := metadata[MetadataSharedRedirect]
		if !ok {
			return nil, nil, fmt.Errorf("missing metadata %s in symbolic folder %s", MetadataSharedRedirect, path)
		}

		redirectUri, err := fs.NewUriFromString(redirect)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid redirect uri %s in symbolic folder %s", redirect, path)
		}
		newUri := redirectUri
		if rebase {
			newUri = redirectUri.Rebase(path, target.Uri(false))
		}
		return f.SharedAddressTranslation(ctx, newUri, opts...)
	}

	target, err := f.getFileByPath(ctx, navigator, path)
	if err != nil {
		if errors.Is(err, ErrSymbolicFolderFound) && target.Type() == types.FileTypeFolder {
			return uriTranslation(target, true)
		}

		if !ent.IsNotFound(err) {
			return nil, nil, fmt.Errorf("failed to get target file: %w", err)
		}

		// Request URI does not exist, return most recent ancestor
		return target, path, err
	}

	if target.IsSymbolic() {
		return uriTranslation(target, false)
	}

	return target, path, nil
}

func (f *DBFS) Get(ctx context.Context, path *fs.URI, opts ...fs.Option) (fs.File, error) {
	o := newDbfsOption()
	for _, opt := range opts {
		o.apply(opt)
	}

	// Get navigator
	navigator, err := f.getNavigator(ctx, path, o.requiredCapabilities...)
	if err != nil {
		return nil, err
	}

	if o.loadFilePublicMetadata || o.extendedInfo {
		ctx = context.WithValue(ctx, inventory.LoadFilePublicMetadata{}, true)
	}

	if o.loadFileEntities || o.extendedInfo || o.loadFolderSummary {
		ctx = context.WithValue(ctx, inventory.LoadFileEntity{}, true)
	}

	if o.extendedInfo {
		ctx = context.WithValue(ctx, inventory.LoadFileDirectLink{}, true)
	}

	if o.loadFileShareIfOwned {
		ctx = context.WithValue(ctx, inventory.LoadFileShare{}, true)
	}

	if o.loadEntityUser {
		ctx = context.WithValue(ctx, inventory.LoadEntityUser{}, true)
	}

	// Get target file
	target, err := f.getFileByPath(ctx, navigator, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get target file: %w", err)
	}

	if o.notRoot && (target == nil || target.IsRootFolder()) {
		return nil, fs.ErrNotSupportedAction.WithError(fmt.Errorf("cannot operate root file"))
	}

	if o.extendedInfo && target != nil {
		extendedInfo := &fs.FileExtendedInfo{
			StorageUsed:           target.SizeUsed(),
			EntityStoragePolicies: make(map[int]*ent.StoragePolicy),
		}

		if f.user.ID == target.OwnerID() {
			extendedInfo.DirectLinks = target.Model.Edges.DirectLinks
		}

		policyID := target.PolicyID()
		if policyID > 0 {
			policy, err := f.storagePolicyClient.GetPolicyByID(ctx, policyID)
			if err == nil {
				extendedInfo.StoragePolicy = policy
			}
		}

		target.FileExtendedInfo = extendedInfo
		if target.OwnerID() == f.user.ID || f.user.Edges.Group.Permissions.Enabled(int(types.GroupPermissionIsAdmin)) {
			target.FileExtendedInfo.Shares = target.Model.Edges.Shares
			if target.Model.Props != nil {
				target.FileExtendedInfo.View = target.Model.Props.View
			}
		}

		entities := target.Entities()
		for _, entity := range entities {
			if _, ok := extendedInfo.EntityStoragePolicies[entity.PolicyID()]; !ok {
				policy, err := f.storagePolicyClient.GetPolicyByID(ctx, entity.PolicyID())
				if err != nil {
					return nil, fmt.Errorf("failed to get policy: %w", err)
				}

				extendedInfo.EntityStoragePolicies[entity.PolicyID()] = policy
			}
		}
	}

	// Calculate folder summary if requested
	if o.loadFolderSummary && target != nil && target.Type() == types.FileTypeFolder {
		if _, ok := ctx.Value(ByPassOwnerCheckCtxKey{}).(bool); !ok && target.OwnerID() != f.user.ID {
			return nil, fs.ErrOwnerOnly
		}

		// first, try to load from cache
		summary, ok := f.cache.Get(fmt.Sprintf("%s%d", folderSummaryCachePrefix, target.ID()))
		if ok {
			summaryTyped := summary.(fs.FolderSummary)
			target.FileFolderSummary = &summaryTyped
		} else {
			// cache miss, walk the folder to get the summary
			newSummary := &fs.FolderSummary{Completed: true}
			if f.user.Edges.Group == nil {
				return nil, fmt.Errorf("user group not loaded")
			}
			limit := max(f.user.Edges.Group.Settings.MaxWalkedFiles, 1)

			// disable load metadata to speed up
			ctxWalk := context.WithValue(ctx, inventory.LoadFilePublicMetadata{}, false)
			if err := navigator.Walk(ctxWalk, []*File{target}, limit, intsets.MaxInt, func(files []*File, l int) error {
				for _, file := range files {
					if file.ID() == target.ID() {
						continue
					}
					if file.Type() == types.FileTypeFile {
						newSummary.Files++
					} else {
						newSummary.Folders++
					}

					newSummary.Size += file.SizeUsed()
				}
				return nil
			}); err != nil {
				if !errors.Is(err, ErrFileCountLimitedReached) {
					return nil, fmt.Errorf("failed to walk: %w", err)
				}

				newSummary.Completed = false
			}

			// cache the summary
			newSummary.CalculatedAt = time.Now()
			f.cache.Set(fmt.Sprintf("%s%d", folderSummaryCachePrefix, target.ID()), newSummary, f.settingClient.FolderPropsCacheTTL(ctx))
			target.FileFolderSummary = newSummary
		}
	}

	if target == nil {
		return nil, fmt.Errorf("cannot get root file with nil root")
	}

	return target, nil
}

func (f *DBFS) CheckCapability(ctx context.Context, uri *fs.URI, opts ...fs.Option) error {
	o := newDbfsOption()
	for _, opt := range opts {
		o.apply(opt)
	}

	// Get navigator
	_, err := f.getNavigator(ctx, uri, o.requiredCapabilities...)
	if err != nil {
		return err
	}

	return nil
}

func (f *DBFS) Walk(ctx context.Context, path *fs.URI, depth int, walk fs.WalkFunc, opts ...fs.Option) error {
	o := newDbfsOption()
	for _, opt := range opts {
		o.apply(opt)
	}

	if o.loadFilePublicMetadata {
		ctx = context.WithValue(ctx, inventory.LoadFilePublicMetadata{}, true)
	}

	if o.loadFileEntities {
		ctx = context.WithValue(ctx, inventory.LoadFileEntity{}, true)
	}

	// Get navigator
	navigator, err := f.getNavigator(ctx, path, o.requiredCapabilities...)
	if err != nil {
		return err
	}

	target, err := f.getFileByPath(ctx, navigator, path)
	if err != nil {
		return err
	}

	// Require Read permission
	if _, ok := ctx.Value(ByPassOwnerCheckCtxKey{}).(bool); !ok && target.OwnerID() != f.user.ID {
		return fs.ErrOwnerOnly
	}

	// Walk
	if f.user.Edges.Group == nil {
		return fmt.Errorf("user group not loaded")
	}
	limit := max(f.user.Edges.Group.Settings.MaxWalkedFiles, 1)

	if err := navigator.Walk(ctx, []*File{target}, limit, depth, func(files []*File, l int) error {
		for _, file := range files {
			if err := walk(file, l); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to walk: %w", err)
	}

	return nil
}

func (f *DBFS) ExecuteNavigatorHooks(ctx context.Context, hookType fs.HookType, file fs.File) error {
	navigator, err := f.getNavigator(ctx, file.Uri(false))
	if err != nil {
		return err
	}

	if dbfsFile, ok := file.(*File); ok {
		return navigator.ExecuteHook(ctx, hookType, dbfsFile)
	}

	return nil
}

// createFile creates a file with given name and type under given parent folder
func (f *DBFS) createFile(ctx context.Context, parent *File, name string, fileType types.FileType, o *dbfsOption) (*File, error) {
	createFileArgs := &inventory.CreateFileParameters{
		FileType:            fileType,
		Name:                name,
		MetadataPrivateMask: make(map[string]bool),
		Metadata:            make(map[string]string),
		IsSymbolic:          o.isSymbolicLink,
	}

	if o.Metadata != nil {
		for k, v := range o.Metadata {
			createFileArgs.Metadata[k] = v
		}
	}

	if o.preferredStoragePolicy != nil {
		createFileArgs.StoragePolicyID = o.preferredStoragePolicy.ID
	} else {
		// get preferred storage policy
		policy, err := f.getPreferredPolicy(ctx, parent)
		if err != nil {
			return nil, err
		}

		createFileArgs.StoragePolicyID = policy.ID
	}

	if o.UploadRequest != nil {
		createFileArgs.EntityParameters = &inventory.EntityParameters{
			EntityType:      types.EntityTypeVersion,
			Source:          o.UploadRequest.Props.SavePath,
			Size:            o.UploadRequest.Props.Size,
			ModifiedAt:      o.UploadRequest.Props.LastModified,
			UploadSessionID: uuid.FromStringOrNil(o.UploadRequest.Props.UploadSessionID),
			Importing:       o.UploadRequest.ImportFrom != nil,
		}
	}

	// Start transaction to create files
	fc, tx, ctx, err := inventory.WithTx(ctx, f.fileClient)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to start transaction", err)
	}

	file, entity, storageDiff, err := fc.CreateFile(ctx, parent.Model, createFileArgs)
	if err != nil {
		_ = inventory.Rollback(tx)
		if ent.IsConstraintError(err) {
			return nil, fs.ErrFileExisted.WithError(err)
		}

		return nil, serializer.NewError(serializer.CodeDBError, "Failed to create file", err)
	}

	tx.AppendStorageDiff(storageDiff)
	if err := inventory.CommitWithStorageDiff(ctx, tx, f.l, f.userClient); err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to commit create change", err)
	}

	file.SetEntities([]*ent.Entity{entity})
	return newFile(parent, file), nil
}

// getPreferredPolicy tries to get the preferred storage policy for the given file.
func (f *DBFS) getPreferredPolicy(ctx context.Context, file *File) (*ent.StoragePolicy, error) {
	ownerGroup := file.Owner().Edges.Group
	if ownerGroup == nil {
		return nil, fmt.Errorf("owner group not loaded")
	}

	groupPolicy, err := f.storagePolicyClient.GetByGroup(ctx, ownerGroup)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get available storage policies", err)
	}

	return groupPolicy, nil
}

func (f *DBFS) getFileByPath(ctx context.Context, navigator Navigator, path *fs.URI) (*File, error) {
	file, err := navigator.To(ctx, path)
	if err != nil && errors.Is(err, ErrFsNotInitialized) {
		// Initialize file system for user if root folder does not exist.
		uid := path.ID(hashid.EncodeUserID(f.hasher, f.user.ID))
		uidInt, err := f.hasher.Decode(uid, hashid.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to decode user ID: %w", err)
		}

		if err := f.initFs(ctx, uidInt); err != nil {
			return nil, fmt.Errorf("failed to initialize file system: %w", err)
		}
		return navigator.To(ctx, path)
	}

	return file, err
}

// initFs initializes the file system for the user.
func (f *DBFS) initFs(ctx context.Context, uid int) error {
	f.l.Info("Initialize database file system for user %q", f.user.Email)
	_, err := f.fileClient.CreateFolder(ctx, nil,
		&inventory.CreateFolderParameters{
			Owner: uid,
			Name:  inventory.RootFolderName,
		})
	if err != nil {
		return fmt.Errorf("failed to create root folder: %w", err)
	}

	return nil
}

func (f *DBFS) getNavigator(ctx context.Context, path *fs.URI, requiredCapabilities ...NavigatorCapability) (Navigator, error) {
	pathFs := path.FileSystem()
	config := f.settingClient.DBFS(ctx)
	navigatorId := f.navigatorId(path)
	var (
		res Navigator
	)
	f.mu.Lock()
	defer f.mu.Unlock()
	if navigator, ok := f.navigators[navigatorId]; ok {
		res = navigator
	} else {
		var n Navigator
		switch pathFs {
		case constants.FileSystemMy:
			n = NewMyNavigator(f.user, f.fileClient, f.userClient, f.l, config, f.hasher)
		case constants.FileSystemShare:
			n = NewShareNavigator(f.user, f.fileClient, f.shareClient, f.l, config, f.hasher)
		case constants.FileSystemTrash:
			n = NewTrashNavigator(f.user, f.fileClient, f.l, config, f.hasher)
		case constants.FileSystemSharedWithMe:
			n = NewSharedWithMeNavigator(f.user, f.fileClient, f.l, config, f.hasher)
		default:
			return nil, fmt.Errorf("unknown file system %q", pathFs)
		}

		// retrieve state if context hint is provided
		if stateID, ok := ctx.Value(ContextHintCtxKey{}).(uuid.UUID); ok && stateID != uuid.Nil {
			cacheKey := NavigatorStateCachePrefix + stateID.String() + "_" + navigatorId
			if stateRaw, ok := f.stateKv.Get(cacheKey); ok {
				if err := n.RestoreState(stateRaw.(State)); err != nil {
					f.l.Warning("Failed to restore state for navigator %q: %s", navigatorId, err)
				} else {
					f.l.Info("Navigator %q restored state (%q) successfully", navigatorId, stateID)
				}
			} else {
				// State expire, refresh it
				n.PersistState(f.stateKv, cacheKey)
			}
		}

		f.navigators[navigatorId] = n
		res = n
	}

	// Check fs capabilities
	capabilities := res.Capabilities(false).Capability
	for _, capability := range requiredCapabilities {
		if !capabilities.Enabled(int(capability)) {
			return nil, fs.ErrNotSupportedAction.WithError(fmt.Errorf("action %q is not supported under current fs", capability))
		}
	}

	return res, nil
}

func (f *DBFS) navigatorId(path *fs.URI) string {
	uidHashed := hashid.EncodeUserID(f.hasher, f.user.ID)
	switch path.FileSystem() {
	case constants.FileSystemMy:
		return fmt.Sprintf("%s/%s/%d", constants.FileSystemMy, path.ID(uidHashed), f.user.ID)
	case constants.FileSystemShare:
		return fmt.Sprintf("%s/%s/%d", constants.FileSystemShare, path.ID(uidHashed), f.user.ID)
	case constants.FileSystemTrash:
		return fmt.Sprintf("%s/%s", constants.FileSystemTrash, path.ID(uidHashed))
	default:
		return fmt.Sprintf("%s/%s/%d", path.FileSystem(), path.ID(uidHashed), f.user.ID)
	}
}

// generateSavePath generates the physical save path for the upload request.
func generateSavePath(policy *ent.StoragePolicy, req *fs.UploadRequest, user *ent.User) string {
	fixedBaseTable := map[string]string{
		"{timestamp}":      strconv.FormatInt(time.Now().Unix(), 10),
		"{timestamp_nano}": strconv.FormatInt(time.Now().UnixNano(), 10),
		"{uid}":            strconv.Itoa(user.ID),
		"{datetime}":       time.Now().Format("20060102150405"),
		"{date}":           time.Now().Format("20060102"),
		"{year}":           time.Now().Format("2006"),
		"{month}":          time.Now().Format("01"),
		"{day}":            time.Now().Format("02"),
		"{hour}":           time.Now().Format("15"),
		"{minute}":         time.Now().Format("04"),
		"{second}":         time.Now().Format("05"),
	}

    dynamicReplace := func(regPattern string, rule string) string {
        re := regexp.MustCompile(regPattern)
        return re.ReplaceAllStringFunc(rule, func(match string) string {
            switch match {
            case "{randomkey16}":
                return util.RandStringRunes(16)
            case "{randomkey8}":
                return util.RandStringRunes(8)
			case "{randomnum8}":
				return strconv.Itoa(rand.Intn(8))
			case "{randomnum4}":
				return strconv.Itoa(rand.Intn(4))
			case "{randomnum3}":
				return strconv.Itoa(rand.Intn(3))
			case "{randomnum2}":
				return strconv.Itoa(rand.Intn(2))
            case "{uuid}":
                return uuid.Must(uuid.NewV4()).String()
            default:
                return match
            }
        })
    }

	dirRule := policy.DirNameRule
	dirRule = filepath.ToSlash(dirRule)
	dirRule = util.Replace(fixedBaseTable, dirRule)
	dirRule = util.Replace(map[string]string{
		"{path}": req.Props.Uri.Dir() + fs.Separator,
	}, dirRule)
	dirRule = dynamicReplace(`\{randomkey16\}|\{randomkey8\}|\{randomnum8\}|\{randomnum4\}|\{randomnum3\}|\{randomnum2\}`, dirRule)

	originName := req.Props.Uri.Name()
	fixedNameTable := map[string]string{
		"{originname}":             originName,
		"{ext}":                    filepath.Ext(originName),
		"{originname_without_ext}": strings.TrimSuffix(originName, filepath.Ext(originName)),
	}

	nameRule := policy.FileNameRule
	nameRule = filepath.ToSlash(nameRule)
	nameRule = util.Replace(fixedBaseTable, nameRule)
	nameRule = util.Replace(fixedNameTable, nameRule)
	nameRule = dynamicReplace(`\{randomkey16\}|\{randomkey8\}|\{randomnum8\}|\{randomnum4\}|\{randomnum3\}|\{randomnum2\}|\{uuid\}`, nameRule)

	return path.Join(path.Clean(dirRule), nameRule)
}

func canMoveOrCopyTo(src, dst *fs.URI, isCopy bool) bool {
	if isCopy {
		return src.FileSystem() == dst.FileSystem() && src.FileSystem() == constants.FileSystemMy
	} else {
		switch src.FileSystem() {
		case constants.FileSystemMy:
			return dst.FileSystem() == constants.FileSystemMy || dst.FileSystem() == constants.FileSystemTrash
		case constants.FileSystemTrash:
			return dst.FileSystem() == constants.FileSystemMy

		}
	}

	return false
}

func allAncestors(targets []*File) []*ent.File {
	return lo.Map(
		lo.UniqBy(
			lo.FlatMap(targets, func(value *File, index int) []*File {
				return value.Ancestors()
			}),
			func(item *File) int {
				return item.ID()
			},
		),
		func(item *File, index int) *ent.File {
			return item.Model
		},
	)
}

func WithBypassOwnerCheck(ctx context.Context) context.Context {
	return context.WithValue(ctx, ByPassOwnerCheckCtxKey{}, true)
}
