package dbfs

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
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
)

type receiveTestSettings struct{}

func (receiveTestSettings) Get(_ context.Context, _ string, fallback any) any { return fallback }

type receiveTestHub struct{ eventhub.EventHub }

func (receiveTestHub) GetSubscribers(context.Context, int) []eventhub.Subscriber { return nil }

type receiveFixture struct {
	ctx    context.Context
	db     *ent.Client
	fs     *DBFS
	source *ent.File
	blob   *ent.Entity
	owner  *ent.User
	admin  *ent.User
	dst    *fs.URI
}

func newReceiveFixture(t *testing.T) *receiveFixture {
	t.Helper()
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "receive.db")+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	c := enttest.NewClient(t, enttest.WithOptions(ent.Driver(entsql.OpenDB(dialect.SQLite, db))))
	t.Cleanup(func() { _ = c.Close() })
	hasher, err := hashid.New("receive-test")
	if err != nil {
		t.Fatal(err)
	}
	adminPerms := &boolset.BooleanSet{}
	boolset.Sets(map[types.GroupPermission]bool{types.GroupPermissionIsAdmin: true}, adminPerms)
	groupA := c.Group.Create().SetName("crawler").SetPermissions(&boolset.BooleanSet{}).SetMaxStorage(4096).SaveX(ctx)
	groupB := c.Group.Create().SetName("administrator").SetPermissions(adminPerms).SetMaxStorage(4096).SaveX(ctx)
	a := c.User.Create().SetEmail("crawler@example.com").SetNick("crawler").SetGroup(groupA).SetStorage(5).SaveX(ctx)
	b := c.User.Create().SetEmail("admin@example.com").SetNick("admin").SetGroup(groupB).SaveX(ctx)
	a = c.User.Query().Where(user.ID(a.ID)).WithGroup().OnlyX(ctx)
	b = c.User.Query().Where(user.ID(b.ID)).WithGroup().OnlyX(ctx)
	rootA := c.File.Create().SetOwner(a).SetName("").SetType(int(types.FileTypeFolder)).SaveX(ctx)
	rootB := c.File.Create().SetOwner(b).SetName("").SetType(int(types.FileTypeFolder)).SaveX(ctx)
	c.File.Create().SetOwner(b).SetParent(rootB).SetName("inbox").SetType(int(types.FileTypeFolder)).SaveX(ctx)
	policy := c.StoragePolicy.Create().SetName("local").SetType(types.PolicyTypeLocal).SetSettings(&types.PolicySetting{}).SaveX(ctx)
	physicalPath := filepath.Join(t.TempDir(), "blob.txt")
	if err := os.WriteFile(physicalPath, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	blob := c.Entity.Create().SetType(int(types.EntityTypeVersion)).SetSource(physicalPath).SetSize(5).SetStoragePolicy(policy).SaveX(ctx)
	source := c.File.Create().SetOwner(a).SetParent(rootA).SetName("X.txt").SetType(int(types.FileTypeFile)).SetSize(5).SetPrimaryEntity(blob.ID).AddEntities(blob).SaveX(ctx)
	c.Metadata.Create().SetFile(source).SetName("tag").SetValue("crawler").SetIsPublic(true).SaveX(ctx)
	logger := logging.NewConsoleLogger(logging.LevelError)
	f := &DBFS{
		user: b, navigators: make(map[string]Navigator),
		fileClient: inventory.NewFileClient(c, conf.SQLiteDB, hasher), userClient: inventory.NewUserClient(c),
		l: logger, ls: lock.NewMemLS(hasher, logger), hasher: hasher,
		settingClient: setting.NewProvider(receiveTestSettings{}), eventHub: receiveTestHub{},
	}
	t.Cleanup(f.Recycle)
	dst, err := fs.NewUriFromString("cloudreve://my/inbox")
	if err != nil {
		t.Fatal(err)
	}
	return &receiveFixture{ctx: ctx, db: c, fs: f, source: source, blob: blob, owner: a, admin: b, dst: dst}
}

