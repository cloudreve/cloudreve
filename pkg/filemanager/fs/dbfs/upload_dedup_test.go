package dbfs

import (
	"context"
	"testing"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	entfile "github.com/cloudreve/Cloudreve/v4/ent/file"
	"github.com/cloudreve/Cloudreve/v4/ent/storagepolicy"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/eventhub"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/lock"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"
)

type dedupSettingProvider struct {
	setting.Provider
	scope string
}

func (p dedupSettingProvider) DBFS(context.Context) *setting.DBFS {
	return &setting.DBFS{DedupScope: p.scope, MaxPageSize: 200}
}

type stubEventHub struct{}

func (stubEventHub) Subscribe(context.Context, int, string) (chan *eventhub.Event, bool, error) {
	return nil, false, nil
}
func (stubEventHub) Unsubscribe(context.Context, int, string) {}
func (stubEventHub) GetSubscribers(context.Context, int) []eventhub.Subscriber {
	return nil
}
func (stubEventHub) Close() {}

func dedupUploadFixture(t *testing.T, client *ent.Client, scope string) (*ent.User, *ent.StoragePolicy, *DBFS) {
	t.Helper()
	ctx := context.Background()
	l := logging.NewConsoleLogger(logging.LevelError)
	hasher, err := hashid.New("dedup-test-salt")
	require.NoError(t, err)

	p := client.StoragePolicy.Create().SetName("local").SetType("local").
		SetStatus(storagepolicy.StatusActive).SetSettings(&types.PolicySetting{}).SaveX(ctx)
	group := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).
		SetMaxStorage(1 << 40).SetStoragePolicies(p).SaveX(ctx)
	u := client.User.Create().SetEmail("u@example.com").SetNick("u").SetGroup(group).SaveX(ctx)
	u.SetGroup(group)
	client.File.Create().SetName(inventory.RootFolderName).
		SetType(int(types.FileTypeFolder)).SetOwner(u).SaveX(ctx)

	f := &DBFS{
		user:                u,
		navigators:          make(map[string]Navigator),
		fileClient:          inventory.NewFileClient(client, conf.SQLiteDB, hasher),
		userClient:          inventory.NewUserClient(client),
		storagePolicyClient: inventory.NewStoragePolicyClient(client, nil),
		settingClient:       dedupSettingProvider{scope: scope},
		hasher:              hasher,
		l:                   l,
		ls:                  lock.NewMemLS(hasher, l),
		eventHub:            stubEventHub{},
	}
	return u, p, f
}

const dedupTestHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

func uploadReq(t *testing.T, hasher hashid.Encoder, u *ent.User, name string, size int64, hash string) *fs.UploadRequest {
	t.Helper()
	uri, err := fs.NewUriFromString(fs.NewMyUri(hashid.EncodeUserID(hasher, u.ID)) + "/" + name)
	require.NoError(t, err)
	return &fs.UploadRequest{
		Props: &fs.UploadProps{
			Uri:             uri,
			Size:            size,
			Hash:            hash,
			UploadSessionID: uuid.Must(uuid.NewV4()).String(),
			ExpireAt:        time.Now().Add(time.Hour),
		},
	}
}

func seedCompletedEntity(t *testing.T, client *ent.Client, u *ent.User, p *ent.StoragePolicy, hash string, size int64) *ent.Entity {
	t.Helper()
	return client.Entity.Create().
		SetType(int(types.EntityTypeVersion)).
		SetSource("cloudreve/data/" + hash).
		SetSize(size).
		SetHash(hash).
		SetReferenceCount(1).
		SetCreatedBy(u.ID).
		SetStoragePolicyEntities(p.ID).
		SaveX(context.Background())
}

func TestPrepareUploadRapid(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	u, p, f := dedupUploadFixture(t, client, "owner")
	existing := seedCompletedEntity(t, client, u, p, dedupTestHash, 1024)

	// Normal upload without hash -> transfer session
	s, err := f.PrepareUpload(ctx, uploadReq(t, f.hasher, u, "first.txt", 2048, ""))
	require.NoError(t, err)
	require.False(t, s.Props.RapidUploaded)
	require.NotEmpty(t, s.Props.UploadSessionID)

	// Same hash + size -> rapid session, no transfer required
	s, err = f.PrepareUpload(ctx, uploadReq(t, f.hasher, u, "copy.txt", 1024, dedupTestHash))
	require.NoError(t, err)
	require.True(t, s.Props.RapidUploaded)
	require.Equal(t, existing.ID, s.EntityID)

	// Entity refcount bumped, new file linked
	require.Equal(t, 2, client.Entity.GetX(ctx, existing.ID).ReferenceCount)
	newFile := client.File.Query().Where(entfile.Name("copy.txt")).OnlyX(ctx)
	require.Equal(t, existing.ID, newFile.PrimaryEntity)
	require.Equal(t, int64(1024), newFile.Size)

	// Size mismatch -> normal session
	s, err = f.PrepareUpload(ctx, uploadReq(t, f.hasher, u, "bigger.txt", 4096, dedupTestHash))
	require.NoError(t, err)
	require.False(t, s.Props.RapidUploaded)
}

func TestPrepareUploadRapidScopeOff(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	u, p, f := dedupUploadFixture(t, client, "off")
	seedCompletedEntity(t, client, u, p, dedupTestHash, 1024)

	s, err := f.PrepareUpload(ctx, uploadReq(t, f.hasher, u, "copy.txt", 1024, dedupTestHash))
	require.NoError(t, err)
	require.False(t, s.Props.RapidUploaded)
}
