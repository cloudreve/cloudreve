package remote

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/stretchr/testify/require"
)

// TestCreateUploadSessionClockSkewRetry reproduces upstream #3242: when the
// slave's clock is behind the master's, the upload-session signature arrives
// already expired. The client must learn the offset from the response Date
// header and retry with a compensated base time.
func TestCreateUploadSessionClockSkewRetry(t *testing.T) {
	const skewSeconds = 3600 // master is 1h ahead of slave
	masterNow := func() time.Time { return time.Now().Add(skewSeconds * time.Second) }

	var calls atomic.Int32
	var lastExpires int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		authHeader := r.Header.Get("Authorization")
		parts := strings.Split(authHeader, ":")
		expires, _ := strconv.ParseInt(parts[len(parts)-1], 10, 64)
		lastExpires = expires

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Date", masterNow().UTC().Format(http.TimeFormat))
		if expires < masterNow().Unix() {
			w.Write([]byte(`{"code":40005,"msg":"signature expired"}`))
			return
		}
		w.Write([]byte(`{"code":0}`))
	}))
	defer srv.Close()

	c := &remoteClient{
		l: logging.NewConsoleLogger(logging.LevelError),
		httpClient: request.NewClient(
			nil,
			request.WithEndpoint(srv.URL),
			request.WithCredential(auth.HMACAuth{SecretKey: []byte("test")}, 60),
		),
	}

	err := c.CreateUploadSession(context.Background(), &fs.UploadSession{}, false)
	require.NoError(t, err)
	require.Equal(t, int32(2), calls.Load(), "expected one expired attempt plus one compensated retry")
	require.InDelta(t, skewSeconds, c.clockOffset.Load(), 1)
	require.GreaterOrEqual(t, lastExpires, masterNow().Unix())
}

// TestLearnClockOffsetIgnoresBadHeaders ensures malformed or missing Date
// headers leave the offset untouched.
func TestLearnClockOffsetIgnoresBadHeaders(t *testing.T) {
	c := &remoteClient{l: logging.NewConsoleLogger(logging.LevelError)}

	c.learnClockOffset(nil)
	require.Zero(t, c.clockOffset.Load())

	c.learnClockOffset(&http.Response{Header: http.Header{}})
	require.Zero(t, c.clockOffset.Load())

	c.learnClockOffset(&http.Response{Header: http.Header{"Date": {"garbage"}}})
	require.Zero(t, c.clockOffset.Load())
}
