package inventory

import (
	"context"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/ent/storagepolicy"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/stretchr/testify/require"
)

func TestParseReferenceCountFilter(t *testing.T) {
	f, err := ParseReferenceCountFilter("stale")
	require.NoError(t, err)
	require.True(t, f.Stale)

	f, err = ParseReferenceCountFilter("gt:2")
	require.NoError(t, err)
	require.Equal(t, "gt", f.Op)
	require.Equal(t, 2, f.Value)

	f, err = ParseReferenceCountFilter("eq:0")
	require.NoError(t, err)
	require.Equal(t, "eq", f.Op)

	for _, bad := range []string{"", "gt", "xx:1", "gt:abc", "1"} {
		_, err := ParseReferenceCountFilter(bad)
		require.Error(t, err, "expr %q should be rejected", bad)
	}
}

func TestListEntitiesReferenceCountFilter(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	p := client.StoragePolicy.Create().SetName("p").SetType("local").
		SetStatus(storagepolicy.StatusActive).SaveX(ctx)
	mk := func(name string, ref int) {
		client.Entity.Create().SetType(0).SetSource(name).SetSize(1).
			SetReferenceCount(ref).SetStoragePolicyEntities(p.ID).SaveX(ctx)
	}
	mk("a", 0)
	mk("b", -1)
	mk("c", 1)
	mk("d", 3)

	fc := NewFileClient(client, conf.SQLite3DB, nil)
	args := &ListEntityParameters{PaginationArgs: &PaginationArgs{Page: 0, PageSize: 50}}

	res, err := fc.ListEntities(ctx, args)
	require.NoError(t, err)
	require.Equal(t, 4, res.TotalItems)

	stale, _ := ParseReferenceCountFilter("stale")
	res, err = fc.ListEntities(ctx, &ListEntityParameters{
		PaginationArgs: args.PaginationArgs, ReferenceCount: stale})
	require.NoError(t, err)
	require.Equal(t, 2, res.TotalItems)

	gt, _ := ParseReferenceCountFilter("gt:1")
	res, err = fc.ListEntities(ctx, &ListEntityParameters{
		PaginationArgs: args.PaginationArgs, ReferenceCount: gt})
	require.NoError(t, err)
	require.Equal(t, 1, res.TotalItems)

	lt, _ := ParseReferenceCountFilter("lt:1")
	res, err = fc.ListEntities(ctx, &ListEntityParameters{
		PaginationArgs: args.PaginationArgs, ReferenceCount: lt})
	require.NoError(t, err)
	require.Equal(t, 2, res.TotalItems)

	eq, _ := ParseReferenceCountFilter("eq:1")
	res, err = fc.ListEntities(ctx, &ListEntityParameters{
		PaginationArgs: args.PaginationArgs, ReferenceCount: eq})
	require.NoError(t, err)
	require.Equal(t, 1, res.TotalItems)
}
