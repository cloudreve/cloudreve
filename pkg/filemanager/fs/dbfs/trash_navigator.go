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
)

var (
	trashNavigatorCapability = &boolset.BooleanSet{}
	defaultTrashView         = &types.ExplorerView{
		View: "list",
		Columns: []types.ListViewColumn{
			{
				Type: 0,
			},
			{
				Type: 2,
			},
			{
				Type: 8,
			},
			{
				Type: 7,
			},
		},
	}
)

// NewTrashNavigator creates a navigator for user's "trash" file system.
func NewTrashNavigator(u *ent.User, fileClient inventory.FileClient, l logging.Logger, config *setting.DBFS,
	hasher hashid.Encoder) Navigator {
	return &trashNavigator{
		user:          u,
		l:             l,
		fileClient:    fileClient,
		config:        config,
		baseNavigator: newBaseNavigator(fileClient, defaultFilter, u, hasher, config),
	}
}

type trashNavigator struct {
	l          logging.Logger
	user       *ent.User
	fileClient inventory.FileClient
	config     *setting.DBFS

	*baseNavigator
}

func (t *trashNavigator) Recycle() {

}

func (n *trashNavigator) PersistState(kv cache.Driver, key string) {
}

func (n *trashNavigator) RestoreState(s State) error {
	return nil
}

func (t *trashNavigator) To(ctx context.Context, path *fs.URI) (*File, error) {
	// Anonymous user does not have a trash folder.
	if inventory.IsAnonymousUser(t.user) {
		return nil, ErrLoginRequired
	}

	elements := path.Elements()
	if len(elements) > 1 {
		// Trash folder is a flatten tree, only 1 layer is supported.
		return nil, fs.ErrPathNotExist.WithError(fmt.Errorf("invalid Path %q", path))
	}

	if len(elements) == 0 {
		// Trash folder has no root.
		return nil, nil
	}

	current, err := t.walkNext(ctx, nil, elements[0], true)
	if err != nil {
		return nil, fmt.Errorf("failed to walk into %q: %w", elements[0], err)
	}

	current.Path[pathIndexUser] = newTrashUri(current.Model.Name)
	current.Path[pathIndexRoot] = current.Path[pathIndexUser]
	current.OwnerModel = t.user
	return current, nil
}

func (t *trashNavigator) Children(ctx context.Context, parent *File, args *ListArgs) (*ListResult, error) {
	if parent != nil {
		return nil, fs.ErrPathNotExist
	}

	res, err := t.baseNavigator.children(ctx, nil, args)
	if err != nil {
		return nil, err
	}

	// Adding user uri for each file.
	for i := 0; i < len(res.Files); i++ {
		res.Files[i].Path[pathIndexUser] = newTrashUri(res.Files[i].Model.Name)
	}

	return res, nil
}

func (t *trashNavigator) Capabilities(isSearching bool) *fs.NavigatorProps {
	res := baseNavigatorProps(trashNavigatorCapability, t.config.MaxPageSize)
	if isSearching {
		res.OrderByOptions = searchLimitedOrderByOption
	}

	return res
}

func (t *trashNavigator) Walk(ctx context.Context, levelFiles []*File, limit, depth int, f WalkFunc) error {
	return t.baseNavigator.walk(ctx, levelFiles, limit, depth, f)
}

func (n *trashNavigator) FollowTx(ctx context.Context) (func(), error) {
	oldBase := n.baseNavigator.fileClient
	revert, err := followTxClients(ctx, &n.fileClient)
	if err != nil {
		return nil, err
	}

	n.baseNavigator.fileClient = n.fileClient
	return func() {
		revert()
		n.baseNavigator.fileClient = oldBase
	}, nil
}

func (n *trashNavigator) ExecuteHook(ctx context.Context, hookType fs.HookType, file *File) error {
	return nil
}

func (n *trashNavigator) GetView(ctx context.Context, file *File) *types.ExplorerView {
	if view, ok := n.user.Settings.FsViewMap[string(constants.FileSystemTrash)]; ok {
		return &view
	}
	return defaultTrashView
}
