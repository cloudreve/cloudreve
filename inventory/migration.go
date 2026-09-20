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
	"github.com/cloudreve/Cloudreve/v4/ent/oauthclient"
	"github.com/cloudreve/Cloudreve/v4/ent/setting"
	"github.com/cloudreve/Cloudreve/v4/ent/storagepolicy"
	"github.com/cloudreve/Cloudreve/v4/inventory/debug"
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
	if c == 0 {
		return true
	}

	c, _ = client.Setting.Query().Where(
		setting.NameEQ(OIDCSigningPrivateKeySetting),
		setting.ValueNEQ(""),
	).Count(ctx)
	return c == 0
}

func migrate(l logging.Logger, client *ent.Client, ctx context.Context, kv cache.Driver, requiredDbVersion string) error {
	l.Info("Start initializing database schema...")
	l.Info("Creating basic table schema...")
	repairLegacyUserGroupColumn(l, client, ctx)
	RepairHeatWaveNotSecondary(l, client, ctx)
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

	if err := migrateOAuthClient(l, client, ctx); err != nil {
		return fmt.Errorf("failed migrating OAuth client: %w", err)
	}

	if err := applyPatches(l, client, ctx, requiredDbVersion); err != nil {
		return fmt.Errorf("failed applying schema patches: %w", err)
	}

	client.Setting.Create().SetName(DBVersionPrefix + requiredDbVersion).SetValue("installed").Save(ctx)
	return nil
}

// repairLegacyUserGroupColumn backfills users.group_users before ent's
// auto-migration. v3 bound user groups via a nullable group_id column (or no
// column at all for groupless users); the v4 schema requires NOT NULL
// group_users, so the schema copy fails mid-upgrade on old databases
// (upstream #2934). All statements are error-tolerant: on a fresh install
// the users/groups tables simply do not exist yet and every statement no-ops.
func repairLegacyUserGroupColumn(l logging.Logger, client *ent.Client, ctx context.Context) {
	if _, err := client.ExecContext(ctx, `ALTER TABLE users ADD COLUMN group_users INTEGER`); err != nil {
		l.Debug("Skip adding group_users column: %s", err)
	}

	// Prefer the legacy group_id binding when present; otherwise fall back to
	// the first non-admin group (id 1 is the seeded admin group), then any group.
	queries := []string{
		`UPDATE users SET group_users = COALESCE(group_id, (SELECT MIN(id) FROM groups WHERE id <> 1), (SELECT MIN(id) FROM groups)) WHERE group_users IS NULL`,
		`UPDATE users SET group_users = COALESCE((SELECT MIN(id) FROM groups WHERE id <> 1), (SELECT MIN(id) FROM groups)) WHERE group_users IS NULL`,
	}
	for _, q := range queries {
		_, err := client.ExecContext(ctx, q)
		if err == nil {
			return
		}
		l.Debug("Skip legacy group_users backfill: %s", err)
	}
}

// RepairHeatWaveNotSecondary strips the MySQL HeatWave `NOT SECONDARY` column
// attribute before ent's schema migration. Atlas treats it as an unknown EXTRA
// value and aborts schema inspection (upstream #3452). For every flagged column
// we replay its own definition from SHOW CREATE TABLE minus the attribute via
// ALTER TABLE ... MODIFY COLUMN, which clears it without touching the logical
// schema. On non-MySQL backends the INFORMATION_SCHEMA query fails and the whole
// function is a no-op.
func RepairHeatWaveNotSecondary(l logging.Logger, client *ent.Client, ctx context.Context) {
	rows, err := client.QueryContext(ctx,
		`SELECT TABLE_NAME, COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS
		 WHERE TABLE_SCHEMA = DATABASE() AND UPPER(EXTRA) LIKE '%NOT SECONDARY%'`)
	if err != nil {
		l.Debug("Skip HeatWave NOT SECONDARY repair: %s", err)
		return
	}

	var flagged [][2]string
	for rows.Next() {
		var table, column string
		if err := rows.Scan(&table, &column); err != nil {
			l.Debug("Skip HeatWave NOT SECONDARY repair scan: %s", err)
			rows.Close()
			return
		}
		flagged = append(flagged, [2]string{table, column})
	}
	rows.Close()

	for _, tc := range flagged {
		def, err := heatWaveColumnDef(client, ctx, tc[0], tc[1])
		if err != nil {
			l.Warning("Skip NOT SECONDARY repair for %s.%s: %s", tc[0], tc[1], err)
			continue
		}

		stm := fmt.Sprintf("ALTER TABLE `%s` MODIFY COLUMN %s",
			strings.ReplaceAll(tc[0], "`", "``"), def)
		if _, err := client.ExecContext(ctx, stm); err != nil {
			l.Warning("Failed to strip NOT SECONDARY on %s.%s: %s", tc[0], tc[1], err)
			continue
		}
		l.Info("Stripped HeatWave NOT SECONDARY attribute on %s.%s", tc[0], tc[1])
	}
}

