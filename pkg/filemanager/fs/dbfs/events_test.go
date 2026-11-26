package dbfs

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
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth/requestinfo"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/eventhub"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"
)

type fileEventTestSettings struct{ setting.Provider }

func (fileEventTestSettings) EventHubMaxOfflineDuration(context.Context) time.Duration {
	return time.Hour
}
func (fileEventTestSettings) EventHubDebounceDelay(context.Context) time.Duration { return time.Hour }

type fileEventFixture struct {
	fs     *DBFS
	users  inventory.UserClient
	owner  *ent.User
	hub    eventhub.EventHub
	root   *File
	client *ent.Client
}

func newFileEventFixture(t *testing.T) *fileEventFixture {
	t.Helper()
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "events.db"))
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.SQLite, db)))
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	require.NoError(t, client.Schema.Create(ctx))
	group, err := client.Group.Create().SetName("test").SetPermissions(&boolset.BooleanSet{}).Save(ctx)
	require.NoError(t, err)
	owner, err := client.User.Create().SetEmail("owner@example.com").SetNick("owner").SetGroupID(group.ID).Save(ctx)
	require.NoError(t, err)
	owner.SetGroup(group)
	users := inventory.NewUserClient(client)
	hub := eventhub.NewEventHub(users, inventory.NewFsEventClient(client, conf.SQLiteDB), fileEventTestSettings{})
	t.Cleanup(hub.Close)
	encoder, err := hashid.New("file-events-test")
	require.NoError(t, err)
	rootURI, err := fs.NewUriFromString("cloudreve://my/root")
	require.NoError(t, err)
	root := newFile(nil, &ent.File{ID: 100, Name: "root", OwnerID: owner.ID, Type: int(types.FileTypeFolder)})
	root.Path = [2]*fs.URI{rootURI, rootURI}
	root.OwnerModel = owner
	t.Cleanup(root.Recycle)
	f := &DBFS{eventHub: hub, hasher: encoder, l: logging.NewConsoleLogger(logging.LevelError)}
	return &fileEventFixture{fs: f, users: users, owner: owner, hub: hub, root: root, client: client}
}

func (f *fileEventFixture) subscribe(t *testing.T, u *ent.User, reconnect bool) eventhub.Subscriber {
	t.Helper()
	ctx := context.WithValue(context.Background(), inventory.UserCtx{}, u)
	ctx = context.WithValue(ctx, logging.LoggerCtx{}, f.fs.l)
	id := uuid.Must(uuid.NewV4()).String()
	_, _, err := f.hub.Subscribe(ctx, f.root.ID(), id)
	require.NoError(t, err)
	if reconnect {
		f.hub.Unsubscribe(ctx, f.root.ID(), id)
		_, resumed, err := f.hub.Subscribe(ctx, f.root.ID(), id)
		require.NoError(t, err)
		require.True(t, resumed)
	}
	for _, sub := range f.hub.GetSubscribers(ctx, f.root.ID()) {
		if sub.ID() == id {
			return sub
		}
	}
	t.Fatal("subscriber not found")
	return nil
}

func TestFileEventsWaitForOutermostCommit(t *testing.T) {
	for _, outcome := range []string{"commit", "rollback", "failed commit"} {
		t.Run(outcome, func(t *testing.T) {
			f := newFileEventFixture(t)
			subscriber := f.subscribe(t, f.owner, true)
			file := newFile(f.root, &ent.File{ID: 200, Name: "original.txt", OwnerID: f.owner.ID, Type: int(types.FileTypeFile)})
			destination := newFile(f.root, &ent.File{ID: 300, Name: "destination", OwnerID: f.owner.ID, Type: int(types.FileTypeFolder)})
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			users, outer, txCtx, err := inventory.WithTx(ctx, f.users)
			require.NoError(t, err)
			defer inventory.Rollback(outer)
			_, err = users.UpdateNickname(txCtx, f.owner, "pending")
			require.NoError(t, err)
			_, nested, nestedCtx, err := inventory.WithTx(txCtx, f.users)
			require.NoError(t, err)

			f.fs.emitFileCreated(nestedCtx, file)
			f.fs.emitFileModified(nestedCtx, file)
			f.fs.emitFileRenamed(nestedCtx, file, "renamed.txt")
			f.fs.emitFileMoved(nestedCtx, file, destination)
			f.fs.emitFileDeleted(nestedCtx, file)
			require.NoError(t, ctx.Err(), "event collection must not wait for another SQLite connection")
			require.Empty(t, subscriber.Buffer())
			require.NoError(t, inventory.Commit(nested))
			require.Empty(t, subscriber.Buffer())

			// Rename replaces and recycles the original File before an enclosing commit.
			file.Replace(&ent.File{ID: 200, Name: "renamed.txt", OwnerID: f.owner.ID, Type: int(types.FileTypeFile)})
			switch outcome {
			case "commit":
				require.NoError(t, inventory.CommitWithStorageDiff(txCtx, outer, f.fs.l, f.users))
			case "rollback":
				require.NoError(t, inventory.Rollback(outer))
			case "failed commit":
				cancel()
				require.Error(t, inventory.Commit(outer))
			}
			if outcome != "commit" {
				require.Empty(t, subscriber.Buffer())
				return
			}
			fileID := hashid.EncodeFileID(f.fs.hasher, 200)
			require.Equal(t, []*eventhub.Event{
				{Type: eventhub.EventTypeCreate, FileID: fileID, From: "/original.txt"},
				{Type: eventhub.EventTypeModify, FileID: fileID, From: "/original.txt"},
				{Type: eventhub.EventTypeRename, FileID: fileID, From: "/original.txt", To: "/renamed.txt"},
				{Type: eventhub.EventTypeRename, FileID: fileID, From: "/original.txt", To: "/destination/original.txt"},
				{Type: eventhub.EventTypeDelete, FileID: fileID, From: "/original.txt"},
			}, subscriber.Buffer())
			owner, err := subscriber.Owner(ctx)
			require.NoError(t, err)
			require.Equal(t, "pending", owner.Nick)
		})
	}
}

func TestFileEventsPreserveOriginFiltering(t *testing.T) {
	f := newFileEventFixture(t)
	ownerSub := f.subscribe(t, f.owner, false)
	viewer, err := f.client.User.Create().SetEmail("viewer@example.com").SetNick("viewer").SetGroupID(f.owner.GroupUsers).Save(context.Background())
	require.NoError(t, err)
	viewer.SetGroup(f.owner.Edges.Group)
	viewerSub := f.subscribe(t, viewer, false)
	file := newFile(f.root, &ent.File{ID: 200, Name: "file.txt", OwnerID: f.owner.ID, Type: int(types.FileTypeFile)})

	ctx := context.WithValue(context.Background(), requestinfo.RequestInfoCtx{}, &requestinfo.RequestInfo{ClientID: ownerSub.ID()})
	f.fs.emitFileCreated(ctx, file)
	require.Empty(t, ownerSub.Buffer(), "the originating subscriber must still be excluded")
	require.Len(t, viewerSub.Buffer(), 1, "other subscribers must still receive events")

	f.fs.emitFileCreated(context.Background(), file)
	require.Len(t, ownerSub.Buffer(), 1)
	require.Len(t, viewerSub.Buffer(), 2)
}
