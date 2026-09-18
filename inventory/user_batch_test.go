package inventory

import (
	"context"
	"testing"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	entuser "github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/stretchr/testify/require"
)

func TestBatchUpdateUsers(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	g1 := client.Group.Create().SetName("g1").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	g2 := client.Group.Create().SetName("g2").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	uc := NewUserClient(client)

	u1 := client.User.Create().SetEmail("b1@example.com").SetNick("u1").SetStatus(entuser.StatusActive).SetGroup(g1).SaveX(ctx)
	u2 := client.User.Create().SetEmail("b2@example.com").SetNick("u2").SetStatus(entuser.StatusActive).SetGroup(g1).SaveX(ctx)
	u3 := client.User.Create().SetEmail("b3@example.com").SetNick("u3").SetStatus(entuser.StatusActive).SetGroup(g1).SaveX(ctx)

	t.Run("status change clears ban fields", func(t *testing.T) {
		banned := client.User.Create().SetEmail("banned@example.com").SetNick("ub").
			SetStatus(entuser.StatusManualBanned).SetBanExpires(time.Now().Add(time.Hour)).
			SetBanReason("spam").SetGroup(g1).SaveX(ctx)

		st := entuser.StatusInactive
		n, err := uc.BatchUpdate(ctx, []int{banned.ID}, &st, 0)
		require.NoError(t, err)
		require.Equal(t, 1, n)

		got := client.User.GetX(ctx, banned.ID)
		require.Equal(t, entuser.StatusInactive, got.Status)
		require.Nil(t, got.BanExpires)
		require.Equal(t, "", got.BanReason)
	})

	t.Run("group change", func(t *testing.T) {
		n, err := uc.BatchUpdate(ctx, []int{u1.ID, u2.ID}, nil, g2.ID)
		require.NoError(t, err)
		require.Equal(t, 2, n)

		require.Equal(t, g2.ID, client.User.GetX(ctx, u1.ID).QueryGroup().OnlyIDX(ctx))
		require.Equal(t, g2.ID, client.User.GetX(ctx, u2.ID).QueryGroup().OnlyIDX(ctx))
		require.Equal(t, g1.ID, client.User.GetX(ctx, u3.ID).QueryGroup().OnlyIDX(ctx))
	})

	t.Run("empty id list is no-op", func(t *testing.T) {
		st := entuser.StatusInactive
		n, err := uc.BatchUpdate(ctx, nil, &st, g2.ID)
		require.NoError(t, err)
		require.Equal(t, 0, n)
	})
}

func TestListUsersByIDs(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	group := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	uc := NewUserClient(client)

	u1 := client.User.Create().SetEmail("id1@example.com").SetNick("u1").SetGroup(group).SaveX(ctx)
	u2 := client.User.Create().SetEmail("id2@example.com").SetNick("u2").SetGroup(group).SaveX(ctx)
	client.User.Create().SetEmail("id3@example.com").SetNick("u3").SetGroup(group).SaveX(ctx)

	res, err := uc.ListUsers(ctx, &ListUserParameters{
		PaginationArgs: &PaginationArgs{Page: 0, PageSize: 10},
		IDs:            []int{u1.ID, u2.ID},
	})
	require.NoError(t, err)
	require.Equal(t, 2, res.PaginationResults.TotalItems)
	require.Len(t, res.Users, 2)
}

func TestUpdateLastLogin(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	group := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	uc := NewUserClient(client)

	u := client.User.Create().SetEmail("ll@example.com").SetNick("u").SetGroup(group).SaveX(ctx)
	require.Nil(t, client.User.GetX(ctx, u.ID).LastLogin)

	require.NoError(t, uc.UpdateLastLogin(ctx, u.ID))
	got := client.User.GetX(ctx, u.ID)
	require.NotNil(t, got.LastLogin)
	require.WithinDuration(t, time.Now(), *got.LastLogin, time.Minute)
}
