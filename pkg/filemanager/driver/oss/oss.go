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

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
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
	settings   setting.Provider
	l          logging.Logger
	config     conf.ConfigProvider
	mime       mime.MimeDetector
	httpClient request.Client

	chunkSize int64
}

type key int

const (
	chunkRetrySleep       = time.Duration(5) * time.Second
	maxDeleteBatch        = 1000
	completeAllHeader     = "x-oss-complete-all"
	forbidOverwriteHeader = "x-oss-forbid-overwrite"
	trafficLimitHeader    = "x-oss-traffic-limit"

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
	_, err := handler.client.PutBucketCors(context.Background(), &oss.PutBucketCorsRequest{
		Bucket: &handler.policy.BucketName,
		CORSConfiguration: &oss.CORSConfiguration{
			CORSRules: []oss.CORSRule{
				{
					AllowedOrigins: []string{"*"},
					AllowedMethods: []string{
						"GET",
						"POST",
						"PUT",
						"DELETE",
						"HEAD",
					},
					ExposeHeaders:  []string{},
					AllowedHeaders: []string{"*"},
					MaxAgeSeconds:  oss.Ptr(int64(3600)),
				},
			},
		}})

	return err
}

// InitOSSClient 初始化OSS鉴权客户端
func (handler *Driver) InitOSSClient(forceUsePublicEndpoint bool) error {
	if handler.policy == nil {
		return errors.New("empty policy")
	}

	// 决定是否使用内网 Endpoint
	endpoint := handler.policy.Server
	useCname := false
	if handler.policy.Settings.ServerSideEndpoint != "" && !forceUsePublicEndpoint {
		endpoint = handler.policy.Settings.ServerSideEndpoint
	} else if handler.policy.Settings.UseCname {
		useCname = true
	}

	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}

	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewStaticCredentialsProvider(handler.policy.AccessKey, handler.policy.SecretKey, "")).
		WithEndpoint(endpoint).
		WithRegion(handler.policy.Settings.Region).
		WithUseCName(useCname)

	// 初始化客户端
	client := oss.NewClient(cfg)
	handler.client = client
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
		objects   []oss.ObjectProperties
		commons   []oss.CommonPrefix
	)
	if !recursive {
		delimiter = "/"
	}

	p := handler.client.NewListObjectsPaginator(&oss.ListObjectsRequest{
		Bucket:    &handler.policy.BucketName,
		Prefix:    &base,
		MaxKeys:   1000,
		Delimiter: &delimiter,
	})

	for p.HasNext() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		objects = append(objects, page.Contents...)
		commons = append(commons, page.CommonPrefixes...)
	}

	// 处理列取结果
	res := make([]fs.PhysicalObject, 0, len(objects)+len(commons))
	// 处理目录
	for _, object := range commons {
		rel, err := filepath.Rel(base, *object.Prefix)
		if err != nil {
			continue
		}
		res = append(res, fs.PhysicalObject{
			Name:         path.Base(*object.Prefix),
			RelativePath: filepath.ToSlash(rel),
			Size:         0,
			IsDir:        true,
			LastModify:   time.Now(),
		})
	}
	onProgress(len(commons))

	// 处理文件
	for _, object := range objects {
		rel, err := filepath.Rel(base, *object.Key)
		if err != nil {
			continue
		}
		res = append(res, fs.PhysicalObject{
			Name:         path.Base(*object.Key),
			Source:       *object.Key,
			RelativePath: filepath.ToSlash(rel),
			Size:         object.Size,
			IsDir:        false,
			LastModify:   *object.LastModified,
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
	forbidOverwrite := oss.Ptr(strconv.FormatBool(!overwrite))
	exipires := oss.Ptr(time.Now().Add(credentialTTL * time.Second).Format(time.RFC3339))

	// 小文件直接上传
	if file.Props.Size < MultiPartUploadThreshold {
		_, err := handler.client.PutObject(ctx, &oss.PutObjectRequest{
			Bucket:          &handler.policy.BucketName,
			Key:             &file.Props.SavePath,
			Body:            file,
			ForbidOverwrite: forbidOverwrite,
			ContentType:     oss.Ptr(mimeType),
		})
		return err
	}

	// 超过阈值时使用分片上传
	imur, err := handler.client.InitiateMultipartUpload(ctx, &oss.InitiateMultipartUploadRequest{
		Bucket:          &handler.policy.BucketName,
		Key:             &file.Props.SavePath,
		ContentType:     oss.Ptr(mimeType),
		ForbidOverwrite: forbidOverwrite,
		Expires:         exipires,
	})
	if err != nil {
		return fmt.Errorf("failed to initiate multipart upload: %w", err)
	}

	parts := make([]*oss.UploadPartResult, 0)

	chunks := chunk.NewChunkGroup(file, handler.chunkSize, &backoff.ConstantBackoff{
		Max:   handler.settings.ChunkRetryLimit(ctx),
		Sleep: chunkRetrySleep,
	}, handler.settings.UseChunkBuffer(ctx), handler.l, handler.settings.TempPath(ctx))

	uploadFunc := func(current *chunk.ChunkGroup, content io.Reader) error {
		part, err := handler.client.UploadPart(ctx, &oss.UploadPartRequest{
			Bucket:     &handler.policy.BucketName,
			Key:        &file.Props.SavePath,
			UploadId:   imur.UploadId,
			PartNumber: int32(current.Index() + 1),
			Body:       content,
		})
		if err == nil {
			parts = append(parts, part)
		}
		return err
	}

	for chunks.Next() {
		if err := chunks.Process(uploadFunc); err != nil {
			handler.cancelUpload(*imur)
			return fmt.Errorf("failed to upload chunk #%d: %w", chunks.Index(), err)
		}
	}

	_, err = handler.client.CompleteMultipartUpload(ctx, &oss.CompleteMultipartUploadRequest{
		Bucket:   &handler.policy.BucketName,
		Key:      imur.Key,
		UploadId: imur.UploadId,
		CompleteMultipartUpload: &oss.CompleteMultipartUpload{
			Parts: lo.Map(parts, func(part *oss.UploadPartResult, i int) oss.UploadPart {
				return oss.UploadPart{
					PartNumber: int32(i + 1),
					ETag:       part.ETag,
				}
			}),
		},
		ForbidOverwrite: oss.Ptr(strconv.FormatBool(!overwrite)),
	})
	if err != nil {
		handler.cancelUpload(*imur)
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
		delRes, err := handler.client.DeleteMultipleObjects(ctx, &oss.DeleteMultipleObjectsRequest{
			Bucket: &handler.policy.BucketName,
			Objects: lo.Map(group, func(v string, i int) oss.DeleteObject {
				return oss.DeleteObject{Key: &v}
			}),
		})
		if err != nil {
			failed = append(failed, group...)
			lastError = err
			continue
		}

		// 统计未删除的文件
		failed = append(
			failed,
			util.SliceDifference(files,
				lo.Map(delRes.DeletedObjects, func(v oss.DeletedInfo, i int) string {
					return *v.Key
				}),
			)...,
		)
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

	req := &oss.GetObjectRequest{
		Process: oss.Ptr(thumbParam),
	}
	thumbURL, err := handler.signSourceURL(
		ctx,
		e.Source(),
		expire,
		req,
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
	req := &oss.GetObjectRequest{}
	if args.IsDownload {
		encodedFilename := url.PathEscape(args.DisplayName)
		req.ResponseContentDisposition = oss.Ptr(fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`,
			encodedFilename, encodedFilename))
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
		req.Parameters = map[string]string{
			trafficLimitHeader: strconv.FormatInt(args.Speed, 10),
		}
	}

	return handler.signSourceURL(ctx, e.Source(), args.Expire, req, false)
}

func (handler *Driver) signSourceURL(ctx context.Context, path string, expire *time.Time, req *oss.GetObjectRequest, forceSign bool) (string, error) {
	ttl := time.Duration(24) * time.Hour * 7 // V4 Sign 最大过期时间为7天
	if expire != nil {
		ttl = time.Until(*expire)
	}

	if req == nil {
		req = &oss.GetObjectRequest{}
	}

	req.Bucket = &handler.policy.BucketName
	req.Key = &path

	// signedURL, err := handler.client.Presign(path, oss.HTTPGet, ttl, options...)
	result, err := handler.client.Presign(ctx, req, oss.PresignExpires(ttl))
	if err != nil {
		return "", err
	}

	// 将最终生成的签名URL域名换成用户自定义的加速域名（如果有）
	finalURL, err := url.Parse(result.URL)
	if err != nil {
		return "", err
	}

	// 公有空间替换掉Key及不支持的头
	if !handler.policy.IsPrivate && !forceSign {
		query := finalURL.Query()
		query.Del("x-oss-credential")
		query.Del("x-oss-date")
		query.Del("x-oss-expires")
		query.Del("x-oss-signature")
		query.Del("x-oss-signature-version")
		query.Del("response-content-disposition")
		// 阿里云 OSS 已支持在公有空间使用限速下载，使用公有空间时可以在 iOS / iPad 客户端等内置下载时实现限速
		// https://help.aliyun.com/zh/oss/single-connection-bandwidth-throttling-4#section-zka-umg-7dx
		// query.Del(trafficLimitHeader)
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
	imur, err := handler.client.InitiateMultipartUpload(ctx, &oss.InitiateMultipartUploadRequest{
		Bucket:          &handler.policy.BucketName,
		Key:             &file.Props.SavePath,
		ContentType:     oss.Ptr(mimeType),
		ForbidOverwrite: oss.Ptr(strconv.FormatBool(true)),
		Expires:         oss.Ptr(uploadSession.Props.ExpireAt.Format(time.RFC3339)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize multipart upload: %w", err)
	}
	uploadSession.UploadID = *imur.UploadId

	// 为每个分片签名上传 URL
	chunks := chunk.NewChunkGroup(file, handler.chunkSize, &backoff.ConstantBackoff{}, false, handler.l, "")
	urls := make([]string, chunks.Num())
	ttl := time.Until(uploadSession.Props.ExpireAt)
	for chunks.Next() {
		err := chunks.Process(func(c *chunk.ChunkGroup, chunk io.Reader) error {
			signedURL, err := handler.client.Presign(ctx, &oss.UploadPartRequest{
				Bucket:     &handler.policy.BucketName,
				Key:        &file.Props.SavePath,
				UploadId:   imur.UploadId,
				PartNumber: int32(c.Index() + 1),
				Body:       chunk,
				RequestCommon: oss.RequestCommon{
					Headers: map[string]string{
						"Content-Type": "application/octet-stream",
					},
				},
			}, oss.PresignExpires(ttl))
			if err != nil {
				return err
			}

			urls[c.Index()] = signedURL.URL
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// 签名完成分片上传的URL
	completeURL, err := handler.client.Presign(ctx, &oss.CompleteMultipartUploadRequest{
		Bucket:   &handler.policy.BucketName,
		Key:      &file.Props.SavePath,
		UploadId: imur.UploadId,
		RequestCommon: oss.RequestCommon{
			Parameters: map[string]string{
				"callback": callbackPolicyEncoded,
			},
			Headers: map[string]string{
				"Content-Type":        "application/octet-stream",
				completeAllHeader:     "yes",
				forbidOverwriteHeader: "true",
			},
		},
	}, oss.PresignExpires(ttl))
	if err != nil {
		return nil, err
	}

	return &fs.UploadCredential{
		UploadID:    *imur.UploadId,
		UploadURLs:  urls,
		CompleteURL: completeURL.URL,
		SessionID:   uploadSession.Props.UploadSessionID,
		ChunkSize:   handler.chunkSize,
		Callback:    callbackPolicyEncoded,
	}, nil
}

// 取消上传凭证
func (handler *Driver) CancelToken(ctx context.Context, uploadSession *fs.UploadSession) error {
	_, err := handler.client.AbortMultipartUpload(ctx, &oss.AbortMultipartUploadRequest{
		Bucket:   &handler.policy.BucketName,
		Key:      &uploadSession.Props.SavePath,
		UploadId: &uploadSession.UploadID,
	})
	return err
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
	if _, err := handler.client.AbortMultipartUpload(context.Background(), &oss.AbortMultipartUploadRequest{
		Bucket:   &handler.policy.BucketName,
		Key:      imur.Key,
		UploadId: imur.UploadId,
	}); err != nil {
		handler.l.Warning("failed to abort multipart upload: %s", err)
	}
}
