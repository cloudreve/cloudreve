package mediameta

import (
	"testing"
	"time"

	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver"
	"github.com/stretchr/testify/assert"
)

func findMeta(metas []driver.MediaMeta, key string) (driver.MediaMeta, bool) {
	for _, m := range metas {
		if m.Key == key {
			return m, true
		}
	}
	return driver.MediaMeta{}, false
}

func TestExtractExifMap(t *testing.T) {
	a := assert.New(t)
	metas := ExtractExifMap(map[string]string{
		"Artist":           "ann",
		"CameraModel":      "12345", // pure-uint vendor garbage, must fall back
		"Model":            "X-T5",
		"CameraMake":       "Fuji",
		"LensModel":        "9999", // uint, falls back
		"Lens":             "XF35",
		"Software":         "darktable",
		"PixelXDimension":  "6000",
		"ImageLength":      "4000",
		"ImageDescription": "desc",
		"ProjectionType":   "equirectangular",
		"DateTimeOriginal": "2020:01:02 03:04:05",
		"ISOSpeedRatings":  "200",
		"Flash":            "1",
		"FNumber":          "28/10",
		"Orientation":      "6",
		"ImageWidth":       "9999", // shadowed by PixelXDimension
	}, time.Time{})

	v, ok := findMeta(metas, CameraModel)
	a.True(ok)
	a.Equal("X-T5", v.Value)

	v, ok = findMeta(metas, LensModel)
	a.True(ok)
	a.Equal("XF35", v.Value)

	v, ok = findMeta(metas, PixelXDimension)
	a.True(ok)
	a.Equal("6000", v.Value)

	v, ok = findMeta(metas, PixelYDimension)
	a.True(ok)
	a.Equal("4000", v.Value)

	v, ok = findMeta(metas, Orientation)
	a.True(ok)
	a.Equal("6", v.Value)

	v, ok = findMeta(metas, TakenAt)
	a.True(ok)
	a.Equal("2020-01-02T03:04:05Z", v.Value)

	v, ok = findMeta(metas, Flash)
	a.True(ok)
	a.Equal("1", v.Value)

	v, ok = findMeta(metas, FNumber)
	a.True(ok)
	a.Equal("2.800000", v.Value)

	v, ok = findMeta(metas, ImageDescription)
	a.True(ok)
	a.Equal("desc", v.Value)

	v, ok = findMeta(metas, ProjectionType)
	a.True(ok)
	a.Equal("equirectangular", v.Value)

	_, ok = findMeta(metas, Copyright)
	a.False(ok)
}

func TestExtractExifMapDefaults(t *testing.T) {
	a := assert.New(t)
	metas := ExtractExifMap(map[string]string{}, time.Time{})

	// Orientation is always emitted, defaulting to "1".
	v, ok := findMeta(metas, Orientation)
	a.True(ok)
	a.Equal("1", v.Value)

	// No date tags and zero GPS time -> no TakenAt.
	_, ok = findMeta(metas, TakenAt)
	a.False(ok)
}
