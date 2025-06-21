package conf

import "github.com/cloudreve/Cloudreve/v4/pkg/util"

type DBType string

var (
	SQLiteDB   DBType = "sqlite"
	SQLite3DB  DBType = "sqlite3"
	MySqlDB    DBType = "mysql"
	MsSqlDB    DBType = "mssql"
	PostgresDB DBType = "postgres"
)

// Database 数据库
type Database struct {
	Type        DBType
	User        string
	Password    string
	Host        string
	Name        string
	TablePrefix string
	DBFile      string
	Port        int
	Charset     string
	UnixSocket  bool
	// 允许直接使用DATABASE_URL来配置数据库连接
	DatabaseURL string
	// SSLMode 允许使用SSL连接数据库, 用户可以在sslmode string中添加证书等配置
	SSLMode string
}

type SysMode string

var (
	MasterMode SysMode = "master"
	SlaveMode  SysMode = "slave"
)

// System 系统通用配置
type System struct {
	Mode          SysMode `validate:"eq=master|eq=slave"`
	Listen        string  `validate:"required"`
	Debug         bool
	SessionSecret string
	HashIDSalt    string // deprecated
	GracePeriod   int    `validate:"gte=0"`
	ProxyHeader   string `validate:"required_with=Listen"`
	LogLevel      string `validate:"oneof=debug info warning error"`
}

type SSL struct {
	CertPath string `validate:"omitempty,required"`
	KeyPath  string `validate:"omitempty,required"`
	Listen   string `validate:"required"`
}

type Unix struct {
	Listen string
	Perm   uint32
}

// Slave 作为slave存储端配置
type Slave struct {
	Secret          string `validate:"omitempty,gte=64"`
	CallbackTimeout int    `validate:"omitempty,gte=1"`
	SignatureTTL    int    `validate:"omitempty,gte=1"`
}

// Redis 配置
type Redis struct {
	Network       string
	Server        string
	User          string
	Password      string
	DB            string
	UseSSL        bool
	TLSSkipVerify bool
}

// 跨域配置
type Cors struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
	ExposeHeaders    []string
	SameSite         string
	Secure           bool
}

// RedisConfig Redis服务器配置
var RedisConfig = &Redis{
	Network:       "tcp",
	Server:        "",
	Password:      "",
	DB:            "0",
	UseSSL:        false,
	TLSSkipVerify: true,
}

// DatabaseConfig 数据库配置
var DatabaseConfig = &Database{
	Charset:     "utf8mb4",
	DBFile:      util.DataPath("cloudreve.db"),
	Port:        3306,
	UnixSocket:  false,
	DatabaseURL: util.DataPath(""),
	SSLMode:     "allow",
}

// SystemConfig 系统公用配置
var SystemConfig = &System{
	Debug:       false,
	Mode:        MasterMode,
	Listen:      ":5212",
	ProxyHeader: "X-Forwarded-For",
	LogLevel:    "info",
}

// CORSConfig 跨域配置
var CORSConfig = &Cors{
	AllowOrigins:     []string{"UNSET"},
	AllowMethods:     []string{"PUT", "POST", "GET", "OPTIONS"},
	AllowHeaders:     []string{"Cookie", "X-Cr-Policy", "Authorization", "Content-Length", "Content-Type", "X-Cr-Path", "X-Cr-FileName"},
	AllowCredentials: false,
	ExposeHeaders:    nil,
	SameSite:         "Default",
	Secure:           false,
}

// SlaveConfig 从机配置
var SlaveConfig = &Slave{
	CallbackTimeout: 20,
	SignatureTTL:    600,
}

var SSLConfig = &SSL{
	Listen:   ":443",
	CertPath: "",
	KeyPath:  "",
}

var UnixConfig = &Unix{
	Listen: "",
}

var OptionOverwrite = map[string]interface{}{}
