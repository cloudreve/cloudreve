package controllers

import (
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/service/explorer"
	"github.com/gin-gonic/gin"
)

func ListTasks(c *gin.Context) {
	service := ParametersFromContext[*explorer.ListTaskService](c, explorer.ListTaskParamCtx{})
	resp, err := service.ListTasks(c)
	if respondErr(c, err) {
		return
	}

	if resp != nil {
		c.JSON(200, serializer.Response{
			Data: resp,
		})
	}
}

func GetTaskPhaseProgress(c *gin.Context) {
	taskId := hashid.FromContext(c)
	resp, err := explorer.TaskPhaseProgress(c, taskId)
	if respondErr(c, err) {
		return
	}

	if resp != nil {
		c.JSON(200, serializer.Response{
			Data: resp,
		})
	} else {
		c.JSON(200, serializer.Response{Data: queue.Progresses{}})
	}
}

func SetDownloadTaskTarget(c *gin.Context) {
	taskId := hashid.FromContext(c)
	service := ParametersFromContext[*explorer.SetDownloadFilesService](c, explorer.SetDownloadFilesParamCtx{})
	err := service.SetDownloadFiles(c, taskId)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{})
}

func CancelDownloadTask(c *gin.Context) {
	taskId := hashid.FromContext(c)
	err := explorer.CancelDownloadTask(c, taskId)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{})
}

// CancelTask terminates a queued or suspending task.
func CancelTask(c *gin.Context) {
	taskId := hashid.FromContext(c)
	err := explorer.CancelTask(c, taskId)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{})
}

// DeleteTask hides a finished task record from the owner's list.
func DeleteTask(c *gin.Context) {
	taskId := hashid.FromContext(c)
	err := explorer.DeleteTask(c, taskId)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{})
}

// RetryTask re-queues a failed task with its original args.
func RetryTask(c *gin.Context) {
	taskId := hashid.FromContext(c)
	err := explorer.RetryTask(c, taskId)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{})
}
