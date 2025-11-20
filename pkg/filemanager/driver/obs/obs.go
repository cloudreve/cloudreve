package obs

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
	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	"github.com/samber/lo"
)

const (
	chunkRetrySleep    = time.Duration(5) * time.Second
	maxDeleteBatch     = 1000
	imageProcessHeader = "x-image-process"
	trafficLimitHeader = "x-obs-traffic-limit"
	partNumberParam    = "partNumber"
	callbackParam      = "x-obs-callback"
	uploadIdParam      = "uploadId"
	mediaInfoTTL       = time.Duration(10) * time.Minute
	imageInfoProcessor = "image/info"

	// MultiPartUploadThreshold 服务端使用分片上传的阈值
	MultiPartUploadThreshold int64 = 5 << 30 // 5GB
)

var (
	features = &boolset.BooleanSet{}
)

type (
	CallbackPolicy struct {
		CallbackURL      string `json:"callbackUrl"`
		CallbackBody     string `json:"callbackBody"`
		CallbackBodyType string `json:"callbackBodyType"`
	}
	JsonError struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	}
)

// Driver Huawei Cloud OBS driver
type Driver struct {
	policy    *ent.StoragePolicy
	chunkSize int64

	settings   setting.Provider
	l          logging.Logger
	config     conf.ConfigProvider
	mime       mime.MimeDetector
	httpClient request.Client
	obs        *obs.ObsClient
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

	useCname := false
	if policy.Settings != nil && policy.Settings.UseCname {
		useCname = true
	}

	obsClient, err := obs.New(policy.AccessKey, policy.SecretKey, policy.Server, obs.WithSignature(obs.SignatureObs), obs.WithCustomDomainName(useCname))
	if err != nil {
		return nil, err
	}

	driver.obs = obsClient
	return driver, nil
}

