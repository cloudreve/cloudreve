package inventory

import (
	"context"
	"testing"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/stretchr/testify/require"
)

func TestActivityRecordAndList(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	defer client.Close()
	c := NewActivityClient(client, conf.SQLiteDB)
	ctx := context.Background()

	_, err := c.Record(ctx, &RecordActivityParams{
		Type:    types.EventFileCreate,
		ActorID: 7,
		IP:      "10.0.0.1",
		FileID:  42,
		Extra:   map[string]any{"uri": "cloudreve://test/a.txt"},
	})
	require.NoError(t, err)
	_, err = c.Record(ctx, &RecordActivityParams{Type: types.EventUserLogin, ActorID: 7})
	require.NoError(t, err)
	_, err = c.Record(ctx, &RecordActivityParams{Type: types.EventFileRename, FileID: 42})
	require.NoError(t, err)

	all, total, err := c.List(ctx, &ListActivityArgs{PaginationArgs: PaginationArgs{Page: 0, PageSize: 10}})
	require.NoError(t, err)
	require.Equal(t, 3, total)
	require.Len(t, all, 3)
	// Newest first.
	require.Equal(t, types.EventFileRename, all[0].Type)

	byFile, total, err := c.List(ctx, &ListActivityArgs{
		PaginationArgs: PaginationArgs{Page: 0, PageSize: 10},
		FileID:         42,
	})
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, byFile, 2)

	login := types.EventUserLogin
	byType, total, err := c.List(ctx, &ListActivityArgs{
		PaginationArgs: PaginationArgs{Page: 0, PageSize: 10},
		Type:           &login,
	})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Equal(t, types.EventUserLogin, byType[0].Type)
	require.Equal(t, "10.0.0.1", byFile[1].IP)
}

func TestActivityDeleteBefore(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	defer client.Close()
	c := NewActivityClient(client, conf.SQLiteDB)
	ctx := context.Background()

	client.ActivityEvent.Create().SetType(types.EventUserLogin).
		SetCreatedAt(time.Now().AddDate(0, 0, -10)).SaveX(ctx)
	client.ActivityEvent.Create().SetType(types.EventFileCreate).SaveX(ctx)

	n, err := c.DeleteBefore(ctx, time.Now().AddDate(0, 0, -5))
	require.NoError(t, err)
	require.Equal(t, 1, n)

	_, total, err := c.List(ctx, &ListActivityArgs{PaginationArgs: PaginationArgs{Page: 0, PageSize: 10}})
	require.NoError(t, err)
	require.Equal(t, 1, total)
}
