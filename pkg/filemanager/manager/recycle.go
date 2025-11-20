package manager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/task"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/crontab"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/samber/lo"
)

type (
	ExplicitEntityRecycleTask struct {
		*queue.DBTask
	}

	ExplicitEntityRecycleTaskState struct {
		EntityIDs []int            `json:"entity_ids,omitempty"`
		Errors    [][]RecycleError `json:"errors,omitempty"`
	}

	RecycleError struct {
		ID    string `json:"id"`
		Error string `json:"error"`
	}
)

func init() {
	queue.RegisterResumableTaskFactory(queue.ExplicitEntityRecycleTaskType, NewExplicitEntityRecycleTaskFromModel)
	queue.RegisterResumableTaskFactory(queue.EntityRecycleRoutineTaskType, NewEntityRecycleRoutineTaskFromModel)
	crontab.Register(setting.CronTypeEntityCollect, func(ctx context.Context) {
		dep := dependency.FromContext(ctx)
		l := dep.Logger()
		t, err := NewEntityRecycleRoutineTask(ctx)
		if err != nil {
			l.Error("Failed to create entity recycle routine task: %s", err)
		}

		if err := dep.EntityRecycleQueue(ctx).QueueTask(ctx, t); err != nil {
			l.Error("Failed to queue entity recycle routine task: %s", err)
		}
	})
	crontab.Register(setting.CronTypeTrashBinCollect, CronCollectTrashBin)
}

func NewExplicitEntityRecycleTaskFromModel(task *ent.Task) queue.Task {
	return &ExplicitEntityRecycleTask{
		DBTask: &queue.DBTask{
			Task: task,
		},
	}
}

func newExplicitEntityRecycleTask(ctx context.Context, entities []int) (*ExplicitEntityRecycleTask, error) {
	state := &ExplicitEntityRecycleTaskState{
		EntityIDs: entities,
		Errors:    make([][]RecycleError, 0),
	}
	stateBytes, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state: %w", err)
	}

	t := &ExplicitEntityRecycleTask{
		DBTask: &queue.DBTask{
			Task: &ent.Task{
				Type:          queue.ExplicitEntityRecycleTaskType,
				CorrelationID: logging.CorrelationID(ctx),
				PrivateState:  string(stateBytes),
				PublicState: &types.TaskPublicState{
					ResumeTime: time.Now().Unix() - 1,
				},
			},
			DirectOwner: inventory.UserFromContext(ctx),
		},
	}
	return t, nil
}

func (m *ExplicitEntityRecycleTask) Do(ctx context.Context) (task.Status, error) {
	dep := dependency.FromContext(ctx)
	fm := NewFileManager(dep, inventory.UserFromContext(ctx)).(*manager)

	// unmarshal state
	state := &ExplicitEntityRecycleTaskState{}
	if err := json.Unmarshal([]byte(m.State()), state); err != nil {
		return task.StatusError, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// recycle entities
	err := fm.RecycleEntities(ctx, false, state.EntityIDs...)
	if err != nil {
		appendAe(&state.Errors, err)
		privateState, err := json.Marshal(state)
		if err != nil {
			return task.StatusError, fmt.Errorf("failed to marshal state: %w", err)
		}
		m.Task.PrivateState = string(privateState)
		return task.StatusError, err
	}

	return task.StatusCompleted, nil
}

type (
	EntityRecycleRoutineTask struct {
		*queue.DBTask
	}

	EntityRecycleRoutineTaskState struct {
		Errors [][]RecycleError `json:"errors,omitempty"`
	}
)

func NewEntityRecycleRoutineTaskFromModel(task *ent.Task) queue.Task {
	return &EntityRecycleRoutineTask{
		DBTask: &queue.DBTask{
			Task: task,
		},
	}
}

func NewEntityRecycleRoutineTask(ctx context.Context) (queue.Task, error) {
	state := &EntityRecycleRoutineTaskState{
		Errors: make([][]RecycleError, 0),
	}
	stateBytes, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state: %w", err)
	}

	t := &EntityRecycleRoutineTask{
		DBTask: &queue.DBTask{
			Task: &ent.Task{
				Type:          queue.EntityRecycleRoutineTaskType,
				CorrelationID: logging.CorrelationID(ctx),
				PrivateState:  string(stateBytes),
				PublicState: &types.TaskPublicState{
					ResumeTime: time.Now().Unix() - 1,
				},
			},
			DirectOwner: inventory.UserFromContext(ctx),
		},
	}
	return t, nil
}

