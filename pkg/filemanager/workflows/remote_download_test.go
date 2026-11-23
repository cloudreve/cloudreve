package workflows

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/task"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster"
	"github.com/cloudreve/Cloudreve/v4/pkg/downloader"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/stretchr/testify/require"
)

func TestRemoteDownloadDestination(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "file.txt", want: "file.txt"},
		{name: "torrent/sub/file.txt", want: "torrent/sub/file.txt"},
		{name: `torrent/a:b\c?.txt`, want: "torrent/a_b_c_.txt"},
		{name: ".hidden/.../file.txt", want: ".hidden/.../file.txt"},
		{name: "%2e%2e/%2ffile.txt", want: "%2e%2e/%2ffile.txt"},
		{name: ""},
		{name: "."},
		{name: ".."},
		{name: "../outside.txt"},
		{name: "../../outside.txt"},
		{name: "../downloads-other/outside.txt"},
		{name: "torrent/../outside.txt"},
		{name: "torrent/./file.txt"},
		{name: "/absolute.txt"},
		{name: "torrent//file.txt"},
		{name: "torrent/"},
		{name: "file\x00.txt"},
	}
	for _, root := range []string{"cloudreve://my/downloads", "cloudreve://my/"} {
		t.Run(root, func(t *testing.T) {
			base, err := fs.NewUriFromString(root)
			require.NoError(t, err)
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					dst, err := remoteDownloadDestination(base, test.name)
					if test.want == "" {
						require.Error(t, err)
						require.Nil(t, dst)
						return
					}
					require.NoError(t, err)
					require.Equal(t, path.Join(base.Path(), test.want), dst.Path())
					require.True(t, dst.EqualOrIsDescendantOf(base, ""))
					require.False(t, dst.IsSame(base, ""))
				})
			}
		})
	}
}

func TestRemoteDownloadRejectsUnsafeFiles(t *testing.T) {
	ctx := context.Background()
	for _, name := range []string{"", "../outside.txt", "torrent/../outside.txt", "/absolute.txt"} {
		for _, mode := range []string{"prevalidate", "master", "slave"} {
			if mode == "prevalidate" && name == "" {
				continue
			}
			t.Run(mode+"/"+name, func(t *testing.T) {
				status := &downloader.TaskStatus{
					SavePath: t.TempDir(),
					Total:    2,
					Files: []downloader.TaskFile{
						{Index: 1, Name: "safe.txt", Selected: true, Size: 1},
						{Index: 2, Name: name, Selected: true, Size: 1},
					},
				}
				m := &RemoteDownloadTask{
					state: &RemoteDownloadTaskState{
						Dst:    "cloudreve://my/",
						Status: status,
						Phase:  RemoteDownloadTaskPhaseTransfer,
					},
				}
				if mode == "prevalidate" {
					require.Error(t, m.validateFiles(ctx, nil, status))
					return
				}
				var next task.Status
				var err error
				if mode == "master" {
					next, err = m.masterTransfer(ctx, nil)
				} else {
					next, err = m.slaveTransfer(ctx, nil)
				}
				require.Equal(t, task.StatusError, next)
				require.ErrorIs(t, err, queue.CriticalErr)
			})
		}
	}
}

type remoteDownloadTestSettings struct {
	setting.Provider
}

func (remoteDownloadTestSettings) MaxParallelTransfer(context.Context) int { return 1 }

type remoteDownloadTestDep struct {
	dependency.Dep
}

func (remoteDownloadTestDep) SettingProvider() setting.Provider {
	return remoteDownloadTestSettings{}
}

type remoteDownloadTestNode struct {
	cluster.Node
	state string
}

func (n *remoteDownloadTestNode) CreateTask(_ context.Context, taskType, state string) (int, error) {
	n.state = state
	return 1, nil
}

