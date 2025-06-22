package conf

import (
	"fmt"
	"os"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/go-ini/ini"
	"github.com/go-playground/validator/v10"
)

const (
	envConfOverrideKey = "CR_CONF_"
)

type ConfigProvider interface {
	Database() *Database
	System() *System
	SSL() *SSL
	Unix() *Unix
	Slave() *Slave
	Redis() *Redis
	Cors() *Cors
	OptionOverwrite() map[string]any
}

// NewIniConfigProvider initializes a new Ini config file provider. A default config file
// will be created if the given path does not exist.
func NewIniConfigProvider(configPath string, l logging.Logger) (ConfigProvider, error) {
	if configPath == "" || !util.Exists(configPath) {
		l.Info("Config file %q not found, creating a new one.", configPath)
		// 创建初始配置文件
		confContent := util.Replace(map[string]string{
			"{SessionSecret}": util.RandStringRunes(64),
		}, defaultConf)
		f, err := util.CreatNestedFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create config file: %w", err)
		}

		// 写入配置文件
		_, err = f.WriteString(confContent)
		if err != nil {
			return nil, fmt.Errorf("failed to write config file: %w", err)
		}

		f.Close()
	}

	cfg, err := ini.Load(configPath, []byte(getOverrideConfFromEnv(l)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file %q: %w", configPath, err)
	}

	provider := &iniConfigProvider{
		database:        *DatabaseConfig,
		system:          *SystemConfig,
		ssl:             *SSLConfig,
		unix:            *UnixConfig,
		slave:           *SlaveConfig,
		redis:           *RedisConfig,
		cors:            *CORSConfig,
		optionOverwrite: make(map[string]interface{}),
	}

	sections := map[string]interface{}{
		"Database":   &provider.database,
		"System":     &provider.system,
		"SSL":        &provider.ssl,
		"UnixSocket": &provider.unix,
		"Redis":      &provider.redis,
		"CORS":       &provider.cors,
		"Slave":      &provider.slave,
	}
	for sectionName, sectionStruct := range sections {
		err = mapSection(cfg, sectionName, sectionStruct)
		if err != nil {
			return nil, fmt.Errorf("failed to parse config section %q: %w", sectionName, err)
		}
	}

	// 映射数据库配置覆盖
	for _, key := range cfg.Section("OptionOverwrite").Keys() {
		provider.optionOverwrite[key.Name()] = key.Value()
	}

	return provider, nil
}

type iniConfigProvider struct {
	database        Database
	system          System
	ssl             SSL
	unix            Unix
	slave           Slave
	redis           Redis
	cors            Cors
	optionOverwrite map[string]any
}

func (i *iniConfigProvider) Database() *Database {
	return &i.database
}

func (i *iniConfigProvider) System() *System {
	return &i.system
}

func (i *iniConfigProvider) SSL() *SSL {
	return &i.ssl
}

func (i *iniConfigProvider) Unix() *Unix {
	return &i.unix
}

func (i *iniConfigProvider) Slave() *Slave {
	return &i.slave
}

func (i *iniConfigProvider) Redis() *Redis {
	return &i.redis
}

func (i *iniConfigProvider) Cors() *Cors {
	return &i.cors
}

func (i *iniConfigProvider) OptionOverwrite() map[string]any {
	return i.optionOverwrite
}

const defaultConf = `[System]
Debug = false
Mode = master
Listen = :5212
SessionSecret = {SessionSecret}
HashIDSalt = {HashIDSalt}
`

// mapSection 将配置文件的 Section 映射到结构体上
func mapSection(cfg *ini.File, section string, confStruct interface{}) error {
	err := cfg.Section(section).MapTo(confStruct)
	if err != nil {
		return err
	}

	// 验证合法性
	validate := validator.New()
	err = validate.Struct(confStruct)
	if err != nil {
		return err
	}

	return nil
}

func getOverrideConfFromEnv(l logging.Logger) string {
	confMaps := make(map[string]map[string]string)
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, envConfOverrideKey) {
			continue
		}

		// split by key=value and get key
		kv := strings.SplitN(env, "=", 2)
		configKey := strings.TrimPrefix(kv[0], envConfOverrideKey)
		configValue := kv[1]
		sectionKey := strings.SplitN(configKey, ".", 2)
		if confMaps[sectionKey[0]] == nil {
			confMaps[sectionKey[0]] = make(map[string]string)
		}

		confMaps[sectionKey[0]][sectionKey[1]] = configValue
		l.Info("Override config %q = %q", configKey, configValue)
	}

	// generate ini content
	var sb strings.Builder
	for section, kvs := range confMaps {
		sb.WriteString(fmt.Sprintf("[%s]\n", section))
		for k, v := range kvs {
			sb.WriteString(fmt.Sprintf("%s = %s\n", k, v))
		}
	}

	return sb.String()
}
