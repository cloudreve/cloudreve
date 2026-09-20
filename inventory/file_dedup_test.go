package inventory

import (
	"context"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/ent/storagepolicy"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"
)

func dedupFixture(t *testing.T, client *ent.Client, suffix string) (*ent.User, *ent.File, *ent.StoragePolicy) {
	ctx := context.Background()
	group := client.Group.Create().SetName("g" + suffix).SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	u := client.User.Create().SetEmail("u" + suffix + "@example.com").SetNick("u" + suffix).SetGroup(group).SaveX(ctx)
	root := client.File.Create().SetName(RootFolderName).SetType(int(types.FileTypeFolder)).SetOwner(u).SaveX(ctx)
	p := client.StoragePolicy.Create().SetName("p" + suffix).SetType("local").
		SetStatus(storagepolicy.StatusActive).SaveX(ctx)
	return u, root, p
}

func mkHashedEntity(client *ent.Client, ctx context.Context, u *ent.User, p *ent.StoragePolicy, hash string, size int64) *ent.Entity {
	return client.Entity.Create().
		SetType(int(types.EntityTypeVersion)).
		SetSource("cloudreve/" + hash).
		SetSize(size).
		SetHash(hash).
		SetReferenceCount(1).
		SetCreatedBy(u.ID).
		SetStoragePolicyEntities(p.ID).
		SaveX(ctx)
}

func TestFindEntityByHash(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	fc := NewFileClient(client, conf.SQLite3DB, nil)

	u, _, p := dedupFixture(t, client, "a")
	other, _, _ := dedupFixture(t, client, "b")

	const hash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	e := mkHashedEntity(client, ctx, u, p, hash, 100)

	// Owner scope: match
	found, err := fc.FindEntityByHash(ctx, hash, 100, u.ID, false)
	require.NoError(t, err)
	require.Equal(t, e.ID, found.ID)

	// Owner scope: other user's entity must not leak
	_, err = fc.FindEntityByHash(ctx, hash, 100, other.ID, false)
	require.True(t, ent.IsNotFound(err))

	// Global scope: cross-user match allowed
	found, err = fc.FindEntityByHash(ctx, hash, 100, other.ID, true)
	require.NoError(t, err)
	require.Equal(t, e.ID, found.ID)

	// Size mismatch: no match
	_, err = fc.FindEntityByHash(ctx, hash, 101, u.ID, false)
	require.True(t, ent.IsNotFound(err))

	// Empty hash: no match
	_, err = fc.FindEntityByHash(ctx, "", 100, u.ID, false)
	require.True(t, ent.IsNotFound(err))

	// Unfinished upload entity (session still attached): excluded
	unfinished := client.Entity.Create().
		SetType(int(types.EntityTypeVersion)).
		SetSource("cloudreve/unfinished").
		SetSize(50).
		SetHash("aaaa").
		SetReferenceCount(0).
		SetCreatedBy(u.ID).
		SetUploadSessionID(uuid.Must(uuid.NewV4())).
		SetStoragePolicyEntities(p.ID).
		SaveX(ctx)
	_, err = fc.FindEntityByHash(ctx, "aaaa", 50, u.ID, false)
	require.True(t, ent.IsNotFound(err))
	_ = unfinished

	// Stale entity (refcount 0, session cleared): excluded
	client.Entity.Create().
		SetType(int(types.EntityTypeVersion)).
		SetSource("cloudreve/stale").
		SetSize(60).
		SetHash("bbbb").
		SetReferenceCount(0).
		SetCreatedBy(u.ID).
		SetStoragePolicyEntities(p.ID).
		SaveX(ctx)
	_, err = fc.FindEntityByHash(ctx, "bbbb", 60, u.ID, false)
	require.True(t, ent.IsNotFound(err))
}

func TestCreateFileLinkedEntity(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	fc := NewFileClient(client, conf.SQLite3DB, nil)

	u, root, p := dedupFixture(t, client, "l")

	const hash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	existing := mkHashedEntity(client, ctx, u, p, hash, 512)

	newFile, linked, diff, err := fc.CreateFile(ctx, root, &CreateFileParameters{
		FileType:        types.FileTypeFile,
		Name:            "copy.txt",
		StoragePolicyID: p.ID,
		EntityParameters: &EntityParameters{
			LinkedEntityID: existing.ID,
		},
	})
	require.NoError(t, err)
	require.Equal(t, existing.ID, linked.ID)

	// Refcount incremented
	reloaded := client.Entity.GetX(ctx, existing.ID)
	require.Equal(t, 2, reloaded.ReferenceCount)

	// File wired to the shared entity (reload: primary/size are set via
	// a follow-up update after the initial insert)
	reloadedFile := client.File.GetX(ctx, newFile.ID)
	require.Equal(t, int64(512), reloadedFile.Size)
	require.Equal(t, existing.ID, reloadedFile.PrimaryEntity)
	require.Equal(t, u.ID, reloadedFile.OwnerID)

	// Logical quota charged to the new file's owner
	require.Equal(t, int64(512), diff[u.ID])

	// Entity->file edge attached
	require.Len(t, reloaded.Edges.File, 0) // edges not eager-loaded
	files := reloaded.QueryFile().AllX(ctx)
	require.Len(t, files, 1)
	require.Equal(t, newFile.ID, files[0].ID)
}
