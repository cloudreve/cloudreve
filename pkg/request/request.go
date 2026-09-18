package request

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/samber/lo"
)

// GeneralClient 通用 HTTP Client
var GeneralClient Client = NewClientDeprecated()

const (
	CorrelationHeader = constants.CrHeaderPrefix + "Correlation-Id"
	SiteURLHeader     = constants.CrHeaderPrefix + "Site-Url"
	SiteVersionHeader = constants.CrHeaderPrefix + "Version"
	SiteIDHeader      = constants.CrHeaderPrefix + "Site-Id"
	SlaveNodeIDHeader = constants.CrHeaderPrefix + "Node-Id"
	ClientIDHeader    = constants.CrHeaderPrefix + "Client-Id"
	LocalIP           = "localhost"
)

// Response 请求的响应或错误信息
type Response struct {
	Err      error
	Response *http.Response
	l        logging.Logger
}

// Client 请求客户端
type Client interface {
	// Apply applies the given options to this client.
	Apply(opts ...Option)
	// Request send a HTTP request
	Request(method, target string, body io.Reader, opts ...Option) *Response
}

// HTTPClient 实现 Client 接口
type HTTPClient struct {
	mu         sync.Mutex
	options    *options
	tpsLimiter TPSLimiter
	l          logging.Logger
	config     conf.ConfigProvider
}

func NewClient(config conf.ConfigProvider, opts ...Option) Client {
	client := &HTTPClient{
		options:    newDefaultOption(),
		tpsLimiter: globalTPSLimiter,
		config:     config,
	}

	for _, o := range opts {
		o.apply(client.options)
	}

	return client
}

// Deprecated
func NewClientDeprecated(opts ...Option) Client {
	client := &HTTPClient{
		options:    newDefaultOption(),
		tpsLimiter: globalTPSLimiter,
	}

	for _, o := range opts {
		o.apply(client.options)
	}

	return client
}

func (c *HTTPClient) Apply(opts ...Option) {
	for _, o := range opts {
		o.apply(c.options)
	}
}

// Request 发送HTTP请求
func (c *HTTPClient) Request(method, target string, body io.Reader, opts ...Option) *Response {
	// 应用额外设置
	c.mu.Lock()
	options := c.options.clone()
	c.mu.Unlock()
	for _, o := range opts {
		o.apply(&options)
	}

	// 创建请求客户端
	client := &http.Client{
		Timeout: options.timeout,
		Jar:     options.cookieJar,
	}

	if options.transport != nil {
		client.Transport = options.transport
	}

	// size为0时将body设为nil
	if options.contentLength == 0 {
		body = nil
	}

	// 确定请求URL
	if options.endpoint != nil {
		targetPath, err := url.Parse(target)
		if err != nil {
			return &Response{Err: err}
		}

		targetURL := *options.endpoint
		target = targetURL.ResolveReference(targetPath).String()
	}

	// 创建请求
	var (
		req *http.Request
		err error
	)
	start := time.Now()
	if options.ctx != nil {
		req, err = http.NewRequestWithContext(options.ctx, method, target, body)
	} else {
		req, err = http.NewRequest(method, target, body)
	}
	if err != nil {
		return &Response{Err: err}
	}

	// 添加请求相关设置
	if options.header != nil {
		for k, v := range options.header {
			req.Header.Add(k, strings.Join(v, " "))
		}
	}

	req.Header.Set("User-Agent", "Cloudreve/"+constants.BackendVersion)

	if options.ctx != nil && options.withCorrelationID {
		req.Header.Add(CorrelationHeader, logging.CorrelationID(options.ctx).String())
	}

	if c.config != nil && options.masterMeta && c.config.System().Mode == conf.MasterMode {
		req.Header.Add(SiteURLHeader, options.siteURL)
		req.Header.Add(SiteIDHeader, options.siteID)
		req.Header.Add(SiteVersionHeader, constants.BackendVersion)
	}

	if options.slaveNodeID > 0 {
		req.Header.Add(SlaveNodeIDHeader, strconv.Itoa(options.slaveNodeID))
	}

	if options.contentLength != -1 {
		req.ContentLength = options.contentLength
	}

	// 签名请求
	if options.sign != nil {
		ctx := options.ctx
		if options.ctx == nil {
			ctx = context.Background()
		}
		expireBase := options.signBaseTime
		if expireBase.IsZero() {
			expireBase = time.Now()
		}
		expire := expireBase.Add(time.Second * time.Duration(options.signTTL))
		switch method {
		case "PUT", "POST", "PATCH":
			auth.SignRequest(ctx, options.sign, req, &expire)
		default:
			if resURL, err := auth.SignURI(ctx, options.sign, req.URL.String(), &expire); err == nil {
				req.URL = resURL
			}
		}
	}

	if options.tps > 0 {
		c.tpsLimiter.Limit(options.ctx, options.tpsLimiterToken, options.tps, options.tpsBurst)
	}

	// 发送请求
	resp, err := client.Do(req)

	// Logging request
	if options.logger != nil {
		statusCode := 0
		errStr := ""
		if resp != nil {
			statusCode = resp.StatusCode
		}

		if err != nil {
			errStr = err.Error()
		}

		logging.Request(options.logger, false, statusCode, req.Method, LocalIP, req.URL.String(), errStr, start)
	}

	// Apply cookies
	if resp != nil && resp.Cookies() != nil && options.cookieJar != nil {
		options.cookieJar.SetCookies(req.URL, resp.Cookies())
	}

	if err != nil {
		return &Response{Err: err}
	}

	return &Response{Err: nil, Response: resp, l: options.logger}
}

