package oss

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
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
	"github.com/samber/lo"
)

// UploadPolicy 阿里云OSS上传策略
type UploadPolicy struct {
	Expiration string        `json:"expiration"`
	Conditions []interface{} `json:"conditions"`
}

// CallbackPolicy 回调策略
type CallbackPolicy struct {
	CallbackURL      string `json:"callbackUrl"`
	CallbackBody     string `json:"callbackBody"`
	CallbackBodyType string `json:"callbackBodyType"`
	CallbackSNI      bool   `json:"callbackSNI"`
}

// Driver 阿里云OSS策略适配器
type Driver struct {
	policy *ent.StoragePolicy

	client     *oss.Client
	bucket     *oss.Bucket
	settings   setting.Provider
	l          logging.Logger
	config     conf.ConfigProvider
	mime       mime.MimeDetector
	httpClient request.Client

	chunkSize int64
}

type key int

const (
	chunkRetrySleep   = time.Duration(5) * time.Second
	uploadIdParam     = "uploadId"
	partNumberParam   = "partNumber"
	callbackParam     = "callback"
	completeAllHeader = "x-oss-complete-all"
	maxDeleteBatch    = 1000

	// MultiPartUploadThreshold 服务端使用分片上传的阈值
	MultiPartUploadThreshold int64 = 5 * (1 << 30) // 5GB
)

var (
	features = &boolset.BooleanSet{}
)

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

	return driver, driver.InitOSSClient(false)
}

// CORS 创建跨域策略
func (handler *Driver) CORS() error {
	return handler.client.SetBucketCORS(handler.policy.BucketName, []oss.CORSRule{
		{
			AllowedOrigin: []string{"*"},
			AllowedMethod: []string{
				"GET",
				"POST",
				"PUT",
				"DELETE",
				"HEAD",
			},
			ExposeHeader:  []string{},
			AllowedHeader: []string{"*"},
			MaxAgeSeconds: 3600,
		},
	})
}

// InitOSSClient 初始化OSS鉴权客户端
func (handler *Driver) InitOSSClient(forceUsePublicEndpoint bool) error {
	if handler.policy == nil {
		return errors.New("empty policy")
	}

	opt := make([]oss.ClientOption, 0)

	// 决定是否使用内网 Endpoint
	endpoint := handler.policy.Server
	if handler.policy.Settings.ServerSideEndpoint != "" && !forceUsePublicEndpoint {
		endpoint = handler.policy.Settings.ServerSideEndpoint
	} else if handler.policy.Settings.UseCname {
		opt = append(opt, oss.UseCname(true))
	}

	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}

	// 设置签名版本 & 存储桶 region
	opt = append(opt, oss.AuthVersion(oss.AuthV4))
	opt = append(opt, oss.Region(handler.policy.Settings.Region))

	// 初始化客户端
	client, err := oss.New(endpoint, handler.policy.AccessKey, handler.policy.SecretKey, opt...)
	if err != nil {
		return err
	}
	handler.client = client

	// 初始化存储桶
	bucket, err := client.Bucket(handler.policy.BucketName)
	if err != nil {
		return err
	}
	handler.bucket = bucket

	return nil
}

// List 列出OSS上的文件
func (handler *Driver) List(ctx context.Context, base string, onProgress driver.ListProgressFunc, recursive bool) ([]fs.PhysicalObject, error) {
	// 列取文件
	base = strings.TrimPrefix(base, "/")
	if base != "" {
		base += "/"
	}

	var (
		delimiter string
		marker    string
		objects   []oss.ObjectProperties
		commons   []string
	)
	if !recursive {
		delimiter = "/"
	}

	for {
		subRes, err := handler.bucket.ListObjects(oss.Marker(marker), oss.Prefix(base),
			oss.MaxKeys(1000), oss.Delimiter(delimiter))
		if err != nil {
			return nil, err
		}
		objects = append(objects, subRes.Objects...)
		commons = append(commons, subRes.CommonPrefixes...)
		marker = subRes.NextMarker
		if marker == "" {
			break
		}
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
			Size:         object.Size,
			IsDir:        false,
			LastModify:   object.LastModified,
		})
	}
	onProgress(len(res))

	return res, nil
}

