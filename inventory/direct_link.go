package inventory

import (
	"context"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/directlink"
	"github.com/cloudreve/Cloudreve/v4/ent/schema"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
)

type (
	DirectLinkClient interface {
		TxOperator
		// GetByNameID get direct link by name and id
		GetByNameID(ctx context.Context, id int, name string) (*ent.DirectLink, error)
		// GetByID get direct link by id
		GetByID(ctx context.Context, id int) (*ent.DirectLink, error)
		// Delete delete direct link by id
		Delete(ctx context.Context, id int) error
	}
	LoadDirectLinkFile struct{}
)

func NewDirectLinkClient(client *ent.Client, dbType conf.DBType, hasher hashid.Encoder) DirectLinkClient {
	return &directLinkClient{
		client:      client,
		hasher:      hasher,
		maxSQlParam: sqlParamLimit(dbType),
	}
}

type directLinkClient struct {
	maxSQlParam int
	client      *ent.Client
	hasher      hashid.Encoder
}

func (c *directLinkClient) SetClient(newClient *ent.Client) TxOperator {
	return &directLinkClient{client: newClient, hasher: c.hasher, maxSQlParam: c.maxSQlParam}
}

func (c *directLinkClient) GetClient() *ent.Client {
	return c.client
}

func (d *directLinkClient) GetByID(ctx context.Context, id int) (*ent.DirectLink, error) {
	return withDirectLinkEagerLoading(ctx, d.client.DirectLink.Query().Where(directlink.ID(id))).
		First(ctx)
}

func (d *directLinkClient) GetByNameID(ctx context.Context, id int, name string) (*ent.DirectLink, error) {
	res, err := withDirectLinkEagerLoading(ctx, d.client.DirectLink.Query().Where(directlink.ID(id), directlink.Name(name))).
		First(ctx)
	if err != nil {
		return nil, err
	}

	// Increase download counter
	_, _ = d.client.DirectLink.Update().Where(directlink.ID(res.ID)).SetDownloads(res.Downloads + 1).Save(ctx)

	return res, nil
}

func (d *directLinkClient) Delete(ctx context.Context, id int) error {
	ctx = schema.SkipSoftDelete(ctx)
	_, err := d.client.DirectLink.Delete().Where(directlink.ID(id)).Exec(ctx)
	return err
}

func withDirectLinkEagerLoading(ctx context.Context, q *ent.DirectLinkQuery) *ent.DirectLinkQuery {
	if v, ok := ctx.Value(LoadDirectLinkFile{}).(bool); ok && v {
		q.WithFile(func(m *ent.FileQuery) {
			withFileEagerLoading(ctx, m)
		})
	}
	return q
}
