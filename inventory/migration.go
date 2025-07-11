package inventory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/group"
	"github.com/cloudreve/Cloudreve/v4/ent/node"
	"github.com/cloudreve/Cloudreve/v4/ent/setting"
	"github.com/cloudreve/Cloudreve/v4/ent/storagepolicy"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/samber/lo"
)

// needMigration exams if required schema version is satisfied.
func needMigration(client *ent.Client, ctx context.Context, requiredDbVersion string) bool {
	c, _ := client.Setting.Query().Where(setting.NameEQ(DBVersionPrefix + requiredDbVersion)).Count(ctx)
	return c == 0
}

func migrate(l logging.Logger, client *ent.Client, ctx context.Context, kv cache.Driver, requiredDbVersion string) error {
	l.Info("Start initializing database schema...")
	l.Info("Creating basic table schema...")
	if err := client.Schema.Create(ctx); err != nil {
		return fmt.Errorf("Failed creating schema resources: %w", err)
	}

	migrateDefaultSettings(l, client, ctx, kv)

	if err := migrateDefaultStoragePolicy(l, client, ctx); err != nil {
		return fmt.Errorf("failed migrating default storage policy: %w", err)
	}

	if err := migrateSysGroups(l, client, ctx); err != nil {
		return fmt.Errorf("failed migrating default storage policy: %w", err)
	}

	if err := applyPatches(l, client, ctx, requiredDbVersion); err != nil {
		return fmt.Errorf("failed applying schema patches: %w", err)
	}

	client.Setting.Create().SetName(DBVersionPrefix + requiredDbVersion).SetValue("installed").Save(ctx)
	return nil
}

func migrateDefaultSettings(l logging.Logger, client *ent.Client, ctx context.Context, kv cache.Driver) {
	// clean kv cache
	if err := kv.DeleteAll(); err != nil {
		l.Warning("Failed to remove all KV entries while schema migration: %s", err)
	}

	// List existing settings into a map
	existingSettings := make(map[string]struct{})
	settings, err := client.Setting.Query().All(ctx)
	if err != nil {
		l.Warning("Failed to query existing settings: %s", err)
	}

	for _, s := range settings {
		existingSettings[s.Name] = struct{}{}
	}

	l.Info("Insert default settings...")
	for k, v := range DefaultSettings {
		if _, ok := existingSettings[k]; ok {
			l.Debug("Skip inserting setting %s, already exists.", k)
			continue
		}

		if override, ok := os.LookupEnv(EnvDefaultOverwritePrefix + k); ok {
			l.Info("Override default setting %q with env value %q", k, override)
			v = override
		}

		client.Setting.Create().SetName(k).SetValue(v).SaveX(ctx)
	}
}

func migrateDefaultStoragePolicy(l logging.Logger, client *ent.Client, ctx context.Context) error {
	if _, err := client.StoragePolicy.Query().Where(storagepolicy.ID(1)).First(ctx); err == nil {
		l.Info("Default storage policy (ID=1) already exists, skip migrating.")
		return nil
	}

	l.Info("Insert default storage policy...")
	if _, err := client.StoragePolicy.Create().
		SetName("Default storage policy").
		SetType(types.PolicyTypeLocal).
		SetDirNameRule(util.DataPath("uploads/{uid}/{path}")).
		SetFileNameRule("{uid}_{randomkey8}_{originname}").
		SetSettings(&types.PolicySetting{
			ChunkSize:   25 << 20, // 25MB
			PreAllocate: true,
		}).
		Save(ctx); err != nil {
		return fmt.Errorf("failed to create default storage policy: %w", err)
	}

	return nil
}

func migrateSysGroups(l logging.Logger, client *ent.Client, ctx context.Context) error {
	if err := migrateAdminGroup(l, client, ctx); err != nil {
		return err
	}

	if err := migrateUserGroup(l, client, ctx); err != nil {
		return err
	}

	if err := migrateAnonymousGroup(l, client, ctx); err != nil {
		return err
	}

	if err := migrateMasterNode(l, client, ctx); err != nil {
		return err
	}

	return nil
}