// Get 获取文件
func (handler *Driver) Open(ctx context.Context, path string) (*os.File, error) {
	return nil, errors.New("not implemented")
}

// Put 将文件流保存到指定目录
func (handler *Driver) Put(ctx context.Context, file *fs.UploadRequest) error {
	defer file.Close()

	// 凭证有效期
	credentialTTL := handler.settings.UploadSessionTTL(ctx)

	mimeType := file.Props.MimeType
	if mimeType == "" {
		mimeType = handler.mime.TypeByName(file.Props.Uri.Name())
	}

	// 是否允许覆盖
	overwrite := file.Mode&fs.ModeOverwrite == fs.ModeOverwrite
	options := []oss.Option{
		oss.WithContext(ctx),
		oss.Expires(time.Now().Add(credentialTTL * time.Second)),
		oss.ForbidOverWrite(!overwrite),
		oss.ContentType(mimeType),
	}

	// 小文件直接上传
	if file.Props.Size < MultiPartUploadThreshold {
		return handler.bucket.PutObject(file.Props.SavePath, file, options...)
	}

	// 超过阈值时使用分片上传
	imur, err := handler.bucket.InitiateMultipartUpload(file.Props.SavePath, options...)
	if err != nil {
		return fmt.Errorf("failed to initiate multipart upload: %w", err)
	}

	parts := make([]oss.UploadPart, 0)

	chunks := chunk.NewChunkGroup(file, handler.chunkSize, &backoff.ConstantBackoff{
		Max:   handler.settings.ChunkRetryLimit(ctx),
		Sleep: chunkRetrySleep,
	}, handler.settings.UseChunkBuffer(ctx), handler.l, handler.settings.TempPath(ctx))

	uploadFunc := func(current *chunk.ChunkGroup, content io.Reader) error {
		part, err := handler.bucket.UploadPart(imur, content, current.Length(), current.Index()+1, oss.WithContext(ctx))
		if err == nil {
			parts = append(parts, part)
		}
		return err
	}

	for chunks.Next() {
		if err := chunks.Process(uploadFunc); err != nil {
			handler.cancelUpload(imur)
			return fmt.Errorf("failed to upload chunk #%d: %w", chunks.Index(), err)
		}
	}

	_, err = handler.bucket.CompleteMultipartUpload(imur, parts, oss.ForbidOverWrite(!overwrite), oss.WithContext(ctx))
	if err != nil {
		handler.cancelUpload(imur)
	}

	return err
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
		delRes, err := handler.bucket.DeleteObjects(group)
		if err != nil {
			failed = append(failed, group...)
			lastError = err
			continue
		}

		// 统计未删除的文件
		failed = append(failed, util.SliceDifference(files, delRes.DeletedObjects)...)
	}

	if len(failed) > 0 && lastError == nil {
		lastError = fmt.Errorf("failed to delete files: %v", failed)
	}

	return failed, lastError
}

// Thumb 获取文件缩略图
func (handler *Driver) Thumb(ctx context.Context, expire *time.Time, ext string, e fs.Entity) (string, error) {
	usePublicEndpoint := true
	if forceUsePublicEndpoint, ok := ctx.Value(driver.ForceUsePublicEndpointCtx{}).(bool); ok {
		usePublicEndpoint = forceUsePublicEndpoint
	}

	// 初始化客户端
	if err := handler.InitOSSClient(usePublicEndpoint); err != nil {
		return "", err
	}

	w, h := handler.settings.ThumbSize(ctx)
	thumbParam := fmt.Sprintf("image/resize,m_lfit,h_%d,w_%d", h, w)

	enco := handler.settings.ThumbEncode(ctx)
	switch enco.Format {
	case "jpg", "webp":
		thumbParam += fmt.Sprintf("/format,%s/quality,q_%d", enco.Format, enco.Quality)
	case "png":
		thumbParam += fmt.Sprintf("/format,%s", enco.Format)
	}

	thumbOption := []oss.Option{oss.Process(thumbParam)}
	thumbURL, err := handler.signSourceURL(
		ctx,
		e.Source(),
		expire,
		thumbOption,
		false,
	)
	if err != nil {
		return "", err
	}

	return thumbURL, nil
}

