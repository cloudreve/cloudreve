package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sort"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/entity"
	"github.com/cloudreve/Cloudreve/v4/ent/storagepolicy"
	"github.com/cloudreve/Cloudreve/v4/ent/task"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
)

type (
	RelocateTask struct {
		*queue.DBTask

		l        logging.Logger
		state    *RelocateTaskState
		progress queue.Progresses
	}
	RelocateTaskPhase string
	RelocateTaskState struct {
		// EntityIDs is an explicit set of entities to relocate, sorted ascending
		// at creation so Cursor resume works uniformly.
		EntityIDs []int `json:"entity_ids,omitempty"`
		// SrcPolicyID relocates every entity still on this policy instead of an
		// explicit list.
		SrcPolicyID int `json:"src_policy_id,omitempty"`
		DstPolicyID int `json:"dst_policy_id"`
		// Cursor is the ID of the last successfully processed entity.
		Cursor int `json:"cursor,omitempty"`
		// Failed maps entity ID to the error that prevented its relocation.
		Failed    map[string]string `json:"failed,omitempty"`
		NodeState `json:",inline"`
		Phase     RelocateTaskPhase `json:"phase,omitempty"`
	}
)

const (
	RelocatePhaseNotStarted RelocateTaskPhase = ""
	RelocatePhaseRelocating RelocateTaskPhase = "relocating"

	ProgressTypeRelocateCount = "relocate_count"
	ProgressTypeRelocateSize  = "relocate_size"

	relocatePageSize = 50

	SummaryKeyDstPolicy      = "dst_policy"
	SummaryKeySrcPolicy      = "src_policy"
	SummaryKeyEntities       = "entities"
	SummaryKeyRelocateFailed = "relocate_failed"
)

func init() {
	queue.RegisterResumableTaskFactory(queue.RelocateTaskType, NewRelocateTaskFromModel)
}

// NewRelocateTask creates a RelocateTask moving the given entities to dstPolicyID.
func NewRelocateTask(ctx context.Context, entityIDs []int, dstPolicyID int) (queue.Task, error) {
	ids := append([]int(nil), entityIDs...)
	sort.Ints(ids)
	return newRelocateTask(ctx, &RelocateTaskState{
		EntityIDs:   ids,
		DstPolicyID: dstPolicyID,
		NodeState:   NodeState{},
	})
}

// NewRelocatePolicyTask creates a RelocateTask moving every entity on
// srcPolicyID to dstPolicyID.
func NewRelocatePolicyTask(ctx context.Context, srcPolicyID, dstPolicyID int) (queue.Task, error) {
	return newRelocateTask(ctx, &RelocateTaskState{
		SrcPolicyID: srcPolicyID,
		DstPolicyID: dstPolicyID,
		NodeState:   NodeState{},
	})
}

func newRelocateTask(ctx context.Context, state *RelocateTaskState) (queue.Task, error) {
	stateBytes, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state: %w", err)
	}

	return &RelocateTask{
		DBTask: &queue.DBTask{
			Task: &ent.Task{
				Type:          queue.RelocateTaskType,
				CorrelationID: logging.CorrelationID(ctx),
				PrivateState:  string(stateBytes),
				PublicState:   &types.TaskPublicState{},
			},
			DirectOwner: inventory.UserFromContext(ctx),
		},
	}, nil
}

func NewRelocateTaskFromModel(task *ent.Task) queue.Task {
	return &RelocateTask{
		DBTask: &queue.DBTask{
			Task: task,
		},
	}
}

func (m *RelocateTask) Do(ctx context.Context) (task.Status, error) {
	dep := dependency.FromContext(ctx)
	m.l = dep.Logger()

	m.Lock()
	if m.progress == nil {
		m.progress = make(queue.Progresses)
	}
	m.Unlock()

	state := &RelocateTaskState{}
	if err := json.Unmarshal([]byte(m.State()), state); err != nil {
		return task.StatusError, fmt.Errorf("failed to unmarshal state: %w", err)
	}
	m.state = state

	next, err := m.relocate(ctx, dep)

	newStateStr, marshalErr := json.Marshal(m.state)
	if marshalErr != nil {
		return task.StatusError, fmt.Errorf("failed to marshal state: %w", marshalErr)
	}

	m.Lock()
	m.Task.PrivateState = string(newStateStr)
	m.Unlock()
	return next, err
}

