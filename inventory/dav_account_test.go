package inventory

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/stretchr/testify/require"
)

func TestUpdateURIPrefix(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	group := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	owner := client.User.Create().SetEmail("dav@example.com").SetNick("dav").SetGroup(group).SaveX(ctx)
	other := client.User.Create().SetEmail("other@example.com").SetNick("other").SetGroup(group).SaveX(ctx)

	seq := 0
	mk := func(userID int, uri string) {
		seq++
		client.DavAccount.Create().
			SetName("mount").
			SetURI(uri).
			SetPassword(fmt.Sprintf("pw-%d", seq)).
			SetOptions(&boolset.BooleanSet{}).
			SetOwnerID(userID).
			SaveX(ctx)
	}
	getURIs := func(userID int) []string {
		accounts, err := client.DavAccount.Query().All(ctx)
		require.NoError(t, err)
		var uris []string
		for _, a := range accounts {
			if a.OwnerID == userID {
				uris = append(uris, a.URI)
			}
		}
		return uris
	}

	mk(owner.ID, "my://hash/dir")
	mk(owner.ID, "my://hash/dir/sub")
	mk(owner.ID, "my://hash/dir-other")
	mk(other.ID, "my://hash/dir")

	c := NewDavAccountClient(client, "sqlite3", nil)
	require.NoError(t, c.UpdateURIPrefix(ctx, owner.ID, "my://hash/dir", "my://hash/moved/dir"))

	got := getURIs(owner.ID)
	require.ElementsMatch(t, []string{"my://hash/moved/dir", "my://hash/moved/dir/sub", "my://hash/dir-other"}, got)
	// Other user's account untouched.
	require.Equal(t, []string{"my://hash/dir"}, getURIs(other.ID))

	// Trailing-slash stored URI is normalized and rewritten.
	mk(owner.ID, "my://hash/base/")
	require.NoError(t, c.UpdateURIPrefix(ctx, owner.ID, "my://hash/base/", "my://hash/base2"))
	require.Contains(t, getURIs(owner.ID), "my://hash/base2")

	// Same-URI no-op and empty results are clean.
	require.NoError(t, c.UpdateURIPrefix(ctx, owner.ID, "my://hash/x", "my://hash/x"))
	require.NoError(t, c.UpdateURIPrefix(ctx, owner.ID, "my://hash/missing", "my://hash/y"))
	require.NoError(t, c.UpdateURIPrefix(ctx, 0, "my://hash/dir", "my://hash/z"))
}
