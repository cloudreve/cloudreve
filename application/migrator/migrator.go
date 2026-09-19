package migrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/application/migrator/conf"
	"github.com/cloudreve/Cloudreve/v4/application/migrator/model"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
)

// State stores the migration progress
type State struct {
	PolicyIDs          map[int]bool    `json:"policy_ids,omitempty"`
	LocalPolicyIDs     map[int]bool    `json:"local_policy_ids,omitempty"`
	UserIDs            map[int]bool    `json:"user_ids,omitempty"`
	FolderIDs          map[int]bool    `json:"folder_ids,omitempty"`
	EntitySources      map[string]int  `json:"entity_sources,omitempty"`
	LastFolderID       int             `json:"last_folder_id,omitempty"`
	Step               int             `json:"step,omitempty"`
	UserOffset         int             `json:"user_offset,omitempty"`
	FolderOffset       int             `json:"folder_offset,omitempty"`
	FileOffset         int             `json:"file_offset,omitempty"`
	ShareOffset        int             `json:"share_offset,omitempty"`
	GiftCodeOffset     int             `json:"gift_code_offset,omitempty"`
	DirectLinkOffset   int             `json:"direct_link_offset,omitempty"`
	WebdavOffset       int             `json:"webdav_offset,omitempty"`
	StoragePackOffset  int             `json:"storage_pack_offset,omitempty"`
	FileConflictRename map[uint]string `json:"file_conflict_rename,omitempty"`
	FolderParentOffset int             `json:"folder_parent_offset,omitempty"`
	ThumbSuffix        string          `json:"thumb_suffix,omitempty"`
	V3AvatarPath       string          `json:"v3_avatar_path,omitempty"`
}

// Step identifiers for migration phases
const (
	StepInitial                = 0
	StepSchema                 = 1
	StepSettings               = 2
	StepNode                   = 3
	StepPolicy                 = 4
	StepGroup                  = 5
	StepUser                   = 6
	StepFolders                = 7
	StepFolderParent           = 8
	StepFile                   = 9
	StepShare                  = 10
	StepDirectLink             = 11
	Step_CommunityPlaceholder1 = 12
	Step_CommunityPlaceholder2 = 13
	StepAvatar                 = 14
	StepWebdav                 = 15
	StepCompleted              = 16
	StateFileName              = "migration_state.json"
)

type Migrator struct {
	dep       dependency.Dep
	l         logging.Logger
	v4client  *ent.Client
	state     *State
	statePath string
}

func NewMigrator(dep dependency.Dep, v3ConfPath string) (*Migrator, error) {
	m := &Migrator{
		dep: dep,
		l:   dep.Logger(),
		state: &State{
			PolicyIDs:    make(map[int]bool),
			UserIDs:      make(map[int]bool),
			Step:         StepInitial,
			UserOffset:   0,
			FolderOffset: 0,
		},
	}

	// Determine state file path
	configDir := filepath.Dir(v3ConfPath)
	m.statePath = filepath.Join(configDir, StateFileName)

	// Try to load existing state
	if util.Exists(m.statePath) {
		m.l.Info("Found existing migration state file, loading from %s", m.statePath)
		if err := m.loadState(); err != nil {
			return nil, fmt.Errorf("failed to load migration state: %w", err)
		}

		stepName := "unknown"
		switch m.state.Step {
		case StepInitial:
			stepName = "initial"
		case StepSchema:
			stepName = "schema creation"
		case StepSettings:
			stepName = "settings migration"
		case StepNode:
			stepName = "node migration"
		case StepPolicy:
			stepName = "policy migration"
		case StepGroup:
			stepName = "group migration"
		case StepUser:
			stepName = "user migration"
		case StepFolders:
			stepName = "folders migration"
		case StepCompleted:
			stepName = "completed"
		case StepWebdav:
			stepName = "webdav migration"
		case StepAvatar:
			stepName = "avatar migration"

		}

		m.l.Info("Resumed migration from step %d (%s)", m.state.Step, stepName)

		// Log batch information if applicable
		if m.state.Step == StepUser && m.state.UserOffset > 0 {
			m.l.Info("Will resume user migration from batch offset %d", m.state.UserOffset)
		}
		if m.state.Step == StepFolders && m.state.FolderOffset > 0 {
			m.l.Info("Will resume folder migration from batch offset %d", m.state.FolderOffset)
		}
	}

	err := conf.Init(m.dep.Logger(), v3ConfPath)
	if err != nil {
		return nil, err
	}

	err = model.Init()
	if err != nil {
		return nil, err
	}

	v4client, err := inventory.NewRawEntClient(m.l, m.dep.ConfigProvider())
	if err != nil {
		return nil, err
	}

	m.v4client = v4client
	return m, nil
}