func (m *RelocateTask) relocate(ctx context.Context, dep dependency.Dep) (task.Status, error) {
	if m.state.SrcPolicyID != 0 && m.state.SrcPolicyID == m.state.DstPolicyID {
		return task.StatusError, fmt.Errorf("source and destination policies are identical (%w)", queue.CriticalErr)
	}

	policyClient := dep.StoragePolicyClient()
	dstPolicy, err := policyClient.GetPolicyByID(ctx, m.state.DstPolicyID)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to get destination policy: %w", err)
	}
	if dstPolicy == nil {
		return task.StatusError, fmt.Errorf("destination policy %d does not exist (%w)", m.state.DstPolicyID, queue.CriticalErr)
	}
	if dstPolicy.Status == storagepolicy.StatusSuspended {
		return task.StatusError, fmt.Errorf("destination policy %q is suspended (%w)", dstPolicy.Name, queue.CriticalErr)
	}

	m.state.Phase = RelocatePhaseRelocating
	if m.state.Failed == nil {
		m.state.Failed = make(map[string]string)
	}

	fm := manager.NewFileManager(dep, nil)
	defer fm.Recycle()

	totalCount, totalSize := m.progressTotals(ctx, dep)
	m.Lock()
	m.progress[ProgressTypeRelocateCount] = &queue.Progress{Total: totalCount}
	m.progress[ProgressTypeRelocateSize] = &queue.Progress{Total: totalSize}
	m.Unlock()

	pending, err := m.pendingEntities(ctx, dep)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to list pending entities: %w", err)
	}

	for len(pending) > 0 {
		for _, e := range pending {
			if err := ctx.Err(); err != nil {
				m.ResumeAfter(0)
				return task.StatusSuspending, nil
			}

			if err := m.relocateEntity(ctx, dep, fm, dstPolicy, e); err != nil {
				m.l.Warning("Failed to relocate entity %d: %s", e.ID, err)
				m.state.Failed[strconv.Itoa(e.ID)] = err.Error()
			}
			m.state.Cursor = e.ID
		}

		pending, err = m.pendingEntities(ctx, dep)
		if err != nil {
			return task.StatusError, fmt.Errorf("failed to list pending entities: %w", err)
		}
	}

	return task.StatusCompleted, nil
}

// pendingEntities returns the next batch of entities still awaiting relocation.
// For explicit lists it walks the sorted IDs past the cursor; for policy
// migration it pages entities still assigned to the source policy.
func (m *RelocateTask) pendingEntities(ctx context.Context, dep dependency.Dep) ([]*ent.Entity, error) {
	if len(m.state.EntityIDs) > 0 {
		var ids []int
		for _, id := range m.state.EntityIDs {
			if id > m.state.Cursor {
				ids = append(ids, id)
			}
		}
		if len(ids) == 0 {
			return nil, nil
		}
		if len(ids) > relocatePageSize {
			ids = ids[:relocatePageSize]
		}
		return dep.DBClient().Entity.Query().
			Where(entity.IDIn(ids...)).
			WithFile().
			WithUser().
			All(ctx)
	}

	if m.state.SrcPolicyID == 0 {
		return nil, nil
	}
	return dep.DBClient().Entity.Query().
		Where(
			entity.StoragePolicyEntities(m.state.SrcPolicyID),
			entity.IDGT(m.state.Cursor),
		).
		Order(ent.Asc(entity.FieldID)).
		Limit(relocatePageSize).
		WithFile().
		WithUser().
		All(ctx)
}

