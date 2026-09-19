package fs

import (
	"math/rand"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gofrs/uuid"
)

// MagicVarProps carries the context available to magic-variable expansion in
// path templates.
type MagicVarProps struct {
	FsSeparator      string
	PathAvailable    bool
	BlobAvailable    bool
	Time             time.Time
	UserID           int
	OriginName       string
	OriginPath       string
	CompleteBlobPath string
}

var magicVarRe = regexp.MustCompile(`\{[^{}]+\}`)

// ReplaceMagicVar expands {var} placeholders in rawString using p.
func ReplaceMagicVar(rawString string, p MagicVarProps) string {
	return magicVarRe.ReplaceAllStringFunc(rawString, func(match string) string {
		switch match {
		case "{randomkey16}":
			return util.RandStringRunes(16)
		case "{randomkey8}":
			return util.RandStringRunes(8)
		case "{timestamp}":
			return strconv.FormatInt(p.Time.Unix(), 10)
		case "{timestamp_nano}":
			return strconv.FormatInt(p.Time.UnixNano(), 10)
		case "{randomnum2}":
			return strconv.Itoa(rand.Intn(2))
		case "{randomnum3}":
			return strconv.Itoa(rand.Intn(3))
		case "{randomnum4}":
			return strconv.Itoa(rand.Intn(4))
		case "{randomnum8}":
			return strconv.Itoa(rand.Intn(8))
		case "{uid}":
			return strconv.Itoa(p.UserID)
		case "{datetime}":
			return p.Time.Format("20060102150405")
		case "{date}":
			return p.Time.Format("20060102")
		case "{year}":
			return p.Time.Format("2006")
		case "{month}":
			return p.Time.Format("01")
		case "{day}":
			return p.Time.Format("02")
		case "{hour}":
			return p.Time.Format("15")
		case "{minute}":
			return p.Time.Format("04")
		case "{second}":
			return p.Time.Format("05")
		case "{uuid}":
			return uuid.Must(uuid.NewV4()).String()
		case "{ext}":
			return filepath.Ext(p.OriginName)
		case "{originname}":
			return p.OriginName
		case "{originname_without_ext}":
			return strings.TrimSuffix(p.OriginName, filepath.Ext(p.OriginName))
		case "{path}":
			if p.PathAvailable {
				return p.OriginPath + p.FsSeparator
			}
			return match
		case "{blob_name}":
			if p.BlobAvailable {
				return filepath.Base(p.CompleteBlobPath)
			}
			return match
		case "{blob_name_without_ext}":
			if p.BlobAvailable {
				return strings.TrimSuffix(filepath.Base(p.CompleteBlobPath), filepath.Ext(p.CompleteBlobPath))
			}
			return match
		case "{blob_path}":
			if p.BlobAvailable {
				return path.Dir(p.CompleteBlobPath) + p.FsSeparator
			}
			return match
		default:
			return match
		}
	})
}
