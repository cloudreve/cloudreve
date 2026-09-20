package inventory

import (
	"context"
	"testing"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/aclentry"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/ent/storagepolicy"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/stretchr/testify/require"
)

func permSet(bits ...types.GroupPermission) *boolset.BooleanSet {
	bs := &boolset.BooleanSet{}
	for _, b := range bits {
		boolset.Set(int(b), true, bs)
	}
	return bs
}

func memberOf(u *ent.User, g *ent.Group, expires *time.Time) *ent.GroupMembership {
	m := &ent.GroupMembership{UserID: u.ID, GroupID: g.ID, Expires: expires}
	m.Edges.Group = g
	return m
}

func TestGroupsOf(t *testing.T) {
	g1 := &ent.Group{ID: 1, Name: "primary"}
	g2 := &ent.Group{ID: 2, Name: "member"}
	g3 := &ent.Group{ID: 3, Name: "expired"}

	u := &ent.User{ID: 1}
	u.Edges.Group = g1

	require.Equal(t, []int{1}, GroupIDsOf(u))

	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)
	u.Edges.Memberships = []*ent.GroupMembership{
		memberOf(u, g2, nil),
		memberOf(u, g3, &past),   // expired — excluded
		memberOf(u, g2, &future), // duplicate group — deduped
		memberOf(u, g1, nil),     // primary duplicated — deduped
	}

	got := GroupIDsOf(u)
	require.Equal(t, []int{1, 2}, got)
}

func TestEffectiveGroupSinglePassthrough(t *testing.T) {
	g := &ent.Group{ID: 1, Name: "primary", MaxStorage: 1024}
	u := &ent.User{ID: 1}
	u.Edges.Group = g

	got := EffectiveGroup(u)
	require.Same(t, g, got)
}

func TestEffectiveGroupUnion(t *testing.T) {
	pa := &ent.StoragePolicy{ID: 10, Status: storagepolicy.StatusActive}
	pb := &ent.StoragePolicy{ID: 11, Status: storagepolicy.StatusActive}

	g1 := &ent.Group{
		ID: 1, Name: "primary",
		MaxStorage:      1024, // capped
		SpeedLimit:      0,    // unlimited
		Permissions:     permSet(types.GroupPermissionShare),
		StoragePolicyID: pa.ID,
		Settings: &types.GroupSetting{
			SourceBatchSize:  5,
			TrashRetention:   3600,
			AllowedNodes:     []int{1},
			LoginIPWhitelist: []string{"10.0.0.0/8"},
		},
	}
	g1.Edges.AllowedPolicies = []*ent.StoragePolicy{pa}
	g1.Edges.StoragePolicies = pa

	g2 := &ent.Group{
		ID: 2, Name: "member",
		MaxStorage:  0, // unlimited wins over 1024
		SpeedLimit:  512,
		Permissions: permSet(types.GroupPermissionWebDAV, types.GroupPermissionRemoteDownload),
		Settings: &types.GroupSetting{
			SourceBatchSize:  0, // disabled in g2 — g1's 5 wins
			TrashRetention:   7200,
			AllowedNodes:     []int{2},
			DefaultPinned:    []int{7},
			WeightedPolicies: true,
		},
	}
	g2.Edges.AllowedPolicies = []*ent.StoragePolicy{pb}

	u := &ent.User{ID: 1}
	u.Edges.Group = g1
	u.Edges.Memberships = []*ent.GroupMembership{memberOf(u, g2, nil)}

	got := EffectiveGroup(u)
	require.NotNil(t, got)
	require.Equal(t, 0, got.ID) // synthetic merged group

	// Permissions OR'ed.
	require.True(t, got.Permissions.Enabled(int(types.GroupPermissionShare)))
	require.True(t, got.Permissions.Enabled(int(types.GroupPermissionWebDAV)))
	require.True(t, got.Permissions.Enabled(int(types.GroupPermissionRemoteDownload)))
	require.False(t, got.Permissions.Enabled(int(types.GroupPermissionIsAdmin)))

	// Quotas: most permissive (0 = unlimited wins).
	require.Equal(t, int64(0), got.MaxStorage)
	require.Equal(t, 0, got.SpeedLimit)

	// Primary group's default policy is kept; allowed set unions.
	require.Equal(t, pa.ID, got.StoragePolicyID)
	require.ElementsMatch(t, []int{pa.ID, pb.ID},
		[]int{got.Edges.AllowedPolicies[0].ID, got.Edges.AllowedPolicies[1].ID})

	// Settings union.
	require.Equal(t, 5, got.Settings.SourceBatchSize)
	require.Equal(t, 7200, got.Settings.TrashRetention)
	require.ElementsMatch(t, []int{1, 2}, got.Settings.AllowedNodes)
	require.Equal(t, []int{7}, got.Settings.DefaultPinned)
	require.True(t, got.Settings.WeightedPolicies)

	// g2's whitelist is empty (unrestricted) — restriction cleared.
	require.Empty(t, got.Settings.LoginIPWhitelist)
}

