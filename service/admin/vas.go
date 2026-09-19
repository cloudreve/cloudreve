package admin

import (
	"errors"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/credittxn"
	"github.com/cloudreve/Cloudreve/v4/ent/giftcode"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
)

type (
	// GiftCodeListService lists gift codes for admins.
	GiftCodeListService struct {
		Page     int `form:"page" json:"page" binding:"required,min=1"`
		PageSize int `form:"page_size" json:"page_size" binding:"required,min=1,max=200"`
	}
	GiftCodeListParamCtx struct{}

	// CreateGiftCodeService generates qty codes sharing one grant definition.
	CreateGiftCodeService struct {
		Type     giftcode.Type `json:"type" binding:"required,oneof=points storage group"`
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

	return nil
}