// progressTotals returns the expected entity count and byte total for the
// task's scope. Failures fall back to zero — progress is informational.
func (m *RelocateTask) progressTotals(ctx context.Context, dep dependency.Dep) (int64, int64) {
	var q *ent.EntityQuery
	if len(m.state.EntityIDs) > 0 {
		q = dep.DBClient().Entity.Query().Where(entity.IDIn(m.state.EntityIDs...))
	} else if m.state.SrcPolicyID != 0 {
		q = dep.DBClient().Entity.Query().Where(entity.StoragePolicyEntities(m.state.SrcPolicyID))
	} else {
		return 0, 0
	}

	count, err := q.Clone().Count(ctx)
	if err != nil {
		return 0, 0
	}
	var v []struct {
		Sum int64 `json:"sum"`
	}
	if err := q.Clone().Aggregate(ent.Sum(entity.FieldSize)).Scan(ctx, &v); err != nil || len(v) == 0 {
		return int64(count), 0
	}
	return int64(count), v[0].Sum
}

// relocateEntity streams one entity's blob to the destination policy, swaps its
// storage record, then deletes the old blob. A failure leaves the entity
// pointing at its original blob — the new object is dropped and recorded as a
// failure, so the entity remains readable.
func (m *RelocateTask) relocateEntity(ctx context.Context, dep dependency.Dep, fm manager.FileManager, dstPolicy *ent.StoragePolicy, e *ent.Entity) error {
	if e.StoragePolicyEntities == m.state.DstPolicyID {
		atomic.AddInt64(&m.progress[ProgressTypeRelocateCount].Current, 1)
		return nil
	}

	es, err := fm.GetEntitySource(ctx, e.ID)
	if err != nil {
		return fmt.Errorf("failed to open entity source: %w", err)
	}
	defer es.Close()

	// Generate the destination blob path from the policy's naming rules.
	savePath := relocateSavePath(dstPolicy, e)

	// The source stream is already decrypted; wrap it again when the
	// destination policy requires at-rest encryption. AES-CTR ciphertext
	// matches the plaintext size, so Props.Size stays e.Size.
	var encryptMetadata *types.EncryptMetadata
	var reader io.ReadCloser = es
	var seeker io.Seeker = es
	if dstPolicy.Settings != nil && dstPolicy.Settings.Encryption {
		cryptor, err := dep.EncryptorFactory(ctx)(types.CipherAES256CTR)
		if err != nil {
			return fmt.Errorf("failed to create cryptor: %w", err)
		}

		encryptMetadata, err = cryptor.GenerateMetadata(ctx)
		if err != nil {
			return fmt.Errorf("failed to generate encrypt metadata: %w", err)
		}
		if err := cryptor.LoadMetadata(ctx, encryptMetadata); err != nil {
			return fmt.Errorf("failed to load encrypt metadata: %w", err)
		}
		if err := cryptor.SetSource(es, es, e.Size, 0); err != nil {
			return fmt.Errorf("failed to set cryptor source: %w", err)
		}
		reader = cryptor
		seeker = cryptor
	}

	req := &fs.UploadRequest{
		Props: &fs.UploadProps{
			Uri:      relocateEntityUri(dep, e),
			Size:     e.Size,
			SavePath: savePath,
		},
		File:   reader,
		Seeker: seeker,
	}

	dstDriver, err := fm.GetStorageDriver(ctx, fm.CastStoragePolicyOnSlave(ctx, dstPolicy))
	if err != nil {
		return fmt.Errorf("failed to get destination driver: %w", err)
	}
	if err := dstDriver.Put(ctx, req); err != nil {
		return fmt.Errorf("failed to write to destination policy: %w", err)
	}

	// Swap the storage record; keep a copy of the old source path for cleanup.
	oldSource := e.Source
	oldPolicyID := e.StoragePolicyEntities

	newProps := types.EntityProps{}
	if e.Props != nil {
		newProps = *e.Props
	}
	newProps.EncryptMetadata = encryptMetadata
	e.Props = &newProps

	if err := dep.FileClient().RelocateEntity(ctx, e, savePath, m.state.DstPolicyID); err != nil {
		// The entity still points at its original blob — remove the orphaned
		// copy on the destination so the entity stays consistent.
		if _, derr := dstDriver.Delete(ctx, savePath); derr != nil {
			m.l.Warning("Failed to clean up orphaned blob %q of entity %d: %s", savePath, e.ID, derr)
		}
		return fmt.Errorf("failed to update entity record: %w", err)
	}

	// Remove the old blob; an orphan left behind is picked up by blob audit.
	srcPolicy, err := dep.StoragePolicyClient().GetPolicyByID(ctx, oldPolicyID)
	if err == nil && srcPolicy != nil {
		if srcDriver, err := fm.GetStorageDriver(ctx, fm.CastStoragePolicyOnSlave(ctx, srcPolicy)); err == nil {
			if failed, err := srcDriver.Delete(ctx, oldSource); err != nil || len(failed) > 0 {
				m.l.Warning("Failed to delete old blob %q of entity %d: %s", oldSource, e.ID, err)
			}
		}
	}

	atomic.AddInt64(&m.progress[ProgressTypeRelocateCount].Current, 1)
	atomic.AddInt64(&m.progress[ProgressTypeRelocateSize].Current, e.Size)
	return nil
}

