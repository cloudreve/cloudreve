package inventory

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/stretchr/testify/require"
)

func backupCodeUser(t *testing.T, client *ent.Client) *ent.User {
	t.Helper()
	group := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).SaveX(context.Background())
	return client.User.Create().
		SetEmail(fmt.Sprintf("%s@example.com", t.Name())).
		SetNick("u").
		SetGroup(group).
		SetTwoFactorSecret("SECRET").
		SaveX(context.Background())
}

func storeCodes(t *testing.T, c UserClient, u *ent.User, codes ...string) *ent.User {
	t.Helper()
	digests := make([]string, 0, len(codes))
	for _, code := range codes {
		d, err := DigestPassword(code)
		require.NoError(t, err)
		digests = append(digests, d)
	}
	u, err := c.UpdateTwoFABackupCodes(context.Background(), u, digests)
	require.NoError(t, err)
	return u
}

func TestTwoFABackupCodeConsume(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	defer client.Close()
	c := NewUserClient(client)
	ctx := context.Background()
	u := backupCodeUser(t, client)

	u = storeCodes(t, c, u, "abcd1234", "wxyz5678")
	require.Len(t, u.TwoFactorBackupCodes, 2)

	// Wrong code consumes nothing.
	ok, err := c.ConsumeTwoFABackupCode(ctx, u, "nope0000")
	require.NoError(t, err)
	require.False(t, ok)

	// Correct code consumes exactly once; separators and case are normalized.
	ok, err = c.ConsumeTwoFABackupCode(ctx, u, "ABCD-1234")
	require.NoError(t, err)
	require.True(t, ok)

	u = client.User.GetX(ctx, u.ID)
	require.Len(t, u.TwoFactorBackupCodes, 1)

	// Replay is rejected.
	ok, err = c.ConsumeTwoFABackupCode(ctx, u, "abcd1234")
	require.NoError(t, err)
	require.False(t, ok)

	// Sibling code still works.
	ok, err = c.ConsumeTwoFABackupCode(ctx, u, "wxyz5678")
	require.NoError(t, err)
	require.True(t, ok)

	u = client.User.GetX(ctx, u.ID)
	require.Empty(t, u.TwoFactorBackupCodes)
}

func TestTwoFABackupCodesClearedWithSecret(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	defer client.Close()
	c := NewUserClient(client)
	ctx := context.Background()
	u := backupCodeUser(t, client)

	u = storeCodes(t, c, u, "abcd1234")

	// Disabling 2FA invalidates outstanding recovery codes.
	u, err := c.UpdateTwoFASecret(ctx, u, "")
	require.NoError(t, err)
	require.Empty(t, u.TwoFactorBackupCodes)

	// Rotating the secret also invalidates them.
	u = storeCodes(t, c, u, "wxyz5678")
	u, err = c.UpdateTwoFASecret(ctx, u, "NEWSECRET")
	require.NoError(t, err)
	require.Equal(t, "NEWSECRET", u.TwoFactorSecret)
	require.Empty(t, u.TwoFactorBackupCodes)
}
