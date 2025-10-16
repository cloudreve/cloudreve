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

	"github.com/bodgit/sevenzip"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager/entitysource"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
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

var ZipEncodings = map[string]encoding.Encoding{
	"ibm866":            charmap.CodePage866,
	"iso8859_2":         charmap.ISO8859_2,
	"iso8859_3":         charmap.ISO8859_3,
	"iso8859_4":         charmap.ISO8859_4,
	"iso8859_5":         charmap.ISO8859_5,
	"iso8859_6":         charmap.ISO8859_6,
	"iso8859_7":         charmap.ISO8859_7,
	"iso8859_8":         charmap.ISO8859_8,
	"iso8859_8I":        charmap.ISO8859_8I,
	"iso8859_10":        charmap.ISO8859_10,
	"iso8859_13":        charmap.ISO8859_13,
	"iso8859_14":        charmap.ISO8859_14,
	"iso8859_15":        charmap.ISO8859_15,
	"iso8859_16":        charmap.ISO8859_16,
	"koi8r":             charmap.KOI8R,
	"koi8u":             charmap.KOI8U,
	"macintosh":         charmap.Macintosh,
	"windows874":        charmap.Windows874,
	"windows1250":       charmap.Windows1250,
	"windows1251":       charmap.Windows1251,
	"windows1252":       charmap.Windows1252,
	"windows1253":       charmap.Windows1253,
	"windows1254":       charmap.Windows1254,
	"windows1255":       charmap.Windows1255,
	"windows1256":       charmap.Windows1256,
	"windows1257":       charmap.Windows1257,
	"windows1258":       charmap.Windows1258,
	"macintoshcyrillic": charmap.MacintoshCyrillic,
	"gbk":               simplifiedchinese.GBK,
	"gb18030":           simplifiedchinese.GB18030,
	"big5":              traditionalchinese.Big5,
	"eucjp":             japanese.EUCJP,
	"iso2022jp":         japanese.ISO2022JP,
	"shiftjis":          japanese.ShiftJIS,
	"euckr":             korean.EUCKR,
	"utf16be":           unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM),
	"utf16le":           unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM),
}

func (m *manager) ListArchiveFiles(ctx context.Context, uri *fs.URI, entity, zipEncoding string) ([]ArchivedFile, error) {
	file, err := m.fs.Get(ctx, uri, dbfs.WithFileEntities(), dbfs.WithRequiredCapabilities(dbfs.NavigatorCapabilityDownloadFile))
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	if file.Type() != types.FileTypeFile {
		return nil, fs.ErrNotSupportedAction.WithError(fmt.Errorf("path %s is not a file", uri))
	}

	// Validate file size
	if m.user.GroupMaxDecompressSize() > 0 && file.Size() > m.user.GroupMaxDecompressSize() {
		return nil, fs.ErrFileSizeTooBig.WithError(fmt.Errorf("file size %d exceeds the limit %d", file.Size(), m.user.GroupMaxDecompressSize()))
	}

	found, targetEntity := fs.FindDesiredEntity(file, entity, m.hasher, nil)
	if !found {
		return nil, fs.ErrEntityNotExist
	}

	var (
		enc encoding.Encoding
		ok  bool
	)
	if zipEncoding != "" {
		enc, ok = ZipEncodings[strings.ToLower(zipEncoding)]
		if !ok {
			return nil, fs.ErrNotSupportedAction.WithError(fmt.Errorf("not supported zip encoding: %s", zipEncoding))
		}
	}

	cacheKey := getArchiveListCacheKey(targetEntity.ID(), zipEncoding)
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

	var readerFunc func(ctx context.Context, file io.ReaderAt, size int64, textEncoding encoding.Encoding) ([]ArchivedFile, error)
	switch file.Ext() {
	case "zip":
		readerFunc = getZipFileList
	case "7z":
		readerFunc = get7zFileList
	default:
		return nil, fs.ErrNotSupportedAction.WithError(fmt.Errorf("not supported archive format: %s", file.Ext()))
	}

	sr := io.NewSectionReader(es, 0, targetEntity.Size())
	fileList, err := readerFunc(ctx, sr, targetEntity.Size(), enc)
	if err != nil {
		return nil, fmt.Errorf("failed to read file list: %w", err)
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

func getZipFileList(ctx context.Context, file io.ReaderAt, size int64, textEncoding encoding.Encoding) ([]ArchivedFile, error) {
	zr, err := zip.NewReader(file, size)
	if err != nil {
		return nil, fmt.Errorf("failed to create zip reader: %w", err)
	}

	fileList := make([]ArchivedFile, 0, len(zr.File))
	for _, f := range zr.File {
		hdr := f.FileHeader
		if hdr.NonUTF8 && textEncoding != nil {
			dec := textEncoding.NewDecoder()
			filename, err := dec.String(hdr.Name)
			if err == nil {
				hdr.Name = filename
			}
			if hdr.Comment != "" {
				comment, err := dec.String(hdr.Comment)
				if err == nil {
					hdr.Comment = comment
				}
			}
		}

		info := f.FileInfo()
		modTime := info.ModTime()
		fileList = append(fileList, ArchivedFile{
			Name:        util.FormSlash(hdr.Name),
			Size:        info.Size(),
			UpdatedAt:   &modTime,
			IsDirectory: info.IsDir(),
		})
	}
	return fileList, nil
}

func get7zFileList(ctx context.Context, file io.ReaderAt, size int64, extEncoding encoding.Encoding) ([]ArchivedFile, error) {
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

func getArchiveListCacheKey(entity int, encoding string) string {
	return fmt.Sprintf("archive_list_%d_%s", entity, encoding)
}
