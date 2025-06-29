package inventory

import (
	"context"
	"fmt"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/group"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
)

type (
	// Ctx keys for eager loading options.
	LoadGroupPolicy struct{}
)

const (
	AnonymousGroupID = 3
)

type (
	GroupClient interface {
		TxOperator
		// AnonymousGroup returns the anonymous group.
		AnonymousGroup(ctx context.Context) (*ent.Group, error)
		// ListAll returns all groups.
		ListAll(ctx context.Context) ([]*ent.Group, error)
		// GetByID returns the group by id.
		GetByID(ctx context.Context, id int) (*ent.Group, error)
		// ListGroups returns a list of groups with pagination.
		ListGroups(ctx context.Context, args *ListGroupParameters) (*ListGroupResult, error)
		// CountUsers returns the number of users in the group.
		CountUsers(ctx context.Context, id int) (int, error)
		// Upsert upserts a group.
		Upsert(ctx context.Context, group *ent.Group) (*ent.Group, error)
		// Delete deletes a group.
		Delete(ctx context.Context, id int) error
	}
	ListGroupParameters struct {
		*PaginationArgs
	}
	ListGroupResult struct {
		*PaginationResults
		Groups []*ent.Group
	}
)

func NewGroupClient(client *ent.Client, dbType conf.DBType, cache cache.Driver) GroupClient {
	return &groupClient{client: client, maxSQlParam: sqlParamLimit(dbType), cache: cache}
}

type groupClient struct {
	client      *ent.Client
	cache       cache.Driver
	maxSQlParam int
}

func (c *groupClient) SetClient(newClient *ent.Client) TxOperator {
	return &groupClient{client: newClient, maxSQlParam: c.maxSQlParam, cache: c.cache}
}

func (c *groupClient) GetClient() *ent.Client {
	return c.client
}

func (c *groupClient) CountUsers(ctx context.Context, id int) (int, error) {
	return c.client.Group.Query().Where(group.ID(id)).QueryUsers().Count(ctx)
}

func (c *groupClient) AnonymousGroup(ctx context.Context) (*ent.Group, error) {
	return withGroupEagerLoading(ctx, c.client.Group.Query().Where(group.ID(AnonymousGroupID))).First(ctx)
}

func (c *groupClient) ListAll(ctx context.Context) ([]*ent.Group, error) {
	return withGroupEagerLoading(ctx, c.client.Group.Query()).All(ctx)
}

func (c *groupClient) Upsert(ctx context.Context, group *ent.Group) (*ent.Group, error) {
	if group.ID == 0 {
		stm := c.client.Group.Create().
			SetName(group.Name).
			SetMaxStorage(group.MaxStorage).
			SetSpeedLimit(group.SpeedLimit).
			SetPermissions(group.Permissions).
			SetSettings(group.Settings)

		if group.Edges.StoragePolicies != nil && group.Edges.StoragePolicies.ID > 0 {
			stm.SetStoragePolicyID(group.Edges.StoragePolicies.ID)
		}

		return stm.Save(ctx)
	}

	stm := c.client.Group.UpdateOne(group).
		SetName(group.Name).
		SetMaxStorage(group.MaxStorage).
		SetSpeedLimit(group.SpeedLimit).
		SetPermissions(group.Permissions).
		SetSettings(group.Settings).
		ClearStoragePolicies()

	if group.Edges.StoragePolicies != nil && group.Edges.StoragePolicies.ID > 0 {
		stm.SetStoragePolicyID(group.Edges.StoragePolicies.ID)
	}

	res, err := stm.Save(ctx)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c *groupClient) Delete(ctx context.Context, id int) error {
	if err := c.client.Group.DeleteOneID(id).Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	return nil
}

func (c *groupClient) ListGroups(ctx context.Context, args *ListGroupParameters) (*ListGroupResult, error) {
	query := withGroupEagerLoading(ctx, c.client.Group.Query())
	pageSize := capPageSize(c.maxSQlParam, args.PageSize, 10)
	queryWithoutOrder := query.Clone()
	query.Order(getGroupOrderOption(args)...)

	// Count total items
	total, err := queryWithoutOrder.Clone().
		Count(ctx)
	if err != nil {
		return nil, err
	}

	groups, err := query.Clone().
		Limit(pageSize).
		Offset(args.Page * pageSize).
		All(ctx)
	if err != nil {
		return nil, err
	}

	return &ListGroupResult{
		Groups: groups,
		PaginationResults: &PaginationResults{
			TotalItems: total,
			Page:       args.Page,
			PageSize:   pageSize,
		},
	}, nil
}

func (c *groupClient) GetByID(ctx context.Context, id int) (*ent.Group, error) {
	return withGroupEagerLoading(ctx, c.client.Group.Query().Where(group.ID(id))).First(ctx)
}

func getGroupOrderOption(args *ListGroupParameters) []group.OrderOption {
	orderTerm := getOrderTerm(args.Order)
	switch args.OrderBy {
	case group.FieldName:
		return []group.OrderOption{group.ByName(orderTerm), group.ByID(orderTerm)}
	case group.FieldMaxStorage:
		return []group.OrderOption{group.ByMaxStorage(orderTerm), group.ByID(orderTerm)}
	default:
		return []group.OrderOption{group.ByID(orderTerm)}
	}
}

func withGroupEagerLoading(ctx context.Context, q *ent.GroupQuery) *ent.GroupQuery {
	if _, ok := ctx.Value(LoadGroupPolicy{}).(bool); ok {
		q.WithStoragePolicies(func(spq *ent.StoragePolicyQuery) {
			withStoragePolicyEagerLoading(ctx, spq)
		})
	}
	return q
}