func (m *EntityRecycleRoutineTask) Do(ctx context.Context) (task.Status, error) {
	dep := dependency.FromContext(ctx)
	fm := NewFileManager(dep, inventory.UserFromContext(ctx)).(*manager)

	// unmarshal state
	state := &EntityRecycleRoutineTaskState{}
	if err := json.Unmarshal([]byte(m.State()), state); err != nil {
		return task.StatusError, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// recycle entities
	err := fm.RecycleEntities(ctx, false)
	if err != nil {
		appendAe(&state.Errors, err)

		privateState, err := json.Marshal(state)
		if err != nil {
			return task.StatusError, fmt.Errorf("failed to marshal state: %w", err)
		}
		m.Task.PrivateState = string(privateState)
		return task.StatusError, err
	}

	return task.StatusCompleted, nil
}

// RecycleEntities delete given entities. If the ID list is empty, it will walk through
// all stale entities in DB.
func (m *manager) RecycleEntities(ctx context.Context, force bool, entityIDs ...int) error {
	ae := serializer.NewAggregateError()
	entities, err := m.fs.StaleEntities(ctx, entityIDs...)
	if err != nil {
		return fmt.Errorf("failed to get entities: %w", err)
	}

	// Group entities by policy ID
	entityGroup := lo.GroupBy(entities, func(entity fs.Entity) int {
		return entity.PolicyID()
	})

	// Delete entity in each group in batch
	for _, entities := range entityGroup {
		entityChunk := lo.Chunk(entities, 100)
		m.l.Info("Recycling %d entities in %d batches", len(entities), len(entityChunk))

		for batch, chunk := range entityChunk {
			m.l.Info("Start to recycle batch #%d, %d entities", batch, len(chunk))
			mapSrcToId := make(map[string]int, len(chunk))
			_, d, err := m.getEntityPolicyDriver(ctx, chunk[0], nil)
			if err != nil {
				for _, entity := range chunk {
					ae.Add(strconv.Itoa(entity.ID()), err)
				}
				continue
			}

			for _, entity := range chunk {
				mapSrcToId[entity.Source()] = entity.ID()
			}

			toBeDeletedSrc := lo.Map(lo.Filter(chunk, func(item fs.Entity, index int) bool {
				// Only delete entities that are not marked as "unlink only"
				return item.Model().Props == nil || !item.Model().Props.UnlinkOnly
			}), func(entity fs.Entity, index int) string {
				return entity.Source()
			})
			if len(toBeDeletedSrc) > 0 {
				res, err := d.Delete(ctx, toBeDeletedSrc...)
				if err != nil {
					for _, src := range res {
						ae.Add(strconv.Itoa(mapSrcToId[src]), err)
					}
				}
			}

			// Delete upload session if it's still valid
			for _, entity := range chunk {
				sid := entity.UploadSessionID()
				if sid == nil {
					continue
				}

				if session, ok := m.kv.Get(UploadSessionCachePrefix + sid.String()); ok {
					session := session.(fs.UploadSession)
					if err := d.CancelToken(ctx, &session); err != nil {
						m.l.Warning("Failed to cancel upload session for %q: %s, this is expected if it's remote policy.", session.Props.Uri.String(), err)
					}
					_ = m.kv.Delete(UploadSessionCachePrefix, sid.String())
				}
			}

			// Filtering out entities that are successfully deleted
			rawAe := ae.Raw()
			successEntities := lo.FilterMap(chunk, func(entity fs.Entity, index int) (int, bool) {
				entityIdStr := fmt.Sprintf("%d", entity.ID())
				_, ok := rawAe[entityIdStr]
				if !ok {
					// No error, deleted
					return entity.ID(), true
				}

				if force {
					ae.Remove(entityIdStr)
				}
				return entity.ID(), force
			})

			// Remove entities from DB
			fc, tx, ctx, err := inventory.WithTx(ctx, m.dep.FileClient())
			if err != nil {
				return fmt.Errorf("failed to start transaction: %w", err)
			}
			storageReduced, err := fc.RemoveEntitiesByID(ctx, successEntities...)
			if err != nil {
				_ = inventory.Rollback(tx)
				return fmt.Errorf("failed to remove entities from DB: %w", err)
			}

			tx.AppendStorageDiff(storageReduced)
			if err := inventory.CommitWithStorageDiff(ctx, tx, m.l, m.dep.UserClient()); err != nil {
				return fmt.Errorf("failed to commit delete change: %w", err)
			}

		}
	}

	return ae.Aggregate()
}

const (
	MinimumTrashCollectBatch = 1000
)

// CronCollectTrashBin walks through all files in trash bin and delete them if they are expired.
func CronCollectTrashBin(ctx context.Context) {
	dep := dependency.FromContext(ctx)
	l := dep.Logger()

	kv := dep.KV()
	if memKv, ok := kv.(*cache.MemoStore); ok {
		memKv.GarbageCollect(l)
	}

	fm := NewFileManager(dep, inventory.UserFromContext(ctx)).(*manager)
	pageSize := dep.SettingProvider().DBFS(ctx).MaxPageSize
	batch := 0
	expiredFiles := make([]fs.File, 0)
	for {
		res, err := fm.fs.AllFilesInTrashBin(ctx, fs.WithPageSize(pageSize))
		if err != nil {
			l.Error("Failed to get files in trash bin: %s", err)
			return
		}

		expired := lo.Filter(res.Files, func(file fs.File, index int) bool {
			if expire, ok := file.Metadata()[dbfs.MetadataExpectedCollectTime]; ok {
				expireUnix, err := strconv.ParseInt(expire, 10, 64)
				if err != nil {
					l.Warning("Failed to parse expected collect time %q: %s, will treat as expired", expire, err)
				}

				if expireUnix < time.Now().Unix() {
					return true
				}
			}

			return false
		})
		l.Info("Found %d files in trash bin pending collect, in batch #%d", len(res.Files), batch)

		expiredFiles = append(expiredFiles, expired...)
		if len(expiredFiles) >= MinimumTrashCollectBatch {
			collectTrashBin(ctx, expiredFiles, dep, l)
			expiredFiles = expiredFiles[:0]
		}

		if res.Pagination.NextPageToken == "" {
			if len(expiredFiles) > 0 {
				collectTrashBin(ctx, expiredFiles, dep, l)
			}
			break
		}

		batch++
	}
}

func collectTrashBin(ctx context.Context, files []fs.File, dep dependency.Dep, l logging.Logger) {
	l.Info("Start to collect %d files in trash bin", len(files))
	uc := dep.UserClient()

	// Group files by Owners
	fileGroup := lo.GroupBy(files, func(file fs.File) int {
		return file.OwnerID()
	})

	for uid, expiredFiles := range fileGroup {
		ctx = context.WithValue(ctx, inventory.LoadUserGroup{}, true)
		user, err := uc.GetByID(ctx, uid)
		if err != nil {
			l.Error("Failed to get user %d: %s", uid, err)
			continue
		}

		ctx = context.WithValue(ctx, inventory.UserCtx{}, user)
		fm := NewFileManager(dep, user).(*manager)
		if err := fm.Delete(ctx, lo.Map(expiredFiles, func(file fs.File, index int) *fs.URI {
			return file.Uri(false)
		}), fs.WithSkipSoftDelete(true)); err != nil {
			l.Error("Failed to delete files for user %d: %s", uid, err)
		}
	}
}

func appendAe(errs *[][]RecycleError, err error) {
	var ae *serializer.AggregateError
	*errs = append(*errs, make([]RecycleError, 0))
	if errors.As(err, &ae) {
		(*errs)[len(*errs)-1] = lo.MapToSlice(ae.Raw(), func(key string, value error) RecycleError {
			return RecycleError{
				ID:    key,
				Error: value.Error(),
			}
		})
	}
}