// Source 获取外链URL
func (handler *Driver) Source(ctx context.Context, e fs.Entity, args *driver.GetSourceArgs) (string, error) {
	// 初始化客户端
	usePublicEndpoint := true
	if forceUsePublicEndpoint, ok := ctx.Value(driver.ForceUsePublicEndpointCtx{}).(bool); ok {
		usePublicEndpoint = forceUsePublicEndpoint
	}
	if err := handler.InitOSSClient(usePublicEndpoint); err != nil {
		return "", err
	}

	// 添加各项设置
	var signOptions = make([]oss.Option, 0, 2)
	if args.IsDownload {
		encodedFilename := url.PathEscape(args.DisplayName)
		signOptions = append(signOptions, oss.ResponseContentDisposition(fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`,
			encodedFilename, encodedFilename)))
	}
	if args.Speed > 0 {
		// Byte 转换为 bit
		args.Speed *= 8

		// OSS对速度值有范围限制
		if args.Speed < 819200 {
			args.Speed = 819200
		}
		if args.Speed > 838860800 {
			args.Speed = 838860800
		}
		signOptions = append(signOptions, oss.TrafficLimitParam(args.Speed))
	}

	return handler.signSourceURL(ctx, e.Source(), args.Expire, signOptions, false)
}

func (handler *Driver) signSourceURL(ctx context.Context, path string, expire *time.Time, options []oss.Option, forceSign bool) (string, error) {
	ttl := int64(86400 * 7)
	if expire != nil {
		ttl = int64(time.Until(*expire).Seconds())
	}

	signedURL, err := handler.bucket.SignURL(path, oss.HTTPGet, ttl, options...)
	if err != nil {
		return "", err
	}

	// 将最终生成的签名URL域名换成用户自定义的加速域名（如果有）
	finalURL, err := url.Parse(signedURL)
	if err != nil {
		return "", err
	}

	// 公有空间替换掉Key及不支持的头
	if !handler.policy.IsPrivate && !forceSign {
		query := finalURL.Query()
		query.Del("OSSAccessKeyId")
		query.Del("Signature")
		query.Del("response-content-disposition")
		query.Del("x-oss-traffic-limit")
		finalURL.RawQuery = query.Encode()
	}
	return finalURL.String(), nil
}

// Token 获取上传策略和认证Token
func (handler *Driver) Token(ctx context.Context, uploadSession *fs.UploadSession, file *fs.UploadRequest) (*fs.UploadCredential, error) {
	// 初始化客户端
	if err := handler.InitOSSClient(true); err != nil {
		return nil, err
	}

	// 生成回调地址
	siteURL := handler.settings.SiteURL(setting.UseFirstSiteUrl(ctx))
	// 在从机端创建上传会话
	uploadSession.ChunkSize = handler.chunkSize
	uploadSession.Callback = routes.MasterSlaveCallbackUrl(siteURL, types.PolicyTypeOss, uploadSession.Props.UploadSessionID, uploadSession.CallbackSecret).String()

	// 回调策略
	callbackPolicy := CallbackPolicy{
		CallbackURL:      uploadSession.Callback,
		CallbackBody:     `{"name":${x:fname},"source_name":${object},"size":${size},"pic_info":"${imageInfo.width},${imageInfo.height}"}`,
		CallbackBodyType: "application/json",
		CallbackSNI:      true,
	}
	callbackPolicyJSON, err := json.Marshal(callbackPolicy)
	if err != nil {
		return nil, fmt.Errorf("failed to encode callback policy: %w", err)
	}
	callbackPolicyEncoded := base64.StdEncoding.EncodeToString(callbackPolicyJSON)

	mimeType := file.Props.MimeType
	if mimeType == "" {
		mimeType = handler.mime.TypeByName(file.Props.Uri.Name())
	}

	// 初始化分片上传
	options := []oss.Option{
		oss.WithContext(ctx),
		oss.Expires(uploadSession.Props.ExpireAt),
		oss.ForbidOverWrite(true),
		oss.ContentType(mimeType),
	}
	imur, err := handler.bucket.InitiateMultipartUpload(file.Props.SavePath, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize multipart upload: %w", err)
	}
	uploadSession.UploadID = imur.UploadID

	// 为每个分片签名上传 URL
	chunks := chunk.NewChunkGroup(file, handler.chunkSize, &backoff.ConstantBackoff{}, false, handler.l, "")
	urls := make([]string, chunks.Num())
	ttl := int64(time.Until(uploadSession.Props.ExpireAt).Seconds())
	for chunks.Next() {
		err := chunks.Process(func(c *chunk.ChunkGroup, chunk io.Reader) error {
			signedURL, err := handler.bucket.SignURL(file.Props.SavePath, oss.HTTPPut,
				ttl,
				oss.AddParam(partNumberParam, strconv.Itoa(c.Index()+1)),
				oss.AddParam(uploadIdParam, imur.UploadID),
				oss.ContentType("application/octet-stream"))
			if err != nil {
				return err
			}

			urls[c.Index()] = signedURL
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// 签名完成分片上传的URL
	completeURL, err := handler.bucket.SignURL(file.Props.SavePath, oss.HTTPPost, ttl,
		oss.ContentType("application/octet-stream"),
		oss.AddParam(uploadIdParam, imur.UploadID),
		oss.Expires(time.Now().Add(time.Duration(ttl)*time.Second)),
		oss.SetHeader(completeAllHeader, "yes"),
		oss.ForbidOverWrite(true),
		oss.AddParam(callbackParam, callbackPolicyEncoded))
	if err != nil {
		return nil, err
	}

	return &fs.UploadCredential{
		UploadID:    imur.UploadID,
		UploadURLs:  urls,
		CompleteURL: completeURL,
		SessionID:   uploadSession.Props.UploadSessionID,
		ChunkSize:   handler.chunkSize,
	}, nil
}

// 取消上传凭证
func (handler *Driver) CancelToken(ctx context.Context, uploadSession *fs.UploadSession) error {
	return handler.bucket.AbortMultipartUpload(oss.InitiateMultipartUploadResult{UploadID: uploadSession.UploadID, Key: uploadSession.Props.SavePath}, oss.WithContext(ctx))
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

func (handler *Driver) MediaMeta(ctx context.Context, path, ext, language string) ([]driver.MediaMeta, error) {
	if util.ContainsString(supportedImageExt, ext) {
		return handler.extractImageMeta(ctx, path)
	}

	if util.ContainsString(supportedVideoExt, ext) {
		return handler.extractIMMMeta(ctx, path, videoInfoProcess)
	}

	if util.ContainsString(supportedAudioExt, ext) {
		return handler.extractIMMMeta(ctx, path, audioInfoProcess)
	}

	return nil, fmt.Errorf("unsupported media type in oss: %s", ext)
}

func (handler *Driver) LocalPath(ctx context.Context, path string) string {
	return ""
}

func (handler *Driver) cancelUpload(imur oss.InitiateMultipartUploadResult) {
	if err := handler.bucket.AbortMultipartUpload(imur); err != nil {
		handler.l.Warning("failed to abort multipart upload: %s", err)
	}
}
