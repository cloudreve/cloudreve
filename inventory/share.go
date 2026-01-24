package inventory

import (
	"context"
	"fmt"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/file"
	"github.com/cloudreve/Cloudreve/v4/ent/predicate"
	"github.com/cloudreve/Cloudreve/v4/ent/share"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/samber/lo"
)

type (
	// Ctx keys for eager loading options.
	LoadShareFile struct{}
	LoadShareUser struct{}
)

// WithShareEagerLoad returns a context with share eager loading options enabled.
// This loads the share's associated user and file when querying.
func WithShareEagerLoad(ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, LoadShareUser{}, true)
	ctx = context.WithValue(ctx, LoadShareFile{}, true)
	return ctx
}

var (
	ErrShareLinkExpired  = fmt.Errorf("share link expired")
	ErrOwnerInactive     = fmt.Errorf("owner is inactive")
	ErrSourceFileInvalid = fmt.Errorf("source file is deleted")
)

type (
	ShareClient interface {
		TxOperator
		// GetByIDs returns the shares with given ids.
		GetByIDs(ctx context.Context, ids []int) ([]*ent.Share, error)
		// GetByID returns the share with given id.
		GetByID(ctx context.Context, id int) (*ent.Share, error)
		// GetByIDUser returns the share with given id and user id.
		GetByIDUser(ctx context.Context, id, uid int) (*ent.Share, error)
		// GetByHashID returns the share with given hash id.
		GetByHashID(ctx context.Context, idRaw string) (*ent.Share, error)
		// Upsert creates or update a new share record.
		Upsert(ctx context.Context, params *CreateShareParams) (*ent.Share, error)
		// Viewed increase the view count of the share.
		Viewed(ctx context.Context, share *ent.Share) error
		// Downloaded increase the download count of the share.
		Downloaded(ctx context.Context, share *ent.Share) error
		// Delete deletes the share.
		Delete(ctx context.Context, shareId int) error
		// List returns a list of shares with the given args.
		List(ctx context.Context, args *ListShareArgs) (*ListShareResult, error)
		// CountByTimeRange counts the number of shares created in the given time range.
		CountByTimeRange(ctx context.Context, start, end *time.Time) (int, error)
		// DeleteBatch deletes the shares with the given ids.
		DeleteBatch(ctx context.Context, shareIds []int) error
	}

	CreateShareParams struct {
		Existed         *ent.Share
		Password        string
		RemainDownloads int
		Expires         *time.Time
		OwnerID         int
		FileID          int
		Props           *types.ShareProps
	}

	ListShareArgs struct {
		*PaginationArgs
		UserID     int
		FileID     int
		PublicOnly bool
		ShareIDs   []int
	}
	ListShareResult struct {
		*PaginationResults
		Shares []*ent.Share
	}
)

func NewShareClient(client *ent.Client, dbType conf.DBType, hasher hashid.Encoder) ShareClient {
	return &shareClient{
		client:      client,
		hasher:      hasher,
		maxSQlParam: sqlParamLimit(dbType),
	}
}

type shareClient struct {
	maxSQlParam int
	client      *ent.Client
	hasher      hashid.Encoder
}

func (c *shareClient) SetClient(newClient *ent.Client) TxOperator {
	return &shareClient{client: newClient, hasher: c.hasher, maxSQlParam: c.maxSQlParam}
}

func (c *shareClient) GetClient() *ent.Client {
	return c.client
}

func (c *shareClient) CountByTimeRange(ctx context.Context, start, end *time.Time) (int, error) {
	if start == nil || end == nil {
		return c.client.Share.Query().Count(ctx)
	}

	return c.client.Share.Query().Where(share.CreatedAtGTE(*start), share.CreatedAtLT(*end)).Count(ctx)
}

