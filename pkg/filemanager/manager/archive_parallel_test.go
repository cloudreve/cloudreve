package manager

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWriteRawEntryRoundTrip verifies that a deflate stream spooled to a
// temp file with precomputed CRC32/sizes produces a valid zip entry via
// CreateRaw — the invariant the parallel path relies on (#3316).
func TestWriteRawEntryRoundTrip(t *testing.T) {
	payload := []byte("hello cloudreve parallel zip payload, repeat repeat repeat")

	var compBuf bytes.Buffer
	cw := &countingWriter{w: &compBuf}
	crc := crc32.NewIEEE()
	fw, err := flate.NewWriter(cw, flate.DefaultCompression)
	require.NoError(t, err)
	_, err = io.Copy(fw, io.TeeReader(bytes.NewReader(payload), crc))
	require.NoError(t, err)
	require.NoError(t, fw.Close())

	tmp := filepath.Join(t.TempDir(), "entry.deflate")
	require.NoError(t, os.WriteFile(tmp, compBuf.Bytes(), 0o600))

	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	err = writeRawEntry(zw, archiveEntryResult{
		header: &zip.FileHeader{
			Name:               "dir/a.txt",
			Method:             zip.Deflate,
			CRC32:              crc.Sum32(),
			CompressedSize64:   uint64(cw.n),
			UncompressedSize64: uint64(len(payload)),
		},
		tmpPath: tmp,
	})
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	// Temp file is consumed and removed.
	_, statErr := os.Stat(tmp)
	require.True(t, os.IsNotExist(statErr))

	zr, err := zip.NewReader(bytes.NewReader(zipBuf.Bytes()), int64(zipBuf.Len()))
	require.NoError(t, err)
	require.Len(t, zr.File, 1)
	require.Equal(t, "dir/a.txt", zr.File[0].Name)
	require.Equal(t, uint64(len(payload)), zr.File[0].UncompressedSize64)

	rc, err := zr.File[0].Open()
	require.NoError(t, err)
	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.NoError(t, rc.Close())
	require.Equal(t, payload, got)
}
