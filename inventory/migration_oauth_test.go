package inventory

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/ent/oauthclient"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/stretchr/testify/require"
)

func TestBuiltinCLIOAuthMigration(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", filepath.Join(t.TempDir(), "migration.db"))
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	logger := logging.NewConsoleLogger(logging.LevelError)

	require.NoError(t, migrateOAuthClient(logger, client, ctx))
	desktop, err := client.OAuthClient.Query().Where(oauthclient.GUID(OAuthClientDesktopGUID)).Only(ctx)
	require.NoError(t, err)
	cli, err := client.OAuthClient.Query().Where(oauthclient.GUID(OAuthClientCLIGUID)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "Cloudreve CLI", cli.Name)
	require.Empty(t, cli.Secret)
	require.Equal(t, []string{"http://127.0.0.1/callback"}, cli.RedirectUris)
	require.Equal(t, desktop.Scopes, cli.Scopes)
	require.Equal(t, int64(7776000), cli.Props.RefreshTokenTTL)
	require.True(t, cli.IsEnabled)

	_, err = client.OAuthClient.UpdateOne(cli).SetIsEnabled(false).SetName("Disabled by administrator").Save(ctx)
	require.NoError(t, err)
	require.NoError(t, migrateOAuthClient(logger, client, ctx))
	preserved, err := client.OAuthClient.Get(ctx, cli.ID)
	require.NoError(t, err)
	require.False(t, preserved.IsEnabled)
	require.Equal(t, "Disabled by administrator", preserved.Name)
	count, err := client.OAuthClient.Query().Where(oauthclient.GUID(OAuthClientCLIGUID)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}
