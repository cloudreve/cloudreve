package inventory

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/credittxn"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/ent/giftcode"
	"github.com/cloudreve/Cloudreve/v4/ent/sku"
	"github.com/cloudreve/Cloudreve/v4/ent/usergrant"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
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

	// Group code → membership added on top of the primary group, which is
	// never replaced; the grant row records the purchase for bookkeeping.
	groupCode, err := c.CreateGiftCodes(ctx, &CreateGiftCodeParams{
		Type: giftcode.TypeGroup, Amount: int64(vip.ID), Duration: time.Hour, Qty: 1,
	})
	require.NoError(t, err)
	_, err = c.RedeemGiftCode(ctx, u.ID, groupCode[0].Code)
	require.NoError(t, err)
	require.Equal(t, group.ID, client.User.GetX(ctx, u.ID).GroupUsers)

	ms := client.GroupMembership.Query().AllX(ctx)
	require.Len(t, ms, 1)
	require.Equal(t, vip.ID, ms[0].GroupID)
	require.NotNil(t, ms[0].Expires)

	grant := client.UserGrant.Query().Where(usergrant.TypeEQ(usergrant.TypeGroup)).OnlyX(ctx)
	require.Equal(t, int64(vip.ID), grant.Amount)
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

	reverted, err := c.ExpireGrants(ctx, 2)
	require.NoError(t, err)
	require.Len(t, reverted, 1)
	require.Equal(t, u.ID, reverted[0].UserID)
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

func TestPurchaseSku(t *testing.T) {
	ctx := context.Background()
	client, c := newVasClient(t)
	group, u := vasFixture(t, client)
	vip := client.Group.Create().SetName("vip").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	require.NoError(t, c.CreditAdjust(ctx, u.ID, 500, credittxn.TypeAdjust, "", "seed"))

	points := int64(200)
	storageSku, err := c.UpsertSku(ctx, &ent.Sku{
		Name: "1GB pack", Type: sku.TypeStorage, Amount: 1024,
		Duration: int64(time.Hour.Seconds()), Points: &points, Enabled: true,
	})
	require.NoError(t, err)

	require.NoError(t, c.PurchaseSku(ctx, u.ID, storageSku))
	u = client.User.GetX(ctx, u.ID)
	require.Equal(t, int64(300), u.Credits)

	bonus, err := c.StorageBonus(ctx, u.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1024), bonus)

	txns, _, _ := c.ListCreditTxns(ctx, u.ID, 1, 10)
	require.Equal(t, credittxn.TypePurchase, txns[0].Type)
	require.Equal(t, int64(-200), txns[0].Amount)

	// Group sku → membership added; primary group preserved.
	groupSku, err := c.UpsertSku(ctx, &ent.Sku{
		Name: "vip month", Type: sku.TypeGroup, Amount: int64(vip.ID),
		Duration: int64(time.Hour.Seconds()), Points: &points, Enabled: true,
	})
	require.NoError(t, err)
	require.NoError(t, c.PurchaseSku(ctx, u.ID, groupSku))
	require.Equal(t, group.ID, client.User.GetX(ctx, u.ID).GroupUsers)
	ms := client.GroupMembership.Query().AllX(ctx)
	require.Len(t, ms, 1)
	require.Equal(t, vip.ID, ms[0].GroupID)
	require.NotNil(t, ms[0].Expires)

	// No points price → not purchasable.
	cashOnly, err := c.UpsertSku(ctx, &ent.Sku{
		Name: "cash only", Type: sku.TypeStorage, Amount: 1, Price: 700, Enabled: true,
	})
	require.NoError(t, err)
	require.ErrorIs(t, c.PurchaseSku(ctx, u.ID, cashOnly), ErrSkuNotPurchasable)

	// Insufficient balance → no grant applied.
	balance := client.User.GetX(ctx, u.ID).Credits
	require.NoError(t, c.CreditAdjust(ctx, u.ID, -balance, credittxn.TypeAdjust, "", "drain"))
	require.ErrorIs(t, c.PurchaseSku(ctx, u.ID, storageSku), ErrInsufficientPoints)
	bonus, err = c.StorageBonus(ctx, u.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1024), bonus)
}

