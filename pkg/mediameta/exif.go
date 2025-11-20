package mediameta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager/entitysource"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
	heicexif "github.com/dsoprea/go-heic-exif-extractor"
	jpegstructure "github.com/dsoprea/go-jpeg-image-structure"
	pngstructure "github.com/dsoprea/go-png-image-structure"
	tiffstructure "github.com/dsoprea/go-tiff-image-structure"
	riimage "github.com/dsoprea/go-utility/image"
)

var (
	exifExts = []string{
		"jpg",
		"jpeg",
		"png",
		"heic",
		"heif",
		"tiff",
		"avif",
		// R
		"3fr", "ari", "arw", "bay", "braw", "crw", "cr2", "cr3", "cap", "data", "dcs", "dcr", "dng", "drf", "eip", "erf", "fff", "gpr", "iiq", "k25", "kdc", "mdc", "mef", "mos", "mrw", "nef", "nrw", "obm", "orf", "pef", "ptx", "pxn", "r3d", "raf", "raw", "rwl", "rw2", "rwz", "sr2", "srf", "srw", "tif", "x3f",
	}
	exifIfdMapping       *exifcommon.IfdMapping
	exifTagIndex         = exif.NewTagIndex()
	exifDateTimeTags     = []string{"DateTimeOriginal", "DateTimeCreated", "CreateDate", "DateTime", "DateTimeDigitized"}
	ExifDateTimeMatch    = make(map[string]int)
	ExifDateTimeRegexp   = regexp.MustCompile("((?P<year>\\d{4})|\\D{4})\\D((?P<month>\\d{2})|\\D{2})\\D((?P<day>\\d{2})|\\D{2})\\D((?P<h>\\d{2})|\\D{2})\\D((?P<m>\\d{2})|\\D{2})\\D((?P<s>\\d{2})|\\D{2})(\\.(?P<subsec>\\d+))?(?P<z>\\D)?(?P<zh>\\d{2})?\\D?(?P<zm>\\d{2})?")
	YearMax              = time.Now().Add(OneYear * 3).Year()
	UnwantedDescriptions = map[string]bool{
		"Created by Imlib":         true, // Apps
		"iClarified":               true,
		"OLYMPUS DIGITAL CAMERA":   true, // Olympus
		"SAMSUNG":                  true, // Samsung
		"SAMSUNG CAMERA PICTURES":  true,
		"<Digimax i5, Samsung #1>": true,
		"SONY DSC":                 true, // Sony
		"rhdr":                     true, // Huawei
		"hdrpl":                    true,
		"oznorWO":                  true,
		"frontbhdp":                true,
		"fbt":                      true,
		"rbt":                      true,
		"ptr":                      true,
		"fbthdr":                   true,
		"btr":                      true,
		"mon":                      true,
		"nor":                      true,
		"dav":                      true,
		"mde":                      true,
		"mde_soft":                 true,
		"edf":                      true,
		"btfmdn":                   true,
		"btf":                      true,
		"btfhdr":                   true,
		"frem":                     true,
		"oznor":                    true,
		"rpt":                      true,
		"burst":                    true,
		"sdr_HDRB":                 true,
		"cof":                      true,
		"qrf":                      true,
		"fshbty":                   true,
		"binary comment":           true, // Other
		"default":                  true,
		"Exif_JPEG_PICTURE":        true,
		"DVC 10.1 HDMI":            true,
		"charset=Ascii":            true,
	}
)

const (
	OneYear = time.Hour * 24 * 365
	LatMax  = 90
	LngMax  = 180

	GpsLat            = "latitude"
	GpsLng            = "longitude"
	GpsAttitude       = "altitude"
	Artist            = "artist"
	Copyright         = "copyright"
	CameraModel       = "camera_model"
	CameraMake        = "camera_make"
	CameraOwnerName   = "camera_owner"
	BodySerialNumber  = "body_serial"
	LensMake          = "lens_make"
	LensModel         = "lens_model"
	Software          = "software"
	ExposureTime      = "exposure_time"
	FNumber           = "f"
	ApertureValue     = "aperture"
	FocalLength       = "focal_length"
	ISOSpeedRatings   = "iso"
	PixelXDimension   = "x"
	PixelYDimension   = "y"
	Orientation       = "orientation"
	TakenAt           = "taken_at"
	Flash             = "flash"
	ImageDescription  = "des"
	ProjectionType    = "projection_type"
	ExposureBiasValue = "exposure_bias"
)

