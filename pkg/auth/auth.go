package auth

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
)

var (
	ErrAuthFailed        = serializer.NewError(serializer.CodeInvalidSign, "invalid sign", nil)
	ErrAuthHeaderMissing = serializer.NewError(serializer.CodeNoPermissionErr, "authorization header is missing", nil)
	ErrExpiresMissing    = serializer.NewError(serializer.CodeNoPermissionErr, "expire timestamp is missing", nil)
	ErrExpired           = serializer.NewError(serializer.CodeSignExpired, "signature expired", nil)
)

const (
	TokenHeaderPrefixCr = "Bearer Cr "
)

// General 通用的认证接口
// Deprecated
var General Auth

type (
	// Auth 鉴权认证
	Auth interface {
		// 对给定Body进行签名,expires为0表示永不过期
		Sign(body string, expires int64) string
		// 对给定Body和Sign进行检查
		Check(body string, sign string) error
	}
)

// SignRequest 对PUT\POST等复杂HTTP请求签名，只会对URI部分、
// 请求正文、`X-Cr-`开头的header进行签名
func SignRequest(ctx context.Context, instance Auth, r *http.Request, expires *time.Time) *http.Request {
	// 处理有效期
	expireTime := int64(0)
	if expires != nil {
		expireTime = expires.Unix()
	}

	// 生成签名
	sign := instance.Sign(getSignContent(ctx, r), expireTime)

	// 将签名加到请求Header中
	r.Header["Authorization"] = []string{TokenHeaderPrefixCr + sign}
	return r
}

// SignRequestDeprecated 对PUT\POST等复杂HTTP请求签名，只会对URI部分、
// 请求正文、`X-Cr-`开头的header进行签名
func SignRequestDeprecated(instance Auth, r *http.Request, expires int64) *http.Request {
	// 处理有效期
	if expires > 0 {
		expires += time.Now().Unix()
	}

	// 生成签名
	sign := instance.Sign(getSignContent(context.Background(), r), expires)

	// 将签名加到请求Header中
	r.Header["Authorization"] = []string{TokenHeaderPrefixCr + sign}
	return r
}

// CheckRequest 对复杂请求进行签名验证
func CheckRequest(ctx context.Context, instance Auth, r *http.Request) error {
	var (
		sign []string
		ok   bool
	)
	if sign, ok = r.Header["Authorization"]; !ok || len(sign) == 0 {
		return ErrAuthHeaderMissing
	}
	sign[0] = strings.TrimPrefix(sign[0], TokenHeaderPrefixCr)

	return instance.Check(getSignContent(ctx, r), sign[0])
}

func isUploadDataRequest(r *http.Request) bool {
	return strings.Contains(r.URL.Path, constants.APIPrefix+"/slave/upload/") && r.Method != http.MethodPut

}

// getSignContent 签名请求 path、正文、以`X-`开头的 Header. 如果请求 path 为从机上传 API，
// 则不对正文签名。返回待签名/验证的字符串
func getSignContent(ctx context.Context, r *http.Request) (rawSignString string) {
	// 读取所有body正文
	var body = []byte{}
	if !isUploadDataRequest(r) {
		if r.Body != nil {
			body, _ = io.ReadAll(r.Body)
			_ = r.Body.Close()
			r.Body = io.NopCloser(bytes.NewReader(body))
		}
	}

	// 决定要签名的header
	var signedHeader []string
	for k, _ := range r.Header {
		if strings.HasPrefix(k, constants.CrHeaderPrefix) && k != constants.CrHeaderPrefix+"Filename" {
			signedHeader = append(signedHeader, fmt.Sprintf("%s=%s", k, r.Header.Get(k)))
		}
	}
	sort.Strings(signedHeader)

	// 读取所有待签名Header
	rawSignString = serializer.NewRequestSignString(getUrlSignContent(ctx, r.URL), strings.Join(signedHeader, "&"), string(body))

	return rawSignString
}

// SignURI 对URI进行签名
func SignURI(ctx context.Context, instance Auth, uri string, expires *time.Time) (*url.URL, error) {
	// 处理有效期
	expireTime := int64(0)
	if expires != nil {
		expireTime = expires.Unix()
	}

	base, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	// 生成签名
	sign := instance.Sign(getUrlSignContent(ctx, base), expireTime)

	// 将签名加到URI中
	queries := base.Query()
	queries.Set("sign", sign)
	base.RawQuery = queries.Encode()

	return base, nil
}

// SignURIDeprecated 对URI进行签名,签名只针对Path部分，query部分不做验证
// Deprecated
func SignURIDeprecated(instance Auth, uri string, expires int64) (*url.URL, error) {
	// 处理有效期
	if expires != 0 {
		expires += time.Now().Unix()
	}

	base, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	// 生成签名
	sign := instance.Sign(base.Path, expires)

	// 将签名加到URI中
	queries := base.Query()
	queries.Set("sign", sign)
	base.RawQuery = queries.Encode()

	return base, nil
}

// CheckURI 对URI进行鉴权
func CheckURI(ctx context.Context, instance Auth, url *url.URL) error {
	//获取待验证的签名正文
	queries := url.Query()
	sign := queries.Get("sign")
	queries.Del("sign")
	url.RawQuery = queries.Encode()

	return instance.Check(getUrlSignContent(ctx, url), sign)
}

func RedactSensitiveValues(errorMessage string) string {
	// Regular expression to match URLs
	urlRegex := regexp.MustCompile(`https?://[^\s]+`)
	// Find all URLs in the error message
	urls := urlRegex.FindAllString(errorMessage, -1)

	for _, urlStr := range urls {
		// Parse the URL
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			continue
		}

		// Get the query parameters
		queryParams := parsedURL.Query()

		// Redact the 'sign' parameter if it exists
		if _, exists := queryParams["sign"]; exists {
			queryParams.Set("sign", "REDACTED")
			parsedURL.RawQuery = queryParams.Encode()
		}

		// Replace the original URL with the redacted one in the error message
		errorMessage = strings.Replace(errorMessage, urlStr, parsedURL.String(), -1)
	}

	return errorMessage
}

func getUrlSignContent(ctx context.Context, url *url.URL) string {
	// host := url.Host
	// if host == "" {
	// 	reqInfo := requestinfo.RequestInfoFromContext(ctx)
	// 	if reqInfo != nil {
	// 		host = reqInfo.Host
	// 	}
	// }
	// host = strings.TrimSuffix(host, "/")
	// // remove port if it exists
	// host = strings.Split(host, ":")[0]
	if url.Path == "" {
		return "/"
	}
	return url.Path
}
