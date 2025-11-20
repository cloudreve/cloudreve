package encrypt

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"

	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
)

const (
	EnvMasterEncryptKey = "CR_ENCRYPT_MASTER_KEY"
)

// MasterEncryptKeyVault is a vault for the master encrypt key.
type MasterEncryptKeyVault interface {
	GetMasterKey(ctx context.Context) ([]byte, error)
}

func NewMasterEncryptKeyVault(ctx context.Context, settings setting.Provider) MasterEncryptKeyVault {
	vaultType := settings.MasterEncryptKeyVault(ctx)
	switch vaultType {
	case setting.MasterEncryptKeyVaultTypeEnv:
		return NewEnvMasterEncryptKeyVault()
	case setting.MasterEncryptKeyVaultTypeFile:
		return NewFileMasterEncryptKeyVault(settings.MasterEncryptKeyFile(ctx))
	default:
		return NewSettingMasterEncryptKeyVault(settings)
	}
}

// settingMasterEncryptKeyVault is a vault for the master encrypt key that gets the key from the setting KV.
type settingMasterEncryptKeyVault struct {
	setting setting.Provider
}

func NewSettingMasterEncryptKeyVault(setting setting.Provider) MasterEncryptKeyVault {
	return &settingMasterEncryptKeyVault{setting: setting}
}

func (v *settingMasterEncryptKeyVault) GetMasterKey(ctx context.Context) ([]byte, error) {
	key := v.setting.MasterEncryptKey(ctx)
	if key == nil {
		return nil, errors.New("master encrypt key is not set")
	}
	return key, nil
}

func NewEnvMasterEncryptKeyVault() MasterEncryptKeyVault {
	return &envMasterEncryptKeyVault{}
}

type envMasterEncryptKeyVault struct {
}

var envMasterKeyCache = []byte{}

func (v *envMasterEncryptKeyVault) GetMasterKey(ctx context.Context) ([]byte, error) {
	if len(envMasterKeyCache) > 0 {
		return envMasterKeyCache, nil
	}

	key := os.Getenv(EnvMasterEncryptKey)
	if key == "" {
		return nil, errors.New("master encrypt key is not set")
	}

	decodedKey, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("failed to decode master encrypt key: %w", err)
	}

	envMasterKeyCache = decodedKey
	return decodedKey, nil
}

func NewFileMasterEncryptKeyVault(path string) MasterEncryptKeyVault {
	return &fileMasterEncryptKeyVault{path: path}
}

var fileMasterKeyCache = []byte{}

type fileMasterEncryptKeyVault struct {
	path string
}

func (v *fileMasterEncryptKeyVault) GetMasterKey(ctx context.Context) ([]byte, error) {
	if len(fileMasterKeyCache) > 0 {
		return fileMasterKeyCache, nil
	}

	key, err := os.ReadFile(v.path)
	if err != nil {
		return nil, fmt.Errorf("invalid master encrypt key file")
	}

	decodedKey, err := base64.StdEncoding.DecodeString(string(key))
	if err != nil {
		return nil, fmt.Errorf("invalid master encrypt key")
	}
	fileMasterKeyCache = decodedKey
	return fileMasterKeyCache, nil
}