func init() {
	exifIfdMapping = exifcommon.NewIfdMapping()
	_ = exifcommon.LoadStandardIfds(exifIfdMapping)
	names := ExifDateTimeRegexp.SubexpNames()
	for i := 0; i < len(names); i++ {
		if name := names[i]; name != "" {
			ExifDateTimeMatch[name] = i
		}
	}
}

type exifExtractor struct {
	settings setting.Provider
	l        logging.Logger
}

func newExifExtractor(settings setting.Provider, l logging.Logger) *exifExtractor {
	return &exifExtractor{
		settings: settings,
		l:        l,
	}
}

func (e *exifExtractor) Exts() []string {
	return exifExts
}

// Reference: https://github.com/photoprism/photoprism/blob/602097635f1c84d91f2d919f7aedaef7a07fc458/internal/meta/exif.go
func (e *exifExtractor) Extract(ctx context.Context, ext string, source entitysource.EntitySource, opts ...optionFunc) ([]driver.MediaMeta, error) {
	option := &option{}
	for _, opt := range opts {
		opt.apply(option)
	}

	localLimit, remoteLimit := e.settings.MediaMetaExifSizeLimit(ctx)
	if err := checkFileSize(localLimit, remoteLimit, source); err != nil {
		return nil, err
	}

	bruteForce := e.settings.MediaMetaExifBruteForce(ctx)
	var (
		err      error
		exifData []byte
	)
	parser := getExifParser(ext)
	if parser == nil {
		if !bruteForce {
			return nil, errors.New("no available exif parser found")
		}

	} else {
		var res riimage.MediaContext
		res, err = parser.Parse(source, int(source.Entity().Size()))
		if err != nil {
			err = fmt.Errorf("failed to parse exif: %s", err)
		} else {
			_, exifData, err = res.Exif()
			if err != nil {
				err = fmt.Errorf("failed to parse exif root: %s", err)
			}
		}
	}

	if !bruteForce && err != nil {
		return nil, err
	} else if bruteForce && (err != nil || parser == nil) {
		e.l.Debug("Failed to parse exif: %s, trying brute force.", err)
		exifData, err = exif.SearchAndExtractExifWithReader(source)
		if err != nil {
			if errors.Is(err, exif.ErrNoExif) {
				e.l.Debug("No exif data found")
				return nil, nil
			}

			return nil, fmt.Errorf("failed to brute force to parse exif: %s", err)
		}
	}

	entries, _, err := exif.GetFlatExifData(exifData, &exif.ScanOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse exif entries: %s", err)
	}

	exifMap := make(map[string]string, len(entries))
	for _, tag := range entries {
		s := strings.Split(tag.FormattedFirst, "\x00")
		if tag.TagName == "" || len(s) == 0 {
		} else if s[0] != "" && (exifMap[tag.TagName] == "" || tag.IfdPath != exif.ThumbnailFqIfdPath) {
			exifMap[tag.TagName] = s[0]
		}
	}

	if len(exifMap) == 0 {
		return nil, errors.New("no exif data found")
	}

	metas := make([]driver.MediaMeta, 0)
	takenTimeGps := time.Time{}

	// Extract GPS info
	var ifdIndex exif.IfdIndex
	_, ifdIndex, err = exif.Collect(exifIfdMapping, exifTagIndex, exifData)
	if err != nil {
		e.l.Debug("Failed to collect exif data: %s", err)
	} else {
		var ifd *exif.Ifd
		if ifd, err = ifdIndex.RootIfd.ChildWithIfdPath(exifcommon.IfdGpsInfoStandardIfdIdentity); err == nil {
			var gi *exif.GpsInfo
			if gi, err = ifd.GpsInfo(); err != nil {
				e.l.Debug("Failed to collect exif gps data: %s", err)
			} else {
				if !math.IsNaN(gi.Latitude.Decimal()) && !math.IsNaN(gi.Longitude.Decimal()) {
					lat, lng := NormalizeGPS(gi.Latitude.Decimal(), gi.Longitude.Decimal())
					metas = append(metas, driver.MediaMeta{
						Key:   GpsLat,
						Value: fmt.Sprintf("%f", lat),
					}, driver.MediaMeta{
						Key:   GpsLng,
						Value: fmt.Sprintf("%f", lng),
					})
				} else if gi.Altitude != 0 || !gi.Timestamp.IsZero() {
					e.l.Warning("GPS data is invalid: %s", gi.String())
				}

				if gi.Altitude != 0 {
					metas = append(metas, driver.MediaMeta{
						Key:   GpsAttitude,
						Value: fmt.Sprintf("%d", gi.Altitude),
					})
				}

				if !gi.Timestamp.IsZero() {
					takenTimeGps = gi.Timestamp
				}
			}
		}
	}

	metas = append(metas, ExtractExifMap(exifMap, takenTimeGps)...)
	for i := 0; i < len(metas); i++ {
		metas[i].Type = driver.MetaTypeExif
	}

	return metas, nil
}

