package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/task"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gofrs/uuid"
	"github.com/samber/lo"
)

type (
	UploadManagement interface {
		// CreateUploadSession creates a upload session for given upload request
		CreateUploadSession(ctx context.Context, req *fs.UploadRequest, opts ...fs.Option) (*fs.UploadCredential, error)
		// ConfirmUploadSession confirms whether upload session is valid for upload.
		ConfirmUploadSession(ctx context.Context, session *fs.UploadSession, chunkIndex int) (fs.File, error)
		// Upload uploads file data to storage
		Upload(ctx context.Context, req *fs.UploadRequest, policy *ent.StoragePolicy) error
		// CompleteUpload completes upload session and returns file object
		CompleteUpload(ctx context.Context, session *fs.UploadSession) (fs.File, error)
		// CancelUploadSession cancels upload session
		CancelUploadSession(ctx context.Context, path *fs.URI, sessionID string) error
		// OnUploadFailed should be called when an unmanaged upload failed before complete.
		OnUploadFailed(ctx context.Context, session *fs.UploadSession)
		// Similar to CompleteUpload, but does not create actual uplaod session in storage.
		PrepareUpload(ctx context.Context, req *fs.UploadRequest, opts ...fs.Option) (*fs.UploadSession, error)
		// PreValidateUpload pre-validates an upload request.
		PreValidateUpload(ctx context.Context, dst *fs.URI, files ...fs.PreValidateFile) error
	}
)

func (m *manager) PreValidateUpload(ctx context.Context, dst *fs.URI, files ...fs.PreValidateFile) error {
	return m.fs.PreValidateUpload(ctx, dst, files...)
}

func (m *manager) CreateUploadSession(ctx context.Context, req *fs.UploadRequest, opts ...fs.Option) (*fs.UploadCredential, error) {
	o := newOption()
	for _, opt := range opts {
		opt.Apply(o)
	}

	// Validate metadata
	if req.Props.Metadata != nil {
		if _, err := m.validateMetadata(ctx, lo.MapToSlice(req.Props.Metadata, func(key string, value string) fs.MetadataPatch {
			return fs.MetadataPatch{
				Key:   key,
				Value: value,
			}
		})...); err != nil {
			return nil, err
		}
	}

	uploadSession := o.UploadSession
	var (
		err error
	)

	if uploadSession == nil {
		// If upload session not specified, invoke DBFS to create one
		sessionID := uuid.Must(uuid.NewV4()).String()
		req.Props.UploadSessionID = sessionID
		ttl := m.settings.UploadSessionTTL(ctx)
		req.Props.ExpireAt = time.Now().Add(ttl)

		// Prepare for upload
		uploadSession, err = m.fs.PrepareUpload(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("faield to prepare uplaod: %w", err)
		}
	}

	d, err := m.GetStorageDriver(ctx, m.CastStoragePolicyOnSlave(ctx, uploadSession.Policy))
	if err != nil {
		m.OnUploadFailed(ctx, uploadSession)
		return nil, err
	}

	uploadSession.ChunkSize = uploadSession.Policy.Settings.ChunkSize
	// Create upload credential for underlying storage driver
	credential := &fs.UploadCredential{}
	if !uploadSession.Policy.Settings.Relay || m.stateless {
		credential, err = d.Token(ctx, uploadSession, req)
		if err != nil {
			m.OnUploadFailed(ctx, uploadSession)
			return nil, err
		}
	} else {
		// For relayed upload, we don't need to create credential
		uploadSession.ChunkSize = 0
		credential.ChunkSize = 0
	}
	credential.SessionID = uploadSession.Props.UploadSessionID
	credential.Expires = req.Props.ExpireAt.Unix()
	credential.StoragePolicy = uploadSession.Policy
	credential.CallbackSecret = uploadSession.CallbackSecret
	credential.Uri = uploadSession.Props.Uri.String()

	// If upload sentinel check is required, queue a check task
	if d.Capabilities().StaticFeatures.Enabled(int(driver.HandlerCapabilityUploadSentinelRequired)) {
		t, err := newUploadSentinelCheckTask(ctx, uploadSession)
		if err != nil {
			m.OnUploadFailed(ctx, uploadSession)
			return nil, fmt.Errorf("failed to create upload sentinel check task: %w", err)
		}

		if err := m.dep.EntityRecycleQueue(ctx).QueueTask(ctx, t); err != nil {
			m.OnUploadFailed(ctx, uploadSession)
			return nil, fmt.Errorf("failed to queue upload sentinel check task: %w", err)
		}

		uploadSession.SentinelTaskID = t.ID()
	}

	err = m.kv.Set(
		UploadSessionCachePrefix+req.Props.UploadSessionID,
		*uploadSession,
		max(1, int(req.Props.ExpireAt.Sub(time.Now()).Seconds())),
	)
	if err != nil {
		m.OnUploadFailed(ctx, uploadSession)
		return nil, err
	}

	return credential, nil
}

