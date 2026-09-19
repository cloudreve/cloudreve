package dbfs

import (
	"context"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	entfile "github.com/cloudreve/Cloudreve/v4/ent/file"
	entmetadata "github.com/cloudreve/Cloudreve/v4/ent/metadata"
	"github.com/cloudreve/Cloudreve/v4/ent/storagepolicy"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/stretchr/testify/require"
)

type defaultShareSettingProvider struct {
	setting.Provider
	ids []int
}

func (p defaultShareSettingProvider) DefaultShares(context.Context) []int {
	return p.ids
}

// defaultShareFixture creates a share owner with shared file/dir, and a
// separate new user with an empty root. Returns (newUser, newUserRoot,
// fileShare, dirShare, owner).
func defaultShareFixture(t *testing.T, client *ent.Client) (*ent.User, *ent.File, *ent.Share, *ent.Share, *ent.User) {
	t.Helper()
	ctx := context.Background()
	policy := client.StoragePolicy.Create().SetName("local").SetType("local").
		SetStatus(storagepolicy.StatusActive).SaveX(ctx)
	permissions := &boolset.BooleanSet{}
	boolset.Set(types.GroupPermissionShare, true, permissions)
	group := client.Group.Create().SetName("seed").SetPermissions(permissions).
		SetStoragePolicies(policy).SaveX(ctx)

	owner := client.User.Create().SetEmail("owner@example.com").SetNick("o").SetGroup(group).SaveX(ctx)
	ownerRoot := client.File.Create().SetName(inventory.RootFolderName).SetType(int(types.FileTypeFolder)).SetOwner(owner).SaveX(ctx)
	sharedFile := client.File.Create().SetName("shared.txt").SetType(int(types.FileTypeFile)).SetOwner(owner).SetParent(ownerRoot).SaveX(ctx)
	sharedDir := client.File.Create().SetName("shared_dir").SetType(int(types.FileTypeFolder)).SetOwner(owner).SetParent(ownerRoot).SaveX(ctx)
	fileShare := client.Share.Create().SetUser(owner).SetFile(sharedFile).SaveX(ctx)
	dirShare := client.Share.Create().SetUser(owner).SetFile(sharedDir).SaveX(ctx)

	newUser := client.User.Create().SetEmail("new@example.com").SetNick("n").SetGroup(group).SaveX(ctx)
	newRoot := client.File.Create().SetName(inventory.RootFolderName).SetType(int(types.FileTypeFolder)).SetOwner(newUser).SaveX(ctx)
	return newUser, newRoot, fileShare, dirShare, owner
}

func seedTestDBFS(t *testing.T, client *ent.Client, ids []int) *DBFS {
	t.Helper()
	hasher, err := hashid.New("seed-test-salt")
	require.NoError(t, err)
	return &DBFS{
		fileClient:          inventory.NewFileClient(client, conf.SQLiteDB, hasher),
		shareClient:         inventory.NewShareClient(client, conf.SQLiteDB, hasher),
		userClient:          inventory.NewUserClient(client),
		storagePolicyClient: inventory.NewStoragePolicyClient(client, nil),
		settingClient:       defaultShareSettingProvider{ids: ids},
		hasher:              hasher,
		l:                   logging.NewConsoleLogger(logging.LevelError),
	}
}

func metadataOf(t *testing.T, client *ent.Client, f *ent.File) map[string]string {
	t.Helper()
	rows := client.Metadata.Query().Where(entmetadata.HasFileWith(entfile.ID(f.ID))).AllX(context.Background())
	res := make(map[string]string, len(rows))
	for _, r := range rows {
		res[r.Name] = r.Value
	}
	return res
}

func TestSeedDefaultSharesCreatesSymbolicShortcuts(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	user, root, fileShare, dirShare, owner := defaultShareFixture(t, client)
	f := seedTestDBFS(t, client, []int{fileShare.ID, dirShare.ID})
	f.seedDefaultShares(ctx, user.ID, root)

	shortcuts := client.File.Query().
		Where(entfile.FileChildren(root.ID), entfile.IsSymbolic(true)).
		AllX(ctx)
	require.Len(t, shortcuts, 2)

	byName := map[string]*ent.File{}
	for _, c := range shortcuts {
		byName[c.Name] = c
	}

	fileShortcut := byName["shared.txt"]
	require.NotNil(t, fileShortcut)
	require.Equal(t, int(types.FileTypeFile), fileShortcut.Type)
	require.Equal(t, user.ID, fileShortcut.OwnerID)

	dirShortcut := byName["shared_dir"]
	require.NotNil(t, dirShortcut)
	require.Equal(t, int(types.FileTypeFolder), dirShortcut.Type)

	fm := metadataOf(t, client, fileShortcut)
	require.Equal(t, fs.NewShareUri(hashid.EncodeShareID(f.hasher, fileShare.ID), ""), fm[MetadataSharedRedirect])
	require.Equal(t, hashid.EncodeUserID(f.hasher, owner.ID), fm[MetadataSharedOwner])

	dm := metadataOf(t, client, dirShortcut)
	require.Equal(t, fs.NewShareUri(hashid.EncodeShareID(f.hasher, dirShare.ID), ""), dm[MetadataSharedRedirect])
}

func TestSeedDefaultSharesSkipsInvalidIDs(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	user, root, fileShare, _, _ := defaultShareFixture(t, client)
	f := seedTestDBFS(t, client, []int{0, -1, 99999, fileShare.ID})
	f.seedDefaultShares(ctx, user.ID, root)

	shortcuts := client.File.Query().
		Where(entfile.FileChildren(root.ID), entfile.IsSymbolic(true)).
		AllX(ctx)
	require.Len(t, shortcuts, 1)
	require.Equal(t, "shared.txt", shortcuts[0].Name)
}

func TestSeedDefaultSharesNoConfigIsNoop(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	user, root, _, _, _ := defaultShareFixture(t, client)
	f := seedTestDBFS(t, client, nil)
	f.seedDefaultShares(ctx, user.ID, root)

	shortcuts := client.File.Query().
		Where(entfile.FileChildren(root.ID), entfile.IsSymbolic(true)).
		AllX(ctx)
	require.Empty(t, shortcuts)
}

func TestSeedDefaultSharesMergesGroupPinned(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	// Site config seeds the file share; the new user's group pins the dir share.
	user, root, fileShare, dirShare, _ := defaultShareFixture(t, client)
	group := user.QueryGroup().OnlyX(ctx)
	client.Group.UpdateOne(group).SetSettings(&types.GroupSetting{DefaultPinned: []int{dirShare.ID}}).SaveX(ctx)
	f := seedTestDBFS(t, client, []int{fileShare.ID})
	f.seedDefaultShares(ctx, user.ID, root)

	shortcuts := client.File.Query().
		Where(entfile.FileChildren(root.ID), entfile.IsSymbolic(true)).
		AllX(ctx)
	require.Len(t, shortcuts, 2)
}
