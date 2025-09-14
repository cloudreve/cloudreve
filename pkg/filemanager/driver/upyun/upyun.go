package upyun

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/mime"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/gin-gonic/gin"
	"github.com/upyun/go-sdk/upyun"
)

type (
	// UploadPolicy 又拍云上传策略
	UploadPolicy struct {
		Bucket             string `json:"bucket"`
		SaveKey            string `json:"save-key"`
		Expiration         int64  `json:"expiration"`
		CallbackURL        string `json:"notify-url"`
		ContentLength      uint64 `json:"content-length"`
		ContentLengthRange string `json:"content-length-range,omitempty"`
		AllowFileType      string `json:"allow-file-type,omitempty"`
	}
	// Driver 又拍云策略适配器
	Driver struct {
		policy *ent.StoragePolicy

		up         *upyun.UpYun
		settings   setting.Provider
		l          logging.Logger
		config     conf.ConfigProvider
		mime       mime.MimeDetector
		httpClient request.Client
	}
)

var (
	features = &boolset.BooleanSet{}
)

func New(ctx context.Context, policy *ent.StoragePolicy, settings setting.Provider,
	config conf.ConfigProvider, l logging.Logger, mime mime.MimeDetector) (*Driver, error) {
	driver := &Driver{
		policy:     policy,
		settings:   settings,
		config:     config,
		l:          l,
		mime:       mime,
		httpClient: request.NewClient(config, request.WithLogger(l)),
		up: upyun.NewUpYun(&upyun.UpYunConfig{
			Bucket:   policy.BucketName,
			Operator: policy.AccessKey,
			Password: policy.SecretKey,
		}),
	}

	return driver, nil
}

func (handler *Driver) List(ctx context.Context, base string, onProgress driver.ListProgressFunc, recursive bool) ([]fs.PhysicalObject, error) {
	base = strings.TrimPrefix(base, "/")

	// 用于接受SDK返回对象的chan
	objChan := make(chan *upyun.FileInfo)
	objects := []*upyun.FileInfo{}

	// 列取配置
	listConf := &upyun.GetObjectsConfig{
		Path:         "/" + base,
		ObjectsChan:  objChan,
		MaxListTries: 1,
	}
	// 递归列取时不限制递归次数
	if recursive {
		listConf.MaxListLevel = -1
	}

	// 启动一个goroutine收集列取的对象信
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func(input chan *upyun.FileInfo, output *[]*upyun.FileInfo, wg *sync.WaitGroup) {
		defer wg.Done()
		for {
			file, ok := <-input
			if !ok {
				return
			}
			*output = append(*output, file)
			onProgress(1)
		}
	}(objChan, &objects, wg)

	up := upyun.NewUpYun(&upyun.UpYunConfig{
		Bucket:   handler.policy.BucketName,
		Operator: handler.policy.AccessKey,
		Password: handler.policy.SecretKey,
	})

	err := up.List(listConf)
	if err != nil {
		return nil, err
	}

	wg.Wait()

	// 汇总处理列取结果
	res := make([]fs.PhysicalObject, 0, len(objects))
	for _, object := range objects {
		res = append(res, fs.PhysicalObject{
			Name:         path.Base(object.Name),
			RelativePath: object.Name,
			Source:       path.Join(base, object.Name),
			Size:         int64(object.Size),
			IsDir:        object.IsDir,
			LastModify:   object.Time,
		})
	}

	return res, nil
}

func (handler *Driver) Open(ctx context.Context, path string) (*os.File, error) {
	return nil, errors.New("not implemented")
}

// Put 将文件流保存到指定目录
func (handler *Driver) Put(ctx context.Context, file *fs.UploadRequest) error {
	defer file.Close()

	// 是否允许覆盖
	overwrite := file.Mode&fs.ModeOverwrite == fs.ModeOverwrite
	if !overwrite {
		if _, err := handler.up.GetInfo(file.Props.SavePath); err == nil {
			return fs.ErrFileExisted
		}
	}

	mimeType := file.Props.MimeType
	if mimeType == "" {
		mimeType = handler.mime.TypeByName(file.Props.Uri.Name())
	}

	err := handler.up.Put(&upyun.PutObjectConfig{
		Path:   file.Props.SavePath,
		Reader: file,
		Headers: map[string]string{
			"Content-Type": mimeType,
		},
	})

	return err
}

// Delete 删除一个或多个文件，
// 返回未删除的文件，及遇到的最后一个错误
func (handler *Driver) Delete(ctx context.Context, files ...string) ([]string, error) {
	failed := make([]string, 0)
	var lastErr error

	for _, file := range files {
		if err := handler.up.Delete(&upyun.DeleteObjectConfig{
			Path:  file,
			Async: true,
		}); err != nil {
			filteredErr := strings.ReplaceAll(err.Error(), file, "")
			if strings.Contains(filteredErr, "Not found") ||
				strings.Contains(filteredErr, "NoSuchKey") {
				continue
			}

			failed = append(failed, file)
			lastErr = err
		}
	}

	return failed, lastErr
}

// Thumb 获取文件缩略图
func (handler *Driver) Thumb(ctx context.Context, expire *time.Time, ext string, e fs.Entity) (string, error) {
	w, h := handler.settings.ThumbSize(ctx)
	thumbParam := fmt.Sprintf("!/fwfh/%dx%d", w, h)

	enco := handler.settings.ThumbEncode(ctx)
	switch enco.Format {
	case "jpg", "webp":
		thumbParam += fmt.Sprintf("/format/%s/quality/%d", enco.Format, enco.Quality)
	case "png":
		thumbParam += fmt.Sprintf("/format/%s", enco.Format)
	}

	thumbURL, err := handler.signURL(ctx, e.Source()+thumbParam, nil, expire)
	if err != nil {
		return "", err
	}

	return thumbURL, nil
}

