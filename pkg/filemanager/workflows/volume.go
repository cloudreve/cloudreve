package workflows

import (
	"fmt"
	"io"
	iofs "io/fs"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager/entitysource"
)

// Multi-volume archive naming conventions. rardecode derives the next volume
// name from the current one and opens it via a custom fs.FS, so siblings only
// need to be resolvable by bare file name within the archive's folder.
// Only RAR conventions are recognized: it is the format rardecode can follow
// across volumes via fs.FS. Split zip/7z sets are intentionally excluded as
// their extractors cannot span files.
var (
	newRarVolumeRe = regexp.MustCompile(`^(.+)\.part(\d+)\.rar$`)
	oldRarVolumeRe = regexp.MustCompile(`^(.+)\.r(\d+)$`)
)

// volumeInfo returns the volume-set key and the ordering number of a file
// name. ok is false when the name is not a recognizable multi-volume member.
// The key is only comparable within one folder.
func volumeInfo(name string) (key string, order int64, ok bool) {
	lower := strings.ToLower(name)
	if m := newRarVolumeRe.FindStringSubmatch(lower); m != nil {
		n, _ := strconv.ParseInt(m[2], 10, 64)
		return "rar:" + m[1], n, true
	}
	if m := oldRarVolumeRe.FindStringSubmatch(lower); m != nil {
		n, _ := strconv.ParseInt(m[2], 10, 64)
		return "rarold:" + m[1], n + 1, true
	}
	if stem, found := strings.CutSuffix(lower, ".rar"); found {
		return "rarold:" + stem, 0, true
	}
	return "", 0, false
}

// firstVolumeName returns the expected file name of the first volume of the
// set the given member belongs to, or "" when the name already is the first
// volume or is not a recognizable later volume.
func firstVolumeName(name string) string {
	lower := strings.ToLower(name)
	if m := newRarVolumeRe.FindStringSubmatchIndex(lower); m != nil {
		if n, _ := strconv.Atoi(lower[m[4]:m[5]]); n > 1 {
			return fmt.Sprintf("%s.part%0*d.rar", name[m[2]:m[3]], len(lower[m[4]:m[5]]), 1)
		}
		return ""
	}
	if m := oldRarVolumeRe.FindStringSubmatchIndex(lower); m != nil {
		return name[m[2]:m[3]] + ".rar"
	}
	return ""
}

// FirstVolumeURI inspects a multi-file selection of cloudreve URIs and, when
// all files are volumes of the same archive set in the same folder, returns
// the URI of the first volume to extract from. ok is false when the
// selection is not a single volume set.
func FirstVolumeURI(srcs []string) (src string, ok bool) {
	if len(srcs) < 2 {
		return "", false
	}
	var key, dir string
	best, bestOrder := -1, int64(0)
	for i, s := range srcs {
		u, err := fs.NewUriFromString(s)
		if err != nil {
			return "", false
		}
		k, order, isVolume := volumeInfo(u.Name())
		if !isVolume {
			return "", false
		}
		if i == 0 {
			key, dir = k, u.Dir()
		} else if k != key || u.Dir() != dir {
			return "", false
		}
		if best < 0 || order < bestOrder {
			best, bestOrder = i, order
		}
	}
	return srcs[best], true
}

// archiveVolumeFS is an fs.FS resolving bare archive volume file names to
// entity sources. rardecode calls Open with volume names derived from the
// first volume's name; a missing sibling must surface fs.ErrNotExist so the
// decoder treats it as the end of the archive.
type archiveVolumeFS struct {
	open func(name string) (entitysource.EntitySource, error)
}

func (f *archiveVolumeFS) Open(name string) (iofs.File, error) {
	if name == "" || strings.ContainsRune(name, '/') || strings.ContainsRune(name, '\\') {
		return nil, &iofs.PathError{Op: "open", Path: name, Err: iofs.ErrNotExist}
	}

	es, err := f.open(name)
	if err != nil || es == nil {
		return nil, &iofs.PathError{Op: "open", Path: name, Err: iofs.ErrNotExist}
	}

	return &archiveVolumeFile{es: es, name: name}, nil
}

// archiveVolumeFile adapts an entity source to fs.File while keeping the
// io.Seeker passthrough rardecode uses for in-volume seeks.
type archiveVolumeFile struct {
	es   entitysource.EntitySource
	name string
}

func (f *archiveVolumeFile) Read(p []byte) (int, error)         { return f.es.Read(p) }
func (f *archiveVolumeFile) Seek(o int64, w int) (int64, error) { return f.es.Seek(o, w) }
func (f *archiveVolumeFile) Close() error                       { return f.es.Close() }
func (f *archiveVolumeFile) Stat() (iofs.FileInfo, error) {
	return archiveVolumeFileInfo{name: f.name, size: f.es.Entity().Size()}, nil
}

var _ io.Seeker = (*archiveVolumeFile)(nil)

type archiveVolumeFileInfo struct {
	name string
	size int64
}

func (i archiveVolumeFileInfo) Name() string         { return i.name }
func (i archiveVolumeFileInfo) Size() int64          { return i.size }
func (i archiveVolumeFileInfo) Mode() iofs.FileMode  { return 0 }
func (i archiveVolumeFileInfo) ModTime() time.Time   { return time.Time{} }
func (i archiveVolumeFileInfo) IsDir() bool          { return false }
func (i archiveVolumeFileInfo) Sys() any             { return nil }
