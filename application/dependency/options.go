package dependency

import (
	"io/fs"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/email"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/gin-contrib/static"
)

// Option 发送请求的额外设置
type Option interface {
	apply(*dependency)
}

type optionFunc func(*dependency)

func (f optionFunc) apply(o *dependency) {
	f(o)
}

// WithConfigPath Set the path of the config file.
func WithConfigPath(p string) Option {
	return optionFunc(func(o *dependency) {
		o.configPath = p
	})
}

// WithLogger Set the default logging.
func WithLogger(l logging.Logger) Option {
	return optionFunc(func(o *dependency) {
		o.logger = l
	})
}

// WithConfigProvider Set the default config provider.
func WithConfigProvider(c conf.ConfigProvider) Option {
	return optionFunc(func(o *dependency) {
		o.configProvider = c
	})
}

// WithStatics Set the default statics FS.
func WithStatics(c fs.FS) Option {
	return optionFunc(func(o *dependency) {
		o.statics = c
	})
}

// WithServerStaticFS Set the default statics FS for server.
func WithServerStaticFS(c static.ServeFileSystem) Option {
	return optionFunc(func(o *dependency) {
		o.serverStaticFS = c
	})
}

// WithProFlag Set if current instance is a pro version.
func WithProFlag(c bool) Option {
	return optionFunc(func(o *dependency) {
		o.isPro = c
	})
}

// WithRawEntClient Set the default raw ent client.
func WithRawEntClient(c *ent.Client) Option {
	return optionFunc(func(o *dependency) {
		o.rawEntClient = c
	})
}

// WithDbClient Set the default ent client.
func WithDbClient(c *ent.Client) Option {
	return optionFunc(func(o *dependency) {
		o.dbClient = c
	})
}

// WithRequiredDbVersion Set the required db version.
func WithRequiredDbVersion(c string) Option {
	return optionFunc(func(o *dependency) {
		o.requiredDbVersion = c
	})
}

// WithKV Set the default KV store driverold
func WithKV(c cache.Driver) Option {
	return optionFunc(func(o *dependency) {
		o.kv = c
	})
}

// WithSettingClient Set the default setting client
func WithSettingClient(s inventory.SettingClient) Option {
	return optionFunc(func(o *dependency) {
		o.settingClient = s
	})
}

// WithSettingProvider Set the default setting provider
func WithSettingProvider(s setting.Provider) Option {
	return optionFunc(func(o *dependency) {
		o.settingProvider = s
	})
}

// WithUserClient Set the default user client
func WithUserClient(s inventory.UserClient) Option {
	return optionFunc(func(o *dependency) {
		o.userClient = s
	})
}

// WithEmailClient Set the default email client
func WithEmailClient(s email.Driver) Option {
	return optionFunc(func(o *dependency) {
		o.emailClient = s
	})
}

// WithGeneralAuth Set the default general auth
func WithGeneralAuth(s auth.Auth) Option {
	return optionFunc(func(o *dependency) {
		o.generalAuth = s
	})
}

// WithHashIDEncoder Set the default hash id encoder
func WithHashIDEncoder(s hashid.Encoder) Option {
	return optionFunc(func(o *dependency) {
		o.hashidEncoder = s
	})
}

// WithTokenAuth Set the default token auth
func WithTokenAuth(s auth.TokenAuth) Option {
	return optionFunc(func(o *dependency) {
		o.tokenAuth = s
	})
}

// WithFileClient Set the default file client
func WithFileClient(s inventory.FileClient) Option {
	return optionFunc(func(o *dependency) {
		o.fileClient = s
	})
}

// WithShareClient Set the default share client
func WithShareClient(s inventory.ShareClient) Option {
	return optionFunc(func(o *dependency) {
		o.shareClient = s
	})
}
