package local

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
)

const (
	Perm = 0744
)

var (
	capabilities = &driver.Capabilities{
		StaticFeatures: &boolset.BooleanSet{},
		MediaMetaProxy: true,
		ThumbProxy:     true,
	}
)

func init() {
	boolset.Sets(map[driver.HandlerCapability]bool{
		driver.HandlerCapabilityProxyRequired: true,
		driver.HandlerCapabilityInboundGet:    true,
	}, capabilities.StaticFeatures)
}

// Driver 本地策略适配器
type Driver struct {
	Policy     *ent.StoragePolicy
	httpClient request.Client
	l          logging.Logger
	config     conf.ConfigProvider
}

// New constructs a new local driver
func New(p *ent.StoragePolicy, l logging.Logger, config conf.ConfigProvider) *Driver {
	return &Driver{
		Policy:     p,
		l:          l,
		httpClient: request.NewClient(config, request.WithLogger(l)),
		config:     config,
	}
}

func (handler *Driver) List(ctx context.Context, path string, onProgress driver.ListProgressFunc, recursive bool) ([]fs.PhysicalObject, error) {
	var res []fs.PhysicalObject
	root := handler.LocalPath(ctx, path)

	err := filepath.Walk(root,
		func(path string, info os.FileInfo, err error) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// Skip root directory
			if path == root {
				return nil
			}

			if err != nil {
				handler.l.Warning("Failed to walk folder %q: %s", path, err)
				return filepath.SkipDir
			}

			// Transform absolute path to relative path
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}

			res = append(res, fs.PhysicalObject{
				Name:         info.Name(),
				RelativePath: filepath.ToSlash(rel),
				Source:       path,
				Size:         info.Size(),
				IsDir:        info.IsDir(),
				LastModify:   info.ModTime(),
			})
			onProgress(1)
			// If not recursive, do not enter directory
			if !recursive && info.IsDir() {
				return filepath.SkipDir
			}

			return nil
		})

	return res, err
}

// Get 获取文件内容
func (handler *Driver) Open(ctx context.Context, path string) (*os.File, error) {
	// 打开文件
	file, err := os.Open(handler.LocalPath(ctx, path))
	if err != nil {
		handler.l.Debug("Failed to open file: %s", err)
		return nil, err
	}

	return file, nil
}

func (handler *Driver) LocalPath(ctx context.Context, path string) string {
	return util.RelativePath(filepath.FromSlash(path))
}

// Put 将文件流保存到指定目录
func (handler *Driver) Put(ctx context.Context, file *fs.UploadRequest) error {
	defer file.Close()
	dst := util.RelativePath(filepath.FromSlash(file.Props.SavePath))

	// 如果非 Overwrite，则检查是否有重名冲突
	if file.Mode&fs.ModeOverwrite != fs.ModeOverwrite {
		if util.Exists(dst) {
			handler.l.Warning("File with the same name existed or unavailable: %s", dst)
			return errors.New("file with the same name existed or unavailable")
		}
	}

	if err := handler.prepareFileDirectory(dst); err != nil {
		return err
	}

	openMode := os.O_CREATE | os.O_RDWR
	// if file.Mode&fs.ModeOverwrite == fs.ModeOverwrite && file.Offset == 0 {
	// 	openMode |= os.O_TRUNC
	// }

	out, err := os.OpenFile(dst, openMode, Perm)
	if err != nil {
		handler.l.Warning("Failed to open or create file: %s", err)
		return err
	}
	defer out.Close()

	stat, err := out.Stat()
	if err != nil {
		handler.l.Warning("Failed to read file info: %s", err)
		return err
	}

	if stat.Size() < file.Offset {
		return errors.New("size of unfinished uploaded chunks is not as expected")
	}

	if _, err := out.Seek(file.Offset, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to desired offset %d: %s", file.Offset, err)
	}

	// 写入文件内容
	_, err = io.Copy(out, file)
	return err
}