func TestEffectiveGroupWhitelistUnionsWhenBothRestricted(t *testing.T) {
	g1 := &ent.Group{ID: 1, Permissions: permSet(), Settings: &types.GroupSetting{
		LoginIPWhitelist: []string{"10.0.0.0/8"},
	}}
	g2 := &ent.Group{ID: 2, Permissions: permSet(), Settings: &types.GroupSetting{
		LoginIPWhitelist: []string{"192.168.0.0/16"},
	}}
	u := &ent.User{ID: 1}
	u.Edges.Group = g1
	u.Edges.Memberships = []*ent.GroupMembership{memberOf(u, g2, nil)}

	got := EffectiveGroup(u)
	require.ElementsMatch(t, []string{"10.0.0.0/8", "192.168.0.0/16"}, got.Settings.LoginIPWhitelist)
}

func TestMembershipClientOps(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	g1 := client.Group.Create().SetName("g1").SetPermissions(permSet()).SaveX(ctx)
	g2 := client.Group.Create().SetName("g2").SetPermissions(permSet()).SaveX(ctx)
	g3 := client.Group.Create().SetName("g3").SetPermissions(permSet()).SaveX(ctx)
	u := client.User.Create().SetEmail("m@example.com").SetNick("m").SetGroup(g1).SaveX(ctx)

	uc := NewUserClient(client)

	// Upsert creates, then updates expiry in place.
	require.NoError(t, uc.UpsertMembership(ctx, u.ID, g2.ID, nil))
	exp := time.Now().Add(time.Hour)
	require.NoError(t, uc.UpsertMembership(ctx, u.ID, g2.ID, &exp))
	ms, err := uc.ListMemberships(ctx, u.ID)
	require.NoError(t, err)
	require.Len(t, ms, 1)
	require.Equal(t, g2.ID, ms[0].GroupID)
	require.NotNil(t, ms[0].Expires)

	// Loaded through the user edge, GroupsOf sees the membership.
	full := client.User.Query().Where(user.ID(u.ID)).
		WithGroup().
		WithMemberships(func(mq *ent.GroupMembershipQuery) { mq.WithGroup() }).
		OnlyX(ctx)
	require.ElementsMatch(t, []int{g1.ID, g2.ID}, GroupIDsOf(full))

	// SetMemberships replaces the set, preserving kept rows' expiry.
	require.NoError(t, uc.SetMemberships(ctx, u.ID, []int{g2.ID, g3.ID}))
	ms, err = uc.ListMemberships(ctx, u.ID)
	require.NoError(t, err)
	require.Len(t, ms, 2)
	require.ElementsMatch(t, []int{g2.ID, g3.ID}, []int{ms[0].GroupID, ms[1].GroupID})

	require.NoError(t, uc.RemoveMembership(ctx, u.ID, g2.ID))
	ms, err = uc.ListMemberships(ctx, u.ID)
	require.NoError(t, err)
	require.Len(t, ms, 1)

	// ExpireMemberships sweeps only expired rows.
	past := time.Now().Add(-time.Hour)
	require.NoError(t, uc.UpsertMembership(ctx, u.ID, g2.ID, &past))
	expired, err := uc.ExpireMemberships(ctx)
	require.NoError(t, err)
	require.Len(t, expired, 1)
	require.Equal(t, g2.ID, expired[0].GroupID)
	ms, err = uc.ListMemberships(ctx, u.ID)
	require.NoError(t, err)
	require.Len(t, ms, 1)
	require.Equal(t, g3.ID, ms[0].GroupID)
}