func (c *shareClient) Upsert(ctx context.Context, params *CreateShareParams) (*ent.Share, error) {
	if params.Existed != nil {
		createQuery := c.client.Share.
			UpdateOne(params.Existed)
		if params.RemainDownloads > 0 {
			createQuery.SetRemainDownloads(params.RemainDownloads)
		} else {
			createQuery.ClearRemainDownloads()
		}
		if params.Expires != nil {
			createQuery.SetNillableExpires(params.Expires)
		} else {
			createQuery.ClearExpires()
		}

		if params.Props != nil {
			createQuery.SetProps(params.Props)
		}

		return createQuery.Save(ctx)
	}

	query := c.client.Share.
		Create().
		SetUserID(params.OwnerID).
		SetFileID(params.FileID)
	if params.Password != "" {
		query.SetPassword(params.Password)
	}
	if params.RemainDownloads > 0 {
		query.SetRemainDownloads(params.RemainDownloads)
	}
	if params.Expires != nil {
		query.SetNillableExpires(params.Expires)
	}
	if params.Props != nil {
		query.SetProps(params.Props)
	}

	return query.Save(ctx)
}

func (c *shareClient) GetByHashID(ctx context.Context, idRaw string) (*ent.Share, error) {
	id, err := c.hasher.Decode(idRaw, hashid.ShareID)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hash id %q: %w", idRaw, err)
	}

	return c.GetByID(ctx, id)
}

func (c *shareClient) GetByID(ctx context.Context, id int) (*ent.Share, error) {
	s, err := withShareEagerLoading(ctx, c.client.Share.Query().Where(share.ID(id))).First(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query share %d: %w", id, err)
	}

	return s, nil
}

func (c *shareClient) GetByIDUser(ctx context.Context, id, uid int) (*ent.Share, error) {
	s, err := withShareEagerLoading(ctx, c.client.Share.Query().
		Where(share.ID(id))).
		Where(share.HasUserWith(user.ID(uid))).First(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query share %d: %w", id, err)
	}

	return s, nil
}

func (c *shareClient) GetByIDs(ctx context.Context, ids []int) ([]*ent.Share, error) {
	s, err := withShareEagerLoading(ctx, c.client.Share.Query().Where(share.IDIn(ids...))).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query shares %v: %w", ids, err)
	}

	return s, nil
}

func (c *shareClient) DeleteBatch(ctx context.Context, shareIds []int) error {
	_, err := c.client.Share.Delete().Where(share.IDIn(shareIds...)).Exec(ctx)
	return err
}

func (c *shareClient) Delete(ctx context.Context, shareId int) error {
	return c.client.Share.DeleteOneID(shareId).Exec(ctx)
}

// Viewed increments the view count of the share.
func (c *shareClient) Viewed(ctx context.Context, share *ent.Share) error {
	_, err := c.client.Share.UpdateOneID(share.ID).AddViews(1).Save(ctx)
	return err
}

// Downloaded increments the download count of the share.
func (c *shareClient) Downloaded(ctx context.Context, share *ent.Share) error {
	stm := c.client.Share.
		UpdateOneID(share.ID).
		AddDownloads(1)
	if share.RemainDownloads != nil && *share.RemainDownloads >= 0 {
		stm.AddRemainDownloads(-1)
	}
	_, err := stm.Save(ctx)
	return err
}

func IsValidShare(share *ent.Share) error {
	// Check if share is expired
	if err := IsShareExpired(share); err != nil {
		return err
	}

	// Check owner status
	owner, err := share.Edges.UserOrErr()
	if err != nil || owner.Status != user.StatusActive {
		// Owner already deleted, or not active.
		return ErrOwnerInactive
	}

	// Check source file status
	file, err := share.Edges.FileOrErr()
	if err != nil || file.FileChildren == 0 || file.OwnerID != owner.ID {
		// Source file already deleted
		return ErrSourceFileInvalid
	}

	return nil
}

func IsShareExpired(share *ent.Share) error {
	// Check if share is expired
	if (share.Expires != nil && share.Expires.Before(time.Now())) ||
		(share.RemainDownloads != nil && *share.RemainDownloads <= 0) {
		return ErrShareLinkExpired
	}

	return nil
}

func (c *shareClient) List(ctx context.Context, args *ListShareArgs) (*ListShareResult, error) {
	rawQuery := c.listQuery(args)
	query := withShareEagerLoading(ctx, rawQuery)

	var (
		shares        []*ent.Share
		err           error
		paginationRes *PaginationResults
	)
	if args.UseCursorPagination {
		shares, paginationRes, err = c.cursorPagination(ctx, query, args, 10)
	} else {
		shares, paginationRes, err = c.offsetPagination(ctx, query, args, 10)
	}

	if err != nil {
		return nil, fmt.Errorf("query failed with paginiation: %w", err)
	}

	return &ListShareResult{
		Shares:            shares,
		PaginationResults: paginationRes,
	}, nil
}

