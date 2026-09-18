package dbfs

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/stretchr/testify/require"
)

// conflictTestFixture creates user -> root -> dst folder, optionally with
// dstChildren file rows inside dst. Returns the user, the dst folder
// wrapped as *File and a real FileClient backed by the enttest client.
func conflictTestFixture(t *testing.T, client *ent.Client, dstChildren ...string) (*ent.User, *ent.File, *File, inventory.FileClient) {
	t.Helper()
	ctx := context.Background()
	group := client.Group.Create().SetName("conflict").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	user := client.User.Create().SetEmail("c@example.com").SetNick("c").SetGroup(group).SaveX(ctx)
	root := client.File.Create().SetName(inventory.RootFolderName).SetType(int(types.FileTypeFolder)).SetOwner(user).SaveX(ctx)
	dst := client.File.Create().SetName("dst").SetType(int(types.FileTypeFolder)).SetOwner(user).SetParent(root).SaveX(ctx)
	for _, name := range dstChildren {
		client.File.Create().SetName(name).SetType(int(types.FileTypeFile)).SetOwner(user).SetParent(dst).SaveX(ctx)
	}

	dstUri, err := fs.NewUriFromString(fmt.Sprintf("%s/dst", fs.NewMyUri("")))
	require.NoError(t, err)
	dstFile := newFile(nil, dst)
	dstFile.OwnerModel = user
	dstFile.Path[0] = dstUri

	return user, root, dstFile, inventory.NewFileClient(client, conf.SQLiteDB, nil)
}

func conflictTestTarget(t *testing.T, model *ent.File, user *ent.User) *File {
	t.Helper()
	u, err := fs.NewUriFromString(fmt.Sprintf("%s/%s", fs.NewMyUri(""), model.Name))
	require.NoError(t, err)
	f := newFile(nil, model)
	f.OwnerModel = user
	f.Path[0] = u
	return f
}

func TestResolveMoveConflictsSkipsColliding(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	user, root, dstFile, fc := conflictTestFixture(t, client, "a.txt")

	srcA := client.File.Create().SetName("a.txt").SetType(int(types.FileTypeFile)).SetOwner(user).SetParent(root).SaveX(ctx)
	srcB := client.File.Create().SetName("b.txt").SetType(int(types.FileTypeFile)).SetOwner(user).SetParent(root).SaveX(ctx)
	targetA := conflictTestTarget(t, srcA, user)
	targetB := conflictTestTarget(t, srcB, user)
	targets := []*File{targetA, targetB}

	f := &DBFS{fileClient: fc}
	nav := Navigator(&myNavigator{})
	navOf := map[*File]Navigator{targetA: nav, targetB: nav}
	ae := serializer.NewAggregateError()

	surviving, group := f.resolveMoveConflicts(ctx, targets, navOf, dstFile, true, MoveConflictSkip, ae)

	require.Equal(t, []*File{targetB}, surviving)
	require.Equal(t, []*File{targetB}, group[nav])
	require.NotNil(t, ae.Aggregate())
}

func TestResolveMoveConflictsKeepsNonColliding(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	user, root, dstFile, fc := conflictTestFixture(t, client, "other.txt")

	srcA := client.File.Create().SetName("a.txt").SetType(int(types.FileTypeFile)).SetOwner(user).SetParent(root).SaveX(ctx)
	targetA := conflictTestTarget(t, srcA, user)

	f := &DBFS{fileClient: fc}
	ae := serializer.NewAggregateError()

	surviving, group := f.resolveMoveConflicts(ctx, []*File{targetA}, map[*File]Navigator{targetA: &myNavigator{}}, dstFile, true, MoveConflictSkip, ae)

	require.Equal(t, []*File{targetA}, surviving)
	require.Len(t, group, 1)
	require.Nil(t, ae.Aggregate())
}

func TestResolveMoveConflictsUsesRestoreDisplayName(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	// Destination holds a file under the restored display name, not the
	// trash-internal UUID name.
	user, root, dstFile, fc := conflictTestFixture(t, client, "original.txt")

	trashed := client.File.Create().SetName("uuid-name").SetType(int(types.FileTypeFile)).SetOwner(user).SetParent(root).SaveX(ctx)
	trashed.Edges.Metadata = []*ent.Metadata{
		{Name: MetadataRestoreUri, Value: fmt.Sprintf("%s/original.txt", fs.NewMyUri(""))},
	}
	target := conflictTestTarget(t, trashed, user)

	f := &DBFS{fileClient: fc}
	ae := serializer.NewAggregateError()

	surviving, _ := f.resolveMoveConflicts(ctx, []*File{target}, map[*File]Navigator{}, dstFile, false, MoveConflictSkip, ae)

	require.Empty(t, surviving)
	require.NotNil(t, ae.Aggregate())
}
