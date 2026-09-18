package dbfs

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/samber/lo"
)

var (
	ErrFsNotInitialized = fmt.Errorf("fs not initialized")
	ErrPermissionDenied = serializer.NewError(serializer.CodeNoPermissionErr, "Permission denied", nil)

	ErrShareIncorrectPassword  = serializer.NewError(serializer.CodeIncorrectPassword, "Incorrect share password", nil)
	ErrFileCountLimitedReached = serializer.NewError(serializer.CodeFileCountLimitedReached, "Walked file count reached limit", nil)
	ErrSymbolicFolderFound     = serializer.NewError(serializer.CodeNoPermissionErr, "Symbolic folder cannot be walked into", nil)
	ErrLoginRequired           = serializer.NewError(serializer.CodeCheckLogin, "Login required", nil)

	fullOrderByOption          = []string{"name", "size", "updated_at", "created_at"}
	searchLimitedOrderByOption = []string{"created_at"}
	fullOrderDirectionOption   = []string{"asc", "desc"}
)

type (
	// Navigator is a navigator for database file system.
	Navigator interface {
		Recycle()
		// To returns the file by path. If given path is not exist, returns ErrFileNotFound and most-recent ancestor.
		To(ctx context.Context, path *fs.URI) (*File, error)
		// Children returns the children of the parent file.
		Children(ctx context.Context, parent *File, args *ListArgs) (*ListResult, error)
		// Capabilities returns the capabilities of the navigator.
		Capabilities(isSearching bool) *fs.NavigatorProps
		// Walk walks the file tree until limit is reached.
		Walk(ctx context.Context, levelFiles []*File, limit, depth int, f WalkFunc) error
		// PersistState tells navigator to persist the state of the navigator before recycle.
		PersistState(kv cache.Driver, key string)
		// RestoreState restores the state of the navigator.
		RestoreState(s State) error
		// FollowTx let the navigator inherit the transaction. Return a function to reset back to previous DB client.
		FollowTx(ctx context.Context) (func(), error)
		// ExecuteHook performs custom operations before or after certain actions.
		ExecuteHook(ctx context.Context, hookType fs.HookType, file *File) error
		// GetView returns the view setting of the given file.
		GetView(ctx context.Context, file *File) *types.ExplorerView
	}

	State interface{}

	NavigatorCapability int
	ListArgs            struct {
		Page           *inventory.PaginationArgs
		Search         *inventory.SearchFileParameters
		SharedWithMe   bool
		StreamCallback func([]*File)
	}
	// ListResult is the result of a list operation.
	ListResult struct {
		Files                 []*File
		MixedType             bool
		Pagination            *inventory.PaginationResults
		RecursionLimitReached bool
		SingleFileView        bool
	}
	WalkFunc func([]*File, int) error
)

const (
	NavigatorCapabilityCreateFile NavigatorCapability = iota
	NavigatorCapabilityRenameFile
	NavigatorCapability_CommunityPlacehodler1
	NavigatorCapability_CommunityPlacehodler2
	NavigatorCapability_CommunityPlacehodler3
	NavigatorCapability_CommunityPlacehodler4
	NavigatorCapabilityUploadFile
	NavigatorCapabilityDownloadFile
	NavigatorCapabilityUpdateMetadata
	NavigatorCapabilityListChildren
	NavigatorCapabilityGenerateThumb
	NavigatorCapability_CommunityPlacehodler5
	NavigatorCapability_CommunityPlacehodler6
	NavigatorCapability_CommunityPlacehodler7
	NavigatorCapabilityDeleteFile
	NavigatorCapabilityLockFile
	NavigatorCapabilitySoftDelete
	NavigatorCapabilityRestore
	NavigatorCapabilityShare
	NavigatorCapabilityInfo
	NavigatorCapabilityVersionControl
	NavigatorCapability_CommunityPlacehodler8
	NavigatorCapability_CommunityPlacehodler9
	NavigatorCapabilityEnterFolder
	NavigatorCapabilityModifyProps

	searchTokenSeparator = "|"
)

