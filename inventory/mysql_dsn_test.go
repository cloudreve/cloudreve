package inventory

import (
	"testing"

	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/stretchr/testify/require"
)

func TestEnsureMySQLParseTime(t *testing.T) {
	dsn := "user:pass@tcp(127.0.0.1:3306)/cloudreve"
	require.Equal(t, dsn+"?parseTime=True", ensureMySQLParseTime(conf.MySqlDB, dsn))

	withParams := dsn + "?charset=utf8mb4"
	require.Equal(t, withParams+"&parseTime=True", ensureMySQLParseTime(conf.MySqlDB, withParams))

	existing := dsn + "?charset=utf8mb4&parseTime=false"
	require.Equal(t, existing, ensureMySQLParseTime(conf.MySqlDB, existing))

	urlForm := "mysql://user:pass@host:3306/cloudreve?tls=true"
	require.Equal(t, urlForm+"&parseTime=True", ensureMySQLParseTime(conf.MySqlDB, urlForm))

	// Non-MySQL DSNs pass through untouched.
	pg := "postgresql://user:pass@host:5432/cloudreve?sslmode=require"
	require.Equal(t, pg, ensureMySQLParseTime(conf.PostgresDB, pg))
	require.Equal(t, dsn, ensureMySQLParseTime(conf.SQLiteDB, dsn))
}
