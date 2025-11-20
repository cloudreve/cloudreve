package explorer

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver/local"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager/entitysource"
	"github.com/cloudreve/Cloudreve/v4/pkg/mediameta"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

// SlaveDownloadService 从机文件下載服务
type SlaveDownloadService struct {
	PathEncoded string `uri:"path" binding:"required"`
	Name        string `uri:"name" binding:"required"`
	Speed       int    `uri:"speed" binding:"min=0"`
}

// SlaveFileService 从机单文件文件相关服务
type SlaveFileService struct {
	PathEncoded string `uri:"path" binding:"required"`
	Ext         string `uri:"ext"`
}

// SlaveFilesService 从机多文件相关服务
type SlaveFilesService struct {
	Files []string `json:"files" binding:"required,gt=0"`
}

// SlaveServe serves file content
func (s *EntityDownloadService) SlaveServe(c *gin.Context) error {
	dep := dependency.FromContext(c)
	m := manager.NewFileManager(dep, nil)
	defer m.Recycle()

	src, err := base64.URLEncoding.DecodeString(s.Src)
	if err != nil {
		return fmt.Errorf("failed to decode src: %w", err)
	}

	entity, err := local.NewLocalFileEntity(types.EntityTypeVersion, string(src))
	if err != nil {
		return fs.ErrPathNotExist.WithError(err)
	}

	entitySource, err := m.GetEntitySource(c, 0, fs.WithEntity(entity))
	if err != nil {
		return fmt.Errorf("failed to get entity source: %w", err)
	}

	defer entitySource.Close()

	// Set cache header for public resource
	settings := dep.SettingProvider()
	maxAge := settings.PublicResourceMaxAge(c)
	c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))

	isDownload := c.Query(routes.IsDownloadQuery) != ""
	entitySource.Serve(c.Writer, c.Request,
		entitysource.WithSpeedLimit(s.SpeedLimit),
		entitysource.WithDownload(isDownload),
		entitysource.WithDisplayName(s.Name),
		entitysource.WithContext(c),
	)
	return nil
}

type (
	SlaveCreateUploadSessionParamCtx struct{}
	// SlaveCreateUploadSessionService 从机上传会话服务
	SlaveCreateUploadSessionService struct {
		Session   fs.UploadSession `json:"session" binding:"required"`
		Overwrite bool             `json:"overwrite"`
	}
)

// Create 从机创建上传会话
func (service *SlaveCreateUploadSessionService) Create(c *gin.Context) error {
	mode := fs.ModeNone
	if service.Overwrite {
		mode = fs.ModeOverwrite
	}

	req := &fs.UploadRequest{
		Mode:  mode,
		Props: service.Session.Props.Copy(),
	}

	dep := dependency.FromContext(c)
	m := manager.NewFileManager(dep, nil)
	_, err := m.CreateUploadSession(c, req, fs.WithUploadSession(&service.Session))
	if err != nil {
		return serializer.NewError(serializer.CodeCacheOperation, "Failed to create upload session in slave node", err)
	}

	return nil
}

type (
	SlaveMetaParamCtx struct{}
	SlaveMetaService  struct {
		Src string `uri:"src" binding:"required"`
		Ext string `uri:"ext" binding:"required"`
	}
)

// MediaMeta retrieves media metadata
func (s *SlaveMetaService) MediaMeta(c *gin.Context) ([]driver.MediaMeta, error) {
	dep := dependency.FromContext(c)
	m := manager.NewFileManager(dep, nil)
	defer m.Recycle()

	src, err := base64.URLEncoding.DecodeString(s.Src)
	if err != nil {
		return nil, fmt.Errorf("failed to decode src: %w", err)
	}

	entity, err := local.NewLocalFileEntity(types.EntityTypeVersion, string(src))
	if err != nil {
		return nil, fs.ErrPathNotExist.WithError(err)
	}

	entitySource, err := m.GetEntitySource(c, 0, fs.WithEntity(entity))
	if err != nil {
		return nil, fmt.Errorf("failed to get entity source: %w", err)
	}
	defer entitySource.Close()

	language := c.Query("language")
	extractor := dep.MediaMetaExtractor(c)
	res, err := extractor.Extract(c, s.Ext, entitySource, mediameta.WithLanguage(language))
	if err != nil {
		return nil, fmt.Errorf("failed to extract media meta: %w", err)
	}

	return res, nil
}