func ExtractExifMap(exifMap map[string]string, gpsTime time.Time) []driver.MediaMeta {
	metas := make([]driver.MediaMeta, 0)
	if value, ok := exifMap["Artist"]; ok {
		metas = append(metas, driver.MediaMeta{
			Key:   Artist,
			Value: SanitizeMeta(value),
		})
	}

	if value, ok := exifMap["Copyright"]; ok {
		metas = append(metas, driver.MediaMeta{
			Key:   Copyright,
			Value: SanitizeString(value),
		})
	}

	cameraMode := ""
	if value, ok := exifMap["CameraModel"]; ok && !IsUInt(value) {
		cameraMode = SanitizeString(value)
	} else if value, ok = exifMap["Model"]; ok && !IsUInt(value) {
		cameraMode = SanitizeString(value)
	} else if value, ok = exifMap["UniqueCameraModel"]; ok && !IsUInt(value) {
		cameraMode = SanitizeString(value)
	}
	if cameraMode != "" {
		metas = append(metas, driver.MediaMeta{
			Key:   CameraModel,
			Value: cameraMode,
		})
	}

	cameraMake := ""
	if value, ok := exifMap["CameraMake"]; ok && !IsUInt(value) {
		cameraMake = SanitizeString(value)
	} else if value, ok = exifMap["Make"]; ok && !IsUInt(value) {
		cameraMake = SanitizeString(value)
	}
	if cameraMake != "" {
		metas = append(metas, driver.MediaMeta{
			Key:   CameraMake,
			Value: cameraMake,
		})
	}

	if value, ok := exifMap["CameraOwnerName"]; ok {
		metas = append(metas, driver.MediaMeta{
			Key:   CameraOwnerName,
			Value: SanitizeString(value),
		})
	}

	if value, ok := exifMap["BodySerialNumber"]; ok {
		metas = append(metas, driver.MediaMeta{
			Key:   BodySerialNumber,
			Value: SanitizeString(value),
		})
	}

	if value, ok := exifMap["LensMake"]; ok && !IsUInt(value) {
		metas = append(metas, driver.MediaMeta{
			Key:   LensMake,
			Value: SanitizeString(value),
		})
	}

	lens := ""
	if value, ok := exifMap["LensModel"]; ok && !IsUInt(value) {
		lens = SanitizeString(value)
	} else if value, ok = exifMap["Lens"]; ok && !IsUInt(value) {
		lens = SanitizeString(value)
	}
	if lens != "" {
		metas = append(metas, driver.MediaMeta{
			Key:   LensModel,
			Value: lens,
		})
	}

	if value, ok := exifMap["Software"]; ok {
		metas = append(metas, driver.MediaMeta{
			Key:   Software,
			Value: SanitizeString(value),
		})
	}

	if value, ok := exifMap["ExposureTime"]; ok {
		value = strings.TrimSuffix(value, " sec.")
		if n := strings.Split(value, "/"); len(n) == 2 {
			if n[0] != "1" && len(n[0]) < len(n[1]) {
				n0, _ := strconv.ParseUint(n[0], 10, 64)
				if n1, err := strconv.ParseUint(n[1], 10, 64); err == nil && n0 > 0 && n1 > 0 {
					value = fmt.Sprintf("1/%d", n1/n0)
				}
			}
		}

		metas = append(metas, driver.MediaMeta{
			Key:   ExposureTime,
			Value: value,
		})
	}

	if value, ok := exifMap["ExposureBiasValue"]; ok {
		if n := strings.Split(value, "/"); len(n) == 2 {
			n0, _ := strconv.ParseInt(n[0], 10, 64)
			if n1, err := strconv.ParseInt(n[1], 10, 64); err == nil {
				v := "0"
				v = fmt.Sprintf("%f", float64(n0)/float64(n1))
				metas = append(metas, driver.MediaMeta{
					Key:   ExposureBiasValue,
					Value: v,
				})
			}
		}
	}

	if value, ok := exifMap["FNumber"]; ok {
		values := strings.Split(value, "/")

		if len(values) == 2 && values[1] != "0" && values[1] != "" {
			number, _ := strconv.ParseFloat(values[0], 64)
			denom, _ := strconv.ParseFloat(values[1], 64)

			metas = append(metas, driver.MediaMeta{
				Key:   FNumber,
				Value: fmt.Sprintf("%f", float32(math.Round((number/denom)*1000)/1000)),
			})
		}
	}

	if value, ok := exifMap["ApertureValue"]; ok {
		values := strings.Split(value, "/")

		if len(values) == 2 && values[1] != "0" && values[1] != "" {
			number, _ := strconv.ParseFloat(values[0], 64)
			denom, _ := strconv.ParseFloat(values[1], 64)

			metas = append(metas, driver.MediaMeta{
				Key:   ApertureValue,
				Value: fmt.Sprintf("%f", float32(math.Round((number/denom)*1000)/1000)),
			})
		}
	}

	focalLength := ""
	if value, ok := exifMap["FocalLengthIn35mmFilm"]; ok {
		focalLength = value
	} else if v, ok := exifMap["FocalLength"]; ok {
		values := strings.Split(v, "/")

		if len(values) == 2 && values[1] != "0" && values[1] != "" {
			number, _ := strconv.ParseFloat(values[0], 64)
			denom, _ := strconv.ParseFloat(values[1], 64)

			focalLength = strconv.Itoa(int(math.Round((number/denom)*1000) / 1000))
		}
	}
	if focalLength != "" {
		metas = append(metas, driver.MediaMeta{
			Key:   FocalLength,
			Value: focalLength,
		})
	}

	if value, ok := exifMap["ISOSpeedRatings"]; ok {
		metas = append(metas, driver.MediaMeta{
			Key:   ISOSpeedRatings,
			Value: value,
		})
	}

	width := ""
	if value, ok := exifMap["PixelXDimension"]; ok {
		width = value
	} else if value, ok := exifMap["ImageWidth"]; ok {
		width = value
	}
	if width != "" {
		metas = append(metas, driver.MediaMeta{
			Key:   PixelXDimension,
			Value: width,
		})
	}

	height := ""
	if value, ok := exifMap["PixelYDimension"]; ok {
		height = value
	} else if value, ok := exifMap["ImageLength"]; ok {
		height = value
	}
	if height != "" {
		metas = append(metas, driver.MediaMeta{
			Key:   PixelYDimension,
			Value: height,
		})
	}

	orientation := "1"
	if value, ok := exifMap["Orientation"]; ok {
		orientation = value
	}
	metas = append(metas, driver.MediaMeta{
		Key:   Orientation,
		Value: orientation,
	})

	takeTime := time.Time{}
	for _, name := range exifDateTimeTags {
		if dateTime := DateTime(exifMap[name], ""); !dateTime.IsZero() {
			takeTime = dateTime
			break
		}
	}
	if takeTime.IsZero() {
		takeTime = gpsTime.UTC()
	}

	if !takeTime.IsZero() {
		metas = append(metas, driver.MediaMeta{
			Key:   TakenAt,
			Value: takeTime.Format(time.RFC3339),
		})
	}

	if value, ok := exifMap["Flash"]; ok {
		flash := "0"
		if i, err := strconv.Atoi(value); err == nil && i&1 == 1 {
			flash = "1"
		}
		metas = append(metas, driver.MediaMeta{
			Key:   Flash,
			Value: flash,
		})
	}

	if value, ok := exifMap["ImageDescription"]; ok {
		metas = append(metas, driver.MediaMeta{
			Key:   ImageDescription,
			Value: SanitizeDescription(value),
		})
	}

	if value, ok := exifMap["ProjectionType"]; ok {
		metas = append(metas, driver.MediaMeta{
			Key:   ProjectionType,
			Value: SanitizeString(value),
		})
	}

	return metas
}