func init() {
	boolset.Sets(map[NavigatorCapability]bool{
		NavigatorCapabilityCreateFile:     true,
		NavigatorCapabilityRenameFile:     true,
		NavigatorCapabilityUploadFile:     true,
		NavigatorCapabilityDownloadFile:   true,
		NavigatorCapabilityUpdateMetadata: true,
		NavigatorCapabilityListChildren:   true,
		NavigatorCapabilityGenerateThumb:  true,
		NavigatorCapabilityDeleteFile:     true,
		NavigatorCapabilityLockFile:       true,
		NavigatorCapabilitySoftDelete:     true,
		NavigatorCapabilityShare:          true,
		NavigatorCapabilityInfo:           true,
		NavigatorCapabilityVersionControl: true,
		NavigatorCapabilityEnterFolder:    true,
		NavigatorCapabilityModifyProps:    true,
	}, myNavigatorCapability)
	// Static superset admitting every action a share's props may grant;
	// per-share props narrow it after share resolution.
	boolset.Sets(map[NavigatorCapability]bool{
		NavigatorCapabilityCreateFile:     true,
		NavigatorCapabilityRenameFile:     true,
		NavigatorCapabilityUploadFile:     true,
		NavigatorCapabilityDownloadFile:   true,
		NavigatorCapabilityListChildren:   true,
		NavigatorCapabilityGenerateThumb:  true,
		NavigatorCapabilityDeleteFile:     true,
		NavigatorCapabilityLockFile:       true,
		NavigatorCapabilitySoftDelete:     true,
		NavigatorCapabilityInfo:           true,
		NavigatorCapabilityVersionControl: true,
		NavigatorCapabilityEnterFolder:    true,
		NavigatorCapabilityModifyProps:    true,
	}, shareNavigatorCapability)
	boolset.Sets(map[NavigatorCapability]bool{
		NavigatorCapabilityListChildren: true,
		NavigatorCapabilityDeleteFile:   true,
		NavigatorCapabilityLockFile:     true,
		NavigatorCapabilityRestore:      true,
		NavigatorCapabilityInfo:         true,
	}, trashNavigatorCapability)
	boolset.Sets(map[NavigatorCapability]bool{
		NavigatorCapabilityListChildren: true,
		NavigatorCapabilityDownloadFile: true,
		NavigatorCapabilityEnterFolder:  true,
	}, sharedWithMeNavigatorCapability)
}

// ==================== Base Navigator ====================
type (
	fileFilter    func(ctx context.Context, f *File) (*File, bool)
	baseNavigator struct {
		fileClient inventory.FileClient
		listFilter fileFilter
		user       *ent.User
		hasher     hashid.Encoder
		config     *setting.DBFS
	}
)

var defaultFilter = func(ctx context.Context, f *File) (*File, bool) { return f, true }

func newBaseNavigator(fileClient inventory.FileClient, filterFunc fileFilter, user *ent.User,
	hasher hashid.Encoder, config *setting.DBFS) *baseNavigator {
	return &baseNavigator{
		fileClient: fileClient,
		listFilter: filterFunc,
		user:       user,
		hasher:     hasher,
		config:     config,
	}
}

func (b *baseNavigator) walkNext(ctx context.Context, root *File, next string, isLeaf bool) (*File, error) {
	var model *ent.File
	if root != nil {
		model = root.Model
		if root.IsSymbolic() {
			return nil, ErrSymbolicFolderFound
		}

		root.mu.Lock()
		if child, ok := root.Children[next]; ok && !isLeaf {
			root.mu.Unlock()
			return child, nil
		}
		root.mu.Unlock()
	}

	child, err := b.fileClient.GetChildFile(ctx, model, b.user.ID, next, isLeaf)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fs.ErrPathNotExist.WithError(err)
		}

		return nil, fmt.Errorf("faield to get child %q: %w", next, err)
	}

	return newFile(root, child), nil
}

// findRoot finds the root folder of the given child.
func (b *baseNavigator) findRoot(ctx context.Context, child *File) (*File, error) {
	root := child
	for {
		newRoot, err := b.walkUp(ctx, root)
		if err != nil {
			if !ent.IsNotFound(err) {
				return nil, err
			}

			break
		}

		root = newRoot
	}

	return root, nil
}

func (b *baseNavigator) walkUp(ctx context.Context, child *File) (*File, error) {
	parent, err := b.fileClient.GetParentFile(ctx, child.Model, false)
	if err != nil {
		return nil, fmt.Errorf("faield to get Parent for %q: %w", child.Name(), err)
	}

	return newParentFile(parent, child), nil
}

func (b *baseNavigator) children(ctx context.Context, parent *File, args *ListArgs) (*ListResult, error) {
	var model *ent.File
	if parent != nil {
		model = parent.Model
		if parent.Model.Type != int(types.FileTypeFolder) {
			return nil, fs.ErrPathNotExist
		}

		if parent.IsSymbolic() {
			return nil, ErrSymbolicFolderFound
		}

		parent.Path[pathIndexUser] = parent.Uri(false)
	}

	if args.Search != nil {
		return b.search(ctx, parent, args)
	}

	children, err := b.fileClient.GetChildFiles(ctx, &inventory.ListFileParameters{
		PaginationArgs: args.Page,
		SharedWithMe:   args.SharedWithMe,
	}, b.user.ID, model)
	if err != nil {
		return nil, fmt.Errorf("failed to get children: %w", err)
	}

	return &ListResult{
		Files: lo.FilterMap(children.Files, func(model *ent.File, index int) (*File, bool) {
			f := newFile(parent, model)
			return b.listFilter(ctx, f)
		}),
		MixedType:  children.MixedType,
		Pagination: children.PaginationResults,
	}, nil
}