func TestPurchaseTrafficSku(t *testing.T) {
	ctx := context.Background()
	client, c := newVasClient(t)
	group, u := vasFixture(t, client)
	require.NoError(t, c.CreditAdjust(ctx, u.ID, 500, credittxn.TypeAdjust, "", "seed"))

	// Start the user with a finite allowance — unlimited users stay unlimited.
	require.NoError(t, client.User.UpdateOne(u).SetDlTraffic(100).Exec(ctx))

	points := int64(200)
	trafficSku, err := c.UpsertSku(ctx, &ent.Sku{
		Name: "1GB traffic", Type: sku.TypeTraffic, Amount: 1024,
		Points: &points, Enabled: true,
	})
	require.NoError(t, err)

	require.NoError(t, c.PurchaseSku(ctx, u.ID, trafficSku))
	u = client.User.GetX(ctx, u.ID)
	require.Equal(t, int64(300), u.Credits)
	require.Equal(t, int64(1124), u.DlTraffic)

	// Unlimited users remain unlimited after purchase.
	u2 := client.User.Create().SetEmail("u2@example.com").SetNick("u2").SetGroup(group).SaveX(ctx)
	require.NoError(t, c.CreditAdjust(ctx, u2.ID, 500, credittxn.TypeAdjust, "", "seed"))
	require.NoError(t, c.PurchaseSku(ctx, u2.ID, trafficSku))
	require.Equal(t, int64(-1), client.User.GetX(ctx, u2.ID).DlTraffic)
}

func TestRedeemTrafficGiftCode(t *testing.T) {
	ctx := context.Background()
	client, c := newVasClient(t)
	group, u := vasFixture(t, client)
	require.NoError(t, client.User.UpdateOne(u).SetDlTraffic(10).Exec(ctx))

	codes, err := c.CreateGiftCodes(ctx, &CreateGiftCodeParams{
		Type: giftcode.TypeTraffic, Amount: 2048, Qty: 2,
	})
	require.NoError(t, err)

	_, err = c.RedeemGiftCode(ctx, u.ID, codes[0].Code)
	require.NoError(t, err)
	require.Equal(t, int64(2058), client.User.GetX(ctx, u.ID).DlTraffic)

	// Unlimited redeemer keeps unlimited balance.
	u2 := client.User.Create().SetEmail("u2@example.com").SetNick("u2").SetGroup(group).SaveX(ctx)
	_, err = c.RedeemGiftCode(ctx, u2.ID, codes[1].Code)
	require.NoError(t, err)
	require.Equal(t, int64(-1), client.User.GetX(ctx, u2.ID).DlTraffic)
}

func paidShareFixture(t *testing.T, client *ent.Client, price int) (*ent.User, *ent.User, *ent.Share) {
	return paidShareFixtureN(t, client, price, 0)
}

func paidShareFixtureN(t *testing.T, client *ent.Client, price, n int) (*ent.User, *ent.User, *ent.Share) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%s-%d", t.Name(), n)
	group := client.Group.Create().SetName("g" + suffix).SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	owner := client.User.Create().SetEmail("owner-" + suffix + "@example.com").SetNick("o" + suffix).SetGroup(group).SaveX(ctx)
	buyer := client.User.Create().SetEmail("buyer-" + suffix + "@example.com").SetNick("b" + suffix).SetGroup(group).SaveX(ctx)
	root := client.File.Create().SetName(RootFolderName).SetType(int(types.FileTypeFolder)).SetOwner(owner).SaveX(ctx)
	file := client.File.Create().SetName("paid" + suffix + ".txt").SetType(int(types.FileTypeFile)).SetOwner(owner).SetParent(root).SaveX(ctx)
	share := client.Share.Create().SetUser(owner).SetFile(file).SetPricePoints(price).SaveX(ctx)
	share.Edges.User = owner
	return owner, buyer, share
}