// heatWaveColumnDef extracts a column's full definition from SHOW CREATE TABLE
// and returns it without the `NOT SECONDARY` attribute, ready for use after
// ALTER TABLE ... MODIFY COLUMN.
func heatWaveColumnDef(client *ent.Client, ctx context.Context, table, column string) (string, error) {
	rows, err := client.QueryContext(ctx,
		fmt.Sprintf("SHOW CREATE TABLE `%s`", strings.ReplaceAll(table, "`", "``")))
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var name, ddl string
	if !rows.Next() {
		return "", fmt.Errorf("table %s not found", table)
	}
	if err := rows.Scan(&name, &ddl); err != nil {
		return "", err
	}

	return extractColumnDef(ddl, column)
}

// extractColumnDef finds `column`'s definition line in SHOW CREATE TABLE output
// and returns it without the `NOT SECONDARY` attribute.
func extractColumnDef(ddl, column string) (string, error) {
	quoted := "`" + strings.ReplaceAll(column, "`", "``") + "`"
	for _, line := range strings.Split(ddl, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, quoted+" ") {
			continue
		}
		def := stripNotSecondary(strings.TrimSuffix(line, ","))
		if findNotSecondary(def) >= 0 {
			return "", fmt.Errorf("failed to strip NOT SECONDARY from column %s", column)
		}
		return def, nil
	}
	return "", fmt.Errorf("column %s not found in table definition", column)
}

func stripNotSecondary(def string) string {
	// Remove the `NOT SECONDARY` attribute clause, but not the same words
	// inside a quoted literal (e.g. DEFAULT 'not secondary').
	out := def
	for {
		idx := findNotSecondary(out)
		if idx < 0 {
			return strings.TrimSpace(out)
		}
		out = out[:idx] + out[idx+len("NOT SECONDARY"):]
	}
}

// findNotSecondary returns the index of the first `NOT SECONDARY` occurrence
// outside single-quoted string literals, or -1.
func findNotSecondary(def string) int {
	const marker = "NOT SECONDARY"
	upper := strings.ToUpper(def)
	inQuote := false
	for i := 0; i+len(marker) <= len(upper); i++ {
		if upper[i] == '\'' {
			if inQuote && i+1 < len(upper) && upper[i+1] == '\'' {
				i++ // escaped '' inside a literal
				continue
			}
			inQuote = !inQuote
		}
		if !inQuote && strings.HasPrefix(upper[i:], marker) {
			end := i + len(marker)
			if (i == 0 || upper[i-1] == ' ') && (end == len(upper) || upper[end] == ' ' || upper[end] == ',') {
				return i
			}
		}
	}
	return -1
}

