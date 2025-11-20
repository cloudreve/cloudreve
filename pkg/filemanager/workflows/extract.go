package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/task"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gofrs/uuid"
	"github.com/mholt/archives"
)

type (
	ExtractArchiveTask struct {
		*queue.DBTask

		l        logging.Logger
		state    *ExtractArchiveTaskState
		progress queue.Progresses
		node     cluster.Node
	}
	ExtractArchiveTaskPhase string
	ExtractArchiveTaskState struct {
		Uri             string   `json:"uri,omitempty"`
		Encoding        string   `json:"encoding,omitempty"`
		Dst             string   `json:"dst,omitempty"`
		TempPath        string   `json:"temp_path,omitempty"`
		TempZipFilePath string   `json:"temp_zip_file_path,omitempty"`
		ProcessedCursor string   `json:"processed_cursor,omitempty"`
		SlaveTaskID     int      `json:"slave_task_id,omitempty"`
		Password        string   `json:"password,omitempty"`
		FileMask        []string `json:"file_mask,omitempty"`
		NodeState       `json:",inline"`
		Phase           ExtractArchiveTaskPhase `json:"phase,omitempty"`
	}
)

const (
	ExtractArchivePhaseNotStarted         ExtractArchiveTaskPhase = ""
	ExtractArchivePhaseDownloadZip        ExtractArchiveTaskPhase = "download_zip"
	ExtractArchivePhaseAwaitSlaveComplete ExtractArchiveTaskPhase = "await_slave_complete"

	ProgressTypeExtractCount = "extract_count"
	ProgressTypeExtractSize  = "extract_size"
	ProgressTypeDownload     = "download"

	SummaryKeySrc         = "src"
	SummaryKeySrcPhysical = "src_physical"
	SummaryKeyDst         = "dst"
)

func init() {
	queue.RegisterResumableTaskFactory(queue.ExtractArchiveTaskType, NewExtractArchiveTaskFromModel)
}

// NewExtractArchiveTask creates a new ExtractArchiveTask
func NewExtractArchiveTask(ctx context.Context, src, dst, encoding, password string, mask []string) (queue.Task, error) {
	state := &ExtractArchiveTaskState{
		Uri:       src,
		Dst:       dst,
		Encoding:  encoding,
		NodeState: NodeState{},
		Password:  password,
		FileMask:  mask,
	}
	stateBytes, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state: %w", err)
	}

	t := &ExtractArchiveTask{
		DBTask: &queue.DBTask{
			Task: &ent.Task{
				Type:          queue.ExtractArchiveTaskType,
				CorrelationID: logging.CorrelationID(ctx),
				PrivateState:  string(stateBytes),
				PublicState:   &types.TaskPublicState{},
			},
			DirectOwner: inventory.UserFromContext(ctx),
		},
	}
	return t, nil
}

func NewExtractArchiveTaskFromModel(task *ent.Task) queue.Task {
	return &ExtractArchiveTask{
		DBTask: &queue.DBTask{
			Task: task,
		},
	}
}

