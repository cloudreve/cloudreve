// Package tcaptcha verifies Tencent Cloud Captcha tickets via the
// DescribeCaptchaResult API (TencentCloud API v3, TC3-HMAC-SHA256 signing).
package tcaptcha

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
)

const (
	endpoint       = "captcha.tencentcloudapi.com"
	service        = "captcha"
	version        = "2019-07-22"
	action         = "DescribeCaptchaResult"
	signAlgorithm  = "TC3-HMAC-SHA256"
	signedHeaders  = "content-type;host;x-tc-action"
	captchaTypeNew = 9 // interactive captcha (popup widget)
)

type describeResultResponse struct {
	Response struct {
		CaptchaCode int    `json:"CaptchaCode"`
		CaptchaMsg  string `json:"CaptchaMsg"`
		Error       *struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error"`
	} `json:"Response"`
}

// Verify checks a ticket produced by the Tencent captcha widget. CaptchaCode
// 1 means the ticket is valid.
func Verify(ctx context.Context, client request.Client, cfg *setting.TcCaptcha, ticket, randstr, userIP string) (bool, error) {
	appID, err := strconv.ParseInt(strings.TrimSpace(cfg.AppID), 10, 64)
	if err != nil {
		return false, fmt.Errorf("invalid CaptchaAppId: %w", err)
	}

	payload, err := json.Marshal(map[string]any{
		"CaptchaType":  captchaTypeNew,
		"Ticket":       ticket,
		"Randstr":      randstr,
		"UserIp":       userIP,
		"CaptchaAppId": appID,
		"AppSecretKey": cfg.AppSecretKey,
	})
	if err != nil {
		return false, err
	}

	timestamp := time.Now().Unix()
	auth := sign(cfg.SecretID, cfg.SecretKey, string(payload), timestamp)

	res, err := client.Request(
		"POST",
		"https://"+endpoint,
		strings.NewReader(string(payload)),
		request.WithContext(ctx),
		request.WithHeader(http.Header{
			"Content-Type":   []string{"application/json; charset=utf-8"},
			"Host":           []string{endpoint},
			"X-TC-Action":    []string{action},
			"X-TC-Version":   []string{version},
			"X-TC-Timestamp": []string{strconv.FormatInt(timestamp, 10)},
			"Authorization":  []string{auth},
		}),
	).CheckHTTPResponse(http.StatusOK).GetResponse()
	if err != nil {
		return false, err
	}

	var parsed describeResultResponse
	if err := json.Unmarshal([]byte(res), &parsed); err != nil {
		return false, fmt.Errorf("failed to parse captcha result: %w", err)
	}
	if parsed.Response.Error != nil {
		return false, fmt.Errorf("captcha api error %s: %s", parsed.Response.Error.Code, parsed.Response.Error.Message)
	}
	return parsed.Response.CaptchaCode == 1, nil
}

// sign builds the TC3-HMAC-SHA256 Authorization header value.
func sign(secretID, secretKey, payload string, timestamp int64) string {
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")

	canonicalRequest := strings.Join([]string{
		"POST",
		"/",
		"",
		"content-type:application/json; charset=utf-8\n" +
			"host:" + endpoint + "\n" +
			"x-tc-action:" + strings.ToLower(action) + "\n",
		signedHeaders,
		sha256Hex(payload),
	}, "\n")

	credentialScope := date + "/" + service + "/tc3_request"
	stringToSign := strings.Join([]string{
		signAlgorithm,
		strconv.FormatInt(timestamp, 10),
		credentialScope,
		sha256Hex(canonicalRequest),
	}, "\n")

	signingKey := hmacSHA256(
		hmacSHA256(
			hmacSHA256([]byte("TC3"+secretKey), date),
			service,
		),
		"tc3_request",
	)
	signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))

	return fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		signAlgorithm, secretID, credentialScope, signedHeaders, signature)
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}
