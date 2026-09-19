package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/entity"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/ent/storagepolicy"
	"github.com/cloudreve/Cloudreve/v4/ent/task"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/stretchr/testify/require"
)

func newRelocateTestDep(t *testing.T, client *ent.Client) (context.Context, dependency.Dep) {
	logger := logging.NewConsoleLogger(logging.LevelError)
	cfg, err := conf.NewIniConfigProvider(t.TempDir()+"/conf.ini", logger)
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
	return ctx, dep
}

func relocateTaskWithState(t *testing.T, state *RelocateTaskState) *RelocateTask {
	stateBytes, err := json.Marshal(state)
	require.NoError(t, err)
	return &RelocateTask{
		DBTask: &queue.DBTask{
			Task: &ent.Task{
				Type:         queue.RelocateTaskType,
				PrivateState: string(stateBytes),
				PublicState:  &types.TaskPublicState{},
			},
		},
	}
}

func TestNewRelocateTaskSortsIDs(t *testing.T) {
	tk, err := NewRelocateTask(context.Background(), []int{9, 1, 5}, 3)
	require.NoError(t, err)

	rt := tk.(*RelocateTask)
	state := &RelocateTaskState{}
	require.NoError(t, json.Unmarshal([]byte(rt.Task.PrivateState), state))
	require.Equal(t, []int{1, 5, 9}, state.EntityIDs)
	require.Equal(t, 3, state.DstPolicyID)
	require.Equal(t, queue.RelocateTaskType, rt.Task.Type)
}

func TestRelocateRejectsIdenticalPolicies(t *testing.T) {
	m := relocateTaskWithState(t, &RelocateTaskState{SrcPolicyID: 4, DstPolicyID: 4})
	m.state = &RelocateTaskState{SrcPolicyID: 4, DstPolicyID: 4}

	status, err := m.relocate(context.Background(), nil)
	require.Error(t, err)
	require.Equal(t, task.StatusError, status)
}

func TestRelocatePendingExplicitList(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx, dep := newRelocateTestDep(t, client)

	p := client.StoragePolicy.Create().SetName("src").SetType("local").
		SetStatus(storagepolicy.StatusActive).SaveX(ctx)
	mk := func(src string, size int64) int {
		return client.Entity.Create().SetType(0).SetSource(src).SetSize(size).
			SetStoragePolicyEntities(p.ID).SaveX(ctx).ID
	}
	a, b, c := mk("a", 1), mk("b", 2), mk("c", 3)

	m := relocateTaskWithState(t, &RelocateTaskState{
		EntityIDs:   []int{c, a, b},
		DstPolicyID: p.ID + 1,
	})
	m.state = &RelocateTaskState{EntityIDs: []int{a, b, c}, DstPolicyID: p.ID + 1}

	pending, err := m.pendingEntities(ctx, dep)
	require.NoError(t, err)
	require.Len(t, pending, 3)

	// Cursor resumes after the last processed ID.
	m.state.Cursor = b
	pending, err = m.pendingEntities(ctx, dep)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, c, pending[0].ID)
}

func TestRelocatePendingPolicyScope(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx, dep := newRelocateTestDep(t, client)

	src := client.StoragePolicy.Create().SetName("src").SetType("local").
		SetStatus(storagepolicy.StatusActive).SaveX(ctx)
	dst := client.StoragePolicy.Create().SetName("dst").SetType("local").
		SetStatus(storagepolicy.StatusActive).SaveX(ctx)

	stay := client.Entity.Create().SetType(0).SetSource("s").SetSize(1).
		SetStoragePolicyEntities(src.ID).SaveX(ctx)
	moved := client.Entity.Create().SetType(0).SetSource("m").SetSize(1).
		SetStoragePolicyEntities(dst.ID).SaveX(ctx)

	m := relocateTaskWithState(t, &RelocateTaskState{SrcPolicyID: src.ID, DstPolicyID: dst.ID})
	m.state = &RelocateTaskState{SrcPolicyID: src.ID, DstPolicyID: dst.ID}

	pending, err := m.pendingEntities(ctx, dep)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, stay.ID, pending[0].ID)

	// After relocation the entity leaves the source-policy scope.
	m.state.Cursor = stay.ID
	pending, err = m.pendingEntities(ctx, dep)
	require.NoError(t, err)
	require.Empty(t, pending)
	_ = moved
}

func TestRelocateProgressTotals(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx, dep := newRelocateTestDep(t, client)

	src := client.StoragePolicy.Create().SetName("src").SetType("local").
		SetStatus(storagepolicy.StatusActive).SaveX(ctx)
	other := client.StoragePolicy.Create().SetName("other").SetType("local").
		SetStatus(storagepolicy.StatusActive).SaveX(ctx)

	mk := func(policyID int, size int64) int {
		return client.Entity.Create().SetType(0).SetSource("s").SetSize(size).
			SetStoragePolicyEntities(policyID).SaveX(ctx).ID
	}
	a := mk(src.ID, 10)
	b := mk(src.ID, 20)
	mk(other.ID, 99)

	m := relocateTaskWithState(t, &RelocateTaskState{SrcPolicyID: src.ID, DstPolicyID: other.ID})
	m.state = &RelocateTaskState{SrcPolicyID: src.ID, DstPolicyID: other.ID}
	count, size := m.progressTotals(ctx, dep)
	require.Equal(t, int64(2), count)
	require.Equal(t, int64(30), size)

	m.state = &RelocateTaskState{EntityIDs: []int{a, b}, DstPolicyID: other.ID}
	count, size = m.progressTotals(ctx, dep)
	require.Equal(t, int64(2), count)
	require.Equal(t, int64(30), size)
}

