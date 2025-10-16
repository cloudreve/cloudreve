package util

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	DataFolder = "data"
)

var UseWorkingDir = true

// DotPathToStandardPath 将","分割的路径转换为标准路径
func DotPathToStandardPath(path string) string {
	return "/" + strings.Replace(path, ",", "/", -1)
}

// FillSlash 给路径补全`/`
func FillSlash(path string) string {
	if path == "/" {
		return path
	}
	return path + "/"
}

// RemoveSlash 移除路径最后的`/`
func RemoveSlash(path string) string {
	if len(path) > 1 {
		return strings.TrimSuffix(path, "/")
	}
	return path
}

// SplitPath 分割路径为列表
func SplitPath(path string) []string {
	if len(path) == 0 || path[0] != '/' {
		return []string{}
	}

	if path == "/" {
		return []string{"/"}
	}

	pathSplit := strings.Split(path, "/")
	pathSplit[0] = "/"
	return pathSplit
}

// FormSlash 将path中的反斜杠'\'替换为'/'
func FormSlash(old string) string {
	return path.Clean(strings.ReplaceAll(old, "\\", "/"))
}

// RelativePath 获取相对可执行文件的路径
func RelativePath(name string) string {
	if UseWorkingDir {
		return name
	}

	if filepath.IsAbs(name) {
		return name
	}
	e, _ := os.Executable()
	return filepath.Join(filepath.Dir(e), name)
}

// DataPath relative path for store persist data file
func DataPath(child string) string {
	dataPath := RelativePath(DataFolder)
	if !Exists(dataPath) {
		os.MkdirAll(dataPath, 0700)
	}

	if filepath.IsAbs(child) {
		return child
	}

	return filepath.Join(dataPath, child)
}

// MkdirIfNotExist create directory if not exist
func MkdirIfNotExist(ctx context.Context, p string) {
	if !Exists(p) {
		os.MkdirAll(p, 0700)
	}
}

// SlashClean is equivalent to but slightly more efficient than
// path.Clean("/" + name).
func SlashClean(name string) string {
	if name == "" || name[0] != '/' {
		name = "/" + name
	}
	return path.Clean(name)
}

// Ext returns the file name extension used by path, without the dot.
func Ext(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if len(ext) > 0 {
		ext = ext[1:]
	}
	return ext
}
