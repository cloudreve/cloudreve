package qiniu

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/chunk"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/chunk/backoff"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/mime"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"
	"github.com/samber/lo"
)

const (
	chunkRetrySleep   = time.Duration(5) * time.Second
	maxDeleteBatch    = 1000
	trafficLimitParam = "X-Qiniu-Traffic-Limit"
)

var (
	features = &boolset.BooleanSet{}
)

// Driver 本地策略适配器
type Driver struct {
	policy *ent.StoragePolicy

	mac        *qbox.Mac
	cfg        *storage.Config
	bucket     *storage.BucketManager
	settings   setting.Provider
	l          logging.Logger
	config     conf.ConfigProvider
	mime       mime.MimeDetector
	httpClient request.Client

	chunkSize int64
}

func New(ctx context.Context, policy *ent.StoragePolicy, settings setting.Provider,
	config conf.ConfigProvider, l logging.Logger, mime mime.MimeDetector) (*Driver, error) {
	chunkSize := policy.Settings.ChunkSize
	if policy.Settings.ChunkSize == 0 {
		chunkSize = 25 << 20 // 25 MB
	}

	mac := qbox.NewMac(policy.AccessKey, policy.SecretKey)
	cfg := &storage.Config{
		UseHTTPS:      true,
		UseCdnDomains: policy.Settings.QiniuUploadCdn,
	}

	driver := &Driver{
		policy:     policy,
		settings:   settings,
		chunkSize:  chunkSize,
		config:     config,
		l:          l,
		mime:       mime,
		mac:        mac,
		cfg:        cfg,
		bucket:     storage.NewBucketManager(mac, cfg),
		httpClient: request.NewClient(config, request.WithLogger(l)),
	}

	return driver, nil
}

// List 列出给定路径下的文件
func (handler *Driver) List(ctx context.Context, base string, onProgress driver.ListProgressFunc, recursive bool) ([]fs.PhysicalObject, error) {
	base = strings.TrimPrefix(base, "/")
	if base != "" {
		base += "/"
	}

	var (
		delimiter string
		marker    string
		objects   []storage.ListItem
		commons   []string
	)
	if !recursive {
		delimiter = "/"
	}

	for {
		entries, folders, nextMarker, hashNext, err := handler.bucket.ListFiles(
			handler.policy.BucketName,
			base, delimiter, marker, 1000)
		if err != nil {
			return nil, err
		}
		objects = append(objects, entries...)
		commons = append(commons, folders...)
		if !hashNext {
			break
		}
		marker = nextMarker
	}

	// 处理列取结果
	res := make([]fs.PhysicalObject, 0, len(objects)+len(commons))
	// 处理目录
	for _, object := range commons {
		rel, err := filepath.Rel(base, object)
		if err != nil {
			continue
		}
		res = append(res, fs.PhysicalObject{
			Name:         path.Base(object),
			RelativePath: filepath.ToSlash(rel),
			Size:         0,
			IsDir:        true,
			LastModify:   time.Now(),
		})
	}
	onProgress(len(commons))

	// 处理文件
	for _, object := range objects {
		rel, err := filepath.Rel(base, object.Key)
		if err != nil {
			continue
		}
		res = append(res, fs.PhysicalObject{
			Name:         path.Base(object.Key),
			Source:       object.Key,
			RelativePath: filepath.ToSlash(rel),
			Size:         int64(object.Fsize),
			IsDir:        false,
			LastModify:   time.Unix(object.PutTime/10000000, 0),
		})
	}
	onProgress(len(objects))

	return res, nil
}

