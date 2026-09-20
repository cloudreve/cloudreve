package share

import (
	"context"
	"errors"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/activity"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
)

type (
	// SharePurchaseService purchases a paid share with the caller's credit
	// balance and returns the resume ticket.
	SharePurchaseService  struct{}
	SharePurchaseParamCtx struct{}
	SharePurchaseResponse struct {
		Ticket string `json:"ticket"`
	}
)

func (s *SharePurchaseService) Purchase(c *gin.Context) (*SharePurchaseResponse, error) {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)

	ctx := context.WithValue(c, inventory.LoadShareUser{}, true)
	ctx = context.WithValue(ctx, inventory.LoadShareFile{}, true)
	share, err := dep.ShareClient().GetByID(ctx, hashid.FromContext(c))
	if err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "Share not found", err)
	}

	if err := inventory.IsValidShare(share); err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "Share link expired", err)
	}
	if share.PricePoints <= 0 {
		return nil, serializer.NewError(serializer.CodeParamErr, "Share is free", nil)
	}
	if share.Edges.User.ID == u.ID {
		return nil, serializer.NewError(serializer.CodeParamErr, "Cannot purchase own share", nil)
	}
	if !inventory.EffectiveGroup(u).Permissions.Enabled(int(types.GroupPermissionShareDownload)) {
		return nil, serializer.NewError(serializer.CodeNoPermissionErr, "You don't have permission to access share links", nil)
	}

	purchase, err := dep.VasClient().PurchaseShare(ctx, share, u.ID, dep.SettingProvider().ShareScoreRate(c))
	if err != nil {
		switch {
		case errors.Is(err, inventory.ErrInsufficientPoints):
			return nil, serializer.NewError(serializer.CodeParamErr, "Insufficient credit balance", err)
		default:
			return nil, serializer.NewError(serializer.CodeDBError, "Failed to purchase share", err)
		}
	}

	activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventPaymentFulfilled,
		activity.Share(share.ID), activity.File(share.Edges.File.ID),
		activity.Extra(map[string]any{
			"points": purchase.Points,
			"seller": share.Edges.User.ID,
		}))

	return &SharePurchaseResponse{Ticket: purchase.Ticket}, nil
}
