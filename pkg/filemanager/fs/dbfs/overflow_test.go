package dbfs

import (
	"context"
	"math"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/ent/storagepolicy"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/stretchr/testify/require"
)

func overflowDBFS(t *testing.T, client *ent.Client) *DBFS {
	t.Helper()
	hasher, err := hashid.New("overflow-test-salt")
	require.NoError(t, err)
	return &DBFS{
		fileClient:          inventory.NewFileClient(client, conf.SQLiteDB, hasher),
		storagePolicyClient: inventory.NewStoragePolicyClient(client, nil),
		hasher:              hasher,
		l:                   logging.NewConsoleLogger(logging.LevelError),
	}
}

func mkPolicy(t *testing.T, client *ent.Client, name string, cap int64, overflow int) *ent.StoragePolicy {
	t.Helper()
	return client.StoragePolicy.Create().SetName(name).SetType("local").
		SetStatus(storagepolicy.StatusActive).
		SetSettings(&types.PolicySetting{MaxTotalSize: cap, OverflowPolicyID: overflow}).
		SaveX(context.Background())
}

func mkUser(t *testing.T, client *ent.Client) *ent.User {
	t.Helper()
	g := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).SaveX(context.Background())
	return client.User.Create().SetEmail("o@example.com").SetNick("o").
		SetGroup(g).SaveX(context.Background())
}

func seedEntity(t *testing.T, client *ent.Client, u *ent.User, p *ent.StoragePolicy, size int64) {
	t.Helper()
	client.Entity.Create().SetType(int(types.EntityTypeVersion)).
		SetSource("cloudreve/data/" + p.Name + "/" + t.Name()).SetSize(size).
		SetReferenceCount(1).SetCreatedBy(u.ID).
		SetStoragePolicyEntities(p.ID).SaveX(context.Background())
}

func TestOverflowChainResolution(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	f := overflowDBFS(t, client)
	u := mkUser(t, client)

	open := mkPolicy(t, client, "open", 0, 0)
	mid := mkPolicy(t, client, "mid", 300, open.ID)
	head := mkPolicy(t, client, "head", 1000, mid.ID)
	seedEntity(t, client, u, head, 900) // head: 100 bytes headroom

	// Fits in the preferred policy — no spill.
	require.Equal(t, head.ID, f.resolveOverflowPolicy(ctx, head, 100).ID)
	// Head full for this size — spills to mid.
	require.Equal(t, mid.ID, f.resolveOverflowPolicy(ctx, head, 200).ID)
	// Head and mid both too small — lands on the uncapped tail.
	require.Equal(t, open.ID, f.resolveOverflowPolicy(ctx, head, 400).ID)
}

func TestOverflowChainExhaustedAndCycles(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	f := overflowDBFS(t, client)
	u := mkUser(t, client)

	// Cycle: a -> b -> a, both full. Resolution must terminate and return
	// the last reachable member so the caller still fails validation.
	b := mkPolicy(t, client, "b", 10, 0)
	a := mkPolicy(t, client, "a", 10, b.ID)
	client.StoragePolicy.UpdateOne(b).SetSettings(&types.PolicySetting{MaxTotalSize: 10, OverflowPolicyID: a.ID}).ExecX(ctx)
	seedEntity(t, client, u, a, 10)
	seedEntity(t, client, u, b, 10)

	require.Equal(t, b.ID, f.resolveOverflowPolicy(ctx, a, 1).ID)
	require.ErrorIs(t, f.validatePolicyCapacity(ctx, 1, f.resolveOverflowPolicy(ctx, a, 1)),
		fs.ErrInsufficientCapacity)

	// Suspended members are skipped over to the next hop.
	suspended := client.StoragePolicy.Create().SetName("sus").SetType("local").
		SetStatus(storagepolicy.StatusSuspended).
		SetSettings(&types.PolicySetting{}).SaveX(ctx)
	tail := mkPolicy(t, client, "tail", 0, 0)
	client.StoragePolicy.UpdateOne(suspended).SetSettings(&types.PolicySetting{OverflowPolicyID: tail.ID}).ExecX(ctx)
	entry := mkPolicy(t, client, "entry", 5, suspended.ID)
	seedEntity(t, client, u, entry, 5)

	require.Equal(t, tail.ID, f.resolveOverflowPolicy(ctx, entry, 1).ID)
}

func TestOverflowChainHeadroom(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	f := overflowDBFS(t, client)
	u := mkUser(t, client)

	// Any uncapped member makes the chain unbounded.
	open := mkPolicy(t, client, "open", 0, 0)
	capped := mkPolicy(t, client, "capped", 100, open.ID)
	seedEntity(t, client, u, capped, 60)
	headroom, err := f.chainHeadroom(ctx, capped)
	require.NoError(t, err)
	require.Equal(t, int64(math.MaxInt64), headroom)

	// Fully capped chain sums per-member headroom.
	p2 := mkPolicy(t, client, "p2", 50, 0)
	p1 := mkPolicy(t, client, "p1", 100, p2.ID)
	seedEntity(t, client, u, p1, 30)
	seedEntity(t, client, u, p2, 10)
	headroom, err = f.chainHeadroom(ctx, p1)
	require.NoError(t, err)
	require.Equal(t, int64(70+40), headroom)
}
