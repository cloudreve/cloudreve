package controllers

import (
	"errors"
	"fmt"

	"github.com/cloudreve/Cloudreve/v4/pkg/cluster"
	"github.com/cloudreve/Cloudreve/v4/pkg/downloader"
	"github.com/cloudreve/Cloudreve/v4/pkg/downloader/slave"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/service/admin"
	"github.com/cloudreve/Cloudreve/v4/service/explorer"
	"github.com/cloudreve/Cloudreve/v4/service/node"
	"github.com/gin-gonic/gin"
)

// SlaveUpload 从机文件上传
func SlaveUpload(c *gin.Context) {
	service := ParametersFromContext[*explorer.UploadService](c, explorer.UploadParameterCtx{})
	err := service.SlaveUpload(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		request.BlackHole(c.Request.Body)
		c.Abort()
		return
	}

	c.JSON(200, serializer.Response{})
}

// SlaveGetUploadSession 从机创建上传会话
func SlaveGetUploadSession(c *gin.Context) {
	service := ParametersFromContext[*explorer.SlaveCreateUploadSessionService](c, explorer.SlaveCreateUploadSessionParamCtx{})
	respond(c, service.Create(c), serializer.Response{})
}

// SlaveDeleteUploadSession 从机删除上传会话
func SlaveDeleteUploadSession(c *gin.Context) {
	service := ParametersFromContext[*explorer.SlaveDeleteUploadSessionService](c, explorer.SlaveDeleteUploadSessionParamCtx{})
	respond(c, service.Delete(c), serializer.Response{})
}

// SlaveServeEntity download entity content
func SlaveServeEntity(c *gin.Context) {
	service := ParametersFromContext[*explorer.EntityDownloadService](c, explorer.EntityDownloadParameterCtx{})
	err := service.SlaveServe(c)
	if err != nil {
		c.JSON(400, serializer.Err(c, err))
		c.Abort()
		return
	}
}

// SlaveMeta retrieve media metadata
func SlaveMeta(c *gin.Context) {
	service := ParametersFromContext[*explorer.SlaveMetaService](c, explorer.SlaveMetaParamCtx{})
	res, err := service.MediaMeta(c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.NewResponseWithGobData(c, res))
}

// SlaveThumb 从机文件缩略图
func SlaveThumb(c *gin.Context) {
	service := ParametersFromContext[*explorer.SlaveThumbService](c, explorer.SlaveThumbParamCtx{})
	err := service.Thumb(c)
	if respondErr(c, err) {
		return
	}
}

// SlaveDelete 从机删除
func SlaveDelete(c *gin.Context) {
	service := ParametersFromContext[*explorer.SlaveDeleteFileService](c, explorer.SlaveDeleteFileParamCtx{})
	if failed, err := service.Delete(c); err != nil {
		c.JSON(200, serializer.NewResponseWithGobData(c, serializer.Response{
			Code:  serializer.CodeNotFullySuccess,
			Data:  failed,
			Msg:   fmt.Sprintf("Failed to delete %d files(s)", len(failed)),
			Error: err.Error(),
		}))
	} else {
		c.JSON(200, serializer.Response{Data: ""})
	}
}

// SlavePing 从机测试
func SlavePing(c *gin.Context) {
	service := ParametersFromContext[*admin.SlavePingService](c, admin.SlavePingParameterCtx{})
	respond(c, service.Test(c), serializer.Response{})
}

// SlaveList 从机列出文件
func SlaveList(c *gin.Context) {
	service := ParametersFromContext[*explorer.SlaveListService](c, explorer.SlaveListParamCtx{})
	objects, err := service.List(c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.NewResponseWithGobData(c, objects))
}

// SlaveDownloadTaskCreate creates a download task on slave
func SlaveDownloadTaskCreate(c *gin.Context) {
	service := ParametersFromContext[*slave.CreateSlaveDownload](c, node.CreateSlaveDownloadTaskParamCtx{})
	d := c.MustGet(downloader.DownloaderCtxKey).(downloader.Downloader)
	handle, err := d.CreateTask(c, service.Url, service.Options)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.NewResponseWithGobData(c, handle))
}

