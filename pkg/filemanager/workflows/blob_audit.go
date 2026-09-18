package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"sync/atomic"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
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
	BlobAuditTask struct {
		*queue.DBTask
		state    *BlobAuditTaskState
		l        logging.Logger
		progress queue.Progresses
	}
	BlobAuditTaskPhase string
	BlobAuditTaskState struct {
		Phase       BlobAuditTaskPhase `json:"phase"`
		PolicyID    int                `json:"policy_id"`
		Delete      bool               `json:"delete"`
		Scanned     int                `json:"scanned"`
		Orphans     []string           `json:"orphans"`
		OrphanCount int                `json:"orphan_count"`
		Deleted     int                `json:"deleted"`
		Failed      int                `json:"failed"`
		Truncated   bool               `json:"truncated"`
	}
)

const (
	BlobAuditPhaseScan   BlobAuditTaskPhase = "scan"
	BlobAuditPhaseDelete BlobAuditTaskPhase = "delete"

	BlobAuditDeleteBatch   = 200
	BlobAuditMaxOrphanList = 5000

	ProgressTypeBlobAudit   = "blob_audit"
	SummaryKeyOrphanCount   = "orphan_count"
	SummaryKeyDeleted       = "deleted"
	SummaryKeyScannedObject = "scanned"
)

func init() {
	queue.RegisterResumableTaskFactory(queue.BlobAuditTaskType, NewBlobAuditTaskFromModel)
}

// NewBlobAuditTask creates a task that lists all physical objects under a
// storage policy and diffs them against the entities table, reporting (and
// optionally deleting) blobs no entity references (upstream #3395).
func NewBlobAuditTask(ctx context.Context, u *ent.User, policyID int, delete bool) (queue.Task, error) {
	state := &BlobAuditTaskState{
		Phase:    BlobAuditPhaseScan,
		PolicyID: policyID,
		Delete:   delete,
	}
	stateBytes, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state: %w", err)
	}

	return &BlobAuditTask{
		DBTask: &queue.DBTask{
			Task: &ent.Task{
				Type:          queue.BlobAuditTaskType,
				CorrelationID: logging.CorrelationID(ctx),
				PrivateState:  string(stateBytes),
				PublicState:   &types.TaskPublicState{},
			},
			DirectOwner: u,
		},
	}, nil
}

func NewBlobAuditTaskFromModel(t *ent.Task) queue.Task {
	return &BlobAuditTask{
		DBTask: &queue.DBTask{
			Task: t,
		},
	}
}

func (m *BlobAuditTask) Do(ctx context.Context) (task.Status, error) {
	dep := dependency.FromContext(ctx)
	m.l = dep.Logger()

	m.Lock()
	if m.progress == nil {
		m.progress = make(queue.Progresses)
	}
	m.progress[ProgressTypeBlobAudit] = &queue.Progress{}
	m.Unlock()

	state := &BlobAuditTaskState{}
	if err := json.Unmarshal([]byte(m.State()), state); err != nil {
		return task.StatusError, fmt.Errorf("failed to unmarshal state: %s (%w)", err, queue.CriticalErr)
	}
	m.state = state

	var (
		next = task.StatusCompleted
		err  error
	)
	switch m.state.Phase {
	case BlobAuditPhaseScan, "":
		next, err = m.scan(ctx, dep)
	case BlobAuditPhaseDelete:
		next, err = m.deleteOrphans(ctx, dep)
	default:
		next, err = task.StatusError, fmt.Errorf("unknown phase %q: %w", m.state.Phase, queue.CriticalErr)
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

// scan lists all physical objects under the policy's blob prefix and diffs
// them against the entity sources recorded for that policy.
func (m *BlobAuditTask) scan(ctx context.Context, dep dependency.Dep) (task.Status, error) {
	policy, err := dep.StoragePolicyClient().GetPolicyByID(ctx, m.state.PolicyID)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to get storage policy %d: %w", m.state.PolicyID, err)
	}
	if policy.Status == storagepolicy.StatusSuspended {
		return task.StatusError, fmt.Errorf("storage policy %d is suspended: %w", m.state.PolicyID, queue.CriticalErr)
	}

	fm := manager.NewFileManager(dep, nil)
	defer fm.Recycle()
	d, err := fm.GetStorageDriver(ctx, policy)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to get storage driver: %w", err)
	}

	// Literal prefix of the dir name rule (e.g. "uploads/{uid}/{path}" ->
	// "uploads") scopes the listing so non-blob areas (temp, avatars) are
	// never touched.
	base := literalDirPrefix(policy.DirNameRule)
	m.l.Info("Blob audit on policy %d: listing objects under %q", policy.ID, base)

	var objects []fs.PhysicalObject
	objects, err = d.List(ctx, base, func(i int) {
		atomic.AddInt64(&m.progress[ProgressTypeBlobAudit].Current, int64(i))
	}, true)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to list objects on policy %d: %w", m.state.PolicyID, err)
	}
	m.state.Scanned = len(objects)

	// Full entity source set for this policy, paginated.
	known := make(map[string]struct{})
	page := 0
	for {
		res, err := dep.FileClient().ListEntities(ctx, &inventory.ListEntityParameters{
			PaginationArgs:  &inventory.PaginationArgs{Page: page, PageSize: 1000},
			StoragePolicyID: policy.ID,
		})
		if err != nil {
			return task.StatusError, fmt.Errorf("failed to list entities: %w", err)
		}
		for _, e := range res.Entities {
			known[e.Source] = struct{}{}
		}
		if len(res.Entities) < 1000 {
			break
		}
		page++
	}

	orphanSet := make(map[string]struct{})
	for _, obj := range objects {
		if obj.IsDir {
			continue
		}
		key := obj.RelativePath
		if base != "" {
			key = path.Join(base, obj.RelativePath)
		}
		if _, ok := known[key]; ok {
			continue
		}
		if _, dup := orphanSet[key]; dup {
			continue
		}
		orphanSet[key] = struct{}{}
		if len(m.state.Orphans) < BlobAuditMaxOrphanList {
			m.state.Orphans = append(m.state.Orphans, key)
		} else {
			m.state.Truncated = true
		}
	}

	m.l.Info("Blob audit on policy %d: %d objects scanned, %d orphans found.", policy.ID, m.state.Scanned, len(orphanSet))
	m.state.OrphanCount = len(orphanSet)

	if m.state.Delete && len(m.state.Orphans) > 0 {
		m.state.Phase = BlobAuditPhaseDelete
		m.ResumeAfter(0)
		return task.StatusSuspending, nil
	}
	return task.StatusCompleted, nil
}

