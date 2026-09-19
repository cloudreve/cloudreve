package user

import (
	"context"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/pkg/crontab"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
)

func init() {
	crontab.Register(setting.CronTypeGrantExpire, func(ctx context.Context) {
		dep := dependency.FromContext(ctx)
		if err := dep.VasClient().ExpireGrants(ctx, dep.SettingProvider().DefaultGroup(ctx)); err != nil {
			dep.Logger().Error("Failed to expire user grants: %s", err)
		}
	})
}
