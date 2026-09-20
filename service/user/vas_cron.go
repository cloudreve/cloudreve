package user

import (
	"context"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/activity"
	"github.com/cloudreve/Cloudreve/v4/pkg/crontab"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
)

func init() {
	crontab.Register(setting.CronTypeGrantExpire, func(ctx context.Context) {
		dep := dependency.FromContext(ctx)
		defaultGroup := dep.SettingProvider().DefaultGroup(ctx)
		reverted, err := dep.VasClient().ExpireGrants(ctx, defaultGroup)
		if err != nil {
			dep.Logger().Error("Failed to expire user grants: %s", err)
			return
		}

		recordMembershipUnsubscribes(ctx, dep, reverted, defaultGroup)

		expired, err := dep.UserClient().ExpireMemberships(ctx)
		if err != nil {
			dep.Logger().Error("Failed to expire group memberships: %s", err)
			return
		}
		for _, m := range expired {
			activity.Record(ctx, dep.SettingProvider(), dep.ActivityClient(), types.EventMembershipUnsubscribe,
				activity.Actor(m.UserID), activity.Extra(map[string]any{
					"from_group": m.GroupID, "to_group": 0,
				}))
		}
	})
}

// recordMembershipUnsubscribes emits an unsubscribe event for each grant
// whose group assignment was reverted by ExpireGrants.
func recordMembershipUnsubscribes(ctx context.Context, dep dependency.Dep, reverted []*ent.UserGrant, defaultGroup int) {
	for _, g := range reverted {
		revertTo := g.PrevGroupID
		if revertTo <= 0 {
			revertTo = defaultGroup
		}
		activity.Record(ctx, dep.SettingProvider(), dep.ActivityClient(), types.EventMembershipUnsubscribe,
			activity.Actor(g.UserID), activity.Extra(map[string]any{
				"from_group": g.Amount, "to_group": revertTo,
			}))
	}
}