func (m *manager) ConfirmUploadSession(ctx context.Context, session *fs.UploadSession, chunkIndex int) (fs.File, error) {
	// Get placeholder file
	file, err := m.fs.Get(ctx, session.Props.Uri)
	if err != nil {
		return nil, fmt.Errorf("failed to get placeholder file: %w", err)
	}

	// Confirm locks on placeholder file
	if session.LockToken == "" {
		release, ls, err := m.fs.ConfirmLock(ctx, file, file.Uri(false), session.LockToken)
		if err != nil {
			return nil, fs.ErrLockExpired.WithError(err)
		}

		defer release()
		ctx = fs.LockSessionToContext(ctx, ls)
	}

	// Make sure this storage policy is OK to receive data from clients to Cloudreve server.
	if session.Policy.Type != types.PolicyTypeLocal && !session.Policy.Settings.Relay {
		return nil, serializer.NewError(serializer.CodePolicyNotAllowed, "", nil)
	}

	actualSizeStart := int64(chunkIndex) * session.ChunkSize
	if session.Policy.Settings.ChunkSize == 0 && chunkIndex > 0 {
		return nil, serializer.NewError(serializer.CodeInvalidChunkIndex, "Chunk index cannot be greater than 0", nil)
	}

	if actualSizeStart > 0 && actualSizeStart >= session.Props.Size {
		return nil, serializer.NewError(serializer.CodeInvalidChunkIndex, "Chunk offset cannot be greater than file size", nil)
	}

	return file, nil
}

func (m *manager) PrepareUpload(ctx context.Context, req *fs.UploadRequest, opts ...fs.Option) (*fs.UploadSession, error) {
	return m.fs.PrepareUpload(ctx, req, opts...)
}

func (m *manager) Upload(ctx context.Context, req *fs.UploadRequest, policy *ent.StoragePolicy) error {
	d, err := m.GetStorageDriver(ctx, m.CastStoragePolicyOnSlave(ctx, policy))
	if err != nil {
		return err
	}

	if err := d.Put(ctx, req); err != nil {
		return serializer.NewError(serializer.CodeIOFailed, "Failed to upload file", err)
	}

	return nil
}

func (m *manager) CancelUploadSession(ctx context.Context, path *fs.URI, sessionID string) error {
	// Get upload session
	var session *fs.UploadSession
	sessionRaw, ok := m.kv.Get(UploadSessionCachePrefix + sessionID)
	if ok {
		sessionTyped := sessionRaw.(fs.UploadSession)
		session = &sessionTyped
	}

	var (
		staleEntities []fs.Entity
		err           error
	)

	if !m.stateless {
		staleEntities, err = m.fs.CancelUploadSession(ctx, path, sessionID, session)
		if err != nil {
			return err
		}

		m.l.Debug("New stale entities: %v", staleEntities)
	}

	if session != nil {
		ctx = context.WithValue(ctx, cluster.SlaveNodeIDCtx{}, strconv.Itoa(session.Policy.NodeID))
		d, err := m.GetStorageDriver(ctx, m.CastStoragePolicyOnSlave(ctx, session.Policy))
		if err != nil {
			return fmt.Errorf("failed to get storage driver: %w", err)
		}

		if m.stateless {
			if _, err = d.Delete(ctx, session.Props.SavePath); err != nil {
				return fmt.Errorf("failed to delete file: %w", err)
			}
		} else {
			if err = d.CancelToken(ctx, session); err != nil {
				return fmt.Errorf("failed to cancel upload session: %w", err)
			}
		}

		m.kv.Delete(UploadSessionCachePrefix, session.Props.UploadSessionID)
	}

	// Delete stale entities
	if len(staleEntities) > 0 {
		t, err := newExplicitEntityRecycleTask(ctx, lo.Map(staleEntities, func(entity fs.Entity, index int) int {
			return entity.ID()
		}))
		if err != nil {
			return fmt.Errorf("failed to create explicit entity recycle task: %w", err)
		}

		if err := m.dep.EntityRecycleQueue(ctx).QueueTask(ctx, t); err != nil {
			return fmt.Errorf("failed to queue explicit entity recycle task: %w", err)
		}
	}

	return nil
}