// GetResponse 检查响应并获取响应正文
func (resp *Response) GetResponse() (string, error) {
	if resp.Err != nil {
		return "", resp.Err
	}
	respBody, err := io.ReadAll(resp.Response.Body)
	_ = resp.Response.Body.Close()

	return string(respBody), err
}

// GetResponseIgnoreErr 获取响应正文
func (resp *Response) GetResponseIgnoreErr() (string, error) {
	if resp.Response == nil {
		return "", resp.Err
	}

	respBody, _ := io.ReadAll(resp.Response.Body)
	_ = resp.Response.Body.Close()

	return string(respBody), resp.Err
}

// CheckHTTPResponse 检查请求响应HTTP状态码
func (resp *Response) CheckHTTPResponse(status ...int) *Response {
	if resp.Err != nil {
		return resp
	}

	// 检查HTTP状态码
	if !lo.Contains(status, resp.Response.StatusCode) {
		resp.Err = fmt.Errorf("Remote returns unexpected status code: %d", resp.Response.StatusCode)
	}
	return resp
}

// DecodeResponse 尝试解析为serializer.Response，并对状态码进行检查
func (resp *Response) DecodeResponse() (*serializer.Response, error) {
	if resp.Err != nil {
		return nil, resp.Err
	}

	respString, err := resp.GetResponse()
	if err != nil {
		return nil, err
	}

	var res serializer.Response
	err = json.Unmarshal([]byte(respString), &res)
	if err != nil {
		if resp.l != nil {
			resp.l.Debug("Failed to parse response: %s", respString)
		}

		return nil, err
	}
	return &res, nil
}

// NopRSCloser 实现不完整seeker
type NopRSCloser struct {
	body   io.ReadCloser
	status *rscStatus
}

type rscStatus struct {
	// http.ServeContent 会读取一小块以决定内容类型，
	// 但是响应body无法实现seek，所以此项为真时第一个read会返回假数据
	IgnoreFirst bool

	Size int64
}

// GetRSCloser 返回带有空seeker的RSCloser，供http.ServeContent使用
func (resp *Response) GetRSCloser() (*NopRSCloser, error) {
	if resp.Err != nil {
		return nil, resp.Err
	}

	return &NopRSCloser{
		body: resp.Response.Body,
		status: &rscStatus{
			Size: resp.Response.ContentLength,
		},
	}, resp.Err
}

// SetFirstFakeChunk 开启第一次read返回空数据
// TODO 测试
func (instance NopRSCloser) SetFirstFakeChunk() {
	instance.status.IgnoreFirst = true
}

// SetContentLength 设置数据流大小
func (instance NopRSCloser) SetContentLength(size int64) {
	instance.status.Size = size
}

// Read 实现 NopRSCloser reader
func (instance NopRSCloser) Read(p []byte) (n int, err error) {
	if instance.status.IgnoreFirst && len(p) == 512 {
		return 0, io.EOF
	}
	return instance.body.Read(p)
}

// Close 实现 NopRSCloser closer
func (instance NopRSCloser) Close() error {
	return instance.body.Close()
}

// Seek 实现 NopRSCloser seeker, 只实现seek开头/结尾以便http.ServeContent用于确定正文大小
func (instance NopRSCloser) Seek(offset int64, whence int) (int64, error) {
	// 进行第一次Seek操作后，取消忽略选项
	if instance.status.IgnoreFirst {
		instance.status.IgnoreFirst = false
	}
	if offset == 0 {
		switch whence {
		case io.SeekStart:
			return 0, nil
		case io.SeekEnd:
			return instance.status.Size, nil
		}
	}
	return 0, errors.New("not implemented")

}