func TestRemoteDownloadSlaveTransferSourceRoot(t *testing.T) {
	ctx := context.WithValue(context.Background(), inventory.UserCtx{}, &ent.User{ID: 1})
	for _, root := range []string{"/downloads/task", "C:/downloads/task"} {
		t.Run(root, func(t *testing.T) {
			node := &remoteDownloadTestNode{}
			m := &RemoteDownloadTask{
				DBTask: &queue.DBTask{},
				node:   node,
				state: &RemoteDownloadTaskState{
					Dst: "cloudreve://my/downloads",
					Status: &downloader.TaskStatus{
						SavePath: root,
						Files: []downloader.TaskFile{
							{Index: 1, Name: "torrent/sub/a:b.txt", Selected: true},
						},
					},
				},
			}
			next, err := m.slaveTransfer(ctx, remoteDownloadTestDep{})
			require.NoError(t, err)
			require.Equal(t, task.StatusSuspending, next)
			var payload SlaveUploadTaskState
			require.NoError(t, json.Unmarshal([]byte(node.state), &payload))
			require.Equal(t, root, payload.SourceRoot)
			require.Len(t, payload.Files, 1)
			require.Equal(t, root+"/torrent/sub/a:b.txt", payload.Files[0].Src)
			require.Equal(t, "/downloads/torrent/sub/a_b.txt", payload.Files[0].Uri.Path())
		})
	}
}

func TestOpenFileInRoot(t *testing.T) {
	temp := t.TempDir()
	root := filepath.Join(temp, "download")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "torrent", "sub"), 0o700))
	inside := filepath.Join(root, "torrent", "sub", "file.txt")
	outside := filepath.Join(temp, "download-other", "file.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(outside), 0o700))
	require.NoError(t, os.WriteFile(inside, []byte("inside"), 0o600))
	require.NoError(t, os.WriteFile(outside, []byte("outside"), 0o600))

	t.Run("nested file", func(t *testing.T) {
		handle, err := openFileInRoot(filepath.ToSlash(root), filepath.ToSlash(inside))
		require.NoError(t, err)
		require.NoError(t, handle.Close())
	})
	t.Run("outside sibling", func(t *testing.T) {
		handle, err := openFileInRoot(filepath.ToSlash(root), filepath.ToSlash(outside))
		require.Error(t, err)
		require.Nil(t, handle)
	})
	t.Run("parent traversal", func(t *testing.T) {
		handle, err := openFileInRoot(root, root+"/../download-other/file.txt")
		require.Error(t, err)
		require.Nil(t, handle)
	})
	t.Run("root directory", func(t *testing.T) {
		handle, err := openFileInRoot(root, root)
		require.Error(t, err)
		require.Nil(t, handle)
	})
	t.Run("empty root", func(t *testing.T) {
		handle, err := openFileInRoot("", outside)
		require.Error(t, err)
		require.Nil(t, handle)
	})
	t.Run("source name is not sanitized", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("colon and literal backslash are not Windows filename characters")
		}
		src := filepath.Join(root, `a:b\c.txt`)
		require.NoError(t, os.WriteFile(src, []byte("inside"), 0o600))
		handle, err := openFileInRoot(root, src)
		require.NoError(t, err)
		require.NoError(t, handle.Close())
	})
	t.Run("symlink outside root", func(t *testing.T) {
		link := filepath.Join(root, "outside.txt")
		if err := os.Symlink(filepath.Join("..", "download-other", "file.txt"), link); err != nil {
			if runtime.GOOS == "windows" {
				t.Skipf("symlinks are unavailable: %v", err)
			}
			require.NoError(t, err)
		}
		handle, err := openFileInRoot(root, link)
		require.Error(t, err)
		require.Nil(t, handle)
	})
	t.Run("native Windows traversal", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			t.Skip("backslashes are path separators only on Windows")
		}
		handle, err := openFileInRoot(root, root+`\..\download-other\file.txt`)
		require.Error(t, err)
		require.Nil(t, handle)
	})
}
