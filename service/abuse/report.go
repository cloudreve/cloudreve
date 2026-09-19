package abuse

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/activity"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
)

type (
	// ReportService accepts an abuse report from any visitor. CAPTCHA and
	// rate limiting are enforced by route middleware.
	ReportService struct {
		TargetType  string `json:"target_type" binding:"required,oneof=share user"`
		Target      string `json:"target" binding:"required"`
		Reason      int    `json:"reason" binding:"min=0"`
		Description string `json:"description" binding:"max=2000"`
	}
	ReportParamCtx struct{}
)

// Create validates the target, stores the report, and emits report_abuse.
func (s *ReportService) Create(c *gin.Context) error {
	dep := dependency.FromContext(c)
	if s.Reason > types.AbuseReasonMax {
		return serializer.NewError(serializer.CodeParamErr, "Invalid reason", nil)
	}

	params := &inventory.CreateAbuseReportParams{
		TargetType:  s.TargetType,
		Reason:      s.Reason,
		Description: s.Description,
	}
	if u := inventory.UserFromContext(c); u != nil && !inventory.IsAnonymousUser(u) {
		params.ReporterID = u.ID
		params.ReporterEmail = u.Email
	}

	var opts []activity.Opt
	switch s.TargetType {
	case types.AbuseTargetShare:
		shareID, err := dep.HashIDEncoder().Decode(s.Target, hashid.ShareID)
		if err != nil {
			return serializer.NewError(serializer.CodeParamErr, "Invalid share", err)
		}
		if _, err := dep.ShareClient().GetByID(c, shareID); err != nil {
			return serializer.NewError(serializer.CodeNotFound, "Share not found", err)
		}
		params.TargetID = shareID
		opts = append(opts, activity.Share(shareID))
	case types.AbuseTargetUser:
		uid, err := dep.HashIDEncoder().Decode(s.Target, hashid.UserID)
		if err != nil {
			return serializer.NewError(serializer.CodeParamErr, "Invalid user", err)
		}
		if _, err := dep.UserClient().GetByID(c, uid); err != nil {
			return serializer.NewError(serializer.CodeNotFound, "User not found", err)
		}
		params.TargetID = uid
	}

	report, err := dep.AbuseReportClient().Create(c, params)
	if err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to submit report", err)
	}

	opts = append(opts, activity.Extra(map[string]any{
		"report_id":   report.ID,
		"target_type": s.TargetType,
		"reason":      s.Reason,
	}))
	activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventReportAbuse, opts...)

	return nil
}
