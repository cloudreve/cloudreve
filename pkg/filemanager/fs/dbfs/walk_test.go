package dbfs

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	entfile "github.com/cloudreve/Cloudreve/v4/ent/file"
	entuser "github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/stretchr/testify/require"
)

// buildWalkTree creates root -> folder children. Each folder gets
// filesPerFolder file children. Returns the root model wrapped as *File.
func buildWalkTree(t *testing.T, client *ent.Client, folders, filesPerFolder int) (*ent.User, *File) {
	ctx := context.Background()
	group := client.Group.Create().SetName("walkers").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	user := client.User.Create().SetEmail("walk@example.com").SetNick("walk").SetGroup(group).SaveX(ctx)
	root := client.File.Create().SetName(inventory.RootFolderName).SetType(int(types.FileTypeFolder)).SetOwner(user).SaveX(ctx)

	folderModels := make([]*ent.File, 0, folders)
	for i := 0; i < folders; i++ {
		folderModels = append(folderModels,
			client.File.Create().SetName(fmt.Sprintf("dir-%04d", i)).SetType(int(types.FileTypeFolder)).SetOwner(user).SetParent(root).SaveX(ctx))
	}
	for i, folder := range folderModels {
		creates := make([]*ent.FileCreate, 0, filesPerFolder)
		for j := 0; j < filesPerFolder; j++ {
			creates = append(creates,
				client.File.Create().SetName(fmt.Sprintf("f-%04d-%04d", i, j)).SetType(int(types.FileTypeFile)).SetOwner(user).SetParent(folder))
		}
		client.File.CreateBulk(creates...).SaveX(ctx)
	}

	rootFile := newFile(nil, root)
	rootFile.OwnerModel = user
	return user, rootFile
}

func walkTestNavigator(t *testing.T, client *ent.Client, user *ent.User) *baseNavigator {
	t.Helper()
	hasher, err := hashid.New("walk-test-salt")
	require.NoError(t, err)
	return newBaseNavigator(inventory.NewFileClient(client, conf.SQLiteDB, hasher), defaultFilter, user, nil, nil)
}

func TestWalkEmitsWholeTree(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	user, rootFile := buildWalkTree(t, client, 3, 5)

	nav := walkTestNavigator(t, client, user)

	emitted := make(map[int]int)  // level -> count
	seen := make(map[string]bool) // dedup guard
	err := nav.walk(context.Background(), []*File{rootFile}, 1000, 10, func(files []*File, level int) error {
		emitted[level] += len(files)
		for _, f := range files {
			key := fmt.Sprintf("%d:%d", level, f.ID())
			require.False(t, seen[key], "file %d emitted twice", f.ID())
			seen[key] = true
		}
		return nil
	})
	require.NoError(t, err)
	// level 0: root. level 1: 3 dirs. level 2: 15 files.
	require.Equal(t, 1, emitted[0])
	require.Equal(t, 3, emitted[1])
	require.Equal(t, 15, emitted[2])
	require.Equal(t, 19, len(seen))
}

func TestWalkBatchesWideLevels(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	// One folder with more children than walkFetchPageSize forces multiple
	// callback invocations for the same level — the pre-fix implementation
	// fetched and emitted the entire level at once.
	user, rootFile := buildWalkTree(t, client, 1, walkFetchPageSize+500)

	nav := walkTestNavigator(t, client, user)

	levelCalls := make(map[int]int)
	total := 0
	err := nav.walk(context.Background(), []*File{rootFile}, 10000, 10, func(files []*File, level int) error {
		levelCalls[level]++
		total += len(files)
		require.LessOrEqual(t, len(files), walkFetchPageSize, "callback batch exceeded page bound")
		return nil
	})
	require.NoError(t, err)
	// root + 1 dir + (walkFetchPageSize+500) files
	require.Equal(t, 2+walkFetchPageSize+500, total)
	require.Greater(t, levelCalls[2], 1, "wide level must be emitted in multiple batches")
}

