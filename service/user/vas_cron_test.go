package user

import (
	"context"
	"testing"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent/activityevent"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/ent/usergrant"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/stretchr/testify/require"
)

type cronSettingProvider struct {
	setting.Provider
	defaultGroup int
}

func (p cronSettingProvider) DefaultGroup(context.Context) int {
	return p.defaultGroup
}

func (p cronSettingProvider) AuditLogEnabled(context.Context, int) bool {
	return true
}

func TestGrantExpireCronRecordsUnsubscribe(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	logger := logging.NewConsoleLogger(logging.LevelError)
	cfg, err := conf.NewIniConfigProvider(t.TempDir()+"/conf.ini", logger)
	require.NoError(t, err)

	base := client.Group.Create().SetName("base").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	vip := client.Group.Create().SetName("vip").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	u := client.User.Create().SetEmail("c@example.com").SetNick("c").SetGroup(vip).SaveX(ctx)

	// Expired group grant, user still on the granted group -> reverted.
	client.UserGrant.Create().SetUserID(u.ID).SetType(usergrant.TypeGroup).
		SetAmount(int64(vip.ID)).SetPrevGroupID(base.ID).
		SetExpiresAt(time.Now().Add(-time.Minute)).SaveX(ctx)
	// Expired grant for a user who moved on independently -> not reverted,
	// no event.
	u2 := client.User.Create().SetEmail("c2@example.com").SetNick("c2").SetGroup(base).SaveX(ctx)
	client.UserGrant.Create().SetUserID(u2.ID).SetType(usergrant.TypeGroup).
		SetAmount(int64(vip.ID)).SetPrevGroupID(base.ID).
		SetExpiresAt(time.Now().Add(-time.Minute)).SaveX(ctx)

	dep := dependency.NewDependency(
		dependency.WithDbClient(client),
		dependency.WithConfigProvider(cfg),
		dependency.WithLogger(logger),
		dependency.WithSettingProvider(cronSettingProvider{defaultGroup: base.ID}),
	)
	ctx = context.WithValue(ctx, dependency.DepCtx{}, dep)

	reverted, err := dep.VasClient().ExpireGrants(ctx, base.ID)
	require.NoError(t, err)
	require.Len(t, reverted, 1)
	require.Equal(t, u.ID, reverted[0].UserID)
	require.Equal(t, base.ID, client.User.GetX(ctx, u.ID).GroupUsers)
	require.Equal(t, base.ID, client.User.GetX(ctx, u2.ID).GroupUsers)

	recordMembershipUnsubscribes(ctx, dep, reverted, base.ID)

	ev := client.ActivityEvent.Query().
		Where(activityevent.TypeEQ(types.EventMembershipUnsubscribe)).OnlyX(ctx)
	require.Equal(t, u.ID, ev.ActorID)
	require.Equal(t, float64(vip.ID), ev.Extra["from_group"])
	require.Equal(t, float64(base.ID), ev.Extra["to_group"])
}