func TestReceiveFilePreservesBlobAfterSourceDeletion(t *testing.T) {
	x := newReceiveFixture(t)
	received, _, err := x.fs.ReceiveFile(x.ctx, x.source.ID, x.dst, "")
	if err != nil {
		t.Fatal(err)
	}
	if received.ID() == x.source.ID || received.OwnerID() != x.admin.ID || received.PrimaryEntityID() != x.blob.ID {
		t.Fatal("received file must have a new ID and owner, sharing the original blob")
	}
	if got := x.db.Entity.GetX(x.ctx, x.blob.ID).ReferenceCount; got != 2 {
		t.Fatalf("references = %d, want 2", got)
	}
	if got := x.db.User.GetX(x.ctx, x.admin.ID).Storage; got != 5 {
		t.Fatalf("admin storage = %d, want 5", got)
	}
	if received.Metadata()["tag"] != "crawler" {
		t.Fatal("file metadata was not copied")
	}
	fc, tx, ctx, err := inventory.WithTx(x.ctx, x.fs.fileClient)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = inventory.Rollback(tx) }()
	model, err := fc.GetByID(context.WithValue(ctx, inventory.LoadFileEntity{}, true), x.source.ID)
	if err != nil {
		t.Fatal(err)
	}
	stale, diff, err := fc.Delete(ctx, []*ent.File{model}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 0 {
		t.Fatal("shared blob incorrectly marked for recycling")
	}
	tx.AppendStorageDiff(diff)
	if err := inventory.CommitWithStorageDiff(ctx, tx, x.fs.l, x.fs.userClient); err != nil {
		t.Fatal(err)
	}
	if _, err := x.db.File.Get(x.ctx, received.ID()); err != nil {
		t.Fatal("received file disappeared:", err)
	}
	if got := x.db.Entity.GetX(x.ctx, x.blob.ID).ReferenceCount; got != 1 {
		t.Fatalf("references = %d, want 1", got)
	}
	data, err := os.ReadFile(x.blob.Source)
	if err != nil || string(data) != "hello" {
		t.Fatal("physical contents were changed or removed", err)
	}
}

func TestReceiveFileRejectsNonAdministrator(t *testing.T) {
	x := newReceiveFixture(t)
	x.fs.user = x.owner
	if _, _, err := x.fs.ReceiveFile(x.ctx, x.source.ID, x.dst, ""); err == nil {
		t.Fatal("ordinary user accepted")
	}
	if got := x.db.Entity.GetX(x.ctx, x.blob.ID).ReferenceCount; got != 1 {
		t.Fatal("denied request changed references")
	}
}

func TestReceiveFileRejectsOtherUsersDestination(t *testing.T) {
	x := newReceiveFixture(t)
	dst, _ := fs.NewUriFromString(fmt.Sprintf("cloudreve://%s@my", hashid.EncodeUserID(x.fs.hasher, x.owner.ID)))
	if _, _, err := x.fs.ReceiveFile(x.ctx, x.source.ID, dst, ""); err == nil {
		t.Fatal("other user's destination accepted")
	}
}

