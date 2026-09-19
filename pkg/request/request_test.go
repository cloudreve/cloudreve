package request

import (
	"context"
	"errors"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/stretchr/testify/assert"
	testMock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

type ClientMock struct {
	testMock.Mock
}

func (m *ClientMock) Request(method, target string, body io.Reader, opts ...Option) *Response {
	args := m.Called(method, target, body, opts)
	return args.Get(0).(*Response)
}

func TestWithTimeout(t *testing.T) {
	asserts := assert.New(t)
	options := newDefaultOption()
	WithTimeout(time.Duration(5) * time.Second).apply(options)
	asserts.Equal(time.Duration(5)*time.Second, options.timeout)
}

func TestWithHeader(t *testing.T) {
	asserts := assert.New(t)
	options := newDefaultOption()
	WithHeader(map[string][]string{"Origin": []string{"123"}}).apply(options)
	asserts.Equal(http.Header{"Origin": []string{"123"}}, options.header)
}

func TestWithContentLength(t *testing.T) {
	asserts := assert.New(t)
	options := newDefaultOption()
	WithContentLength(10).apply(options)
	asserts.EqualValues(10, options.contentLength)
}

func TestWithContext(t *testing.T) {
	asserts := assert.New(t)
	options := newDefaultOption()
	WithContext(context.Background()).apply(options)
	asserts.NotNil(options.ctx)
}

func TestHTTPClient_Request(t *testing.T) {
	asserts := assert.New(t)
	client := NewClient(nil, WithSlaveMeta(1))

	// 正常
	{
		resp := client.Request(
			"POST",
			"/test",
			strings.NewReader(""),
			WithTimeout(time.Duration(1)*time.Microsecond),
			WithCredential(auth.HMACAuth{SecretKey: []byte("123")}, 10),
			WithoutHeader([]string{"origin", "origin"}),
		)
		asserts.Error(resp.Err)
		asserts.Nil(resp.Response)
	}

	// 正常 带有ctx
	{
		resp := client.Request(
			"GET",
			"http://cloudreveisnotexist.com",
			strings.NewReader(""),
			WithContentLength(0),
			WithEndpoint("http://cloudreveisnotexist.com"),
			WithTimeout(time.Duration(1)*time.Microsecond),
			WithCredential(auth.HMACAuth{SecretKey: []byte("123")}, 10),
			WithContext(context.Background()),
		)
		asserts.Error(resp.Err)
		asserts.Nil(resp.Response)
	}

}

func TestResponse_GetResponse(t *testing.T) {
	asserts := assert.New(t)

	// 直接返回错误
	{
		resp := Response{
			Err: errors.New("error"),
		}
		content, err := resp.GetResponse()
		asserts.Empty(content)
		asserts.Error(err)
	}

	// 正常
	{
		resp := Response{
			Response: &http.Response{Body: ioutil.NopCloser(strings.NewReader("123"))},
		}
		content, err := resp.GetResponse()
		asserts.Equal("123", content)
		asserts.NoError(err)
	}
}

func TestResponse_CheckHTTPResponse(t *testing.T) {
	asserts := assert.New(t)

	// 直接返回错误
	{
		resp := Response{
			Err: errors.New("error"),
		}
		res := resp.CheckHTTPResponse(200)
		asserts.Error(res.Err)
	}

	// 404错误
	{
		resp := Response{
			Response: &http.Response{StatusCode: 404},
		}
		res := resp.CheckHTTPResponse(200)
		asserts.Error(res.Err)
	}

	// 通过
	{
		resp := Response{
			Response: &http.Response{StatusCode: 200},
		}
		res := resp.CheckHTTPResponse(200)
		asserts.NoError(res.Err)
	}
}

func TestResponse_GetRSCloser(t *testing.T) {
	asserts := assert.New(t)

	// 直接返回错误
	{
		resp := Response{
			Err: errors.New("error"),
		}
		res, err := resp.GetRSCloser()
		asserts.Error(err)
		asserts.Nil(res)
	}

	// 正常
	{
		resp := Response{
			Response: &http.Response{ContentLength: 3, Body: ioutil.NopCloser(strings.NewReader("123"))},
		}
		res, err := resp.GetRSCloser()
		asserts.NoError(err)
		content, err := ioutil.ReadAll(res)
		asserts.NoError(err)
		asserts.Equal("123", string(content))
		offset, err := res.Seek(0, 0)
		asserts.NoError(err)
		asserts.Equal(int64(0), offset)
		offset, err = res.Seek(0, 2)
		asserts.NoError(err)
		asserts.Equal(int64(3), offset)
		_, err = res.Seek(1, 2)
		asserts.Error(err)
		asserts.NoError(res.Close())
	}

}

func TestResponse_DecodeResponse(t *testing.T) {
	asserts := assert.New(t)

	// 直接返回错误
	{
		resp := Response{Err: errors.New("error")}
		response, err := resp.DecodeResponse()
		asserts.Error(err)
		asserts.Nil(response)
	}

	// 无法解析响应
	{
		resp := Response{
			Response: &http.Response{
				Body: ioutil.NopCloser(strings.NewReader("test")),
			},
		}
		response, err := resp.DecodeResponse()
		asserts.Error(err)
		asserts.Nil(response)
	}

	// 成功
	{
		resp := Response{
			Response: &http.Response{
				Body: ioutil.NopCloser(strings.NewReader("{\"code\":0}")),
			},
		}
		response, err := resp.DecodeResponse()
		asserts.NoError(err)
		asserts.NotNil(response)
		asserts.Equal(0, response.Code)
	}
}

func TestNopRSCloser_SetFirstFakeChunk(t *testing.T) {
	asserts := assert.New(t)
	rsc := NopRSCloser{
		status: &rscStatus{},
	}
	rsc.SetFirstFakeChunk()
	asserts.True(rsc.status.IgnoreFirst)

	rsc.SetContentLength(20)
	asserts.EqualValues(20, rsc.status.Size)
}

func TestBlackHole(t *testing.T) {
	a := assert.New(t)
	a.NotPanics(func() {
		BlackHole(strings.NewReader("TestBlackHole"))
	})
}

func TestHTTPClient_TPSLimit(t *testing.T) {
	a := assert.New(t)
	client := NewClient(nil, )

	finished := make(chan struct{})
	go func() {
		client.Request(
			"POST",
			"/test",
			strings.NewReader(""),
			WithTPSLimit("TestHTTPClient_TPSLimit", 1, 1),
		)
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(10 * time.Second):
		a.Fail("Request should be finished instantly.")
	}

	finished = make(chan struct{})
	go func() {
		client.Request(
			"POST",
			"/test",
			strings.NewReader(""),
			WithTPSLimit("TestHTTPClient_TPSLimit", 1, 1),
		)
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		a.Fail("Request should be finished in 1 second.")
	}

}

func TestWithSignBaseTime(t *testing.T) {
	asserts := assert.New(t)
	options := newDefaultOption()
	asserts.True(options.signBaseTime.IsZero())

	future := time.Now().Add(time.Hour)
	WithSignBaseTime(future).apply(options)
	asserts.Equal(future, options.signBaseTime)
}

func TestSignBaseTimeShiftsExpiry(t *testing.T) {
	asserts := assert.New(t)

	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"code":0}`))
	}))
	defer srv.Close()

	client := NewClient(nil, WithEndpoint(srv.URL))
	base := time.Now().Add(2 * time.Hour)
	resp := client.Request(
		"POST",
		"/upload",
		strings.NewReader("{}"),
		WithCredential(auth.HMACAuth{SecretKey: []byte("k")}, 60),
		WithSignBaseTime(base),
	)
	asserts.NoError(resp.Err)

	// Authorization: Cr <base64sign>:<expires>
	parts := strings.Split(authHeader, ":")
	require.GreaterOrEqual(t, len(parts), 2)
	expires, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	asserts.NoError(err)
	asserts.InDelta(base.Add(60*time.Second).Unix(), expires, 5)
}
