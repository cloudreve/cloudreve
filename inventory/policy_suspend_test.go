package inventory

import (
	"context"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/ent/storagepolicy"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/stretchr/testify/require"
)

func TestGetByGroupSkipsSuspended(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	pc := NewStoragePolicyClient(client, nil)

	t.Run("suspended group policy yields not found", func(t *testing.T) {
		p := client.StoragePolicy.Create().
			SetName("suspended").SetType("local").
			SetStatus(storagepolicy.StatusSuspended).SaveX(ctx)
		group := client.Group.Create().SetName("g1").SetPermissions(&boolset.BooleanSet{}).
			SetStoragePolicies(p).SaveX(ctx)

		_, err := pc.GetByGroup(ctx, group)
		require.Error(t, err)
	})

	t.Run("active group policy returned", func(t *testing.T) {
		p := client.StoragePolicy.Create().
			SetName("active").SetType("local").
			SetStatus(storagepolicy.StatusActive).SaveX(ctx)
		group := client.Group.Create().SetName("g2").SetPermissions(&boolset.BooleanSet{}).
			SetStoragePolicies(p).SaveX(ctx)

		got, err := pc.GetByGroup(ctx, group)
		require.NoError(t, err)
		require.Equal(t, "active", got.Name)
	})
}
