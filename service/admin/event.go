package admin

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/service/explorer"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

type (
	// EventListService lists audit events for the admin feed.
	EventListService struct {
		Page     int `form:"page" binding:"required,min=1"`
		PageSize int `form:"page_size" binding:"required,min=1,max=200"`
		Type     int `form:"type" binding:"omitempty,min=0"`
		ActorID  int `form:"actor_id" binding:"omitempty,min=0"`
		FileID   int `form:"file_id" binding:"omitempty,min=0"`
		ShareID  int `form:"share_id" binding:"omitempty,min=0"`
	}
	EventListParamCtx struct{}

	EventListResponse struct {
		Events []*explorer.EventResponse `json:"events"`
		Total  int                       `json:"total"`
	}
)

func (s *EventListService) Get(c *gin.Context) (*EventListResponse, error) {
	dep := dependency.FromContext(c)

	var typeFilter *int
	if s.Type > 0 {
		typeFilter = &s.Type
	}

	events, total, err := dep.ActivityClient().List(c, &inventory.ListActivityArgs{
		PaginationArgs: inventory.PaginationArgs{
			Page:     s.Page - 1,
			PageSize: s.PageSize,
		},
		Type:    typeFilter,
		FileID:  s.FileID,
		ActorID: s.ActorID,
		ShareID: s.ShareID,
	})
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list events", err)
	}

	actorIDs := lo.Uniq(lo.FilterMap(events, func(e *ent.ActivityEvent, _ int) (int, bool) {
		return e.ActorID, e.ActorID > 0
	}))
	actors, err := dep.UserClient().ListByIDs(c, actorIDs)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to resolve actors", err)
	}

	hasher := dep.HashIDEncoder()
	return &EventListResponse{
		Events: lo.Map(events, func(e *ent.ActivityEvent, _ int) *explorer.EventResponse {
			return explorer.BuildEventResponse(e, actors, hasher)
		}),
		Total: total,
	}, nil
}