type (
	exifParser interface {
		Parse(rs io.ReadSeeker, size int) (ec riimage.MediaContext, err error)
	}
)

func getExifParser(ext string) exifParser {
	switch ext {
	case "jpg", "jpeg":
		return jpegstructure.NewJpegMediaParser()
	case "png":
		return pngstructure.NewPngMediaParser()
	case "tiff":
		return tiffstructure.NewTiffMediaParser()
	case "heic", "heif", "avif":
		return heicexif.NewHeicExifMediaParser()
	default:
		return nil
	}
}

// NormalizeGPS normalizes the longitude and latitude of the GPS position to a generally valid range.
func NormalizeGPS(lat, lng float64) (float32, float32) {
	if lat < LatMax || lat > LatMax || lng < LngMax || lng > LngMax {
		// Clip the latitude. Normalise the longitude.
		lat, lng = clipLat(lat), normalizeLng(lng)
	}

	return float32(lat), float32(lng)
}

func clipLat(lat float64) float64 {
	if lat > LatMax*2 {
		return math.Mod(lat, LatMax)
	} else if lat > LatMax {
		return lat - LatMax
	}

	if lat < -LatMax*2 {
		return math.Mod(lat, LatMax)
	} else if lat < -LatMax {
		return lat + LatMax
	}

	return lat
}

