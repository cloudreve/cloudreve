package workflows

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"sync/atomic"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/task"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/activity"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
)

type (
	ImportTask struct {
		*queue.DBTask

		l        logging.Logger
		state    *ImportTaskState
		progress queue.Progresses
	}
	ImportTaskState struct {
		PolicyID         int             `json:"policy_id"`
		PolicyName       string          `json:"policy_name,omitempty"`
		Src              string          `json:"src"`
		Recursive        bool            `json:"is_recursive"`
		Dst              string          `json:"dst"`
		Phase            ImportTaskPhase `json:"phase"`
		Failed           int             `json:"failed,omitempty"`
		ExtractMediaMeta bool            `json:"extract_media_meta"`
	}
	ImportTaskPhase string
)

const (
	ProgressTypeImported = "imported"
	ProgressTypeIndexed  = "indexed"

	// ImportBatchSize is the number of files to process in each batch
	// to control memory usage during large imports.
	ImportBatchSize = 100
)

func init() {
	queue.RegisterResumableTaskFactory(queue.ImportTaskType, NewImportTaskFromModel)
}

func NewImportTask(ctx context.Context, u *ent.User, src string, recursive bool, dst string, policyID int, policyName string) (queue.Task, error) {
	state := &ImportTaskState{
		Src:        src,
		Recursive:  recursive,
		Dst:        dst,
		PolicyID:   policyID,
		PolicyName: policyName,
	}
	stateBytes, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state: %w", err)
	}

	t := &ImportTask{
		DBTask: &queue.DBTask{
			Task: &ent.Task{
				Type:          queue.ImportTaskType,
				CorrelationID: logging.CorrelationID(ctx),
				PrivateState:  string(stateBytes),
				PublicState:   &types.TaskPublicState{},
			},
			DirectOwner: u,
		},
	}

	return t, nil
}

func NewImportTaskFromModel(task *ent.Task) queue.Task {
	return &ImportTask{
		DBTask: &queue.DBTask{
			Task: task,
		},
	}
}

func (m *ImportTask) Do(ctx context.Context) (task.Status, error) {
	dep := dependency.FromContext(ctx)
	m.l = dep.Logger()

	m.Lock()
	if m.progress == nil {
		m.progress = make(queue.Progresses)
	}
	m.progress[ProgressTypeIndexed] = &queue.Progress{}
	m.Unlock()

	// unmarshal state
	state := &ImportTaskState{}
	if err := json.Unmarshal([]byte(m.State()), state); err != nil {
		return task.StatusError, fmt.Errorf("failed to unmarshal state: %w", err)
	}
	m.state = state

	next, err := m.processImport(ctx, dep)

	newStateStr, marshalErr := json.Marshal(m.state)
	if marshalErr != nil {
		return task.StatusError, fmt.Errorf("failed to marshal state: %w", marshalErr)
	}

	m.Lock()
	m.Task.PrivateState = string(newStateStr)
	m.Unlock()
	return next, err
}

func (m *ImportTask) processImport(ctx context.Context, dep dependency.Dep) (task.Status, error) {
	user := inventory.UserFromContext(ctx)

	dst, err := fs.NewUriFromString(m.state.Dst)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to parse dst: %s (%w)", err, queue.CriticalErr)
	}

	// Use a temporary file manager just for listing physical files
	listFm := manager.NewFileManager(dep, user)
	physicalFiles, err := listFm.ListPhysical(ctx, m.state.Src, m.state.PolicyID, m.state.Recursive,
		func(i int) {
			atomic.AddInt64(&m.progress[ProgressTypeIndexed].Current, int64(i))
		})
	listFm.Recycle()
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to list physical files: %w", err)
	}

	m.l.Info("Importing %d physical files", len(physicalFiles))

	m.Lock()
	m.progress[ProgressTypeImported] = &queue.Progress{
		Total: int64(len(physicalFiles)),
	}
	delete(m.progress, ProgressTypeIndexed)
	m.Unlock()

	failed := 0
	totalFiles := len(physicalFiles)

	// Process files in batches to control memory usage
	for batchStart := 0; batchStart < totalFiles; batchStart += ImportBatchSize {
		batchEnd := min(batchStart+ImportBatchSize, totalFiles)

		batch := physicalFiles[batchStart:batchEnd]
		batchFailed := m.processBatch(ctx, dep, user, dst, batch)
		failed += batchFailed

		// Clear batch elements to allow GC of individual items
		for i := batchStart; i < batchEnd; i++ {
			physicalFiles[i] = fs.PhysicalObject{}
		}

		// Run GC after each batch to free memory
		runtime.GC()
	}

	// Clear the entire slice to allow GC
	physicalFiles = nil
	runtime.GC()

	m.state.Failed = failed
	return task.StatusCompleted, nil
}

// processBatch processes a batch of physical files with a fresh file manager.
func (m *ImportTask) processBatch(ctx context.Context, dep dependency.Dep, user *ent.User, dst *fs.URI, batch []fs.PhysicalObject) int {
	fm := manager.NewFileManager(dep, user)
	defer fm.Recycle()

	failed := 0
	for _, physicalFile := range batch {
		if physicalFile.IsDir {
			m.l.Info("Creating folder %s", physicalFile.RelativePath)
			_, err := fm.Create(ctx, dst.Join(physicalFile.RelativePath), types.FileTypeFolder)
			atomic.AddInt64(&m.progress[ProgressTypeImported].Current, 1)
			if err != nil {
				m.l.Warning("Failed to create folder %s: %s", physicalFile.RelativePath, err)
				failed++
			}
		} else {
			m.l.Info("Importing file %s", physicalFile.RelativePath)
			err := fm.ImportPhysical(ctx, dst, m.state.PolicyID, physicalFile, m.state.ExtractMediaMeta)
			atomic.AddInt64(&m.progress[ProgressTypeImported].Current, 1)
			if err != nil {
				var appErr serializer.AppError
				if errors.As(err, &appErr) && appErr.Code == serializer.CodeObjectExist {
					m.l.Info("File %s already exists, skipping", physicalFile.RelativePath)
					continue
				}
				m.l.Error("Failed to import file %s: %s, skipping", physicalFile.RelativePath, err)
				failed++
			} else {
				activity.Record(ctx, dep.SettingProvider(), dep.ActivityClient(), types.EventFileImported,
					activity.Extra(map[string]any{"uri": dst.Join(physicalFile.RelativePath).String(), "src": physicalFile.Source}))
			}
		}
	}

	return failed
}

func (m *ImportTask) Progress(ctx context.Context) queue.Progresses {
	m.Lock()
	defer m.Unlock()
	return m.progress
}

func (m *ImportTask) Summarize(hasher hashid.Encoder) *queue.Summary {
	// unmarshal state
	if m.state == nil {
		if err := json.Unmarshal([]byte(m.State()), &m.state); err != nil {
			return nil
		}
	}

	return &queue.Summary{
		Phase: string(m.state.Phase),
		Props: map[string]any{
			SummaryKeyDst:            m.state.Dst,
			SummaryKeySrcStr:         m.state.Src,
			SummaryKeyFailed:         m.state.Failed,
			SummaryKeySrcDstPolicyID: hashid.EncodePolicyID(hasher, m.state.PolicyID),
			SummaryKeyDstPolicyName:  m.state.PolicyName,
		},
	}
}
