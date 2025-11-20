package admin

import (
	"context"
	"path"
	"strconv"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager/entitysource"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

// FileService 文件ID服务
type FileService struct {
	ID uint `uri:"id" json:"id" binding:"required"`
}

// FileBatchService 文件批量操作服务
type FileBatchService struct {
	ID         []uint `json:"id" binding:"min=1"`
	Force      bool   `json:"force"`
	UnlinkOnly bool   `json:"unlink"`
}

// ListFolderService 列目录结构
type ListFolderService struct {
	Path string `uri:"path" binding:"required,max=65535"`
	ID   uint   `uri:"id" binding:"required"`
	Type string `uri:"type" binding:"eq=policy|eq=user"`
}

// List 列出指定路径下的目录
func (service *ListFolderService) List(c *gin.Context) serializer.Response {
	//if service.Type == "policy" {
	//	// 列取存储策略中的目录
	//	policy, err := model.GetPolicyByID(service.ID)
	//	if err != nil {
	//		return serializer.ErrDeprecated(serializer.CodePolicyNotExist, "", err)
	//	}
	//
	//	// 创建文件系统
	//	fs, err := filesystem.NewAnonymousFileSystem()
	//	if err != nil {
	//		return serializer.ErrDeprecated(serializer.CodeCreateFSError, "", err)
	//	}
	//	defer fs.Recycle()
	//
	//	// 列取存储策略中的文件
	//	fs.Policy = &policy
	//	res, err := fs.ListPhysical(c.Request.Context(), service.Path)
	//	if err != nil {
	//		return serializer.ErrDeprecated(serializer.CodeListFilesError, "", err)
	//	}
	//
	//	return serializer.Response{
	//		Data: serializer.BuildObjectList(0, res, nil),
	//	}
	//
	//}
	//
	//// 列取用户空间目录
	//// 查找用户
	//user, err := model.GetUserByID(service.ID)
	//if err != nil {
	//	return serializer.ErrDeprecated(serializer.CodeUserNotFound, "", err)
	//}
	//
	//// 创建文件系统
	//fs, err := filesystem.NewFileSystem(&user)
	//if err != nil {
	//	return serializer.ErrDeprecated(serializer.CodeCreateFSError, "", err)
	//}
	//defer fs.Recycle()
	//
	//// 列取目录
	//res, err := fs.List(c.Request.Context(), service.Path, nil)
	//if err != nil {
	//	return serializer.ErrDeprecated(serializer.CodeListFilesError, "", err)
	//}

	//return serializer.Response{
	//	Data: serializer.BuildObjectList(0, res, nil),
	//}

	return serializer.Response{}
}

// Delete 删除文件
func (service *FileBatchService) Delete(c *gin.Context) serializer.Response {
	//files, err := model.GetFilesByIDs(service.ID, 0)
	//if err != nil {
	//	return serializer.DBErrDeprecated("Failed to list files for deleting", err)
	//}
	//
	//// 根据用户分组
	//userFile := make(map[uint][]model.File)
	//for i := 0; i < len(files); i++ {
	//	if _, ok := userFile[files[i].UserID]; !ok {
	//		userFile[files[i].UserID] = []model.File{}
	//	}
	//	userFile[files[i].UserID] = append(userFile[files[i].UserID], files[i])
	//}
	//
	//// 异步执行删除
	//go func(files map[uint][]model.File) {
	//	for uid, file := range files {
	//		var (
	//			fs  *filesystem.FileSystem
	//			err error
	//		)
	//		user, err := model.GetUserByID(uid)
	//		if err != nil {
	//			fs, err = filesystem.NewAnonymousFileSystem()
	//			if err != nil {
	//				continue
	//			}
	//		} else {
	//			fs, err = filesystem.NewFileSystem(&user)
	//			if err != nil {
	//				fs.Recycle()
	//				continue
	//			}
	//		}
	//
	//		// 汇总文件ID
	//		ids := make([]uint, 0, len(file))
	//		for i := 0; i < len(file); i++ {
	//			ids = append(ids, file[i].ID)
	//		}
	//
	//		// 执行删除
	//		fs.Delete(context.Background(), []uint{}, ids, service.Force, service.UnlinkOnly)
	//		fs.Recycle()
	//	}
	//}(userFile)

	// 分组执行删除
	return serializer.Response{}

}

const (
	fileNameCondition       = "file_name"
	fileUserCondition       = "file_user"
	filePolicyCondition     = "file_policy"
	fileMetadataCondition   = "file_metadata"
	fileSharedCondition     = "file_shared"
	fileDirectLinkCondition = "file_direct_link"
)

func (service *AdminListService) Files(c *gin.Context) (*ListFileResponse, error) {
	dep := dependency.FromContext(c)
	hasher := dep.HashIDEncoder()
	fileClient := dep.FileClient()

	ctx := context.WithValue(c, inventory.LoadFileEntity{}, true)
	ctx = context.WithValue(ctx, inventory.LoadFileMetadata{}, true)
	ctx = context.WithValue(ctx, inventory.LoadFileShare{}, true)
	ctx = context.WithValue(ctx, inventory.LoadFileUser{}, true)
	ctx = context.WithValue(ctx, inventory.LoadFileDirectLink{}, true)

	var (
		err        error
		userID     int
		policyID   int
		metadata   string
		shared     bool
		directLink bool
	)

	if service.Conditions[fileUserCondition] != "" {
		userID, err = strconv.Atoi(service.Conditions[fileUserCondition])
		if err != nil {
			return nil, serializer.NewError(serializer.CodeParamErr, "Invalid user ID", err)
		}
	}

	if service.Conditions[filePolicyCondition] != "" {
		policyID, err = strconv.Atoi(service.Conditions[filePolicyCondition])
		if err != nil {
			return nil, serializer.NewError(serializer.CodeParamErr, "Invalid policy ID", err)
		}
	}

	if service.Conditions[fileMetadataCondition] != "" {
		metadata = service.Conditions[fileMetadataCondition]
	}

	if service.Conditions[fileSharedCondition] != "" && setting.IsTrueValue(service.Conditions[fileSharedCondition]) {
		shared = true
	}

	if service.Conditions[fileDirectLinkCondition] != "" && setting.IsTrueValue(service.Conditions[fileDirectLinkCondition]) {
		directLink = true
	}

	res, err := fileClient.FlattenListFiles(ctx, &inventory.FlattenListFileParameters{
		PaginationArgs: &inventory.PaginationArgs{
			Page:     service.Page - 1,
			PageSize: service.PageSize,
			OrderBy:  service.OrderBy,
			Order:    inventory.OrderDirection(service.OrderDirection),
		},
		UserID:          userID,
		StoragePolicyID: policyID,
		Name:            service.Conditions[fileNameCondition],
		HasMetadata:     metadata,
		Shared:          shared,
		HasDirectLink:   directLink,
	})

	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list files", err)
	}

	return &ListFileResponse{
		Pagination: res.PaginationResults,
		Files: lo.Map(res.Files, func(file *ent.File, _ int) GetFileResponse {
			return GetFileResponse{
				File:       file,
				UserHashID: hashid.EncodeUserID(hasher, file.OwnerID),
				FileHashID: hashid.EncodeFileID(hasher, file.ID),
			}
		}),
	}, nil
}

