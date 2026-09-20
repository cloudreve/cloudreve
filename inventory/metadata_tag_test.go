package inventory

import (
	"context"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/stretchr/testify/require"
)

func mkTagFile(t *testing.T, client *ent.Client, u *ent.User, name string, tags map[string]string) *ent.File {
	t.Helper()
	ctx := context.Background()
	f := client.File.Create().SetName(name).SetType(int(types.FileTypeFile)).SetOwner(u).SaveX(ctx)
	for k, v := range tags {
		client.Metadata.Create().SetName(k).SetValue(v).SetFileID(f.ID).SaveX(ctx)
	}
	return f
}

func TestTagMetadataManagement(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	fc := NewFileClient(client, conf.SQLite3DB, nil)

	u, _, _ := dedupFixture(t, client, "a")
	other, _, _ := dedupFixture(t, client, "b")

	mkTagFile(t, client, u, "a.txt", map[string]string{"tag:work": "#ff0000", "tag:proj": "#00ff00"})
	mkTagFile(t, client, u, "b.txt", map[string]string{"tag:work": "#ff0000"})
	mkTagFile(t, client, other, "c.txt", map[string]string{"tag:work": "#0000ff"})

	// List: owner-scoped, prefix-scoped, sorted by name.
	stats, err := fc.ListMetadataStats(ctx, u.ID, "tag:")
	require.NoError(t, err)
	require.Len(t, stats, 2)
	require.Equal(t, "tag:proj", stats[0].Name)
	require.Equal(t, 1, stats[0].Count)
	require.Equal(t, "tag:work", stats[1].Name)
	require.Equal(t, 2, stats[1].Count)
	require.Equal(t, "#ff0000", stats[1].Value)

	// Rename onto an existing tag merges: a.txt already carries tag:work,
	// so its tag:proj row is dropped rather than duplicated.
	require.NoError(t, fc.RenameMetadataName(ctx, u.ID, "tag:proj", "tag:work", nil))
	stats, err = fc.ListMetadataStats(ctx, u.ID, "tag:")
	require.NoError(t, err)
	require.Len(t, stats, 1)
	require.Equal(t, "tag:work", stats[0].Name)
	require.Equal(t, 2, stats[0].Count)

	// Recolor across all of the owner's files.
	red := "#123456"
	require.NoError(t, fc.RenameMetadataName(ctx, u.ID, "tag:work", "tag:work", &red))
	stats, err = fc.ListMetadataStats(ctx, u.ID, "tag:")
	require.NoError(t, err)
	require.Equal(t, "#123456", stats[0].Value)

	// Other owner's tag untouched by all of the above.
	ostats, err := fc.ListMetadataStats(ctx, other.ID, "tag:")
	require.NoError(t, err)
	require.Len(t, ostats, 1)
	require.Equal(t, "#0000ff", ostats[0].Value)

	// Delete scoped to owner.
	require.NoError(t, fc.DeleteMetadataByNameForOwner(ctx, u.ID, "tag:work"))
	stats, err = fc.ListMetadataStats(ctx, u.ID, "tag:")
	require.NoError(t, err)
	require.Empty(t, stats)
	ostats, err = fc.ListMetadataStats(ctx, other.ID, "tag:")
	require.NoError(t, err)
	require.Len(t, ostats, 1)
}
