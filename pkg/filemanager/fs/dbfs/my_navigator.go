package dbfs

import (
	"context"
	"fmt"

	"github.com/cloudreve/Cloudreve/v4/inventory/types"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
)

var myNavigatorCapability = &boolset.BooleanSet{}

// NewMyNavigator creates a navigator for user's "my" file system.
func NewMyNavigator(u *ent.User, fileClient inventory.FileClient, userClient inventory.UserClient, l logging.Logger,
	config *setting.DBFS, hasher hashid.Encoder) Navigator {
	return &myNavigator{
		user:          u,
		l:             l,
		fileClient:    fileClient,
		userClient:    userClient,
		config:        config,
		baseNavigator: newBaseNavigator(fileClient, defaultFilter, u, hasher, config),
	}
}

type myNavigator struct {
	l          logging.Logger
	user       *ent.User
	fileClient inventory.FileClient
	userClient inventory.UserClient

	config *setting.DBFS
	*baseNavigator
	root           *File
	disableRecycle bool
	persist        func()
}

func (n *myNavigator) Recycle() {
	if n.persist != nil {
		n.persist()
		n.persist = nil
	}
	if n.root != nil && !n.disableRecycle {
		n.root.Recycle()
	}
}

func (n *myNavigator) PersistState(kv cache.Driver, key string) {
	n.disableRecycle = true
	n.persist = func() {
		kv.Set(key, n.root, ContextHintTTL)
	}
}

func (n *myNavigator) RestoreState(s State) error {
	n.disableRecycle = true
	if state, ok := s.(*File); ok {
		n.root = state
		return nil
	}

	return fmt.Errorf("invalid state type: %T", s)
}

func (n *myNavigator) To(ctx context.Context, path *fs.URI) (*File, error) {
	if n.root == nil {
		// Anonymous user does not have a root folder.
		if inventory.IsAnonymousUser(n.user) {
			return nil, ErrLoginRequired
		}

		fsUid, err := n.hasher.Decode(path.ID(hashid.EncodeUserID(n.hasher, n.user.ID)), hashid.UserID)
		if err != nil {
			return nil, fs.ErrPathNotExist.WithError(fmt.Errorf("invalid user id"))
		}
		if fsUid != n.user.ID {
			return nil, ErrPermissionDenied
		}

		ctx = context.WithValue(ctx, inventory.LoadUserGroup{}, true)
		targetUser, err := n.userClient.GetByID(ctx, fsUid)
		if err != nil {
			return nil, fs.ErrPathNotExist.WithError(fmt.Errorf("user not found: %w", err))
		}

		if targetUser.Status != user.StatusActive && !n.user.EnforceGroupPermission(types.GroupPermissionIsAdmin) {
			return nil, fs.ErrPathNotExist.WithError(fmt.Errorf("inactive user"))
		}

		rootFile, err := n.fileClient.Root(ctx, targetUser)
		if err != nil {
			n.l.Info("User's root folder not found: %s, will initialize it.", err)
			return nil, ErrFsNotInitialized
		}

		n.root = newFile(nil, rootFile)
		rootPath := path.Root()
		n.root.Path[pathIndexRoot], n.root.Path[pathIndexUser] = rootPath, rootPath
		n.root.OwnerModel = targetUser
		n.root.disableView = fsUid != n.user.ID
		n.root.IsUserRoot = true
		n.root.CapabilitiesBs = n.Capabilities(false).Capability
	}

	current, lastAncestor := n.root, n.root
	elements := path.Elements()
	var err error
	for index, element := range elements {
		lastAncestor = current
		current, err = n.walkNext(ctx, current, element, index == len(elements)-1)
		if err != nil {
			return lastAncestor, fmt.Errorf("failed to walk into %q: %w", element, err)
		}
	}

	return current, nil
}

func (n *myNavigator) Children(ctx context.Context, parent *File, args *ListArgs) (*ListResult, error) {
	return n.baseNavigator.children(ctx, parent, args)
}

func (n *myNavigator) walkNext(ctx context.Context, root *File, next string, isLeaf bool) (*File, error) {
	return n.baseNavigator.walkNext(ctx, root, next, isLeaf)
}

func (n *myNavigator) Capabilities(isSearching bool) *fs.NavigatorProps {
	res := &fs.NavigatorProps{
		Capability:            myNavigatorCapability,
		OrderDirectionOptions: fullOrderDirectionOption,
		OrderByOptions:        fullOrderByOption,
		MaxPageSize:           n.config.MaxPageSize,
	}
	if isSearching {
		res.OrderByOptions = nil
		res.OrderDirectionOptions = nil
	}

	return res
}

func (n *myNavigator) Walk(ctx context.Context, levelFiles []*File, limit, depth int, f WalkFunc) error {
	return n.baseNavigator.walk(ctx, levelFiles, limit, depth, f)
}

func (n *myNavigator) FollowTx(ctx context.Context) (func(), error) {
	if _, ok := ctx.Value(inventory.TxCtx{}).(*inventory.Tx); !ok {
		return nil, fmt.Errorf("navigator: no inherited transaction found in context")
	}
	newFileClient, _, _, err := inventory.WithTx(ctx, n.fileClient)
	if err != nil {
		return nil, err
	}

	newUserClient, _, _, err := inventory.WithTx(ctx, n.userClient)

	oldFileClient, oldUserClient := n.fileClient, n.userClient
	revert := func() {
		n.fileClient = oldFileClient
		n.userClient = oldUserClient
		n.baseNavigator.fileClient = oldFileClient
	}

	n.fileClient = newFileClient
	n.userClient = newUserClient
	n.baseNavigator.fileClient = newFileClient
	return revert, nil
}

func (n *myNavigator) ExecuteHook(ctx context.Context, hookType fs.HookType, file *File) error {
	return nil
}

func (n *myNavigator) GetView(ctx context.Context, file *File) *types.ExplorerView {
	return file.View()
}
