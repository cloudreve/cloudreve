package dependency

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/mime"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/stretchr/testify/require"
)

type queueTestConfig struct{ conf.ConfigProvider }

func (queueTestConfig) System() *conf.System { return &conf.System{Mode: conf.MasterMode} }

type queueTestSettings struct{ setting.Provider }

func (queueTestSettings) Queue(context.Context, setting.QueueType) *setting.QueueSetting {
	return &setting.QueueSetting{WorkerNum: 1, MaxExecution: time.Minute}
}

type queueTestTasks struct{ inventory.TaskClient }
type queueTestMime struct{ mime.MimeDetector }
type queueTestQueue struct {
	queue.Queue
	shutdown func()
}

func (q *queueTestQueue) Shutdown() { q.shutdown() }

func TestQueueReloadAllowsCleanupDependencyLookups(t *testing.T) {
	for _, test := range []struct {
		name string
		get  func(*dependency, context.Context) queue.Queue
		slot func(*dependency) *queue.Queue
	}{
		{"thumb", (*dependency).ThumbQueue, func(d *dependency) *queue.Queue { return &d.thumbQueue }},
		{"media", (*dependency).MediaMetaQueue, func(d *dependency) *queue.Queue { return &d.mediaMetaQueue }},
		{"io", (*dependency).IoIntenseQueue, func(d *dependency) *queue.Queue { return &d.ioIntenseQueue }},
		{"download", (*dependency).RemoteDownloadQueue, func(d *dependency) *queue.Queue { return &d.remoteDownloadQueue }},
		{"recycle", (*dependency).EntityRecycleQueue, func(d *dependency) *queue.Queue { return &d.entityRecycleQueue }},
		{"slave", (*dependency).SlaveQueue, func(d *dependency) *queue.Queue { return &d.slaveQueue }},
	} {
		t.Run(test.name, func(t *testing.T) {
			d := &dependency{
				logger: logging.NewConsoleLogger(logging.LevelError), configProvider: queueTestConfig{},
				settingProvider: queueTestSettings{}, taskClient: queueTestTasks{}, mimeDetector: queueTestMime{},
			}
			escape := make(chan struct{})
			defer close(escape)
			cleanupDone := make(chan error, 1)
			old := &queueTestQueue{}
			old.shutdown = func() {
				go func() {
					d.MimeDetector(context.Background())
					if test.get(d, context.Background()) != old {
						cleanupDone <- fmt.Errorf("cleanup did not receive the existing queue")
						return
					}
					cleanupDone <- nil
				}()
				select {
				case err := <-cleanupDone:
					cleanupDone <- err
				case <-escape:
				}
			}
			*test.slot(d) = old
			done := make(chan queue.Queue, 1)
			go func() {
				done <- test.get(d, context.WithValue(context.Background(), ReloadCtx{}, true))
			}()
			select {
			case replacement := <-done:
				require.NotSame(t, old, replacement)
				require.NoError(t, <-cleanupDone)
				require.Same(t, replacement, test.get(d, context.Background()))
				replacement.Shutdown()
			case <-time.After(2 * time.Second):
				t.Fatal("queue reload deadlocked with cleanup dependency lookup")
			}
		})
	}
}

func TestQueueReloadSerializesReplacementWithoutBlockingOtherQueues(t *testing.T) {
	d := &dependency{}
	started := make(chan struct{})
	release := make(chan struct{})
	releaseShutdown := sync.OnceFunc(func() { close(release) })
	defer releaseShutdown()
	var shutdowns, creations atomic.Int32
	old := &queueTestQueue{shutdown: func() {
		shutdowns.Add(1)
		close(started)
		<-release
	}}
	var slot queue.Queue = old
	create := func() queue.Queue {
		creations.Add(1)
		return &queueTestQueue{shutdown: func() { shutdowns.Add(1) }}
	}
	reloadCtx := context.WithValue(context.Background(), ReloadCtx{}, true)
	done := make(chan queue.Queue, 2)
	go func() { done <- d.queueWithReload(reloadCtx, setting.QueueTypeIOIntense, &slot, create) }()
	<-started
	go func() { done <- d.queueWithReload(reloadCtx, setting.QueueTypeIOIntense, &slot, create) }()

	lookupDone := make(chan struct{})
	go func() {
		d.queueWithReload(context.Background(), setting.QueueTypeIOIntense, &slot, create)
		var other queue.Queue
		d.queueWithReload(context.Background(), setting.QueueTypeThumb, &other, func() queue.Queue {
			return &queueTestQueue{shutdown: func() {}}
		})
		close(lookupDone)
	}()
	select {
	case <-lookupDone:
	case <-time.After(time.Second):
		t.Fatal("reload blocked ordinary lookups or initialization of another queue")
	}
	require.Zero(t, creations.Load())
	releaseShutdown()
	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("serialized reload did not finish")
		}
	}
	require.EqualValues(t, 2, creations.Load())
	require.EqualValues(t, 2, shutdowns.Load())
}
