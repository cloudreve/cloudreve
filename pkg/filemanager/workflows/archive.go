package workflows

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/task"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/encrypt"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager/entitysource"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gofrs/uuid"
)

type (
	CreateArchiveTask struct {
		*queue.DBTask

		l        logging.Logger
		state    *CreateArchiveTaskState
		progress queue.Progresses
		node     cluster.Node
	}

	CreateArchiveTaskPhase string

	CreateArchiveTaskState struct {
		Uris               []string                     `json:"uris,omitempty"`
		Dst                string                       `json:"dst,omitempty"`
		TempPath           string                       `json:"temp_path,omitempty"`
		ArchiveFile        string                       `json:"archive_file,omitempty"`
		Phase              CreateArchiveTaskPhase       `json:"phase,omitempty"`
		SlaveUploadTaskID  int                          `json:"slave__upload_task_id,omitempty"`
		SlaveArchiveTaskID int                          `json:"slave__archive_task_id,omitempty"`
		SlaveCompressState *SlaveCreateArchiveTaskState `json:"slave_compress_state,omitempty"`
		Failed             int                          `json:"failed,omitempty"`
		NodeState          `json:",inline"`
	}
)

const (
	CreateArchiveTaskPhaseNotStarted    CreateArchiveTaskPhase = "not_started"
	CreateArchiveTaskPhaseCompressFiles CreateArchiveTaskPhase = "compress_files"
	CreateArchiveTaskPhaseUploadArchive CreateArchiveTaskPhase = "upload_archive"

	CreateArchiveTaskPhaseAwaitSlaveCompressing        CreateArchiveTaskPhase = "await_slave_compressing"
	CreateArchiveTaskPhaseCreateAndAwaitSlaveUploading CreateArchiveTaskPhase = "await_slave_uploading"
	CreateArchiveTaskPhaseCompleteUpload               CreateArchiveTaskPhase = "complete_upload"

	ProgressTypeArchiveCount = "archive_count"
	ProgressTypeArchiveSize  = "archive_size"
	ProgressTypeUpload       = "upload"
	ProgressTypeUploadCount  = "upload_count"
)

func init() {
	queue.RegisterResumableTaskFactory(queue.CreateArchiveTaskType, NewCreateArchiveTaskFromModel)
}

// NewCreateArchiveTask creates a new CreateArchiveTask
func NewCreateArchiveTask(ctx context.Context, src []string, dst string) (queue.Task, error) {
	state := &CreateArchiveTaskState{
		Uris:      src,
		Dst:       dst,
		NodeState: NodeState{},
	}
	stateBytes, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state: %w", err)
	}

	t := &CreateArchiveTask{
		DBTask: &queue.DBTask{
			Task: &ent.Task{
				Type:          queue.CreateArchiveTaskType,
				CorrelationID: logging.CorrelationID(ctx),
				PrivateState:  string(stateBytes),
				PublicState:   &types.TaskPublicState{},
			},
			DirectOwner: inventory.UserFromContext(ctx),
		},
	}
	return t, nil
}

func NewCreateArchiveTaskFromModel(task *ent.Task) queue.Task {
	return &CreateArchiveTask{
		DBTask: &queue.DBTask{
			Task: task,
		},
	}
}