func (c *shareClient) cursorPagination(ctx context.Context, query *ent.ShareQuery, args *ListShareArgs, paramMargin int) ([]*ent.Share, *PaginationResults, error) {
	pageSize := capPageSize(c.maxSQlParam, args.PageSize, paramMargin)
	query.Order(getShareOrderOption(args)...)

	var (
		pageToken *PageToken
		err       error
	)
	if args.PageToken != "" {
		pageToken, err = pageTokenFromString(args.PageToken, c.hasher, hashid.ShareID)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid page token %q: %w", args.PageToken, err)
		}
	}
	queryPaged := getShareCursorQuery(args, pageToken, query)

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
		nextToken, err := getShareNextPageToken(c.hasher, lastItem, args)
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

func (c *shareClient) offsetPagination(ctx context.Context, query *ent.ShareQuery, args *ListShareArgs, paramMargin int) ([]*ent.Share, *PaginationResults, error) {
	pageSize := capPageSize(c.maxSQlParam, args.PageSize, paramMargin)
	query.Order(getShareOrderOption(args)...)

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	logs, err := query.Limit(pageSize).Offset(args.Page * args.PageSize).All(ctx)
	if err != nil {
		return nil, nil, err
	}

	return logs, &PaginationResults{
		PageSize:   pageSize,
		TotalItems: total,
		Page:       args.Page,
	}, nil
}

func (c *shareClient) listQuery(args *ListShareArgs) *ent.ShareQuery {
	query := c.client.Share.Query()
	if args.UserID > 0 {
		query.Where(share.HasUserWith(user.ID(args.UserID)))
	}

	if args.PublicOnly {
		query.Where(share.PasswordIsNil())
	}

	if args.FileID > 0 {
		query.Where(share.HasFileWith(file.ID(args.FileID)))
	}

	if len(args.ShareIDs) > 0 {
		query.Where(share.IDIn(args.ShareIDs...))
	}

	return query
}

// getShareNextPageToken returns the next page token for the given last share.
func getShareNextPageToken(hasher hashid.Encoder, last *ent.Share, args *ListShareArgs) (string, error) {
	token := &PageToken{
		ID: last.ID,
	}

	return token.Encode(hasher, hashid.EncodeShareID)
}

func getShareCursorQuery(args *ListShareArgs, token *PageToken, query *ent.ShareQuery) *ent.ShareQuery {
	o := &sql.OrderTermOptions{}
	getOrderTerm(args.Order)(o)

	predicates, ok := shareCursorQuery[args.OrderBy]
	if !ok {
		predicates = shareCursorQuery[share.FieldID]
	}

	if token != nil {
		query.Where(predicates[o.Desc](token))
	}

	return query
}

var shareCursorQuery = map[string]map[bool]func(token *PageToken) predicate.Share{
	share.FieldID: {
		true: func(token *PageToken) predicate.Share {
			return share.IDLT(token.ID)
		},
		false: func(token *PageToken) predicate.Share {
			return share.IDGT(token.ID)
		},
	},
}

func getShareOrderOption(args *ListShareArgs) []share.OrderOption {
	orderTerm := getOrderTerm(args.Order)
	switch args.OrderBy {
	case share.FieldViews:
		return []share.OrderOption{share.ByViews(orderTerm), share.ByID(orderTerm)}
	case share.FieldDownloads:
		return []share.OrderOption{share.ByDownloads(orderTerm), share.ByID(orderTerm)}
	case share.FieldRemainDownloads:
		return []share.OrderOption{share.ByRemainDownloads(orderTerm), share.ByID(orderTerm)}
	default:
		return []share.OrderOption{share.ByID(orderTerm)}
	}
}

func withShareEagerLoading(ctx context.Context, q *ent.ShareQuery) *ent.ShareQuery {
	if v, ok := ctx.Value(LoadShareFile{}).(bool); ok && v {
		q.WithFile(func(q *ent.FileQuery) {
			withFileEagerLoading(ctx, q)
		})
	}
	if v, ok := ctx.Value(LoadShareUser{}).(bool); ok && v {
		q.WithUser(func(q *ent.UserQuery) {
			withUserEagerLoading(ctx, q)
		})
	}

	return q
}
