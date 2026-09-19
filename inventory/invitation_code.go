package inventory

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/invitationcode"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/samber/lo"
)

var (
	ErrInvitationCodeNotFound  = errors.New("invitation code not found")
	ErrInvitationCodeExpired   = errors.New("invitation code expired")
	ErrInvitationCodeExhausted = errors.New("invitation code fully used")
)

type (
	InvitationCodeClient interface {
		TxOperator
		// Create creates a new invitation code.
		Create(ctx context.Context, params *CreateInvitationCodeParams) (*ent.InvitationCode, error)
		// List returns a paginated list of invitation codes.
		List(ctx context.Context, args *ListInvitationCodeArgs) (*ListInvitationCodeResult, error)
		// Delete revokes an invitation code.
		Delete(ctx context.Context, id int) error
		// Consume validates the code and atomically increments its use count.
		// The returned record carries the group assignment for the new user.
		Consume(ctx context.Context, code string) (*ent.InvitationCode, error)
	}

	CreateInvitationCodeParams struct {
		Code      string
		GroupID   int
		MaxUses   int
		ExpiresAt *time.Time
	}

	ListInvitationCodeArgs struct {
		*PaginationArgs
		Keyword string
	}

	ListInvitationCodeResult struct {
		*PaginationResults
		Codes []*ent.InvitationCode
	}
)

func NewInvitationCodeClient(client *ent.Client, dbType conf.DBType, hasher hashid.Encoder) InvitationCodeClient {
	return &invitationCodeClient{
		client:      client,
		hasher:      hasher,
		maxSQlParam: sqlParamLimit(dbType),
	}
}

type invitationCodeClient struct {
	maxSQlParam int
	client      *ent.Client
	hasher      hashid.Encoder
}

func (c *invitationCodeClient) SetClient(newClient *ent.Client) TxOperator {
	return &invitationCodeClient{client: newClient, hasher: c.hasher, maxSQlParam: c.maxSQlParam}
}

func (c *invitationCodeClient) GetClient() *ent.Client {
	return c.client
}

func (c *invitationCodeClient) Create(ctx context.Context, params *CreateInvitationCodeParams) (*ent.InvitationCode, error) {
	create := c.client.InvitationCode.Create().
		SetCode(params.Code).
		SetGroupID(params.GroupID).
		SetMaxUses(params.MaxUses)
	if params.ExpiresAt != nil {
		create.SetExpiresAt(*params.ExpiresAt)
	}

	return create.Save(ctx)
}

func (c *invitationCodeClient) Delete(ctx context.Context, id int) error {
	return c.client.InvitationCode.DeleteOneID(id).Exec(ctx)
}

func (c *invitationCodeClient) Consume(ctx context.Context, code string) (*ent.InvitationCode, error) {
	record, err := c.client.InvitationCode.Query().
		Where(invitationcode.CodeEQ(code)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrInvitationCodeNotFound
		}
		return nil, err
	}

	if record.ExpiresAt != nil && record.ExpiresAt.Before(time.Now()) {
		return nil, ErrInvitationCodeExpired
	}
	if record.MaxUses > 0 && record.UsedCount >= record.MaxUses {
		return nil, ErrInvitationCodeExhausted
	}

	// Guarded increment keeps concurrent registrations honest: only one can win
	// the last remaining use.
	affected, err := c.client.InvitationCode.Update().
		Where(
			invitationcode.ID(record.ID),
			invitationcode.Or(
				invitationcode.ExpiresAtIsNil(),
				invitationcode.ExpiresAtGT(time.Now()),
			),
			invitationcode.Or(
				invitationcode.MaxUsesEQ(0),
				invitationcode.UsedCountLT(record.MaxUses),
			),
		).
		AddUsedCount(1).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, ErrInvitationCodeExhausted
	}

	return record, nil
}

func (c *invitationCodeClient) List(ctx context.Context, args *ListInvitationCodeArgs) (*ListInvitationCodeResult, error) {
	query := c.client.InvitationCode.Query()
	if args.Keyword != "" {
		query.Where(invitationcode.CodeContainsFold(args.Keyword))
	}
	query.Order(invitationcode.ByID(sql.OrderDesc()))

	pageSize := capPageSize(c.maxSQlParam, args.PageSize, 10)
	var pageToken *PageToken
	if args.PageToken != "" {
		var err error
		pageToken, err = pageTokenFromString(args.PageToken, c.hasher, hashid.InvitationCodeID)
		if err != nil {
			return nil, fmt.Errorf("invalid page token %q: %w", args.PageToken, err)
		}
		query.Where(invitationcode.IDLT(pageToken.ID))
	}

	query.Limit(pageSize + 1)
	codes, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	nextTokenStr := ""
	if len(codes) > pageSize {
		nextToken, err := (&PageToken{ID: codes[len(codes)-2].ID}).Encode(c.hasher, hashid.EncodeInvitationCodeID)
		if err != nil {
			return nil, fmt.Errorf("failed to generate next page token: %w", err)
		}
		nextTokenStr = nextToken
	}

	return &ListInvitationCodeResult{
		Codes: lo.Subset(codes, 0, uint(pageSize)),
		PaginationResults: &PaginationResults{
			PageSize:      pageSize,
			NextPageToken: nextTokenStr,
			IsCursor:      true,
		},
	}, nil
}