// walkFetchPageSize caps how many child rows a single GetChildFiles query
// may return during walk. Bounding the page (instead of fetching a whole
// level at once) keeps peak memory proportional to page size + folder
// count rather than tree size.
const walkFetchPageSize = 2000

func (b *baseNavigator) walk(ctx context.Context, levelFiles []*File, limit, depth int, f WalkFunc) error {
	walked := 0
	if len(levelFiles) == 0 {
		return nil
	}

	owner := levelFiles[0].Owner()
	level := 0

	// Files still to emit at the current level. For level 0 this is the
	// caller-provided slice; deeper levels are paged from the DB below.
	pending := levelFiles
	var parentMap map[int]*File
	var parentModels []*ent.File
	token := ""

	for depth >= 0 {
		depth--
		folders := make([]*File, 0, 64)

		// Emit this level in bounded batches, tracking folder nodes for
		// the next level.
		for {
			var batch []*File
			if parentModels == nil {
				if len(pending) == 0 {
					break
				}
				remaining := limit - walked
				if remaining <= 0 {
					return ErrFileCountLimitedReached
				}
				batch = pending[:min(len(pending), remaining)]
				pending = pending[len(batch):]
			} else {
				remaining := limit - walked
				if remaining <= 0 {
					return ErrFileCountLimitedReached
				}
				res, err := b.fileClient.GetChildFiles(ctx,
					&inventory.ListFileParameters{
						PaginationArgs: &inventory.PaginationArgs{
							UseCursorPagination: true,
							PageToken:           token,
							PageSize:            min(remaining, walkFetchPageSize),
						},
						MixedType: true,
					},
					owner.ID,
					parentModels...)
				if err != nil {
					return serializer.NewError(serializer.CodeDBError, "Failed to list children", err)
				}
				if len(res.Files) == 0 {
					break
				}
				batch = lo.Map(res.Files, func(model *ent.File, index int) *File {
					return newFile(parentMap[model.FileChildren], model)
				})
				token = res.NextPageToken
			}

			if err := f(batch, level); err != nil {
				return err
			}
			walked += len(batch)
			for _, file := range batch {
				if file.Model.Type == int(types.FileTypeFolder) && !file.IsSymbolic() {
					folders = append(folders, file)
				}
			}

			if parentModels == nil {
				continue
			}
			if token == "" {
				break
			}
		}

		if len(folders) == 0 {
			return nil
		}

		level++
		parentMap = lo.SliceToMap(folders, func(file *File) (int, *File) {
			return file.Model.ID, file
		})
		parentModels = lo.Map(folders, func(item *File, index int) *ent.File {
			return item.Model
		})
		token = ""
	}

	return nil
}