func (m *ExtractArchiveTask) Do(ctx context.Context) (task.Status, error) {
	dep := dependency.FromContext(ctx)
	m.l = dep.Logger()

	m.Lock()
	if m.progress == nil {
		m.progress = make(queue.Progresses)
	}
	m.Unlock()

	// unmarshal state
	state := &ExtractArchiveTaskState{}
	if err := json.Unmarshal([]byte(m.State()), state); err != nil {
		return task.StatusError, fmt.Errorf("failed to unmarshal state: %w", err)
	}
	m.state = state

	// select node
	node, err := allocateNode(ctx, dep, &m.state.NodeState, types.NodeCapabilityExtractArchive)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to allocate node: %w", err)
	}
	m.node = node

	next := task.StatusCompleted

	if node.IsMaster() {
		switch m.state.Phase {
		case ExtractArchivePhaseNotStarted:
			next, err = m.masterExtractArchive(ctx, dep)
		case ExtractArchivePhaseDownloadZip:
			next, err = m.masterDownloadZip(ctx, dep)
		default:
			next, err = task.StatusError, fmt.Errorf("unknown phase %q: %w", m.state.Phase, queue.CriticalErr)
		}
	} else {
		switch m.state.Phase {
		case ExtractArchivePhaseNotStarted:
			next, err = m.createSlaveExtractTask(ctx, dep)
		case ExtractArchivePhaseAwaitSlaveComplete:
			next, err = m.awaitSlaveExtractComplete(ctx, dep)
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

func (m *ExtractArchiveTask) createSlaveExtractTask(ctx context.Context, dep dependency.Dep) (task.Status, error) {
	uri, err := fs.NewUriFromString(m.state.Uri)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to parse src uri: %s (%w)", err, queue.CriticalErr)
	}

	user := inventory.UserFromContext(ctx)
	fm := manager.NewFileManager(dep, user)

	// Get entity source to extract
	archiveFile, err := fm.Get(ctx, uri, dbfs.WithFileEntities(), dbfs.WithRequiredCapabilities(dbfs.NavigatorCapabilityDownloadFile), dbfs.WithNotRoot())
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to get archive file: %s (%w)", err, queue.CriticalErr)
	}

	// Validate file size
	if user.Edges.Group.Settings.DecompressSize > 0 && archiveFile.Size() > user.Edges.Group.Settings.DecompressSize {
		return task.StatusError,
			fmt.Errorf("file size %d exceeds the limit %d (%w)", archiveFile.Size(), user.Edges.Group.Settings.DecompressSize, queue.CriticalErr)
	}

	// Create slave task
	storagePolicyClient := dep.StoragePolicyClient()
	policy, err := storagePolicyClient.GetPolicyByID(ctx, archiveFile.PrimaryEntity().PolicyID())
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to get policy: %w", err)
	}

	masterKey, _ := dep.MasterEncryptKeyVault(ctx).GetMasterKey(ctx)
	entityModel, err := decryptEntityKeyIfNeeded(masterKey, archiveFile.PrimaryEntity().Model())
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to decrypt entity key for archive file %q: %s", archiveFile.DisplayName(), err)
	}

	payload := &SlaveExtractArchiveTaskState{
		FileName: archiveFile.DisplayName(),
		Entity:   entityModel,
		Policy:   policy,
		Encoding: m.state.Encoding,
		Dst:      m.state.Dst,
		UserID:   user.ID,
		Password: m.state.Password,
		FileMask: m.state.FileMask,
	}

	payloadStr, err := json.Marshal(payload)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to marshal payload: %w", err)
	}

	taskId, err := m.node.CreateTask(ctx, queue.SlaveExtractArchiveType, string(payloadStr))
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to create slave task: %w", err)
	}

	m.state.Phase = ExtractArchivePhaseAwaitSlaveComplete
	m.state.SlaveTaskID = taskId
	m.ResumeAfter((10 * time.Second))
	return task.StatusSuspending, nil
}

func (m *ExtractArchiveTask) awaitSlaveExtractComplete(ctx context.Context, dep dependency.Dep) (task.Status, error) {
	t, err := m.node.GetTask(ctx, m.state.SlaveTaskID, true)
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
		return task.StatusCompleted, nil
	}

	m.l.Info("Slave task %d is still compressing, resume after 30s.", m.state.SlaveTaskID)
	m.ResumeAfter((time.Second * 30))
	return task.StatusSuspending, nil
}