func (m *manager) CompleteUpload(ctx context.Context, session *fs.UploadSession) (fs.File, error) {
	d, err := m.GetStorageDriver(ctx, m.CastStoragePolicyOnSlave(ctx, session.Policy))
	if err != nil {
		return nil, err
	}

	if err := d.CompleteUpload(ctx, session); err != nil {
		return nil, err
	}

	var (
		file fs.File
	)
	if m.fs != nil {
		file, err = m.fs.CompleteUpload(ctx, session)
		if err != nil {
			return nil, fmt.Errorf("failed to complete upload: %w", err)
		}
	}

	if session.SentinelTaskID > 0 {
		// Cancel sentinel check task
		m.l.Debug("Cancel upload sentinel check task [%d].", session.SentinelTaskID)
		if err := m.dep.TaskClient().SetCompleteByID(ctx, session.SentinelTaskID); err != nil {
			m.l.Warning("Failed to set upload sentinel check task [%d] to complete: %s", session.SentinelTaskID, err)
		}
	}

	m.onNewEntityUploaded(ctx, session, d)
	// Remove upload session
	_ = m.kv.Delete(UploadSessionCachePrefix, session.Props.UploadSessionID)
	return file, nil
}

func (m *manager) Update(ctx context.Context, req *fs.UploadRequest, opts ...fs.Option) (fs.File, error) {
	o := newOption()
	for _, opt := range opts {
		opt.Apply(o)
	}
	entityType := types.EntityTypeVersion
	if o.EntityType != nil {
		entityType = *o.EntityType
	}

	req.Props.EntityType = &entityType
	if o.EntityTypeNil {
		req.Props.EntityType = nil
	}

	req.Props.UploadSessionID = uuid.Must(uuid.NewV4()).String()

	if m.stateless {
		return m.updateStateless(ctx, req, o)
	}

	// Prepare for upload
	uploadSession, err := m.fs.PrepareUpload(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("faield to prepare uplaod: %w", err)
	}

	if err := m.Upload(ctx, req, uploadSession.Policy); err != nil {
		m.OnUploadFailed(ctx, uploadSession)
		return nil, fmt.Errorf("failed to upload new entity: %w", err)
	}

	file, err := m.CompleteUpload(ctx, uploadSession)
	if err != nil {
		m.OnUploadFailed(ctx, uploadSession)
		return nil, fmt.Errorf("failed to complete update: %w", err)
	}

	return file, nil
}

func (m *manager) OnUploadFailed(ctx context.Context, session *fs.UploadSession) {
	ctx = context.WithoutCancel(ctx)
	if !m.stateless {
		if session.LockToken != "" {
			if err := m.Unlock(ctx, session.LockToken); err != nil {
				m.l.Warning("OnUploadFailed hook failed to unlock: %s", err)
			}
		}

		if session.NewFileCreated {
			if err := m.Delete(ctx, []*fs.URI{session.Props.Uri}, fs.WithSysSkipSoftDelete(true)); err != nil {
				m.l.Warning("OnUploadFailed hook failed to delete file: %s", err)
			}
		} else if !session.Importing {
			if err := m.fs.VersionControl(ctx, session.Props.Uri, session.EntityID, true); err != nil {
				m.l.Warning("OnUploadFailed hook failed to version control: %s", err)
			}
		}
	} else {
		d, err := m.GetStorageDriver(ctx, m.CastStoragePolicyOnSlave(ctx, session.Policy))
		if err != nil {
			m.l.Warning("OnUploadFailed hook failed: %s", err)
		}

		if failed, err := d.Delete(ctx, session.Props.SavePath); err != nil {
			m.l.Warning("OnUploadFailed hook failed to remove uploaded file: %s, failed file: %v", err, failed)
		}
	}
}

// similar to Update, but expected to be executed on slave node.
func (m *manager) updateStateless(ctx context.Context, req *fs.UploadRequest, o *fs.FsOption) (fs.File, error) {
	// Prepare for upload
	res, err := o.Node.PrepareUpload(ctx, &fs.StatelessPrepareUploadService{
		UploadRequest: req,
		UserID:        o.StatelessUserID,
	})
	if err != nil {
		return nil, fmt.Errorf("faield to prepare uplaod: %w", err)
	}

	if res.Req != nil {
		req.Props = res.Req.Props
	}
	if err := m.Upload(ctx, req, res.Session.Policy); err != nil {
		if err := o.Node.OnUploadFailed(ctx, &fs.StatelessOnUploadFailedService{
			UploadSession: res.Session,
			UserID:        o.StatelessUserID,
		}); err != nil {
			m.l.Warning("Failed to call stateless OnUploadFailed: %s", err)
		}
		return nil, fmt.Errorf("failed to upload new entity: %w", err)
	}

	err = o.Node.CompleteUpload(ctx, &fs.StatelessCompleteUploadService{
		UploadSession: res.Session,
		UserID:        o.StatelessUserID,
	})
	if err != nil {
		if err := o.Node.OnUploadFailed(ctx, &fs.StatelessOnUploadFailedService{
			UploadSession: res.Session,
			UserID:        o.StatelessUserID,
		}); err != nil {
			m.l.Warning("Failed to call stateless OnUploadFailed: %s", err)
		}
		return nil, fmt.Errorf("failed to complete update: %w", err)
	}

	return nil, nil
}

