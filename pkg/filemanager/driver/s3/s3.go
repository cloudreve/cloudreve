package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
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
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/samber/lo"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Driver S3 compatible driver
type Driver struct {
	policy    *ent.StoragePolicy
	chunkSize int64

	settings setting.Provider
	l        logging.Logger
	config   conf.ConfigProvider
	mime     mime.MimeDetector

	sess *session.Session
	svc  *s3.S3
}

// UploadPolicy S3上传策略
type UploadPolicy struct {
	Expiration string        `json:"expiration"`
	Conditions []interface{} `json:"conditions"`
}

// MetaData 文件信息
type MetaData struct {
	Size int64
	Etag string
}

var (
	features = &boolset.BooleanSet{}
)

func init() {
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
		policy:    policy,
		settings:  settings,
		chunkSize: chunkSize,
		config:    config,
		l:         l,
		mime:      mime,
	}

	sess, err := session.NewSession(&aws.Config{
		Credentials:      credentials.NewStaticCredentials(policy.AccessKey, policy.SecretKey, ""),
		Endpoint:         &policy.Server,
		Region:           &policy.Settings.Region,
		S3ForcePathStyle: &policy.Settings.S3ForcePathStyle,
	})

	if err != nil {
		return nil, err
	}
	driver.sess = sess
	driver.svc = s3.New(sess)

	return driver, nil
}

// List 列出给定路径下的文件
func (handler *Driver) List(ctx context.Context, base string, onProgress driver.ListProgressFunc, recursive bool) ([]fs.PhysicalObject, error) {
	// 初始化列目录参数
	base = strings.TrimPrefix(base, "/")
	if base != "" {
		base += "/"
	}

	opt := &s3.ListObjectsInput{
		Bucket:  &handler.policy.BucketName,
		Prefix:  &base,
		MaxKeys: aws.Int64(1000),
	}

	// 是否为递归列出
	if !recursive {
		opt.Delimiter = aws.String("/")
	}

	var (
		objects []*s3.Object
		commons []*s3.CommonPrefix
	)

	for {
		res, err := handler.svc.ListObjectsWithContext(ctx, opt)
		if err != nil {
			return nil, err
		}
		objects = append(objects, res.Contents...)
		commons = append(commons, res.CommonPrefixes...)

		// 如果本次未列取完，则继续使用marker获取结果
		if *res.IsTruncated {
			opt.Marker = res.NextMarker
		} else {
			break
		}
	}

	// 处理列取结果
	res := make([]fs.PhysicalObject, 0, len(objects)+len(commons))

	// 处理目录
	for _, object := range commons {
		rel, err := filepath.Rel(*opt.Prefix, *object.Prefix)
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
		rel, err := filepath.Rel(*opt.Prefix, *object.Key)
		if err != nil {
			continue
		}
		res = append(res, fs.PhysicalObject{
			Name:         path.Base(*object.Key),
			Source:       *object.Key,
			RelativePath: filepath.ToSlash(rel),
			Size:         int64(*object.Size),
			IsDir:        false,
			LastModify:   time.Now(),
		})
	}
	onProgress(len(objects))

	return res, nil

}

// Open 打开文件
func (handler *Driver) Open(ctx context.Context, path string) (*os.File, error) {
	return nil, errors.New("not implemented")
}

// Put 将文件流保存到指定目录
func (handler *Driver) Put(ctx context.Context, file *fs.UploadRequest) error {
	defer file.Close()

	// 是否允许覆盖
	overwrite := file.Mode&fs.ModeOverwrite == fs.ModeOverwrite
	if !overwrite {
		// Check for duplicated file
		if _, err := handler.Meta(ctx, file.Props.SavePath); err == nil {
			return fs.ErrFileExisted
		}
	}

	uploader := s3manager.NewUploader(handler.sess, func(u *s3manager.Uploader) {
		u.PartSize = handler.chunkSize
	})

	mimeType := file.Props.MimeType
	if mimeType == "" {
		mimeType = handler.mime.TypeByName(file.Props.Uri.Name())
	}

	_, err := uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket:      &handler.policy.BucketName,
		Key:         &file.Props.SavePath,
		Body:        io.LimitReader(file, file.Props.Size),
		ContentType: aws.String(mimeType),
	})

	if err != nil {
		return err
	}

	return nil
}