func (m *CreateArchiveTask) Do(ctx context.Context) (task.Status, error) {
	dep := dependency.FromContext(ctx)
	m.l = dep.Logger()

	m.Lock()
	if m.progress == nil {
		m.progress = make(queue.Progresses)
	}
	m.Unlock()

	// unmarshal state
	state := &CreateArchiveTaskState{}
	if err := json.Unmarshal([]byte(m.State()), state); err != nil {
		return task.StatusError, fmt.Errorf("failed to unmarshal state: %w", err)
	}
	m.state = state

	// select node
	node, err := allocateNode(ctx, dep, &m.state.NodeState, types.NodeCapabilityCreateArchive)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to allocate node: %w", err)
	}
	m.node = node

	next := task.StatusCompleted

	if m.node.IsMaster() {
		// Initialize temp folder
		// Compress files
		// Upload files to dst
		switch m.state.Phase {
		case CreateArchiveTaskPhaseNotStarted, "":
			next, err = m.initializeTempFolder(ctx, dep)
		case CreateArchiveTaskPhaseCompressFiles:
			next, err = m.createArchiveFile(ctx, dep)
		case CreateArchiveTaskPhaseUploadArchive:
			next, err = m.uploadArchive(ctx, dep)
		default:
			next, err = task.StatusError, fmt.Errorf("unknown phase %q: %w", m.state.Phase, queue.CriticalErr)
		}
	} else {
		// Listing all files and send to slave node for compressing
		// Await compressing and send to slave for uploading
		// Await uploading and complete upload
		switch m.state.Phase {
		case CreateArchiveTaskPhaseNotStarted, "":
			next, err = m.listEntitiesAndSendToSlave(ctx, dep)
		case CreateArchiveTaskPhaseAwaitSlaveCompressing:
			next, err = m.awaitSlaveCompressing(ctx, dep)
		case CreateArchiveTaskPhaseCreateAndAwaitSlaveUploading:
			next, err = m.createAndAwaitSlaveUploading(ctx, dep)
		case CreateArchiveTaskPhaseCompleteUpload:
			next, err = m.completeUpload(ctx, dep)
		default:
			next, err = task.StatusError, fmt.Errorf("unknown phase %q: %w", m.state.Phase, queue.CriticalErr)
		}
	}

	newStateStr, marshalErr := json.Marshal(m.state)
	if marshalErr != nil {
		return task.StatusError, fmt.Errorf("failed to marshal state: %w", marshalErr)
	}

	m.Lock()
	m.Task.PrivateState = string(newStateStr)
	m.Unlock()
	return next, err
}

func (m *CreateArchiveTask) Cleanup(ctx context.Context) error {
	if m.state.SlaveCompressState != nil && m.state.SlaveCompressState.TempPath != "" && m.node != nil {
		if err := m.node.CleanupFolders(context.Background(), m.state.SlaveCompressState.TempPath); err != nil {
			m.l.Warning("Failed to cleanup slave temp folder %s: %s", m.state.SlaveCompressState.TempPath, err)
		}
	}

	if m.state.TempPath != "" {
		time.Sleep(time.Duration(1) * time.Second)
		return os.RemoveAll(m.state.TempPath)
	}

	return nil
}

func (m *CreateArchiveTask) initializeTempFolder(ctx context.Context, dep dependency.Dep) (task.Status, error) {
	tempPath, err := prepareTempFolder(ctx, dep, m)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to prepare temp folder: %w", err)
	}

	m.state.TempPath = tempPath
	m.state.Phase = CreateArchiveTaskPhaseCompressFiles
	m.ResumeAfter(0)
	return task.StatusSuspending, nil
}