// deleteOrphans removes the recorded orphan blobs in batches.
func (m *BlobAuditTask) deleteOrphans(ctx context.Context, dep dependency.Dep) (task.Status, error) {
	if len(m.state.Orphans) == 0 {
		return task.StatusCompleted, nil
	}

	policy, err := dep.StoragePolicyClient().GetPolicyByID(ctx, m.state.PolicyID)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to get storage policy %d: %w", m.state.PolicyID, err)
	}

	fm := manager.NewFileManager(dep, nil)
	defer fm.Recycle()
	d, err := fm.GetStorageDriver(ctx, policy)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to get storage driver: %w", err)
	}

	batch := m.state.Orphans
	if len(batch) > BlobAuditDeleteBatch {
		batch = batch[:BlobAuditDeleteBatch]
	}

	failed, err := d.Delete(ctx, batch...)
	if err != nil {
		m.l.Warning("Blob audit delete partially failed on policy %d: %s", policy.ID, err)
	}
	m.state.Deleted += len(batch) - len(failed)
	m.state.Failed += len(failed)
	// Failed paths are dropped from the list — a re-run of the audit flags
	// them again if they are still orphaned.
	m.state.Orphans = m.state.Orphans[len(batch):]

	atomic.StoreInt64(&m.progress[ProgressTypeBlobAudit].Current, int64(m.state.Deleted))

	if len(m.state.Orphans) > 0 {
		m.ResumeAfter(0)
		return task.StatusSuspending, nil
	}
	return task.StatusCompleted, nil
}

// literalDirPrefix returns the static path prefix before the first magic
// variable, e.g. "uploads/{uid}/{path}" -> "uploads". An empty result means
// the rule starts with a variable and the policy root cannot be narrowed.
func literalDirPrefix(rule string) string {
	if i := strings.Index(rule, "{"); i >= 0 {
		rule = rule[:i]
	}
	return strings.Trim(rule, "/")
}

func (m *BlobAuditTask) Progress(ctx context.Context) queue.Progresses {
	m.Lock()
	defer m.Unlock()
	return m.progress
}

func (m *BlobAuditTask) Summarize(hasher hashid.Encoder) *queue.Summary {
	if m.state == nil {
		if err := json.Unmarshal([]byte(m.State()), &m.state); err != nil {
			return nil
		}
	}

	return &queue.Summary{
		Phase: string(m.state.Phase),
		Props: map[string]any{
			SummaryKeyOrphanCount:   m.state.OrphanCount,
			SummaryKeyScannedObject: m.state.Scanned,
			SummaryKeyDeleted:       m.state.Deleted,
			SummaryKeyFailed:        m.state.Failed,
			"truncated":             m.state.Truncated,
			"policy_id":             m.state.PolicyID,
		},
	}
}