// Delete 删除一个或多个文件，
// 返回未删除的文件，及遇到的最后一个错误
func (handler *Driver) Delete(ctx context.Context, files ...string) ([]string, error) {
	failed := make([]string, 0, len(files))
	batchSize := handler.policy.Settings.S3DeleteBatchSize
	if batchSize == 0 {
		// https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteObjects.html
		// The request can contain a list of up to 1000 keys that you want to delete.
		batchSize = 1000
	}

	var lastErr error

	groups := lo.Chunk(files, batchSize)
	for _, group := range groups {
		if len(group) == 1 {
			// Invoke single file delete API
			_, err := handler.svc.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
				Bucket: &handler.policy.BucketName,
				Key:    &group[0],
			})

			if err != nil {
				if aerr, ok := err.(awserr.Error); ok {
					// Ignore NoSuchKey error
					if aerr.Code() == s3.ErrCodeNoSuchKey {
						continue
					}
				}
				failed = append(failed, group[0])
				lastErr = err
			}
		} else {
			// Invoke batch delete API
			res, err := handler.svc.DeleteObjects(
				&s3.DeleteObjectsInput{
					Bucket: &handler.policy.BucketName,
					Delete: &s3.Delete{
						Objects: lo.Map(group, func(s string, i int) *s3.ObjectIdentifier {
							return &s3.ObjectIdentifier{Key: &s}
						}),
					},
				})

			if err != nil {
				failed = append(failed, group...)
				lastErr = err
				continue
			}

			for _, v := range res.Errors {
				handler.l.Debug("Failed to delete file: %s, Code:%s, Message:%s", v.Key, v.Code, v.Key)
				failed = append(failed, *v.Key)
			}
		}
	}

	return failed, lastErr

}

// Thumb 获取文件缩略图
func (handler *Driver) Thumb(ctx context.Context, expire *time.Time, ext string, e fs.Entity) (string, error) {
	return "", errors.New("not implemented")
}

