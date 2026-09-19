package explorer

import (
	"fmt"
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/activity"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager/entitysource"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
)

type (
	EntityDownloadParameterCtx struct{}
	EntityDownloadService      struct {
		Name       string `uri:"name" binding:"required"`
		SpeedLimit int64  `uri:"speed"`
		Src        string `uri:"src"`
	}
)

// Serve serves file content
func (s *EntityDownloadService) Serve(c *gin.Context) error {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	m := manager.NewFileManager(dep, user)
	defer m.Recycle()

	entitySource, err := m.GetEntitySource(c, hashid.FromContext(c))
	if err != nil {
		return fmt.Errorf("failed to get entity source: %w", err)
	}

	defer entitySource.Close()

	// Set cache header for public resource
	settings := dep.SettingProvider()
	maxAge := settings.PublicResourceMaxAge(c)
	c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))

	isDownload := c.Query(routes.IsDownloadQuery) != ""
	isThumb := c.Query(routes.IsThumbQuery) != ""
	entitySource.Serve(c.Writer, c.Request,
		entitysource.WithSpeedLimit(s.SpeedLimit),
		entitysource.WithDownload(isDownload),
		entitysource.WithDisplayName(s.Name),
		entitysource.WithContext(c),
		entitysource.WithThumb(isThumb),
	)
	return nil
}

type (
	SetCurrentVersionParamCtx struct{}
	SetCurrentVersionService  struct {
		Uri     string `json:"uri" uri:"uri" binding:"required"`
		Version string `json:"version" uri:"version" binding:"required"`
	}
)

// Set sets the current version of the file
func (s *SetCurrentVersionService) Set(c *gin.Context) error {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	m := manager.NewFileManager(dep, user)
	defer m.Recycle()

	uri, err := fs.NewUriFromString(s.Uri)
	if err != nil {
		return serializer.NewError(serializer.CodeParamErr, "unknown uri", err)
	}

	versionId, err := dep.HashIDEncoder().Decode(s.Version, hashid.EntityID)
	if err != nil {
		return serializer.NewError(serializer.CodeParamErr, "unknown version id", err)
	}

	if err := m.SetCurrentVersion(c, uri, versionId); err != nil {
		return fmt.Errorf("failed to set current version: %w", err)
	}

	opts := []activity.Opt{activity.Extra(map[string]any{"version": versionId})}
	if target, err := m.Get(c, uri); err == nil {
		opts = append(opts, activity.File(target.ID()))
	}
	activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventSetCurrentVersion, opts...)

	return nil
}

type (
	DeleteVersionParamCtx struct{}
	DeleteVersionService  struct {
		Uri     string `json:"uri" uri:"uri" binding:"required"`
		Version string `json:"version" uri:"version" binding:"required"`
	}
)

// Delete deletes the version of the file
func (s *DeleteVersionService) Delete(c *gin.Context) error {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	m := manager.NewFileManager(dep, user)
	defer m.Recycle()

	uri, err := fs.NewUriFromString(s.Uri)
	if err != nil {
		return serializer.NewError(serializer.CodeParamErr, "unknown uri", err)
	}

	versionId, err := dep.HashIDEncoder().Decode(s.Version, hashid.EntityID)
	if err != nil {
		return serializer.NewError(serializer.CodeParamErr, "unknown version id", err)
	}

	if err := m.DeleteVersion(c, uri, versionId); err != nil {
		return fmt.Errorf("failed to delete version: %w", err)
	}

	opts := []activity.Opt{activity.Extra(map[string]any{"version": versionId})}
	if target, err := m.Get(c, uri); err == nil {
		opts = append(opts, activity.File(target.ID()))
	}
	activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventDeleteVersion, opts...)

	return nil
}
