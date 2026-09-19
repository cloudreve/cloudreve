package inventory

import (
	"context"
	"errors"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/credittxn"
	"github.com/cloudreve/Cloudreve/v4/ent/giftcode"
	"github.com/cloudreve/Cloudreve/v4/ent/schema"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/ent/usergrant"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/gofrs/uuid"
)

var (
	ErrGiftCodeNotFound = errors.New("gift code not found")
	ErrGiftCodeUsed     = errors.New("gift code already used")
)

type (
	VasClient interface {
		TxOperator
		// CreditAdjust atomically moves userID's balance by delta and records a
		// ledger row. Negative deltas that would overdraw the balance fail with
		// ErrInsufficientPoints; no ledger row is written in that case.
		CreditAdjust(ctx context.Context, userID int, delta int64, typ credittxn.Type, ref, des string) error
		// ListCreditTxns returns a page of ledger rows newest-first plus the
		// total count for pagination.
		ListCreditTxns(ctx context.Context, userID, page, pageSize int) ([]*ent.CreditTxn, int, error)
		// CreateGiftCodes inserts qty codes sharing the given grant definition
		// and returns the created rows (with their generated code strings).
		CreateGiftCodes(ctx context.Context, params *CreateGiftCodeParams) ([]*ent.GiftCode, error)
		// ListGiftCodes returns a page of codes newest-first plus total count.
		ListGiftCodes(ctx context.Context, page, pageSize int) ([]*ent.GiftCode, int, error)
		// DeleteGiftCodes removes the given codes. Used codes are skipped.
		DeleteGiftCodes(ctx context.Context, ids []int) error
		// RedeemGiftCode atomically claims code for userID and applies its
		// grant inside the same transaction. Returns ErrGiftCodeNotFound or
		// ErrGiftCodeUsed for invalid claims.
		RedeemGiftCode(ctx context.Context, userID int, code string) (*ent.GiftCode, error)
		// StorageBonus sums the amount of all unexpired storage grants.
		StorageBonus(ctx context.Context, userID int) (int64, error)
		// ListGrants returns all grants of a user, active and expired.
		ListGrants(ctx context.Context, userID int) ([]*ent.UserGrant, error)
		// ExpireGrants deletes expired storage grants and reverts group
		// upgrades whose users still sit on the granted group. Users without a
		// recorded previous group fall back to defaultGroupID.
		ExpireGrants(ctx context.Context, defaultGroupID int) error
	}

	CreateGiftCodeParams struct {
		Type     giftcode.Type
		Amount   int64
		Duration time.Duration
		Des      string
		Qty      int
	}
)

func NewVasClient(client *ent.Client, dbType conf.DBType) VasClient {
	return &vasClient{
		client:      client,
		maxSQlParam: sqlParamLimit(dbType),
	}
}

type vasClient struct {
	maxSQlParam int
	client      *ent.Client
}

func (c *vasClient) SetClient(newClient *ent.Client) TxOperator {
	return &vasClient{client: newClient, maxSQlParam: c.maxSQlParam}
}

func (c *vasClient) GetClient() *ent.Client {
	return c.client
}

func (c *vasClient) CreditAdjust(ctx context.Context, userID int, delta int64, typ credittxn.Type, ref, des string) error {
	txVc, tx, ctx, err := WithTx(ctx, c)
	if err != nil {
		return err
	}

	q := txVc.client.User.Update().Where(user.ID(userID)).AddCredits(delta)
	if delta < 0 {
		q = q.Where(user.CreditsGTE(-delta))
	}
	n, err := q.Save(ctx)
	if err != nil {
		return Rollback(tx)
	}
	if n == 0 {
		_ = Rollback(tx)
		return ErrInsufficientPoints
	}

	if _, err := txVc.client.CreditTxn.Create().
		SetUserID(userID).
		SetAmount(delta).
		SetType(typ).
		SetRef(ref).
		SetDes(des).
		Save(ctx); err != nil {
		return Rollback(tx)
	}

	return Commit(tx)
}