func normalizeLng(value float64) float64 {
	return normalizeCoord(value, LngMax)
}

func normalizeCoord(value, max float64) float64 {
	for value < -max {
		value += 2 * max
	}
	for value >= max {
		value -= 2 * max
	}
	return value
}

// SanitizeString removes unwanted character from an exif value string.
func SanitizeString(s string) string {
	if s == "" {
		return ""
	}

	if strings.HasPrefix(s, "string with binary data") {
		return ""
	} else if strings.HasPrefix(s, "(Binary data") {
		return ""
	}

	return SanitizeUnicode(strings.Replace(s, "\"", "", -1))
}

// SanitizeUnicode returns the string as valid Unicode with whitespace trimmed.
func SanitizeUnicode(s string) string {
	if s == "" {
		return ""
	}

	return unicode(strings.TrimSpace(s))
}

// SanitizeMeta normalizes metadata fields that may contain JSON arrays like keywords and subject.
func SanitizeMeta(s string) string {
	if s == "" {
		return ""
	}

	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		var words []string

		if err := json.Unmarshal([]byte(s), &words); err != nil {
			return s
		}

		s = strings.Join(words, ", ")
	} else {
		s = SanitizeString(s)
	}

	return s
}

func unicode(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder

	for _, c := range s {
		if c == '\uFFFD' {
			continue
		}
		b.WriteRune(c)
	}

	return b.String()
}

func IsUInt(s string) bool {
	if s == "" {
		return false
	}

	for _, r := range s {
		if r < 48 || r > 57 {
			return false
		}
	}

	return true
}

