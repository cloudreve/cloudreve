package cos

import (
	"context"
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
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/google/go-querystring/query"
	"github.com/samber/lo"
	cossdk "github.com/tencentyun/cos-go-sdk-v5"
)

// UploadPolicy 腾讯云COS上传策略
type UploadPolicy struct {
	Expiration string        `json:"expiration"`
	Conditions []interface{} `json:"conditions"`
}

// MetaData 文件元信息
type MetaData struct {
	Size        uint64
	CallbackKey string
	CallbackURL string
}

type urlOption struct {
	Speed              int64   `url:"x-cos-traffic-limit,omitempty"`
	ContentDescription string  `url:"response-content-disposition,omitempty"`
	Exif               *string `url:"exif,omitempty"`
	CiProcess          string  `url:"ci-process,omitempty"`
}

type (
	CosParts struct {
		ETag       string
		PartNumber int
	}
)

// Driver 腾讯云COS适配器模板
type Driver struct {
	policy     *ent.StoragePolicy
	client     *cossdk.Client
	settings   setting.Provider
	config     conf.ConfigProvider
	httpClient request.Client
	l          logging.Logger
	mime       mime.MimeDetector

	chunkSize int64
}

const (
	// MultiPartUploadThreshold 服务端使用分片上传的阈值
	MultiPartUploadThreshold int64 = 5 * (1 << 30) // 5GB

	maxDeleteBatch        = 1000
	chunkRetrySleep       = time.Duration(5) * time.Second
	overwriteOptionHeader = "x-cos-forbid-overwrite"
	partNumberParam       = "partNumber"
	uploadIdParam         = "uploadId"
	contentTypeHeader     = "Content-Type"
	contentLengthHeader   = "Content-Length"
)

var (
	features = &boolset.BooleanSet{}
)

func init() {
	cossdk.SetNeedSignHeaders("host", false)
	cossdk.SetNeedSignHeaders("origin", false)
	boolset.Sets(map[driver.HandlerCapability]bool{
		driver.HandlerCapabilityUploadSentinelRequired: true,
	}, features)
}

func New(ctx context.Context, policy *ent.StoragePolicy, settings setting.Provider,
	config conf.ConfigProvider, l logging.Logger, mime mime.MimeDetector) (*Driver, error) {
	chunkSize := policy.Settings.ChunkSize
	if policy.Settings.ChunkSize == 0 {
		chunkSize = 25 << 20 // 25 MB
	}

	driver := &Driver{
		policy:     policy,
		settings:   settings,
		chunkSize:  chunkSize,
		config:     config,
		l:          l,
		mime:       mime,
		httpClient: request.NewClient(config, request.WithLogger(l)),
	}

	u, err := url.Parse(policy.Server)
	if err != nil {
		return nil, fmt.Errorf("failed to parse COS bucket server url: %w", err)
	}
	driver.client = cossdk.NewClient(&cossdk.BaseURL{BucketURL: u}, &http.Client{
		Transport: &cossdk.AuthorizationTransport{
			SecretID:  policy.AccessKey,
			SecretKey: policy.SecretKey,
		},
	})

	return driver, nil
}