func (m *CreateArchiveTask) listEntitiesAndSendToSlave(ctx context.Context, dep dependency.Dep) (task.Status, error) {
	uris, err := fs.NewUriFromStrings(m.state.Uris...)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to create uri from strings: %s (%w)", err, queue.CriticalErr)
	}

	payload := &SlaveCreateArchiveTaskState{
		Entities: make([]SlaveCreateArchiveEntity, 0, len(uris)),
		Policies: make(map[int]*ent.StoragePolicy),
	}

	user := inventory.UserFromContext(ctx)
	fm := manager.NewFileManager(dep, user)
	storagePolicyClient := dep.StoragePolicyClient()
	masterKey, _ := dep.MasterEncryptKeyVault(ctx).GetMasterKey(ctx)

	failed, err := fm.CreateArchive(ctx, uris, io.Discard,
		fs.WithDryRun(func(name string, e fs.Entity) {
			entityModel, err := decryptEntityKeyIfNeeded(masterKey, e.Model())
			if err != nil {
				m.l.Warning("Failed to decrypt entity key for %q: %s", name, err)
				return
			}

			payload.Entities = append(payload.Entities, SlaveCreateArchiveEntity{
				Entity: entityModel,
				Path:   name,
			})
			if _, ok := payload.Policies[e.PolicyID()]; !ok {
				policy, err := storagePolicyClient.GetPolicyByID(ctx, e.PolicyID())
				if err != nil {
					m.l.Warning("Failed to get policy %d: %s", e.PolicyID(), err)
				} else {
					payload.Policies[e.PolicyID()] = policy
				}
			}
		}),
		fs.WithMaxArchiveSize(user.Edges.Group.Settings.CompressSize),
	)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to compress files: %w", err)
	}

	m.state.Failed = failed
	payloadStr, err := json.Marshal(payload)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to marshal payload: %w", err)
	}

	taskId, err := m.node.CreateTask(ctx, queue.SlaveCreateArchiveTaskType, string(payloadStr))
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to create slave task: %w", err)
	}

	m.state.Phase = CreateArchiveTaskPhaseAwaitSlaveCompressing
	m.state.SlaveArchiveTaskID = taskId
	m.ResumeAfter((10 * time.Second))
	return task.StatusSuspending, nil
}

func (m *CreateArchiveTask) awaitSlaveCompressing(ctx context.Context, dep dependency.Dep) (task.Status, error) {
	t, err := m.node.GetTask(ctx, m.state.SlaveArchiveTaskID, false)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to get slave task: %w", err)
	}

	m.Lock()
	m.state.NodeState.progress = t.Progress
	m.Unlock()

	m.state.SlaveCompressState = &SlaveCreateArchiveTaskState{}
	if err := json.Unmarshal([]byte(t.PrivateState), m.state.SlaveCompressState); err != nil {
		return task.StatusError, fmt.Errorf("failed to unmarshal slave compress state: %s (%w)", err, queue.CriticalErr)
	}

	if t.Status == task.StatusError {
		return task.StatusError, fmt.Errorf("slave task failed: %s (%w)", t.Error, queue.CriticalErr)
	}

	if t.Status == task.StatusCanceled {
		return task.StatusError, fmt.Errorf("slave task canceled (%w)", queue.CriticalErr)
	}

	if t.Status == task.StatusCompleted {
		m.state.Phase = CreateArchiveTaskPhaseCreateAndAwaitSlaveUploading
		m.ResumeAfter(0)
		return task.StatusSuspending, nil
	}

	m.l.Info("Slave task %d is still compressing, resume after 30s.", m.state.SlaveArchiveTaskID)
	m.ResumeAfter((time.Second * 30))
	return task.StatusSuspending, nil
}