func TestListByGroupsUnionsPolicies(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	pc := NewStoragePolicyClient(client, nil)
	mk := func(name string) *ent.StoragePolicy {
		return client.StoragePolicy.Create().SetName(name).SetType("local").
			SetStatus(storagepolicy.StatusActive).SaveX(ctx)
	}
	pa, pb, shared := mk("a"), mk("b"), mk("shared")

	g1 := client.Group.Create().SetName("g1").SetPermissions(permSet()).
		SetStoragePolicies(pa).AddAllowedPolicies(shared).SaveX(ctx)
	g2 := client.Group.Create().SetName("g2").SetPermissions(permSet()).
		SetStoragePolicies(pb).AddAllowedPolicies(shared).SaveX(ctx)

	got, err := pc.ListByGroups(ctx, []*ent.Group{g1, g2})
	require.NoError(t, err)
	require.Len(t, got, 3)
	require.ElementsMatch(t, []int{pa.ID, pb.ID, shared.ID},
		[]int{got[0].ID, got[1].ID, got[2].ID})
}

func TestAclEffectivePermissionsMultiGroup(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	g1 := client.Group.Create().SetName("g1").SetPermissions(permSet()).SaveX(ctx)
	g2 := client.Group.Create().SetName("g2").SetPermissions(permSet()).SaveX(ctx)
	u := client.User.Create().SetEmail("a@example.com").SetNick("a").SetGroup(g1).SaveX(ctx)
	file := client.File.Create().SetName("dir").SetType(int(types.FileTypeFolder)).SetOwner(u).SaveX(ctx)

	// ACL grants to the membership group only.
	aclEntry(t, client, file.ID, aclentry.SubjectTypeGroup, g2.ID, types.AclPermCreate)

	c := NewAclClient(client, conf.SQLiteDB)

	// Without the membership edge loaded, no match.
	res, err := c.EffectivePermissions(ctx, file.ID, u)
	require.NoError(t, err)
	require.Nil(t, res)

	// With membership loaded, the group ACL applies.
	u.Edges.Group = g1
	u.Edges.Memberships = []*ent.GroupMembership{memberOf(u, g2, nil)}
	res, err = c.EffectivePermissions(ctx, file.ID, u)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.True(t, res.Enabled(int(types.AclPermCreate)))
}

func TestApplyGroupGrantAddsMembership(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	base := client.Group.Create().SetName("base").SetPermissions(permSet()).SaveX(ctx)
	vip := client.Group.Create().SetName("vip").SetPermissions(permSet(types.GroupPermissionWebDAV)).SaveX(ctx)
	u := client.User.Create().SetEmail("v@example.com").SetNick("v").SetGroup(base).SaveX(ctx)

	vc := NewVasClient(client, conf.SQLiteDB).(*vasClient)
	require.NoError(t, vc.applyGroupGrant(ctx, u.ID, vip.ID, 3600))

	// Primary group untouched; membership created with expiry.
	u2 := client.User.GetX(ctx, u.ID)
	require.Equal(t, base.ID, u2.GroupUsers)
	ms := client.GroupMembership.Query().AllX(ctx)
	require.Len(t, ms, 1)
	require.Equal(t, vip.ID, ms[0].GroupID)
	require.NotNil(t, ms[0].Expires)

	// Re-purchase extends rather than duplicating.
	require.NoError(t, vc.applyGroupGrant(ctx, u.ID, vip.ID, 7200))
	ms = client.GroupMembership.Query().AllX(ctx)
	require.Len(t, ms, 1)
}

func TestListUsersGroupFilterIncludesMemberships(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	g1 := client.Group.Create().SetName("g1").SetPermissions(permSet()).SaveX(ctx)
	g2 := client.Group.Create().SetName("g2").SetPermissions(permSet()).SaveX(ctx)
	u1 := client.User.Create().SetEmail("p@example.com").SetNick("p").SetGroup(g2).SaveX(ctx)
	u2 := client.User.Create().SetEmail("q@example.com").SetNick("q").SetGroup(g1).SaveX(ctx)
	client.User.Create().SetEmail("r@example.com").SetNick("r").SetGroup(g1).SaveX(ctx)

	uc := NewUserClient(client)
	require.NoError(t, uc.UpsertMembership(ctx, u2.ID, g2.ID, nil))

	res, err := uc.ListUsers(ctx, &ListUserParameters{
		PaginationArgs: &PaginationArgs{PageSize: 10},
		GroupID:        g2.ID,
	})
	require.NoError(t, err)
	require.Equal(t, 2, res.TotalItems)
	require.ElementsMatch(t, []int{u1.ID, u2.ID},
		[]int{res.Users[0].ID, res.Users[1].ID})

	// CountUsers counts primary + membership holders once.
	gc := NewGroupClient(client, conf.SQLiteDB, nil)
	n, err := gc.CountUsers(ctx, g2.ID)
	require.NoError(t, err)
	require.Equal(t, 2, n)
}
