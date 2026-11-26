package inventory

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/cloudreve/Cloudreve/v4/ent"
	_ "github.com/cloudreve/Cloudreve/v4/ent/runtime"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/stretchr/testify/require"
)

func TestAfterCommitUsesOutermostTransaction(t *testing.T) {
	for _, outcome := range []string{"commit", "storage diff", "rollback", "failed commit"} {
		t.Run(outcome, func(t *testing.T) {
			db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "transactions.db"))
			require.NoError(t, err)
			db.SetMaxOpenConns(1)
			client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.SQLite, db)))
			t.Cleanup(func() { require.NoError(t, client.Close()) })
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			require.NoError(t, client.Schema.Create(ctx))
			group, err := client.Group.Create().SetName("test").SetPermissions(&boolset.BooleanSet{}).Save(ctx)
			require.NoError(t, err)
			u, err := client.User.Create().SetEmail("owner@example.com").SetNick("before").SetGroupID(group.ID).Save(ctx)
			require.NoError(t, err)
			users := NewUserClient(client)

			txUsers, outer, txCtx, err := WithTx(ctx, users)
			require.NoError(t, err)
			defer Rollback(outer)
			_, err = txUsers.UpdateNickname(txCtx, u, "committed")
			require.NoError(t, err)

			var calls, completions []string
			AfterCompletion(txCtx, func() {
				require.False(t, InTransaction(txCtx))
				readCtx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				stored, err := users.GetByID(readCtx, u.ID)
				require.NoError(t, err, "completion callbacks must run after the connection is released")
				if outcome == "commit" || outcome == "storage diff" {
					require.Equal(t, "committed", stored.Nick)
				} else {
					require.Equal(t, "before", stored.Nick)
				}
				completions = append(completions, "outer")
			})
			AfterCommit(txCtx, func() {
				require.Equal(t, []string{"outer", "nested"}, completions)
				calls = append(calls, "outer")
				readCtx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				stored, err := users.GetByID(readCtx, u.ID)
				require.NoError(t, err, "callbacks must not wait for the transaction's only connection")
				require.Equal(t, "committed", stored.Nick)
				if outcome == "storage diff" {
					require.EqualValues(t, 7, stored.Storage)
				}
			})
			_, nested, _, err := WithTx(txCtx, users)
			require.NoError(t, err)
			nestedCtx := context.WithValue(txCtx, TxCtx{}, nested)
			require.True(t, InTransaction(nestedCtx))
			AfterCompletion(nestedCtx, func() {
				completions = append(completions, "nested")
			})
			AfterCommit(nestedCtx, func() {
				_, inherited := InheritTx(nestedCtx, users)
				require.Nil(t, inherited, "completed parent transactions must not be inherited")
				calls = append(calls, "nested")
			})
			require.NoError(t, Commit(nested))
			require.Empty(t, calls)
			require.Empty(t, completions)

			switch outcome {
			case "commit":
				require.NoError(t, Commit(outer))
			case "storage diff":
				nested.AppendStorageDiff(StorageDiff{u.ID: 7})
				require.NoError(t, CommitWithStorageDiff(txCtx, outer, logging.NewConsoleLogger(logging.LevelError), users))
			case "rollback":
				require.NoError(t, Rollback(outer))
			case "failed commit":
				cancel()
				require.Error(t, Commit(outer))
			}
			AfterCommit(txCtx, func() { calls = append(calls, "late") })
			AfterCompletion(txCtx, func() { completions = append(completions, "late") })
			require.Equal(t, []string{"outer", "nested", "late"}, completions)
			require.False(t, InTransaction(nestedCtx))
			if outcome == "commit" || outcome == "storage diff" {
				require.Equal(t, []string{"outer", "nested", "late"}, calls)
			} else {
				require.Empty(t, calls)
			}
			require.Empty(t, outer.afterCommit)
			require.Empty(t, outer.afterCompletion)
		})
	}

	called := false
	AfterCommit(context.Background(), func() { called = true })
	require.True(t, called)
	completed := false
	AfterCompletion(context.Background(), func() { completed = true })
	require.True(t, completed)
}