func TestWalkLimit(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	user, rootFile := buildWalkTree(t, client, 2, 10)

	nav := walkTestNavigator(t, client, user)

	emitted := 0
	err := nav.walk(context.Background(), []*File{rootFile}, 5, 10, func(files []*File, level int) error {
		emitted += len(files)
		return nil
	})
	require.ErrorIs(t, err, ErrFileCountLimitedReached)
	require.LessOrEqual(t, emitted, 5)
}

// TestDeleteFilesBatched walks a nested tree inside a tx and asserts the
// batched delete removes every row — including grandchildren of folders
// whose deletion is deferred until after the walk (HasParentWith relies
// on parent rows staying alive mid-walk).
func TestDeleteFilesBatched(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	group := client.Group.Create().SetName("deleters").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	user := client.User.Create().SetEmail("del@example.com").SetNick("del").SetGroup(group).SaveX(ctx)
	policy := client.StoragePolicy.Create().SetName("local").SetType("local").SaveX(ctx)
	root := client.File.Create().SetName(inventory.RootFolderName).SetType(int(types.FileTypeFolder)).SetOwner(user).SaveX(ctx)

	// dir -> sub -> leaf; dir also holds two files.
	dir := client.File.Create().SetName("dir").SetType(int(types.FileTypeFolder)).SetOwner(user).SetParent(root).SaveX(ctx)
	sub := client.File.Create().SetName("sub").SetType(int(types.FileTypeFolder)).SetOwner(user).SetParent(dir).SaveX(ctx)
	leaf := client.File.Create().SetName("leaf.txt").SetType(int(types.FileTypeFile)).SetOwner(user).SetParent(sub).SaveX(ctx)
	f1 := client.File.Create().SetName("a.txt").SetType(int(types.FileTypeFile)).SetOwner(user).SetParent(dir).SaveX(ctx)
	f2 := client.File.Create().SetName("b.txt").SetType(int(types.FileTypeFile)).SetOwner(user).SetParent(dir).SaveX(ctx)

	for _, fm := range []*ent.File{leaf, f1, f2} {
		client.Entity.Create().
			SetType(1).SetSource("src").SetSize(10).
			SetStoragePolicyEntities(policy.ID).
			AddFileIDs(fm.ID).
			SaveX(ctx)
	}

	fc := inventory.NewFileClient(client, conf.SQLiteDB, nil)
	uc := inventory.NewUserClient(client)
	txFc, tx, txCtx, err := inventory.WithTx(ctx, fc)
	require.NoError(t, err)
	txCtx = context.WithValue(txCtx, inventory.LoadFileEntity{}, true)

	// Targets: dir wrapped as *File with entities edge loaded.
	dirModel := client.File.Query().WithEntities().Where(entfile.IDEQ(dir.ID)).OnlyX(txCtx)
	dirFile := newFile(nil, dirModel)
	dirFile.OwnerModel = user

	user = client.User.Query().WithGroup().Where(entuser.IDEQ(user.ID)).OnlyX(txCtx)
	f := &DBFS{user: user}
	targets := map[Navigator][]*File{
		&myNavigator{baseNavigator: newBaseNavigator(txFc, defaultFilter, user, nil, nil), user: user, fileClient: txFc, userClient: uc}: {dirFile},
	}

	stale, diff, indexToDelete, err := f.deleteFiles(txCtx, targets, txFc, nil)
	require.NoError(t, err)
	require.NoError(t, inventory.Commit(tx))

	// dir, sub, leaf, f1, f2 all deleted; root survives.
	remaining := client.File.Query().AllX(ctx)
	require.Len(t, remaining, 1)
	require.Equal(t, root.ID, remaining[0].ID)
	require.ElementsMatch(t, []int{dir.ID, sub.ID, leaf.ID, f1.ID, f2.ID}, indexToDelete)
	require.Len(t, stale, 3)
	require.Equal(t, int64(-30), diff[user.ID])
}

func TestWalkDepthLimit(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	user, rootFile := buildWalkTree(t, client, 2, 4)

	nav := walkTestNavigator(t, client, user)

	maxLevel := -1
	err := nav.walk(context.Background(), []*File{rootFile}, 1000, 1, func(files []*File, level int) error {
		if level > maxLevel {
			maxLevel = level
		}
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, maxLevel)
}
