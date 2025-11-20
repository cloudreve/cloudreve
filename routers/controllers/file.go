package controllers

import (
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/service/explorer"
	"github.com/gin-gonic/gin"
)

func DownloadArchive(c *gin.Context) {
	service := ParametersFromContext[*explorer.ArchiveService](c, explorer.ArchiveParamCtx{})
	err := service.DownloadArchived(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}
}

// CreateArchive 创建文件压缩任务
func CreateArchive(c *gin.Context) {
	service := ParametersFromContext[*explorer.ArchiveWorkflowService](c, explorer.CreateArchiveParamCtx{})
	resp, err := service.CreateCompressTask(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	if resp != nil {
		c.JSON(200, serializer.Response{
			Data: resp,
		})
	}
}

// ImportFiles imports files
func ImportFiles(c *gin.Context) {
	service := ParametersFromContext[*explorer.ImportWorkflowService](c, explorer.CreateImportParamCtx{})
	resp, err := service.CreateImportTask(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	if resp != nil {
		c.JSON(200, serializer.Response{
			Data: resp,
		})
	}
}

// CreateRemoteDownload creates remote download task
func CreateRemoteDownload(c *gin.Context) {
	service := ParametersFromContext[*explorer.DownloadWorkflowService](c, explorer.CreateDownloadParamCtx{})
	resp, err := service.CreateDownloadTask(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	if resp != nil {
		c.JSON(200, serializer.Response{
			Data: resp,
		})
	}
}

// ExtractArchive creates extract archive task
func ExtractArchive(c *gin.Context) {
	service := ParametersFromContext[*explorer.ArchiveWorkflowService](c, explorer.CreateArchiveParamCtx{})
	resp, err := service.CreateExtractTask(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	if resp != nil {
		c.JSON(200, serializer.Response{
			Data: resp,
		})
	}
}

// AnonymousPermLink 文件中转后的永久直链接
func AnonymousPermLink(download bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if err := explorer.RedirectDirectLink(c, name, download); err != nil {
			c.JSON(404, serializer.Err(c, err))
			c.Abort()
			return
		}
	}
}

// GetSource 获取文件的外链地址
func GetSource(c *gin.Context) {
	service := ParametersFromContext[*explorer.GetDirectLinkService](c, explorer.GetDirectLinkParamCtx{})
	res, err := service.Get(c)
	if err != nil && len(res) == 0 {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	if err != nil {
		// Not fully completed
		errResp := serializer.Err(c, err)
		errResp.Data = res
		c.JSON(200, errResp)
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

func DeleteDirectLink(c *gin.Context) {
	err := explorer.DeleteDirectLink(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{})
}

// Thumb 获取文件缩略图
func Thumb(c *gin.Context) {
	service := ParametersFromContext[*explorer.FileThumbService](c, explorer.FileThumbParameterCtx{})
	res, err := service.Get(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

// FileURL get temporary file url for preview or download
func FileURL(c *gin.Context) {
	service := ParametersFromContext[*explorer.FileURLService](c, explorer.FileURLParameterCtx{})
	resp, err := service.Get(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	if resp != nil {
		c.JSON(200, serializer.Response{
			Data: resp,
		})
	}
}

// ServeEntity download entity content
func ServeEntity(c *gin.Context) {
	service := ParametersFromContext[*explorer.EntityDownloadService](c, explorer.EntityDownloadParameterCtx{})
	err := service.Serve(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}
}

// CreateViewerSession creates a viewer session
func CreateViewerSession(c *gin.Context) {
	service := ParametersFromContext[*explorer.CreateViewerSessionService](c, explorer.CreateViewerSessionParamCtx{})
	resp, err := service.Create(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	if resp != nil {
		c.JSON(200, serializer.Response{
			Data: resp,
		})
	}
}

// PutContent 更新文件内容
func PutContent(c *gin.Context) {
	service := ParametersFromContext[*explorer.FileUpdateService](c, explorer.FileUpdateParameterCtx{})
	res, err := service.PutContent(c, nil)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		request.BlackHole(c.Request.Body)
		c.Abort()
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

// FileUpload 本地策略文件上传
func FileUpload(c *gin.Context) {
	service := ParametersFromContext[*explorer.UploadService](c, explorer.UploadParameterCtx{})
	err := service.LocalUpload(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		request.BlackHole(c.Request.Body)
		c.Abort()
		return
	}

	c.JSON(200, serializer.Response{})
}

// DeleteUploadSession 删除上传会话
func DeleteUploadSession(c *gin.Context) {
	service := ParametersFromContext[*explorer.DeleteUploadSessionService](c, explorer.DeleteUploadSessionParameterCtx{})
	err := service.Delete(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	c.JSON(200, serializer.Response{})
}

// CreateUploadSession 创建上传会话
func CreateUploadSession(c *gin.Context) {
	service := ParametersFromContext[*explorer.CreateUploadSessionService](c, explorer.CreateUploadSessionParameterCtx{})
	resp, err := service.Create(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	c.JSON(200, serializer.Response{
		Data: resp,
	})
}

// CreateFile 创建空白文件
func CreateFile(c *gin.Context) {
	service := ParametersFromContext[*explorer.CreateFileService](c, explorer.CreateFileParameterCtx{})
	resp, err := service.Create(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	c.JSON(200, serializer.Response{
		Data: resp,
	})
}

// RenameFile Renames a file.
func RenameFile(c *gin.Context) {
	service := ParametersFromContext[*explorer.RenameFileService](c, explorer.RenameFileParameterCtx{})
	resp, err := service.Rename(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	c.JSON(200, serializer.Response{
		Data: resp,
	})
}

// MoveFile Moves or Copy files.
func MoveFile(c *gin.Context) {
	service := ParametersFromContext[*explorer.MoveFileService](c, explorer.MoveFileParameterCtx{})
	if err := service.Move(c); err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	c.JSON(200, serializer.Response{})
}

// Delete 删除文件或目录
func Delete(c *gin.Context) {
	service := ParametersFromContext[*explorer.DeleteFileService](c, explorer.DeleteFileParameterCtx{})
	err := service.Delete(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	c.JSON(200, serializer.Response{})
}

// Restore restore file or directory
func Restore(c *gin.Context) {
	service := ParametersFromContext[*explorer.DeleteFileService](c, explorer.DeleteFileParameterCtx{})
	err := service.Restore(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	c.JSON(200, serializer.Response{})
}

// Unlock unlocks files by given tokens
func Unlock(c *gin.Context) {
	service := ParametersFromContext[*explorer.UnlockFileService](c, explorer.UnlockFileParameterCtx{})
	err := service.Unlock(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	c.JSON(200, serializer.Response{})
}

// Pin pins files by given uri
func Pin(c *gin.Context) {
	service := ParametersFromContext[*explorer.PinFileService](c, explorer.PinFileParameterCtx{})
	err := service.PinFile(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{})
}

// Unpin unpins files by given uri
func Unpin(c *gin.Context) {
	service := ParametersFromContext[*explorer.PinFileService](c, explorer.PinFileParameterCtx{})
	err := service.UnpinFile(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{})
}

// PatchMetadata patch metadata
func PatchMetadata(c *gin.Context) {
	service := ParametersFromContext[*explorer.PatchMetadataService](c, explorer.PatchMetadataParameterCtx{})
	err := service.Patch(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	c.JSON(200, serializer.Response{})
}

// GetFileInfo gets file info
func GetFileInfo(c *gin.Context) {
	service := ParametersFromContext[*explorer.GetFileInfoService](c, explorer.GetFileInfoParameterCtx{})
	resp, err := service.Get(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	c.JSON(200, serializer.Response{
		Data: resp,
	})
}

// SetCurrentVersion sets current version
func SetCurrentVersion(c *gin.Context) {
	service := ParametersFromContext[*explorer.SetCurrentVersionService](c, explorer.SetCurrentVersionParamCtx{})
	err := service.Set(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	c.JSON(200, serializer.Response{})
}

// DeleteVersion deletes a version
func DeleteVersion(c *gin.Context) {
	service := ParametersFromContext[*explorer.DeleteVersionService](c, explorer.DeleteVersionParamCtx{})
	err := service.Delete(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	c.JSON(200, serializer.Response{})
}

func PatchView(c *gin.Context) {
	service := ParametersFromContext[*explorer.PatchViewService](c, explorer.PatchViewParameterCtx{})
	err := service.Patch(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	c.JSON(200, serializer.Response{})
}

func ListArchiveFiles(c *gin.Context) {
	service := ParametersFromContext[*explorer.ArchiveListFilesService](c, explorer.ArchiveListFilesParamCtx{})
	resp, err := service.List(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	c.JSON(200, serializer.Response{
		Data: resp,
	})
}
