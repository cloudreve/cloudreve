package inventory

import (
	"context"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/stretchr/testify/require"
)

func TestConsumeDirectTraffic(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	g := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	uc := NewUserClient(client)

	t.Run("unlimited user always succeeds", func(t *testing.T) {
		// Default dl_traffic is -1 (unlimited).
		u := client.User.Create().SetEmail("un@example.com").SetNick("un").SetGroup(g).SaveX(ctx)
		require.Equal(t, int64(-1), u.DlTraffic)

		ok, err := uc.ConsumeDirectTraffic(ctx, u.ID, 1<<40)
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, int64(-1), client.User.GetX(ctx, u.ID).DlTraffic)
	})

	t.Run("finite balance decremented", func(t *testing.T) {
		u := client.User.Create().SetEmail("fin@example.com").SetNick("fin").SetGroup(g).SetDlTraffic(1000).SaveX(ctx)

		ok, err := uc.ConsumeDirectTraffic(ctx, u.ID, 400)
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, int64(600), client.User.GetX(ctx, u.ID).DlTraffic)

		// Exact remainder is consumable.
		ok, err = uc.ConsumeDirectTraffic(ctx, u.ID, 600)
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, int64(0), client.User.GetX(ctx, u.ID).DlTraffic)
	})

	t.Run("insufficient balance rejected without going negative", func(t *testing.T) {
		u := client.User.Create().SetEmail("ins@example.com").SetNick("ins").SetGroup(g).SetDlTraffic(100).SaveX(ctx)

		ok, err := uc.ConsumeDirectTraffic(ctx, u.ID, 101)
		require.NoError(t, err)
		require.False(t, ok)
		require.Equal(t, int64(100), client.User.GetX(ctx, u.ID).DlTraffic)

		// Zero balance rejects any positive size.
		empty := client.User.Create().SetEmail("empty@example.com").SetNick("empty").SetGroup(g).SetDlTraffic(0).SaveX(ctx)
		ok, err = uc.ConsumeDirectTraffic(ctx, empty.ID, 1)
		require.NoError(t, err)
		require.False(t, ok)
		require.Equal(t, int64(0), client.User.GetX(ctx, empty.ID).DlTraffic)
	})

	t.Run("non-positive size is a no-op", func(t *testing.T) {
		u := client.User.Create().SetEmail("z@example.com").SetNick("z").SetGroup(g).SetDlTraffic(10).SaveX(ctx)
		ok, err := uc.ConsumeDirectTraffic(ctx, u.ID, 0)
		require.NoError(t, err)
		require.True(t, ok)
		ok, err = uc.ConsumeDirectTraffic(ctx, u.ID, -5)
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, int64(10), client.User.GetX(ctx, u.ID).DlTraffic)
	})
}

func TestAddDirectTraffic(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	g := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	uc := NewUserClient(client)

	// Finite balance grows.
	limited := client.User.Create().SetEmail("lim@example.com").SetNick("lim").SetGroup(g).SetDlTraffic(100).SaveX(ctx)
	require.NoError(t, uc.AddDirectTraffic(ctx, limited.ID, 50))
	require.Equal(t, int64(150), client.User.GetX(ctx, limited.ID).DlTraffic)

	// Unlimited stays unlimited.
	unlimited := client.User.Create().SetEmail("unl@example.com").SetNick("unl").SetGroup(g).SaveX(ctx)
	require.NoError(t, uc.AddDirectTraffic(ctx, unlimited.ID, 1<<40))
	require.Equal(t, int64(-1), client.User.GetX(ctx, unlimited.ID).DlTraffic)

	// Non-positive size is a no-op.
	require.NoError(t, uc.AddDirectTraffic(ctx, limited.ID, 0))
	require.NoError(t, uc.AddDirectTraffic(ctx, limited.ID, -10))
	require.Equal(t, int64(150), client.User.GetX(ctx, limited.ID).DlTraffic)
}
