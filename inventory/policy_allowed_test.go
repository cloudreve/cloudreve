package inventory

import (
	"context"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/ent/storagepolicy"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/stretchr/testify/require"
)

func TestListByGroup(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	pc := NewStoragePolicyClient(client, nil)
	mk := func(name string, status storagepolicy.Status) *ent.StoragePolicy {
		return client.StoragePolicy.Create().
			SetName(name).SetType("local").SetStatus(status).SaveX(ctx)
	}

	t.Run("legacy default included when no M2M rows", func(t *testing.T) {
		def := mk("def", storagepolicy.StatusActive)
		g := client.Group.Create().SetName("g1").SetPermissions(&boolset.BooleanSet{}).
			SetStoragePolicies(def).SaveX(ctx)

		got, err := pc.ListByGroup(ctx, g)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, def.ID, got[0].ID)
	})

	t.Run("allowed set union with default, active only, ordered by id", func(t *testing.T) {
		def := mk("def", storagepolicy.StatusActive)
		a := mk("a", storagepolicy.StatusActive)
		b := mk("b", storagepolicy.StatusActive)
		susp := mk("susp", storagepolicy.StatusSuspended)
		g := client.Group.Create().SetName("g2").SetPermissions(&boolset.BooleanSet{}).
			SetStoragePolicies(def).AddAllowedPolicies(a, b, susp).SaveX(ctx)

		got, err := pc.ListByGroup(ctx, g)
		require.NoError(t, err)
		require.Len(t, got, 3)
		ids := []int{got[0].ID, got[1].ID, got[2].ID}
		require.IsIncreasing(t, ids)
		require.ElementsMatch(t, []int{a.ID, b.ID, def.ID}, ids)
	})

	t.Run("suspended default not included", func(t *testing.T) {
		def := mk("def", storagepolicy.StatusSuspended)
		a := mk("a", storagepolicy.StatusActive)
		g := client.Group.Create().SetName("g3").SetPermissions(&boolset.BooleanSet{}).
			SetStoragePolicies(def).AddAllowedPolicies(a).SaveX(ctx)

		got, err := pc.ListByGroup(ctx, g)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, a.ID, got[0].ID)
	})

	t.Run("no usable policy returns empty", func(t *testing.T) {
		def := mk("def", storagepolicy.StatusSuspended)
		g := client.Group.Create().SetName("g4").SetPermissions(&boolset.BooleanSet{}).
			SetStoragePolicies(def).SaveX(ctx)

		got, err := pc.ListByGroup(ctx, g)
		require.NoError(t, err)
		require.Empty(t, got)
	})
}

func TestResolveLoadBalance(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	pc := NewStoragePolicyClient(client, nil)

	t.Run("skips suspended, missing and nested children", func(t *testing.T) {
		nested := client.StoragePolicy.Create().SetName("nested").SetType(types.PolicyTypeLoadBalance).
			SetStatus(storagepolicy.StatusActive).SaveX(ctx)
		susp := client.StoragePolicy.Create().SetName("susp").SetType("local").
			SetStatus(storagepolicy.StatusSuspended).SaveX(ctx)
		ok := client.StoragePolicy.Create().SetName("ok").SetType("local").
			SetStatus(storagepolicy.StatusActive).SaveX(ctx)
		lb := client.StoragePolicy.Create().SetName("lb").SetType(types.PolicyTypeLoadBalance).
			SetStatus(storagepolicy.StatusActive).
			SetSettings(&types.PolicySetting{LBPolicies: []types.LBPolicyRef{
				{PolicyID: nested.ID, Weight: 10},
				{PolicyID: susp.ID, Weight: 10},
				{PolicyID: 99999, Weight: 10},
				{PolicyID: ok.ID, Weight: 10},
			}}).SaveX(ctx)

		for i := 0; i < 20; i++ {
			got, err := pc.ResolveLoadBalance(ctx, lb)
			require.NoError(t, err)
			require.Equal(t, ok.ID, got.ID)
		}
	})

	t.Run("weight of zero treated as one", func(t *testing.T) {
		zero := client.StoragePolicy.Create().SetName("zero").SetType("local").
			SetStatus(storagepolicy.StatusActive).SaveX(ctx)
		lb := client.StoragePolicy.Create().SetName("lb").SetType(types.PolicyTypeLoadBalance).
			SetStatus(storagepolicy.StatusActive).
			SetSettings(&types.PolicySetting{LBPolicies: []types.LBPolicyRef{
				{PolicyID: zero.ID, Weight: 0},
			}}).SaveX(ctx)

		got, err := pc.ResolveLoadBalance(ctx, lb)
		require.NoError(t, err)
		require.Equal(t, zero.ID, got.ID)
	})

	t.Run("no usable children errors", func(t *testing.T) {
		lb := client.StoragePolicy.Create().SetName("lb").SetType(types.PolicyTypeLoadBalance).
			SetStatus(storagepolicy.StatusActive).
			SetSettings(&types.PolicySetting{LBPolicies: []types.LBPolicyRef{
				{PolicyID: 424242, Weight: 5},
			}}).SaveX(ctx)

		_, err := pc.ResolveLoadBalance(ctx, lb)
		require.Error(t, err)
	})
}
