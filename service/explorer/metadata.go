package explorer

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/activity"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
)

type (
	PatchMetadataService struct {
		Uris    []string           `json:"uris" binding:"required"`
		Patches []fs.MetadataPatch `json:"patches" binding:"required,dive"`
	}

	PatchMetadataParameterCtx struct{}
)

func (s *PatchMetadataService) GetUris() []string {
	return s.Uris
}

func (s *PatchMetadataService) Patch(c *gin.Context) error {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	m := manager.NewFileManager(dep, user)
	defer m.Recycle()

	uris, err := fs.NewUriFromStrings(s.Uris...)
	if err != nil {
		return serializer.NewError(serializer.CodeParamErr, "unknown uri", err)
	}

	if err := m.PatchMedata(c, uris, s.Patches...); err != nil {
		return err
	}

	keys := make([]string, 0, len(s.Patches))
	for _, p := range s.Patches {
		keys = append(keys, p.Key)
	}
	for _, uri := range uris {
		opts := []activity.Opt{activity.Extra(map[string]any{"keys": keys})}
		if target, err := m.Get(c, uri); err == nil {
			opts = append(opts, activity.File(target.ID()))
		}
		activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventUpdateMetadata, opts...)
	}

	return nil
}