// saveState persists migration state to file
func (m *Migrator) saveState() error {
	data, err := json.Marshal(m.state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	return os.WriteFile(m.statePath, data, 0644)
}

// loadState reads migration state from file
func (m *Migrator) loadState() error {
	data, err := os.ReadFile(m.statePath)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	return json.Unmarshal(data, m.state)
}

// updateStep updates current step and persists state
func (m *Migrator) updateStep(step int) error {
	m.state.Step = step
	return m.saveState()
}

func (m *Migrator) Migrate() error {
	// Continue from the current step
	if m.state.Step <= StepSchema {
		m.l.Info("Creating basic v4 table schema...")
		inventory.RepairHeatWaveNotSecondary(m.l, m.v4client, context.Background())
		if err := m.v4client.Schema.Create(context.Background()); err != nil {
			return fmt.Errorf("failed creating schema resources: %w", err)
		}
		if err := m.updateStep(StepSettings); err != nil {
			return fmt.Errorf("failed to update step: %w", err)
		}
	}

	if m.state.Step <= StepSettings {
		if err := m.migrateSettings(); err != nil {
			return err
		}
		if err := m.updateStep(StepNode); err != nil {
			return fmt.Errorf("failed to update step: %w", err)
		}
	}

	if m.state.Step <= StepNode {
		if err := m.migrateNode(); err != nil {
			return err
		}
		if err := m.updateStep(StepPolicy); err != nil {
			return fmt.Errorf("failed to update step: %w", err)
		}
	}

	if m.state.Step <= StepPolicy {
		allPolicyIDs, err := m.migratePolicy()
		if err != nil {
			return err
		}
		m.state.PolicyIDs = allPolicyIDs
		if err := m.updateStep(StepGroup); err != nil {
			return fmt.Errorf("failed to update step: %w", err)
		}
	}

	if m.state.Step <= StepGroup {
		if err := m.migrateGroup(); err != nil {
			return err
		}
		if err := m.updateStep(StepUser); err != nil {
			return fmt.Errorf("failed to update step: %w", err)
		}
	}

	if m.state.Step <= StepUser {
		if err := m.migrateUser(); err != nil {
			m.saveState()
			return err
		}
		// Reset user offset after completion
		m.state.UserOffset = 0
		if err := m.updateStep(StepFolders); err != nil {
			return fmt.Errorf("failed to update step: %w", err)
		}
	}

	if m.state.Step <= StepFolders {
		if err := m.migrateFolders(); err != nil {
			m.saveState()
			return err
		}
		// Reset folder offset after completion
		m.state.FolderOffset = 0
		if err := m.updateStep(StepFolderParent); err != nil {
			return fmt.Errorf("failed to update step: %w", err)
		}
	}

	if m.state.Step <= StepFolderParent {
		if err := m.migrateFolderParent(); err != nil {
			return err
		}
		if err := m.updateStep(StepFile); err != nil {
			return fmt.Errorf("failed to update step: %w", err)
		}
	}

	if m.state.Step <= StepFile {
		if err := m.migrateFile(); err != nil {
			return err
		}
		if err := m.updateStep(StepShare); err != nil {
			return fmt.Errorf("failed to update step: %w", err)
		}
	}

	if m.state.Step <= StepShare {
		if err := m.migrateShare(); err != nil {
			return err
		}
		if err := m.updateStep(StepDirectLink); err != nil {
			return fmt.Errorf("failed to update step: %w", err)
		}
	}

	if m.state.Step <= StepDirectLink {
		if err := m.migrateDirectLink(); err != nil {
			return err
		}
		if err := m.updateStep(StepAvatar); err != nil {
			return fmt.Errorf("failed to update step: %w", err)
		}
	}

	if m.state.Step <= StepAvatar {
		if err := migrateAvatars(m); err != nil {
			return err
		}
		if err := m.updateStep(StepWebdav); err != nil {
			return fmt.Errorf("failed to update step: %w", err)
		}
	}

	if m.state.Step <= StepWebdav {
		if err := m.migrateWebdav(); err != nil {
			return err
		}
		if err := m.updateStep(StepCompleted); err != nil {
			return fmt.Errorf("failed to update step: %w", err)
		}
	}
	m.l.Info("Migration completed successfully")
	return nil
}

func formatTime(t time.Time) time.Time {
	newTime := time.UnixMilli(t.UnixMilli())
	return newTime
}