// Source 获取外链URL
func (handler *Driver) Source(ctx context.Context, e fs.Entity, args *driver.GetSourceArgs) (string, error) {
	var contentDescription *string
	if args.IsDownload {
		encodedFilename := url.PathEscape(args.DisplayName)
		contentDescription = aws.String(fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`,
			encodedFilename, encodedFilename))
	}

	req, _ := handler.svc.GetObjectRequest(
		&s3.GetObjectInput{
			Bucket:                     &handler.policy.BucketName,
			Key:                        aws.String(e.Source()),
			ResponseContentDisposition: contentDescription,
		})

	ttl := time.Duration(604800) * time.Second // 7 days
	if args.Expire != nil {
		ttl = time.Until(*args.Expire)
	}
	signedURL, err := req.Presign(ttl)
	if err != nil {
		return "", err
	}

	// 将最终生成的签名URL域名换成用户自定义的加速域名（如果有）
	finalURL, err := url.Parse(signedURL)
	if err != nil {
		return "", err
	}

	// 公有空间替换掉Key及不支持的头
	if !handler.policy.IsPrivate {
		finalURL.RawQuery = ""
	}

	return finalURL.String(), nil
}

// Token 获取上传策略和认证Token
func (handler *Driver) Token(ctx context.Context, uploadSession *fs.UploadSession, file *fs.UploadRequest) (*fs.UploadCredential, error) {
	// Check for duplicated file
	if _, err := handler.Meta(ctx, file.Props.SavePath); err == nil {
		return nil, fs.ErrFileExisted
	}

	// 生成回调地址
	siteURL := handler.settings.SiteURL(setting.UseFirstSiteUrl(ctx))
	// 在从机端创建上传会话
	uploadSession.ChunkSize = handler.chunkSize
	uploadSession.Callback = routes.MasterSlaveCallbackUrl(siteURL, types.PolicyTypeS3, uploadSession.Props.UploadSessionID, uploadSession.CallbackSecret).String()

	mimeType := file.Props.MimeType
	if mimeType == "" {
		mimeType = handler.mime.TypeByName(file.Props.Uri.Name())
	}

	// 创建分片上传
	res, err := handler.svc.CreateMultipartUploadWithContext(ctx, &s3.CreateMultipartUploadInput{
		Bucket:      &handler.policy.BucketName,
		Key:         &uploadSession.Props.SavePath,
		Expires:     &uploadSession.Props.ExpireAt,
		ContentType: aws.String(mimeType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create multipart upload: %w", err)
	}

	uploadSession.UploadID = *res.UploadId

	// 为每个分片签名上传 URL
	chunks := chunk.NewChunkGroup(file, handler.chunkSize, &backoff.ConstantBackoff{}, false, handler.l, "")
	urls := make([]string, chunks.Num())
	for chunks.Next() {
		err := chunks.Process(func(c *chunk.ChunkGroup, chunk io.Reader) error {
			signedReq, _ := handler.svc.UploadPartRequest(&s3.UploadPartInput{
				Bucket:        &handler.policy.BucketName,
				Key:           &uploadSession.Props.SavePath,
				PartNumber:    aws.Int64(int64(c.Index() + 1)),
				ContentLength: aws.Int64(c.Length()),
				UploadId:      res.UploadId,
			})

			signedURL, err := signedReq.Presign(time.Until(uploadSession.Props.ExpireAt))
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

	// 签名完成分片上传的请求URL
	signedReq, _ := handler.svc.CompleteMultipartUploadRequest(&s3.CompleteMultipartUploadInput{
		Bucket:   &handler.policy.BucketName,
		Key:      &file.Props.SavePath,
		UploadId: res.UploadId,
	})

	signedURL, err := signedReq.Presign(time.Until(uploadSession.Props.ExpireAt))
	if err != nil {
		return nil, err
	}

	// 生成上传凭证
	return &fs.UploadCredential{
		UploadID:    *res.UploadId,
		UploadURLs:  urls,
		CompleteURL: signedURL,
		SessionID:   uploadSession.Props.UploadSessionID,
		ChunkSize:   handler.chunkSize,
	}, nil
}

// Meta 获取文件信息
func (handler *Driver) Meta(ctx context.Context, path string) (*MetaData, error) {
	res, err := handler.svc.HeadObjectWithContext(ctx,
		&s3.HeadObjectInput{
			Bucket: &handler.policy.BucketName,
			Key:    &path,
		})

	if err != nil {
		return nil, err
	}

	return &MetaData{
		Size: *res.ContentLength,
		Etag: *res.ETag,
	}, nil

}

// CORS 创建跨域策略
func (handler *Driver) CORS() error {
	rule := s3.CORSRule{
		AllowedMethods: aws.StringSlice([]string{
			"GET",
			"POST",
			"PUT",
			"DELETE",
			"HEAD",
		}),
		AllowedOrigins: aws.StringSlice([]string{"*"}),
		AllowedHeaders: aws.StringSlice([]string{"*"}),
		ExposeHeaders:  aws.StringSlice([]string{"ETag"}),
		MaxAgeSeconds:  aws.Int64(3600),
	}

	_, err := handler.svc.PutBucketCors(&s3.PutBucketCorsInput{
		Bucket: &handler.policy.BucketName,
		CORSConfiguration: &s3.CORSConfiguration{
			CORSRules: []*s3.CORSRule{&rule},
		},
	})

	return err
}

// 取消上传凭证
func (handler *Driver) CancelToken(ctx context.Context, uploadSession *fs.UploadSession) error {
	_, err := handler.svc.AbortMultipartUploadWithContext(ctx, &s3.AbortMultipartUploadInput{
		UploadId: &uploadSession.UploadID,
		Bucket:   &handler.policy.BucketName,
		Key:      &uploadSession.Props.SavePath,
	})
	return err
}

func (handler *Driver) cancelUpload(key, id *string) {
	if _, err := handler.svc.AbortMultipartUpload(&s3.AbortMultipartUploadInput{
		Bucket:   &handler.policy.BucketName,
		UploadId: id,
		Key:      key,
	}); err != nil {
		handler.l.Warning("failed to abort multipart upload: %s", err)
	}
}

func (handler *Driver) Capabilities() *driver.Capabilities {
	return &driver.Capabilities{
		StaticFeatures:  features,
		MediaMetaProxy:  handler.policy.Settings.MediaMetaGeneratorProxy,
		ThumbProxy:      handler.policy.Settings.ThumbGeneratorProxy,
		MaxSourceExpire: time.Duration(604800) * time.Second,
	}
}

func (handler *Driver) MediaMeta(ctx context.Context, path, ext string) ([]driver.MediaMeta, error) {
	return nil, errors.New("not implemented")
}

func (handler *Driver) LocalPath(ctx context.Context, path string) string {
	return ""
}

func (handler *Driver) CompleteUpload(ctx context.Context, session *fs.UploadSession) error {
	if session.SentinelTaskID == 0 {
		return nil
	}

	// Make sure uploaded file size is correct
	res, err := handler.Meta(ctx, session.Props.SavePath)
	if err != nil {
		return fmt.Errorf("failed to get uploaded file size: %w", err)
	}

	if res.Size != session.Props.Size {
		return serializer.NewError(
			serializer.CodeMetaMismatch,
			fmt.Sprintf("File size not match, expected: %d, actual: %d", session.Props.Size, res.Size),
			nil,
		)
	}
	return nil
}

type Reader struct {
	r io.Reader
}

func (r Reader) Read(p []byte) (int, error) {
	return r.r.Read(p)
}
