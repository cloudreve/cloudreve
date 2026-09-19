package explorer

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

type (
	// EventResponse is one row in an activity feed.
	EventResponse struct {
		ID        string         `json:"id"`
		Type      int            `json:"type"`
		ActorID   string         `json:"actor_id,omitempty"`
		ActorName string         `json:"actor_name,omitempty"`
		IP        string         `json:"ip,omitempty"`
		CID       string         `json:"cid,omitempty"`
		FileID    string         `json:"file_id,omitempty"`
		ShareID   string         `json:"share_id,omitempty"`
		Extra     map[string]any `json:"extra,omitempty"`
		CreatedAt int64          `json:"created_at"`
	}

	// FileActivityService lists audit events for a file the caller owns.
	FileActivityService struct {
		Uri      string `form:"uri" binding:"required"`
		Page     int    `form:"page" binding:"min=1"`
		PageSize int    `form:"page_size" binding:"min=1,max=100"`
	}
	FileActivityParamCtx struct{}

	FileActivityResponse struct {
		Events []*EventResponse `json:"events"`
		Total  int              `json:"total"`
	}
)

// BuildEventResponse serializes one audit event with actor names resolved
// from the batch-loaded map.
func BuildEventResponse(e *ent.ActivityEvent, actors map[int]*ent.User, hasher hashid.Encoder) *EventResponse {
	res := &EventResponse{
		ID:        hashid.EncodeAuditLogID(hasher, e.ID),
		Type:      e.Type,
		IP:        e.IP,
		CID:       e.Cid,
		Extra:     e.Extra,
		CreatedAt: e.CreatedAt.Unix(),
	}
	if e.ActorID > 0 {
		res.ActorID = hashid.EncodeUserID(hasher, e.ActorID)
		if actor, ok := actors[e.ActorID]; ok {
			res.ActorName = actor.Nick
			if res.ActorName == "" {
				res.ActorName = actor.Email
			}
		}
	}
	if e.FileID > 0 {
		res.FileID = hashid.EncodeFileID(hasher, e.FileID)
	}
	if e.ShareID > 0 {
		res.ShareID = hashid.EncodeShareID(hasher, e.ShareID)
	}
	return res
}

func (s *FileActivityService) Get(c *gin.Context) (*FileActivityResponse, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	m := manager.NewFileManager(dep, user)
	defer m.Recycle()

	uri, err := fs.NewUriFromString(s.Uri)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "unknown uri", err)
	}

	file, err := m.Get(c, uri, dbfs.WithNotRoot())
	if err != nil {
		return nil, err
	}
	dbFile, ok := file.(*dbfs.File)
	if !ok || dbFile.Model.OwnerID != user.ID {
		return nil, serializer.NewError(serializer.CodeNoPermissionErr, "Only the owner can view activity", nil)
	}

	pageSize := s.PageSize
	if pageSize == 0 {
		pageSize = 20
	}
	events, total, err := dep.ActivityClient().List(c, &inventory.ListActivityArgs{
		PaginationArgs: inventory.PaginationArgs{
			Page:     max(s.Page-1, 0),
			PageSize: pageSize,
		},
		FileID: dbFile.Model.ID,
	})
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list activity", err)
	}

	actorIDs := lo.Uniq(lo.FilterMap(events, func(e *ent.ActivityEvent, _ int) (int, bool) {
		return e.ActorID, e.ActorID > 0
	}))
	actors, err := dep.UserClient().ListByIDs(c, actorIDs)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to resolve actors", err)
	}

	hasher := dep.HashIDEncoder()
	return &FileActivityResponse{
		Events: lo.Map(events, func(e *ent.ActivityEvent, _ int) *EventResponse {
			return BuildEventResponse(e, actors, hasher)
		}),
		Total: total,
	}, nil
}
