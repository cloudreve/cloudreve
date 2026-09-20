package explorer

import (
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/activity"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/samber/lo"
)

type (
	// ListTagsService lists all tag metadata keys used by the current user's files.
	ListTagsService struct{}

	ListTagsParameterCtx struct{}

	// TagResponse is one managed tag entry, name without the "tag:" prefix.
	TagResponse struct {
		Name  string `json:"name"`
		Color string `json:"color"`
		Count int    `json:"file_count"`
	}

	// PatchTagService renames and/or recolors a tag across all of the
	// current user's files. Renaming onto an existing tag merges them.
	PatchTagService struct {
		Name    string  `json:"name" binding:"required,min=1,max=64"`
		NewName string  `json:"new_name" binding:"omitempty,min=1,max=64"`
		Color   *string `json:"color"`
	}

	PatchTagParameterCtx struct{}

	// DeleteTagService removes a tag from all of the current user's files.
	DeleteTagService struct {
		Name string `json:"name" binding:"required,min=1,max=64"`
	}

	DeleteTagParameterCtx struct{}
)

func (s *ListTagsService) Get(c *gin.Context) (any, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)

	stats, err := dep.FileClient().ListMetadataStats(c, user.ID, manager.TagMetadataPrefix)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list tags", err)
	}

	return lo.Map(stats, func(item *inventory.MetadataNameStat, _ int) *TagResponse {
		return &TagResponse{
			Name:  strings.TrimPrefix(item.Name, manager.TagMetadataPrefix),
			Color: item.Value,
			Count: item.Count,
		}
	}), nil
}

func (s *PatchTagService) Patch(c *gin.Context) error {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)

	newName := s.Name
	if s.NewName != "" {
		newName = s.NewName
	}
	if strings.ContainsAny(s.Name, ":") || strings.ContainsAny(newName, ":") {
		return serializer.NewError(serializer.CodeParamErr, "tag name cannot contain ':'", nil)
	}
	if s.Color != nil {
		if err := validator.New().Var(*s.Color, "iscolor"); err != nil {
			return serializer.NewError(serializer.CodeParamErr, "invalid color", err)
		}
	}
	if err := dep.FileClient().RenameMetadataName(c, user.ID,
		manager.TagMetadataPrefix+s.Name, manager.TagMetadataPrefix+newName, s.Color); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to update tag", err)
	}

	activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventUpdateMetadata,
		activity.Extra(map[string]any{"tag": s.Name, "new_tag": newName}))
	return nil
}

func (s *DeleteTagService) Delete(c *gin.Context) error {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)

	if strings.Contains(s.Name, ":") {
		return serializer.NewError(serializer.CodeParamErr, "tag name cannot contain ':'", nil)
	}
	if err := dep.FileClient().DeleteMetadataByNameForOwner(c, user.ID, manager.TagMetadataPrefix+s.Name); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to delete tag", err)
	}

	activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventUpdateMetadata,
		activity.Extra(map[string]any{"tag": s.Name, "removed": true}))
	return nil
}
