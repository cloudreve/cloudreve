package migrator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/migrator/model"
	"github.com/cloudreve/Cloudreve/v4/ent/node"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"

	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/samber/lo"
)

func (m *Migrator) migratePolicy() (map[int]bool, error) {
	m.l.Info("Migrating storage policies...")
	var policies []model.Policy
	if err := model.DB.Find(&policies).Error; err != nil {
		return nil, fmt.Errorf("failed to list v3 storage policies: %w", err)
	}

	if m.state.LocalPolicyIDs == nil {
		m.state.LocalPolicyIDs = make(map[int]bool)
	}

	if m.state.PolicyIDs == nil {
		m.state.PolicyIDs = make(map[int]bool)
	}

	m.l.Info("Found %d v3 storage policies to be migrated.", len(policies))

	// get thumb proxy settings
	var (
		thumbProxySettings []model.Setting
		thumbProxyEnabled  bool
		thumbProxyPolicy   []int
	)
	if err := model.DB.Where("name in (?)", []string{"thumb_proxy_enabled", "thumb_proxy_policy"}).Find(&thumbProxySettings).Error; err != nil {
		m.l.Warning("Failed to list v3 thumb proxy settings: %w", err)
	}

	tx, err := m.v4client.Tx(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	for _, s := range thumbProxySettings {
		if s.Name == "thumb_proxy_enabled" {
			thumbProxyEnabled = setting.IsTrueValue(s.Value)
		} else if s.Name == "thumb_proxy_policy" {
			if err := json.Unmarshal([]byte(s.Value), &thumbProxyPolicy); err != nil {
				m.l.Warning("Failed to unmarshal v3 thumb proxy policy: %w", err)
			}
		}
	}

	for _, policy := range policies {
		m.l.Info("Migrating storage policy %q...", policy.Name)
		if err := json.Unmarshal([]byte(policy.Options), &policy.OptionsSerialized); err != nil {
			return nil, fmt.Errorf("failed to unmarshal options for policy %q: %w", policy.Name, err)
		}

		settings := &types.PolicySetting{
			Token:              policy.OptionsSerialized.Token,
			FileType:           policy.OptionsSerialized.FileType,
			OauthRedirect:      policy.OptionsSerialized.OauthRedirect,
			OdDriver:           policy.OptionsSerialized.OdDriver,
			Region:             policy.OptionsSerialized.Region,
			ServerSideEndpoint: policy.OptionsSerialized.ServerSideEndpoint,
			ChunkSize:          int64(policy.OptionsSerialized.ChunkSize),
			TPSLimit:           policy.OptionsSerialized.TPSLimit,
			TPSLimitBurst:      policy.OptionsSerialized.TPSLimitBurst,
			S3ForcePathStyle:   policy.OptionsSerialized.S3ForcePathStyle,
			ThumbExts:          policy.OptionsSerialized.ThumbExts,
		}

		if policy.Type == types.PolicyTypeOd {
			settings.ThumbSupportAllExts = true
		} else {
			switch policy.Type {
			case types.PolicyTypeCos:
				settings.ThumbExts = []string{"png", "jpg", "jpeg", "gif", "bmp", "webp", "heif", "heic"}
			case types.PolicyTypeOss:
				settings.ThumbExts = []string{"png", "jpg", "jpeg", "gif", "bmp", "webp", "heic", "tiff", "avif"}
			case types.PolicyTypeUpyun:
				settings.ThumbExts = []string{"png", "jpg", "jpeg", "gif", "bmp", "webp", "svg"}
			case types.PolicyTypeQiniu:
				settings.ThumbExts = []string{"png", "jpg", "jpeg", "gif", "bmp", "webp", "tiff", "avif", "psd"}
			case types.PolicyTypeRemote:
				settings.ThumbExts = []string{"png", "jpg", "jpeg", "gif"}
			}
		}

		if policy.Type != types.PolicyTypeOd && policy.BaseURL != "" {
			settings.CustomProxy = true
			settings.ProxyServer = policy.BaseURL
		} else if policy.OptionsSerialized.OdProxy != "" {
			settings.CustomProxy = true
			settings.ProxyServer = policy.OptionsSerialized.OdProxy
		}

		if policy.Type == types.PolicyTypeCos {
			settings.ChunkSize = 1024 * 1024 * 25
		}

		if thumbProxyEnabled && lo.Contains(thumbProxyPolicy, int(policy.ID)) {
			settings.ThumbGeneratorProxy = true
		}

		mustContain := []string{"{randomkey16}", "{randomkey8}", "{uuid}"}
		hasRandomElement := false
		for _, c := range mustContain {
			if strings.Contains(policy.FileNameRule, c) {
				hasRandomElement = true
				break
			}

			if strings.Contains(policy.DirNameRule, c) {
				hasRandomElement = true
				break
			}
		}
		if !hasRandomElement {
			policy.FileNameRule = "{uid}_{randomkey8}_{originname}"
			m.l.Warning("Storage policy %q has no random element in file name rule, using default file name rule.", policy.Name)
		}

		stm := tx.StoragePolicy.Create().
			SetRawID(int(policy.ID)).
			SetCreatedAt(formatTime(policy.CreatedAt)).
			SetUpdatedAt(formatTime(policy.UpdatedAt)).
			SetName(policy.Name).
			SetType(policy.Type).
			SetServer(policy.Server).
			SetBucketName(policy.BucketName).
			SetIsPrivate(policy.IsPrivate).
			SetAccessKey(policy.AccessKey).
			SetSecretKey(policy.SecretKey).
			SetMaxSize(int64(policy.MaxSize)).
			SetDirNameRule(policy.DirNameRule).
			SetFileNameRule(policy.FileNameRule).
			SetSettings(settings)

		if policy.Type == types.PolicyTypeRemote {
			m.l.Info("Storage policy %q is remote, creating node for it...", policy.Name)
			bs := &boolset.BooleanSet{}
			n, err := tx.Node.Create().
				SetName(policy.Name).
				SetStatus(node.StatusActive).
				SetServer(policy.Server).
				SetSlaveKey(policy.SecretKey).
				SetType(node.TypeSlave).
				SetCapabilities(bs).
				SetSettings(&types.NodeSetting{
					Provider: types.DownloaderProviderAria2,
				}).
				Save(context.Background())
			if err != nil {
				return nil, fmt.Errorf("failed to create node for storage policy %q: %w", policy.Name, err)
			}

			stm.SetNodeID(n.ID)
		}

		if _, err := stm.Save(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to create storage policy %q: %w", policy.Name, err)
		}

		m.state.PolicyIDs[int(policy.ID)] = true
		if policy.Type == types.PolicyTypeLocal {
			m.state.LocalPolicyIDs[int(policy.ID)] = true
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	if m.dep.ConfigProvider().Database().Type == conf.PostgresDB {
		m.l.Info("Resetting storage policy ID sequence for postgres...")
		m.v4client.StoragePolicy.ExecContext(context.Background(), "SELECT SETVAL('storage_policies_id_seq',  (SELECT MAX(id) FROM storage_policies))")
	}

	if m.dep.ConfigProvider().Database().Type == conf.PostgresDB {
		m.l.Info("Resetting node ID sequence for postgres...")
		m.v4client.Node.ExecContext(context.Background(), "SELECT SETVAL('nodes_id_seq',  (SELECT MAX(id) FROM nodes))")
	}

	return m.state.PolicyIDs, nil
}
