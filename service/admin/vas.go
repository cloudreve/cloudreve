package admin

import (
	"errors"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/credittxn"
	"github.com/cloudreve/Cloudreve/v4/ent/giftcode"
	"github.com/cloudreve/Cloudreve/v4/ent/sku"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/activity"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
)

type (
	// SkuListService lists all products for admins.
	SkuListService  struct{}
	SkuListParamCtx struct{}

	// SkuUpsertService creates or updates one product. ID zero creates.
	SkuUpsertService struct {
		Sku *ent.Sku `json:"sku" binding:"required"`
	}
	SkuUpsertParamCtx struct{}

	// SingleSkuService targets one product by ID.
	SingleSkuService struct {
		ID int `uri:"id" json:"id" binding:"required"`
	}
	SingleSkuParamCtx struct{}

	// GiftCodeListService lists gift codes for admins.
	GiftCodeListService struct {
		Page     int `form:"page" json:"page" binding:"required,min=1"`
		PageSize int `form:"page_size" json:"page_size" binding:"required,min=1,max=200"`
	}
	GiftCodeListParamCtx struct{}

	// CreateGiftCodeService generates qty codes sharing one grant definition.
	CreateGiftCodeService struct {
		Type     giftcode.Type `json:"type" binding:"required,oneof=points storage group traffic"`
		Amount   int64         `json:"amount" binding:"required,min=1"`
		Duration int64         `json:"duration" binding:"omitempty,min=0"`
		Qty      int           `json:"qty" binding:"required,min=1,max=500"`
		Des      string        `json:"des" binding:"omitempty,max=255"`
	}
	CreateGiftCodeParamCtx struct{}

	// SingleGiftCodeService targets one code by ID.
	SingleGiftCodeService struct {
		ID int `uri:"id" json:"id" binding:"required"`
	}
	SingleGiftCodeParamCtx struct{}

	// AdjustCreditService performs a manual balance adjustment on a user.
	AdjustCreditService struct {
		Email string `json:"email" binding:"required,email"`
		Delta int64  `json:"delta" binding:"required"`
		Des   string `json:"des" binding:"omitempty,max=255"`
	}
	AdjustCreditParamCtx struct{}

	GiftCodeListResponse struct {
		Codes []*ent.GiftCode `json:"codes"`
		Total int             `json:"total"`
	}
)

func (service *GiftCodeListService) List(c *gin.Context) (*GiftCodeListResponse, error) {
	dep := dependency.FromContext(c)
	codes, total, err := dep.VasClient().ListGiftCodes(c, service.Page, service.PageSize)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list gift codes", err)
	}

	return &GiftCodeListResponse{Codes: codes, Total: total}, nil
}

func (service *CreateGiftCodeService) Create(c *gin.Context) ([]*ent.GiftCode, error) {
	dep := dependency.FromContext(c)

	if service.Type == giftcode.TypeGroup {
		if _, err := dep.GroupClient().GetByID(c, int(service.Amount)); err != nil {
			return nil, serializer.NewError(serializer.CodeParamErr, "Invalid target group", err)
		}
	}

	codes, err := dep.VasClient().CreateGiftCodes(c, &inventory.CreateGiftCodeParams{
		Type:     service.Type,
		Amount:   service.Amount,
		Duration: time.Duration(service.Duration) * time.Second,
		Des:      service.Des,
		Qty:      service.Qty,
	})
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to create gift codes", err)
	}

	return codes, nil
}

func (service *SingleGiftCodeService) Delete(c *gin.Context) error {
	dep := dependency.FromContext(c)
	if err := dep.VasClient().DeleteGiftCodes(c, []int{service.ID}); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to delete gift code", err)
	}
	return nil
}

func (service *SkuListService) List(c *gin.Context) ([]*ent.Sku, error) {
	skus, err := dependency.FromContext(c).VasClient().ListSkus(c, false)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list products", err)
	}
	return skus, nil
}

func (service *SkuUpsertService) Update(c *gin.Context) (*ent.Sku, error) {
	return upsertSku(c, service.Sku)
}

func (service *SkuUpsertService) Create(c *gin.Context) (*ent.Sku, error) {
	return upsertSku(c, service.Sku)
}

func upsertSku(c *gin.Context, s *ent.Sku) (*ent.Sku, error) {
	dep := dependency.FromContext(c)

	if s.Name == "" || s.Amount <= 0 || s.Duration < 0 || s.Price < 0 || s.Weight < 0 {
		return nil, serializer.NewError(serializer.CodeParamErr, "Invalid product fields", nil)
	}
	if s.Points != nil && *s.Points <= 0 {
		return nil, serializer.NewError(serializer.CodeParamErr, "Invalid points price", nil)
	}
	if s.Type == sku.TypeGroup {
		if _, err := dep.GroupClient().GetByID(c, int(s.Amount)); err != nil {
			return nil, serializer.NewError(serializer.CodeParamErr, "Invalid target group", err)
		}
	}

	res, err := dep.VasClient().UpsertSku(c, s)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to save product", err)
	}
	return res, nil
}

func (service *SingleSkuService) Delete(c *gin.Context) error {
	if err := dependency.FromContext(c).VasClient().DeleteSkus(c, []int{service.ID}); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to delete product", err)
	}
	return nil
}

func (service *AdjustCreditService) Create(c *gin.Context) error {
	dep := dependency.FromContext(c)

	target, err := dep.UserClient().GetByEmail(c, service.Email)
	if err != nil {
		return serializer.NewError(serializer.CodeNotFound, "User not found", err)
	}

	if err := dep.VasClient().CreditAdjust(c, target.ID, service.Delta, credittxn.TypeAdjust, "", service.Des); err != nil {
		if errors.Is(err, inventory.ErrInsufficientPoints) {
			return serializer.NewError(serializer.CodeParamErr, "Adjustment would overdraw the balance", err)
		}
		return serializer.NewError(serializer.CodeDBError, "Failed to adjust credits", err)
	}

	activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventPointsChange,
		activity.Extra(map[string]any{"target_user": target.ID, "delta": service.Delta, "des": service.Des}))
	return nil
}
