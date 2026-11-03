package admin

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/service/explorer"
	"github.com/gin-gonic/gin"
)

type ReceiveFileParamCtx struct{}

type ReceiveFileService struct {
	SourceID string `json:"source_id" binding:"required"`
	Dst      string `json:"dst" binding:"required"`
}

func (s *ReceiveFileService) Receive(c *gin.Context) (*explorer.FileResponse, error) {
	dep := dependency.FromContext(c)
	fileID, err := dep.HashIDEncoder().Decode(s.SourceID, hashid.FileID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "invalid source file ID", err)
	}
	dst, err := fs.NewUriFromString(s.Dst)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "invalid destination URI", err)
	}
	u := inventory.UserFromContext(c)
	fm := manager.NewFileManager(dep, u)
	defer fm.Recycle()
	file, err := fm.ReceiveFile(c, fileID, dst)
	if err != nil {
		return nil, err
	}
	return explorer.BuildFileResponse(c, u, file, dep.HashIDEncoder(), nil), nil
}