func (m *ExtractArchiveTask) masterExtractArchive(ctx context.Context, dep dependency.Dep) (task.Status, error) {
	uri, err := fs.NewUriFromString(m.state.Uri)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to parse src uri: %s (%w)", err, queue.CriticalErr)
	}

	dst, err := fs.NewUriFromString(m.state.Dst)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to parse dst uri: %s (%w)", err, queue.CriticalErr)
	}

	user := inventory.UserFromContext(ctx)
	fm := manager.NewFileManager(dep, user)

	// Get entity source to extract
	archiveFile, err := fm.Get(ctx, uri, dbfs.WithFileEntities(), dbfs.WithRequiredCapabilities(dbfs.NavigatorCapabilityDownloadFile), dbfs.WithNotRoot())
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to get archive file: %s (%w)", err, queue.CriticalErr)
	}

	// Validate file size
	if user.Edges.Group.Settings.DecompressSize > 0 && archiveFile.Size() > user.Edges.Group.Settings.DecompressSize {
		return task.StatusError,
			fmt.Errorf("file size %d exceeds the limit %d (%w)", archiveFile.Size(), user.Edges.Group.Settings.DecompressSize, queue.CriticalErr)
	}

	es, err := fm.GetEntitySource(ctx, 0, fs.WithEntity(archiveFile.PrimaryEntity()))
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to get entity source: %w", err)
	}

	defer es.Close()

	m.l.Info("Extracting archive %q to %q", uri, m.state.Dst)
	// Identify file format
	format, readStream, err := archives.Identify(ctx, archiveFile.DisplayName(), es)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to identify archive format: %w", err)
	}

	m.l.Info("Archive file %q format identified as %q", uri, format.Extension())

	extractor, ok := format.(archives.Extractor)
	if !ok {
		return task.StatusError, fmt.Errorf("format not an extractor %s", format.Extension())
	}

	formatExt := format.Extension()
	if formatExt == ".zip" || formatExt == ".7z" {
		// Zip/7Z extractor requires a Seeker+ReadAt
		if m.state.TempZipFilePath == "" && !es.IsLocal() {
			m.state.Phase = ExtractArchivePhaseDownloadZip
			m.ResumeAfter(0)
			return task.StatusSuspending, nil
		}

		if m.state.TempZipFilePath != "" {
			// Use temp zip file path
			zipFile, err := os.Open(m.state.TempZipFilePath)
			if err != nil {
				return task.StatusError, fmt.Errorf("failed to open temp zip file: %w", err)
			}

			defer zipFile.Close()
			readStream = zipFile
		}

		if es.IsLocal() {
			if _, err = es.Seek(0, 0); err != nil {
				return task.StatusError, fmt.Errorf("failed to seek entity source: %w", err)
			}

			readStream = es
		}
	}

	if zipExtractor, ok := extractor.(archives.Zip); ok {
		if m.state.Encoding != "" {
			m.l.Info("Using encoding %q for zip archive", m.state.Encoding)
			encoding, ok := manager.ZipEncodings[strings.ToLower(m.state.Encoding)]
			if !ok {
				m.l.Warning("Unknown encoding %q, fallback to default encoding", m.state.Encoding)
			} else {
				zipExtractor.TextEncoding = encoding
				extractor = zipExtractor
			}
		}
	} else if rarExtractor, ok := extractor.(archives.Rar); ok && m.state.Password != "" {
		rarExtractor.Password = m.state.Password
		extractor = rarExtractor
	} else if sevenZipExtractor, ok := extractor.(archives.SevenZip); ok && m.state.Password != "" {
		sevenZipExtractor.Password = m.state.Password
		extractor = sevenZipExtractor
	}

	needSkipToCursor := false
	if m.state.ProcessedCursor != "" {
		needSkipToCursor = true
	}
	m.Lock()
	m.progress[ProgressTypeExtractCount] = &queue.Progress{}
	m.progress[ProgressTypeExtractSize] = &queue.Progress{}
	m.Unlock()

	// extract and upload
	err = extractor.Extract(ctx, readStream, func(ctx context.Context, f archives.FileInfo) error {
		if needSkipToCursor && f.NameInArchive != m.state.ProcessedCursor {
			atomic.AddInt64(&m.progress[ProgressTypeExtractCount].Current, 1)
			atomic.AddInt64(&m.progress[ProgressTypeExtractSize].Current, f.Size())
			m.l.Info("File %q already processed, skipping...", f.NameInArchive)
			return nil
		}

		// Found cursor, start from cursor +1
		if m.state.ProcessedCursor == f.NameInArchive {
			atomic.AddInt64(&m.progress[ProgressTypeExtractCount].Current, 1)
			atomic.AddInt64(&m.progress[ProgressTypeExtractSize].Current, f.Size())
			needSkipToCursor = false
			return nil
		}

		rawPath := util.FormSlash(f.NameInArchive)
		savePath := dst.JoinRaw(rawPath)

		// If file mask is not empty, check if the path is in the mask
		if len(m.state.FileMask) > 0 && !isFileInMask(rawPath, m.state.FileMask) {
			m.l.Warning("File %q is not in the mask, skipping...", f.NameInArchive)
			atomic.AddInt64(&m.progress[ProgressTypeExtractCount].Current, 1)
			atomic.AddInt64(&m.progress[ProgressTypeExtractSize].Current, f.Size())
			return nil
		}

		// Check if path is legit
		if !strings.HasPrefix(savePath.Path(), util.FillSlash(path.Clean(dst.Path()))) {
			m.l.Warning("Path %q is not legit, skipping...", f.NameInArchive)
			atomic.AddInt64(&m.progress[ProgressTypeExtractCount].Current, 1)
			atomic.AddInt64(&m.progress[ProgressTypeExtractSize].Current, f.Size())
			return nil
		}

		if f.FileInfo.IsDir() {
			_, err := fm.Create(ctx, savePath, types.FileTypeFolder)
			if err != nil {
				m.l.Warning("Failed to create directory %q: %s, skipping...", rawPath, err)
			}

			atomic.AddInt64(&m.progress[ProgressTypeExtractCount].Current, 1)
			m.state.ProcessedCursor = f.NameInArchive
			return nil
		}

		fileStream, err := f.Open()
		if err != nil {
			m.l.Warning("Failed to open file %q in archive file: %s, skipping...", rawPath, err)
			return nil
		}

		fileData := &fs.UploadRequest{
			Props: &fs.UploadProps{
				Uri:  savePath,
				Size: f.Size(),
				LastModified: func() *time.Time {
					t := f.FileInfo.ModTime().Local()
					return &t
				}(),
			},
			ProgressFunc: func(current, diff int64, total int64) {
				atomic.AddInt64(&m.progress[ProgressTypeExtractSize].Current, diff)
			},
			File: fileStream,
		}

		_, err = fm.Update(ctx, fileData, fs.WithNoEntityType())
		if err != nil {
			return fmt.Errorf("failed to upload file %q in archive file: %w", rawPath, err)
		}

		atomic.AddInt64(&m.progress[ProgressTypeExtractCount].Current, 1)
		m.state.ProcessedCursor = f.NameInArchive
		return nil
	})

	if err != nil {
		return task.StatusError, fmt.Errorf("failed to extract archive: %w", err)
	}

	return task.StatusCompleted, nil
}