func (m *CreateArchiveTask) createAndAwaitSlaveUploading(ctx context.Context, dep dependency.Dep) (task.Status, error) {
	u := inventory.UserFromContext(ctx)

	if m.state.SlaveUploadTaskID == 0 {
		dst, err := fs.NewUriFromString(m.state.Dst)
		if err != nil {
			return task.StatusError, fmt.Errorf("failed to parse dst uri %q: %s (%w)", m.state.Dst, err, queue.CriticalErr)
		}

		// Create slave upload task
		payload := &SlaveUploadTaskState{
			Files: []SlaveUploadEntity{
				{
					Size: m.state.SlaveCompressState.CompressedSize,
					Uri:  dst,
					Src:  m.state.SlaveCompressState.ZipFilePath,
				},
			},
			MaxParallel: dep.SettingProvider().MaxParallelTransfer(ctx),
			UserID:      u.ID,
		}

		payloadStr, err := json.Marshal(payload)
		if err != nil {
			return task.StatusError, fmt.Errorf("failed to marshal payload: %w", err)
		}

		taskId, err := m.node.CreateTask(ctx, queue.SlaveUploadTaskType, string(payloadStr))
		if err != nil {
			return task.StatusError, fmt.Errorf("failed to create slave task: %w", err)
		}

		m.state.NodeState.progress = nil
		m.state.SlaveUploadTaskID = taskId
		m.ResumeAfter(0)
		return task.StatusSuspending, nil
	}

	m.l.Info("Checking slave upload task %d...", m.state.SlaveUploadTaskID)
	t, err := m.node.GetTask(ctx, m.state.SlaveUploadTaskID, true)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to get slave task: %w", err)
	}

	m.Lock()
	m.state.NodeState.progress = t.Progress
	m.Unlock()

	if t.Status == task.StatusError {
		return task.StatusError, fmt.Errorf("slave task failed: %s (%w)", t.Error, queue.CriticalErr)
	}

	if t.Status == task.StatusCanceled {
		return task.StatusError, fmt.Errorf("slave task canceled (%w)", queue.CriticalErr)
	}

	if t.Status == task.StatusCompleted {
		m.state.Phase = CreateArchiveTaskPhaseCompleteUpload
		m.ResumeAfter(0)
		return task.StatusSuspending, nil
	}

	m.l.Info("Slave task %d is still uploading, resume after 30s.", m.state.SlaveUploadTaskID)
	m.ResumeAfter(time.Second * 30)
	return task.StatusSuspending, nil
}

func (m *CreateArchiveTask) completeUpload(ctx context.Context, dep dependency.Dep) (task.Status, error) {
	return task.StatusCompleted, nil
}

func (m *CreateArchiveTask) createArchiveFile(ctx context.Context, dep dependency.Dep) (task.Status, error) {
	uris, err := fs.NewUriFromStrings(m.state.Uris...)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to create uri from strings: %s (%w)", err, queue.CriticalErr)
	}

	user := inventory.UserFromContext(ctx)
	fm := manager.NewFileManager(dep, user)

	// Create temp zip file
	fileName := fmt.Sprintf("%s.zip", uuid.Must(uuid.NewV4()))
	zipFilePath := filepath.Join(
		m.state.TempPath,
		fileName,
	)
	zipFile, err := util.CreatNestedFile(zipFilePath)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to create zip file: %w", err)
	}

	defer zipFile.Close()

	// Start compressing
	m.Lock()
	m.progress[ProgressTypeArchiveCount] = &queue.Progress{}
	m.progress[ProgressTypeArchiveSize] = &queue.Progress{}
	m.Unlock()
	failed, err := fm.CreateArchive(ctx, uris, zipFile,
		fs.WithArchiveCompression(true),
		fs.WithMaxArchiveSize(user.Edges.Group.Settings.CompressSize),
		fs.WithProgressFunc(func(current, diff int64, total int64) {
			atomic.AddInt64(&m.progress[ProgressTypeArchiveSize].Current, diff)
			atomic.AddInt64(&m.progress[ProgressTypeArchiveCount].Current, 1)
		}),
	)
	if err != nil {
		zipFile.Close()
		_ = os.Remove(zipFilePath)
		return task.StatusError, fmt.Errorf("failed to compress files: %w", err)
	}

	m.state.Failed = failed
	m.Lock()
	delete(m.progress, ProgressTypeArchiveSize)
	delete(m.progress, ProgressTypeArchiveCount)
	m.Unlock()

	m.state.Phase = CreateArchiveTaskPhaseUploadArchive
	m.state.ArchiveFile = fileName
	m.ResumeAfter(0)
	return task.StatusSuspending, nil
}