// Delete 删除一个或多个文件，
// 返回未删除的文件，及遇到的最后一个错误
func (handler *Driver) Delete(ctx context.Context, files ...string) ([]string, error) {
	deleteFailed := make([]string, 0, len(files))
	var retErr error

	for _, value := range files {
		filePath := util.RelativePath(filepath.FromSlash(value))
		if util.Exists(filePath) {
			err := os.Remove(filePath)
			if err != nil {
				handler.l.Warning("Failed to delete file: %s", err)
				retErr = err
				deleteFailed = append(deleteFailed, value)
			}
		}

		//// 尝试删除文件的缩略图（如果有）
		//_ = os.Remove(util.RelativePath(value + model.GetSettingByNameWithDefault("thumb_file_suffix", "._thumb")))
	}

	return deleteFailed, retErr
}

// Thumb 获取文件缩略图
func (handler *Driver) Thumb(ctx context.Context, expire *time.Time, ext string, e fs.Entity) (string, error) {
	return "", errors.New("not implemented")
}

// Source 获取外链URL
func (handler *Driver) Source(ctx context.Context, e fs.Entity, args *driver.GetSourceArgs) (string, error) {
	return "", errors.New("not implemented")
}

// Token 获取上传策略和认证Token，本地策略直接返回空值
func (handler *Driver) Token(ctx context.Context, uploadSession *fs.UploadSession, file *fs.UploadRequest) (*fs.UploadCredential, error) {
	if file.Mode&fs.ModeOverwrite != fs.ModeOverwrite && util.Exists(uploadSession.Props.SavePath) {
		return nil, errors.New("placeholder file already exist")
	}

	dst := util.RelativePath(filepath.FromSlash(uploadSession.Props.SavePath))
	if err := handler.prepareFileDirectory(dst); err != nil {
		return nil, fmt.Errorf("failed to prepare file directory: %w", err)
	}

	f, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, Perm)
	if err != nil {
		return nil, fmt.Errorf("failed to create placeholder file: %w", err)
	}

	// Preallocate disk space
	defer f.Close()
	if handler.Policy.Settings.PreAllocate {
		if err := Fallocate(f, 0, uploadSession.Props.Size); err != nil {
			handler.l.Warning("Failed to preallocate file: %s", err)
		}
	}

	return &fs.UploadCredential{
		SessionID: uploadSession.Props.UploadSessionID,
		ChunkSize: handler.Policy.Settings.ChunkSize,
	}, nil
}

func (h *Driver) prepareFileDirectory(dst string) error {
	basePath := filepath.Dir(dst)
	if !util.Exists(basePath) {
		err := os.MkdirAll(basePath, Perm)
		if err != nil {
			h.l.Warning("Failed to create directory: %s", err)
			return err
		}
	}

	return nil
}

// 取消上传凭证
func (handler *Driver) CancelToken(ctx context.Context, uploadSession *fs.UploadSession) error {
	return nil
}

func (handler *Driver) CompleteUpload(ctx context.Context, session *fs.UploadSession) error {
	if session.Callback == "" {
		return nil
	}

	if session.Policy.Edges.Node == nil {
		return serializer.NewError(serializer.CodeCallbackError, "Node not found", nil)
	}

	// If callback is set, indicating this handler is used in slave node as a shadowed handler for remote policy,
	// we need to send callback request to master node.
	resp := handler.httpClient.Request(
		"POST",
		session.Callback,
		nil,
		request.WithTimeout(time.Duration(handler.config.Slave().CallbackTimeout)*time.Second),
		request.WithCredential(
			auth.HMACAuth{[]byte(session.Policy.Edges.Node.SlaveKey)},
			int64(handler.config.Slave().SignatureTTL),
		),
		request.WithContext(ctx),
		request.WithCorrelationID(),
	)

	if resp.Err != nil {
		return serializer.NewError(serializer.CodeCallbackError, "Slave cannot send callback request", resp.Err)
	}

	// 解析回调服务端响应
	res, err := resp.DecodeResponse()
	if err != nil {
		msg := fmt.Sprintf("Slave cannot parse callback response from master (StatusCode=%d).", resp.Response.StatusCode)
		return serializer.NewError(serializer.CodeCallbackError, msg, err)
	}

	if res.Code != 0 {
		return serializer.NewError(res.Code, res.Msg, errors.New(res.Error))
	}

	return nil
}

func (handler *Driver) Capabilities() *driver.Capabilities {
	return capabilities
}

func (handler *Driver) MediaMeta(ctx context.Context, path, ext, language string) ([]driver.MediaMeta, error) {
	return nil, errors.New("not implemented")
}