func (d *Driver) List(ctx context.Context, base string, onProgress driver.ListProgressFunc, recursive bool) ([]fs.PhysicalObject, error) {
	opt := &obs.ListObjectsInput{
		ListObjsInput: obs.ListObjsInput{
			Prefix:       strings.TrimPrefix(base, "/"),
			EncodingType: "",
			MaxKeys:      1000,
		},
		Bucket: d.policy.BucketName,
	}

	if !recursive {
		opt.Delimiter = "/"
	}

	if opt.Prefix != "" {
		opt.Prefix += "/"
	}

	var (
		marker  string
		objects []obs.Content
		commons []string
	)

	for {
		res, err := d.obs.ListObjects(opt, obs.WithRequestContext(ctx))
		if err != nil {
			d.l.Warning("Failed to list objects: %s", err)
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
			d.l.Warning("Failed to get relative path: %s", err)
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
			d.l.Warning("Failed to get relative path: %s", err)
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

func (d *Driver) Put(ctx context.Context, file *fs.UploadRequest) error {
	defer file.Close()

	// 是否允许覆盖
	overwrite := file.Mode&fs.ModeOverwrite == fs.ModeOverwrite
	if !overwrite {
		// Check for duplicated file
		if _, err := d.obs.HeadObject(&obs.HeadObjectInput{
			Bucket: d.policy.BucketName,
			Key:    file.Props.SavePath,
		}, obs.WithRequestContext(ctx)); err == nil {
			return fs.ErrFileExisted
		}
	}

	mimeType := file.Props.MimeType
	if mimeType == "" {
		d.mime.TypeByName(file.Props.Uri.Name())
	}

	// 小文件直接上传
	if file.Props.Size < MultiPartUploadThreshold {
		_, err := d.obs.PutObject(&obs.PutObjectInput{
			PutObjectBasicInput: obs.PutObjectBasicInput{
				ObjectOperationInput: obs.ObjectOperationInput{
					Key:    file.Props.SavePath,
					Bucket: d.policy.BucketName,
				},
				HttpHeader: obs.HttpHeader{
					ContentType: mimeType,
				},
				ContentLength: file.Props.Size,
			},
			Body: file,
		}, obs.WithRequestContext(ctx))
		return err
	}

	// 超过阈值时使用分片上传
	imur, err := d.obs.InitiateMultipartUpload(&obs.InitiateMultipartUploadInput{
		ObjectOperationInput: obs.ObjectOperationInput{
			Bucket: d.policy.BucketName,
			Key:    file.Props.SavePath,
		},
		HttpHeader: obs.HttpHeader{
			ContentType: d.mime.TypeByName(file.Props.Uri.Name()),
		},
	}, obs.WithRequestContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to initiate multipart upload: %w", err)
	}

	chunks := chunk.NewChunkGroup(file, d.chunkSize, &backoff.ConstantBackoff{
		Max:   d.settings.ChunkRetryLimit(ctx),
		Sleep: chunkRetrySleep,
	}, d.settings.UseChunkBuffer(ctx), d.l, d.settings.TempPath(ctx))

	parts := make([]*obs.UploadPartOutput, 0, chunks.Num())

	uploadFunc := func(current *chunk.ChunkGroup, content io.Reader) error {
		part, err := d.obs.UploadPart(&obs.UploadPartInput{
			Bucket:     d.policy.BucketName,
			Key:        file.Props.SavePath,
			PartNumber: current.Index() + 1,
			UploadId:   imur.UploadId,
			Body:       content,
			SourceFile: "",
			PartSize:   current.Length(),
		}, obs.WithRequestContext(ctx))
		if err == nil {
			parts = append(parts, part)
		}
		return err
	}

	for chunks.Next() {
		if err := chunks.Process(uploadFunc); err != nil {
			d.cancelUpload(file.Props.SavePath, imur)
			return fmt.Errorf("failed to upload chunk #%d: %w", chunks.Index(), err)
		}
	}

	_, err = d.obs.CompleteMultipartUpload(&obs.CompleteMultipartUploadInput{
		Bucket:   d.policy.BucketName,
		Key:      file.Props.SavePath,
		UploadId: imur.UploadId,
		Parts: lo.Map(parts, func(part *obs.UploadPartOutput, i int) obs.Part {
			return obs.Part{
				PartNumber: i + 1,
				ETag:       part.ETag,
			}
		}),
	}, obs.WithRequestContext(ctx))
	if err != nil {
		d.cancelUpload(file.Props.SavePath, imur)
	}

	return err
}

func (d *Driver) Delete(ctx context.Context, files ...string) ([]string, error) {
	groups := lo.Chunk(files, maxDeleteBatch)
	failed := make([]string, 0)
	var lastError error
	for index, group := range groups {
		d.l.Debug("Process delete group #%d: %v", index, group)
		// 删除文件
		delRes, err := d.obs.DeleteObjects(&obs.DeleteObjectsInput{
			Bucket: d.policy.BucketName,
			Quiet:  true,
			Objects: lo.Map(group, func(item string, index int) obs.ObjectToDelete {
				return obs.ObjectToDelete{
					Key: item,
				}
			}),
		}, obs.WithRequestContext(ctx))
		if err != nil {
			failed = append(failed, group...)
			lastError = err
			continue
		}

		for _, v := range delRes.Errors {
			d.l.Debug("Failed to delete file: %s, Code:%s, Message:%s", v.Key, v.Code, v.Key)
			failed = append(failed, v.Key)
		}
	}

	if len(failed) > 0 && lastError == nil {
		lastError = fmt.Errorf("failed to delete files: %v", failed)
	}

	return failed, lastError
}

func (d *Driver) Open(ctx context.Context, path string) (*os.File, error) {
	return nil, errors.New("not implemented")
}

func (d *Driver) LocalPath(ctx context.Context, path string) string {
	return ""
}

func (d *Driver) Thumb(ctx context.Context, expire *time.Time, ext string, e fs.Entity) (string, error) {
	w, h := d.settings.ThumbSize(ctx)
	thumbParam := fmt.Sprintf("image/resize,m_lfit,w_%d,h_%d", w, h)

	enco := d.settings.ThumbEncode(ctx)
	switch enco.Format {
	case "jpg", "webp":
		thumbParam += fmt.Sprintf("/format,%s/quality,q_%d", enco.Format, enco.Quality)
	case "png":
		thumbParam += fmt.Sprintf("/format,%s", enco.Format)
	}

	thumbURL, err := d.signSourceURL(&obs.CreateSignedUrlInput{
		Method:  obs.HttpMethodGet,
		Bucket:  d.policy.BucketName,
		Key:     e.Source(),
		Expires: int(time.Until(*expire).Seconds()),
		QueryParams: map[string]string{
			imageProcessHeader: thumbParam,
		},
	})

	if err != nil {
		return "", err
	}

	return thumbURL, nil
}

func (d *Driver) Source(ctx context.Context, e fs.Entity, args *driver.GetSourceArgs) (string, error) {
	params := make(map[string]string)
	if args.IsDownload {
		encodedFilename := url.PathEscape(args.DisplayName)
		params["response-content-disposition"] = fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s",
			args.DisplayName, encodedFilename)
	}

	expires := 86400 * 265 * 20
	if args.Expire != nil {
		expires = int(time.Until(*args.Expire).Seconds())
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
	}

	if args.Speed > 0 {
		params[trafficLimitHeader] = strconv.FormatInt(args.Speed, 10)
	}

	return d.signSourceURL(&obs.CreateSignedUrlInput{
		Method:      obs.HttpMethodGet,
		Bucket:      d.policy.BucketName,
		Key:         e.Source(),
		Expires:     expires,
		QueryParams: params,
	})
}

func (d *Driver) Token(ctx context.Context, uploadSession *fs.UploadSession, file *fs.UploadRequest) (*fs.UploadCredential, error) {
	// Check for duplicated file
	if _, err := d.obs.HeadObject(&obs.HeadObjectInput{
		Bucket: d.policy.BucketName,
		Key:    file.Props.SavePath,
	}, obs.WithRequestContext(ctx)); err == nil {
		return nil, fs.ErrFileExisted
	}

	// 生成回调地址
	siteURL := d.settings.SiteURL(setting.UseFirstSiteUrl(ctx))
	// 在从机端创建上传会话
	uploadSession.ChunkSize = d.chunkSize
	uploadSession.Callback = routes.MasterSlaveCallbackUrl(siteURL, types.PolicyTypeObs, uploadSession.Props.UploadSessionID, uploadSession.CallbackSecret).String()
	// 回调策略
	callbackPolicy := CallbackPolicy{
		CallbackURL:      uploadSession.Callback,
		CallbackBody:     `{"name":${key},"source_name":${fname},"size":${size}}`,
		CallbackBodyType: "application/json",
	}

	callbackPolicyJSON, err := json.Marshal(callbackPolicy)
	if err != nil {
		return nil, fmt.Errorf("failed to encode callback policy: %w", err)
	}
	callbackPolicyEncoded := base64.StdEncoding.EncodeToString(callbackPolicyJSON)

	mimeType := file.Props.MimeType
	if mimeType == "" {
		d.mime.TypeByName(file.Props.Uri.Name())
	}

	imur, err := d.obs.InitiateMultipartUpload(&obs.InitiateMultipartUploadInput{
		ObjectOperationInput: obs.ObjectOperationInput{
			Bucket: d.policy.BucketName,
			Key:    file.Props.SavePath,
		},
		HttpHeader: obs.HttpHeader{
			ContentType: mimeType,
		},
	}, obs.WithRequestContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize multipart upload: %w", err)
	}
	uploadSession.UploadID = imur.UploadId

	// 为每个分片签名上传 URL
	chunks := chunk.NewChunkGroup(file, d.chunkSize, &backoff.ConstantBackoff{}, false, d.l, "")
	urls := make([]string, chunks.Num())
	ttl := int64(time.Until(uploadSession.Props.ExpireAt).Seconds())
	for chunks.Next() {
		err := chunks.Process(func(c *chunk.ChunkGroup, chunk io.Reader) error {
			signedURL, err := d.obs.CreateSignedUrl(&obs.CreateSignedUrlInput{
				Method: obs.HttpMethodPut,
				Bucket: d.policy.BucketName,
				Key:    file.Props.SavePath,
				QueryParams: map[string]string{
					partNumberParam: strconv.Itoa(c.Index() + 1),
					uploadIdParam:   uploadSession.UploadID,
				},
				Expires: int(ttl),
				Headers: map[string]string{
					"Content-Length": strconv.FormatInt(c.Length(), 10),
					"Content-Type":   "application/octet-stream",
				}, //TODO: Validate +1
			})
			if err != nil {
				return err
			}

			urls[c.Index()] = signedURL.SignedUrl
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// 签名完成分片上传的URL
	completeURL, err := d.obs.CreateSignedUrl(&obs.CreateSignedUrlInput{
		Method: obs.HttpMethodPost,
		Bucket: d.policy.BucketName,
		Key:    file.Props.SavePath,
		QueryParams: map[string]string{
			uploadIdParam: uploadSession.UploadID,
			callbackParam: callbackPolicyEncoded,
		},
		Headers: map[string]string{
			"Content-Type": "application/octet-stream",
		},
		Expires: int(ttl),
	})
	if err != nil {
		return nil, err
	}

	return &fs.UploadCredential{
		UploadID:    imur.UploadId,
		UploadURLs:  urls,
		CompleteURL: completeURL.SignedUrl,
		SessionID:   uploadSession.Props.UploadSessionID,
		ChunkSize:   d.chunkSize,
	}, nil
}

func (d *Driver) CancelToken(ctx context.Context, uploadSession *fs.UploadSession) error {
	_, err := d.obs.AbortMultipartUpload(&obs.AbortMultipartUploadInput{
		Bucket:   d.policy.BucketName,
		Key:      uploadSession.Props.SavePath,
		UploadId: uploadSession.UploadID,
	}, obs.WithRequestContext(ctx))
	return err
}

func (d *Driver) CompleteUpload(ctx context.Context, session *fs.UploadSession) error {
	return nil
}

//func (d *Driver) List(ctx context.Context, path string, recursive bool) ([]response.Object, error) {
//	return nil, errors.New("not implemented")
//}

func (d *Driver) Capabilities() *driver.Capabilities {
	mediaMetaExts := d.policy.Settings.MediaMetaExts
	if !d.policy.Settings.NativeMediaProcessing {
		mediaMetaExts = nil
	}
	return &driver.Capabilities{
		StaticFeatures:         features,
		MediaMetaSupportedExts: mediaMetaExts,
		MediaMetaProxy:         d.policy.Settings.MediaMetaGeneratorProxy,
		ThumbSupportedExts:     d.policy.Settings.ThumbExts,
		ThumbProxy:             d.policy.Settings.ThumbGeneratorProxy,
		ThumbSupportAllExts:    d.policy.Settings.ThumbSupportAllExts,
		ThumbMaxSize:           d.policy.Settings.ThumbMaxSize,
	}
}

// CORS 创建跨域策略
func (d *Driver) CORS() error {
	_, err := d.obs.SetBucketCors(&obs.SetBucketCorsInput{
		Bucket: d.policy.BucketName,
		BucketCors: obs.BucketCors{
			CorsRules: []obs.CorsRule{
				{
					AllowedOrigin: []string{"*"},
					AllowedMethod: []string{
						"GET",
						"POST",
						"PUT",
						"DELETE",
						"HEAD",
					},
					ExposeHeader:  []string{"Etag"},
					AllowedHeader: []string{"*"},
					MaxAgeSeconds: 3600,
				},
			},
		},
	})
	return err
}

func (d *Driver) cancelUpload(path string, imur *obs.InitiateMultipartUploadOutput) {
	if _, err := d.obs.AbortMultipartUpload(&obs.AbortMultipartUploadInput{
		Bucket:   d.policy.BucketName,
		Key:      path,
		UploadId: imur.UploadId,
	}); err != nil {
		d.l.Warning("failed to abort multipart upload: %s", err)
	}
}

func (handler *Driver) signSourceURL(input *obs.CreateSignedUrlInput) (string, error) {
	signedURL, err := handler.obs.CreateSignedUrl(input)
	if err != nil {
		return "", err
	}

	finalURL, err := url.Parse(signedURL.SignedUrl)
	if err != nil {
		return "", err
	}

	// 公有空间替换掉Key及不支持的头
	if !handler.policy.IsPrivate {
		query := finalURL.Query()
		query.Del("AccessKeyId")
		query.Del("Signature")
		finalURL.RawQuery = query.Encode()
	}
	return finalURL.String(), nil
}

func handleJsonError(resp string, originErr error) error {
	if resp == "" {
		return originErr
	}

	var err JsonError
	if err := json.Unmarshal([]byte(resp), &err); err != nil {
		return fmt.Errorf("failed to unmarshal cos error: %w", err)
	}

	return fmt.Errorf("obs error: %s", err.Message)
}
