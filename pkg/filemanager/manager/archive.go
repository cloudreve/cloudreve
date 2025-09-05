package manager

import (
	"archive/zip"
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bodgit/sevenzip"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager/entitysource"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"golang.org/x/tools/container/intsets"
)

type (
	ArchivedFile struct {
		Name        string     `json:"name"`
		Size        int64      `json:"size"`
		UpdatedAt   *time.Time `json:"updated_at"`
		IsDirectory bool       `json:"is_directory"`
	}
)

const (
	ArchiveListCacheTTL = 3600 // 1 hour
)

func init() {
	gob.Register([]ArchivedFile{})
}

func normalizeArchiveName(rawString string) string {
	// 1. 有效 UTF-8
	if utf8.ValidString(rawString) {
		return rawString
	}

	// 2. 尝试 GB18030（GBK 的超集）
	if dec, _, err := transform.String(simplifiedchinese.GB18030.NewDecoder(), rawString); err == nil && utf8.ValidString(dec) {
		return dec
	}

	// 3. 尝试 GBK
	if dec, _, err := transform.String(simplifiedchinese.GBK.NewDecoder(), rawString); err == nil && utf8.ValidString(dec) {
		return dec
	}

	// 4. 尝试 CP437（ZIP 规范默认）
	if dec, _, err := transform.String(charmap.CodePage437.NewDecoder(), rawString); err == nil && utf8.ValidString(dec) {
		return dec
	}

	return rawString
}

func (m *manager) ListArchiveFiles(ctx context.Context, uri *fs.URI, entity string) ([]ArchivedFile, error) {
	file, err := m.fs.Get(ctx, uri, dbfs.WithFileEntities(), dbfs.WithRequiredCapabilities(dbfs.NavigatorCapabilityDownloadFile))
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	if file.Type() != types.FileTypeFile {
		return nil, fs.ErrNotSupportedAction.WithError(fmt.Errorf("path %s is not a file", uri))
	}

	// Validate file size
	if m.user.Edges.Group.Settings.DecompressSize > 0 && file.Size() > m.user.Edges.Group.Settings.DecompressSize {
		return nil, fs.ErrFileSizeTooBig.WithError(fmt.Errorf("file size %d exceeds the limit %d", file.Size(), m.user.Edges.Group.Settings.DecompressSize))
	}

	found, targetEntity := fs.FindDesiredEntity(file, entity, m.hasher, nil)
	if !found {
		return nil, fs.ErrEntityNotExist
	}

	cacheKey := getArchiveListCacheKey(targetEntity.ID())
	kv := m.kv
	res, found := kv.Get(cacheKey)
	if found {
		return res.([]ArchivedFile), nil
	}

	es, err := m.GetEntitySource(ctx, 0, fs.WithEntity(targetEntity))
	if err != nil {
		return nil, fmt.Errorf("failed to get entity source: %w", err)
	}

	es.Apply(entitysource.WithContext(ctx))
	defer es.Close()

	var readerFunc func(ctx context.Context, file io.ReaderAt, size int64) ([]ArchivedFile, error)
	switch file.Ext() {
	case "zip":
		readerFunc = getZipFileList
	case "7z":
		readerFunc = get7zFileList
	default:
		return nil, fs.ErrNotSupportedAction.WithError(fmt.Errorf("not supported archive format: %s", file.Ext()))
	}

	sr := io.NewSectionReader(es, 0, targetEntity.Size())
	fileList, err := readerFunc(ctx, sr, targetEntity.Size())
	if err != nil {
		return nil, fmt.Errorf("failed to read file list: %w", err)
	}

	for i := range fileList {
		fileList[i].Name = normalizeArchiveName(fileList[i].Name)
	}

	kv.Set(cacheKey, fileList, ArchiveListCacheTTL)
	return fileList, nil
}

