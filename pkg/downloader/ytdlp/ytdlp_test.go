package ytdlp

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/downloader"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/stretchr/testify/require"
)

type stubSettingProvider struct {
	setting.Provider
}

func TestParseSize(t *testing.T) {
	cases := map[string]int64{
		"10.50MiB": 10.5 * (1 << 20),
		"2 GiB":    2 << 30,
		"500KB":    500 * 1e3,
		"1.5B":     1,
		"":         0,
		"10.50ZiB": 0,
		"notasize": 0,
	}
	for in, want := range cases {
		require.Equal(t, want, parseSize(in), "input %q", in)
	}
}

func TestOptionArgs(t *testing.T) {
	args := optionArgs(map[string]any{
		"no_playlist":   true,
		"ignore_errors": false,
		"format":        "best",
		"retries":       3,
		"nil_key":       nil,
		"add_headers":   []string{"A: 1", "B: 2"},
	})
	require.Equal(t, []string{
		"--add-headers", "A: 1", "--add-headers", "B: 2",
		"--format", "best",
		"--no-playlist",
		"--retries", "3",
	}, args)
}

// fakeYtdlp writes a shell script emulating yt-dlp's progress output and file
// creation under the -P directory, then returns its path.
func fakeYtdlp(t *testing.T, script string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell-script stub requires a POSIX shell")
	}
	path := filepath.Join(t.TempDir(), "yt-dlp")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755))
	return path
}

func newTestClient(binary, temp string) downloader.Downloader {
	return New(
		logging.NewConsoleLogger(logging.LevelError),
		&stubSettingProvider{},
		&types.YtdlpSetting{Binary: binary, TempPath: temp},
	)
}

func awaitInfo(t *testing.T, d downloader.Downloader, h *downloader.TaskHandle, want downloader.Status) *downloader.TaskStatus {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	var last *downloader.TaskStatus
	for time.Now().Before(deadline) {
		s, err := d.Info(context.Background(), h)
		if err != nil {
			// Terminal entries are released after being reported once.
			if want == downloader.StatusCompleted || want == downloader.StatusError {
				return last
			}
			t.Fatalf("info failed: %v", err)
		}
		last = s
		if s.State == want {
			return s
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("state %q not reached, last: %+v", want, last)
	return nil
}

func TestCreateTaskCompletes(t *testing.T) {
	binary := fakeYtdlp(t, `
dir=""
prev=""
for a in "$@"; do
  if [ "$prev" = "-P" ]; then dir="$a"; fi
  prev="$a"
done
echo "[download]  50.0% of ~10.00MiB at 1.00MiB/s ETA 00:05"
echo "[download] 100% of 10.00MiB in 00:01"
echo data > "$dir/video.mp4"
`)
	d := newTestClient(binary, t.TempDir())

	handle, err := d.CreateTask(context.Background(), "https://example.com/v", map[string]interface{}{
		"no_playlist": true,
	})
	require.NoError(t, err)
	require.NotEmpty(t, handle.ID)

	status := awaitInfo(t, d, handle, downloader.StatusCompleted)
	require.NotNil(t, status)
	require.Equal(t, int64(10<<20), status.Total)
	require.Len(t, status.Files, 1)
	require.Equal(t, "video.mp4", status.Files[0].Name)
	require.True(t, status.Files[0].Selected)
	require.Equal(t, filepath.ToSlash(handle.Hash), status.SavePath)
}

func TestCreateTaskError(t *testing.T) {
	binary := fakeYtdlp(t, `echo "ERROR: Video unavailable" >&2; exit 1`)
	d := newTestClient(binary, t.TempDir())

	handle, err := d.CreateTask(context.Background(), "https://example.com/v", nil)
	require.NoError(t, err)

	status := awaitInfo(t, d, handle, downloader.StatusError)
	require.NotNil(t, status)
	require.Contains(t, status.ErrorMessage, "Video unavailable")
}

func TestCancel(t *testing.T) {
	binary := fakeYtdlp(t, `sleep 30`)
	d := newTestClient(binary, t.TempDir())

	handle, err := d.CreateTask(context.Background(), "https://example.com/v", nil)
	require.NoError(t, err)
	require.NoError(t, d.Cancel(context.Background(), handle))

	_, err = d.Info(context.Background(), handle)
	require.ErrorIs(t, err, downloader.ErrTaskNotFount)
}

func TestCustomOutputTemplate(t *testing.T) {
	binary := fakeYtdlp(t, `
dir=""
tmpl=""
prev=""
for a in "$@"; do
  if [ "$prev" = "-P" ]; then dir="$a"; fi
  if [ "$prev" = "-o" ]; then tmpl="$a"; fi
  prev="$a"
done
echo data > "$dir/$tmpl"
`)
	d := newTestClient(binary, t.TempDir())
	handle, err := d.CreateTask(context.Background(), "https://example.com/v", map[string]interface{}{
		"output": "custom-name.mp4",
	})
	require.NoError(t, err)
	status := awaitInfo(t, d, handle, downloader.StatusCompleted)
	require.NotNil(t, status)
	require.Len(t, status.Files, 1)
	require.Equal(t, "custom-name.mp4", status.Files[0].Name)
}
