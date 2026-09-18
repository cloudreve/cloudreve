package workflows

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster"
	"github.com/cloudreve/Cloudreve/v4/pkg/downloader"
	"github.com/stretchr/testify/assert"
)

type stubNode struct {
	cluster.Node
	settings *types.NodeSetting
}

func (s *stubNode) Settings(ctx context.Context) *types.NodeSetting {
	return s.settings
}

func newRemoteDownloadTaskForOptions(state *RemoteDownloadTaskState, provider types.DownloaderProvider) *RemoteDownloadTask {
	return &RemoteDownloadTask{
		state: state,
		node:  &stubNode{settings: &types.NodeSetting{Provider: provider}},
	}
}

func TestBuildDownloadOptionsAria2(t *testing.T) {
	a := assert.New(t)
	m := newRemoteDownloadTaskForOptions(&RemoteDownloadTaskState{
		SrcUri:       "https://example.com/file.zip",
		FileName:     "renamed.zip",
		HTTPUsername: "user",
		HTTPPassword: "pass",
	}, types.DownloaderProviderAria2)

	base := map[string]interface{}{"max-connection-per-server": "4"}
	opts, taskUrl := m.buildDownloadOptions(context.Background(), base, "https://example.com/file.zip")

	a.Equal("renamed.zip", opts["out"])
	a.Equal("user", opts["http-user"])
	a.Equal("pass", opts["http-passwd"])
	a.Equal("4", opts["max-connection-per-server"])
	a.Equal("https://example.com/file.zip", taskUrl)
	_, exists := base["out"]
	a.False(exists, "base options must not be mutated")
}

func TestBuildDownloadOptionsQBittorrent(t *testing.T) {
	a := assert.New(t)
	m := newRemoteDownloadTaskForOptions(&RemoteDownloadTaskState{
		SrcUri:       "https://example.com/file.torrent",
		FileName:     "renamed",
		HTTPUsername: "user",
		HTTPPassword: "p@ss:word",
	}, types.DownloaderProviderQBittorrent)

	opts, taskUrl := m.buildDownloadOptions(context.Background(), nil, "https://example.com/file.torrent")

	a.Equal("renamed", opts["rename"])
	_, exists := opts["out"]
	a.False(exists)
	a.Equal("https://user:p%40ss%3Aword@example.com/file.torrent", taskUrl)
}

func TestBuildDownloadOptionsNonHttpSrc(t *testing.T) {
	a := assert.New(t)

	// Torrent source file: aria2 "out" and HTTP credentials must not apply.
	m := newRemoteDownloadTaskForOptions(&RemoteDownloadTaskState{
		SrcFileUri:   "cloudreve://my/file.torrent",
		FileName:     "renamed",
		HTTPUsername: "user",
		HTTPPassword: "pass",
	}, types.DownloaderProviderAria2)

	opts, taskUrl := m.buildDownloadOptions(context.Background(), nil, "https://slave.example.com/entity/1")
	_, hasOut := opts["out"]
	a.False(hasOut)
	_, hasUser := opts["http-user"]
	a.False(hasUser)
	a.Equal("https://slave.example.com/entity/1", taskUrl)

	// Magnet link: same for aria2.
	m = newRemoteDownloadTaskForOptions(&RemoteDownloadTaskState{
		SrcUri:       "magnet:?xt=urn:btih:abc",
		FileName:     "renamed",
		HTTPUsername: "user",
	}, types.DownloaderProviderAria2)
	opts, _ = m.buildDownloadOptions(context.Background(), nil, "magnet:?xt=urn:btih:abc")
	_, hasOut = opts["out"]
	a.False(hasOut)
	_, hasUser = opts["http-user"]
	a.False(hasUser)
}

func TestBuildDownloadOptionsNoExtras(t *testing.T) {
	a := assert.New(t)
	m := newRemoteDownloadTaskForOptions(&RemoteDownloadTaskState{
		SrcUri: "https://example.com/file.zip",
	}, types.DownloaderProviderAria2)

	base := map[string]interface{}{"k": "v"}
	opts, taskUrl := m.buildDownloadOptions(context.Background(), base, "https://example.com/file.zip")
	a.Equal("v", opts["k"])
	a.Equal("https://example.com/file.zip", taskUrl)
}

func TestBuildDownloadOptionsHeaders(t *testing.T) {
	a := assert.New(t)

	// aria2: arbitrary headers pass through as a list.
	m := newRemoteDownloadTaskForOptions(&RemoteDownloadTaskState{
		SrcUri:      "https://example.com/f.zip",
		HTTPHeaders: []string{"Cookie: sid=abc", "Referer: https://example.com/"},
	}, types.DownloaderProviderAria2)
	opts, _ := m.buildDownloadOptions(context.Background(), nil, "https://example.com/f.zip")
	a.Equal([]string{"Cookie: sid=abc", "Referer: https://example.com/"}, opts["header"])

	// qBittorrent: only the Cookie line maps onto the cookie field.
	m = newRemoteDownloadTaskForOptions(&RemoteDownloadTaskState{
		SrcUri:      "https://example.com/f.torrent",
		HTTPHeaders: []string{"Referer: https://x/", "Cookie: sid=abc"},
	}, types.DownloaderProviderQBittorrent)
	opts, _ = m.buildDownloadOptions(context.Background(), nil, "https://example.com/f.torrent")
	a.Equal("sid=abc", opts["cookie"])
	_, hasHeader := opts["header"]
	a.False(hasHeader)
}

func TestHasEarlyTransferCandidates(t *testing.T) {
	a := assert.New(t)
	m := &RemoteDownloadTask{state: &RemoteDownloadTaskState{}}

	status := &downloader.TaskStatus{Files: []downloader.TaskFile{
		{Index: 0, Selected: true, Progress: 1},
		{Index: 1, Selected: true, Progress: 0.5},
		{Index: 2, Selected: false, Progress: 1},
	}}
	a.True(m.hasEarlyTransferCandidates(status))

	m.state.Transferred = map[int]interface{}{0: struct{}{}}
	a.False(m.hasEarlyTransferCandidates(status))

	status.Files[1].Progress = 1
	a.True(m.hasEarlyTransferCandidates(status))
}

func TestNewRemoteDownloadTaskSanitizesFileName(t *testing.T) {
	a := assert.New(t)
	tsk, err := NewRemoteDownloadTask(context.Background(), "https://example.com/f", "", "cloudreve://my/dst", &RemoteDownloadTaskOption{
		FileName: "../../etc/passwd",
	})
	a.NoError(err)

	state := &RemoteDownloadTaskState{}
	a.NoError(json.Unmarshal([]byte(tsk.(*RemoteDownloadTask).Task.PrivateState), state))
	a.Equal(".._.._etc_passwd", state.FileName)
}
