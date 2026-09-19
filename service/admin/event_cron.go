package admin

import (
	"context"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/pkg/crontab"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
)

func init() {
	// audit_cleanup removes events older than `audit_log_retention_days`.
	// 0/unset keeps events forever.
	crontab.Register(setting.CronTypeAuditCleanup, func(ctx context.Context) {
		dep := dependency.FromContext(ctx)
		days := dep.SettingProvider().AuditLogRetentionDays(ctx)
		if days <= 0 {
			return
		}
		if _, err := dep.ActivityClient().DeleteBefore(ctx, time.Now().AddDate(0, 0, -days)); err != nil {
			dep.Logger().Error("Failed to clean up audit events: %s", err)
		}
	})
}
