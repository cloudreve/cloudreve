package inventory

import (
	"context"
	"entgo.io/ent/dialect/sql"
	"fmt"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/davaccount"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/samber/lo"
	"strings"
)

type (
	DavAccountClient interface {
		TxOperator
		// List returns a list of dav accounts with the given args.
		List(ctx context.Context, args *ListDavAccountArgs) (*ListDavAccountResult, error)
		// Create creates a new dav account.
		Create(ctx context.Context, params *CreateDavAccountParams) (*ent.DavAccount, error)
		// Update updates a dav account.
		Update(ctx context.Context, id int, params *CreateDavAccountParams) (*ent.DavAccount, error)
		// GetByIDAndUserID returns the dav account with given id and user id.
		GetByIDAndUserID(ctx context.Context, id, userID int) (*ent.DavAccount, error)
		// Delete deletes the dav account.
		Delete(ctx context.Context, id int) error
		// UpdateURIPrefix rewrites the mount URIs of a user's dav accounts after
		// a file rename or move: an account mounting exactly oldURI, or any path
		// under it, is repointed to newURI. This keeps mounts working when the
		// bound folder changes location.
		UpdateURIPrefix(ctx context.Context, userID int, oldURI, newURI string) error
	}

	ListDavAccountArgs struct {
		*PaginationArgs
		UserID int
	}
	ListDavAccountResult struct {
		*PaginationResults
		Accounts []*ent.DavAccount
	}
	CreateDavAccountParams struct {
		UserID   int
		Name     string
		URI      string
		Password string
		Options  *boolset.BooleanSet
	}
)

func NewDavAccountClient(client *ent.Client, dbType conf.DBType, hasher hashid.Encoder) DavAccountClient {
	return &davAccountClient{
		client:      client,
		hasher:      hasher,
		maxSQlParam: sqlParamLimit(dbType),
	}
}

type davAccountClient struct {
	maxSQlParam int
	client      *ent.Client
	hasher      hashid.Encoder
}

func (c *davAccountClient) SetClient(newClient *ent.Client) TxOperator {
	return &davAccountClient{client: newClient, hasher: c.hasher, maxSQlParam: c.maxSQlParam}
}

func (c *davAccountClient) GetClient() *ent.Client {
	return c.client
}

func (c *davAccountClient) Create(ctx context.Context, params *CreateDavAccountParams) (*ent.DavAccount, error) {
	account := c.client.DavAccount.Create().
		SetOwnerID(params.UserID).
		SetName(params.Name).
		SetURI(params.URI).
		SetPassword(params.Password).
		SetOptions(params.Options)

	return account.Save(ctx)
}

func (c *davAccountClient) GetByIDAndUserID(ctx context.Context, id, userID int) (*ent.DavAccount, error) {
	return c.client.DavAccount.Query().
		Where(davaccount.ID(id), davaccount.OwnerID(userID)).
		First(ctx)
}

func (c *davAccountClient) Update(ctx context.Context, id int, params *CreateDavAccountParams) (*ent.DavAccount, error) {
	account := c.client.DavAccount.UpdateOneID(id).
		SetName(params.Name).
		SetURI(params.URI).
		SetOptions(params.Options)

	return account.Save(ctx)
}

func (c *davAccountClient) Delete(ctx context.Context, id int) error {
	return c.client.DavAccount.DeleteOneID(id).Exec(ctx)
}

func (c *davAccountClient) UpdateURIPrefix(ctx context.Context, userID int, oldURI, newURI string) error {
	oldURI = strings.TrimSuffix(oldURI, "/")
	newURI = strings.TrimSuffix(newURI, "/")
	if oldURI == newURI || userID == 0 {
		return nil
	}

	accounts, err := c.client.DavAccount.Query().
		Where(
			davaccount.OwnerIDEQ(userID),
			davaccount.Or(
				davaccount.URIEQ(oldURI),
				davaccount.URIHasPrefix(oldURI+"/"),
			),
		).
		All(ctx)
	if err != nil {
		return fmt.Errorf("failed to query dav accounts for URI rewrite: %w", err)
	}

	for _, account := range accounts {
		updated := strings.TrimSuffix(newURI+strings.TrimPrefix(account.URI, oldURI), "/")
		if err := c.client.DavAccount.UpdateOneID(account.ID).SetURI(updated).Exec(ctx); err != nil {
			return fmt.Errorf("failed to rewrite dav account URI: %w", err)
		}
	}
	return nil
}

func (c *davAccountClient) List(ctx context.Context, args *ListDavAccountArgs) (*ListDavAccountResult, error) {
	query := c.listQuery(args)

	var (
		accounts      []*ent.DavAccount
		err           error
		paginationRes *PaginationResults
	)
	accounts, paginationRes, err = c.cursorPagination(ctx, query, args, 10)

	if err != nil {
		return nil, fmt.Errorf("query failed with paginiation: %w", err)
	}

	return &ListDavAccountResult{
		Accounts:          accounts,
		PaginationResults: paginationRes,
	}, nil
}

func (c *davAccountClient) cursorPagination(ctx context.Context, query *ent.DavAccountQuery, args *ListDavAccountArgs, paramMargin int) ([]*ent.DavAccount, *PaginationResults, error) {
	pageSize := capPageSize(c.maxSQlParam, args.PageSize, paramMargin)
	query.Order(davaccount.ByID(sql.OrderDesc()))

	var (
		pageToken *PageToken
		err       error
	)
	if args.PageToken != "" {
		pageToken, err = pageTokenFromString(args.PageToken, c.hasher, hashid.DavAccountID)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid page token %q: %w", args.PageToken, err)
		}
	}
	queryPaged := getDavAccountCursorQuery(pageToken, query)

	// Use page size + 1 to determine if there are more items to come
	queryPaged.Limit(pageSize + 1)

	logs, err := queryPaged.
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	// More items to come
	nextTokenStr := ""
	if len(logs) > pageSize {
		lastItem := logs[len(logs)-2]
		nextToken, err := getDavAccountNextPageToken(c.hasher, lastItem)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate next page token: %w", err)
		}

		nextTokenStr = nextToken
	}

	return lo.Subset(logs, 0, uint(pageSize)), &PaginationResults{
		PageSize:      pageSize,
		NextPageToken: nextTokenStr,
		IsCursor:      true,
	}, nil
}

func (c *davAccountClient) listQuery(args *ListDavAccountArgs) *ent.DavAccountQuery {
	query := c.client.DavAccount.Query()
	if args.UserID > 0 {
		query.Where(davaccount.OwnerID(args.UserID))
	}

	return query
}

// getDavAccountNextPageToken returns the next page token for the given last dav account.
func getDavAccountNextPageToken(hasher hashid.Encoder, last *ent.DavAccount) (string, error) {
	token := &PageToken{
		ID: last.ID,
	}

	return token.Encode(hasher, hashid.EncodeDavAccountID)
}

func getDavAccountCursorQuery(token *PageToken, query *ent.DavAccountQuery) *ent.DavAccountQuery {
	if token != nil {
		query.Where(davaccount.IDLT(token.ID))
	}

	return query
}
