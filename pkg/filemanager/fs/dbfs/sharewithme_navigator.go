package dbfs

import (
	"context"
	"fmt"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/samber/lo"
)

var sharedWithMeNavigatorCapability = &boolset.BooleanSet{}

// sharedWithMeRootCapability is stamped on the synthetic flat-listing root;
// the listing itself accepts no writes — only entries beneath it do.
var sharedWithMeRootCapability = &boolset.BooleanSet{}

// NewSharedWithMeNavigator creates a navigator for user's "shared with me" file system.
func NewSharedWithMeNavigator(u *ent.User, fileClient inventory.FileClient, aclClient inventory.AclClient,
	l logging.Logger, config *setting.DBFS, hasher hashid.Encoder) Navigator {
	n := &sharedWithMeNavigator{
		user:       u,
		l:          l,
		fileClient: fileClient,
		aclClient:  aclClient,
		config:     config,
		hasher:     hasher,
		aclRoots:   map[int]*File{},
	}
	n.baseNavigator = newBaseNavigator(fileClient, defaultFilter, u, hasher, config)
	return n
}

type sharedWithMeNavigator struct {
	l          logging.Logger
	user       *ent.User
	fileClient inventory.FileClient
	aclClient  inventory.AclClient
	config     *setting.DBFS
	hasher     hashid.Encoder

	root *File
	// aclRoots caches resolved group/ACL-shared subtrees by root file ID so
	// repeated To() calls within a request reuse the built parent chain.
	aclRoots map[int]*File
	// aclCaps carries the ACL-derived capability set of the most recently
	// resolved subtree root; nil at the flat listing root.
	aclCaps *boolset.BooleanSet
	// sharedFiles caches SharedFileIDs for the request scope.
	sharedFiles    map[int]*boolset.BooleanSet
	sharedResolved bool
	*baseNavigator
}

func (t *sharedWithMeNavigator) Recycle() {
	for _, root := range t.aclRoots {
		root.Recycle()
	}
	if t.root != nil {
		t.root.Recycle()
	}
}

func (n *sharedWithMeNavigator) PersistState(kv cache.Driver, key string) {
}

func (n *sharedWithMeNavigator) RestoreState(s State) error {
	return nil
}

func (t *sharedWithMeNavigator) To(ctx context.Context, path *fs.URI) (*File, error) {
	// Anonymous user does not have a shared-with-me folder.
	if inventory.IsAnonymousUser(t.user) {
		return nil, ErrLoginRequired
	}

	elements := path.Elements()
	if len(elements) == 0 {
		t.aclCaps = nil
		return t.flatRoot(ctx)
	}

	// The first path element is the hashid of the shared file itself;
	// deeper elements walk into it like a regular folder subtree.
	root, err := t.resolveSharedRoot(ctx, elements[0])
	if err != nil {
		return nil, err
	}

	current, lastAncestor := root, root
	for index := 1; index < len(elements); index++ {
		lastAncestor = current
		current, err = t.walkNext(ctx, current, elements[index], index == len(elements)-1)
		if err != nil {
			return lastAncestor, fmt.Errorf("failed to walk into %q: %w", elements[index], err)
		}
	}

	return current, nil
}

// flatRoot builds the synthetic listing root shown at cloudreve://shared_with_me.
func (t *sharedWithMeNavigator) flatRoot(ctx context.Context) (*File, error) {
	if t.root == nil {
		rootFile, err := t.fileClient.Root(ctx, t.user)
		if err != nil {
			t.l.Info("User's root folder not found: %s, will initialize it.", err)
			return nil, ErrFsNotInitialized
		}

		t.root = newFile(nil, rootFile)
		rootPath := newSharedWithMeUri("")
		t.root.Path[pathIndexRoot], t.root.Path[pathIndexUser] = rootPath, rootPath
		t.root.OwnerModel = t.user
		t.root.IsUserRoot = true
		t.root.CapabilitiesBs = sharedWithMeRootCapability
	}

	return t.root, nil
}

// resolveSharedRoot resolves the hashid path element to a browsable subtree
// root: either the user's own symbolic share shortcut (whose redirect the
// caller translates) or a file explicitly shared to the user via ACL.
func (t *sharedWithMeNavigator) resolveSharedRoot(ctx context.Context, elem string) (*File, error) {
	fileID, err := t.hasher.Decode(elem, hashid.FileID)
	if err != nil {
		return nil, fs.ErrPathNotExist.WithError(err)
	}

	if cached, ok := t.aclRoots[fileID]; ok {
		t.aclCaps = cached.CapabilitiesBs
		return cached, nil
	}

	loadCtx := context.WithValue(ctx, inventory.LoadFileUser{}, true)
	model, err := t.fileClient.GetByID(loadCtx, fileID)
	if err != nil {
		return nil, fs.ErrPathNotExist.WithError(err)
	}

	res := newFile(nil, model)
	res.IsUserRoot = true
	res.Path[pathIndexUser] = newSharedWithMeUri(elem)

	if model.OwnerID == t.user.ID && model.IsSymbolic {
		// The user's own saved share shortcut: resolution exists so that
		// walking deeper surfaces ErrSymbolicFolderFound and DBFS can
		// translate the address to the share URI.
		res.OwnerModel = t.user
		res.CapabilitiesBs = myNavigatorCapability
		t.aclCaps = res.CapabilitiesBs
		t.aclRoots[fileID] = res
		return res, nil
	}

	perms, err := t.aclClient.EffectivePermissions(ctx, fileID, t.user)
	if err != nil {
		return nil, fmt.Errorf("failed to query ACL permissions: %w", err)
	}
	if perms == nil || !perms.Enabled(int(types.AclPermRead)) {
		return nil, fs.ErrPathNotExist
	}

	owner := model.Edges.Owner
	if owner == nil {
		return nil, fs.ErrPathNotExist
	}
	res.OwnerModel = owner
	res.CapabilitiesBs = aclPermsToCapabilities(perms)

	// Rebuild the owner-side parent chain so storage operations resolve
	// against the owner's filesystem, then confirm the file still lives
	// under a real user root (not the trash bin).
	ownerRoot, err := t.findRoot(ctx, res)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve shared file root: %w", err)
	}
	if ownerRoot.Name() != inventory.RootFolderName {
		return nil, fs.ErrPathNotExist
	}
	ownerRoot.Path[pathIndexRoot] = newMyIDUri(hashid.EncodeUserID(t.hasher, owner.ID))

	t.aclCaps = res.CapabilitiesBs
	t.aclRoots[fileID] = res
	return res, nil
}

