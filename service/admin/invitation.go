package admin

import (
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

type (
	// InvitationCodeListService lists invitation codes for admins.
	InvitationCodeListService struct {
		Keyword   string `form:"keyword" json:"keyword"`
		PageSize  int    `form:"page_size" json:"page_size" binding:"required,min=1"`
		PageToken string `form:"page_token" json:"page_token"`
	}
	InvitationCodeListParamCtx struct{}

	// UpsertInvitationCodeService creates an invitation code. An empty Code
	// auto-generates one.
	UpsertInvitationCodeService struct {
		Code      string     `json:"code" binding:"omitempty,min=4,max=64"`
		GroupID   int        `json:"group_id" binding:"omitempty,min=0"`
		MaxUses   int        `json:"max_uses" binding:"omitempty,min=0"`
		ExpiresAt *time.Time `json:"expires_at"`
	}
	UpsertInvitationCodeParamCtx struct{}

	// SingleInvitationCodeService targets one code by ID.
	SingleInvitationCodeService struct {
		ID int `uri:"id" json:"id" binding:"required"`
	}
	SingleInvitationCodeParamCtx struct{}

	InvitationCodeResponse struct {
		*ent.InvitationCode
		HashID string `json:"hash_id"`
	}
)

func (service *InvitationCodeListService) List(c *gin.Context) (*ListInvitationCodeResult, error) {
	dep := dependency.FromContext(c)
	res, err := dep.InvitationCodeClient().List(c, &inventory.ListInvitationCodeArgs{
		PaginationArgs: &inventory.PaginationArgs{
			PageSize:  service.PageSize,
			PageToken: service.PageToken,
		},
		Keyword: service.Keyword,
	})
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list invitation codes", err)
	}

	return &ListInvitationCodeResult{
		Pagination: res.PaginationResults,
		Codes: lo.Map(res.Codes, func(code *ent.InvitationCode, _ int) *InvitationCodeResponse {
			return &InvitationCodeResponse{
				InvitationCode: code,
				HashID:         hashid.EncodeInvitationCodeID(dep.HashIDEncoder(), code.ID),
			}
		}),
	}, nil
}

func (service *UpsertInvitationCodeService) Create(c *gin.Context) (*InvitationCodeResponse, error) {
	dep := dependency.FromContext(c)

	code := service.Code
	if code == "" {
		code = util.RandString(16, util.RandomLowerCases)
	}

	if service.GroupID > 0 {
		if _, err := dep.GroupClient().GetByID(c, service.GroupID); err != nil {
			return nil, serializer.NewError(serializer.CodeParamErr, "Invalid group", err)
		}
	}

	record, err := dep.InvitationCodeClient().Create(c, &inventory.CreateInvitationCodeParams{
		Code:      code,
		GroupID:   service.GroupID,
		MaxUses:   service.MaxUses,
		ExpiresAt: service.ExpiresAt,
	})
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to create invitation code", err)
	}

	return &InvitationCodeResponse{
		InvitationCode: record,
		HashID:         hashid.EncodeInvitationCodeID(dep.HashIDEncoder(), record.ID),
	}, nil
}

func (service *SingleInvitationCodeService) Delete(c *gin.Context) error {
	dep := dependency.FromContext(c)
	if err := dep.InvitationCodeClient().Delete(c, service.ID); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to delete invitation code", err)
	}
	return nil
}
