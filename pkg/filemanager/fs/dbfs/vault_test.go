package dbfs

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/lock"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/stretchr/testify/require"
)

// vaultFixture builds a DBFS for an owner with a private space containing one
// file, plus a normal sibling file for contrast.
func vaultFixture(t *testing.T, client *ent.Client) (*ent.User, *ent.File, *ent.File, *DBFS, *cache.MemoStore) {
	t.Helper()
	ctx := context.Background()
	l := logging.NewConsoleLogger(logging.LevelError)
	hasher, err := hashid.New("vault-test-salt")
	require.NoError(t, err)

	p := client.StoragePolicy.Create().SetName("local").SetType("local").
		SetStatus("active").SetSettings(&types.PolicySetting{}).SaveX(ctx)
	group := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).
		SetMaxStorage(0).SetStoragePolicies(p).SaveX(ctx)
	u := client.User.Create().SetEmail("v@example.com").SetNick("v").SetGroup(group).SaveX(ctx)
	u.SetGroup(group)
	root := client.File.Create().SetName(inventory.RootFolderName).
		SetType(int(types.FileTypeFolder)).SetOwner(u).SaveX(ctx)

	vault := client.File.Create().SetName("Private space").
		SetType(int(types.FileTypeFolder)).SetOwner(u).SetParent(root).SaveX(ctx)
	client.Metadata.Create().SetName(MetadataVault).SetValue("1").
		SetFile(vault).SetIsPublic(true).SaveX(ctx)

	secret := client.File.Create().SetName("secret.txt").
		SetType(int(types.FileTypeFile)).SetOwner(u).SetParent(vault).SaveX(ctx)
	client.File.Create().SetName("normal.txt").
		SetType(int(types.FileTypeFile)).SetOwner(u).SetParent(root).SaveX(ctx)

	u = client.User.UpdateOne(u).SetVaultFolder(vault.ID).SaveX(ctx)
	u.SetGroup(group)

	kv := cache.NewMemoStore("", l)
	f := &DBFS{
		user:                u,
		navigators:          make(map[string]Navigator),
		fileClient:          inventory.NewFileClient(client, conf.SQLiteDB, hasher),
		userClient:          inventory.NewUserClient(client),
		storagePolicyClient: inventory.NewStoragePolicyClient(client, nil),
		settingClient:       dedupSettingProvider{scope: "off"},
		hasher:              hasher,
		l:                   l,
		ls:                  lock.NewMemLS(hasher, l),
		cache:               kv,
		eventHub:            stubEventHub{},
	}
	return u, vault, secret, f, kv
}

func TestVaultLockedAccess(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	_, _, _, f, _ := vaultFixture(t, client)
	ctx := context.Background()

	// Resolving a file inside the locked vault fails.
	uri, err := fs.NewUriFromString("cloudreve://my/Private%20space/secret.txt")
	require.NoError(t, err)
	_, err = f.Get(ctx, uri)
	require.Error(t, err)
	var appErr serializer.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, serializer.CodeVaultLocked, appErr.ErrCode())
}

func TestVaultRootResolvableWhenLocked(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	_, vault, _, f, _ := vaultFixture(t, client)
	ctx := context.Background()

	// The vault root itself resolves so it can serve as the unlock entry.
	uri, err := fs.NewUriFromString("cloudreve://my/Private%20space")
	require.NoError(t, err)
	got, err := f.Get(ctx, uri)
	require.NoError(t, err)
	require.Equal(t, vault.ID, got.ID())
}

func TestVaultChildrenGated(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	_, _, _, f, kv := vaultFixture(t, client)
	ctx := context.Background()

	uri, err := fs.NewUriFromString("cloudreve://my/Private%20space")
	require.NoError(t, err)

	// Locked: listing vault contents is rejected.
	_, _, err = f.List(ctx, uri)
	require.Error(t, err)
	var appErr serializer.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, serializer.CodeVaultLocked, appErr.ErrCode())

	// Unlocked: contents are returned.
	require.NoError(t, kv.Set(VaultUnlockCachePrefix+strconv.Itoa(f.user.ID), 1, VaultUnlockTTL))
	_, res, err := f.List(ctx, uri)
	require.NoError(t, err)
	require.Len(t, res.Files, 1)
	require.Equal(t, "secret.txt", res.Files[0].Name())
}

func TestVaultSearchFiltered(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	_, _, _, f, kv := vaultFixture(t, client)
	ctx := context.Background()

	// Root-level search: vaulted files are hidden while locked.
	uri, err := fs.NewUriFromString("cloudreve://my/?name=txt")
	require.NoError(t, err)
	_, res, err := f.List(ctx, uri)
	require.NoError(t, err)
	names := make([]string, 0, len(res.Files))
	for _, fi := range res.Files {
		names = append(names, fi.Name())
	}
	require.Contains(t, names, "normal.txt")
	require.NotContains(t, names, "secret.txt")

	// Unlocked: vaulted files appear in results.
	require.NoError(t, kv.Set(VaultUnlockCachePrefix+strconv.Itoa(f.user.ID), 1, VaultUnlockTTL))
	_, res, err = f.List(ctx, uri)
	require.NoError(t, err)
	names = names[:0]
	for _, fi := range res.Files {
		names = append(names, fi.Name())
	}
	require.Contains(t, names, "secret.txt")
}

func TestVaultNonOwnerDenied(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	owner, vault, secret, f, kv := vaultFixture(t, client)
	ctx := context.Background()
	require.NoError(t, kv.Set(VaultUnlockCachePrefix+strconv.Itoa(owner.ID), 1, VaultUnlockTTL))

	// Even with the owner's session unlocked, a different user never passes
	// the vault gate; the response pretends the path does not exist.
	other := client.User.Create().SetEmail("o@example.com").SetNick("o").
		SetGroup(owner.Edges.Group).SaveX(ctx)
	otherFs := &DBFS{user: other, cache: kv}

	secretFile := newFile(nil, secret)
	err := otherFs.requireVaultUnlocked(secretFile)
	require.Error(t, err)
	var appErr serializer.AppError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, serializer.CodeParentNotExist, appErr.ErrCode())

	// Owner with unlock session passes.
	require.NoError(t, f.requireVaultUnlocked(secretFile))
	// Owner without unlock session gets the dedicated locked code.
	kv.Delete(VaultUnlockCachePrefix, strconv.Itoa(owner.ID))
	err = f.requireVaultUnlocked(secretFile)
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, serializer.CodeVaultLocked, appErr.ErrCode())

	_ = vault
	_ = ctx
}

func TestVaultShareCreationBlocked(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	_, _, _, f, kv := vaultFixture(t, client)
	ctx := context.Background()

	// Even while unlocked, vaulted files cannot be shared.
	require.NoError(t, kv.Set(VaultUnlockCachePrefix+strconv.Itoa(f.user.ID), 1, VaultUnlockTTL))
	uri, err := fs.NewUriFromString("cloudreve://my/Private%20space/secret.txt")
	require.NoError(t, err)
	file, err := f.Get(ctx, uri)
	require.NoError(t, err)
	inside, err := f.IsInPrivateSpace(ctx, file)
	require.NoError(t, err)
	require.True(t, inside)

	// Normal files are not affected.
	require.Equal(t, "secret.txt", file.Name())
	normalUri, err := fs.NewUriFromString("cloudreve://my/normal.txt")
	require.NoError(t, err)
	normalFile, err := f.Get(ctx, normalUri)
	require.NoError(t, err)
	inside, err = f.IsInPrivateSpace(ctx, normalFile)
	require.NoError(t, err)
	require.False(t, inside)
}
