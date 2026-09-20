package explorer

import (
	"context"
	"testing"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/ent/storagepolicy"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/stretchr/testify/require"
)

type shareShortcutFixture struct {
	client *ent.Client
	shares inventory.ShareClient
	hasher hashid.Encoder
	owner  *ent.User
	group  *ent.Group
	root   *ent.File
}

func newShareShortcutFixture(t *testing.T) *shareShortcutFixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	hasher, err := hashid.New("share-shortcut-test-salt")
	require.NoError(t, err)

	policy := client.StoragePolicy.Create().SetName("local").SetType("local").
		SetStatus(storagepolicy.StatusActive).SaveX(ctx)
	permissions := &boolset.BooleanSet{}
	boolset.Set(types.GroupPermissionShare, true, permissions)
	group := client.Group.Create().SetName("seed").SetPermissions(permissions).
		SetStoragePolicies(policy).SaveX(ctx)

	owner := client.User.Create().SetEmail("owner@example.com").SetNick("o").SetGroup(group).SaveX(ctx)
	root := client.File.Create().SetName(inventory.RootFolderName).SetType(int(types.FileTypeFolder)).SetOwner(owner).SaveX(ctx)

	return &shareShortcutFixture{
		client: client,
		shares: inventory.NewShareClient(client, conf.SQLiteDB, hasher),
		hasher: hasher,
		owner:  owner,
		group:  group,
		root:   root,
	}
}

func (f *shareShortcutFixture) anchor(name string, fileType types.FileType) *ent.File {
	return f.client.File.Create().SetName(name).SetType(int(fileType)).
		SetOwner(f.owner).SetParent(f.root).SaveX(context.Background())
}

func (f *shareShortcutFixture) load(t *testing.T, shareID int) *ent.Share {
	ctx := context.WithValue(context.Background(), inventory.LoadShareUser{}, true)
	ctx = context.WithValue(ctx, inventory.LoadShareFile{}, true)
	ctx = context.WithValue(ctx, inventory.LoadShareFiles{}, true)
	share, err := f.shares.GetByID(ctx, shareID)
	require.NoError(t, err)
	return share
}

func TestShareShortcutEntry(t *testing.T) {
	fx := newShareShortcutFixture(t)
	ctx := context.Background()

	t.Run("folder share resolves to symbolic folder", func(t *testing.T) {
		anchor := fx.anchor("shared_dir", types.FileTypeFolder)
		share := fx.client.Share.Create().SetUser(fx.owner).SetFile(anchor).SaveX(ctx)
		loaded := fx.load(t, share.ID)

		fileType, metadata, err := shareShortcutEntry(loaded, fx.hasher, "")
		require.NoError(t, err)
		require.Equal(t, types.FileTypeFolder, fileType)
		require.Equal(t,
			fs.NewShareUri(hashid.EncodeShareID(fx.hasher, share.ID), ""),
			metadata[dbfs.MetadataSharedRedirect])
		require.Equal(t, hashid.EncodeUserID(fx.hasher, fx.owner.ID), metadata[dbfs.MetadataSharedOwner])
	})

	t.Run("file share resolves to symbolic file", func(t *testing.T) {
		anchor := fx.anchor("shared.txt", types.FileTypeFile)
		share := fx.client.Share.Create().SetUser(fx.owner).SetFile(anchor).SaveX(ctx)
		loaded := fx.load(t, share.ID)

		fileType, metadata, err := shareShortcutEntry(loaded, fx.hasher, "")
		require.NoError(t, err)
		require.Equal(t, types.FileTypeFile, fileType)
		require.NotEmpty(t, metadata[dbfs.MetadataSharedRedirect])
	})

	t.Run("multi-file share resolves to symbolic folder", func(t *testing.T) {
		anchorA := fx.anchor("a.txt", types.FileTypeFile)
		anchorB := fx.anchor("b.txt", types.FileTypeFile)
		share := fx.client.Share.Create().SetUser(fx.owner).AddFiles(anchorA, anchorB).SaveX(ctx)
		loaded := fx.load(t, share.ID)

		fileType, _, err := shareShortcutEntry(loaded, fx.hasher, "")
		require.NoError(t, err)
		require.Equal(t, types.FileTypeFolder, fileType)
	})

	t.Run("password is embedded into the redirect uri", func(t *testing.T) {
		anchor := fx.anchor("protected_dir", types.FileTypeFolder)
		share := fx.client.Share.Create().SetUser(fx.owner).SetFile(anchor).SetPassword("secret").SaveX(ctx)
		loaded := fx.load(t, share.ID)

		_, metadata, err := shareShortcutEntry(loaded, fx.hasher, "secret")
		require.NoError(t, err)
		require.Equal(t,
			fs.NewShareUri(hashid.EncodeShareID(fx.hasher, share.ID), "secret"),
			metadata[dbfs.MetadataSharedRedirect])
	})

	t.Run("expired share is rejected", func(t *testing.T) {
		anchor := fx.anchor("expired_dir", types.FileTypeFolder)
		past := time.Now().Add(-time.Hour)
		share := fx.client.Share.Create().SetUser(fx.owner).SetFile(anchor).SetExpires(past).SaveX(ctx)
		loaded := fx.load(t, share.ID)

		_, _, err := shareShortcutEntry(loaded, fx.hasher, "")
		require.ErrorIs(t, err, inventory.ErrShareLinkExpired)
	})

	t.Run("inactive owner is rejected", func(t *testing.T) {
		inactive := fx.client.User.Create().SetEmail("inactive@example.com").SetNick("i").
			SetGroup(fx.group).SetStatus(user.StatusInactive).SaveX(ctx)
		inactiveRoot := fx.client.File.Create().SetName(inventory.RootFolderName).
			SetType(int(types.FileTypeFolder)).SetOwner(inactive).SaveX(ctx)
		anchor := fx.client.File.Create().SetName("inactive_dir").SetType(int(types.FileTypeFolder)).
			SetOwner(inactive).SetParent(inactiveRoot).SaveX(ctx)
		share := fx.client.Share.Create().SetUser(inactive).SetFile(anchor).SaveX(ctx)
		loaded := fx.load(t, share.ID)

		_, _, err := shareShortcutEntry(loaded, fx.hasher, "")
		require.ErrorIs(t, err, inventory.ErrOwnerInactive)
	})
}
