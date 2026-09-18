package workflows

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster"
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
