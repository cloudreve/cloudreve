package explorer

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	enttask "github.com/cloudreve/Cloudreve/v4/ent/task"
	entuser "github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"
)

// captureQueue records submitted tasks without running workers.
type captureQueue struct {
	queue.Queue
	submitted []queue.Task
}

func (q *captureQueue) QueueTask(ctx context.Context, t queue.Task) error {
	q.submitted = append(q.submitted, t)
	return nil
}

// retryDepStub exposes only the dependencies RetryTask touches.
type retryDepStub struct {
	dependency.Dep
	taskClient inventory.TaskClient
	ioQueue    queue.Queue
	downloadQ  queue.Queue
	recycleQ   queue.Queue
	mediaMetaQ queue.Queue
}

func (d *retryDepStub) TaskClient() inventory.TaskClient                { return d.taskClient }
func (d *retryDepStub) IoIntenseQueue(context.Context) queue.Queue      { return d.ioQueue }
func (d *retryDepStub) RemoteDownloadQueue(context.Context) queue.Queue { return d.downloadQ }
func (d *retryDepStub) EntityRecycleQueue(context.Context) queue.Queue  { return d.recycleQ }
func (d *retryDepStub) MediaMetaQueue(context.Context) queue.Queue      { return d.mediaMetaQ }

// TestRetryTask verifies failed-task retry: owner/admin gating, status gate,
// and queue routing by task type (upstream #2823).
func TestRetryTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	group := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	newUser := func(email string) *ent.User {
		u := client.User.Create().SetEmail(email).SetNick("u").SetStatus("active").SetGroup(group).SaveX(ctx)
		return client.User.Query().WithGroup().Where(entuser.ID(u.ID)).OnlyX(ctx)
	}
	owner := newUser("owner@example.com")
	other := newUser("other@example.com")

	ioQ := &captureQueue{}
	downloadQ := &captureQueue{}
	dep := &retryDepStub{
		taskClient: inventory.NewTaskClient(client, conf.SQLiteDB, nil),
		ioQueue:    ioQ,
		downloadQ:  downloadQ,
		recycleQ:   &captureQueue{},
		mediaMetaQ: &captureQueue{},
	}

	newCtx := func(u *ent.User) *gin.Context {
		engine := gin.New()
		engine.ContextWithFallback = true
		c := gin.CreateTestContextOnly(httptest.NewRecorder(), engine)
		c.Request = httptest.NewRequest("POST", "/", nil)
		util.WithValue(c, dependency.DepCtx{}, dep)
		util.WithValue(c, inventory.UserCtx{}, u)
		return c
	}

	newTask := func(ownerID int, taskType string, status enttask.Status) *ent.Task {
		return client.Task.Create().
			SetType(taskType).
			SetStatus(status).
			SetPublicState(&types.TaskPublicState{}).
			SetCorrelationID(uuid.Must(uuid.NewV4())).
			SetUserTasks(ownerID).
			SaveX(ctx)
	}

	t.Run("failed extract task re-queues to io-intense queue", func(t *testing.T) {
		ioQ.submitted = nil
		m := newTask(owner.ID, queue.ExtractArchiveTaskType, enttask.StatusError)
		require.NoError(t, RetryTask(newCtx(owner), m.ID))
		require.Len(t, ioQ.submitted, 1)
		require.Equal(t, m.ID, ioQ.submitted[0].ID())
	})

	t.Run("failed download task re-queues to download queue", func(t *testing.T) {
		downloadQ.submitted = nil
		m := newTask(owner.ID, queue.RemoteDownloadTaskType, enttask.StatusError)
		require.NoError(t, RetryTask(newCtx(owner), m.ID))
		require.Len(t, downloadQ.submitted, 1)
	})

	t.Run("non-error task rejected", func(t *testing.T) {
		ioQ.submitted = nil
		m := newTask(owner.ID, queue.ExtractArchiveTaskType, enttask.StatusCompleted)
		require.Error(t, RetryTask(newCtx(owner), m.ID))
		require.Empty(t, ioQ.submitted)
	})

	t.Run("other user's task rejected", func(t *testing.T) {
		ioQ.submitted = nil
		m := newTask(owner.ID, queue.ExtractArchiveTaskType, enttask.StatusError)
		require.Error(t, RetryTask(newCtx(other), m.ID))
		require.Empty(t, ioQ.submitted)
	})

	t.Run("slave task type not retryable", func(t *testing.T) {
		m := newTask(owner.ID, queue.SlaveUploadTaskType, enttask.StatusError)
		require.Error(t, RetryTask(newCtx(owner), m.ID))
	})
}
