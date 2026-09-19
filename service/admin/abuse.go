package admin

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
)

type (
	// AbuseListService pages the abuse report queue.
	AbuseListService struct {
		Page     int    `form:"page" binding:"required,min=0"`
		PageSize int    `form:"page_size" binding:"required,min=1,max=100"`
		Status   string `form:"status" binding:"omitempty,oneof=open resolved dismissed"`
	}
	AbuseListParamCtx struct{}

	// AbuseReport is one queue entry.
	AbuseReport struct {
		ID            string `json:"id"`
		ReporterID    string `json:"reporter_id,omitempty"`
		ReporterEmail string `json:"reporter_email,omitempty"`
		TargetType    string `json:"target_type"`
		TargetID      string `json:"target_id"`
		Reason        int    `json:"reason"`
		Description   string `json:"description"`
		Status        string `json:"status"`
		AdminNote     string `json:"admin_note"`
		CreatedAt     int64  `json:"created_at"`
	}

	AbuseListResponse struct {
		Reports []*AbuseReport `json:"reports"`
		Total   int            `json:"total"`
	}

	// AbuseUpdateService updates one report; block_share additionally expires
	// the reported share.
	AbuseUpdateService struct {
		Status     string `json:"status" binding:"required,oneof=open resolved dismissed"`
		AdminNote  string `json:"admin_note" binding:"max=2000"`
		BlockShare bool   `json:"block_share"`
	}
	AbuseUpdateParamCtx struct{}
)

func (s *AbuseListService) List(c *gin.Context) (*AbuseListResponse, error) {
	dep := dependency.FromContext(c)
	res, err := dep.AbuseReportClient().List(c, &inventory.ListAbuseReportArgs{
		PaginationArgs: &inventory.PaginationArgs{
			UseCursorPagination: false,
			Page:                s.Page,
			PageSize:            s.PageSize,
		},
		Status: s.Status,
	})
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list reports", err)
	}

	return BuildAbuseListResponse(res, dep.HashIDEncoder()), nil
}

func BuildAbuseListResponse(res *inventory.ListAbuseReportResult, hasher hashid.Encoder) *AbuseListResponse {
	reports := make([]*AbuseReport, 0, len(res.Reports))
	for _, r := range res.Reports {
		reports = append(reports, BuildAbuseReport(r, hasher))
	}
	return &AbuseListResponse{Reports: reports, Total: res.Total}
}

func BuildAbuseReport(r *ent.AbuseReport, hasher hashid.Encoder) *AbuseReport {
	res := &AbuseReport{
		ID:            hashid.EncodeAuditLogID(hasher, r.ID),
		ReporterEmail: r.ReporterEmail,
		TargetType:    r.TargetType,
		Reason:        r.Reason,
		Description:   r.Description,
		Status:        r.Status,
		AdminNote:     r.AdminNote,
		CreatedAt:     r.CreatedAt.Unix(),
	}
	if r.ReporterID > 0 {
		res.ReporterID = hashid.EncodeUserID(hasher, r.ReporterID)
	}
	if r.TargetType == types.AbuseTargetShare {
		res.TargetID = hashid.EncodeShareID(hasher, r.TargetID)
	} else {
		res.TargetID = hashid.EncodeUserID(hasher, r.TargetID)
	}
	return res
}

func (s *AbuseUpdateService) Update(c *gin.Context) (*AbuseReport, error) {
	dep := dependency.FromContext(c)
	reportID := hashid.FromContext(c)

	report, err := dep.AbuseReportClient().SetStatus(c, reportID, s.Status, s.AdminNote)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to update report", err)
	}

	if s.BlockShare && report.TargetType == types.AbuseTargetShare {
		if err := dep.ShareClient().Expire(c, report.TargetID); err != nil {
			return nil, serializer.NewError(serializer.CodeDBError, "Failed to block share", err)
		}
	}

	return BuildAbuseReport(report, dep.HashIDEncoder()), nil
}
