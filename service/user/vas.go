package user

import (
	"errors"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/credittxn"
	"github.com/cloudreve/Cloudreve/v4/ent/sku"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/activity"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/gin-gonic/gin"
)

type (
	// CreditService returns the caller's credit balance and active grants.
	CreditService  struct{}
	CreditParamCtx struct{}

	// CreditTxnListService lists the caller's credit ledger.
	CreditTxnListService struct {
		Page     int `form:"page" json:"page" binding:"required,min=1"`
		PageSize int `form:"page_size" json:"page_size" binding:"required,min=1,max=100"`
	}
	CreditTxnListParamCtx struct{}

	// RedeemGiftCodeService redeems a gift code for the caller.
	RedeemGiftCodeService struct {
		Code string `json:"code" form:"code" binding:"required,min=4,max=64"`
	}
	RedeemGiftCodeParamCtx struct{}

	// SkuListService lists enabled products for the shop page.
	SkuListService  struct{}
	SkuListParamCtx struct{}

	// PurchaseSkuService buys a product with credit points.
	PurchaseSkuService struct {
		Sku string `json:"sku" form:"sku" binding:"required,max=64"`
	}
	PurchaseSkuParamCtx struct{}

	// SkuResponse is the public product view; amount carries the target
	// group hashid for membership products, bytes for storage packs.
	SkuResponse struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Type     string `json:"type"`
		Amount   int64  `json:"amount"`
		Group    string `json:"group,omitempty"`
		GroupID  string `json:"group_id,omitempty"`
		Duration int64  `json:"duration"`
		Price    int64  `json:"price"`
		Points   *int64 `json:"points,omitempty"`
		Label    string `json:"label,omitempty"`
		Des      string `json:"des,omitempty"`
	}

	CreditResponse struct {
		Credits      int64            `json:"credits"`
		StorageBonus int64            `json:"storage_bonus"`
		// DlTraffic is the remaining direct-link allowance in bytes;
		// -1 means unlimited.
		DlTraffic int64            `json:"dl_traffic"`
		Grants    []*ent.UserGrant `json:"grants"`
	}

	CreditTxnListResponse struct {
		Txns  []*ent.CreditTxn `json:"txns"`
		Total int              `json:"total"`
	}
)

func (service *CreditService) Get(c *gin.Context) (*CreditResponse, error) {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)

	// Re-read the user: balances may have just changed (e.g. after a
	// purchase) while the context still carries the pre-request snapshot.
	if fresh, err := dep.UserClient().GetByID(c, u.ID); err == nil {
		u = fresh
	}

	bonus, err := dep.VasClient().StorageBonus(c, u.ID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get storage bonus", err)
	}

	grants, err := dep.VasClient().ListGrants(c, u.ID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list grants", err)
	}

	return &CreditResponse{
		Credits:      u.Credits,
		StorageBonus: bonus,
		DlTraffic:    u.DlTraffic,
		Grants:       grants,
	}, nil
}

func (service *CreditTxnListService) List(c *gin.Context) (*CreditTxnListResponse, error) {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)

	txns, total, err := dep.VasClient().ListCreditTxns(c, u.ID, service.Page, service.PageSize)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list credit transactions", err)
	}

	return &CreditTxnListResponse{Txns: txns, Total: total}, nil
}

func (service *RedeemGiftCodeService) Create(c *gin.Context) (*ent.GiftCode, error) {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)

	gc, err := dep.VasClient().RedeemGiftCode(c, u.ID, service.Code)
	if err != nil {
		switch {
		case errors.Is(err, inventory.ErrGiftCodeNotFound):
			return nil, serializer.NewError(serializer.CodeNotFound, "Invalid gift code", err)
		case errors.Is(err, inventory.ErrGiftCodeUsed):
			return nil, serializer.NewError(serializer.CodeConflict, "Gift code already used", err)
		default:
			return nil, serializer.NewError(serializer.CodeDBError, "Failed to redeem gift code", err)
		}
	}

	activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventRedeemGiftCode,
		activity.Extra(map[string]any{"code": gc.Code, "type": string(gc.Type), "amount": gc.Amount}))
	return gc, nil
}

func (service *SkuListService) List(c *gin.Context) ([]*SkuResponse, error) {
	dep := dependency.FromContext(c)

	skus, err := dep.VasClient().ListSkus(c, true)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list products", err)
	}

	lang := ""
	if u := inventory.UserFromContext(c); u != nil {
		lang = u.Settings.Language
	}
	if lang == "" {
		lang = setting.ParseAcceptLanguage(c.GetHeader("Accept-Language"))
	}

	res := make([]*SkuResponse, 0, len(skus))
	for _, s := range skus {
		r := &SkuResponse{
			ID:       hashid.EncodeSkuID(dep.HashIDEncoder(), s.ID),
			Name:     setting.MatchLanguage(s.NameI18n, lang, s.Name),
			Type:     string(s.Type),
			Amount:   s.Amount,
			Duration: s.Duration,
			Price:    s.Price,
			Points:   s.Points,
			Label:    s.Label,
			Des:      setting.MatchLanguage(s.DesI18n, lang, s.Des),
		}
		if s.Type == sku.TypeGroup {
			if g, err := dep.GroupClient().GetByID(c, int(s.Amount)); err == nil {
				r.Group = g.Name
				r.GroupID = hashid.EncodeGroupID(dep.HashIDEncoder(), g.ID)
			}
		}
		res = append(res, r)
	}
	return res, nil
}

func (service *PurchaseSkuService) Create(c *gin.Context) (*CreditResponse, error) {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)

	skuID, err := dep.HashIDEncoder().Decode(service.Sku, hashid.SkuID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "Invalid product", err)
	}
	s, err := dep.VasClient().GetSku(c, skuID)
	if err != nil || !s.Enabled {
		return nil, serializer.NewError(serializer.CodeNotFound, "Product not found", err)
	}

	if err := dep.VasClient().PurchaseSku(c, u.ID, s); err != nil {
		switch {
		case errors.Is(err, inventory.ErrSkuNotPurchasable):
			return nil, serializer.NewError(serializer.CodeParamErr, "Product cannot be purchased with points", err)
		case errors.Is(err, inventory.ErrInsufficientPoints):
			return nil, serializer.NewError(serializer.CodeParamErr, "Insufficient credit balance", err)
		default:
			return nil, serializer.NewError(serializer.CodeDBError, "Failed to purchase product", err)
		}
	}

	activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventPointsChange,
		activity.Extra(map[string]any{"sku": s.ID, "name": s.Name, "delta": -*s.Points}))

	return (&CreditService{}).Get(c)
}

// TxnDescription maps ledger types to stable English descriptors consumed by
// the Finance settings tab.
func TxnDescription(t *ent.CreditTxn) string {
	switch t.Type {
	case credittxn.TypePurchase:
		return "Shop purchase"
	case credittxn.TypeGift:
		return "Gift code"
	case credittxn.TypeShareIncome:
		return "Share link purchased"
	case credittxn.TypeAdjust:
		return "Manual adjustment"
	default:
		return string(t.Type)
	}
}
