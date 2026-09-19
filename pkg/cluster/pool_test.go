package cluster

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/ent/node"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubSettings struct {
	setting.Provider
}

func capsOf(bits ...types.NodeCapability) *boolset.BooleanSet {
	b := &boolset.BooleanSet{}
	for _, c := range bits {
		boolset.Set(int(c), true, b)
	}
	return b
}

func newPoolFixture(t *testing.T) (*ent.Client, NodePool, context.Context) {
	t.Helper()
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	t.Cleanup(func() { client.Close() })

	logger := logging.NewConsoleLogger(logging.LevelError)
	cfg, err := conf.NewIniConfigProvider(filepath.Join(t.TempDir(), "conf.ini"), logger)
	require.NoError(t, err)

	pool, err := NewNodePool(ctx, logger, cfg, &stubSettings{}, inventory.NewNodeClient(client))
	require.NoError(t, err)
	return client, pool, ctx
}

func TestWeightedNodePoolGet(t *testing.T) {
	a := assert.New(t)
	client, pool, ctx := newPoolFixture(t)

	n1 := client.Node.Create().
		SetName("n1").SetType(node.TypeMaster).SetStatus(node.StatusActive).
		SetServer("http://n1").SetCapabilities(capsOf(types.NodeCapabilityRemoteDownload)).
		SetWeight(10).SaveX(ctx)
	client.Node.Create().
		SetName("n2").SetType(node.TypeMaster).SetStatus(node.StatusActive).
		SetServer("http://n2").SetCapabilities(capsOf(types.NodeCapabilityRemoteDownload)).
		SetWeight(1).SaveX(ctx)

	pool.(*weightedNodePool).Upsert(ctx, n1)
	// Upsert requires a full refresh — rebuild pool from the seeded DB instead.
	pool, err := NewNodePool(ctx, logging.NewConsoleLogger(logging.LevelError), pool.(*weightedNodePool).conf, &stubSettings{}, inventory.NewNodeClient(client))
	require.NoError(t, err)

	// Capability with no matching nodes errors.
	_, err = pool.Get(ctx, types.NodeCapabilityCreateArchive, 0, nil)
	a.True(errors.Is(err, ErrNoAvailableNode))

	// Preferred node wins regardless of weight.
	selected, err := pool.Get(ctx, types.NodeCapabilityRemoteDownload, n1.ID, nil)
	require.NoError(t, err)
	a.Equal(n1.ID, selected.ID())

	// Without preference, the heavier node wins the first pick.
	selected, err = pool.Get(ctx, types.NodeCapabilityRemoteDownload, 0, nil)
	require.NoError(t, err)
	a.Equal(n1.ID, selected.ID())
}

func TestWeightedNodePoolGetAllowed(t *testing.T) {
	a := assert.New(t)
	client, pool, ctx := newPoolFixture(t)

	n1 := client.Node.Create().
		SetName("n1").SetType(node.TypeMaster).SetStatus(node.StatusActive).
		SetServer("http://n1").SetCapabilities(capsOf(types.NodeCapabilityRemoteDownload)).
		SetWeight(10).SaveX(ctx)
	n2 := client.Node.Create().
		SetName("n2").SetType(node.TypeMaster).SetStatus(node.StatusActive).
		SetServer("http://n2").SetCapabilities(capsOf(types.NodeCapabilityRemoteDownload)).
		SetWeight(1).SaveX(ctx)

	pool, err := NewNodePool(ctx, logging.NewConsoleLogger(logging.LevelError), pool.(*weightedNodePool).conf, &stubSettings{}, inventory.NewNodeClient(client))
	require.NoError(t, err)

	// Allowed list narrows dispatch: heavier n1 excluded, n2 wins.
	selected, err := pool.Get(ctx, types.NodeCapabilityRemoteDownload, 0, []int{n2.ID})
	require.NoError(t, err)
	a.Equal(n2.ID, selected.ID())

	// Preferred node outside the allowed set falls back within the set.
	selected, err = pool.Get(ctx, types.NodeCapabilityRemoteDownload, n1.ID, []int{n2.ID})
	require.NoError(t, err)
	a.Equal(n2.ID, selected.ID())

	// Preferred node inside the allowed set wins.
	selected, err = pool.Get(ctx, types.NodeCapabilityRemoteDownload, n1.ID, []int{n1.ID, n2.ID})
	require.NoError(t, err)
	a.Equal(n1.ID, selected.ID())

	// Empty allowed intersection errors.
	_, err = pool.Get(ctx, types.NodeCapabilityRemoteDownload, 0, []int{9999})
	a.True(errors.Is(err, ErrNoAvailableNode))
}

func TestWeightedNodePoolGetUnknownCapability(t *testing.T) {
	a := assert.New(t)
	_, pool, ctx := newPoolFixture(t)

	_, err := pool.Get(ctx, types.NodeCapabilityRemoteDownload, 0, nil)
	a.True(errors.Is(err, ErrNoAvailableNode))
}