// Source 获取外链URL
func (handler *Driver) Source(ctx context.Context, e fs.Entity, args *driver.GetSourceArgs) (string, error) {
	query := url.Values{}

	// 如果是下载文件URL
	if args.IsDownload {
		query.Add("_upd", args.DisplayName)
	}

	return handler.signURL(ctx, e.Source(), &query, args.Expire)
}

func (handler *Driver) signURL(ctx context.Context, path string, query *url.Values, expire *time.Time) (string, error) {
	sourceURL, err := url.Parse(handler.policy.Settings.ProxyServer)
	if err != nil {
		return "", err
	}

	fileKey, err := url.Parse(url.PathEscape(path))
	if err != nil {
		return "", err
	}

	sourceURL = sourceURL.ResolveReference(fileKey)
	if query != nil {
		sourceURL.RawQuery = query.Encode()

	}

	if !handler.policy.IsPrivate {
		// 未开启Token防盗链时，直接返回
		return sourceURL.String(), nil
	}

	etime := time.Now().Add(time.Duration(24) * time.Hour * 365 * 20).Unix()
	if expire != nil {
		etime = expire.Unix()
	}
	signStr := fmt.Sprintf(
		"%s&%d&%s",
		handler.policy.Settings.Token,
		etime,
		sourceURL.Path,
	)
	signMd5 := fmt.Sprintf("%x", md5.Sum([]byte(signStr)))
	finalSign := signMd5[12:20] + strconv.FormatInt(etime, 10)

	// 将签名添加到URL中
	q := sourceURL.Query()
	q.Add("_upt", finalSign)
	sourceURL.RawQuery = q.Encode()

	return sourceURL.String(), nil
}

// Token 获取上传策略和认证Token
func (handler *Driver) Token(ctx context.Context, uploadSession *fs.UploadSession, file *fs.UploadRequest) (*fs.UploadCredential, error) {
	if _, err := handler.up.GetInfo(file.Props.SavePath); err == nil {
		return nil, fs.ErrFileExisted
	}

	// 生成回调地址
	siteURL := handler.settings.SiteURL(setting.UseFirstSiteUrl(ctx))
	apiUrl := routes.MasterSlaveCallbackUrl(siteURL, types.PolicyTypeUpyun, uploadSession.Props.UploadSessionID, uploadSession.CallbackSecret).String()

	// 上传策略
	putPolicy := UploadPolicy{
		Bucket:             handler.policy.BucketName,
		SaveKey:            file.Props.SavePath,
		Expiration:         uploadSession.Props.ExpireAt.Unix(),
		CallbackURL:        apiUrl,
		ContentLength:      uint64(file.Props.Size),
		ContentLengthRange: fmt.Sprintf("0,%d", file.Props.Size),
	}

	// 生成上传凭证
	policyJSON, err := json.Marshal(putPolicy)
	if err != nil {
		return nil, err
	}
	policyEncoded := base64.StdEncoding.EncodeToString(policyJSON)

	// 生成签名
	elements := []string{"POST", "/" + handler.policy.BucketName, policyEncoded}
	signStr := sign(handler.policy.AccessKey, handler.policy.SecretKey, elements)

	mimeType := file.Props.MimeType
	if mimeType == "" {
		mimeType = handler.mime.TypeByName(file.Props.Uri.Name())
	}

	return &fs.UploadCredential{
		UploadPolicy: policyEncoded,
		UploadURLs:   []string{"https://v0.api.upyun.com/" + handler.policy.BucketName},
		Credential:   signStr,
		MimeType:     mimeType,
	}, nil
}

// 取消上传凭证
func (handler *Driver) CancelToken(ctx context.Context, uploadSession *fs.UploadSession) error {
	return nil
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
		ThumbMaxSize:           handler.policy.Settings.ThumbMaxSize,
		ThumbSupportAllExts:    handler.policy.Settings.ThumbSupportAllExts,
	}
}

func (handler *Driver) MediaMeta(ctx context.Context, path, ext string) ([]driver.MediaMeta, error) {
	return handler.extractImageMeta(ctx, path)
}

func (handler *Driver) LocalPath(ctx context.Context, path string) string {
	return ""
}

func ValidateCallback(c *gin.Context, session *fs.UploadSession) error {
	body, err := io.ReadAll(c.Request.Body)
	c.Request.Body.Close()
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}

	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	contentMD5 := c.Request.Header.Get("Content-Md5")
	date := c.Request.Header.Get("Date")
	actualSignature := c.Request.Header.Get("Authorization")
	actualContentMD5 := fmt.Sprintf("%x", md5.Sum(body))
	if actualContentMD5 != contentMD5 {
		return errors.New("MD5 mismatch")
	}

	// Compare signature
	signature := sign(session.Policy.AccessKey, session.Policy.SecretKey, []string{
		"POST",
		c.Request.URL.Path,
		date,
		contentMD5,
	})
	if signature != actualSignature {
		return errors.New("Signature not match")
	}

	return nil
}

// Sign 计算又拍云的签名头
func sign(ak, sk string, elements []string) string {
	password := fmt.Sprintf("%x", md5.Sum([]byte(sk)))
	mac := hmac.New(sha1.New, []byte(password))
	value := strings.Join(elements, "&")
	mac.Write([]byte(value))
	signStr := base64.StdEncoding.EncodeToString((mac.Sum(nil)))
	return fmt.Sprintf("UPYUN %s:%s", ak, signStr)
}
