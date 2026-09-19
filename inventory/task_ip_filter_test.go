package inventory

import (
	"context"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"
)

func TestListTaskCreatorIPFilter(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	mk := func(ip string) {
		client.Task.Create().
			SetType("test").
			SetPublicState(&types.TaskPublicState{}).
			SetCorrelationID(uuid.Must(uuid.NewV4())).
			SetCreatorIP(ip).
			SaveX(ctx)
	}
	mk("192.168.1.10")
	mk("192.168.1.20")
	mk("10.0.0.5")
	mk("2001:db8::1")

	hasher, err := hashid.New("test")
	require.NoError(t, err)
	tc := NewTaskClient(client, conf.SQLite3DB, hasher)

	list := func(filter string) []string {
		res, err := tc.List(ctx, &ListTaskArgs{
			PaginationArgs: &PaginationArgs{Page: 0, PageSize: 50},
			CreatorIP:      filter,
		})
		require.NoError(t, err)
		ips := make([]string, 0, len(res.Tasks))
		for _, task := range res.Tasks {
			ips = append(ips, task.CreatorIP)
		}
		return ips
	}

	// CIDR prefix matches only in-range addresses.
	require.ElementsMatch(t, []string{"192.168.1.10", "192.168.1.20"}, list("192.168.1.0/24"))
	require.ElementsMatch(t, []string{"2001:db8::1"}, list("2001:db8::/64"))

	// Exact IP normalizes and matches.
	require.ElementsMatch(t, []string{"10.0.0.5"}, list("10.0.0.5"))

	// Non-IP input falls back to substring matching.
	require.ElementsMatch(t, []string{"192.168.1.10", "192.168.1.20"}, list("192.168"))
	require.Empty(t, list("192.168.1.0/30"))
}