// DateTime parses a time string and returns a valid time.Time if possible.
func DateTime(s, timeZone string) (t time.Time) {
	defer func() {
		if r := recover(); r != nil {
			// Panic? Return unknown time.
			t = time.Time{}
		}
	}()

	// Ignore defaults.
	if DateTimeDefault(s) {
		return time.Time{}
	}

	s = strings.TrimLeft(s, " ")

	// Timestamp too short?
	if len(s) < 4 {
		return time.Time{}
	} else if len(s) > 50 {
		// Clip to max length.
		s = s[:50]
	}

	// Pad short timestamp with whitespace at the end.
	s = fmt.Sprintf("%-19s", s)

	v := ExifDateTimeMatch
	m := ExifDateTimeRegexp.FindStringSubmatch(s)

	// Pattern doesn't match? Return unknown time.
	if len(m) == 0 {
		return time.Time{}
	}

	// Default to UTC.
	tz := time.UTC

	// Local time zone currently not supported (undefined).
	if timeZone == time.Local.String() {
		timeZone = ""
	}

	// Set time zone.
	loc := TimeZone(timeZone)

	// Location found?
	if loc != nil && timeZone != "" && tz != time.Local {
		tz = loc
		timeZone = tz.String()
	} else {
		timeZone = ""
	}

	// Does the timestamp contain a time zone offset?
	z := m[v["z"]]                     // Supported values, if not empty: Z, +, -
	zh := IntVal(m[v["zh"]], 0, 23, 0) // Hours.
	zm := IntVal(m[v["zm"]], 0, 59, 0) // Minutes.

	// Valid time zone offset found?
	if offset := (zh*60 + zm) * 60; offset > 0 && offset <= 86400 {
		// Offset timezone name example: UTC+03:30
		if z == "+" {
			// Positive offset relative to UTC.
			tz = time.FixedZone(fmt.Sprintf("UTC+%02d:%02d", zh, zm), offset)
		} else if z == "-" {
			// Negative offset relative to UTC.
			tz = time.FixedZone(fmt.Sprintf("UTC-%02d:%02d", zh, zm), -1*offset)
		}
	}

	var nsec int

	if subsec := m[v["subsec"]]; subsec != "" {
		nsec = Int(subsec + strings.Repeat("0", 9-len(subsec)))
	} else {
		nsec = 0
	}

	// Create rounded timestamp from parsed input values.
	// Year 0 is treated separately as it has a special meaning in exiftool. Golang
	// does not seem to accept value 0 for the year, but considers a date to be
	// "zero" when year is 1.
	year := IntVal(m[v["year"]], 0, YearMax, time.Now().Year())
	if year == 0 {
		year = 1
	}
	t = time.Date(
		year,
		time.Month(IntVal(m[v["month"]], 1, 12, 1)),
		IntVal(m[v["day"]], 1, 31, 1),
		IntVal(m[v["h"]], 0, 23, 0),
		IntVal(m[v["m"]], 0, 59, 0),
		IntVal(m[v["s"]], 0, 59, 0),
		nsec,
		tz)

	if timeZone != "" && loc != nil && loc != tz {
		return t.In(loc)
	}

	return t
}

// Int converts a string to a signed integer or 0 if invalid.
func Int(s string) int {
	if s == "" {
		return 0
	}

	result, err := strconv.ParseInt(strings.TrimSpace(s), 10, 32)

	if err != nil {
		return 0
	}

	return int(result)
}

// IntVal converts a string to a validated integer or a default if invalid.
func IntVal(s string, min, max, def int) (i int) {
	if s == "" {
		return def
	} else if s[0] == ' ' {
		s = strings.TrimSpace(s)
	}

	result, err := strconv.ParseInt(s, 10, 32)

	if err != nil {
		return def
	}

	i = int(result)

	if i < min {
		return def
	} else if max != 0 && i > max {
		return def
	}

	return i
}

// DateTimeDefault tests if the datetime string is not empty and not a default value.
func DateTimeDefault(s string) bool {
	switch s {
	case "1970-01-01", "1970-01-01 00:00:00", "1970:01:01 00:00:00":
		// Unix epoch.
		return true
	case "1980-01-01", "1980-01-01 00:00:00", "1980:01:01 00:00:00":
		// Windows default.
		return true
	case "2002-12-08 12:00:00", "2002:12:08 12:00:00":
		// Android Bug: https://issuetracker.google.com/issues/36967504
		return true
	default:
		return EmptyDateTime(s)
	}
}

// EmptyDateTime tests if the string is empty or matches an unknown time pattern.
func EmptyDateTime(s string) bool {
	switch s {
	case "", "-", ":", "z", "Z", "nil", "null", "none", "nan", "NaN":
		return true
	case "0", "00", "0000", "0000:00:00", "00:00:00", "0000-00-00", "00-00-00":
		return true
	case "    :  :     :  :  ", "    -  -     -  -  ", "    -  -     :  :  ":
		// Exif default.
		return true
	case "0000:00:00 00:00:00", "0000-00-00 00-00-00", "0000-00-00 00:00:00":
		return true
	case "0001:01:01 00:00:00", "0001-01-01 00-00-00", "0001-01-01 00:00:00":
		// Go default.
		return true
	case "0001:01:01 00:00:00 +0000 UTC", "0001-01-01 00-00-00 +0000 UTC", "0001-01-01 00:00:00 +0000 UTC":
		// Go default with time zone.
		return true
	default:
		return false
	}
}

