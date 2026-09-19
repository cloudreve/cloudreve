package inventory

import (
	"context"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/activityevent"
	"github.com/cloudreve/Cloudreve/v4/ent/schema"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
)

type (
	ActivityClient interface {
		TxOperator
		// Record appends one audit event.
		Record(ctx context.Context, params *RecordActivityParams) (*ent.ActivityEvent, error)
		// List returns a page of events matching the filter plus the total
		// count. Results are newest-first.
		List(ctx context.Context, args *ListActivityArgs) ([]*ent.ActivityEvent, int, error)
		// DeleteBefore removes events older than the given timestamp.
		// Returns the number of rows removed.
		DeleteBefore(ctx context.Context, before time.Time) (int, error)
	}

	RecordActivityParams struct {
		Type    int
		ActorID int
		IP      string
		CID     string
		FileID  int
		ShareID int
		Extra   map[string]any
	}

	ListActivityArgs struct {
		PaginationArgs
		Type    *int
		FileID  int
		ActorID int
		ShareID int
	}
)

func NewActivityClient(client *ent.Client, dbType conf.DBType) ActivityClient {
	return &activityClient{
		client:      client,
		maxSQlParam: sqlParamLimit(dbType),
	}
}

type activityClient struct {
	maxSQlParam int
	client      *ent.Client
}

func (c *activityClient) SetClient(newClient *ent.Client) TxOperator {
	return &activityClient{client: newClient, maxSQlParam: c.maxSQlParam}
}

func (c *activityClient) GetClient() *ent.Client {
	return c.client
}

func (c *activityClient) Record(ctx context.Context, params *RecordActivityParams) (*ent.ActivityEvent, error) {
	create := c.client.ActivityEvent.Create().
		SetType(params.Type).
		SetActorID(params.ActorID).
		SetIP(params.IP).
		SetCid(params.CID).
		SetExtra(params.Extra)
	if params.FileID > 0 {
		create.SetFileID(params.FileID)
	}
	if params.ShareID > 0 {
		create.SetShareID(params.ShareID)
	}
	return create.Save(ctx)
}

func (c *activityClient) List(ctx context.Context, args *ListActivityArgs) ([]*ent.ActivityEvent, int, error) {
	query := c.client.ActivityEvent.Query()
	if args.Type != nil {
		query = query.Where(activityevent.TypeEQ(*args.Type))
	}
	if args.FileID > 0 {
		query = query.Where(activityevent.FileIDEQ(args.FileID))
	}
	if args.ActorID > 0 {
		query = query.Where(activityevent.ActorIDEQ(args.ActorID))
	}
	if args.ShareID > 0 {
		query = query.Where(activityevent.ShareIDEQ(args.ShareID))
	}

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	events, err := query.
		Order(ent.Desc(activityevent.FieldCreatedAt)).
		Limit(args.PageSize).
		Offset(args.Page * args.PageSize).
		All(ctx)
	return events, total, err
}

func (c *activityClient) DeleteBefore(ctx context.Context, before time.Time) (int, error) {
	return c.client.ActivityEvent.Delete().
		Where(activityevent.CreatedAtLT(before)).
		Exec(schema.SkipSoftDelete(ctx))
}