func (m *ExtractArchiveTask) masterDownloadZip(ctx context.Context, dep dependency.Dep) (task.Status, error) {
	uri, err := fs.NewUriFromString(m.state.Uri)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to parse src uri: %s (%w)", err, queue.CriticalErr)
	}

	user := inventory.UserFromContext(ctx)
	fm := manager.NewFileManager(dep, user)

	// Get entity source to extract
	archiveFile, err := fm.Get(ctx, uri, dbfs.WithFileEntities(), dbfs.WithRequiredCapabilities(dbfs.NavigatorCapabilityDownloadFile), dbfs.WithNotRoot())
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to get archive file: %s (%w)", err, queue.CriticalErr)
	}

	es, err := fm.GetEntitySource(ctx, 0, fs.WithEntity(archiveFile.PrimaryEntity()))
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to get entity source: %w", err)
	}

	defer es.Close()

	// For non-local entity, we need to download the whole zip file first
	tempPath, err := prepareTempFolder(ctx, dep, m)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to prepare temp folder: %w", err)
	}
	m.state.TempPath = tempPath

	fileName := fmt.Sprintf("%s.zip", uuid.Must(uuid.NewV4()))
	zipFilePath := filepath.Join(
		m.state.TempPath,
		fileName,
	)

	zipFile, err := util.CreatNestedFile(zipFilePath)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to create zip file: %w", err)
	}

	m.Lock()
	m.progress[ProgressTypeDownload] = &queue.Progress{Total: es.Entity().Size()}
	m.Unlock()

	defer zipFile.Close()
	if _, err := io.Copy(zipFile, util.NewCallbackReader(es, func(i int64) {
		atomic.AddInt64(&m.progress[ProgressTypeDownload].Current, i)
	})); err != nil {
		zipFile.Close()
		if err := os.Remove(zipFilePath); err != nil {
			m.l.Warning("Failed to remove temp zip file %q: %s", zipFilePath, err)
		}
		return task.StatusError, fmt.Errorf("failed to copy zip file to local temp: %w", err)
	}

	m.Lock()
	delete(m.progress, ProgressTypeDownload)
	m.Unlock()
	m.state.TempZipFilePath = zipFilePath
	m.state.Phase = ExtractArchivePhaseNotStarted
	m.ResumeAfter(0)
	return task.StatusSuspending, nil
}

