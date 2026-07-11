package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/mediaprocesstask"
	"github.com/cloudreve/Cloudreve/v4/ent/task"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
)

type (
	MediaBackfillTask struct {
		*queue.DBTask

		l        logging.Logger
		state    *MediaBackfillTaskState
		progress queue.Progresses
	}
	MediaBackfillTaskPhase string
	MediaBackfillTaskState struct {
		Phase                 MediaBackfillTaskPhase `json:"phase"`
		UserID                int                    `json:"user_id"`
		FilteredStoragePolicy []int                  `json:"filtered_storage_policy"`
		Total                 int                    `json:"total"`
		Scanned               int                    `json:"scanned"`
		Seeded                int                    `json:"seeded"`
		LastFileID            int                    `json:"last_file_id"`
	}
)

const (
	MediaBackfillPhaseCount MediaBackfillTaskPhase = "count"
	MediaBackfillPhaseSeed  MediaBackfillTaskPhase = "seed"

	MediaBackfillBatchSize = 1000

	ProgressTypeMediaBackfill = "media_backfill"
	SummaryKeySeeded          = "seeded"
)

func init() {
	queue.RegisterResumableTaskFactory(queue.MediaBackfillTaskType, NewMediaBackfillTaskFromModel)
}

func NewMediaBackfillTask(ctx context.Context, u *ent.User, filteredStoragePolicy []int) (queue.Task, error) {
	state := &MediaBackfillTaskState{
		Phase:                 MediaBackfillPhaseCount,
		UserID:                u.ID,
		FilteredStoragePolicy: filteredStoragePolicy,
	}
	stateBytes, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state: %w", err)
	}

	return &MediaBackfillTask{
		DBTask: &queue.DBTask{
			Task: &ent.Task{
				Type:          queue.MediaBackfillTaskType,
				CorrelationID: logging.CorrelationID(ctx),
				PrivateState:  string(stateBytes),
				PublicState:   &types.TaskPublicState{},
			},
			DirectOwner: u,
		},
	}, nil
}

func NewMediaBackfillTaskFromModel(t *ent.Task) queue.Task {
	return &MediaBackfillTask{
		DBTask: &queue.DBTask{
			Task: t,
		},
	}
}

func (m *MediaBackfillTask) Do(ctx context.Context) (task.Status, error) {
	dep := dependency.FromContext(ctx)
	m.l = dep.Logger()

	m.Lock()
	if m.progress == nil {
		m.progress = make(queue.Progresses)
	}
	m.progress[ProgressTypeMediaBackfill] = &queue.Progress{}
	m.Unlock()

	state := &MediaBackfillTaskState{}
	if err := json.Unmarshal([]byte(m.State()), state); err != nil {
		return task.StatusError, fmt.Errorf("failed to unmarshal state: %s (%w)", err, queue.CriticalErr)
	}
	m.state = state

	var (
		next = task.StatusCompleted
		err  error
	)
	switch m.state.Phase {
	case MediaBackfillPhaseCount, "":
		next, err = m.count(ctx, dep)
	case MediaBackfillPhaseSeed:
		next, err = m.seed(ctx, dep)
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

// count computes the number of candidate files to scan, then moves to seeding.
func (m *MediaBackfillTask) count(ctx context.Context, dep dependency.Dep) (task.Status, error) {
	minSize := dep.SettingProvider().MediaProcess(ctx).MinSize
	total, err := dep.FileClient().CountBackfillCandidateFiles(ctx, m.state.UserID, m.state.FilteredStoragePolicy, minSize)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to count backfill candidate files: %w", err)
	}

	m.state.Total = total
	m.state.Phase = MediaBackfillPhaseSeed
	m.state.LastFileID = 0
	m.state.Scanned = 0
	m.state.Seeded = 0

	m.l.Info("Media backfill: %d candidate file(s) to scan for user %d.", total, m.state.UserID)
	m.ResumeAfter(0)
	return task.StatusSuspending, nil
}

// seed scans a batch of candidate files and enqueues each image as a pending
// media_process_task row, then suspends for the next batch.
func (m *MediaBackfillTask) seed(ctx context.Context, dep dependency.Dep) (task.Status, error) {
	atomic.StoreInt64(&m.progress[ProgressTypeMediaBackfill].Total, int64(m.state.Total))
	atomic.StoreInt64(&m.progress[ProgressTypeMediaBackfill].Current, int64(m.state.Scanned))

	minSize := dep.SettingProvider().MediaProcess(ctx).MinSize
	files, err := dep.FileClient().ListBackfillCandidateFiles(ctx, m.state.LastFileID, MediaBackfillBatchSize, m.state.UserID, m.state.FilteredStoragePolicy, minSize)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to list backfill candidate files after ID %d: %w", m.state.LastFileID, err)
	}

	if len(files) == 0 {
		m.l.Info("Media backfill complete for user %d. %d image(s) enqueued.", m.state.UserID, m.state.Seeded)
		return task.StatusCompleted, nil
	}

	mpClient := dep.MediaProcessClient()
	detector := dep.MimeDetector(ctx)
	for _, f := range files {
		mimeType := detector.TypeByName(f.Name)
		if !strings.HasPrefix(strings.ToLower(mimeType), "image/") {
			continue
		}
		// Skip files already handled by a previous backfill/upload pass, so a
		// re-run does not re-compress the compressed output.
		if handled, err := mpClient.HasHandledForFile(ctx, f.ID); err != nil {
			m.l.Warning("Media backfill: handled-check failed for file %d: %s", f.ID, err)
			continue
		} else if handled {
			continue
		}
		if _, err := mpClient.Enqueue(ctx, &inventory.MediaProcessEnqueueArgs{
			EntityID:  f.PrimaryEntity,
			FileID:    f.ID,
			OwnerID:   f.OwnerID,
			MediaType: mediaprocesstask.MediaTypeImage,
		}); err != nil {
			m.l.Warning("Media backfill: failed to enqueue file %d: %s", f.ID, err)
			continue
		}
		m.state.Seeded++
	}

	m.state.Scanned += len(files)
	m.state.LastFileID = files[len(files)-1].ID
	atomic.StoreInt64(&m.progress[ProgressTypeMediaBackfill].Current, int64(m.state.Scanned))

	m.ResumeAfter(0)
	return task.StatusSuspending, nil
}

func (m *MediaBackfillTask) Progress(ctx context.Context) queue.Progresses {
	m.Lock()
	defer m.Unlock()
	return m.progress
}

func (m *MediaBackfillTask) Summarize(hasher hashid.Encoder) *queue.Summary {
	if m.state == nil {
		if err := json.Unmarshal([]byte(m.State()), &m.state); err != nil {
			return nil
		}
	}

	return &queue.Summary{
		Phase: string(m.state.Phase),
		Props: map[string]any{
			SummaryKeyTotal:  m.state.Total,
			SummaryKeySeeded: m.state.Seeded,
		},
	}
}
