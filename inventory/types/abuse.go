package types

// Abuse report target and status constants.
const (
	AbuseTargetShare = "share"
	AbuseTargetUser  = "user"

	AbuseStatusOpen      = "open"
	AbuseStatusResolved  = "resolved"
	AbuseStatusDismissed = "dismissed"
)

// Abuse reasons, matching the frontend reportReasonOptions order.
const (
	AbuseReasonCopyright = iota
	AbuseReasonHarmful
	AbuseReasonSpam
	AbuseReasonOther

	AbuseReasonMax = AbuseReasonOther
)
