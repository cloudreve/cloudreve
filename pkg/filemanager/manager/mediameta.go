package manager

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/task"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/mediameta"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/samber/lo"
)

type (
	MediaMetaTask struct {
		*queue.DBTask
	}

	MediaMetaTaskState struct {
		Uri      *fs.URI `json:"uri"`
		EntityID int     `json:"entity_id"`
	}
)

func init() {
	queue.RegisterResumableTaskFactory(queue.MediaMetaTaskType, NewMediaMetaTaskFromModel)
}

// NewMediaMetaTask creates a new MediaMetaTask to
func NewMediaMetaTask(ctx context.Context, uri *fs.URI, entityID int, creator *ent.User) (*MediaMetaTask, error) {
	state := &MediaMetaTaskState{
		Uri:      uri,
		EntityID: entityID,
	}
	stateBytes, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state: %w", err)
	}

	return &MediaMetaTask{
		DBTask: &queue.DBTask{
			DirectOwner: creator,
			Task: &ent.Task{
				Type:          queue.MediaMetaTaskType,
				CorrelationID: logging.CorrelationID(ctx),
				PrivateState:  string(stateBytes),
				PublicState:   &types.TaskPublicState{},
			},
		},
	}, nil
}

func NewMediaMetaTaskFromModel(task *ent.Task) queue.Task {
	return &MediaMetaTask{
		DBTask: &queue.DBTask{
			Task: task,
		},
	}
}

func (m *MediaMetaTask) Do(ctx context.Context) (task.Status, error) {
	dep := dependency.FromContext(ctx)
	fm := NewFileManager(dep, inventory.UserFromContext(ctx)).(*manager)

	// unmarshal state
	var state MediaMetaTaskState
	if err := json.Unmarshal([]byte(m.State()), &state); err != nil {
		return task.StatusError, fmt.Errorf("failed to unmarshal state: %s (%w)", err, queue.CriticalErr)
	}

	err := fm.ExtractAndSaveMediaMeta(ctx, state.Uri, state.EntityID)
	if err != nil {
		return task.StatusError, err
	}

	return task.StatusCompleted, nil
}

func (m *manager) ExtractAndSaveMediaMeta(ctx context.Context, uri *fs.URI, entityID int) error {
	// 1. retrieve file info
	file, err := m.fs.Get(ctx, uri, dbfs.WithFileEntities())
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	versions := lo.Filter(file.Entities(), func(i fs.Entity, index int) bool {
		return i.Type() == types.EntityTypeVersion
	})
	targetVersion, versionIndex, found := lo.FindIndexOf(versions, func(i fs.Entity) bool {
		return i.ID() == entityID
	})
	if !found {
		return fmt.Errorf("failed to find version: %s (%w)", err, queue.CriticalErr)
	}

	if versionIndex != 0 {
		m.l.Debug("Skip media meta task for non-latest version.")
		return nil
	}

	language := ""
	if file.Owner().Settings != nil {
		language = file.Owner().Settings.Language
	}

	var (
		metas []driver.MediaMeta
	)
	// 2. try using native driver
	_, d, err := m.getEntityPolicyDriver(ctx, targetVersion, nil)
	if err != nil {
		return fmt.Errorf("failed to get storage driver: %s (%w)", err, queue.CriticalErr)
	}
	driverCaps := d.Capabilities()
	if util.IsInExtensionList(driverCaps.MediaMetaSupportedExts, file.Name()) {
		m.l.Debug("Using native driver to generate media meta.")
		metas, err = d.MediaMeta(ctx, targetVersion.Source(), file.Ext(), language)
		if err != nil {
			return fmt.Errorf("failed to get media meta using native driver: %w", err)
		}
	} else if driverCaps.MediaMetaProxy && util.IsInExtensionList(m.dep.MediaMetaExtractor(ctx).Exts(), file.Name()) {
		m.l.Debug("Using local extractor to generate media meta.")
		extractor := m.dep.MediaMetaExtractor(ctx)
		source, err := m.GetEntitySource(ctx, targetVersion.ID())
		defer source.Close()
		if err != nil {
			return fmt.Errorf("failed to get entity source: %w", err)
		}

		metas, err = extractor.Extract(ctx, file.Ext(), source, mediameta.WithLanguage(language))
		if err != nil {
			return fmt.Errorf("failed to extract media meta using local extractor: %w", err)
		}

	} else {
		m.l.Debug("No available generator for media meta.")
		return nil
	}

	m.l.Debug("%d media meta generated.", len(metas))
	m.l.Debug("Media meta: %v", metas)

	// 3. save meta
	if len(metas) > 0 {
		if err := m.fs.PatchMetadata(ctx, []*fs.URI{uri}, lo.Map(metas, func(i driver.MediaMeta, index int) fs.MetadataPatch {
			return fs.MetadataPatch{
				Key:   fmt.Sprintf("%s:%s", i.Type, i.Key),
				Value: i.Value,
			}
		})...); err != nil {
			return fmt.Errorf("failed to save media meta: %s (%w)", err, queue.CriticalErr)
		}
	}

	return nil
}

func (m *manager) shouldGenerateMediaMeta(ctx context.Context, d driver.Handler, fileName string) bool {
	driverCaps := d.Capabilities()
	if util.IsInExtensionList(driverCaps.MediaMetaSupportedExts, fileName) {
		// Handler support it natively
		return true
	}

	if driverCaps.MediaMetaProxy && util.IsInExtensionList(m.dep.MediaMetaExtractor(ctx).Exts(), fileName) {
		// Handler does not support. but proxy is enabled.
		return true
	}

	return false
}

func (m *manager) mediaMetaForNewEntity(ctx context.Context, session *fs.UploadSession, d driver.Handler) {
	if session.Props.EntityType == nil || *session.Props.EntityType == types.EntityTypeVersion {
		if !m.shouldGenerateMediaMeta(ctx, d, session.Props.Uri.Name()) {
			return
		}

		mediaMetaTask, err := NewMediaMetaTask(ctx, session.Props.Uri, session.EntityID, m.user)
		if err != nil {
			m.l.Warning("Failed to create media meta task: %s", err)
			return
		}
		if err := m.dep.MediaMetaQueue(ctx).QueueTask(ctx, mediaMetaTask); err != nil {
			m.l.Warning("Failed to queue media meta task: %s", err)
		}

		return
	}
}
