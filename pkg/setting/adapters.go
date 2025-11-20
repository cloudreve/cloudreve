package setting

import (
	"context"
	"os"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/samber/lo"
)

const (
	KvSettingPrefix           = "setting_"
	EnvSettingOverwritePrefix = "CR_SETTING_"
)

// SettingStoreAdapter chains a setting get operation, if current adapter cannot locate setting value,
// it will invoke next adapter until last one.
type SettingStoreAdapter interface {
	// Get a string setting from underlying store, if setting not found, default
	// value will be used.
	Get(ctx context.Context, name string, defaultVal any) any
}

// NewDbSettingStore creates an adapter using DB setting store. Only string type value is supported.
func NewDbSettingStore(c inventory.SettingClient, next SettingStoreAdapter) SettingStoreAdapter {
	return &dbSettingStore{
		c:    c,
		next: next,
	}
}

// NewKvSettingStore creates an adapter using KV setting store.
func NewKvSettingStore(c cache.Driver, next SettingStoreAdapter) SettingStoreAdapter {
	return &kvSettingStore{
		kv:   c,
		next: next,
	}
}

// NewConfSettingStore creates an adapter using static config overwrite. Only string type value is supported.
func NewConfSettingStore(c conf.ConfigProvider, next SettingStoreAdapter) SettingStoreAdapter {
	return &staticSettingStore{
		settings: c.OptionOverwrite(),
		next:     next,
	}
}

// NewDbDefaultStore creates an adapter that always returns default setting value defined in inventory.DefaultSettings.
// Only string type value is supported.
func NewDbDefaultStore(next SettingStoreAdapter) SettingStoreAdapter {
	return &staticSettingStore{
		settings: lo.MapValues(inventory.DefaultSettings, func(v string, _ string) any {
			return v
		}),
		next: next,
	}
}

// NewEnvOverwriteStore creates an adapter that always returns overrided setting value defined in environment variables.
func NewEnvOverrideStore(next SettingStoreAdapter, l logging.Logger) SettingStoreAdapter {
	allEnv := os.Environ()
	defaults := make(map[string]any)
	for _, env := range allEnv {
		kv := strings.SplitN(env, "=", 2)
		if strings.HasPrefix(kv[0], EnvSettingOverwritePrefix) {
			key := strings.TrimPrefix(kv[0], EnvSettingOverwritePrefix)
			defaults[key] = kv[1]
			l.Info("Override setting %q with value %q from environment", key, kv[1])
		}
	}

	return &staticSettingStore{
		settings: defaults,
		next:     next,
	}
}

type dbSettingStore struct {
	c    inventory.SettingClient
	next SettingStoreAdapter
}

func (s *dbSettingStore) Get(ctx context.Context, name string, defaultVal any) any {
	if val, err := s.c.Get(ctx, name); err == nil {
		return val
	}

	if s.next != nil {
		return s.next.Get(ctx, name, defaultVal)
	}

	return defaultVal
}

type kvSettingStore struct {
	kv   cache.Driver
	next SettingStoreAdapter
}

func (s *kvSettingStore) Get(ctx context.Context, name string, defaultVal any) any {
	if val, exist := s.kv.Get(KvSettingPrefix + name); exist {
		return val
	}

	if s.next != nil {
		nextVal := s.next.Get(ctx, name, defaultVal)
		// cache setting value
		s.kv.Set(KvSettingPrefix+name, nextVal, 0)
		return nextVal
	}

	return defaultVal
}

type staticSettingStore struct {
	settings map[string]any
	next     SettingStoreAdapter
}

func (s *staticSettingStore) Get(ctx context.Context, name string, defaultVal any) any {
	if val, ok := s.settings[name]; ok {
		return val
	}

	if s.next != nil {
		return s.next.Get(ctx, name, defaultVal)
	}

	return defaultVal
}