func (m *manager) CreateArchive(ctx context.Context, uris []*fs.URI, writer io.Writer, opts ...fs.Option) (int, error) {
	o := newOption()
	for _, opt := range opts {
		opt.Apply(o)
	}

	failed := 0

	// List all top level files
	files := make([]fs.File, 0, len(uris))
	for _, uri := range uris {
		file, err := m.Get(ctx, uri, dbfs.WithFileEntities(), dbfs.WithRequiredCapabilities(dbfs.NavigatorCapabilityDownloadFile), dbfs.WithNotRoot())
		if err != nil {
			return 0, fmt.Errorf("failed to get file %s: %w", uri, err)
		}

		files = append(files, file)
	}

	zipWriter := zip.NewWriter(writer)
	defer zipWriter.Close()

	var compressed int64
	for _, file := range files {
		if file.Type() == types.FileTypeFile {
			if err := m.compressFileToArchive(ctx, "/", file, zipWriter, o.ArchiveCompression, o.DryRun); err != nil {
				failed++
				m.l.Warning("Failed to compress file %s: %s, skipping it...", file.Uri(false), err)
			}

			compressed += file.Size()
			if o.ProgressFunc != nil {
				o.ProgressFunc(compressed, file.Size(), 0)
			}

			if o.MaxArchiveSize > 0 && compressed > o.MaxArchiveSize {
				return 0, fs.ErrArchiveSrcSizeTooBig
			}

		} else {
			if err := m.Walk(ctx, file.Uri(false), intsets.MaxInt, func(f fs.File, level int) error {
				if f.Type() == types.FileTypeFolder || f.IsSymbolic() {
					return nil
				}
				if err := m.compressFileToArchive(ctx, strings.TrimPrefix(f.Uri(false).Dir(),
					file.Uri(false).Dir()), f, zipWriter, o.ArchiveCompression, o.DryRun); err != nil {
					failed++
					m.l.Warning("Failed to compress file %s: %s, skipping it...", f.Uri(false), err)
				}

				compressed += f.Size()
				if o.ProgressFunc != nil {
					o.ProgressFunc(compressed, f.Size(), 0)
				}

				if o.MaxArchiveSize > 0 && compressed > o.MaxArchiveSize {
					return fs.ErrArchiveSrcSizeTooBig
				}

				return nil
			}); err != nil {
				m.l.Warning("Failed to walk folder %s: %s, skipping it...", file.Uri(false), err)
				failed++
			}
		}
	}

	return failed, nil
}

func (m *manager) compressFileToArchive(ctx context.Context, parent string, file fs.File, zipWriter *zip.Writer,
	compression bool, dryrun fs.CreateArchiveDryRunFunc) error {
	es, err := m.GetEntitySource(ctx, file.PrimaryEntityID())
	if err != nil {
		return fmt.Errorf("failed to get entity source for file %s: %w", file.Uri(false), err)
	}

	zipName := filepath.FromSlash(path.Join(parent, file.DisplayName()))
	if dryrun != nil {
		dryrun(zipName, es.Entity())
		return nil
	}

	m.l.Debug("Compressing %s to archive...", file.Uri(false))
	header := &zip.FileHeader{
		Name:               zipName,
		Modified:           file.UpdatedAt(),
		UncompressedSize64: uint64(file.Size()),
	}

	if !compression {
		header.Method = zip.Store
	} else {
		header.Method = zip.Deflate
	}

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed to create zip header for %s: %w", file.Uri(false), err)
	}

	es.Apply(entitysource.WithContext(ctx))
	_, err = io.Copy(writer, es)
	return err

}

func getZipFileList(ctx context.Context, file io.ReaderAt, size int64) ([]ArchivedFile, error) {
	zr, err := zip.NewReader(file, size)
	if err != nil {
		return nil, fmt.Errorf("failed to create zip reader: %w", err)
	}

	fileList := make([]ArchivedFile, 0, len(zr.File))
	for _, f := range zr.File {
		info := f.FileInfo()
		modTime := info.ModTime()
		fileList = append(fileList, ArchivedFile{
			Name:        util.FormSlash(f.Name),
			Size:        info.Size(),
			UpdatedAt:   &modTime,
			IsDirectory: info.IsDir(),
		})
	}
	return fileList, nil
}

func get7zFileList(ctx context.Context, file io.ReaderAt, size int64) ([]ArchivedFile, error) {
	zr, err := sevenzip.NewReader(file, size)
	if err != nil {
		return nil, fmt.Errorf("failed to create 7z reader: %w", err)
	}

	fileList := make([]ArchivedFile, 0, len(zr.File))
	for _, f := range zr.File {
		info := f.FileInfo()
		modTime := info.ModTime()
		fileList = append(fileList, ArchivedFile{
			Name:        util.FormSlash(f.Name),
			Size:        info.Size(),
			UpdatedAt:   &modTime,
			IsDirectory: info.IsDir(),
		})
	}
	return fileList, nil
}

func getArchiveListCacheKey(entity int) string {
	return fmt.Sprintf("archive_list_%d", entity)
}
