package user

import (
	"errors"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/credittxn"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/activity"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
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

	CreditResponse struct {
		Credits      int64            `json:"credits"`
		StorageBonus int64            `json:"storage_bonus"`
		Grants       []*ent.UserGrant `json:"grants"`
	}

	CreditTxnListResponse struct {
		Txns  []*ent.CreditTxn `json:"txns"`
		Total int              `json:"total"`
	}
)

func (service *CreditService) Get(c *gin.Context) (*CreditResponse, error) {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)

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
