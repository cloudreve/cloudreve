package setting

import (
	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
)

// WebDAVAccountService WebDAV 账号管理服务
type WebDAVAccountService struct {
	ID uint `uri:"id" binding:"required,min=1"`
}

// WebDAVAccountCreateService WebDAV 账号创建服务
type WebDAVAccountCreateService struct {
	Path string `json:"path" binding:"required,min=1,max=65535"`
	Name string `json:"name" binding:"required,min=1,max=255"`
}

// WebDAVAccountUpdateReadonlyService WebDAV 修改只读性服务
type WebDAVAccountUpdateReadonlyService struct {
	ID       uint `json:"id" binding:"required,min=1"`
	Readonly bool `json:"readonly"`
}

// WebDAVMountCreateService WebDAV 挂载创建服务
type WebDAVMountCreateService struct {
	Path   string `json:"path" binding:"required,min=1,max=65535"`
	Policy string `json:"policy" binding:"required,min=1"`
}

//// Unmount 取消目录挂载
//func (service *WebDAVListService) Unmount(c *gin.Context, user *model.User) serializer.Response {
//	folderID, _ := c.Get("object_id")
//	folder, err := model.GetFoldersByIDs([]uint{folderID.(uint)}, user.ID)
//	if err != nil || len(folder) == 0 {
//		return serializer.ErrDeprecated(serializer.CodeParentNotExist, "", err)
//	}
//
//	if err := folder[0].Mount(0); err != nil {
//		return serializer.DBErrDeprecated("Failed to update folder record", err)
//	}
//
//	return serializer.Response{}
//}

type (
	ListDavAccountsService struct {
		PageSize      int    `form:"page_size" binding:"required,min=10,max=100"`
		NextPageToken string `form:"next_page_token"`
	}
	ListDavAccountParamCtx struct{}
)

// Accounts 列出WebDAV账号
func (service *ListDavAccountsService) List(c *gin.Context) (*ListDavAccountResponse, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	hasher := dep.HashIDEncoder()
	davAccountClient := dep.DavAccountClient()

	args := &inventory.ListDavAccountArgs{
		UserID: user.ID,
		PaginationArgs: &inventory.PaginationArgs{
			UseCursorPagination: true,
			PageSize:            service.PageSize,
			PageToken:           service.NextPageToken,
		},
	}

	res, err := davAccountClient.List(c, args)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list dav accounts", err)
	}

	return BuildListDavAccountResponse(res, hasher), nil
}

type (
	CreateDavAccountService struct {
		Uri             string `json:"uri" binding:"required"`
		Name            string `json:"name" binding:"required,min=1,max=255"`
		Readonly        bool   `json:"readonly"`
		Proxy           bool   `json:"proxy"`
		DisableSysFiles bool   `json:"disable_sys_files"`
	}
	CreateDavAccountParamCtx struct{}
)

// Create 创建WebDAV账号
func (service *CreateDavAccountService) Create(c *gin.Context) (*DavAccount, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)

	bs, err := service.validateAndGetBs(user)
	if err != nil {
		return nil, err
	}

	davAccountClient := dep.DavAccountClient()
	account, err := davAccountClient.Create(c, &inventory.CreateDavAccountParams{
		UserID:   user.ID,
		Name:     service.Name,
		URI:      service.Uri,
		Password: util.RandString(32, util.RandomLowerCases),
		Options:  bs,
	})
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to create dav account", err)
	}

	accountRes := BuildDavAccount(account, dep.HashIDEncoder())
	return &accountRes, nil
}

// Update updates an existing account
func (service *CreateDavAccountService) Update(c *gin.Context) (*DavAccount, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	accountId := hashid.FromContext(c)

	// Find existing account
	davAccountClient := dep.DavAccountClient()
	account, err := davAccountClient.GetByIDAndUserID(c, accountId, user.ID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "Account not exist", err)
	}

	bs, err := service.validateAndGetBs(user)
	if err != nil {
		return nil, err
	}

	// Update account
	account, err = davAccountClient.Update(c, accountId, &inventory.CreateDavAccountParams{
		Name:    service.Name,
		URI:     service.Uri,
		Options: bs,
	})
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to update dav account", err)
	}

	accountRes := BuildDavAccount(account, dep.HashIDEncoder())
	return &accountRes, nil
}

func (service *CreateDavAccountService) validateAndGetBs(user *ent.User) (*boolset.BooleanSet, error) {
	if !user.Edges.Group.Permissions.Enabled(int(types.GroupPermissionWebDAV)) {
		return nil, serializer.NewError(serializer.CodeGroupNotAllowed, "WebDAV is not enabled for this user group", nil)
	}

	uri, err := fs.NewUriFromString(service.Uri)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "Invalid URI", err)
	}

	// Only "my" and "share" fs is allowed in WebDAV
	if uriFs := uri.FileSystem(); uri.SearchParameters() != nil ||
		(uriFs != constants.FileSystemMy && uriFs != constants.FileSystemShare) {
		return nil, serializer.NewError(serializer.CodeParamErr, "Invalid URI", nil)
	}

	bs := boolset.BooleanSet{}
	if service.Readonly {
		boolset.Set(types.DavAccountReadOnly, true, &bs)
	}

	if service.DisableSysFiles {
		boolset.Set(types.DavAccountDisableSysFiles, true, &bs)
	}

	if service.Proxy && user.Edges.Group.Permissions.Enabled(int(types.GroupPermissionWebDAVProxy)) {
		boolset.Set(types.DavAccountProxy, true, &bs)
	}
	return &bs, nil
}

func DeleteDavAccount(c *gin.Context) error {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	accountId := hashid.FromContext(c)

	// Find existing account
	davAccountClient := dep.DavAccountClient()
	_, err := davAccountClient.GetByIDAndUserID(c, accountId, user.ID)
	if err != nil {
		return serializer.NewError(serializer.CodeNotFound, "Account not exist", err)
	}

	if err := davAccountClient.Delete(c, accountId); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to delete dav account", err)
	}

	return nil
}
