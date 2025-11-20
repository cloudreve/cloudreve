package conf

import (
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/go-ini/ini"
	"github.com/go-playground/validator/v10"
)

// database 数据库
type database struct {
	Type        string
	User        string
	Password    string
	Host        string
	Name        string
	TablePrefix string
	DBFile      string
	Port        int
	Charset     string
	UnixSocket  bool
}

// system 系统通用配置
type system struct {
	Mode          string `validate:"eq=master|eq=slave"`
	Listen        string `validate:"required"`
	Debug         bool
	SessionSecret string
	HashIDSalt    string
	GracePeriod   int `validate:"gte=0"`
	ProxyHeader   string
}

type ssl struct {
	CertPath string `validate:"omitempty,required"`
	KeyPath  string `validate:"omitempty,required"`
	Listen   string `validate:"required"`
}

type unix struct {
	Listen string
	Perm   uint32
}

// slave 作为slave存储端配置
type slave struct {
	Secret          string `validate:"omitempty,gte=64"`
	CallbackTimeout int    `validate:"omitempty,gte=1"`
	SignatureTTL    int    `validate:"omitempty,gte=1"`
}

// redis 配置
type redis struct {
	Network  string
	Server   string
	User     string
	Password string
	DB       string
}

// 跨域配置
type cors struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
	ExposeHeaders    []string
	SameSite         string
	Secure           bool
}

var cfg *ini.File

// Init 初始化配置文件
func Init(l logging.Logger, path string) error {
	var err error

	cfg, err = ini.Load(path)
	if err != nil {
		l.Error("Failed to parse config file %q: %s", path, err)
		return err
	}

	sections := map[string]interface{}{
		"Database":   DatabaseConfig,
		"System":     SystemConfig,
		"SSL":        SSLConfig,
		"UnixSocket": UnixConfig,
		"Redis":      RedisConfig,
		"CORS":       CORSConfig,
		"Slave":      SlaveConfig,
	}
	for sectionName, sectionStruct := range sections {
		err = mapSection(sectionName, sectionStruct)
		if err != nil {
			l.Error("Failed to parse config section %q: %s", sectionName, err)
			return err
		}
	}

	// 映射数据库配置覆盖
	for _, key := range cfg.Section("OptionOverwrite").Keys() {
		OptionOverwrite[key.Name()] = key.Value()
	}

	return nil
}

// mapSection 将配置文件的 Section 映射到结构体上
func mapSection(section string, confStruct interface{}) error {
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
