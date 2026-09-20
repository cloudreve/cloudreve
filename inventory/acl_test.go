package inventory

import (
	"context"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/aclentry"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/stretchr/testify/require"
)

func aclFixture(t *testing.T, client *ent.Client) (*ent.User, *ent.Group, *ent.File) {
	t.Helper()
	ctx := context.Background()
	group := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	user := client.User.Create().SetEmail("u@example.com").SetNick("u").SetGroup(group).SaveX(ctx)
	file := client.File.Create().SetName("dir").SetType(int(types.FileTypeFolder)).SetOwner(user).SaveX(ctx)
	user.SetGroup(group)
	return user, group, file
}

func aclEntry(t *testing.T, client *ent.Client, fileID int, st aclentry.SubjectType, sid int, perms ...types.AclPermission) *ent.AclEntry {
	t.Helper()
	bs := &boolset.BooleanSet{}
	for _, p := range perms {
		boolset.Set(int(p), true, bs)
	}
	return client.AclEntry.Create().
		SetFileID(fileID).
		SetSubjectType(st).
		SetSubjectID(sid).
		SetPermissions(bs).
		SaveX(context.Background())
}

func TestAclUpsertAndList(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	_, _, file := aclFixture(t, client)
	c := NewAclClient(client, conf.SQLiteDB)

	bs := &boolset.BooleanSet{}
	boolset.Set(int(types.AclPermRead), true, bs)
	e, err := c.Upsert(ctx, &UpsertAclEntryParams{
		FileID: file.ID, SubjectType: aclentry.SubjectTypeUser, SubjectID: 7, Permissions: bs,
	})
	require.NoError(t, err)
	require.NotZero(t, e.ID)

	// Second upsert on same (file, subject) updates rather than duplicating.
	boolset.Set(int(types.AclPermDelete), true, bs)
	e2, err := c.Upsert(ctx, &UpsertAclEntryParams{
		FileID: file.ID, SubjectType: aclentry.SubjectTypeUser, SubjectID: 7, Permissions: bs,
	})
	require.NoError(t, err)
	require.Equal(t, e.ID, e2.ID)

	entries, err := c.List(ctx, file.ID)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.True(t, entries[0].Permissions.Enabled(int(types.AclPermDelete)))

	require.NoError(t, c.Delete(ctx, file.ID, e.ID))
	entries, err = c.List(ctx, file.ID)
	require.NoError(t, err)
	require.Empty(t, entries)

	// Delete of a missing row reports not found.
	require.Error(t, c.Delete(ctx, file.ID, e.ID))
}

func TestAclEffectivePermissionsAnonymous(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	_, _, file := aclFixture(t, client)
	c := NewAclClient(client, conf.SQLiteDB)
	anon := &ent.User{ID: 0}

	// No rows at all -> nil (fall back to share defaults).
	res, err := c.EffectivePermissions(ctx, file.ID, anon)
	require.NoError(t, err)
	require.Nil(t, res)

	// An anonymous row applies to anonymous visitors; other tiers do not.
	aclEntry(t, client, file.ID, aclentry.SubjectTypeEveryone, 0, types.AclPermRead)
	res, err = c.EffectivePermissions(ctx, file.ID, anon)
	require.NoError(t, err)
	require.Nil(t, res)

	aclEntry(t, client, file.ID, aclentry.SubjectTypeAnonymous, 0, types.AclPermCreate)
	res, err = c.EffectivePermissions(ctx, file.ID, anon)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.True(t, res.Enabled(int(types.AclPermCreate)))
	require.False(t, res.Enabled(int(types.AclPermRead)))
}