type (
	SingleFileService struct {
		ID int `uri:"id" json:"id" binding:"required"`
	}
	SingleFileParamCtx struct{}
)

func (service *SingleFileService) Get(c *gin.Context) (*GetFileResponse, error) {
	dep := dependency.FromContext(c)
	hasher := dep.HashIDEncoder()
	fileClient := dep.FileClient()

	ctx := context.WithValue(c, inventory.LoadFileEntity{}, true)
	ctx = context.WithValue(ctx, inventory.LoadFileMetadata{}, true)
	ctx = context.WithValue(ctx, inventory.LoadFileShare{}, true)
	ctx = context.WithValue(ctx, inventory.LoadFileUser{}, true)
	ctx = context.WithValue(ctx, inventory.LoadEntityUser{}, true)
	ctx = context.WithValue(ctx, inventory.LoadEntityStoragePolicy{}, true)
	ctx = context.WithValue(ctx, inventory.LoadFileDirectLink{}, true)

	file, err := fileClient.GetByID(ctx, service.ID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, serializer.NewError(serializer.CodeNotFound, "File not found", nil)
		}

		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get file", err)
	}

	directLinkMap := make(map[int]string)
	siteURL := dep.SettingProvider().SiteURL(c)
	for _, directLink := range file.Edges.DirectLinks {
		directLinkMap[directLink.ID] = routes.MasterDirectLink(siteURL, hashid.EncodeSourceLinkID(hasher, directLink.ID), directLink.Name).String()
	}

	return &GetFileResponse{
		File:          file,
		UserHashID:    hashid.EncodeUserID(hasher, file.OwnerID),
		FileHashID:    hashid.EncodeFileID(hasher, file.ID),
		DirectLinkMap: directLinkMap,
	}, nil
}

type (
	UpsertFileService struct {
		File *ent.File `json:"file" binding:"required"`
	}
	UpsertFileParamCtx struct{}
)