func TestPurchaseShare(t *testing.T) {
	ctx := context.Background()
	client, c := newVasClient(t)
	owner, buyer, share := paidShareFixture(t, client, 100)
	require.NoError(t, c.CreditAdjust(ctx, buyer.ID, 250, credittxn.TypeAdjust, "", "grant"))

	// 80% commission: buyer pays 100, owner earns 80.
	purchase, err := c.PurchaseShare(ctx, share, buyer.ID, 0.8)
	require.NoError(t, err)
	require.NotEmpty(t, purchase.Ticket)
	require.Equal(t, 100, purchase.Points)
	require.Equal(t, int64(150), client.User.GetX(ctx, buyer.ID).Credits)
	require.Equal(t, int64(80), client.User.GetX(ctx, owner.ID).Credits)

	// Idempotent: second purchase returns the same row without re-debiting.
	again, err := c.PurchaseShare(ctx, share, buyer.ID, 0.8)
	require.NoError(t, err)
	require.Equal(t, purchase.ID, again.ID)
	require.Equal(t, purchase.Ticket, again.Ticket)
	require.Equal(t, int64(150), client.User.GetX(ctx, buyer.ID).Credits)

	// Ticket is scoped to its own share.
	_, err = c.SharePurchaseByTicket(ctx, share.ID, purchase.Ticket)
	require.NoError(t, err)
	_, _, otherShare := paidShareFixtureN(t, client, 100, 1)
	_, err = c.SharePurchaseByTicket(ctx, otherShare.ID, purchase.Ticket)
	require.Error(t, err)
	_, err = c.SharePurchaseByTicket(ctx, share.ID, "bogus-ticket")
	require.Error(t, err)
}

func TestPurchaseShareInsufficient(t *testing.T) {
	ctx := context.Background()
	client, c := newVasClient(t)
	owner, buyer, share := paidShareFixture(t, client, 100)
	require.NoError(t, c.CreditAdjust(ctx, buyer.ID, 50, credittxn.TypeAdjust, "", "grant"))

	_, err := c.PurchaseShare(ctx, share, buyer.ID, 1)
	require.ErrorIs(t, err, ErrInsufficientPoints)
	require.Equal(t, int64(50), client.User.GetX(ctx, buyer.ID).Credits)
	require.Equal(t, int64(0), client.User.GetX(ctx, owner.ID).Credits)
	_, err = c.SharePurchase(ctx, share.ID, buyer.ID)
	require.Error(t, err)
}

func TestPurchaseShareIncomeRates(t *testing.T) {
	ctx := context.Background()
	client, c := newVasClient(t)

	// Zero commission: owner earns nothing, purchase still recorded.
	owner, buyer, share := paidShareFixture(t, client, 40)
	require.NoError(t, c.CreditAdjust(ctx, buyer.ID, 100, credittxn.TypeAdjust, "", "grant"))
	p, err := c.PurchaseShare(ctx, share, buyer.ID, 0)
	require.NoError(t, err)
	require.NotNil(t, p)
	require.Equal(t, int64(0), client.User.GetX(ctx, owner.ID).Credits)

	// Rate above 1 clamps to full price.
	owner2, buyer2, share2 := paidShareFixtureN(t, client, 30, 1)
	require.NoError(t, c.CreditAdjust(ctx, buyer2.ID, 100, credittxn.TypeAdjust, "", "grant"))
	_, err = c.PurchaseShare(ctx, share2, buyer2.ID, 1.5)
	require.NoError(t, err)
	require.Equal(t, int64(30), client.User.GetX(ctx, owner2.ID).Credits)
}
