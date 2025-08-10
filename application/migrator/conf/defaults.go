package conf

// RedisConfig Redis服务器配置
var RedisConfig = &redis{
	Network:  "tcp",
	Server:   "",
	Password: "",
	DB:       "0",
}

// DatabaseConfig 数据库配置
var DatabaseConfig = &database{
	Type:       "UNSET",
	Charset:    "utf8",
	DBFile:     "cloudreve.db",
	Port:       3306,
	UnixSocket: false,
}

// SystemConfig 系统公用配置
var SystemConfig = &system{
	Debug:       false,
	Mode:        "master",
	Listen:      ":5212",
	ProxyHeader: "",
}

// CORSConfig 跨域配置
var CORSConfig = &cors{
	AllowOrigins:     []string{"UNSET"},
	AllowMethods:     []string{"PUT", "POST", "GET", "OPTIONS"},
	AllowHeaders:     []string{"Cookie", "X-Cr-Policy", "Authorization", "Content-Length", "Content-Type", "X-Cr-Path", "X-Cr-FileName"},
	AllowCredentials: false,
	ExposeHeaders:    nil,
	SameSite:         "Default",
	Secure:           false,
}

// SlaveConfig 从机配置
var SlaveConfig = &slave{
	CallbackTimeout: 20,
	SignatureTTL:    60,
}

var SSLConfig = &ssl{
	Listen:   ":443",
	CertPath: "",
	KeyPath:  "",
}

var UnixConfig = &unix{
	Listen: "",
}

var OptionOverwrite = map[string]interface{}{}
