package manager

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/stretchr/testify/require"
)

// TestRecycleEntitiesForceBypassesBrokenPolicy covers upstream #2909: when a
// storage policy is deleted or too broken to initialize a driver, its blob
// rows used to be undeletable even with force=true, which also deadlocked
// removal of the policy itself.
func TestRecycleEntitiesForceBypassesBrokenPolicy(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	logger := logging.NewConsoleLogger(logging.LevelError)
	cfg, err := conf.NewIniConfigProvider(filepath.Join(t.TempDir(), "conf.ini"), logger)
	require.NoError(t, err)
	hasher, err := hashid.New("test-salt")
	require.NoError(t, err)

	dep := dependency.NewDependency(
		dependency.WithDbClient(client),
		dependency.WithConfigProvider(cfg),
		dependency.WithKV(cache.NewMemoStore("", logger)),
		dependency.WithLogger(logger),
		dependency.WithHashIDEncoder(hasher),
	)
	ctx := context.WithValue(context.Background(), dependency.DepCtx{}, dep)

	group := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	user := client.User.Create().SetEmail("u@example.com").SetNick("u").SetGroup(group).SaveX(ctx)

	// Unknown policy type - GetStorageDriver rejects it with
	// ErrUnknownPolicyType, simulating a misconfigured policy.
	brokenPolicy := client.StoragePolicy.Create().
		SetName("broken").
		SetType("bogus").
		SetAccessKey("x").
		SetSecretKey("x").
		SetMaxSize(1).
		SetDirNameRule("x").
		SetFileNameRule("x").
		SaveX(ctx)

	newStaleEntity := func() int {
		return client.Entity.Create().
			SetType(0).
			SetSource("uploads/blob.bin").
			SetSize(123).
			SetStoragePolicyEntities(brokenPolicy.ID).
			SetReferenceCount(0).
			SetCreatedBy(user.ID).
			SaveX(ctx).ID
	}

	fm := NewFileManager(dep, user)

	t.Run("without force the row is retained and error returned", func(t *testing.T) {
		id := newStaleEntity()
		err := fm.RecycleEntities(ctx, false, id)
		require.Error(t, err)
		require.NotNil(t, client.Entity.GetX(ctx, id))
	})

	t.Run("with force the row is dropped despite driver failure", func(t *testing.T) {
		id := newStaleEntity()
		err := fm.RecycleEntities(ctx, true, id)
		require.NoError(t, err)
		_, getErr := client.Entity.Get(ctx, id)
		require.Error(t, getErr)
	})
}
