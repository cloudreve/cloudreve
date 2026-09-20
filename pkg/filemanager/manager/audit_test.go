package manager

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/stretchr/testify/require"
)

func newAuditTestClient(t *testing.T) request.Client {
	return request.NewClient(nil, request.WithTimeout(0))
}

func TestCallAuditEndpoint(t *testing.T) {
	t.Run("flagged", func(t *testing.T) {
		var gotBody []byte
		var gotName, gotSize, gotMime string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotBody, _ = io.ReadAll(r.Body)
			gotName = r.Header.Get("X-Entity-Name")
			gotSize = r.Header.Get("X-Entity-Size")
			gotMime = r.Header.Get("Content-Type")
			w.Write([]byte(`{"flagged":true,"reason":"nsfw"}`))
		}))
		t.Cleanup(srv.Close)

		flagged, reason, err := callAuditEndpoint(newAuditTestClient(t), srv.URL, "pic.png", 4, "image/png", strings.NewReader("data"))
		require.NoError(t, err)
		require.True(t, flagged)
		require.Equal(t, "nsfw", reason)
		require.Equal(t, "data", string(gotBody))
		require.Equal(t, "pic.png", gotName)
		require.Equal(t, "4", gotSize)
		require.Equal(t, "image/png", gotMime)
	})

	t.Run("clean", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"flagged":false}`))
		}))
		t.Cleanup(srv.Close)

		flagged, _, err := callAuditEndpoint(newAuditTestClient(t), srv.URL, "pic.png", 4, "image/png", strings.NewReader("data"))
		require.NoError(t, err)
		require.False(t, flagged)
	})

	t.Run("fail closed on endpoint error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		t.Cleanup(srv.Close)

		_, _, err := callAuditEndpoint(newAuditTestClient(t), srv.URL, "pic.png", 4, "image/png", strings.NewReader("data"))
		require.Error(t, err)
		require.Equal(t, serializer.CodeContentAuditFailed, err.(serializer.AppError).Code)
	})

	t.Run("fail closed on bad json", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`not json`))
		}))
		t.Cleanup(srv.Close)

		_, _, err := callAuditEndpoint(newAuditTestClient(t), srv.URL, "pic.png", 4, "image/png", strings.NewReader("data"))
		require.Error(t, err)
		require.Equal(t, serializer.CodeContentAuditFailed, err.(serializer.AppError).Code)
	})

	t.Run("rejects non-http schemes", func(t *testing.T) {
		for _, endpoint := range []string{"file:///etc/passwd", "ftp://x/", "://bad", ""} {
			_, _, err := callAuditEndpoint(newAuditTestClient(t), endpoint, "pic.png", 4, "image/png", strings.NewReader("data"))
			require.Error(t, err, endpoint)
			require.Equal(t, serializer.CodeContentAuditFailed, err.(serializer.AppError).Code)
		}
	})
}
