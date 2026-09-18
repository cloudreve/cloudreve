package conf

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/stretchr/testify/require"
)

func TestEnvOverrideDatabaseName(t *testing.T) {
	dir := t.TempDir()
	confPath := filepath.Join(dir, "conf.ini")
	require.NoError(t, os.WriteFile(confPath, []byte("[System]\nDebug = false\nMode = master\nListen = :5212\nSessionSecret = s\nHashIDSalt = s\n"), 0644))

	t.Setenv("CR_CONF_Database.Type", "postgres")
	t.Setenv("CR_CONF_Database.Host", "postgresql")
	t.Setenv("CR_CONF_Database.User", "user")
	t.Setenv("CR_CONF_Database.Name", "database")
	t.Setenv("CR_CONF_Database.Port", "5432")

	p, err := NewIniConfigProvider(confPath, logging.NewConsoleLogger(logging.LevelDebug))
	require.NoError(t, err)
	require.Equal(t, "user", p.Database().User)
	require.Equal(t, "database", p.Database().Name)
}