func TestRelocateSavePathNamingRules(t *testing.T) {
	p := &ent.StoragePolicy{
		DirNameRule:  "vault/{uid}",
		FileNameRule: "{originname}",
	}
	e := &ent.Entity{Source: "uploads/blob.bin"}
	e.Edges.File = []*ent.File{{Name: "report.pdf"}}
	e.Edges.User = &ent.User{ID: 7}

	require.Equal(t, "vault/7/report.pdf", relocateSavePath(p, e))
}

func TestRelocateEntityUriOwnerScoped(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx, dep := newRelocateTestDep(t, client)
	_ = ctx

	e := &ent.Entity{Source: "uploads/blob.bin"}
	e.Edges.File = []*ent.File{{Name: "report.pdf"}}
	e.Edges.User = &ent.User{ID: 7}

	uri := relocateEntityUri(dep, e)
	require.NotNil(t, uri)
	require.Equal(t, "report.pdf", uri.Name())
	require.Equal(t, constants.FileSystemMy, uri.FileSystem())
	require.NotEmpty(t, uri.ID(""))
}

func TestRelocateSummarize(t *testing.T) {
	m := relocateTaskWithState(t, &RelocateTaskState{
		SrcPolicyID: 1,
		DstPolicyID: 2,
		Failed:      map[string]string{"5": "io error"},
	})
	s := m.Summarize(nil)
	require.NotNil(t, s)
	require.Equal(t, 2, s.Props[SummaryKeyDstPolicy])
	require.Equal(t, 1, s.Props[SummaryKeySrcPolicy])
	require.Equal(t, 1, s.Props[SummaryKeyRelocateFailed])
}

func TestRelocateEntitySkipsSamePolicy(t *testing.T) {
	m := relocateTaskWithState(t, &RelocateTaskState{DstPolicyID: 3})
	m.state = &RelocateTaskState{DstPolicyID: 3}
	m.progress = queue.Progresses{ProgressTypeRelocateCount: &queue.Progress{}}

	e := &ent.Entity{ID: 1, StoragePolicyEntities: 3}
	require.NoError(t, m.relocateEntity(context.Background(), nil, nil, nil, e))
	require.Equal(t, int64(1), m.progress[ProgressTypeRelocateCount].Current)
}

// TestRelocateEntityLocalEndToEnd streams a real blob between two local
// policies: content lands under the destination naming rules, the entity
// record swaps to the new policy, and the old blob is removed.
func TestRelocateEntityLocalEndToEnd(t *testing.T) {
	tmp := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmp))
	oldUseWd := util.UseWorkingDir
	util.UseWorkingDir = true
	t.Cleanup(func() {
		util.UseWorkingDir = oldUseWd
		require.NoError(t, os.Chdir(oldWd))
	})

	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx, dep := newRelocateTestDep(t, client)

	group := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	user := client.User.Create().SetEmail("u@example.com").SetNick("u").SetGroup(group).SaveX(ctx)

	srcPolicy := client.StoragePolicy.Create().SetName("src").SetType("local").
		SetStatus(storagepolicy.StatusActive).SaveX(ctx)
	dstPolicy := client.StoragePolicy.Create().SetName("dst").SetType("local").
		SetStatus(storagepolicy.StatusActive).
		SetDirNameRule("vault/{uid}").SetFileNameRule("{originname}").SaveX(ctx)

	content := []byte("relocate-me")
	srcFile := filepath.Join(tmp, "blob.bin")
	require.NoError(t, os.WriteFile(srcFile, content, 0o644))

	e := client.Entity.Create().
		SetType(0).
		SetSource(srcFile).
		SetSize(int64(len(content))).
		SetReferenceCount(1).
		SetStoragePolicyEntities(srcPolicy.ID).
		SetCreatedBy(user.ID).
		SaveX(ctx)
	e = client.Entity.Query().Where(entity.ID(e.ID)).WithFile().WithUser().OnlyX(ctx)

	fm := manager.NewFileManager(dep, user)
	defer fm.Recycle()

	m := relocateTaskWithState(t, &RelocateTaskState{DstPolicyID: dstPolicy.ID})
	m.state = &RelocateTaskState{DstPolicyID: dstPolicy.ID}
	m.l = logging.NewConsoleLogger(logging.LevelError)
	m.progress = queue.Progresses{
		ProgressTypeRelocateCount: &queue.Progress{},
		ProgressTypeRelocateSize:  &queue.Progress{},
	}

	require.NoError(t, m.relocateEntity(ctx, dep, fm, dstPolicy, e))

	updated := client.Entity.GetX(ctx, e.ID)
	require.Equal(t, dstPolicy.ID, updated.StoragePolicyEntities)
	require.Equal(t, fmt.Sprintf("vault/%d/blob.bin", user.ID), updated.Source)
	require.NoFileExists(t, srcFile)
	got, err := os.ReadFile(filepath.Join(tmp, updated.Source))
	require.NoError(t, err)
	require.Equal(t, content, got)
	require.Equal(t, int64(1), m.progress[ProgressTypeRelocateCount].Current)
	require.Equal(t, int64(len(content)), m.progress[ProgressTypeRelocateSize].Current)
}