// relocateSavePath replicates the upload save-path rules (dir rule + name rule)
// for an entity being relocated.
func relocateSavePath(policy *ent.StoragePolicy, e *ent.Entity) string {
	currentTime := time.Now()
	originName := entityDisplayName(e)
	dynamicReplace := func(rule string, pathAvailable bool) string {
		return fs.ReplaceMagicVar(rule, fs.MagicVarProps{
			FsSeparator:   fs.Separator,
			PathAvailable: pathAvailable,
			Time:          currentTime,
			UserID:        entityOwnerID(e),
			OriginName:    originName,
		})
	}

	dirRule := dynamicReplace(policy.DirNameRule, true)
	nameRule := dynamicReplace(policy.FileNameRule, false)

	return path.Join(path.Clean(dirRule), nameRule)
}

// relocateEntityUri builds a best-effort owner-scoped URI for the relocated
// blob's upload session record. The true file path is not reconstructed — the
// URI is informational on this path and never navigated.
func relocateEntityUri(dep dependency.Dep, e *ent.Entity) *fs.URI {
	owner := hashid.EncodeUserID(dep.HashIDEncoder(), entityOwnerID(e))
	uri, err := fs.NewUriFromString(fs.NewMyUri(owner))
	if err != nil {
		return nil
	}
	return uri.Join(entityDisplayName(e))
}

// entityDisplayName returns the display name of a file referencing the entity
// when the file edge is loaded, falling back to the blob's base name.
func entityDisplayName(e *ent.Entity) string {
	if len(e.Edges.File) > 0 && e.Edges.File[0].Name != "" {
		return e.Edges.File[0].Name
	}
	return path.Base(e.Source)
}

func entityOwnerID(e *ent.Entity) int {
	if e.Edges.User != nil {
		return e.Edges.User.ID
	}
	return e.CreatedBy
}

func (m *RelocateTask) Summarize(hasher hashid.Encoder) *queue.Summary {
	if m.state == nil {
		if err := json.Unmarshal([]byte(m.State()), &m.state); err != nil {
			return nil
		}
	}

	props := map[string]any{
		SummaryKeyDstPolicy: m.state.DstPolicyID,
	}
	if m.state.SrcPolicyID != 0 {
		props[SummaryKeySrcPolicy] = m.state.SrcPolicyID
	}
	if len(m.state.EntityIDs) > 0 {
		props[SummaryKeyEntities] = len(m.state.EntityIDs)
	}
	if len(m.state.Failed) > 0 {
		props[SummaryKeyRelocateFailed] = len(m.state.Failed)
	}

	return &queue.Summary{
		NodeID: m.state.NodeID,
		Phase:  string(m.state.Phase),
		Props:  props,
	}
}

func (m *RelocateTask) Progress(ctx context.Context) queue.Progresses {
	m.Lock()
	defer m.Unlock()
	return m.progress
}