func (b *baseNavigator) search(ctx context.Context, parent *File, args *ListArgs) (*ListResult, error) {
	if parent == nil {
		// Performs mega search for all files in trash fs.
		children, err := b.fileClient.GetChildFiles(ctx, &inventory.ListFileParameters{
			PaginationArgs: args.Page,
			MixedType:      true,
			Search:         args.Search,
			SharedWithMe:   args.SharedWithMe,
		}, b.user.ID, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get children: %w", err)
		}

		return &ListResult{
			Files: lo.FilterMap(children.Files, func(model *ent.File, index int) (*File, bool) {
				f := newFile(parent, model)
				return b.listFilter(ctx, f)
			}),
			MixedType:  children.MixedType,
			Pagination: children.PaginationResults,
		}, nil
	}
	// Performs recursive search for all files under the given folder.
	walkedFolder := 1
	parents := []map[int]*File{{parent.Model.ID: parent}}
	startLevel, innerPageToken, err := parseSearchPageToken(args.Page.PageToken)
	if err != nil {
		return nil, err
	}
	args.Page.PageToken = innerPageToken

	stepLevel := func(level int) (bool, error) {
		token := ""
		// We don't need metadata in level search.
		listCtx := context.WithValue(ctx, inventory.LoadFilePublicMetadata{}, nil)
		for walkedFolder <= b.config.MaxRecursiveSearchedFolder {
			// TODO: chunk parents into 30000 per group
			res, err := b.fileClient.GetChildFiles(listCtx,
				&inventory.ListFileParameters{
					PaginationArgs: &inventory.PaginationArgs{
						UseCursorPagination: true,
						PageToken:           token,
					},
					FolderOnly: true,
				},
				parent.Model.OwnerID,
				lo.MapToSlice(parents[level], func(k int, f *File) *ent.File {
					return f.Model
				})...)
			if err != nil {
				return false, serializer.NewError(serializer.CodeDBError, "Failed to list children", err)
			}

			parents = append(parents, lo.SliceToMap(
				lo.FilterMap(res.Files, func(model *ent.File, index int) (*File, bool) {
					p := parents[level][model.FileChildren]
					f := newFile(p, model)
					f.Path[pathIndexUser] = p.Uri(false).Join(model.Name)
					return f, true
				}),
				func(f *File) (int, *File) {
					return f.Model.ID, f
				}))

			walkedFolder += len(parents[level+1])
			if res.NextPageToken == "" {
				break
			}

			token = res.NextPageToken
		}

		if len(parents) <= level+1 || len(parents[level+1]) == 0 {
			// All possible folders is searched
			return true, nil
		}

		return false, nil
	}

	// We need to walk from root folder to get the correct level.
	for level := 0; level < startLevel; level++ {
		stop, err := stepLevel(level)
		if err != nil {
			return nil, err
		}

		if stop {
			return &ListResult{}, nil
		}
	}

	// Search files starting from current level
	res := make([]*File, 0, args.Page.PageSize)
	args.Page.UseCursorPagination = true
	originalPageSize := args.Page.PageSize
	stop := false
	for len(res) < originalPageSize && walkedFolder <= b.config.MaxRecursiveSearchedFolder {
		// Only requires minimum number of files
		args.Page.PageSize = min(originalPageSize, originalPageSize-len(res))
		searchRes, err := b.fileClient.GetChildFiles(ctx,
			&inventory.ListFileParameters{
				PaginationArgs: args.Page,
				MixedType:      true,
				Search:         args.Search,
			},
			parent.Model.OwnerID,
			lo.MapToSlice(parents[startLevel], func(k int, f *File) *ent.File {
				return f.Model
			})...)

		if err != nil {
			return nil, serializer.NewError(serializer.CodeDBError, "Failed to search files", err)
		}

		newRes := lo.FilterMap(searchRes.Files, func(model *ent.File, index int) (*File, bool) {
			p := parents[startLevel][model.FileChildren]
			f := newFile(p, model)
			f.Path[pathIndexUser] = p.Uri(false).Join(model.Name)
			return b.listFilter(ctx, f)
		})
		res = append(res, newRes...)
		if args.StreamCallback != nil {
			args.StreamCallback(newRes)
		}

		args.Page.PageToken = searchRes.NextPageToken
		// If no more results under current level, move to next level
		if args.Page.PageToken == "" {
			if len(res) == originalPageSize {
				// Current page is full, no need to search more
				startLevel++
				break
			}

			finished, err := stepLevel(startLevel)
			if err != nil {
				return nil, err
			}

			if finished {
				stop = true
				// No more folders under next level, all result is presented
				break
			}

			startLevel++
		}
	}

	if args.StreamCallback != nil {
		// Clear res if it's streamed
		res = res[:0]
	}

	searchRes := &ListResult{
		Files:                 res,
		MixedType:             true,
		Pagination:            &inventory.PaginationResults{IsCursor: true},
		RecursionLimitReached: walkedFolder > b.config.MaxRecursiveSearchedFolder,
	}

	if walkedFolder <= b.config.MaxRecursiveSearchedFolder && !stop {
		searchRes.Pagination.NextPageToken = fmt.Sprintf("%d%s%s", startLevel, searchTokenSeparator, args.Page.PageToken)
	}

	return searchRes, nil
}

func parseSearchPageToken(token string) (int, string, error) {
	if token == "" {
		return 0, "", nil
	}

	tokens := strings.Split(token, searchTokenSeparator)
	if len(tokens) != 2 {
		return 0, "", fmt.Errorf("invalid page token")
	}

	level, err := strconv.Atoi(tokens[0])
	if err != nil || level < 0 {
		return 0, "", fmt.Errorf("invalid page token level")
	}

	return level, tokens[1], nil
}

func newMyUri() *fs.URI {
	res, _ := fs.NewUriFromString(constants.CloudreveScheme + "://" + string(constants.FileSystemMy))
	return res
}

func newMyIDUri(uid string) *fs.URI {
	res, _ := fs.NewUriFromString(fmt.Sprintf("%s://%s@%s", constants.CloudreveScheme, uid, constants.FileSystemMy))
	return res
}

func newTrashUri(name string) *fs.URI {
	res, _ := fs.NewUriFromString(fmt.Sprintf("%s://%s", constants.CloudreveScheme, constants.FileSystemTrash))
	return res.Join(name)
}

func newSharedWithMeUri(id string) *fs.URI {
	res, _ := fs.NewUriFromString(fmt.Sprintf("%s://%s", constants.CloudreveScheme, constants.FileSystemSharedWithMe))
	return res.Join(id)
}