func (s *UpsertFileService) Update(c *gin.Context) (*GetFileResponse, error) {
	dep := dependency.FromContext(c)
	fileClient := dep.FileClient()

	fc, tx, ctx, err := inventory.WithTx(c, fileClient)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to start transaction", err)
	}

	newFile, err := fc.Update(ctx, s.File)
	if err != nil {
		_ = inventory.Rollback(tx)
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to update file", err)
	}

	if err := inventory.Commit(tx); err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to commit transaction", err)
	}

	service := &SingleFileService{ID: newFile.ID}
	return service.Get(c)
}

func (s *SingleFileService) Url(c *gin.Context) (string, error) {
	dep := dependency.FromContext(c)
	fileClient := dep.FileClient()

	ctx := context.WithValue(c, inventory.LoadFileEntity{}, true)
	file, err := fileClient.GetByID(ctx, s.ID)
	if err != nil {
		return "", serializer.NewError(serializer.CodeDBError, "Failed to get file", err)
	}

	// find primary entity
	var primaryEntity *ent.Entity
	for _, entity := range file.Edges.Entities {
		if entity.Type == int(types.EntityTypeVersion) && entity.ID == file.PrimaryEntity {
			primaryEntity = entity
			break
		}
	}

	if primaryEntity == nil {
		return "", serializer.NewError(serializer.CodeNotFound, "Primary entity not exist", nil)
	}

	// find policy
	policy, err := dep.StoragePolicyClient().GetPolicyByID(ctx, primaryEntity.StoragePolicyEntities)
	if err != nil {
		return "", serializer.NewError(serializer.CodeDBError, "Failed to get policy", err)
	}

	m := manager.NewFileManager(dep, inventory.UserFromContext(c))
	defer m.Recycle()

	driver, err := m.GetStorageDriver(ctx, policy)
	if err != nil {
		return "", serializer.NewError(serializer.CodeInternalSetting, "Failed to get storage driver", err)
	}

	es := entitysource.NewEntitySource(fs.NewEntity(primaryEntity), driver, policy, dep.GeneralAuth(),
		dep.SettingProvider(), dep.HashIDEncoder(), dep.RequestClient(), dep.Logger(), dep.ConfigProvider(), dep.MimeDetector(ctx), dep.EncryptorFactory(ctx))

	expire := time.Now().Add(time.Hour * 1)
	url, err := es.Url(ctx, entitysource.WithExpire(&expire), entitysource.WithDisplayName(file.Name))
	if err != nil {
		return "", serializer.NewError(serializer.CodeInternalSetting, "Failed to get url", err)
	}

	return url.Url, nil
}

type (
	BatchFileService struct {
		IDs []int `json:"ids" binding:"min=1"`
	}
	BatchFileParamCtx struct{}
)

func (s *BatchFileService) Delete(c *gin.Context) error {
	dep := dependency.FromContext(c)
	fileClient := dep.FileClient()

	ctx := context.WithValue(c, inventory.LoadFileEntity{}, true)
	files, _, err := fileClient.GetByIDs(ctx, s.IDs, 0)
	if err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to get files", err)
	}

	fc, tx, ctx, err := inventory.WithTx(c, fileClient)
	if err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to start transaction", err)
	}

	_, diff, err := fc.Delete(ctx, files, nil)
	if err != nil {
		_ = inventory.Rollback(tx)
		return serializer.NewError(serializer.CodeDBError, "Failed to delete files", err)
	}

	tx.AppendStorageDiff(diff)
	if err := inventory.CommitWithStorageDiff(ctx, tx, dep.Logger(), dep.UserClient()); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to commit transaction", err)
	}

	return nil
}

const (
	entityUserCondition   = "entity_user"
	entityPolicyCondition = "entity_policy"
	entityTypeCondition   = "entity_type"
)