func (m *manager) onNewEntityUploaded(ctx context.Context, session *fs.UploadSession, d driver.Handler) {
	if !m.stateless {
		// Submit media meta task for new entity
		m.mediaMetaForNewEntity(ctx, session, d)
	}
}

// Upload sentinel check task is used for compliant storage policy (COS, S3...), it will delete the marked entity.
// It is expected to be queued after upload session is created, and canceled after upload callback is completed.
// If this task is executed, it means the upload callback does not complete in time.
type (
	UploadSentinelCheckTask struct {
		*queue.DBTask
	}
	UploadSentinelCheckTaskState struct {
		Session *fs.UploadSession `json:"session"`
	}
)

const (
	uploadSentinelCheckMargin = 5 * time.Minute
)

func init() {
	queue.RegisterResumableTaskFactory(queue.UploadSentinelCheckTaskType, NewUploadSentinelCheckTaskFromModel)
}

func NewUploadSentinelCheckTaskFromModel(task *ent.Task) queue.Task {
	return &UploadSentinelCheckTask{
		DBTask: &queue.DBTask{
			Task: task,
		},
	}
}

func newUploadSentinelCheckTask(ctx context.Context, uploadSession *fs.UploadSession) (*ExplicitEntityRecycleTask, error) {
	state := &UploadSentinelCheckTaskState{
		Session: uploadSession,
	}
	stateBytes, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state: %w", err)
	}

	resumeAfter := uploadSession.Props.ExpireAt.Add(uploadSentinelCheckMargin)
	t := &ExplicitEntityRecycleTask{
		DBTask: &queue.DBTask{
			Task: &ent.Task{
				Type:          queue.UploadSentinelCheckTaskType,
				CorrelationID: logging.CorrelationID(ctx),
				PrivateState:  string(stateBytes),
				PublicState: &types.TaskPublicState{
					ResumeTime: resumeAfter.Unix(),
				},
			},
			DirectOwner: inventory.UserFromContext(ctx),
		},
	}
	return t, nil
}

func (m *UploadSentinelCheckTask) Do(ctx context.Context) (task.Status, error) {
	dep := dependency.FromContext(ctx)
	taskClient := dep.TaskClient()
	l := dep.Logger()
	fm := NewFileManager(dep, inventory.UserFromContext(ctx)).(*manager)

	// Check if sentinel is canceled due to callback complete
	t, err := taskClient.GetTaskByID(ctx, m.ID())
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to get task by ID: %w", err)
	}

	if t.Status == task.StatusCompleted {
		l.Info("Upload sentinel check task [%d] is canceled due to callback complete.", m.ID())
		return task.StatusCompleted, nil
	}

	// unmarshal state
	state := &UploadSentinelCheckTaskState{}
	if err := json.Unmarshal([]byte(m.State()), state); err != nil {
		return task.StatusError, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	l.Info("Upload sentinel check triggered, clean up stale place holder entity [%d].", state.Session.EntityID)
	entity, err := fm.fs.GetEntity(ctx, state.Session.EntityID)
	if err != nil {
		l.Debug("Failed to get entity [%d]: %s, skip sentinel check.", state.Session.EntityID, err)
		return task.StatusCompleted, nil
	}

	_, d, err := fm.getEntityPolicyDriver(ctx, entity, nil)
	if err != nil {
		l.Debug("Failed to get storage driver for entity [%d]: %s", state.Session.EntityID, err)
		return task.StatusError, err
	}

	_, err = d.Delete(ctx, entity.Source())
	if err != nil {
		l.Debug("Failed to delete entity source [%d]: %s", state.Session.EntityID, err)
		return task.StatusError, err
	}

	if err := d.CancelToken(ctx, state.Session); err != nil {
		l.Debug("Failed to cancel token [%d]: %s", state.Session.EntityID, err)
	}

	return task.StatusCompleted, nil
}