func TestReceiveFileNameConflictRollsBackQuotaAndReferences(t *testing.T) {
	x := newReceiveFixture(t)
	if _, _, err := x.fs.ReceiveFile(x.ctx, x.source.ID, x.dst, ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := x.fs.ReceiveFile(x.ctx, x.source.ID, x.dst, ""); err == nil {
		t.Fatal("name conflict accepted")
	}
	if got := x.db.Entity.GetX(x.ctx, x.blob.ID).ReferenceCount; got != 2 {
		t.Fatalf("references = %d, want 2", got)
	}
	if got := x.db.User.GetX(x.ctx, x.admin.ID).Storage; got != 5 {
		t.Fatalf("quota was not rolled back: %d", got)
	}
}

func TestReceiveFileRejectsIncompleteAndDeletedSources(t *testing.T) {
	for _, state := range []string{"uploading", "trash", "folder", "symbolic"} {
		t.Run(state, func(t *testing.T) {
			x := newReceiveFixture(t)
			switch state {
			case "uploading":
				x.db.Entity.UpdateOne(x.blob).SetUploadSessionID(uuid.Must(uuid.NewV4())).SaveX(x.ctx)
			case "trash":
				x.db.File.UpdateOne(x.source).ClearParent().SetName("trash-entry").SaveX(x.ctx)
			case "folder":
				x.db.File.UpdateOne(x.source).SetType(int(types.FileTypeFolder)).SaveX(x.ctx)
			case "symbolic":
				x.db.File.UpdateOne(x.source).SetIsSymbolic(true).SaveX(x.ctx)
			}
			if _, _, err := x.fs.ReceiveFile(x.ctx, x.source.ID, x.dst, ""); err == nil {
				t.Fatal("invalid source accepted")
			}
			if got := x.db.User.GetX(x.ctx, x.admin.ID).Storage; got != 0 {
				t.Fatal("failed receive changed quota")
			}
		})
	}
}

func TestReceiveFileInsufficientQuota(t *testing.T) {
	x := newReceiveFixture(t)
	x.fs.user.Edges.Group.MaxStorage = 4
	if _, _, err := x.fs.ReceiveFile(x.ctx, x.source.ID, x.dst, ""); err == nil {
		t.Fatal("insufficient quota accepted")
	}
	if got := x.db.Entity.GetX(x.ctx, x.blob.ID).ReferenceCount; got != 1 {
		t.Fatal("failed receive changed references")
	}
	if got := x.db.User.GetX(x.ctx, x.admin.ID).Storage; got != 0 {
		t.Fatal("failed receive changed quota")
	}
}

func TestReceiveFileUsesRequestedNameWithoutChangingSource(t *testing.T) {
	x := newReceiveFixture(t)
	name := "2026-10-1_2026-10-31_销售主题分析.xlsx"
	received, _, err := x.fs.ReceiveFile(x.ctx, x.source.ID, x.dst, name)
	if err != nil {
		t.Fatal(err)
	}
	if received.Name() != name || received.Uri(false).Name() != name {
		t.Fatal("received file did not use requested business name")
	}
	if x.db.File.GetX(x.ctx, x.source.ID).Name != "X.txt" {
		t.Fatal("receive renamed the crawler source")
	}
	if received.PrimaryEntityID() != x.blob.ID {
		t.Fatal("receive changed the physical blob")
	}
}

func TestReceiveFileConflictsOnRequestedNameOnly(t *testing.T) {
	x := newReceiveFixture(t)
	// A file using the source name does not block a different business name.
	if _, _, err := x.fs.ReceiveFile(x.ctx, x.source.ID, x.dst, ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := x.fs.ReceiveFile(x.ctx, x.source.ID, x.dst, "business.xlsx"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := x.fs.ReceiveFile(x.ctx, x.source.ID, x.dst, "business.xlsx"); err == nil {
		t.Fatal("duplicate requested name accepted")
	}
	if got := x.db.Entity.GetX(x.ctx, x.blob.ID).ReferenceCount; got != 3 {
		t.Fatalf("references = %d, want 3 after rejected duplicate", got)
	}
	if got := x.db.User.GetX(x.ctx, x.admin.ID).Storage; got != 10 {
		t.Fatalf("storage = %d, want 10 after rejected duplicate", got)
	}
}

func TestReceiveFileRejectsInvalidRequestedNames(t *testing.T) {
	for _, name := range []string{"..", "../escape.xlsx", "a/b.xlsx", "a\\b.xlsx", "bad\x00.xlsx", "bad\n.xlsx", "bad\r.xlsx"} {
		t.Run(fmt.Sprintf("%q", name), func(t *testing.T) {
			x := newReceiveFixture(t)
			if _, _, err := x.fs.ReceiveFile(x.ctx, x.source.ID, x.dst, name); err == nil {
				t.Fatal("invalid requested name accepted")
			}
			if x.db.User.GetX(x.ctx, x.admin.ID).Storage != 0 || x.db.Entity.GetX(x.ctx, x.blob.ID).ReferenceCount != 1 {
				t.Fatal("invalid requested name changed quota or references")
			}
		})
	}
}