func (m *CreateArchiveTask) uploadArchive(ctx context.Context, dep dependency.Dep) (task.Status, error) {
	fm := manager.NewFileManager(dep, inventory.UserFromContext(ctx))
	zipFilePath := filepath.Join(
		m.state.TempPath,
		m.state.ArchiveFile,
	)

	m.l.Info("Uploading archive file %s to %s...", zipFilePath, m.state.Dst)

	uri, err := fs.NewUriFromString(m.state.Dst)
	if err != nil {
		return task.StatusError, fmt.Errorf(
			"failed to parse dst uri %q: %s (%w)",
			m.state.Dst,
			err,
			queue.CriticalErr,
		)
	}

	file, err := os.Open(zipFilePath)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to open compressed archive %q: %s", m.state.ArchiveFile, err)
	}
	defer file.Close()
	fi, err := file.Stat()
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to get file info: %w", err)
	}
	size := fi.Size()

	m.Lock()
	m.progress[ProgressTypeUpload] = &queue.Progress{}
	m.Unlock()
	fileData := &fs.UploadRequest{
		Props: &fs.UploadProps{
			Uri:  uri,
			Size: size,
		},
		ProgressFunc: func(current, diff int64, total int64) {
			atomic.StoreInt64(&m.progress[ProgressTypeUpload].Current, current)
			atomic.StoreInt64(&m.progress[ProgressTypeUpload].Total, total)
		},
		File:   file,
		Seeker: file,
	}

	_, err = fm.Update(ctx, fileData)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to upload archive file: %w", err)
	}

	return task.StatusCompleted, nil
}

func (m *CreateArchiveTask) Progress(ctx context.Context) queue.Progresses {
	m.Lock()
	defer m.Unlock()

	if m.state.NodeState.progress != nil {
		merged := make(queue.Progresses)
		for k, v := range m.progress {
			merged[k] = v
		}

		for k, v := range m.state.NodeState.progress {
			merged[k] = v
		}

		return merged
	}
	return m.progress
}

func (m *CreateArchiveTask) Summarize(hasher hashid.Encoder) *queue.Summary {
	// unmarshal state
	if m.state == nil {
		if err := json.Unmarshal([]byte(m.State()), &m.state); err != nil {
			return nil
		}
	}

	failed := m.state.Failed
	if m.state.SlaveCompressState != nil {
		failed = m.state.SlaveCompressState.Failed
	}

	return &queue.Summary{
		NodeID: m.state.NodeID,
		Phase:  string(m.state.Phase),
		Props: map[string]any{
			SummaryKeySrcMultiple: m.state.Uris,
			SummaryKeyDst:         m.state.Dst,
			SummaryKeyFailed:      failed,
		},
	}
}

type (
	SlaveCreateArchiveEntity struct {
		Entity *ent.Entity `json:"entity"`
		Path   string      `json:"path"`
	}
	SlaveCreateArchiveTaskState struct {
		Entities       []SlaveCreateArchiveEntity `json:"entities"`
		Policies       map[int]*ent.StoragePolicy `json:"policies"`
		CompressedSize int64                      `json:"compressed_size"`
		TempPath       string                     `json:"temp_path"`
		ZipFilePath    string                     `json:"zip_file_path"`
		Failed         int                        `json:"failed"`
	}
	SlaveCreateArchiveTask struct {
		*queue.InMemoryTask

		mu       sync.RWMutex
		progress queue.Progresses
		l        logging.Logger
		state    *SlaveCreateArchiveTaskState
	}
)

