package util

import (
	"context"
	cryptoRand "crypto/rand"
	"math/big"
	"math/rand"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var (
	RandomVariantAll = []rune("1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	RandomLowerCases = []rune("1234567890abcdefghijklmnopqrstuvwxyz")
)

// RandStringRunes 返回随机字符串
func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = RandomVariantAll[rand.Intn(len(RandomVariantAll))]
	}
	return string(b)
}

func RandStringRunesCrypto(n int) string {
	return randStringCrypto(n, RandomVariantAll)
}

// RandString returns random string in given length and variant
func RandString(n int, variant []rune) string {
	return randStringCrypto(n, variant)
}

func randStringCrypto(n int, variant []rune) string {
	b := make([]rune, n)
	for i := range b {
		num, err := cryptoRand.Int(cryptoRand.Reader, big.NewInt(int64(len(variant))))
		if err != nil {
			// fallback to math/rand on crypto failure
			b[i] = variant[rand.Intn(len(variant))]
		} else {
			b[i] = variant[num.Int64()]
		}
	}
	return string(b)
}

// ContainsUint 返回list中是否包含
func ContainsUint(s []uint, e uint) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// IsInExtensionList 返回文件的扩展名是否在给定的列表范围内
func IsInExtensionList(extList []string, fileName string) bool {
	ext := Ext(fileName)
	// 无扩展名时
	if len(ext) == 0 {
		return false
	}

	if ContainsString(extList, ext) {
		return true
	}

	return false
}

// IsInExtensionList 返回文件的扩展名是否在给定的列表范围内
func IsExtInList(extList []string, ext string) bool {
	// 无扩展名时
	if len(ext) == 0 {
		return false
	}

	if ContainsString(extList, ext) {
		return true
	}

	return false
}

// ContainsString 返回list中是否包含
func ContainsString(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// Replace 根据替换表执行批量替换
func Replace(table map[string]string, s string) string {
	for key, value := range table {
		s = strings.Replace(s, key, value, -1)
	}
	return s
}

// ReplaceMagicVar 动态替换字符串中的魔法变量
func ReplaceMagicVar(rawString string, fsSeparator string, pathAvailable bool, blobAvailable bool,
	timeConst time.Time, userId int, originName string, originPath string, completeBlobPath string) string {
	re := regexp.MustCompile(`\{[^{}]+\}`)
	return re.ReplaceAllStringFunc(rawString, func(match string) string {
		switch match {
		case "{randomkey16}":
			return RandStringRunes(16)
		case "{randomkey8}":
			return RandStringRunes(8)
		case "{timestamp}":
			return strconv.FormatInt(timeConst.Unix(), 10)
		case "{timestamp_nano}":
			return strconv.FormatInt(timeConst.UnixNano(), 10)
		case "{randomnum2}":
			return strconv.Itoa(rand.Intn(2))
		case "{randomnum3}":
			return strconv.Itoa(rand.Intn(3))
		case "{randomnum4}":
			return strconv.Itoa(rand.Intn(4))
		case "{randomnum8}":
			return strconv.Itoa(rand.Intn(8))
		case "{uid}":
			return strconv.Itoa(userId)
		case "{datetime}":
			return timeConst.Format("20060102150405")
		case "{date}":
			return timeConst.Format("20060102")
		case "{year}":
			return timeConst.Format("2006")
		case "{month}":
			return timeConst.Format("01")
		case "{day}":
			return timeConst.Format("02")
		case "{hour}":
			return timeConst.Format("15")
		case "{minute}":
			return timeConst.Format("04")
		case "{second}":
			return timeConst.Format("05")
		case "{uuid}":
			return uuid.Must(uuid.NewV4()).String()
		case "{ext}":
			return filepath.Ext(originName)
		case "{originname}":
			return originName
		case "{originname_without_ext}":
			return strings.TrimSuffix(originName, filepath.Ext(originName))
		case "{path}":
			if pathAvailable {
				return originPath + fsSeparator
			}
			return match
		case "{blob_name}":
			if blobAvailable {
				return filepath.Base(completeBlobPath)
			}
			return match
		case "{blob_name_without_ext}":
			if blobAvailable {
				return strings.TrimSuffix(filepath.Base(completeBlobPath), filepath.Ext(completeBlobPath))
			}
			return match
		case "{blob_path}":
			if blobAvailable {
				return path.Dir(completeBlobPath) + fsSeparator
			}
			return match
		default:
			return match
		}
	})
}

// BuildRegexp 构建用于SQL查询用的多条件正则
func BuildRegexp(search []string, prefix, suffix, condition string) string {
	var res string
	for key, value := range search {
		res += prefix + regexp.QuoteMeta(value) + suffix
		if key < len(search)-1 {
			res += condition
		}
	}
	return res
}

// BuildConcat 根据数据库类型构建字符串连接表达式
func BuildConcat(str1, str2 string, DBType string) string {
	switch DBType {
	case "mysql":
		return "CONCAT(" + str1 + "," + str2 + ")"
	default:
		return str1 + "||" + str2
	}
}

// SliceIntersect 求两个切片交集
func SliceIntersect(slice1, slice2 []string) []string {
	m := make(map[string]int)
	nn := make([]string, 0)
	for _, v := range slice1 {
		m[v]++
	}

	for _, v := range slice2 {
		times, _ := m[v]
		if times == 1 {
			nn = append(nn, v)
		}
	}
	return nn
}

// SliceDifference 求两个切片差集
func SliceDifference(slice1, slice2 []string) []string {
	m := make(map[string]int)
	nn := make([]string, 0)
	inter := SliceIntersect(slice1, slice2)
	for _, v := range inter {
		m[v]++
	}

	for _, value := range slice1 {
		times, _ := m[value]
		if times == 0 {
			nn = append(nn, value)
		}
	}
	return nn
}

// WithValue inject key-value pair into request context.
func WithValue(c *gin.Context, key any, value any) {
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), key, value))
}

// BoolToString transform bool to string
func BoolToString(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func ToPtr[T any](v T) *T {
	return &v
}
