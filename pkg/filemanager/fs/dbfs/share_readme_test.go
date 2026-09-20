package dbfs

import (
	"context"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

// readmeShareFixture seeds an owner with a folder containing a README.md and
// a regular file, shared via a folder share with the given props.
func readmeShareFixture(t *testing.T, client *ent.Client, hasher hashid.Encoder, props *types.ShareProps) (*ent.User, *ent.Share) {
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
	dir := client.File.Create().SetName("docs").SetType(int(types.FileTypeFolder)).
		SetOwner(owner).SetParent(ownerRoot).SaveX(ctx)
	client.File.Create().SetName("README.md").SetType(int(types.FileTypeFile)).
		SetOwner(owner).SetParent(dir).SaveX(ctx)
	client.File.Create().SetName("readme.txt").SetType(int(types.FileTypeFile)).
		SetOwner(owner).SetParent(dir).SaveX(ctx)
	client.File.Create().SetName("notes.txt").SetType(int(types.FileTypeFile)).
		SetOwner(owner).SetParent(dir).SaveX(ctx)

	share := client.Share.Create().SetUser(owner).SetFile(dir).SetProps(props).SaveX(ctx)

	visitor := client.User.Create().SetEmail("visitor@example.com").SetNick("v").SetGroup(group).SaveX(ctx)
	visitor.SetGroup(group)
	return visitor, share
}

func TestShareHideReadMe(t *testing.T) {
	newNav := func(t *testing.T, props *types.ShareProps) (Navigator, *fs.URI, *ent.Client) {
		client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
		t.Cleanup(func() { require.NoError(t, client.Close()) })
		hasher, err := hashid.New("seed-test-salt")
		require.NoError(t, err)

		visitor, share := readmeShareFixture(t, client, hasher, props)
		nav := multiShareNavigator(t, client, hasher, visitor)
		uri, err := fs.NewUriFromString(fs.NewShareUri(hashid.EncodeShareID(hasher, share.ID), ""))
		require.NoError(t, err)
		return nav, uri, client
	}

	names := func(t *testing.T, nav Navigator, uri *fs.URI) []string {
		t.Helper()
		root, err := nav.To(context.Background(), uri)
		require.NoError(t, err)
		res, err := nav.Children(context.Background(), root, &ListArgs{Page: &inventory.PaginationArgs{PageSize: 100}})
		require.NoError(t, err)
		return lo.Map(res.Files, func(f *File, _ int) string { return f.Name() })
	}

	t.Run("hidden when both props set", func(t *testing.T) {
		nav, uri, _ := newNav(t, &types.ShareProps{ShowReadMe: true, HideReadMe: true})
		require.ElementsMatch(t, []string{"notes.txt"}, names(t, nav, uri))

		// Hidden files remain resolvable by path for the readme viewer.
		f, err := nav.To(context.Background(), uri.Join("README.md"))
		require.NoError(t, err)
		require.Equal(t, "README.md", f.Name())
	})

	t.Run("listed without hide prop", func(t *testing.T) {
		nav, uri, _ := newNav(t, &types.ShareProps{ShowReadMe: true})
		require.ElementsMatch(t, []string{"README.md", "readme.txt", "notes.txt"}, names(t, nav, uri))
	})

	t.Run("listed without readme props", func(t *testing.T) {
		nav, uri, _ := newNav(t, &types.ShareProps{})
		require.ElementsMatch(t, []string{"README.md", "readme.txt", "notes.txt"}, names(t, nav, uri))
	})
}
