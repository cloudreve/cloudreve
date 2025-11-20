package driver

import (
	"context"
	"encoding/gob"
	"os"
	"time"

	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
)

const (
	// HandlerCapabilityProxyRequired this handler requires Cloudreve's proxy to get file content
	HandlerCapabilityProxyRequired HandlerCapability = iota
	// HandlerCapabilityInboundGet this handler supports directly get file's RSCloser, usually
	// indicates that the file is stored in the same machine as Cloudreve
	HandlerCapabilityInboundGet
	// HandlerCapabilityUploadSentinelRequired this handler does not support compliance callback mechanism,
	// thus it requires Cloudreve's sentinel to guarantee the upload is under control. Cloudreve will try
	// to delete the placeholder file and cancel the upload session if upload callback is not made after upload
	// session expire.
	HandlerCapabilityUploadSentinelRequired
)

type (
	MetaType  string
	MediaMeta struct {
		Key   string   `json:"key"`
		Value string   `json:"value"`
		Type  MetaType `json:"type"`
	}

	HandlerCapability int

	GetSourceArgs struct {
		Expire      *time.Time
		IsDownload  bool
		Speed       int64
		DisplayName string
	}

	// Handler 存储策略适配器
	Handler interface {
		// 上传文件, dst为文件存储路径，size 为文件大小。上下文关闭
		// 时，应取消上传并清理临时文件
		Put(ctx context.Context, file *fs.UploadRequest) error

		// 删除一个或多个给定路径的文件，返回删除失败的文件路径列表及错误
		Delete(ctx context.Context, files ...string) ([]string, error)

		// Open physical files. Only implemented if HandlerCapabilityInboundGet capability is set.
		// Returns file path and an os.File object.
		Open(ctx context.Context, path string) (*os.File, error)

		// LocalPath returns the local path of a file.
		// Only implemented if HandlerCapabilityInboundGet capability is set.
		LocalPath(ctx context.Context, path string) string

		// Thumb returns the URL for a thumbnail of given entity.
		Thumb(ctx context.Context, expire *time.Time, ext string, e fs.Entity) (string, error)

		// 获取外链/下载地址，
		// url - 站点本身地址,
		// isDownload - 是否直接下载
		Source(ctx context.Context, e fs.Entity, args *GetSourceArgs) (string, error)

		// Token 获取有效期为ttl的上传凭证和签名
		Token(ctx context.Context, uploadSession *fs.UploadSession, file *fs.UploadRequest) (*fs.UploadCredential, error)

		// CancelToken 取消已经创建的有状态上传凭证
		CancelToken(ctx context.Context, uploadSession *fs.UploadSession) error

		// CompleteUpload completes a previously created upload session.
		CompleteUpload(ctx context.Context, session *fs.UploadSession) error

		// List 递归列取远程端path路径下文件、目录，不包含path本身，
		// 返回的对象路径以path作为起始根目录.
		// recursive - 是否递归列出
		List(ctx context.Context, base string, onProgress ListProgressFunc, recursive bool) ([]fs.PhysicalObject, error)

		// Capabilities returns the capabilities of this handler
		Capabilities() *Capabilities

		// MediaMeta extracts media metadata from the given file.
		MediaMeta(ctx context.Context, path, ext, language string) ([]MediaMeta, error)
	}

	Capabilities struct {
		StaticFeatures *boolset.BooleanSet
		// MaxSourceExpire indicates the maximum allowed expiration duration of a source URL
		MaxSourceExpire time.Duration
		// MinSourceExpire indicates the minimum allowed expiration duration of a source URL
		MinSourceExpire time.Duration
		// MediaMetaSupportedExts indicates the extensions of files that support media metadata. Empty list
		// indicates that no file supports extracting media metadata.
		MediaMetaSupportedExts []string
		// GenerateMediaMeta indicates whether to generate media metadata using local generators.
		MediaMetaProxy bool
		// ThumbSupportedExts indicates the extensions of files that support thumbnail generation. Empty list
		// indicates that no file supports thumbnail generation.
		ThumbSupportedExts []string
		// ThumbSupportAllExts indicates whether to generate thumbnails for all files, regardless of their extensions.
		ThumbSupportAllExts bool
		// ThumbMaxSize indicates the maximum allowed size of a thumbnail. 0 indicates that no limit is set.
		ThumbMaxSize int64
		// ThumbProxy indicates whether to generate thumbnails using local generators.
		ThumbProxy bool
		// BrowserRelayedDownload indicates whether to relay download via stream-saver.
		BrowserRelayedDownload bool
	}

	ListProgressFunc func(int)
)

const (
	MetaTypeExif        MetaType = "exif"
	MediaTypeMusic      MetaType = "music"
	MetaTypeStreamMedia MetaType = "stream"
	MetaTypeGeocoding   MetaType = "geocoding"
)

type ForceUsePublicEndpointCtx struct{}

// WithForcePublicEndpoint sets the context to force using public endpoint for supported storage policies.
func WithForcePublicEndpoint(ctx context.Context, value bool) context.Context {
	return context.WithValue(ctx, ForceUsePublicEndpointCtx{}, value)
}

func init() {
	gob.Register(fs.PhysicalObject{})
}