func (c *vasClient) ListCreditTxns(ctx context.Context, userID, page, pageSize int) ([]*ent.CreditTxn, int, error) {
	q := c.client.CreditTxn.Query().Where(credittxn.UserID(userID))
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	txns, err := q.
		Order(ent.Desc(credittxn.FieldCreatedAt)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	return txns, total, err
}

func (c *vasClient) CreateGiftCodes(ctx context.Context, params *CreateGiftCodeParams) ([]*ent.GiftCode, error) {
	bulk := make([]*ent.GiftCodeCreate, 0, params.Qty)
	for i := 0; i < params.Qty; i++ {
		create := c.client.GiftCode.Create().
			SetCode(newGiftCodeString()).
			SetType(params.Type).
			SetAmount(params.Amount).
			SetDes(params.Des)
		if params.Duration > 0 {
			create = create.SetDuration(int64(params.Duration.Seconds()))
		}
		bulk = append(bulk, create)
	}

	return c.client.GiftCode.CreateBulk(bulk...).Save(ctx)
}

func (c *vasClient) ListGiftCodes(ctx context.Context, page, pageSize int) ([]*ent.GiftCode, int, error) {
	q := c.client.GiftCode.Query().WithRedeemer()
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	codes, err := q.
		Order(ent.Desc(giftcode.FieldCreatedAt)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	return codes, total, err
}

func (c *vasClient) DeleteGiftCodes(ctx context.Context, ids []int) error {
	_, err := c.client.GiftCode.Delete().
		Where(giftcode.IDIn(ids...), giftcode.UsedByIDIsNil()).
		Exec(schema.SkipSoftDelete(ctx))
	return err
}

func (c *vasClient) RedeemGiftCode(ctx context.Context, userID int, code string) (*ent.GiftCode, error) {
	gc, err := c.client.GiftCode.Query().
		Where(giftcode.Code(code)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrGiftCodeNotFound
		}
		return nil, err
	}

	// Atomic claim in autocommit: only one redeemer can flip used_by_id
	// from NULL. Running this inside the grant tx would let concurrent
	// redeemers race on tx snapshots (same pitfall as invitation codes).
	n, err := c.client.GiftCode.Update().
		Where(giftcode.ID(gc.ID), giftcode.UsedByIDIsNil()).
		SetUsedByID(userID).
		SetUsedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, ErrGiftCodeUsed
	}

	origCtx := ctx
	txVc, tx, ctx, err := WithTx(ctx, c)
	if err != nil {
		return nil, err
	}

	switch gc.Type {
	case giftcode.TypePoints:
		err = txVc.CreditAdjust(ctx, userID, gc.Amount, credittxn.TypeGift, code, "gift code redeemed")
	case giftcode.TypeStorage:
		err = txVc.createGrant(ctx, userID, usergrant.TypeStorage, gc.Amount, gc.Duration, 0)
	case giftcode.TypeGroup:
		err = txVc.applyGroupCode(ctx, userID, gc)
	}
	if err != nil {
		_ = Rollback(tx)
		// Compensate the claim so a failed grant does not burn the code.
		// Must run on the pre-tx ctx — the rolled-back tx client is dead.
		// A crash between claim and apply still burns it — same accepted
		// window as invitation-code consumption.
		_, _ = c.client.GiftCode.Update().
			Where(giftcode.ID(gc.ID), giftcode.UsedByID(userID)).
			ClearUsedByID().
			ClearUsedAt().
			Save(origCtx)
		return nil, err
	}

	if err := Commit(tx); err != nil {
		return nil, err
	}
	gc.UsedByID = userID
	return gc, nil
}

func (c *vasClient) applyGroupCode(ctx context.Context, userID int, gc *ent.GiftCode) error {
	u, err := c.client.User.Get(ctx, userID)
	if err != nil {
		return err
	}
	targetGroup := int(gc.Amount)
	if _, err := c.client.Group.Get(ctx, targetGroup); err != nil {
		return err
	}
	if err := c.createGrant(ctx, userID, usergrant.TypeGroup, gc.Amount, gc.Duration, u.GroupUsers); err != nil {
		return err
	}
	return c.client.User.Update().Where(user.ID(userID)).SetGroupUsers(targetGroup).Exec(ctx)
}

func (c *vasClient) createGrant(ctx context.Context, userID int, typ usergrant.Type, amount, durationSeconds int64, prevGroupID int) error {
	create := c.client.UserGrant.Create().
		SetUserID(userID).
		SetType(typ).
		SetAmount(amount)
	if durationSeconds > 0 {
		create = create.SetExpiresAt(time.Now().Add(time.Duration(durationSeconds) * time.Second))
	}
	if prevGroupID > 0 {
		create = create.SetPrevGroupID(prevGroupID)
	}
	return create.Exec(ctx)
}

func (c *vasClient) StorageBonus(ctx context.Context, userID int) (int64, error) {
	var v []struct {
		Sum int64 `json:"sum"`
	}
	err := c.client.UserGrant.Query().
		Where(
			usergrant.UserID(userID),
			usergrant.TypeEQ(usergrant.TypeStorage),
			usergrant.Or(
				usergrant.ExpiresAtIsNil(),
				usergrant.ExpiresAtGT(time.Now()),
			),
		).
		Aggregate(ent.Sum(usergrant.FieldAmount)).
		Scan(ctx, &v)
	if err != nil {
		return 0, err
	}
	if len(v) == 0 {
		return 0, nil
	}
	return v[0].Sum, nil
}

func (c *vasClient) ListGrants(ctx context.Context, userID int) ([]*ent.UserGrant, error) {
	return c.client.UserGrant.Query().
		Where(usergrant.UserID(userID)).
		Order(ent.Desc(usergrant.FieldCreatedAt)).
		All(ctx)
}

func (c *vasClient) ExpireGrants(ctx context.Context, defaultGroupID int) error {
	now := time.Now()
	expired, err := c.client.UserGrant.Query().
		Where(usergrant.ExpiresAtLT(now)).
		All(ctx)
	if err != nil {
		return err
	}

	for _, g := range expired {
		if g.Type == usergrant.TypeGroup {
			revertTo := g.PrevGroupID
			if revertTo <= 0 {
				revertTo = defaultGroupID
			}
			// Revert only when the user still sits on the granted group —
			// intervening group changes win.
			if _, err := c.client.User.Update().
				Where(user.ID(g.UserID), user.GroupUsers(int(g.Amount))).
				SetGroupUsers(revertTo).
				Save(ctx); err != nil {
				return err
			}
		}
		if _, err := c.client.UserGrant.Delete().
			Where(usergrant.ID(g.ID)).
			Exec(schema.SkipSoftDelete(ctx)); err != nil {
			return err
		}
	}
	return nil
}

func newGiftCodeString() string {
	return uuid.Must(uuid.NewV4()).String()
}