// TimeZone returns a time zone for the given UTC offset string.
func TimeZone(offset string) *time.Location {
	if offset == "" {
		// Local time.
	} else if offset == "UTC" || offset == "Z" {
		return time.UTC
	} else if seconds, err := TimeOffset(offset); err == nil {
		if h := seconds / 3600; h > 0 || h < 0 {
			return time.FixedZone(fmt.Sprintf("UTC%+d", h), seconds)
		}
	} else if zone, zoneErr := time.LoadLocation(offset); zoneErr == nil {
		return zone
	}

	return time.FixedZone("", 0)
}

// TimeOffset returns the UTC time offset in seconds or an error if it is invalid.
func TimeOffset(utcOffset string) (seconds int, err error) {
	switch utcOffset {
	case "-12", "-12:00", "UTC-12", "UTC-12:00":
		seconds = -12 * 3600
	case "-11", "-11:00", "UTC-11", "UTC-11:00":
		seconds = -11 * 3600
	case "-10", "-10:00", "UTC-10", "UTC-10:00":
		seconds = -10 * 3600
	case "-9", "-09", "-09:00", "UTC-9", "UTC-09:00":
		seconds = -9 * 3600
	case "-8", "-08", "-08:00", "UTC-8", "UTC-08:00":
		seconds = -8 * 3600
	case "-7", "-07", "-07:00", "UTC-7", "UTC-07:00":
		seconds = -7 * 3600
	case "-6", "-06", "-06:00", "UTC-6", "UTC-06:00":
		seconds = -6 * 3600
	case "-5", "-05", "-05:00", "UTC-5", "UTC-05:00":
		seconds = -5 * 3600
	case "-4", "-04", "-04:00", "UTC-4", "UTC-04:00":
		seconds = -4 * 3600
	case "-3", "-03", "-03:00", "UTC-3", "UTC-03:00":
		seconds = -3 * 3600
	case "-2", "-02", "-02:00", "UTC-2", "UTC-02:00":
		seconds = -2 * 3600
	case "-1", "-01", "-01:00", "UTC-1", "UTC-01:00":
		seconds = -1 * 3600
	case "01:00", "+1", "+01", "+01:00", "UTC+1", "UTC+01:00":
		seconds = 1 * 3600
	case "02:00", "+2", "+02", "+02:00", "UTC+2", "UTC+02:00":
		seconds = 2 * 3600
	case "03:00", "+3", "+03", "+03:00", "UTC+3", "UTC+03:00":
		seconds = 3 * 3600
	case "04:00", "+4", "+04", "+04:00", "UTC+4", "UTC+04:00":
		seconds = 4 * 3600
	case "05:00", "+5", "+05", "+05:00", "UTC+5", "UTC+05:00":
		seconds = 5 * 3600
	case "06:00", "+6", "+06", "+06:00", "UTC+6", "UTC+06:00":
		seconds = 6 * 3600
	case "07:00", "+7", "+07", "+07:00", "UTC+7", "UTC+07:00":
		seconds = 7 * 3600
	case "08:00", "+8", "+08", "+08:00", "UTC+8", "UTC+08:00":
		seconds = 8 * 3600
	case "09:00", "+9", "+09", "+09:00", "UTC+9", "UTC+09:00":
		seconds = 9 * 3600
	case "10:00", "+10", "+10:00", "UTC+10", "UTC+10:00":
		seconds = 10 * 3600
	case "11:00", "+11", "+11:00", "UTC+11", "UTC+11:00":
		seconds = 11 * 3600
	case "12:00", "+12", "+12:00", "UTC+12", "UTC+12:00":
		seconds = 12 * 3600
	case "Z", "UTC", "UTC+0", "UTC-0", "UTC+00:00", "UTC-00:00":
		seconds = 0
	default:
		return 0, fmt.Errorf("invalid UTC offset")
	}

	return seconds, nil
}

func SanitizeDescription(s string) string {
	s = SanitizeString(s)

	switch {
	case s == "":
		return ""
	case UnwantedDescriptions[s]:
		return ""
	case strings.HasPrefix(s, "DCIM\\") && !strings.Contains(s, " "):
		return ""
	default:
		return s
	}
}
