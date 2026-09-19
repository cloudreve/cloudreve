package aria2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/downloader"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/stretchr/testify/require"
)

type stubSettingProvider struct {
	setting.Provider
}

// rpcStub answers aria2 JSON-RPC over HTTP for the methods under test.
func rpcStub(t *testing.T, responses map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			Id     any    `json:"id"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))

		result, ok := responses[req.Method]
		require.True(t, ok, "unexpected rpc method: %s", req.Method)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.Id,
			"result":  result,
		})
	}))
}

func newTestClient(server string) downloader.Downloader {
	return New(
		logging.NewConsoleLogger(logging.LevelError),
		&stubSettingProvider{},
		&types.Aria2Setting{Server: server, TempPath: "/tmp"},
	)
}

func TestCreateTaskReturnsHandle(t *testing.T) {
	srv := rpcStub(t, map[string]any{"aria2.addUri": "gid-abc"})
	defer srv.Close()

	c := newTestClient(srv.URL)
	handle, err := c.CreateTask(context.Background(), "https://example.com/file", nil)
	require.NoError(t, err)
	require.Equal(t, "gid-abc", handle.ID)
}

func TestInfoMapsStatuses(t *testing.T) {
	cases := []struct {
		aria2Status string
		want        downloader.Status
	}{
		{"complete", downloader.StatusCompleted},
		{"error", downloader.StatusError},
		{"waiting", downloader.StatusDownloading},
		{"paused", downloader.StatusDownloading},
	}
	for _, tc := range cases {
		srv := rpcStub(t, map[string]any{
			"aria2.tellStatus": map[string]any{
				"gid":             "g1",
				"status":          tc.aria2Status,
				"totalLength":     "100",
				"completedLength": "50",
				"bittorrent":      map[string]any{},
			},
		})
		c := newTestClient(srv.URL)
		status, err := c.Info(context.Background(), &downloader.TaskHandle{ID: "g1"})
		require.NoError(t, err)
		require.Equal(t, tc.want, status.State, "aria2 status %q", tc.aria2Status)
		srv.Close()
	}
}

func TestInfoSeedingWhenTorrentComplete(t *testing.T) {
	srv := rpcStub(t, map[string]any{
		"aria2.tellStatus": map[string]any{
			"gid":             "g1",
			"status":          "active",
			"totalLength":     "100",
			"completedLength": "100",
			"bittorrent":      map[string]any{"mode": "single"},
		},
	})
	defer srv.Close()

	c := newTestClient(srv.URL)
	status, err := c.Info(context.Background(), &downloader.TaskHandle{ID: "g1"})
	require.NoError(t, err)
	require.Equal(t, downloader.StatusSeeding, status.State)
}

func TestCancelRemovesTask(t *testing.T) {
	srv := rpcStub(t, map[string]any{
		"aria2.tellStatus": map[string]any{
			"gid":             "g1",
			"status":          "active",
			"totalLength":     "100",
			"completedLength": "10",
			"bittorrent":      map[string]any{},
			"dir":             "/tmp",
		},
		"aria2.remove": "g1",
	})
	defer srv.Close()

	c := newTestClient(srv.URL)
	require.NoError(t, c.Cancel(context.Background(), &downloader.TaskHandle{ID: "g1"}))
}
