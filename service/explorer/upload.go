package explorer

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
)

// CreateUploadSessionService 获取上传凭证服务
type (
	CreateUploadSessionParameterCtx struct{}
	CreateUploadSessionService      struct {
		Uri                 string            `json:"uri" binding:"required"`
		Size                int64             `json:"size" binding:"min=0"`
		LastModified        int64             `json:"last_modified"`
		MimeType            string            `json:"mime_type"`
		PolicyID            string            `json:"policy_id"`
		Metadata            map[string]string `json:"metadata" binding:"max=256"`
		EntityType          string            `json:"entity_type" binding:"eq=|eq=live_photo|eq=version"`
		EncryptionSupported []types.Cipher    `json:"encryption_supported"`
	}
)

// Create 创建新的上传会话
func (service *CreateUploadSessionService) Create(c context.Context) (*UploadSessionResponse, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	m := manager.NewFileManager(dep, user)
	defer m.Recycle()

	uri, err := fs.NewUriFromString(service.Uri)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "unknown uri", err)
	}

	var entityType *types.EntityType
	switch service.EntityType {
	case "live_photo":
		livePhoto := types.EntityTypeLivePhoto
		entityType = &livePhoto
	case "version":
		version := types.EntityTypeVersion
		entityType = &version
	}

	hasher := dep.HashIDEncoder()
	policyId, err := hasher.Decode(service.PolicyID, hashid.PolicyID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "unknown policy id", err)
	}

	uploadRequest := &fs.UploadRequest{
		Props: &fs.UploadProps{
			Uri:  uri,
			Size: service.Size,

			MimeType:               service.MimeType,
			Metadata:               service.Metadata,
			EntityType:             entityType,
			PreferredStoragePolicy: policyId,
			EncryptionSupported:    service.EncryptionSupported,
			ClientSideEncrypted:    len(service.EncryptionSupported) > 0,
		},
	}

	if service.LastModified > 0 {
		lastModified := time.UnixMilli(service.LastModified)
		uploadRequest.Props.LastModified = &lastModified
	}

	credential, err := m.CreateUploadSession(c, uploadRequest)
	if err != nil {
		return nil, err
	}

	return BuildUploadSessionResponse(credential, hasher), nil
}

type (
	UploadParameterCtx struct{}
	// UploadService 本机及从机策略上传服务
	UploadService struct {
		ID    string `uri:"sessionId" binding:"required"`
		Index int    `uri:"index" form:"index" binding:"min=0"`
	}
)

// LocalUpload 处理本机文件分片上传
func (service *UploadService) LocalUpload(c *gin.Context) error {
	dep := dependency.FromContext(c)
	kv := dep.KV()

	uploadSessionRaw, ok := kv.Get(manager.UploadSessionCachePrefix + service.ID)
	if !ok {
		return serializer.NewError(serializer.CodeUploadSessionExpired, "", nil)
	}

	uploadSession := uploadSessionRaw.(fs.UploadSession)

	user := inventory.UserFromContext(c)
	m := manager.NewFileManager(dep, user)
	defer m.Recycle()

	if uploadSession.UID != user.ID {
		return serializer.NewError(serializer.CodeUploadSessionExpired, "", nil)
	}

	// Confirm upload session and chunk index
	placeholder, err := m.ConfirmUploadSession(c, &uploadSession, service.Index)
	if err != nil {
		return err
	}

	return processChunkUpload(c, m, &uploadSession, service.Index, placeholder, fs.ModeOverwrite)
}

// SlaveUpload 处理从机文件分片上传
func (service *UploadService) SlaveUpload(c *gin.Context) error {
	dep := dependency.FromContext(c)
	kv := dep.KV()

	uploadSessionRaw, ok := kv.Get(manager.UploadSessionCachePrefix + service.ID)
	if !ok {
		return serializer.NewError(serializer.CodeUploadSessionExpired, "", nil)
	}

	uploadSession := uploadSessionRaw.(fs.UploadSession)
	uploadSession.Props.ClientSideEncrypted = true

	// Parse chunk index from query
	service.Index, _ = strconv.Atoi(c.Query("chunk"))

	m := manager.NewFileManager(dep, nil)
	defer m.Recycle()

	return processChunkUpload(c, m, &uploadSession, service.Index, nil, fs.ModeOverwrite)
}

func processChunkUpload(c *gin.Context, m manager.FileManager, session *fs.UploadSession, index int, file fs.File, mode fs.WriteMode) error {
	// 取得并校验文件大小是否符合分片要求
	chunkSize := session.ChunkSize
	isLastChunk := session.ChunkSize == 0 || int64(index+1)*chunkSize >= session.Props.Size
	expectedLength := chunkSize
	if isLastChunk {
		expectedLength = session.Props.Size - int64(index)*chunkSize
	}

	rc, fileSize, err := request.SniffContentLength(c.Request)
	if err != nil || (expectedLength != fileSize) {
		return serializer.NewError(
			serializer.CodeInvalidContentLength,
			fmt.Sprintf("Invalid Content-Length (expected: %d)", expectedLength),
			err,
		)
	}

	// 非首个分片时需要允许覆盖
	if index > 0 {
		mode |= fs.ModeOverwrite
	}

	req := &fs.UploadRequest{
		File:   rc,
		Offset: chunkSize * int64(index),
		Props:  session.Props.Copy(),
		Mode:   mode,
	}

	// 执行上传
	ctx := context.WithValue(c, cluster.SlaveNodeIDCtx{}, strconv.Itoa(session.Policy.NodeID))
	err = m.Upload(ctx, req, session.Policy, session)
	if err != nil {
		return err
	}

	if rc, ok := req.File.(request.LimitReaderCloser); ok {
		if rc.Count() != expectedLength {
			err := fmt.Errorf("uploaded data(%d) does not match purposed size(%d)", rc.Count(), req.Props.Size)
			return serializer.NewError(serializer.CodeIOFailed, "Uploaded data does not match purposed size", err)
		}
	}

	// Finish upload
	if isLastChunk {
		_, err := m.CompleteUpload(ctx, session)
		if err != nil {
			return fmt.Errorf("failed to complete upload: %w", err)
		}
	}

	return nil
}

type (
	DeleteUploadSessionParameterCtx struct{}
	DeleteUploadSessionService      struct {
		ID  string `json:"id" binding:"required"`
		Uri string `json:"uri" binding:"required"`
	}
)

// Delete deletes the specified upload session
func (service *DeleteUploadSessionService) Delete(c *gin.Context) error {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	m := manager.NewFileManager(dep, user)
	defer m.Recycle()

	uri, err := fs.NewUriFromString(service.Uri)
	if err != nil {
		return serializer.NewError(serializer.CodeParamErr, "unknown uri", err)
	}

	return m.CancelUploadSession(c, uri, service.ID)
}
