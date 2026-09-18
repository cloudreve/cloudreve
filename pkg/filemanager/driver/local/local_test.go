package local

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/stretchr/testify/require"
)

func TestDeletePrunesEmptyAncestors(t *testing.T) {
	old := util.UseWorkingDir
	util.UseWorkingDir = true
	t.Cleanup(func() { util.UseWorkingDir = old })
	t.Chdir(t.TempDir())

	nested := filepath.Join("uploads", "3", "deep", "nested")
	require.NoError(t, os.MkdirAll(nested, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(nested, "a.txt"), []byte("a"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(nested, "b.txt"), []byte("b"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join("uploads", "keep.txt"), []byte("k"), 0644))

	d := New(nil, logging.NewConsoleLogger(logging.LevelDebug), nil)

	// Deleting both files should remove deep/nested/3 but stop at uploads
	// (still holds keep.txt).
	failed, err := d.Delete(context.Background(),
		filepath.Join("uploads", "3", "deep", "nested", "a.txt"),
		filepath.Join("uploads", "3", "deep", "nested", "b.txt"),
	)
	require.NoError(t, err)
	require.Empty(t, failed)
	require.NoDirExists(t, filepath.Join("uploads", "3"))
	require.DirExists(t, "uploads")
	require.FileExists(t, filepath.Join("uploads", "keep.txt"))
}

func TestPutOutOfOrderChunks(t *testing.T) {
	old := util.UseWorkingDir
	util.UseWorkingDir = true
	t.Cleanup(func() { util.UseWorkingDir = old })
	t.Chdir(t.TempDir())

	d := New(nil, logging.NewConsoleLogger(logging.LevelDebug), nil)
	ctx := context.Background()
	savePath := filepath.Join("uploads", "blob.bin")

	props := &fs.UploadProps{SavePath: savePath, Size: 12}
	put := func(offset int64, data string) {
		require.NoError(t, d.Put(ctx, &fs.UploadRequest{
			Props:  props,
			Mode:   fs.ModeOverwrite,
			Offset: offset,
			File:   io.NopCloser(strings.NewReader(data)),
		}))
	}

	// Later chunk lands first — must not be rejected by the offset check.
	put(8, "IJXL")
	put(4, "EFGH")
	put(0, "ABCD")

	content, err := os.ReadFile(savePath)
	require.NoError(t, err)
	require.Equal(t, "ABCDEFGHIJXL", string(content))

	// Completion verifies the assembled size.
	require.NoError(t, d.CompleteUpload(ctx, &fs.UploadSession{Props: props}))

	// A lost chunk fails loudly at completion instead of silently corrupting.
	require.NoError(t, os.WriteFile(savePath, []byte("short"), 0644))
	err = d.CompleteUpload(ctx, &fs.UploadSession{Props: props})
	require.Error(t, err)
}