// Put 将文件流保存到指定目录
func (handler *Driver) Put(ctx context.Context, file *fs.UploadRequest) error {
	defer file.Close()

	// 凭证有效期
	credentialTTL := handler.settings.UploadSessionTTL(ctx)

	// 是否允许覆盖
	overwrite := file.Mode&fs.ModeOverwrite == fs.ModeOverwrite

	// 生成上传策略
	scope := handler.policy.BucketName
	if overwrite {
		scope = fmt.Sprintf("%s:%s", handler.policy.BucketName, file.Props.SavePath)
	}
	putPolicy := storage.PutPolicy{
		// 指定为覆盖策略
		Scope:        scope,
		SaveKey:      file.Props.SavePath,
		ForceSaveKey: true,
		FsizeLimit:   file.Props.Size,
		Expires:      uint64(time.Now().Add(credentialTTL).Unix()),
	}
	upToken := putPolicy.UploadToken(handler.mac)

	// 初始化分片上传
	resumeUploader := storage.NewResumeUploaderV2(handler.cfg)
	upHost, err := resumeUploader.UpHost(handler.policy.AccessKey, handler.policy.BucketName)
	if err != nil {
		return fmt.Errorf("failed to get upload host: %w", err)
	}

	ret := &storage.InitPartsRet{}
	err = resumeUploader.InitParts(ctx, upToken, upHost, handler.policy.BucketName, file.Props.SavePath, true, ret)
	if err != nil {
		return fmt.Errorf("failed to initiate multipart upload: %w", err)
	}

	chunks := chunk.NewChunkGroup(file, handler.chunkSize, &backoff.ConstantBackoff{
		Max:   handler.settings.ChunkRetryLimit(ctx),
		Sleep: chunkRetrySleep,
	}, handler.settings.UseChunkBuffer(ctx), handler.l, handler.settings.TempPath(ctx))

	parts := make([]*storage.UploadPartsRet, 0, chunks.Num())

	uploadFunc := func(current *chunk.ChunkGroup, content io.Reader) error {
		partRet := &storage.UploadPartsRet{}
		err := resumeUploader.UploadParts(
			ctx, upToken, upHost, handler.policy.BucketName, file.Props.SavePath, true, ret.UploadID,
			int64(current.Index()+1), "", partRet, content, int(current.Length()))
		if err == nil {
			parts = append(parts, partRet)
		}
		return err
	}

	for chunks.Next() {
		if err := chunks.Process(uploadFunc); err != nil {
			_ = handler.cancelUpload(upHost, file.Props.SavePath, ret.UploadID, upToken)
			return fmt.Errorf("failed to upload chunk #%d: %w", chunks.Index(), err)
		}
	}

	mimeType := file.Props.MimeType
	if mimeType == "" {
		handler.mime.TypeByName(file.Props.Uri.Name())
	}

	err = resumeUploader.CompleteParts(ctx, upToken, upHost, nil, handler.policy.BucketName,
		file.Props.SavePath, true, ret.UploadID, &storage.RputV2Extra{
			MimeType: mimeType,
			Progresses: lo.Map(parts, func(part *storage.UploadPartsRet, i int) storage.UploadPartInfo {
				return storage.UploadPartInfo{
					Etag:       part.Etag,
					PartNumber: int64(i) + 1,
				}
			}),
		})
	if err != nil {
		_ = handler.cancelUpload(upHost, file.Props.SavePath, ret.UploadID, upToken)
	}
	return nil
}

// Delete 删除一个或多个文件，
// 返回未删除的文件
func (handler *Driver) Delete(ctx context.Context, files ...string) ([]string, error) {
	groups := lo.Chunk(files, maxDeleteBatch)
	failed := make([]string, 0)
	var lastError error

	for index, group := range groups {
		handler.l.Debug("Process delete group #%d: %v", index, group)
		// 删除文件
		rets, err := handler.bucket.BatchWithContext(ctx, handler.policy.BucketName, lo.Map(group, func(key string, index int) string {
			return storage.URIDelete(handler.policy.BucketName, key)
		}))

		// 处理删除结果
		if err != nil {
			for k, ret := range rets {
				if ret.Code != 200 && ret.Code != 612 {
					failed = append(failed, group[k])
					lastError = err
				}
			}
		}
	}

	if len(failed) > 0 && lastError == nil {
		lastError = fmt.Errorf("failed to delete files: %v", failed)
	}

	return failed, lastError
}

