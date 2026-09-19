package inventory

import (
	"context"
	rawsql "database/sql"
	"path/filepath"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/stretchr/testify/require"
)

// legacyV3DB builds a minimal v3-shaped sqlite database: users bind their
// group via a nullable group_id column, and groupless users carry NULL.
func legacyV3DB(t *testing.T, dbPath string) {
	t.Helper()
	raw, err := rawsql.Open("sqlite3", "file:"+dbPath)
	require.NoError(t, err)
	defer raw.Close()

	stmts := []string{
		`CREATE TABLE groups (id INTEGER PRIMARY KEY, created_at DATETIME, updated_at DATETIME, name TEXT, permissions BLOB)`,
		`INSERT INTO groups (id, created_at, updated_at, name, permissions) VALUES
			(1, '2024-01-01 00:00:00', '2024-01-01 00:00:00', 'Admin', X'00'),
			(2, '2024-01-01 00:00:00', '2024-01-01 00:00:00', 'User', X'00')`,
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			email VARCHAR(100),
			nick VARCHAR(100),
			password VARCHAR(255),
			status VARCHAR(20),
			storage BIGINT,
			group_id INTEGER
		)`,
		`INSERT INTO users (created_at, updated_at, email, nick, status, storage, group_id) VALUES
			('2024-01-01 00:00:00', '2024-01-01 00:00:00', 'admin@x', 'admin', 'active', 0, 1),
			('2024-01-01 00:00:00', '2024-01-01 00:00:00', 'user@x', 'user', 'active', 0, 2),
			('2024-01-01 00:00:00', '2024-01-01 00:00:00', 'groupless@x', 'groupless', 'active', 0, NULL)`,
	}
	for _, q := range stmts {
		_, err := raw.Exec(q)
		require.NoError(t, err, q)
	}
}

// TestRepairLegacyUserGroupColumn reproduces upstream #2934: upgrading a v3
// database fails inside ent's copy-migration because group_users is NOT NULL
// in v4 while v3 rows can be groupless. The repair backfills the column so
// Schema.Create succeeds.
func TestRepairLegacyUserGroupColumn(t *testing.T) {
	ctx := context.Background()
	l := logging.NewConsoleLogger(logging.LevelError)

	openClient := func(t *testing.T) *ent.Client {
		dbPath := filepath.Join(t.TempDir(), "legacy.db")
		legacyV3DB(t, dbPath)
		client, err := ent.Open("sqlite3", "file:"+dbPath)
		require.NoError(t, err)
		return client
	}

	t.Run("schema create fails without repair", func(t *testing.T) {
		client := openClient(t)
		defer client.Close()
		err := client.Schema.Create(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "group_users")
	})

	t.Run("repair backfills and migration succeeds", func(t *testing.T) {
		client := openClient(t)
		defer client.Close()

		repairLegacyUserGroupColumn(l, client, ctx)
		require.NoError(t, client.Schema.Create(ctx))

		rows, err := client.QueryContext(ctx, `SELECT id, group_users FROM users ORDER BY id`)
		require.NoError(t, err)
		defer rows.Close()

		got := map[int]int{}
		for rows.Next() {
			var id, gid int
			require.NoError(t, rows.Scan(&id, &gid))
			got[id] = gid
		}
		require.NoError(t, rows.Err())

		// group_id bindings preserved; groupless user lands on the
		// non-admin group (id 2), not the admin group.
		require.Equal(t, map[int]int{1: 1, 2: 2, 3: 2}, got)
	})
}