func TestAclEffectivePermissionsUnion(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	user, group, file := aclFixture(t, client)
	c := NewAclClient(client, conf.SQLiteDB)

	aclEntry(t, client, file.ID, aclentry.SubjectTypeEveryone, 0, types.AclPermRead)
	aclEntry(t, client, file.ID, aclentry.SubjectTypeGroup, group.ID, types.AclPermCreate)
	aclEntry(t, client, file.ID, aclentry.SubjectTypeUser, user.ID, types.AclPermDelete)
	// Unrelated rows must not leak in.
	aclEntry(t, client, file.ID, aclentry.SubjectTypeUser, user.ID+99, types.AclPermUpdate)
	aclEntry(t, client, file.ID, aclentry.SubjectTypeGroup, group.ID+99, types.AclPermUpdate)

	res, err := c.EffectivePermissions(ctx, file.ID, user)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.True(t, res.Enabled(int(types.AclPermRead)))
	require.True(t, res.Enabled(int(types.AclPermCreate)))
	require.True(t, res.Enabled(int(types.AclPermDelete)))
	require.False(t, res.Enabled(int(types.AclPermUpdate)))
}

func TestAclEffectivePermissionsNoMatchIsNil(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	user, _, file := aclFixture(t, client)
	c := NewAclClient(client, conf.SQLiteDB)

	aclEntry(t, client, file.ID, aclentry.SubjectTypeUser, user.ID+99, types.AclPermRead)
	res, err := c.EffectivePermissions(ctx, file.ID, user)
	require.NoError(t, err)
	require.Nil(t, res)
}

func TestAclSharedFileIDs(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	user, group, file := aclFixture(t, client)
	c := NewAclClient(client, conf.SQLiteDB)

	other := client.File.Create().SetName("other").SetType(int(types.FileTypeFolder)).SetOwner(user).SaveX(ctx)
	writeOnly := client.File.Create().SetName("wo").SetType(int(types.FileTypeFolder)).SetOwner(user).SaveX(ctx)
	everyoneFile := client.File.Create().SetName("ev").SetType(int(types.FileTypeFolder)).SetOwner(user).SaveX(ctx)

	aclEntry(t, client, file.ID, aclentry.SubjectTypeUser, user.ID, types.AclPermRead)
	aclEntry(t, client, file.ID, aclentry.SubjectTypeGroup, group.ID, types.AclPermCreate)
	aclEntry(t, client, other.ID, aclentry.SubjectTypeGroup, group.ID, types.AclPermRead)
	aclEntry(t, client, writeOnly.ID, aclentry.SubjectTypeUser, user.ID, types.AclPermCreate)
	aclEntry(t, client, everyoneFile.ID, aclentry.SubjectTypeEveryone, 0, types.AclPermRead)

	res, err := c.SharedFileIDs(ctx, user)
	require.NoError(t, err)
	require.Len(t, res, 2)

	// Unioned perms across user + group rows.
	require.True(t, res[file.ID].Enabled(int(types.AclPermRead)))
	require.True(t, res[file.ID].Enabled(int(types.AclPermCreate)))
	require.True(t, res[other.ID].Enabled(int(types.AclPermRead)))

	// Anonymous gets no discovery listing.
	anonRes, err := c.SharedFileIDs(ctx, &ent.User{ID: 0})
	require.NoError(t, err)
	require.Empty(t, anonRes)
}

func TestAclSharedFileIDsViaMembership(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	user, _, file := aclFixture(t, client)
	c := NewAclClient(client, conf.SQLiteDB)

	// Grant targets a group the user holds via membership, not primary.
	team := client.Group.Create().SetName("team").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	m := client.GroupMembership.Create().SetUserID(user.ID).SetGroupID(team.ID).SaveX(ctx)
	m.Edges.Group = team
	user.Edges.Memberships = []*ent.GroupMembership{m}

	aclEntry(t, client, file.ID, aclentry.SubjectTypeGroup, team.ID, types.AclPermRead)

	res, err := c.SharedFileIDs(ctx, user)
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.True(t, res[file.ID].Enabled(int(types.AclPermRead)))
}

func TestAclEffectivePermissionsExplicitEmptyRevokes(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	user, _, file := aclFixture(t, client)
	c := NewAclClient(client, conf.SQLiteDB)

	// An everyone row with zero bits = explicit revocation for all
	// authenticated visitors; distinct from nil (no match).
	aclEntry(t, client, file.ID, aclentry.SubjectTypeEveryone, 0)
	res, err := c.EffectivePermissions(ctx, file.ID, user)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.False(t, res.Enabled(int(types.AclPermRead)))
}
