package inventory

import (
	"context"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient opens an in-memory SQLite ent client with the schema created.
func newTestClient(t *testing.T) *ent.Client {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:apptest?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func makePolicy(t *testing.T, client *ent.Client, name string) *ent.StoragePolicy {
	t.Helper()
	p, err := client.StoragePolicy.Create().SetName(name).SetType("local").Save(context.Background())
	require.NoError(t, err)
	return p
}

// TestMultiStoragePolicyPerGroup covers APP-100: a group may hold several allowed
// policies, ListPoliciesByGroup returns the full set (always including the default),
// and membership can be decided from it.
func TestMultiStoragePolicyPerGroup(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	gc := NewGroupClient(client, "sqlite", nil)
	sc := NewStoragePolicyClient(client, nil)

	p1 := makePolicy(t, client, "local-1")
	p2 := makePolicy(t, client, "s3-2")
	p3 := makePolicy(t, client, "unassigned-3")

	// Create a group with default = p1 and allowed = {p1, p2}.
	created, err := gc.Upsert(ctx, &ent.Group{
		Name:        "multi",
		Permissions: &boolset.BooleanSet{},
		Settings:    &types.GroupSetting{},
		Edges: ent.GroupEdges{
			StoragePolicies:        p1,
			StoragePoliciesAllowed: []*ent.StoragePolicy{p1, p2},
		},
	})
	require.NoError(t, err)

	// The allowed set must be exactly {p1, p2}.
	allowed, err := sc.ListPoliciesByGroup(ctx, created)
	require.NoError(t, err)
	ids := policyIDSet(allowed)
	assert.ElementsMatch(t, []int{p1.ID, p2.ID}, keysOf(ids), "allowed set should be {p1,p2}")

	// Membership: p1/p2 allowed, p3 rejected (the guard used at upload time).
	assert.True(t, ids[p1.ID], "p1 should be allowed")
	assert.True(t, ids[p2.ID], "p2 should be allowed")
	assert.False(t, ids[p3.ID], "p3 must NOT be allowed (out-of-set upload rejected)")

	// The default remains p1.
	def, err := sc.GetByGroup(ctx, created)
	require.NoError(t, err)
	assert.Equal(t, p1.ID, def.ID)
}

// TestSinglePolicyGroupNoRegression covers the "no regression" criterion: a group
// created with only a default policy still resolves that policy as its allowed set.
func TestSinglePolicyGroupNoRegression(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	gc := NewGroupClient(client, "sqlite", nil)
	sc := NewStoragePolicyClient(client, nil)

	p1 := makePolicy(t, client, "only")

	created, err := gc.Upsert(ctx, &ent.Group{
		Name:        "single",
		Permissions: &boolset.BooleanSet{},
		Settings:    &types.GroupSetting{},
		Edges:       ent.GroupEdges{StoragePolicies: p1},
	})
	require.NoError(t, err)

	allowed, err := sc.ListPoliciesByGroup(ctx, created)
	require.NoError(t, err)
	ids := policyIDSet(allowed)
	// Even though no allowed edge was set, the default is guaranteed to be a member.
	assert.ElementsMatch(t, []int{p1.ID}, keysOf(ids))
}

// TestBackfillGroupAvailablePolicies verifies the idempotent migration that seeds the
// allowed set from the legacy single storage_policy_id.
func TestBackfillGroupAvailablePolicies(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	sc := NewStoragePolicyClient(client, nil)

	p1 := makePolicy(t, client, "legacy")

	// Legacy group: only storage_policy_id set, no allowed edge.
	legacy, err := client.Group.Create().
		SetName("legacy").
		SetPermissions(&boolset.BooleanSet{}).
		SetSettings(&types.GroupSetting{}).
		SetStoragePolicyID(p1.ID).
		Save(ctx)
	require.NoError(t, err)

	// Before backfill: empty allowed set at the edge level.
	pre, err := client.Group.QueryStoragePoliciesAllowed(legacy).All(ctx)
	require.NoError(t, err)
	assert.Len(t, pre, 0)

	// Run the backfill twice to assert idempotency.
	require.NoError(t, migrateGroupAvailablePolicies(logging.NewConsoleLogger(logging.LevelError), client, ctx))
	require.NoError(t, migrateGroupAvailablePolicies(logging.NewConsoleLogger(logging.LevelError), client, ctx))

	post, err := client.Group.QueryStoragePoliciesAllowed(legacy).All(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []int{p1.ID}, policyIDsOf(post), "backfill should seed the default into the allowed set exactly once")

	// And ListPoliciesByGroup now reflects it.
	reloaded, err := client.Group.Get(ctx, legacy.ID)
	require.NoError(t, err)
	allowed, err := sc.ListPoliciesByGroup(ctx, reloaded)
	require.NoError(t, err)
	assert.ElementsMatch(t, []int{p1.ID}, policyIDsOf(allowed))
}

func policyIDSet(ps []*ent.StoragePolicy) map[int]bool {
	m := make(map[int]bool, len(ps))
	for _, p := range ps {
		m[p.ID] = true
	}
	return m
}

func keysOf(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func policyIDsOf(ps []*ent.StoragePolicy) []int {
	out := make([]int, 0, len(ps))
	for _, p := range ps {
		out = append(out, p.ID)
	}
	return out
}