func migrateDefaultSettings(l logging.Logger, client *ent.Client, ctx context.Context, kv cache.Driver) {
	// clean kv cache
	if err := kv.DeleteAll(); err != nil {
		l.Warning("Failed to remove all KV entries while schema migration: %s", err)
	}

	// List existing settings into a map
	existingSettings := make(map[string]*ent.Setting)
	settings, err := client.Setting.Query().All(ctx)
	if err != nil {
		l.Warning("Failed to query existing settings: %s", err)
	}

	for _, s := range settings {
		existingSettings[s.Name] = s
	}

	l.Info("Insert default settings...")
	for k, v := range DefaultSettings {
		if existing, ok := existingSettings[k]; ok {
			if k == OIDCSigningPrivateKeySetting && existing.Value == "" {
				client.Setting.UpdateOne(existing).SetValue(v).SaveX(ctx)
			}
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

const (
	OAuthClientDesktopGUID        = "393a1839-f52e-498e-9972-e77cc2241eee"
	OAuthClientDesktopName        = "application:oauth.desktop"
	OAuthClientDesktopRedirectURI = "/callback/desktop"
	OAuthClientCLIGUID            = "6326d2af-2fef-4a99-94da-1ee8ef0ca53f"
	OAuthClientCLIName            = "Cloudreve CLI"
	OAuthClientCLIRedirectURI     = "http://127.0.0.1/callback"
	OAuthClientiOSGUID            = "220db97a-44a3-44f7-99b6-d767262b4daa"
	OAuthClientiOSName            = "application:setting.iOSApp"
	OAuthClientiOSRedirectURI     = "/callback/ios"
)

func migrateOAuthClient(l logging.Logger, client *ent.Client, ctx context.Context) error {
	if err := migrateOAuthClientDesktop(l, client, ctx); err != nil {
		return err
	}

	if err := migrateOAuthClientiOS(l, client, ctx); err != nil {
		return err
	}

	if err := migrateOAuthClientCLI(l, client, ctx); err != nil {
		return err
	}

	return nil
}

// migrateOAuthClientCLI preserves administrator changes to an existing built-in client.
func migrateOAuthClientCLI(l logging.Logger, client *ent.Client, ctx context.Context) error {
	if _, err := client.OAuthClient.Query().Where(oauthclient.GUID(OAuthClientCLIGUID)).First(ctx); err == nil {
		l.Info("Default OAuth client (GUID=%s) already exists, skip migrating.", OAuthClientCLIGUID)
		return nil
	} else if !ent.IsNotFound(err) {
		return fmt.Errorf("failed to query default CLI OAuth client: %w", err)
	}

	if _, err := client.OAuthClient.Create().
		SetGUID(OAuthClientCLIGUID).
		SetSecret("").
		SetName(OAuthClientCLIName).
		SetRedirectUris([]string{OAuthClientCLIRedirectURI}).
		SetScopes([]string{"profile", "email", "openid", "offline_access", "UserInfo.Write", "Workflow.Write", "Files.Write", "Shares.Write"}).
		SetProps(&types.OAuthClientProps{Icon: "/static/img/cloudreve.svg", RefreshTokenTTL: 7776000}).
		SetIsEnabled(true).
		Save(ctx); err != nil {
		return fmt.Errorf("failed to create default CLI OAuth client: %w", err)
	}
	return nil
}

func migrateOAuthClientiOS(l logging.Logger, client *ent.Client, ctx context.Context) error {
	if _, err := client.OAuthClient.Query().Where(oauthclient.GUID(OAuthClientiOSGUID)).First(ctx); err == nil {
		l.Info("Default OAuth client (GUID=%s) already exists, skip migrating.", OAuthClientiOSGUID)
		return nil
	}
	if _, err := client.OAuthClient.Create().
		SetGUID(OAuthClientiOSGUID).
		SetSecret("").
		SetName(OAuthClientiOSName).
		SetRedirectUris([]string{OAuthClientiOSRedirectURI}).
		SetScopes([]string{"profile", "email", "openid", "offline_access", "UserInfo.Write", "UserSecurityInfo.Write", "Workflow.Write", "Files.Write", "Shares.Write", "Finance.Write", "DavAccount.Write"}).
		SetProps(&types.OAuthClientProps{Icon: "/static/img/cloudreve_ios.svg", RefreshTokenTTL: 7776000}).
		SetIsEnabled(true).
		Save(ctx); err != nil {
		return fmt.Errorf("failed to create default OAuth client: %w", err)
	}

	return nil
}

func migrateOAuthClientDesktop(l logging.Logger, client *ent.Client, ctx context.Context) error {
	if _, err := client.OAuthClient.Query().Where(oauthclient.GUID(OAuthClientDesktopGUID)).First(ctx); err == nil {
		l.Info("Default OAuth client (GUID=%s) already exists, skip migrating.", OAuthClientDesktopGUID)
		return nil
	}

	if _, err := client.OAuthClient.Create().
		SetGUID(OAuthClientDesktopGUID).
		SetSecret("").
		SetName(OAuthClientDesktopName).
		SetRedirectUris([]string{OAuthClientDesktopRedirectURI}).
		SetScopes([]string{"profile", "email", "openid", "offline_access", "UserInfo.Write", "Workflow.Write", "Files.Write", "Shares.Write"}).
		SetProps(&types.OAuthClientProps{Icon: "/static/img/cloudreve.svg", RefreshTokenTTL: 7776000}).
		SetIsEnabled(true).
		Save(ctx); err != nil {
		return fmt.Errorf("failed to create default OAuth client: %w", err)
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
		Name:       "apply_default_archive_viewer",
		EndVersion: "4.7.0",
		Func: func(l logging.Logger, client *ent.Client, ctx context.Context) error {
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
				if viewer.ID == "archive" {
					fileViewerExisted = true
					break
				}
			}

			// 2.2 If not existed, add it
			if !fileViewerExisted {
				// Found existing archive viewer default setting
				var defaultArchiveViewer types.Viewer
				for _, viewer := range defaultFileViewers[0].Viewers {
					if viewer.ID == "archive" {
						defaultArchiveViewer = viewer
						break
					}
				}

				fileViewers[0].Viewers = append(fileViewers[0].Viewers, defaultArchiveViewer)
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
	{
		Name:       "apply_email_title_magic_var",
		EndVersion: "4.7.0",
		Func: func(l logging.Logger, client *ent.Client, ctx context.Context) error {
			// 1. Activate Template
			mailActivationTemplateSetting, err := client.Setting.Query().Where(setting.Name("mail_activation_template")).First(ctx)
			if err != nil {
				return fmt.Errorf("failed to query mail_activation_template setting: %w", err)
			}

			var mailActivationTemplate []struct {
				Title    string `json:"title"`
				Body     string `json:"body"`
				Language string `json:"language"`
			}
			if err := json.Unmarshal([]byte(mailActivationTemplateSetting.Value), &mailActivationTemplate); err != nil {
				return fmt.Errorf("failed to unmarshal mail_activation_template setting: %w", err)
			}

			for i, t := range mailActivationTemplate {
				mailActivationTemplate[i].Title = fmt.Sprintf("[{{ .CommonContext.SiteBasic.Name }}] %s", t.Title)
			}

			newMailActivationTemplate, err := json.Marshal(mailActivationTemplate)
			if err != nil {
				return fmt.Errorf("failed to marshal mail_activation_template setting: %w", err)
			}

			if _, err := client.Setting.UpdateOne(mailActivationTemplateSetting).SetValue(string(newMailActivationTemplate)).Save(ctx); err != nil {
				return fmt.Errorf("failed to update mail_activation_template setting: %w", err)
			}

			// 2. Reset Password Template
			mailResetTemplateSetting, err := client.Setting.Query().Where(setting.Name("mail_reset_template")).First(ctx)
			if err != nil {
				return fmt.Errorf("failed to query mail_reset_template setting: %w", err)
			}

			var mailResetTemplate []struct {
				Title    string `json:"title"`
				Body     string `json:"body"`
				Language string `json:"language"`
			}
			if err := json.Unmarshal([]byte(mailResetTemplateSetting.Value), &mailResetTemplate); err != nil {
				return fmt.Errorf("failed to unmarshal mail_reset_template setting: %w", err)
			}

			for i, t := range mailResetTemplate {
				mailResetTemplate[i].Title = fmt.Sprintf("[{{ .CommonContext.SiteBasic.Name }}] %s", t.Title)
			}

			newMailResetTemplate, err := json.Marshal(mailResetTemplate)
			if err != nil {
				return fmt.Errorf("failed to marshal mail_reset_template setting: %w", err)
			}

			if _, err := client.Setting.UpdateOne(mailResetTemplateSetting).SetValue(string(newMailResetTemplate)).Save(ctx); err != nil {
				return fmt.Errorf("failed to update mail_reset_template setting: %w", err)
			}

			return nil
		},
	},
	{
		Name:       "apply_thumb_path_magic_var",
		EndVersion: "4.10.0",
		Func: func(l logging.Logger, client *ent.Client, ctx context.Context) error {
			thumbSuffixSetting, err := client.Setting.Query().Where(setting.Name("thumb_entity_suffix")).First(ctx)
			if err != nil {
				return fmt.Errorf("failed to query thumb_entity_suffix setting: %w", err)
			}

			newThumbSuffix := fmt.Sprintf("{blob_path}/{blob_name}%s", thumbSuffixSetting.Value)
			if _, err := client.Setting.UpdateOne(thumbSuffixSetting).SetValue(newThumbSuffix).Save(ctx); err != nil {
				return fmt.Errorf("failed to update thumb_entity_suffix setting: %w", err)
			}

			return nil
		},
	},
	{
		Name:       "apply_default_model3d_viewer",
		EndVersion: "4.20.0",
		Func: func(l logging.Logger, client *ent.Client, ctx context.Context) error {
			fileViewersSetting, err := client.Setting.Query().Where(setting.Name("file_viewers")).First(ctx)
			if err != nil {
				return fmt.Errorf("failed to query file_viewers setting: %w", err)
			}

			var fileViewers []types.ViewerGroup
			if err := json.Unmarshal([]byte(fileViewersSetting.Value), &fileViewers); err != nil {
				return fmt.Errorf("failed to unmarshal file_viewers setting: %w", err)
			}

			for _, viewer := range fileViewers[0].Viewers {
				if viewer.ID == "model3d" {
					return nil
				}
			}

			var defaultModelViewer types.Viewer
			for _, viewer := range defaultFileViewers[0].Viewers {
				if viewer.ID == "model3d" {
					defaultModelViewer = viewer
					break
				}
			}

			fileViewers[0].Viewers = append(fileViewers[0].Viewers, defaultModelViewer)
			newFileViewersSetting, err := json.Marshal(fileViewers)
			if err != nil {
				return fmt.Errorf("failed to marshal file_viewers setting: %w", err)
			}

			if _, err := client.Setting.UpdateOne(fileViewersSetting).SetValue(string(newFileViewersSetting)).Save(ctx); err != nil {
				return fmt.Errorf("failed to update file_viewers setting: %w", err)
			}

			return nil
		},
	},
	{
		Name:       "reset_secret_key",
		EndVersion: "4.13.0",
		Func: func(l logging.Logger, client *ent.Client, ctx context.Context) error {
			newSecretKey := util.RandStringRunesCrypto(256)
			ctx = context.WithValue(ctx, debug.SkipDbLogging{}, true)
			if err := client.Setting.Update().Where(setting.Name("secret_key")).SetValue(newSecretKey).Exec(ctx); err != nil {
				return fmt.Errorf("failed to update secret_key setting: %w", err)
			}

			return nil
		},
	},
	{
		// Built-in desktop/iOS clients are public clients shipped in binaries —
		// a shared hardcoded secret authenticates nothing. Blank it so token
		// exchange relies on PKCE (RFC 8252) instead.
		Name:       "oauth_builtin_public_clients",
		EndVersion: "4.20.0",
		Func: func(l logging.Logger, client *ent.Client, ctx context.Context) error {
			if _, err := client.OAuthClient.Update().
				Where(oauthclient.GUIDIn(OAuthClientDesktopGUID, OAuthClientiOSGUID)).
				SetSecret("").
				Save(ctx); err != nil {
				return fmt.Errorf("failed to clear built-in OAuth client secrets: %w", err)
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

	if latestAppliedVersion == nil || requiredVersion.Compare(latestAppliedVersion) > 0 {
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