type (
	SlaveThumbParamCtx struct{}
	SlaveThumbService  struct {
		Src string `uri:"src" binding:"required"`
		Ext string `uri:"ext" binding:"required"`
	}
)

func (s *SlaveThumbService) Thumb(c *gin.Context) error {
	dep := dependency.FromContext(c)
	m := manager.NewFileManager(dep, nil)
	defer m.Recycle()

	src, err := base64.URLEncoding.DecodeString(s.Src)
	if err != nil {
		return fmt.Errorf("failed to decode src: %w", err)
	}

	settings := dep.SettingProvider()
	var entity fs.Entity
	entity, err = local.NewLocalFileEntity(types.EntityTypeThumbnail, string(src)+settings.ThumbSlaveSidecarSuffix(c))
	if err != nil {
		srcEntity, err := local.NewLocalFileEntity(types.EntityTypeVersion, string(src))
		if err != nil {
			return fs.ErrPathNotExist.WithError(err)
		}

		entity, err = m.SubmitAndAwaitThumbnailTask(c, nil, s.Ext, srcEntity)
		if err != nil {
			return fmt.Errorf("failed to submit and await thumbnail task: %w", err)
		}
	}

	entitySource, err := m.GetEntitySource(c, 0, fs.WithEntity(entity))
	if err != nil {
		return fmt.Errorf("failed to get thumb entity source: %w", err)
	}

	defer entitySource.Close()

	// Set cache header for public resource
	maxAge := settings.PublicResourceMaxAge(c)
	c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))

	entitySource.Serve(c.Writer, c.Request,
		entitysource.WithContext(c),
	)
	return nil
}

type (
	SlaveDeleteUploadSessionParamCtx struct{}
	SlaveDeleteUploadSessionService  struct {
		ID string `uri:"sessionId" binding:"required"`
	}
)

// Delete deletes an upload session from slave node
func (service *SlaveDeleteUploadSessionService) Delete(c *gin.Context) error {
	dep := dependency.FromContext(c)
	m := manager.NewFileManager(dep, nil)
	defer m.Recycle()

	err := m.CancelUploadSession(c, nil, service.ID)
	if err != nil {
		return fmt.Errorf("slave failed to delete upload session: %w", err)
	}

	return nil
}

type (
	SlaveDeleteFileParamCtx struct{}
	SlaveDeleteFileService  struct {
		Files []string `json:"files" binding:"required,gt=0"`
	}
)

func (service *SlaveDeleteFileService) Delete(c *gin.Context) ([]string, error) {
	dep := dependency.FromContext(c)
	m := manager.NewFileManager(dep, nil)
	defer m.Recycle()
	d := m.LocalDriver(nil)

	// Try to delete thumbnail sidecar
	sidecarSuffix := dep.SettingProvider().ThumbSlaveSidecarSuffix(c)
	failed, err := d.Delete(c, lo.Map(service.Files, func(item string, index int) string {
		return item + sidecarSuffix
	})...)
	if err != nil {
		dep.Logger().Warning("Failed to delete thumbnail sidecar [%s]: %s", strings.Join(failed, ", "), err)
	}

	failed, err = d.Delete(c, service.Files...)
	if err != nil {
		return failed, fmt.Errorf("slave failed to delete file: %w", err)
	}

	return nil, nil
}

type (
	SlaveListParamCtx struct{}
	SlaveListService  struct {
		Path      string `form:"path" binding:"required"`
		Recursive bool   `form:"recursive"`
	}
)

func (s *SlaveListService) List(c *gin.Context) ([]fs.PhysicalObject, error) {
	dep := dependency.FromContext(c)
	m := manager.NewFileManager(dep, nil)
	defer m.Recycle()
	d := m.LocalDriver(nil)

	objects, err := d.List(c, s.Path, func(i int) {}, s.Recursive)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	return objects, nil
}
