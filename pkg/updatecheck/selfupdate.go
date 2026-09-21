package updatecheck

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const releaseBase = "https://github.com/" + ReleaseRepo + "/releases/download/"

// ApplyUpdate downloads the release asset matching this platform, verifies it
// against checksums.txt, replaces the running binary and restarts the process.
// On success it does not return (the process image is replaced / exited).
func ApplyUpdate(ctx context.Context, rel *ReleaseInfo) error {
	asset, err := platformAssetName("v" + rel.Version)
	if err != nil {
		return err
	}

	tmp, err := os.MkdirTemp("", "cloudreve-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	archivePath := filepath.Join(tmp, asset)
	if err := downloadTo(ctx, releaseBase+"v"+rel.Version+"/"+asset, archivePath); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	sumPath := filepath.Join(tmp, "checksums.txt")
	if err := downloadTo(ctx, releaseBase+"v"+rel.Version+"/checksums.txt", sumPath); err != nil {
		return fmt.Errorf("checksum download failed: %w", err)
	}
	if err := verifyChecksum(sumPath, asset, archivePath); err != nil {
		return err
	}

	binName := "cloudreve"
	if runtime.GOOS == "windows" {
		binName = "cloudreve.exe"
	}
	newBin := filepath.Join(tmp, binName)
	if err := extractBinary(archivePath, binName, newBin); err != nil {
		return fmt.Errorf("extract failed: %w", err)
	}

	if err := installAndRestart(newBin); err != nil {
		return err
	}
	// Windows staged a swap helper — exit so it can replace the binary.
	// os.Exit skips defers, so clean the temp dir explicitly first.
	// (Unix never reaches here: syscall.Exec replaced the process image.)
	if runtime.GOOS == "windows" {
		_ = os.RemoveAll(tmp)
		os.Exit(0)
	}
	return nil
}

// downloadTo streams url into path.
func downloadTo(ctx context.Context, url, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "cloudreve-update-check")
	resp, err := (&http.Client{Timeout: downloadTimeout}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", url, resp.Status)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

// verifyChecksum finds the asset's sha256 in checksums.txt and compares it to
// the downloaded file's digest.
func verifyChecksum(sumFile, asset, file string) error {
	data, err := os.ReadFile(sumFile)
	if err != nil {
		return err
	}
	var want string
	for _, line := range strings.Split(string(data), "\n") {
		// format: "<sha256>  <filename>"
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == asset {
			want = fields[0]
			break
		}
	}
	if want == "" {
		return fmt.Errorf("no checksum entry for %s", asset)
	}

	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	if got := hex.EncodeToString(h.Sum(nil)); !strings.EqualFold(got, want) {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", want, got)
	}
	return nil
}

// extractBinary pulls binName out of a .tar.gz or .zip archive into dst with
// executable permissions.
func extractBinary(archive, binName, dst string) error {
	if strings.HasSuffix(archive, ".zip") {
		return extractZip(archive, binName, dst)
	}
	return extractTarGz(archive, binName, dst)
}

func extractTarGz(archive, binName, dst string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg || filepath.Base(hdr.Name) != binName {
			continue
		}
		return writeExecutable(dst, tr)
	}
	return fmt.Errorf("%s not found in archive", binName)
}

func extractZip(archive, binName, dst string) error {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, zf := range zr.File {
		if zf.FileInfo().IsDir() || filepath.Base(zf.Name) != binName {
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			return err
		}
		defer rc.Close()
		return writeExecutable(dst, rc)
	}
	return fmt.Errorf("%s not found in archive", binName)
}

func writeExecutable(dst string, r io.Reader) error {
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}