// NewSlaveCreateArchiveTask creates a new SlaveCreateArchiveTask from raw private state
func NewSlaveCreateArchiveTask(ctx context.Context, props *types.SlaveTaskProps, id int, state string) queue.Task {
	return &SlaveCreateArchiveTask{
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

func (t *SlaveCreateArchiveTask) Do(ctx context.Context) (task.Status, error) {
	ctx = prepareSlaveTaskCtx(ctx, t.Model().PublicState.SlaveTaskProps)
	dep := dependency.FromContext(ctx)
	t.l = dep.Logger()
	fm := manager.NewFileManager(dep, nil)

	// unmarshal state
	state := &SlaveCreateArchiveTaskState{}
	if err := json.Unmarshal([]byte(t.State()), state); err != nil {
		return task.StatusError, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	t.state = state

	totalFiles := int64(0)
	totalFileSize := int64(0)
	for _, e := range t.state.Entities {
		totalFiles++
		totalFileSize += e.Entity.Size
	}

	t.Lock()
	t.progress[ProgressTypeArchiveCount] = &queue.Progress{Total: totalFiles}
	t.progress[ProgressTypeArchiveSize] = &queue.Progress{Total: totalFileSize}
	t.Unlock()

	// 3. Create temp workspace
	tempPath, err := prepareTempFolder(ctx, dep, t)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to prepare temp folder: %w", err)
	}
	t.state.TempPath = tempPath

	// 2. Create archive file
	fileName := fmt.Sprintf("%s.zip", uuid.Must(uuid.NewV4()))
	zipFilePath := filepath.Join(
		t.state.TempPath,
		fileName,
	)
	zipFile, err := util.CreatNestedFile(zipFilePath)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to create zip file: %w", err)
	}

	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 3. Download each entity and write into zip file
	for _, e := range t.state.Entities {
		policy, ok := t.state.Policies[e.Entity.StoragePolicyEntities]
		if !ok {
			state.Failed++
			t.l.Warning("Policy not found for entity %d, skipping...", e.Entity.ID)
			continue
		}

		entity := fs.NewEntity(e.Entity)
		es, err := fm.GetEntitySource(ctx, 0,
			fs.WithEntity(entity),
			fs.WithPolicy(fm.CastStoragePolicyOnSlave(ctx, policy)),
		)
		if err != nil {
			state.Failed++
			t.l.Warning("Failed to get entity source for entity %d: %s, skipping...", e.Entity.ID, err)
			continue
		}

		// Write to zip file
		header := &zip.FileHeader{
			Name:               e.Path,
			Modified:           entity.UpdatedAt(),
			UncompressedSize64: uint64(entity.Size()),
			Method:             zip.Deflate,
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			es.Close()
			state.Failed++
			t.l.Warning("Failed to create zip header for %s: %s, skipping...", e.Path, err)
			continue
		}

		es.Apply(entitysource.WithContext(ctx))
		_, err = io.Copy(writer, es)
		es.Close()
		if err != nil {
			state.Failed++
			t.l.Warning("Failed to write entity %d to zip file: %s, skipping...", e.Entity.ID, err)
		}

		atomic.AddInt64(&t.progress[ProgressTypeArchiveSize].Current, entity.Size())
		atomic.AddInt64(&t.progress[ProgressTypeArchiveCount].Current, 1)
	}

	zipWriter.Close()
	stat, err := zipFile.Stat()
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to get compressed file info: %w", err)
	}

	t.state.CompressedSize = stat.Size()
	t.state.ZipFilePath = zipFilePath
	// Clear unused fields to save space
	t.state.Entities = nil
	t.state.Policies = nil

	newStateStr, marshalErr := json.Marshal(t.state)
	if marshalErr != nil {
		return task.StatusError, fmt.Errorf("failed to marshal state: %w", marshalErr)
	}

	t.Lock()
	t.Task.PrivateState = string(newStateStr)
	t.Unlock()
	return task.StatusCompleted, nil
}

func (m *SlaveCreateArchiveTask) Progress(ctx context.Context) queue.Progresses {
	m.Lock()
	defer m.Unlock()

	return m.progress
}

func decryptEntityKeyIfNeeded(masterKey []byte, entity *ent.Entity) (*ent.Entity, error) {
	if entity.Props == nil || entity.Props.EncryptMetadata == nil || entity.Props.EncryptMetadata.KeyPlainText != nil {
		return entity, nil
	}

	decryptedKey, err := encrypt.DecryptWithMasterKey(masterKey, entity.Props.EncryptMetadata.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt entity key: %w", err)
	}

	entity.Props.EncryptMetadata.KeyPlainText = decryptedKey
	entity.Props.EncryptMetadata.Key = nil
	return entity, nil
}
