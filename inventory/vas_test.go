package inventory

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/credittxn"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/ent/giftcode"
	"github.com/cloudreve/Cloudreve/v4/ent/usergrant"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/stretchr/testify/require"
)

func newVasClient(t *testing.T) (*ent.Client, VasClient) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	return client, NewVasClient(client, "sqlite3")
}

func vasFixture(t *testing.T, client *ent.Client) (*ent.Group, *ent.User) {
	ctx := context.Background()
	group := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	u := client.User.Create().SetEmail("u@example.com").SetNick("u").SetGroup(group).SaveX(ctx)
	return group, u
}

func TestCreditAdjust(t *testing.T) {
	ctx := context.Background()
	client, c := newVasClient(t)
	_, u := vasFixture(t, client)

	require.NoError(t, c.CreditAdjust(ctx, u.ID, 100, credittxn.TypeAdjust, "", "grant"))
	require.NoError(t, c.CreditAdjust(ctx, u.ID, -30, credittxn.TypePurchase, "sku:1", ""))
	require.ErrorIs(t, c.CreditAdjust(ctx, u.ID, -1000, credittxn.TypePurchase, "", ""), ErrInsufficientPoints)

	u = client.User.GetX(ctx, u.ID)
	require.Equal(t, int64(70), u.Credits)

	txns, total, err := c.ListCreditTxns(ctx, u.ID, 1, 10)
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, txns, 2)
	require.Equal(t, credittxn.TypePurchase, txns[0].Type)
}

func TestRedeemGiftCode(t *testing.T) {
	ctx := context.Background()
	client, c := newVasClient(t)
	_, u := vasFixture(t, client)

	codes, err := c.CreateGiftCodes(ctx, &CreateGiftCodeParams{
		Type: giftcode.TypePoints, Amount: 50, Qty: 1, Des: "promo",
	})
	require.NoError(t, err)
	code := codes[0].Code

	_, err = c.RedeemGiftCode(ctx, u.ID, "nonexistent-code")
	require.ErrorIs(t, err, ErrGiftCodeNotFound)

	gc, err := c.RedeemGiftCode(ctx, u.ID, code)
	require.NoError(t, err)
	require.Equal(t, u.ID, gc.UsedByID)
	require.Equal(t, int64(50), client.User.GetX(ctx, u.ID).Credits)

	_, err = c.RedeemGiftCode(ctx, u.ID, code)
	require.ErrorIs(t, err, ErrGiftCodeUsed)
}

func TestRedeemGiftCodeConcurrent(t *testing.T) {
	ctx := context.Background()
	client, c := newVasClient(t)
	_, u := vasFixture(t, client)

	codes, err := c.CreateGiftCodes(ctx, &CreateGiftCodeParams{
		Type: giftcode.TypePoints, Amount: 10, Qty: 1,
	})
	require.NoError(t, err)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.RedeemGiftCode(ctx, u.ID, codes[0].Code)
			mu.Lock()
			errs = append(errs, err)
			mu.Unlock()
		}()
	}
	wg.Wait()

	succeeded := 0
	for _, err := range errs {
		if err == nil {
			succeeded++
		} else {
			require.ErrorIs(t, err, ErrGiftCodeUsed)
		}
	}
	require.Equal(t, 1, succeeded)
	require.Equal(t, int64(10), client.User.GetX(ctx, u.ID).Credits)
}

func TestRedeemStorageAndGroupCodes(t *testing.T) {
	ctx := context.Background()
	client, c := newVasClient(t)
	group, u := vasFixture(t, client)
	vip := client.Group.Create().SetName("vip").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)

	// Storage code → expiring grant, counted in StorageBonus.
	storageCode, err := c.CreateGiftCodes(ctx, &CreateGiftCodeParams{
		Type: giftcode.TypeStorage, Amount: 1024, Duration: time.Hour, Qty: 1,
	})
	require.NoError(t, err)
	_, err = c.RedeemGiftCode(ctx, u.ID, storageCode[0].Code)
	require.NoError(t, err)

	bonus, err := c.StorageBonus(ctx, u.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1024), bonus)

	// Group code → grant records prev group, user moved to target.
	groupCode, err := c.CreateGiftCodes(ctx, &CreateGiftCodeParams{
		Type: giftcode.TypeGroup, Amount: int64(vip.ID), Duration: time.Hour, Qty: 1,
	})
	require.NoError(t, err)
	_, err = c.RedeemGiftCode(ctx, u.ID, groupCode[0].Code)
	require.NoError(t, err)
	require.Equal(t, vip.ID, client.User.GetX(ctx, u.ID).GroupUsers)

	grant := client.UserGrant.Query().Where(usergrant.TypeEQ(usergrant.TypeGroup)).OnlyX(ctx)
	require.Equal(t, group.ID, grant.PrevGroupID)
}

func TestExpireGrants(t *testing.T) {
	ctx := context.Background()
	client, c := newVasClient(t)
	group, u := vasFixture(t, client)
	vip := client.Group.Create().SetName("vip").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)

	// Expired storage grant: excluded from bonus, swept by ExpireGrants.
	client.UserGrant.Create().
		SetUserID(u.ID).SetType(usergrant.TypeStorage).SetAmount(2048).
		SetExpiresAt(time.Now().Add(-time.Minute)).SaveX(ctx)
	// Active storage grant survives.
	client.UserGrant.Create().
		SetUserID(u.ID).SetType(usergrant.TypeStorage).SetAmount(512).
		SetExpiresAt(time.Now().Add(time.Hour)).SaveX(ctx)
	// Expired group grant with user still on granted group → revert.
	client.UserGrant.Create().
		SetUserID(u.ID).SetType(usergrant.TypeGroup).SetAmount(int64(vip.ID)).
		SetPrevGroupID(group.ID).SetExpiresAt(time.Now().Add(-time.Minute)).SaveX(ctx)
	client.User.UpdateOneID(u.ID).SetGroupUsers(vip.ID).ExecX(ctx)

	bonus, err := c.StorageBonus(ctx, u.ID)
	require.NoError(t, err)
	require.Equal(t, int64(512), bonus)

	require.NoError(t, c.ExpireGrants(ctx, 2))
	require.Equal(t, group.ID, client.User.GetX(ctx, u.ID).GroupUsers)
	require.Equal(t, 1, client.UserGrant.Query().CountX(ctx))
}

func TestDeleteGiftCodes(t *testing.T) {
	ctx := context.Background()
	client, c := newVasClient(t)
	_, u := vasFixture(t, client)

	codes, err := c.CreateGiftCodes(ctx, &CreateGiftCodeParams{
		Type: giftcode.TypePoints, Amount: 1, Qty: 2,
	})
	require.NoError(t, err)
	require.Len(t, codes, 2)

	_, err = c.RedeemGiftCode(ctx, u.ID, codes[0].Code)
	require.NoError(t, err)

	require.NoError(t, c.DeleteGiftCodes(ctx, []int{codes[0].ID, codes[1].ID}))
	// Used code survives revocation; unused is gone.
	require.Equal(t, 1, client.GiftCode.Query().CountX(ctx))
}