// sharedFilePerms returns (and caches) the file→permissions map of ACL
// grants visible to the acting user.
func (t *sharedWithMeNavigator) sharedFilePerms(ctx context.Context) (map[int]*boolset.BooleanSet, error) {
	if t.sharedResolved {
		return t.sharedFiles, nil
	}

	perms, err := t.aclClient.SharedFileIDs(ctx, t.user)
	if err != nil {
		return nil, err
	}
	t.sharedFiles = perms
	t.sharedResolved = true
	return t.sharedFiles, nil
}

func (t *sharedWithMeNavigator) Children(ctx context.Context, parent *File, args *ListArgs) (*ListResult, error) {
	// Inside an ACL-shared subtree the folder is a real model — list it
	// like any other owner tree.
	if parent != nil && parent != t.root {
		return t.baseNavigator.children(ctx, parent, args)
	}

	perms, err := t.sharedFilePerms(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query shared ACL entries: %w", err)
	}

	// decorate stamps the shared_with_me address, ACL-derived capabilities
	// and the real owner on a listed entry; applied to both the returned
	// page and every streamed batch.
	decorate := func(files []*File) []*File {
		res := files[:0]
		for _, f := range files {
			if p, ok := perms[f.Model.ID]; ok {
				// ACL-granted entry: stamp the effective capability set and
				// the real owner so writePermitted and Owned resolve
				// correctly. Ownerless rows are dropped.
				if f.Model.Edges.Owner == nil {
					continue
				}
				f.OwnerModel = f.Model.Edges.Owner
				f.CapabilitiesBs = aclPermsToCapabilities(p)
			}
			f.Path[pathIndexUser] = newSharedWithMeUri(hashid.EncodeFileID(t.hasher, f.Model.ID))
			f.IsUserRoot = true
			res = append(res, f)
		}
		return res
	}

	var streamCallback func([]*File)
	if args.StreamCallback != nil {
		streamCallback = func(files []*File) {
			args.StreamCallback(decorate(files))
		}
	}

	loadCtx := context.WithValue(ctx, inventory.LoadFileUser{}, true)
	res, err := t.baseNavigator.children(loadCtx, nil, &ListArgs{
		Page:           args.Page,
		Search:         args.Search,
		SharedWithMe:   true,
		StreamCallback: streamCallback,
		AclSharedIDs:   lo.Keys(perms),
	})
	if err != nil {
		return nil, err
	}

	res.Files = decorate(res.Files)
	return res, nil
}

func (t *sharedWithMeNavigator) Capabilities(isSearching bool) *fs.NavigatorProps {
	res := baseNavigatorProps(sharedWithMeNavigatorCapability, t.config.MaxPageSize)
	if t.aclCaps != nil {
		res.Capability = t.aclCaps
	}

	if isSearching {
		res.OrderByOptions = searchLimitedOrderByOption
	}

	return res
}

func (t *sharedWithMeNavigator) Walk(ctx context.Context, levelFiles []*File, limit, depth int, f WalkFunc) error {
	return t.baseNavigator.walk(ctx, levelFiles, limit, depth, f)
}

func (n *sharedWithMeNavigator) FollowTx(ctx context.Context) (func(), error) {
	oldBase := n.baseNavigator.fileClient
	revertFile, err := followTxClients(ctx, &n.fileClient)
	if err != nil {
		return nil, err
	}

	revertAcl, err := followTxClients(ctx, &n.aclClient)
	if err != nil {
		revertFile()
		return nil, err
	}

	n.baseNavigator.fileClient = n.fileClient
	return func() {
		revertAcl()
		revertFile()
		n.baseNavigator.fileClient = oldBase
	}, nil
}

func (n *sharedWithMeNavigator) ExecuteHook(ctx context.Context, hookType fs.HookType, file *File) error {
	return nil
}

func (n *sharedWithMeNavigator) GetView(ctx context.Context, file *File) *types.ExplorerView {
	if file != nil && file != n.root {
		return file.View()
	}
	if n.user.Settings != nil {
		if view, ok := n.user.Settings.FsViewMap[string(constants.FileSystemSharedWithMe)]; ok {
			return &view
		}
	}
	return getDefaultView()
}
