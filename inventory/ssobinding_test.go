package inventory

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/stretchr/testify/require"
)

func ssoBindingUser(t *testing.T, client *ent.Client, group *ent.Group, n int) *ent.User {
	t.Helper()
	return client.User.Create().
		SetEmail(fmt.Sprintf("%s-%d@example.com", t.Name(), n)).
		SetNick(fmt.Sprintf("u%d", n)).
		SetGroup(group).
		SaveX(context.Background())
}

func TestSsoBindingBindGetUnbind(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	defer client.Close()
	c := NewSsoBindingClient(client, conf.SQLiteDB)
	ctx := context.Background()

	group := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	u1 := ssoBindingUser(t, client, group, 1)
	u2 := ssoBindingUser(t, client, group, 2)

	b, err := c.Bind(ctx, u1.ID, SsoProviderQQ, "openid-a")
	require.NoError(t, err)
	require.Equal(t, u1.ID, b.UserID)
	require.Equal(t, "openid-a", b.Subject)

	// Lookup by (provider, subject) resolves the owner.
	got, err := c.Get(ctx, SsoProviderQQ, "openid-a")
	require.NoError(t, err)
	require.Equal(t, b.ID, got.ID)

	// ListByUser returns only the caller's bindings.
	_, err = c.Bind(ctx, u2.ID, SsoProviderQQ, "openid-b")
	require.NoError(t, err)
	mine, err := c.ListByUser(ctx, u1.ID)
	require.NoError(t, err)
	require.Len(t, mine, 1)
	require.Equal(t, "openid-a", mine[0].Subject)

	// Re-binding the same (user, provider) moves the row to the new subject.
	moved, err := c.Bind(ctx, u1.ID, SsoProviderQQ, "openid-c")
	require.NoError(t, err)
	require.Equal(t, b.ID, moved.ID)
	require.Equal(t, "openid-c", moved.Subject)
	mine, err = c.ListByUser(ctx, u1.ID)
	require.NoError(t, err)
	require.Len(t, mine, 1)

	// A subject owned by a different user conflicts.
	_, err = c.Bind(ctx, u2.ID, SsoProviderQQ, "openid-c")
	require.ErrorIs(t, err, ErrSsoBindingConflict)

	// Re-binding the same subject to its owner is a no-op.
	again, err := c.Bind(ctx, u1.ID, SsoProviderQQ, "openid-c")
	require.NoError(t, err)
	require.Equal(t, moved.ID, again.ID)

	// Unbind removes only the caller's row and is idempotent.
	require.NoError(t, c.Unbind(ctx, u1.ID, SsoProviderQQ))
	_, err = c.Get(ctx, SsoProviderQQ, "openid-c")
	require.Error(t, err)
	require.NoError(t, c.Unbind(ctx, u1.ID, SsoProviderQQ))

	// User 2's binding is untouched.
	still, err := c.Get(ctx, SsoProviderQQ, "openid-b")
	require.NoError(t, err)
	require.Equal(t, u2.ID, still.UserID)
}