func (handler *Driver) List(ctx context.Context, base string, onProgress driver.ListProgressFunc, recursive bool) ([]fs.PhysicalObject, error) {
	// 初始化列目录参数
	opt := &cossdk.BucketGetOptions{
		Prefix:       strings.TrimPrefix(base, "/"),
		EncodingType: "",
		MaxKeys:      1000,
	}

	// 是否为递归列出

	if !recursive {
		opt.Delimiter = "/"
	}

	// 手动补齐结尾的slash
	if opt.Prefix != "" {
		opt.Prefix += "/"
	}

	var (
		marker  string
		objects []cossdk.Object
		commons []string
	)

	for {
		res, _, err := handler.client.Bucket.Get(ctx, opt)
		if err != nil {
			handler.l.Warning("Failed to list objects: %s", err)
			return nil, err
		}
		objects = append(objects, res.Contents...)
		commons = append(commons, res.CommonPrefixes...)
		// 如果本次未列取完，则继续使用marker获取结果
		marker = res.NextMarker
		// marker 为空时结果列取完毕，跳出
		if marker == "" {
			break
		}
	}

	// 处理列取结果
	res := make([]fs.PhysicalObject, 0, len(objects)+len(commons))
	// 处理目录

	for _, object := range commons {
		rel, err := filepath.Rel(opt.Prefix, object)
		if err != nil {
			handler.l.Warning("Failed to get relative path: %s", err)
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
		rel, err := filepath.Rel(opt.Prefix, object.Key)
		if err != nil {
			handler.l.Warning("Failed to get relative path: %s", err)
			continue
		}
		res = append(res, fs.PhysicalObject{
			Name:         path.Base(object.Key),
			Source:       object.Key,
			RelativePath: filepath.ToSlash(rel),
			Size:         object.Size,
			IsDir:        false,
			LastModify:   time.Now(),
		})
	}
	onProgress(len(res))

	return res, nil
}

// CORS 创建跨域策略
func (handler Driver) CORS() error {
	_, err := handler.client.Bucket.PutCORS(context.Background(), &cossdk.BucketPutCORSOptions{
		Rules: []cossdk.BucketCORSRule{{
			AllowedMethods: []string{
				"GET",
				"POST",
				"PUT",
				"DELETE",
				"HEAD",
			},
			AllowedOrigins: []string{"*"},
			AllowedHeaders: []string{"*"},
			MaxAgeSeconds:  3600,
			ExposeHeaders:  []string{"ETag"},
		}},
	})

	return err
}

// Get 获取文件
func (handler *Driver) Open(ctx context.Context, path string) (*os.File, error) {
	return nil, errors.New("not implemented")
}

// Put 将文件流保存到指定目录
func (handler *Driver) Put(ctx context.Context, file *fs.UploadRequest) error {
	defer file.Close()

	mimeType := file.Props.MimeType
	if mimeType == "" {
		mimeType = handler.mime.TypeByName(file.Props.Uri.Name())
	}

	// 是否允许覆盖
	overwrite := file.Mode&fs.ModeOverwrite == fs.ModeOverwrite
	opt := &cossdk.ObjectPutHeaderOptions{
		ContentType: mimeType,
		XOptionHeader: &http.Header{
			overwriteOptionHeader: []string{fmt.Sprintf("%t", overwrite)},
		},
	}

	// 小文件直接上传
	if file.Props.Size < MultiPartUploadThreshold {
		_, err := handler.client.Object.Put(ctx, file.Props.SavePath, file, &cossdk.ObjectPutOptions{
			ObjectPutHeaderOptions: opt,
		})
		return err
	}

	imur, _, err := handler.client.Object.InitiateMultipartUpload(ctx, file.Props.SavePath, &cossdk.InitiateMultipartUploadOptions{
		ObjectPutHeaderOptions: opt,
	})

	chunks := chunk.NewChunkGroup(file, handler.chunkSize, &backoff.ConstantBackoff{
		Max:   handler.settings.ChunkRetryLimit(ctx),
		Sleep: chunkRetrySleep,
	}, handler.settings.UseChunkBuffer(ctx), handler.l, handler.settings.TempPath(ctx))

	parts := make([]CosParts, 0, chunks.Num())
	uploadFunc := func(current *chunk.ChunkGroup, content io.Reader) error {
		res, err := handler.client.Object.UploadPart(ctx, file.Props.SavePath, imur.UploadID, current.Index()+1, content, &cossdk.ObjectUploadPartOptions{
			ContentLength: current.Length(),
		})
		if err == nil {
			parts = append(parts, CosParts{
				ETag:       res.Header.Get("ETag"),
				PartNumber: current.Index() + 1,
			})
		}
		return err
	}

	for chunks.Next() {
		if err := chunks.Process(uploadFunc); err != nil {
			handler.cancelUpload(file.Props.SavePath, imur.UploadID)
			return fmt.Errorf("failed to upload chunk #%d: %w", chunks.Index(), err)
		}
	}

	_, _, err = handler.client.Object.CompleteMultipartUpload(ctx, file.Props.SavePath, imur.UploadID, &cossdk.CompleteMultipartUploadOptions{
		Parts: lo.Map(parts, func(v CosParts, i int) cossdk.Object {
			return cossdk.Object{
				ETag:       v.ETag,
				PartNumber: v.PartNumber,
			}
		}),
		XOptionHeader: &http.Header{
			overwriteOptionHeader: []string{fmt.Sprintf("%t", overwrite)},
		},
	})

	if err != nil {
		handler.cancelUpload(file.Props.SavePath, imur.UploadID)
	}

	return err
}

// Delete 删除一个或多个文件，
// 返回未删除的文件，及遇到的最后一个错误
func (handler Driver) Delete(ctx context.Context, files ...string) ([]string, error) {
	groups := lo.Chunk(files, maxDeleteBatch)
	failed := make([]string, 0)
	var lastError error
	for index, group := range groups {
		handler.l.Debug("Process delete group #%d: %v", index, group)
		res, _, err := handler.client.Object.DeleteMulti(ctx,
			&cossdk.ObjectDeleteMultiOptions{
				Objects: lo.Map(group, func(item string, index int) cossdk.Object {
					return cossdk.Object{Key: item}
				}),
				Quiet: true,
			})
		if err != nil {
			lastError = err
			failed = append(failed, group...)
			continue
		}

		for _, v := range res.Errors {
			handler.l.Debug("Failed to delete file: %s, Code:%s, Message:%s", v.Key, v.Code, v.Key)
			failed = append(failed, v.Key)
		}
	}

	if len(failed) > 0 && lastError == nil {
		lastError = fmt.Errorf("failed to delete files: %v", failed)
	}

	return failed, lastError
}

// Thumb 获取文件缩略图
func (handler Driver) Thumb(ctx context.Context, expire *time.Time, ext string, e fs.Entity) (string, error) {
	w, h := handler.settings.ThumbSize(ctx)
	thumbParam := fmt.Sprintf("imageMogr2/thumbnail/%dx%d", w, h)

	enco := handler.settings.ThumbEncode(ctx)
	switch enco.Format {
	case "jpg", "webp":
		thumbParam += fmt.Sprintf("/format/%s/rquality/%d", enco.Format, enco.Quality)
	case "png":
		thumbParam += fmt.Sprintf("/format/%s", enco.Format)
	}

	source, err := handler.signSourceURL(
		ctx,
		e.Source(),
		expire,
		&urlOption{},
	)
	if err != nil {
		return "", err
	}

	thumbURL, _ := url.Parse(source)
	thumbQuery := thumbURL.Query()
	thumbQuery.Add(thumbParam, "")
	thumbURL.RawQuery = thumbQuery.Encode()

	return thumbURL.String(), nil
}

// Source 获取外链URL
func (handler Driver) Source(ctx context.Context, e fs.Entity, args *driver.GetSourceArgs) (string, error) {
	// 添加各项设置
	options := urlOption{}

	if args.Speed > 0 {
		// Byte 转换为 bit
		args.Speed *= 8

		// COS对速度值有范围限制
		if args.Speed < 819200 {
			args.Speed = 819200
		}
		if args.Speed > 838860800 {
			args.Speed = 838860800
		}
		options.Speed = args.Speed
	}

	if args.IsDownload {
		encodedFilename := url.PathEscape(args.DisplayName)
		options.ContentDescription = fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`,
			encodedFilename, encodedFilename)
	}

	return handler.signSourceURL(ctx, e.Source(), args.Expire, &options)
}

func (handler Driver) signSourceURL(ctx context.Context, path string, expire *time.Time, options *urlOption) (string, error) {
	// 公有空间不需要签名
	if !handler.policy.IsPrivate || (handler.policy.Settings.SourceAuth && handler.policy.Settings.CustomProxy) {
		file, err := url.Parse(handler.policy.Server)
		if err != nil {
			return "", err
		}

		file.Path = path

		// 非签名URL不支持设置响应header
		options.ContentDescription = ""

		optionQuery, err := query.Values(*options)
		if err != nil {
			return "", err
		}
		file.RawQuery = optionQuery.Encode()

		return file.String(), nil
	}

	ttl := time.Duration(0)
	if expire != nil {
		ttl = time.Until(*expire)
	} else {
		// 20 years for permanent link
		ttl = time.Duration(24) * time.Hour * 365 * 20
	}

	presignedURL, err := handler.client.Object.GetPresignedURL(ctx, http.MethodGet, path,
		handler.policy.AccessKey, handler.policy.SecretKey, ttl, options)
	if err != nil {
		return "", err
	}

	return presignedURL.String(), nil
}

// Token 获取上传策略和认证Token
func (handler Driver) Token(ctx context.Context, uploadSession *fs.UploadSession, file *fs.UploadRequest) (*fs.UploadCredential, error) {
	// 生成回调地址
	siteURL := handler.settings.SiteURL(setting.UseFirstSiteUrl(ctx))
	// 在从机端创建上传会话
	uploadSession.ChunkSize = handler.chunkSize
	uploadSession.Callback = routes.MasterSlaveCallbackUrl(siteURL, types.PolicyTypeCos, uploadSession.Props.UploadSessionID, uploadSession.CallbackSecret).String()

	mimeType := file.Props.MimeType
	if mimeType == "" {
		mimeType = handler.mime.TypeByName(file.Props.Uri.Name())
	}

	// 初始化分片上传
	opt := &cossdk.ObjectPutHeaderOptions{
		ContentType: mimeType,
		XOptionHeader: &http.Header{
			overwriteOptionHeader: []string{"true"},
		},
	}

	imur, _, err := handler.client.Object.InitiateMultipartUpload(ctx, file.Props.SavePath, &cossdk.InitiateMultipartUploadOptions{
		ObjectPutHeaderOptions: opt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize multipart upload: %w", err)
	}
	uploadSession.UploadID = imur.UploadID

	// 为每个分片签名上传 URL
	chunks := chunk.NewChunkGroup(file, handler.chunkSize, &backoff.ConstantBackoff{}, false, handler.l, "")
	urls := make([]string, chunks.Num())
	ttl := time.Until(uploadSession.Props.ExpireAt)
	for chunks.Next() {
		err := chunks.Process(func(c *chunk.ChunkGroup, chunk io.Reader) error {
			signedURL, err := handler.client.Object.GetPresignedURL(
				ctx,
				http.MethodPut,
				file.Props.SavePath,
				handler.policy.AccessKey,
				handler.policy.SecretKey,
				ttl,
				&cossdk.PresignedURLOptions{
					Query: &url.Values{
						partNumberParam: []string{fmt.Sprintf("%d", c.Index()+1)},
						uploadIdParam:   []string{imur.UploadID},
					},
					Header: &http.Header{
						contentTypeHeader:   []string{"application/octet-stream"},
						contentLengthHeader: []string{fmt.Sprintf("%d", c.Length())},
					},
				})
			if err != nil {
				return err
			}

			urls[c.Index()] = signedURL.String()
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// 签名完成分片上传的URL
	completeURL, err := handler.client.Object.GetPresignedURL(
		ctx,
		http.MethodPost,
		file.Props.SavePath,
		handler.policy.AccessKey,
		handler.policy.SecretKey,
		time.Until(uploadSession.Props.ExpireAt),
		&cossdk.PresignedURLOptions{
			Query: &url.Values{
				uploadIdParam: []string{imur.UploadID},
			},
			Header: &http.Header{
				overwriteOptionHeader: []string{"true"},
			},
		})
	if err != nil {
		return nil, err
	}

	return &fs.UploadCredential{
		UploadID:    imur.UploadID,
		UploadURLs:  urls,
		CompleteURL: completeURL.String(),
		SessionID:   uploadSession.Props.UploadSessionID,
		ChunkSize:   handler.chunkSize,
	}, nil
}

// 取消上传凭证
func (handler *Driver) CancelToken(ctx context.Context, uploadSession *fs.UploadSession) error {
	_, err := handler.client.Object.AbortMultipartUpload(ctx, uploadSession.Props.SavePath, uploadSession.UploadID)
	return err
}

func (handler *Driver) CompleteUpload(ctx context.Context, session *fs.UploadSession) error {
	if session.SentinelTaskID == 0 {
		return nil
	}

	// Make sure uploaded file size is correct
	res, err := handler.client.Object.Head(ctx, session.Props.SavePath, &cossdk.ObjectHeadOptions{})
	if err != nil {
		return fmt.Errorf("failed to get uploaded file size: %w", err)
	}

	if res.ContentLength != session.Props.Size {
		return serializer.NewError(
			serializer.CodeMetaMismatch,
			fmt.Sprintf("File size not match, expected: %d, actual: %d", session.Props.Size, res.ContentLength),
			nil,
		)
	}
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
		ThumbMaxSize:           handler.policy.Settings.ThumbMaxSize,
		ThumbSupportAllExts:    handler.policy.Settings.ThumbSupportAllExts,
	}
}

// Meta 获取文件信息
func (handler Driver) Meta(ctx context.Context, path string) (*MetaData, error) {
	res, err := handler.client.Object.Head(ctx, path, &cossdk.ObjectHeadOptions{})
	if err != nil {
		return nil, err
	}
	return &MetaData{
		Size:        uint64(res.ContentLength),
		CallbackKey: res.Header.Get("x-cos-meta-key"),
		CallbackURL: res.Header.Get("x-cos-meta-callback"),
	}, nil
}

func (handler *Driver) MediaMeta(ctx context.Context, path, ext, language string) ([]driver.MediaMeta, error) {
	if util.ContainsString(supportedImageExt, ext) {
		return handler.extractImageMeta(ctx, path)
	}

	return handler.extractStreamMeta(ctx, path)
}

func (handler *Driver) LocalPath(ctx context.Context, path string) string {
	return ""
}

func (handler *Driver) cancelUpload(path, uploadId string) {
	if _, err := handler.client.Object.AbortMultipartUpload(context.Background(), path, uploadId); err != nil {
		handler.l.Warning("failed to abort multipart upload: %s", err)
	}
}
