package inventory

import (
	"context"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/abusereport"
)

type (
	// AbuseReportClient persists abuse reports for the admin review queue.
	AbuseReportClient interface {
		TxOperator
		// Create inserts a new open report.
		Create(ctx context.Context, params *CreateAbuseReportParams) (*ent.AbuseReport, error)
		// List returns a page of reports plus the filtered total.
		List(ctx context.Context, args *ListAbuseReportArgs) (*ListAbuseReportResult, error)
		// SetStatus updates status/admin note of one report.
		SetStatus(ctx context.Context, id int, status, adminNote string) (*ent.AbuseReport, error)
	}

	CreateAbuseReportParams struct {
		ReporterID    int
		ReporterEmail string
		TargetType    string
		TargetID      int
		Reason        int
		Description   string
	}

	ListAbuseReportArgs struct {
		*PaginationArgs
		Status string
	}

	ListAbuseReportResult struct {
		Reports []*ent.AbuseReport
		Total   int
	}

	abuseReportClient struct {
		client *ent.Client
	}
)

// NewAbuseReportClient creates an AbuseReportClient.
func NewAbuseReportClient(client *ent.Client) AbuseReportClient {
	return &abuseReportClient{client: client}
}

func (c *abuseReportClient) SetClient(newClient *ent.Client) TxOperator {
	return &abuseReportClient{client: newClient}
}

func (c *abuseReportClient) GetClient() *ent.Client {
	return c.client
}

func (c *abuseReportClient) Create(ctx context.Context, params *CreateAbuseReportParams) (*ent.AbuseReport, error) {
	return c.client.AbuseReport.Create().
		SetReporterID(params.ReporterID).
		SetReporterEmail(params.ReporterEmail).
		SetTargetType(params.TargetType).
		SetTargetID(params.TargetID).
		SetReason(params.Reason).
		SetDescription(params.Description).
		Save(ctx)
}

func (c *abuseReportClient) List(ctx context.Context, args *ListAbuseReportArgs) (*ListAbuseReportResult, error) {
	q := c.client.AbuseReport.Query().Order(ent.Desc(abusereport.FieldCreatedAt))
	if args.Status != "" {
		q = q.Where(abusereport.StatusEQ(args.Status))
	}

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}

	reports, err := q.
		Offset(args.Page * args.PageSize).
		Limit(args.PageSize).
		All(ctx)
	if err != nil {
		return nil, err
	}

	return &ListAbuseReportResult{Reports: reports, Total: total}, nil
}

func (c *abuseReportClient) SetStatus(ctx context.Context, id int, status, adminNote string) (*ent.AbuseReport, error) {
	return c.client.AbuseReport.UpdateOneID(id).
		SetStatus(status).
		SetAdminNote(adminNote).
		Save(ctx)
}