// SlaveDownloadTaskStatus 查询从机 Aria2 任务状态
func SlaveDownloadTaskStatus(c *gin.Context) {
	service := ParametersFromContext[*slave.GetSlaveDownload](c, node.GetSlaveDownloadTaskParamCtx{})
	d := c.MustGet(downloader.DownloaderCtxKey).(downloader.Downloader)
	info, err := d.Info(c, service.Handle)
	if err != nil {
		if errors.Is(err, downloader.ErrTaskNotFount) {
			c.JSON(200, serializer.NewError(serializer.CodeNotFound, "task not found", err))
			c.Abort()
			return
		}

		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	c.JSON(200, serializer.NewResponseWithGobData(c, info))
}

// SlaveCancelDownloadTask 取消从机离线下载任务
func SlaveCancelDownloadTask(c *gin.Context) {
	service := ParametersFromContext[*slave.CancelSlaveDownload](c, node.CancelSlaveDownloadTaskParamCtx{})
	d := c.MustGet(downloader.DownloaderCtxKey).(downloader.Downloader)
	err := d.Cancel(c, service.Handle)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{})
}

// SlaveSelectFilesToDownload 从机选取离线下载文件
func SlaveSelectFilesToDownload(c *gin.Context) {
	service := ParametersFromContext[*slave.SetSlaveFilesToDownload](c, node.SelectSlaveDownloadFilesParamCtx{})
	d := c.MustGet(downloader.DownloaderCtxKey).(downloader.Downloader)
	err := d.SetFilesToDownload(c, service.Handle, service.Args...)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{})
}

// SlaveTestDownloader 从机测试下载器连接
func SlaveTestDownloader(c *gin.Context) {
	d := c.MustGet(downloader.DownloaderCtxKey).(downloader.Downloader)
	res, err := d.Test(c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

// SlaveGetOauthCredential 从机获取主机的OneDrive存储策略凭证
func SlaveGetCredential(c *gin.Context) {
	service := ParametersFromContext[*node.OauthCredentialService](c, node.OauthCredentialParamCtx{})
	cred, err := service.Get(c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.NewResponseWithGobData(c, cred))
}

// SlaveCreateTask creates tasks and register it in registry
func SlaveCreateTask(c *gin.Context) {
	service := ParametersFromContext[*cluster.CreateSlaveTask](c, node.CreateSlaveTaskParamCtx{})
	taskId, err := node.CreateTaskInSlave(service, c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.NewResponseWithGobData(c, taskId))
}

// SlaveGetTask returns the slave node's task info
func SlaveGetTask(c *gin.Context) {
	service := ParametersFromContext[*node.GetSlaveTaskService](c, node.GetSlaveTaskParamCtx{})
	task, err := service.Get(c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.NewResponseWithGobData(c, task))
}

func SlaveCleanupFolder(c *gin.Context) {
	service := ParametersFromContext[*cluster.FolderCleanup](c, node.FolderCleanupParamCtx{})
	respond(c, node.Cleanup(service, c), serializer.Response{})
}

func StatelessPrepareUpload(c *gin.Context) {
	service := ParametersFromContext[*fs.StatelessPrepareUploadService](c, node.StatelessPrepareUploadParamCtx{})
	uploadSession, err := node.StatelessPrepareUpload(service, c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.NewResponseWithGobData(c, uploadSession))
}

func StatelessCompleteUpload(c *gin.Context) {
	service := ParametersFromContext[*fs.StatelessCompleteUploadService](c, node.StatelessCompleteUploadParamCtx{})
	_, err := node.StatelessCompleteUpload(service, c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{})
}

func StatelessOnUploadFailed(c *gin.Context) {
	service := ParametersFromContext[*fs.StatelessOnUploadFailedService](c, node.StatelessOnUploadFailedParamCtx{})
	respond(c, node.StatelessOnUploadFailed(service, c), serializer.Response{})
}

func StatelessCreateFile(c *gin.Context) {
	service := ParametersFromContext[*fs.StatelessCreateFileService](c, node.StatelessCreateFileParamCtx{})
	respond(c, node.StatelessCreateFile(service, c), serializer.Response{})
}
