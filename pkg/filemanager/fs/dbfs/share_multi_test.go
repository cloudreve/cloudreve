package dbfs

import (
	"context"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	entfile "github.com/cloudreve/Cloudreve/v4/ent/file"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

// multiShareFixture seeds an owner with three shareable files spread across
// different folders and one multi-file share covering all of them. The
// visitor has share-download permission like a normal logged-in link opener.
func multiShareFixture(t *testing.T, client *ent.Client, hasher hashid.Encoder) (*ent.User, *ent.Share) {
	t.Helper()
	ctx := context.Background()

	policy := client.StoragePolicy.Create().SetName("local").SetType("local").SaveX(ctx)
	permissions := &boolset.BooleanSet{}
	boolset.Sets(map[types.GroupPermission]bool{
		types.GroupPermissionShare:         true,
		types.GroupPermissionShareDownload: true,
	}, permissions)
	group := client.Group.Create().SetName("g").SetPermissions(permissions).
		SetStoragePolicies(policy).SaveX(ctx)

	owner := client.User.Create().SetEmail("owner@example.com").SetNick("o").SetGroup(group).SaveX(ctx)
	ownerRoot := client.File.Create().SetName(inventory.RootFolderName).
		SetType(int(types.FileTypeFolder)).SetOwner(owner).SaveX(ctx)
	fileA := client.File.Create().SetName("a.txt").SetType(int(types.FileTypeFile)).
		SetOwner(owner).SetParent(ownerRoot).SaveX(ctx)
	dirB := client.File.Create().SetName("b_dir").SetType(int(types.FileTypeFolder)).
		SetOwner(owner).SetParent(ownerRoot).SaveX(ctx)
	client.File.Create().SetName("inner.txt").SetType(int(types.FileTypeFile)).
		SetOwner(owner).SetParent(dirB).SaveX(ctx)
	sub := client.File.Create().SetName("sub").SetType(int(types.FileTypeFolder)).
		SetOwner(owner).SetParent(ownerRoot).SaveX(ctx)
	fileC := client.File.Create().SetName("c.txt").SetType(int(types.FileTypeFile)).
		SetOwner(owner).SetParent(sub).SaveX(ctx)

	share := client.Share.Create().SetUser(owner).SetFile(fileA).
		AddFiles(fileA, dirB, fileC).SaveX(ctx)

	visitor := client.User.Create().SetEmail("visitor@example.com").SetNick("v").SetGroup(group).SaveX(ctx)
	visitor.SetGroup(group)
	return visitor, share
}

func multiShareNavigator(t *testing.T, client *ent.Client, hasher hashid.Encoder, u *ent.User) Navigator {
	t.Helper()
	return NewShareNavigator(
		u,
		inventory.NewFileClient(client, conf.SQLiteDB, hasher),
		inventory.NewShareClient(client, conf.SQLiteDB, hasher),
		nil,
		nil,
		logging.NewConsoleLogger(logging.LevelError),
		&setting.DBFS{},
		hasher,
	)
}

func TestMultiFileShareNavigator(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	hasher, err := hashid.New("seed-test-salt")
	require.NoError(t, err)

	visitor, share := multiShareFixture(t, client, hasher)
	nav := multiShareNavigator(t, client, hasher, visitor)

	uri, err := fs.NewUriFromString(fs.NewShareUri(hashid.EncodeShareID(hasher, share.ID), ""))
	require.NoError(t, err)

	root, err := nav.To(ctx, uri)
	require.NoError(t, err)
	require.Equal(t, types.FileTypeFolder, root.Type())

	// Root lists the union of linked files from different folders.
	res, err := nav.Children(ctx, root, &ListArgs{Page: &inventory.PaginationArgs{PageSize: 100}})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"a.txt", "b_dir", "c.txt"},
		lo.Map(res.Files, func(f *File, _ int) string { return f.Name() }))

	// Paths resolve through the union root into shared folders.
	target, err := nav.To(ctx, uri.Join("b_dir", "inner.txt"))
	require.NoError(t, err)
	require.Equal(t, "inner.txt", target.Name())
	require.Equal(t, types.FileTypeFile, target.Type())

	// Real filesystem URIs resolve to the owner's actual path.
	real := target.Uri(true)
	require.NotNil(t, real)
	require.Contains(t, real.String(), "b_dir")

	// Share URIs stay inside the share filesystem.
	require.Contains(t, target.Uri(false).String(), "share")

	// Unknown names and files outside the linked set are not resolvable.
	_, err = nav.To(ctx, uri.Join("nope.txt"))
	require.Error(t, err)
	_, err = nav.To(ctx, uri.Join("sub", "x.txt"))
	require.Error(t, err)
}

func TestMultiFileShareDropsDeadEntries(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	hasher, err := hashid.New("seed-test-salt")
	require.NoError(t, err)

	visitor, share := multiShareFixture(t, client, hasher)

	// Deleting one linked file removes it from the listing without
	// invalidating the whole share.
	dead := share.QueryFiles().Where(entfile.Name("c.txt")).OnlyX(ctx)
	client.File.DeleteOne(dead).ExecX(ctx)

	nav := multiShareNavigator(t, client, hasher, visitor)
	uri, err := fs.NewUriFromString(fs.NewShareUri(hashid.EncodeShareID(hasher, share.ID), ""))
	require.NoError(t, err)

	root, err := nav.To(ctx, uri)
	require.NoError(t, err)

	res, err := nav.Children(ctx, root, &ListArgs{Page: &inventory.PaginationArgs{PageSize: 100}})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"a.txt", "b_dir"},
		lo.Map(res.Files, func(f *File, _ int) string { return f.Name() }))

	_, err = nav.To(ctx, uri.Join("c.txt"))
	require.Error(t, err)
}
