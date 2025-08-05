package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/task"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
)

type (
	SlaveUploadEntity struct {
		Uri   *fs.URI `json:"uri"`
		Src   string  `json:"src"`
		Size  int64   `json:"size"`
		Index int     `json:"index"`
	}
	SlaveUploadTaskState struct {
		MaxParallel          int                 `json:"max_parallel"`
		Files                []SlaveUploadEntity `json:"files"`
		Transferred          map[int]interface{} `json:"transferred"`
		UserID               int                 `json:"user_id"`
		First5TransferErrors string              `json:"first_5_transfer_errors,omitempty"`
	}
	SlaveUploadTask struct {
		*queue.InMemoryTask

		progress queue.Progresses
		l        logging.Logger
		state    *SlaveUploadTaskState
		node     cluster.Node
	}
)

// NewSlaveUploadTask creates a new SlaveUploadTask from raw private state
func NewSlaveUploadTask(ctx context.Context, props *types.SlaveTaskProps, id int, state string) queue.Task {
	return &SlaveUploadTask{
		InMemoryTask: &queue.InMemoryTask{
			DBTask: &queue.DBTask{
				Task: &ent.Task{
					ID:            id,
					CorrelationID: logging.CorrelationID(ctx),
					PublicState: &types.TaskPublicState{
						SlaveTaskProps: props,
					},
					PrivateState: state,
				},
			},
		},

		progress: make(queue.Progresses),
	}
}

func (t *SlaveUploadTask) Do(ctx context.Context) (task.Status, error) {
	ctx = prepareSlaveTaskCtx(ctx, t.Model().PublicState.SlaveTaskProps)
	dep := dependency.FromContext(ctx)
	t.l = dep.Logger()

	np, err := dep.NodePool(ctx)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to get node pool: %w", err)
	}

	t.node, err = np.Get(ctx, types.NodeCapabilityNone, 0)
	if err != nil || !t.node.IsMaster() {
		return task.StatusError, fmt.Errorf("failed to get master node: %w", err)
	}

	fm := manager.NewFileManager(dep, nil)

	// unmarshal state
	state := &SlaveUploadTaskState{}
	if err := json.Unmarshal([]byte(t.State()), state); err != nil {
		return task.StatusError, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	t.state = state
	if t.state.Transferred == nil {
		t.state.Transferred = make(map[int]interface{})
	}

	wg := sync.WaitGroup{}
	worker := make(chan int, t.state.MaxParallel)
	for i := 0; i < t.state.MaxParallel; i++ {
		worker <- i
	}

	// Sum up total count
	totalCount := 0
	totalSize := int64(0)
	for _, res := range state.Files {
		totalSize += res.Size
		totalCount++
	}
	t.Lock()
	t.progress[ProgressTypeUploadCount] = &queue.Progress{}
	t.progress[ProgressTypeUpload] = &queue.Progress{}
	t.Unlock()
	atomic.StoreInt64(&t.progress[ProgressTypeUploadCount].Total, int64(totalCount))
	atomic.StoreInt64(&t.progress[ProgressTypeUpload].Total, totalSize)
	ae := serializer.NewAggregateError()
	transferFunc := func(workerId, fileId int, file SlaveUploadEntity) {
		t.l.Info("Uploading file %s to %s...", file.Src, file.Uri.String())

		progressKey := fmt.Sprintf("%s%d", ProgressTypeUploadSinglePrefix, workerId)
		t.Lock()
		t.progress[progressKey] = &queue.Progress{Identifier: file.Uri.String(), Total: file.Size}
		fileProgress := t.progress[progressKey]
		uploadProgress := t.progress[ProgressTypeUpload]
		uploadCountProgress := t.progress[ProgressTypeUploadCount]
		t.Unlock()

		defer func() {
			atomic.AddInt64(&uploadCountProgress.Current, 1)
			worker <- workerId
			wg.Done()
		}()

		handle, err := os.Open(filepath.FromSlash(file.Src))
		if err != nil {
			t.l.Warning("Failed to open file %s: %s", file.Src, err.Error())
			atomic.AddInt64(&fileProgress.Current, file.Size)
			ae.Add(path.Base(file.Src), fmt.Errorf("failed to open file: %w", err))
			return
		}

		stat, err := handle.Stat()
		if err != nil {
			t.l.Warning("Failed to get file stat for %s: %s", file.Src, err.Error())
			handle.Close()
			atomic.AddInt64(&fileProgress.Current, file.Size)
			ae.Add(path.Base(file.Src), fmt.Errorf("failed to get file stat: %w", err))
			return
		}

		fileData := &fs.UploadRequest{
			Props: &fs.UploadProps{
				Uri:  file.Uri,
				Size: stat.Size(),
			},
			ProgressFunc: func(current, diff int64, total int64) {
				atomic.AddInt64(&fileProgress.Current, diff)
				atomic.AddInt64(&uploadCountProgress.Current, 1)
				atomic.StoreInt64(&fileProgress.Total, total)
			},
			File:   handle,
			Seeker: handle,
		}

		_, err = fm.Update(ctx, fileData, fs.WithNode(t.node), fs.WithStatelessUserID(t.state.UserID), fs.WithNoEntityType())
		if err != nil {
			handle.Close()
			t.l.Warning("Failed to upload file %s: %s", file.Src, err.Error())
			atomic.AddInt64(&uploadProgress.Current, file.Size)
			ae.Add(path.Base(file.Src), fmt.Errorf("failed to upload file: %w", err))
			return
		}

		t.Lock()
		t.state.Transferred[fileId] = nil
		t.Unlock()
		handle.Close()
	}

	// Start upload files
	for fileId, file := range t.state.Files {
		// Check if file is already transferred
		if _, ok := t.state.Transferred[fileId]; ok {
			t.l.Info("File %s already transferred, skipping...", file.Src)
			t.Lock()
			atomic.AddInt64(&t.progress[ProgressTypeUpload].Current, file.Size)
			atomic.AddInt64(&t.progress[ProgressTypeUploadCount].Current, 1)
			t.Unlock()
			continue
		}

		select {
		case <-ctx.Done():
			return task.StatusError, ctx.Err()
		case workerId := <-worker:
			wg.Add(1)

			go transferFunc(workerId, fileId, file)
		}
	}

	wg.Wait()

	t.state.First5TransferErrors = ae.FormatFirstN(5)
	newStateStr, marshalErr := json.Marshal(t.state)
	if marshalErr != nil {
		return task.StatusError, fmt.Errorf("failed to marshal state: %w", marshalErr)
	}
	t.Lock()
	t.Task.PrivateState = string(newStateStr)
	t.Unlock()

	// If all files are failed to transfer, return error
	if len(t.state.Transferred) != len(t.state.Files) {
		t.l.Warning("%d files not transferred", len(t.state.Files)-len(t.state.Transferred))
		if len(t.state.Transferred) == 0 {
			return task.StatusError, fmt.Errorf("all file failed to transfer")
		}

	}

	return task.StatusCompleted, nil
}

func (m *SlaveUploadTask) Progress(ctx context.Context) queue.Progresses {
	m.Lock()
	defer m.Unlock()

	res := make(queue.Progresses)
	for k, v := range m.progress {
		res[k] = v
	}
	return res
}
