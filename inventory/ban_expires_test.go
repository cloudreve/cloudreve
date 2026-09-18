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

func TestLiftExpiredBan(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	group := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)
	uc := NewUserClient(client)

	t.Run("expired manual ban lifted", func(t *testing.T) {
		u := client.User.Create().SetEmail("lift-exp@example.com").SetNick("u").
			SetStatus(entuser.StatusManualBanned).SetBanExpires(past).SetGroup(group).SaveX(ctx)
		out, err := uc.LiftExpiredBan(ctx, u)
		require.NoError(t, err)
		require.Equal(t, entuser.StatusActive, out.Status)
		require.Nil(t, client.User.GetX(ctx, u.ID).BanExpires)
	})

	t.Run("expired sys ban lifted", func(t *testing.T) {
		u := client.User.Create().SetEmail("lift-sys@example.com").SetNick("u").
			SetStatus(entuser.StatusSysBanned).SetBanExpires(past).SetGroup(group).SaveX(ctx)
		out, err := uc.LiftExpiredBan(ctx, u)
		require.NoError(t, err)
		require.Equal(t, entuser.StatusActive, out.Status)
	})

	t.Run("future ban untouched", func(t *testing.T) {
		u := client.User.Create().SetEmail("lift-future@example.com").SetNick("u").
			SetStatus(entuser.StatusManualBanned).SetBanExpires(future).SetGroup(group).SaveX(ctx)
		out, err := uc.LiftExpiredBan(ctx, u)
		require.NoError(t, err)
		require.Equal(t, entuser.StatusManualBanned, out.Status)
		require.NotNil(t, client.User.GetX(ctx, u.ID).BanExpires)
	})

	t.Run("permanent ban untouched", func(t *testing.T) {
		u := client.User.Create().SetEmail("lift-perm@example.com").SetNick("u").
			SetStatus(entuser.StatusManualBanned).SetGroup(group).SaveX(ctx)
		out, err := uc.LiftExpiredBan(ctx, u)
		require.NoError(t, err)
		require.Equal(t, entuser.StatusManualBanned, out.Status)
	})

	t.Run("active user untouched", func(t *testing.T) {
		u := client.User.Create().SetEmail("lift-active@example.com").SetNick("u").
			SetStatus(entuser.StatusActive).SetGroup(group).SaveX(ctx)
		out, err := uc.LiftExpiredBan(ctx, u)
		require.NoError(t, err)
		require.Equal(t, entuser.StatusActive, out.Status)
	})
}

func TestUpsertBanExpires(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	group := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	uc := NewUserClient(client)
	future := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)

	u := client.User.Create().SetEmail("upsert-ban@example.com").SetNick("u").
		SetStatus(entuser.StatusActive).SetGroup(group).SaveX(ctx)

	// Ban with expiry and reason
	u.Status = entuser.StatusManualBanned
	u.BanExpires = &future
	u.BanReason = "spam"
	_, err := uc.Upsert(ctx, u, "", "")
	require.NoError(t, err)
	got := client.User.GetX(ctx, u.ID)
	require.Equal(t, future, got.BanExpires.UTC())
	require.Equal(t, "spam", got.BanReason)

	// Unban clears the expiry and reason
	u.Status = entuser.StatusActive
	u.BanExpires = nil
	u.BanReason = ""
	_, err = uc.Upsert(ctx, u, "", "")
	require.NoError(t, err)
	got = client.User.GetX(ctx, u.ID)
	require.Nil(t, got.BanExpires)
	require.Equal(t, "", got.BanReason)
}