func migrateAdminGroup(l logging.Logger, client *ent.Client, ctx context.Context) error {
	if _, err := client.Group.Query().Where(group.ID(1)).First(ctx); err == nil {
		l.Info("Default admin group (ID=1) already exists, skip migrating.")
		return nil
	}

	l.Info("Insert default admin group...")
	permissions := &boolset.BooleanSet{}
	boolset.Sets(map[types.GroupPermission]bool{
		types.GroupPermissionIsAdmin:             true,
		types.GroupPermissionShare:               true,
		types.GroupPermissionWebDAV:              true,
		types.GroupPermissionWebDAVProxy:         true,
		types.GroupPermissionArchiveDownload:     true,
		types.GroupPermissionArchiveTask:         true,
		types.GroupPermissionShareDownload:       true,
		types.GroupPermissionRemoteDownload:      true,
		types.GroupPermissionRedirectedSource:    true,
		types.GroupPermissionAdvanceDelete:       true,
		types.GroupPermissionIgnoreFileOwnership: true,
		// TODO: review default permission
	}, permissions)
	if _, err := client.Group.Create().
		SetName("Admin").
		SetStoragePoliciesID(1).
		SetMaxStorage(1 * constants.TB). // 1 TB default storage
		SetPermissions(permissions).
		SetSettings(&types.GroupSetting{
			SourceBatchSize:  1000,
			Aria2BatchSize:   50,
			MaxWalkedFiles:   100000,
			TrashRetention:   7 * 24 * 3600,
			RedirectedSource: true,
		}).
		Save(ctx); err != nil {
		return fmt.Errorf("failed to create default admin group: %w", err)
	}

	return nil
}

func migrateUserGroup(l logging.Logger, client *ent.Client, ctx context.Context) error {
	if _, err := client.Group.Query().Where(group.ID(2)).First(ctx); err == nil {
		l.Info("Default user group (ID=2) already exists, skip migrating.")
		return nil
	}

	l.Info("Insert default user group...")
	permissions := &boolset.BooleanSet{}
	boolset.Sets(map[types.GroupPermission]bool{
		types.GroupPermissionShare:            true,
		types.GroupPermissionShareDownload:    true,
		types.GroupPermissionRedirectedSource: true,
	}, permissions)
	if _, err := client.Group.Create().
		SetName("User").
		SetStoragePoliciesID(1).
		SetMaxStorage(1 * constants.GB). // 1 GB default storage
		SetPermissions(permissions).
		SetSettings(&types.GroupSetting{
			SourceBatchSize:  10,
			Aria2BatchSize:   1,
			MaxWalkedFiles:   100000,
			TrashRetention:   7 * 24 * 3600,
			RedirectedSource: true,
		}).
		Save(ctx); err != nil {
		return fmt.Errorf("failed to create default user group: %w", err)
	}

	return nil
}

func migrateAnonymousGroup(l logging.Logger, client *ent.Client, ctx context.Context) error {
	if _, err := client.Group.Query().Where(group.ID(AnonymousGroupID)).First(ctx); err == nil {
		l.Info("Default anonymous group (ID=3) already exists, skip migrating.")
		return nil
	}

	l.Info("Insert default anonymous group...")
	permissions := &boolset.BooleanSet{}
	boolset.Sets(map[types.GroupPermission]bool{
		types.GroupPermissionIsAnonymous:   true,
		types.GroupPermissionShareDownload: true,
	}, permissions)
	if _, err := client.Group.Create().
		SetName("Anonymous").
		SetPermissions(permissions).
		SetSettings(&types.GroupSetting{
			MaxWalkedFiles:   100000,
			RedirectedSource: true,
		}).
		Save(ctx); err != nil {
		return fmt.Errorf("failed to create default anonymous group: %w", err)
	}

	return nil
}

func migrateMasterNode(l logging.Logger, client *ent.Client, ctx context.Context) error {
	if _, err := client.Node.Query().Where(node.TypeEQ(node.TypeMaster)).First(ctx); err == nil {
		l.Info("Default master node already exists, skip migrating.")
		return nil
	}

	capabilities := &boolset.BooleanSet{}
	boolset.Sets(map[types.NodeCapability]bool{
		types.NodeCapabilityCreateArchive:  true,
		types.NodeCapabilityExtractArchive: true,
		types.NodeCapabilityRemoteDownload: true,
	}, capabilities)

	stm := client.Node.Create().
		SetType(node.TypeMaster).
		SetCapabilities(capabilities).
		SetName("Master").
		SetSettings(&types.NodeSetting{
			Provider: types.DownloaderProviderAria2,
		}).
		SetStatus(node.StatusActive)

	_, enableAria2 := os.LookupEnv(EnvEnableAria2)
	if enableAria2 {
		l.Info("Aria2 is override as enabled.")
		stm.SetSettings(&types.NodeSetting{
			Provider: types.DownloaderProviderAria2,
			Aria2Setting: &types.Aria2Setting{
				Server: "http://127.0.0.1:6800/jsonrpc",
			},
		})
	}

	l.Info("Insert default master node...")
	if _, err := stm.Save(ctx); err != nil {
		return fmt.Errorf("failed to create default master node: %w", err)
	}

	return nil
}

type (
	PatchFunc func(l logging.Logger, client *ent.Client, ctx context.Context) error
	Patch     struct {
		Name       string
		EndVersion string
		Func       PatchFunc
	}
)