func (m *ExtractArchiveTask) Summarize(hasher hashid.Encoder) *queue.Summary {
	if m.state == nil {
		if err := json.Unmarshal([]byte(m.State()), &m.state); err != nil {
			return nil
		}
	}

	return &queue.Summary{
		NodeID: m.state.NodeID,
		Phase:  string(m.state.Phase),
		Props: map[string]any{
			SummaryKeySrc: m.state.Uri,
			SummaryKeyDst: m.state.Dst,
		},
	}
}

func (m *ExtractArchiveTask) Progress(ctx context.Context) queue.Progresses {
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

func (m *ExtractArchiveTask) Cleanup(ctx context.Context) error {
	if m.state.TempPath != "" {
		time.Sleep(time.Duration(1) * time.Second)
		return os.RemoveAll(m.state.TempPath)
	}

	return nil
}

type (
	SlaveExtractArchiveTask struct {
		*queue.InMemoryTask

		l        logging.Logger
		state    *SlaveExtractArchiveTaskState
		progress queue.Progresses
		node     cluster.Node
	}

	SlaveExtractArchiveTaskState struct {
		FileName        string             `json:"file_name"`
		Entity          *ent.Entity        `json:"entity"`
		Policy          *ent.StoragePolicy `json:"policy"`
		Encoding        string             `json:"encoding,omitempty"`
		Dst             string             `json:"dst,omitempty"`
		UserID          int                `json:"user_id"`
		TempPath        string             `json:"temp_path,omitempty"`
		TempZipFilePath string             `json:"temp_zip_file_path,omitempty"`
		ProcessedCursor string             `json:"processed_cursor,omitempty"`
		Password        string             `json:"password,omitempty"`
		FileMask        []string           `json:"file_mask,omitempty"`
	}
)

// NewSlaveExtractArchiveTask creates a new SlaveExtractArchiveTask from raw private state
func NewSlaveExtractArchiveTask(ctx context.Context, props *types.SlaveTaskProps, id int, state string) queue.Task {
	return &SlaveExtractArchiveTask{
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

func (m *SlaveExtractArchiveTask) Do(ctx context.Context) (task.Status, error) {
	ctx = prepareSlaveTaskCtx(ctx, m.Model().PublicState.SlaveTaskProps)
	dep := dependency.FromContext(ctx)
	m.l = dep.Logger()
	np, err := dep.NodePool(ctx)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to get node pool: %w", err)
	}

	m.node, err = np.Get(ctx, types.NodeCapabilityNone, 0)
	if err != nil || !m.node.IsMaster() {
		return task.StatusError, fmt.Errorf("failed to get master node: %w", err)
	}

	fm := manager.NewFileManager(dep, nil)

	// unmarshal state
	state := &SlaveExtractArchiveTaskState{}
	if err := json.Unmarshal([]byte(m.State()), state); err != nil {
		return task.StatusError, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	m.state = state
	m.Lock()
	if m.progress == nil {
		m.progress = make(queue.Progresses)
	}
	m.progress[ProgressTypeExtractCount] = &queue.Progress{}
	m.progress[ProgressTypeExtractSize] = &queue.Progress{}
	m.Unlock()

	dst, err := fs.NewUriFromString(m.state.Dst)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to parse dst uri: %s (%w)", err, queue.CriticalErr)
	}

	// 1. Get entity source
	entity := fs.NewEntity(m.state.Entity)
	es, err := fm.GetEntitySource(ctx, 0, fs.WithEntity(entity), fs.WithPolicy(fm.CastStoragePolicyOnSlave(ctx, m.state.Policy)))
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to get entity source: %w", err)
	}

	defer es.Close()

	// 2. Identify file format
	format, readStream, err := archives.Identify(ctx, m.state.FileName, es)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to identify archive format: %w", err)
	}
	m.l.Info("Archive file %q format identified as %q", m.state.FileName, format.Extension())

	extractor, ok := format.(archives.Extractor)
	if !ok {
		return task.StatusError, fmt.Errorf("format not an extractor %q", format.Extension())
	}

	formatExt := format.Extension()
	if formatExt == ".zip" || formatExt == ".7z" {
		if _, err = es.Seek(0, 0); err != nil {
			return task.StatusError, fmt.Errorf("failed to seek entity source: %w", err)
		}

		if m.state.TempZipFilePath == "" && !es.IsLocal() {
			tempPath, err := prepareTempFolder(ctx, dep, m)
			if err != nil {
				return task.StatusError, fmt.Errorf("failed to prepare temp folder: %w", err)
			}
			m.state.TempPath = tempPath

			fileName := fmt.Sprintf("%s.zip", uuid.Must(uuid.NewV4()))
			zipFilePath := filepath.Join(
				m.state.TempPath,
				fileName,
			)
			zipFile, err := util.CreatNestedFile(zipFilePath)
			if err != nil {
				return task.StatusError, fmt.Errorf("failed to create zip file: %w", err)
			}

			m.Lock()
			m.progress[ProgressTypeDownload] = &queue.Progress{Total: es.Entity().Size()}
			m.Unlock()

			defer zipFile.Close()
			if _, err := io.Copy(zipFile, util.NewCallbackReader(es, func(i int64) {
				atomic.AddInt64(&m.progress[ProgressTypeDownload].Current, i)
			})); err != nil {
				return task.StatusError, fmt.Errorf("failed to copy zip file to local temp: %w", err)
			}

			zipFile.Close()
			m.state.TempZipFilePath = zipFilePath
		}

		if es.IsLocal() {
			readStream = es
		} else if m.state.TempZipFilePath != "" {
			// Use temp zip file path
			zipFile, err := os.Open(m.state.TempZipFilePath)
			if err != nil {
				return task.StatusError, fmt.Errorf("failed to open temp zip file: %w", err)
			}

			defer zipFile.Close()
			readStream = zipFile
		}

		if es.IsLocal() {
			readStream = es
		}
	}

	if zipExtractor, ok := extractor.(archives.Zip); ok {
		if m.state.Encoding != "" {
			m.l.Info("Using encoding %q for zip archive", m.state.Encoding)
			encoding, ok := manager.ZipEncodings[strings.ToLower(m.state.Encoding)]
			if !ok {
				m.l.Warning("Unknown encoding %q, fallback to default encoding", m.state.Encoding)
			} else {
				zipExtractor.TextEncoding = encoding
				extractor = zipExtractor
			}
		}
	} else if rarExtractor, ok := extractor.(archives.Rar); ok && m.state.Password != "" {
		rarExtractor.Password = m.state.Password
		extractor = rarExtractor
	} else if sevenZipExtractor, ok := extractor.(archives.SevenZip); ok && m.state.Password != "" {
		sevenZipExtractor.Password = m.state.Password
		extractor = sevenZipExtractor
	}

	needSkipToCursor := false
	if m.state.ProcessedCursor != "" {
		needSkipToCursor = true
	}

	// 3. Extract and upload
	err = extractor.Extract(ctx, readStream, func(ctx context.Context, f archives.FileInfo) error {
		if needSkipToCursor && f.NameInArchive != m.state.ProcessedCursor {
			atomic.AddInt64(&m.progress[ProgressTypeExtractCount].Current, 1)
			atomic.AddInt64(&m.progress[ProgressTypeExtractSize].Current, f.Size())
			m.l.Info("File %q already processed, skipping...", f.NameInArchive)
			return nil
		}

		// Found cursor, start from cursor +1
		if m.state.ProcessedCursor == f.NameInArchive {
			atomic.AddInt64(&m.progress[ProgressTypeExtractCount].Current, 1)
			atomic.AddInt64(&m.progress[ProgressTypeExtractSize].Current, f.Size())
			needSkipToCursor = false
			return nil
		}

		rawPath := util.FormSlash(f.NameInArchive)
		savePath := dst.JoinRaw(rawPath)

		// If file mask is not empty, check if the path is in the mask
		if len(m.state.FileMask) > 0 && !isFileInMask(rawPath, m.state.FileMask) {
			m.l.Debug("File %q is not in the mask, skipping...", f.NameInArchive)
			return nil
		}

		// Check if path is legit
		if !strings.HasPrefix(savePath.Path(), util.FillSlash(path.Clean(dst.Path()))) {
			atomic.AddInt64(&m.progress[ProgressTypeExtractCount].Current, 1)
			atomic.AddInt64(&m.progress[ProgressTypeExtractSize].Current, f.Size())
			m.l.Warning("Path %q is not legit, skipping...", f.NameInArchive)
			return nil
		}

		if f.FileInfo.IsDir() {
			_, err := fm.Create(ctx, savePath, types.FileTypeFolder, fs.WithNode(m.node), fs.WithStatelessUserID(m.state.UserID))
			if err != nil {
				m.l.Warning("Failed to create directory %q: %s, skipping...", rawPath, err)
			}

			atomic.AddInt64(&m.progress[ProgressTypeExtractCount].Current, 1)
			m.state.ProcessedCursor = f.NameInArchive
			return nil
		}

		fileStream, err := f.Open()
		if err != nil {
			m.l.Warning("Failed to open file %q in archive file: %s, skipping...", rawPath, err)
			return nil
		}

		fileData := &fs.UploadRequest{
			Props: &fs.UploadProps{
				Uri:  savePath,
				Size: f.Size(),
				LastModified: func() *time.Time {
					t := f.FileInfo.ModTime().Local()
					return &t
				}(),
			},
			ProgressFunc: func(current, diff int64, total int64) {
				atomic.AddInt64(&m.progress[ProgressTypeExtractSize].Current, diff)
			},
			File: fileStream,
		}

		_, err = fm.Update(ctx, fileData, fs.WithNode(m.node), fs.WithStatelessUserID(m.state.UserID), fs.WithNoEntityType())
		if err != nil {
			return fmt.Errorf("failed to upload file %q in archive file: %w", rawPath, err)
		}

		atomic.AddInt64(&m.progress[ProgressTypeExtractCount].Current, 1)
		m.state.ProcessedCursor = f.NameInArchive
		return nil
	})

	if err != nil {
		return task.StatusError, fmt.Errorf("failed to extract archive: %w", err)
	}

	return task.StatusCompleted, nil
}

func (m *SlaveExtractArchiveTask) Cleanup(ctx context.Context) error {
	if m.state.TempPath != "" {
		time.Sleep(time.Duration(1) * time.Second)
		return os.RemoveAll(m.state.TempPath)
	}

	return nil
}

func (m *SlaveExtractArchiveTask) Progress(ctx context.Context) queue.Progresses {
	m.Lock()
	defer m.Unlock()
	return m.progress
}

func isFileInMask(path string, mask []string) bool {
	if len(mask) == 0 {
		return true
	}

	for _, m := range mask {
		if path == m || strings.HasPrefix(path, m+"/") {
			return true
		}
	}

	return false
}
