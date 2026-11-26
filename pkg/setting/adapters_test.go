package setting

import (
	"context"
	"database/sql"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/cloudreve/Cloudreve/v4/ent"
	_ "github.com/cloudreve/Cloudreve/v4/ent/runtime"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/stretchr/testify/require"
)

const transactionTestSetting = "transaction_test"

func newSettingTransactionFixture(t *testing.T) (inventory.SettingClient, *cache.MemoStore) {
	t.Helper()
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "settings.db"))
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.SQLite, db)))
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	require.NoError(t, client.Schema.Create(context.Background()))
	_, err = client.Setting.Create().SetName(transactionTestSetting).SetValue("before").Save(context.Background())
	require.NoError(t, err)
	kv := cache.NewMemoStore("", logging.NewConsoleLogger(logging.LevelError))
	return inventory.NewSettingClient(client, kv), kv
}

func TestSettingAdaptersUseTransactionWithoutSharingUncommittedValues(t *testing.T) {
	for _, warm := range []bool{false, true} {
		for _, commit := range []bool{false, true} {
			t.Run("warm="+strconv.FormatBool(warm)+"/commit="+strconv.FormatBool(commit), func(t *testing.T) {
				settings, kv := newSettingTransactionFixture(t)
				store := NewKvSettingStore(kv, NewDbSettingStore(settings, nil))
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if warm {
					require.Equal(t, "before", store.Get(ctx, transactionTestSetting, "default"))
				}
				writer, tx, txCtx, err := inventory.WithTx(ctx, settings)
				require.NoError(t, err)
				defer inventory.Rollback(tx)
				require.NoError(t, writer.Set(txCtx, map[string]string{transactionTestSetting: "after"}))
				require.Equal(t, "after", store.Get(txCtx, transactionTestSetting, "default"))
				require.NoError(t, ctx.Err())
				value, exists := kv.Get(KvSettingPrefix + transactionTestSetting)
				require.Equal(t, warm, exists)
				if warm {
					require.Equal(t, "before", value)
				}

				expected := "before"
				if commit {
					require.NoError(t, inventory.Commit(tx))
					expected = "after"
				} else {
					require.NoError(t, inventory.Rollback(tx))
				}
				value, err = settings.Get(ctx, transactionTestSetting)
				require.NoError(t, err)
				require.Equal(t, expected, value)
			})
		}
	}
}
