package local

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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
