package controllers

import (
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/service/admin"
	"github.com/gin-gonic/gin"
)

func AdminReceiveFile(c *gin.Context) {
	s := ParametersFromContext[*admin.ReceiveFileService](c, admin.ReceiveFileParamCtx{})
	file, err := s.Receive(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: file})
}
