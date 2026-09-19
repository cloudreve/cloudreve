package callback

import (
	"fmt"
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver/onedrive"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/gin-gonic/gin"
)

// RemoteUploadCallbackService 远程存储上传回调请求服务
type RemoteUploadCallbackService struct {
}

// UploadCallbackService OOS/七牛云存储上传回调请求服务
type UploadCallbackService struct {
	Name       string `json:"name"`
	SourceName string `json:"source_name"`
	PicInfo    string `json:"pic_info"`
	Size       int64  `json:"size"`
}

// UpyunCallbackService 又拍云上传回调请求服务
type UpyunCallbackService struct {
	Code       int    `form:"code" binding:"required"`
	Message    string `form:"message" binding:"required"`
	SourceName string `form:"url" binding:"required"`
	Width      string `form:"image-width"`
	Height     string `form:"image-height"`
	Size       uint64 `form:"file_size"`
}

// OneDriveCallback OneDrive 客户端回调正文
type OneDriveCallback struct {
	Meta *onedrive.FileInfo
}

// S3Callback S3 客户端回调正文
type S3Callback struct {
}

// ProcessCallback 处理上传结果回调
func ProcessCallback(c *gin.Context) error {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	m := manager.NewFileManager(dep, user)
	defer m.Recycle()

	// 获取上传会话
	uploadSession := c.MustGet(manager.UploadSessionCtx).(*fs.UploadSession)

	_, err := m.CompleteUpload(c, uploadSession)
	if err != nil {
		return fmt.Errorf("failed to complete upload: %w", err)
	}

	return nil
}

//// PreProcess 对OneDrive客户端回调进行预处理验证
//func (service *OneDriveCallback) PreProcess(c *gin.Context) serializer.Response {
//	// 创建文件系统
//	fs, err := filesystem.NewFileSystemFromCallback(c)
//	if err != nil {
//	}
//	defer fs.Recycle()
//
//	// 获取回调会话
//	uploadSession := c.MustGet(filesystem.UploadSessionCtx).(*serializer.UploadSession)
//
//	// 获取文件信息
//	info, err := fs.Handler.(onedrive.Driver).Client.Meta(context.Background(), "", uploadSession.SavePath)
//	if err != nil {
//	}
//
//	// 验证与回调会话中是否一致
//	actualPath := strings.TrimPrefix(uploadSession.SavePath, "/")
//	isSizeCheckFailed := uploadSession.Size != info.Size
//
//	// SharePoint 会对 Office 文档增加 meta data 导致文件大小不一致，这里增加 1 MB 宽容
//	// See: https://github.com/OneDrive/onedrive-api-docs/issues/935
//	if (strings.Contains(fs.Policy.OptionsSerialized.OdDriver, "sharepoint.com") || strings.Contains(fs.Policy.OptionsSerialized.OdDriver, "sharepoint.cn")) && isSizeCheckFailed && (info.Size > uploadSession.Size) && (info.Size-uploadSession.Size <= 1048576) {
//		isSizeCheckFailed = false
//	}
//
//	if isSizeCheckFailed || !strings.EqualFold(info.GetSourcePath(), actualPath) {
//		fs.Handler.(onedrive.Driver).Client.Delete(context.Background(), []string{info.GetSourcePath()})
//	}
//	service.Meta = info
//	return ProcessCallback(c)
//}
//

//
//// PreProcess 对S3客户端回调进行预处理
//func (service *S3Callback) PreProcess(c *gin.Context) serializer.Response {
//	// 创建文件系统
//	fs, err := filesystem.NewFileSystemFromCallback(c)
//	if err != nil {
//	}
//	defer fs.Recycle()
//
//	// 获取回调会话
//	uploadSession := c.MustGet(filesystem.UploadSessionCtx).(*serializer.UploadSession)
//
//	// 获取文件信息
//	info, err := fs.Handler.(*s3.Driver).Meta(context.Background(), uploadSession.SavePath)
//	if err != nil {
//	}
//
//	// 验证实际文件信息与回调会话中是否一致
//	if uploadSession.Size != info.Size {
//	}
//
//	return ProcessCallback(service, c)
//}
//
//// PreProcess 对OneDrive客户端回调进行预处理验证
//func (service *UploadCallbackService) PreProcess(c *gin.Context) serializer.Response {
//	// 创建文件系统
//	fs, err := filesystem.NewFileSystemFromCallback(c)
//	if err != nil {
//	}
//	defer fs.Recycle()
//
//	// 获取回调会话
//	uploadSession := c.MustGet(filesystem.UploadSessionCtx).(*serializer.UploadSession)
//
//	// 验证文件大小
//	if uploadSession.Size != service.Size {
//		fs.Handler.Delete(context.Background(), []string{uploadSession.SavePath})
//	}
//
//	return ProcessCallback(service, c)
//}
