package inventory

import (
	"context"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/stretchr/testify/require"
)

func TestUpdateEntityProps(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	policy := client.StoragePolicy.Create().
		SetName("local").
		SetType(string(types.PolicyTypeLocal)).
		SetBucketName("bucket").
		SetAccessKey("ak").
		SetSecretKey("sk").
		SetMaxSize(0).
		SetDirNameRule("d").
		SetFileNameRule("f").
		SaveX(ctx)

	e := client.Entity.Create().
		SetType(int(types.EntityTypeVersion)).
		SetSource("uploads/1/test.bin").
		SetSize(100).
		SetStoragePolicyEntities(policy.ID).
		SaveX(ctx)

	c := NewFileClient(client, "sqlite3", nil)

	e.Props = &types.EntityProps{RecycleFailCount: 3}
	require.NoError(t, c.UpdateEntityProps(ctx, e))

	got := client.Entity.GetX(ctx, e.ID)
	require.NotNil(t, got.Props)
	require.Equal(t, 3, got.Props.RecycleFailCount)

	// Other props survive a partial update.
	e.Props.RecycleFailCount = 5
	e.Props.UnlinkOnly = true
	require.NoError(t, c.UpdateEntityProps(ctx, e))

	got = client.Entity.GetX(ctx, e.ID)
	require.Equal(t, 5, got.Props.RecycleFailCount)
	require.True(t, got.Props.UnlinkOnly)
}