// Thumb 获取文件缩略图
func (handler *Driver) Thumb(ctx context.Context, expire *time.Time, ext string, e fs.Entity) (string, error) {
	w, h := handler.settings.ThumbSize(ctx)
	thumbParam := fmt.Sprintf("imageView2/1/w/%d/h/%d", w, h)

	enco := handler.settings.ThumbEncode(ctx)
	switch enco.Format {
	case "jpg", "webp":
		thumbParam += fmt.Sprintf("/format/%s/q/%d", enco.Format, enco.Quality)
	case "png":
		thumbParam += fmt.Sprintf("/format/%s", enco.Format)
	}

	return handler.signSourceURL(
		e.Source(),
		url.Values{
			thumbParam: []string{},
		},
		expire,
	), nil
}

// Source 获取外链URL
func (handler *Driver) Source(ctx context.Context, e fs.Entity, args *driver.GetSourceArgs) (string, error) {
	path := e.Source()

	query := url.Values{}

	// 加入下载相关设置
	if args.IsDownload {
		query.Add("attname", args.DisplayName)
	}

	if args.Speed > 0 {
		// Byte 转换为 bit
		args.Speed *= 8

		// Qiniu 对速度值有范围限制
		if args.Speed < 819200 {
			args.Speed = 819200
		}
		if args.Speed > 838860800 {
			args.Speed = 838860800
		}
		query.Add(trafficLimitParam, fmt.Sprintf("%d", args.Speed))
	}

	// 取得原始文件地址
	return handler.signSourceURL(path, query, args.Expire), nil
}

func (handler *Driver) signSourceURL(path string, query url.Values, expire *time.Time) string {
	var sourceURL string
	deadline := time.Now().Add(time.Duration(24) * time.Hour * 365 * 20).Unix()
	if expire != nil {
		deadline = expire.Unix()
	}

	// If only one query key with empty value, use RawQuery
	firstKey := ""
	for key := range query {
		firstKey = key
		break
	}

	if len(query) == 1 && query.Get(firstKey) == "" {
		if handler.policy.IsPrivate {
			sourceURL = storage.MakePrivateURLv2WithQueryString(handler.mac, handler.policy.Settings.ProxyServer, path, firstKey, deadline)
		} else {
			sourceURL = storage.MakePublicURLv2WithQueryString(handler.policy.Settings.ProxyServer, path, firstKey)
		}
	} else {
		if handler.policy.IsPrivate {
			sourceURL = storage.MakePrivateURLv2WithQuery(handler.mac, handler.policy.Settings.ProxyServer, path, query, deadline)
		} else {
			sourceURL = storage.MakePublicURLv2WithQuery(handler.policy.Settings.ProxyServer, path, query)
		}
	}
	return sourceURL
}