var patches = []Patch{
	{
		Name:       "apply_default_excalidraw_viewer",
		EndVersion: "4.1.0",
		Func: func(l logging.Logger, client *ent.Client, ctx context.Context) error {
			// 1. Apply excalidraw file icons
			// 1.1 Check if it's already applied
			iconSetting, err := client.Setting.Query().Where(setting.Name("explorer_icons")).First(ctx)
			if err != nil {
				return fmt.Errorf("failed to query explorer_icons setting: %w", err)
			}

			var icons []types.FileTypeIconSetting
			if err := json.Unmarshal([]byte(iconSetting.Value), &icons); err != nil {
				return fmt.Errorf("failed to unmarshal explorer_icons setting: %w", err)
			}

			iconExisted := false
			for _, icon := range icons {
				if lo.Contains(icon.Exts, "excalidraw") {
					iconExisted = true
					break
				}
			}

			// 1.2 If not existed, add it
			if !iconExisted {
				// Found existing excalidraw icon default setting
				var defaultExcalidrawIcon types.FileTypeIconSetting
				for _, icon := range defaultIcons {
					if lo.Contains(icon.Exts, "excalidraw") {
						defaultExcalidrawIcon = icon
						break
					}
				}

				icons = append(icons, defaultExcalidrawIcon)
				newIconSetting, err := json.Marshal(icons)
				if err != nil {
					return fmt.Errorf("failed to marshal explorer_icons setting: %w", err)
				}

				if _, err := client.Setting.UpdateOne(iconSetting).SetValue(string(newIconSetting)).Save(ctx); err != nil {
					return fmt.Errorf("failed to update explorer_icons setting: %w", err)
				}
			}

			// 2. Apply default file viewers
			// 2.1 Check if it's already applied
			fileViewersSetting, err := client.Setting.Query().Where(setting.Name("file_viewers")).First(ctx)
			if err != nil {
				return fmt.Errorf("failed to query file_viewers setting: %w", err)
			}

			var fileViewers []types.ViewerGroup
			if err := json.Unmarshal([]byte(fileViewersSetting.Value), &fileViewers); err != nil {
				return fmt.Errorf("failed to unmarshal file_viewers setting: %w", err)
			}

			fileViewerExisted := false
			for _, viewer := range fileViewers[0].Viewers {
				if viewer.ID == "excalidraw" {
					fileViewerExisted = true
					break
				}
			}

			// 2.2 If not existed, add it
			if !fileViewerExisted {
				// Found existing excalidraw viewer default setting
				var defaultExcalidrawViewer types.Viewer
				for _, viewer := range defaultFileViewers[0].Viewers {
					if viewer.ID == "excalidraw" {
						defaultExcalidrawViewer = viewer
						break
					}
				}

				fileViewers[0].Viewers = append(fileViewers[0].Viewers, defaultExcalidrawViewer)
				newFileViewersSetting, err := json.Marshal(fileViewers)
				if err != nil {
					return fmt.Errorf("failed to marshal file_viewers setting: %w", err)
				}

				if _, err := client.Setting.UpdateOne(fileViewersSetting).SetValue(string(newFileViewersSetting)).Save(ctx); err != nil {
					return fmt.Errorf("failed to update file_viewers setting: %w", err)
				}
			}

			return nil
		},
	},
}

func applyPatches(l logging.Logger, client *ent.Client, ctx context.Context, requiredDbVersion string) error {
	allVersionMarks, err := client.Setting.Query().Where(setting.NameHasPrefix(DBVersionPrefix)).All(ctx)
	if err != nil {
		return err
	}

	requiredDbVersion = strings.TrimSuffix(requiredDbVersion, "-pro")

	// Find the latest applied version
	var latestAppliedVersion *semver.Version
	for _, v := range allVersionMarks {
		v.Name = strings.TrimSuffix(v.Name, "-pro")
		version, err := semver.NewVersion(strings.TrimPrefix(v.Name, DBVersionPrefix))
		if err != nil {
			l.Warning("Failed to parse past version %s: %s", v.Name, err)
			continue
		}
		if latestAppliedVersion == nil || version.Compare(latestAppliedVersion) > 0 {
			latestAppliedVersion = version
		}
	}

	requiredVersion, err := semver.NewVersion(requiredDbVersion)
	if err != nil {
		return fmt.Errorf("failed to parse required version %s: %w", requiredDbVersion, err)
	}

	if latestAppliedVersion == nil || requiredVersion.Compare(requiredVersion) > 0 {
		latestAppliedVersion = requiredVersion
	}

	for _, patch := range patches {
		if latestAppliedVersion.Compare(semver.MustParse(patch.EndVersion)) < 0 {
			l.Info("Applying schema patch %s...", patch.Name)
			if err := patch.Func(l, client, ctx); err != nil {
				return err
			}
		}
	}

	return nil
}
