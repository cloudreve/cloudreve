package oss

import (
	"bytes"
	"crypto"
	"crypto/md5"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
)

const (
	pubKeyHeader      = "x-oss-pub-key-url"
	pubKeyPrefix      = "http://gosspublic.alicdn.com/"
	pubKeyPrefixHttps = "https://gosspublic.alicdn.com/"
	pubKeyCacheKey    = "oss_public_key"
)

// GetPublicKey 从回调请求或缓存中获取OSS的回调签名公钥
func GetPublicKey(r *http.Request, kv cache.Driver, client request.Client) ([]byte, error) {
	var pubKey []byte

	// 尝试从缓存中获取
	pub, exist := kv.Get(pubKeyCacheKey)
	if exist {
		return pub.([]byte), nil
	}

	// 从请求中获取
	pubURL, err := base64.StdEncoding.DecodeString(r.Header.Get(pubKeyHeader))
	if err != nil {
		return pubKey, err
	}

	// 确保这个 public key 是由 OSS 颁发的
	if !strings.HasPrefix(string(pubURL), pubKeyPrefix) &&
		!strings.HasPrefix(string(pubURL), pubKeyPrefixHttps) {
		return pubKey, errors.New("public key url invalid")
	}

	// 获取公钥
	body, err := client.Request("GET", string(pubURL), nil).
		CheckHTTPResponse(200).
		GetResponse()
	if err != nil {
		return pubKey, err
	}

	// 写入缓存
	_ = kv.Set(pubKeyCacheKey, []byte(body), 86400*7)

	return []byte(body), nil
}

func getRequestMD5(r *http.Request) ([]byte, error) {
	var byteMD5 []byte

	// 获取请求正文
	body, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		return byteMD5, err
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	strURLPathDecode, err := url.PathUnescape(r.URL.Path)
	if err != nil {
		return byteMD5, err
	}

	strAuth := fmt.Sprintf("%s\n%s", strURLPathDecode, string(body))
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(strAuth))
	byteMD5 = md5Ctx.Sum(nil)

	return byteMD5, nil
}

// VerifyCallbackSignature 验证OSS回调请求
func VerifyCallbackSignature(r *http.Request, kv cache.Driver, client request.Client) error {
	bytePublicKey, err := GetPublicKey(r, kv, client)
	if err != nil {
		return err
	}

	byteMD5, err := getRequestMD5(r)
	if err != nil {
		return err
	}

	strAuthorizationBase64 := r.Header.Get("authorization")
	if strAuthorizationBase64 == "" {
		return errors.New("no authorization field in Request header")
	}
	authorization, _ := base64.StdEncoding.DecodeString(strAuthorizationBase64)

	pubBlock, _ := pem.Decode(bytePublicKey)
	if pubBlock == nil {
		return errors.New("pubBlock not exist")
	}
	pubInterface, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if (pubInterface == nil) || (err != nil) {
		return err
	}
	pub := pubInterface.(*rsa.PublicKey)

	errorVerifyPKCS1v15 := rsa.VerifyPKCS1v15(pub, crypto.MD5, byteMD5, authorization)
	if errorVerifyPKCS1v15 != nil {
		return errorVerifyPKCS1v15
	}

	return nil
}