func (s *AdminListService) Entities(c *gin.Context) (*ListEntityResponse, error) {
	dep := dependency.FromContext(c)
	fileClient := dep.FileClient()
	hasher := dep.HashIDEncoder()
	ctx := context.WithValue(c, inventory.LoadEntityUser{}, true)
	ctx = context.WithValue(ctx, inventory.LoadEntityStoragePolicy{}, true)

	var (
		userID     int
		policyID   int
		err        error
		entityType *types.EntityType
	)

	if s.Conditions[entityUserCondition] != "" {
		userID, err = strconv.Atoi(s.Conditions[entityUserCondition])
		if err != nil {
			return nil, serializer.NewError(serializer.CodeParamErr, "Invalid user ID", err)
		}
	}

	if s.Conditions[entityPolicyCondition] != "" {
		policyID, err = strconv.Atoi(s.Conditions[entityPolicyCondition])
		if err != nil {
			return nil, serializer.NewError(serializer.CodeParamErr, "Invalid policy ID", err)
		}
	}

	if s.Conditions[entityTypeCondition] != "" {
		typeId, err := strconv.Atoi(s.Conditions[entityTypeCondition])
		if err != nil {
			return nil, serializer.NewError(serializer.CodeParamErr, "Invalid entity type", err)
		}

		t := types.EntityType(typeId)
		entityType = &t
	}

	res, err := fileClient.ListEntities(ctx, &inventory.ListEntityParameters{
		PaginationArgs: &inventory.PaginationArgs{
			Page:     s.Page - 1,
			PageSize: s.PageSize,
			OrderBy:  s.OrderBy,
			Order:    inventory.OrderDirection(s.OrderDirection),
		},
		UserID:          userID,
		StoragePolicyID: policyID,
		EntityType:      entityType,
	})

	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list entities", err)
	}

	return &ListEntityResponse{
		Pagination: res.PaginationResults,
		Entities: lo.Map(res.Entities, func(entity *ent.Entity, _ int) GetEntityResponse {
			return GetEntityResponse{
				Entity:     entity,
				UserHashID: hashid.EncodeUserID(hasher, entity.CreatedBy),
			}
		}),
	}, nil
}

type (
	SingleEntityService struct {
		ID int `uri:"id" json:"id" binding:"required"`
	}
	SingleEntityParamCtx struct{}
)

func (s *SingleEntityService) Get(c *gin.Context) (*GetEntityResponse, error) {
	dep := dependency.FromContext(c)
	fileClient := dep.FileClient()
	hasher := dep.HashIDEncoder()

	ctx := context.WithValue(c, inventory.LoadEntityUser{}, true)
	ctx = context.WithValue(ctx, inventory.LoadEntityStoragePolicy{}, true)
	ctx = context.WithValue(ctx, inventory.LoadEntityFile{}, true)
	ctx = context.WithValue(ctx, inventory.LoadFileUser{}, true)

	userHashIDMap := make(map[int]string)
	entity, err := fileClient.GetEntityByID(ctx, s.ID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, serializer.NewError(serializer.CodeNotFound, "Entity not found", nil)
		}
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get entity", err)
	}

	for _, file := range entity.Edges.File {
		userHashIDMap[file.OwnerID] = hashid.EncodeUserID(hasher, file.OwnerID)
	}

	return &GetEntityResponse{
		Entity:        entity,
		UserHashID:    hashid.EncodeUserID(hasher, entity.CreatedBy),
		UserHashIDMap: userHashIDMap,
	}, nil
}

type (
	BatchEntityService struct {
		IDs   []int `json:"ids" binding:"min=1"`
		Force bool  `json:"force"`
	}
	BatchEntityParamCtx struct{}
)

func (s *BatchEntityService) Delete(c *gin.Context) error {
	dep := dependency.FromContext(c)
	m := manager.NewFileManager(dep, inventory.UserFromContext(c))
	defer m.Recycle()

	err := m.RecycleEntities(c.Request.Context(), s.Force, s.IDs...)
	if err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to recycle entities", err)
	}

	return nil
}

func (s *SingleEntityService) Url(c *gin.Context) (string, error) {
	dep := dependency.FromContext(c)
	fileClient := dep.FileClient()

	entity, err := fileClient.GetEntityByID(c, s.ID)
	if err != nil {
		return "", serializer.NewError(serializer.CodeDBError, "Failed to get file", err)
	}

	// find policy
	policy, err := dep.StoragePolicyClient().GetPolicyByID(c, entity.StoragePolicyEntities)
	if err != nil {
		return "", serializer.NewError(serializer.CodeDBError, "Failed to get policy", err)
	}

	m := manager.NewFileManager(dep, inventory.UserFromContext(c))
	defer m.Recycle()

	driver, err := m.GetStorageDriver(c, policy)
	if err != nil {
		return "", serializer.NewError(serializer.CodeInternalSetting, "Failed to get storage driver", err)
	}

	es := entitysource.NewEntitySource(fs.NewEntity(entity), driver, policy, dep.GeneralAuth(),
		dep.SettingProvider(), dep.HashIDEncoder(), dep.RequestClient(), dep.Logger(), dep.ConfigProvider(), dep.MimeDetector(c), dep.EncryptorFactory(c))

	expire := time.Now().Add(time.Hour * 1)
	url, err := es.Url(c, entitysource.WithDownload(true), entitysource.WithExpire(&expire), entitysource.WithDisplayName(path.Base(entity.Source)))
	if err != nil {
		return "", serializer.NewError(serializer.CodeInternalSetting, "Failed to get url", err)
	}

	return url.Url, nil
}