// Token 获取上传策略和认证Token
func (handler *Driver) Token(ctx context.Context, uploadSession *fs.UploadSession, file *fs.UploadRequest) (*fs.UploadCredential, error) {
	// 生成回调地址
	siteURL := handler.settings.SiteURL(setting.UseFirstSiteUrl(ctx))
	apiUrl := routes.MasterSlaveCallbackUrl(siteURL, types.PolicyTypeQiniu, uploadSession.Props.UploadSessionID, uploadSession.CallbackSecret).String()

	// 创建上传策略
	putPolicy := storage.PutPolicy{
		Scope:            fmt.Sprintf("%s:%s", handler.policy.BucketName, file.Props.SavePath),
		CallbackURL:      apiUrl,
		CallbackBody:     `{"size":$(fsize),"pic_info":"$(imageInfo.width),$(imageInfo.height)"}`,
		CallbackBodyType: "application/json",
		SaveKey:          file.Props.SavePath,
		ForceSaveKey:     true,
		FsizeLimit:       file.Props.Size,
		Expires:          uint64(file.Props.ExpireAt.Unix()),
	}

	// 初始化分片上传
	upToken := putPolicy.UploadToken(handler.mac)
	resumeUploader := storage.NewResumeUploaderV2(handler.cfg)
	upHost, err := resumeUploader.UpHost(handler.policy.AccessKey, handler.policy.BucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to get upload host: %w", err)
	}

	ret := &storage.InitPartsRet{}
	err = resumeUploader.InitParts(ctx, upToken, upHost, handler.policy.BucketName, file.Props.SavePath, true, ret)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate multipart upload: %w", err)
	}

	mimeType := file.Props.MimeType
	if mimeType == "" {
		handler.mime.TypeByName(file.Props.Uri.Name())
	}

	uploadSession.UploadID = ret.UploadID
	return &fs.UploadCredential{
		UploadID:   ret.UploadID,
		UploadURLs: []string{getUploadUrl(upHost, handler.policy.BucketName, file.Props.SavePath, ret.UploadID)},
		Credential: upToken,
		SessionID:  uploadSession.Props.UploadSessionID,
		ChunkSize:  handler.chunkSize,
		MimeType:   mimeType,
	}, nil
}

func (handler *Driver) Open(ctx context.Context, path string) (*os.File, error) {
	return nil, errors.New("not implemented")
}

// 取消上传凭证
func (handler *Driver) CancelToken(ctx context.Context, uploadSession *fs.UploadSession) error {
	resumeUploader := storage.NewResumeUploaderV2(handler.cfg)
	return resumeUploader.Client.CallWith(ctx, nil, "DELETE", uploadSession.UploadURL, http.Header{"Authorization": {"UpToken " + uploadSession.Credential}}, nil, 0)
}

func (handler *Driver) CompleteUpload(ctx context.Context, session *fs.UploadSession) error {
	return nil
}

func (handler *Driver) Capabilities() *driver.Capabilities {
	mediaMetaExts := handler.policy.Settings.MediaMetaExts
	if !handler.policy.Settings.NativeMediaProcessing {
		mediaMetaExts = nil
	}
	return &driver.Capabilities{
		StaticFeatures:         features,
		MediaMetaSupportedExts: mediaMetaExts,
		MediaMetaProxy:         handler.policy.Settings.MediaMetaGeneratorProxy,
		ThumbSupportedExts:     handler.policy.Settings.ThumbExts,
		ThumbProxy:             handler.policy.Settings.ThumbGeneratorProxy,
		ThumbSupportAllExts:    handler.policy.Settings.ThumbSupportAllExts,
		ThumbMaxSize:           handler.policy.Settings.ThumbMaxSize,
	}
}

func (handler *Driver) MediaMeta(ctx context.Context, path, ext string) ([]driver.MediaMeta, error) {
	if util.ContainsString(supportedImageExt, ext) {
		return handler.extractImageMeta(ctx, path)
	}

	return handler.extractAvMeta(ctx, path)
}

func (handler *Driver) LocalPath(ctx context.Context, path string) string {
	return ""
}

func (handler *Driver) cancelUpload(upHost, savePath, uploadId, upToken string) error {
	resumeUploader := storage.NewResumeUploaderV2(handler.cfg)
	uploadUrl := getUploadUrl(upHost, handler.policy.BucketName, savePath, uploadId)
	err := resumeUploader.Client.CallWith(context.Background(), nil, "DELETE", uploadUrl, http.Header{"Authorization": {"UpToken " + upToken}}, nil, 0)
	if err != nil {
		handler.l.Error("Failed to cancel upload session for %q: %s", savePath, err)
	}
	return err
}

func getUploadUrl(upHost, bucket, key, uploadId string) string {
	return upHost + "/buckets/" + bucket + "/objects/" + base64.URLEncoding.EncodeToString([]byte(key)) + "/uploads/" + uploadId
}
