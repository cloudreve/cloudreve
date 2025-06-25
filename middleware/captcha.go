package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/recaptcha"
	request2 "github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/gin-gonic/gin"
	"github.com/mojocn/base64Captcha"
)

type req struct {
	Captcha string `json:"captcha"`
	Ticket  string `json:"ticket"`
	Randstr string `json:"randstr"`
}

const (
	captchaNotMatch = "CAPTCHA not match."
	captchaRefresh  = "Verification failed, please refresh the page and retry."

	tcCaptchaEndpoint = "captcha.tencentcloudapi.com"
	turnstileEndpoint = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
)

// CaptchaIDCtx defines keys for captcha ID
type (
	CaptchaIDCtx      struct{}
	turnstileResponse struct {
		Success bool `json:"success"`
	}
	capResponse struct {
		Success bool `json:"success"`
	}
)

// CaptchaRequired 验证请求签名
func CaptchaRequired(enabled func(c *gin.Context) bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if enabled(c) {
			dep := dependency.FromContext(c)
			settings := dep.SettingProvider()
			l := logging.FromContext(c)

			var service req
			bodyCopy := new(bytes.Buffer)
			_, err := io.Copy(bodyCopy, c.Request.Body)
			if err != nil {
				c.JSON(200, serializer.ErrWithDetails(c, serializer.CodeCaptchaError, captchaNotMatch, err))
				c.Abort()
				return
			}

			bodyData := bodyCopy.Bytes()
			err = json.Unmarshal(bodyData, &service)
			if err != nil {
				c.JSON(200, serializer.ErrWithDetails(c, serializer.CodeCaptchaError, captchaNotMatch, err))
				c.Abort()
				return
			}

			c.Request.Body = io.NopCloser(bytes.NewReader(bodyData))
			switch settings.CaptchaType(c) {
			case setting.CaptchaNormal, setting.CaptchaTcaptcha:
				if service.Ticket == "" || !base64Captcha.VerifyCaptcha(service.Ticket, service.Captcha) {
					c.JSON(200, serializer.ErrWithDetails(c, serializer.CodeCaptchaError, captchaNotMatch, err))
					c.Abort()
					return
				}

				break
			case setting.CaptchaReCaptcha:
				captchaSetting := settings.ReCaptcha(c)
				reCAPTCHA, err := recaptcha.NewReCAPTCHA(captchaSetting.Secret, recaptcha.V2, 10*time.Second)
				if err != nil {
					l.Warning("reCAPTCHA verification failed, %s", err)
					c.Abort()
					break
				}

				err = reCAPTCHA.Verify(service.Captcha)
				if err != nil {
					l.Warning("reCAPTCHA verification failed, %s", err)
					c.JSON(200, serializer.ErrWithDetails(c, serializer.CodeCaptchaError, captchaRefresh, err))
					c.Abort()
					return
				}

				break
			case setting.CaptchaTurnstile:
				captchaSetting := settings.TurnstileCaptcha(c)
				r := dep.RequestClient(
					request2.WithContext(c),
					request2.WithLogger(logging.FromContext(c)),
					request2.WithHeader(http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}}),
				)
				formData := url.Values{}
				formData.Set("secret", captchaSetting.Secret)
				formData.Set("response", service.Ticket)
				res, err := r.Request("POST", turnstileEndpoint, strings.NewReader(formData.Encode())).
					CheckHTTPResponse(http.StatusOK).
					GetResponse()
				if err != nil {
					c.JSON(200, serializer.ErrWithDetails(c, serializer.CodeCaptchaError, "Captcha validation failed", err))
					c.Abort()
					return
				}

				var trunstileRes turnstileResponse
				err = json.Unmarshal([]byte(res), &trunstileRes)
				if err != nil {
					l.Warning("Turnstile verification failed, %s", err)
					c.JSON(200, serializer.ErrWithDetails(c, serializer.CodeCaptchaError, "Captcha validation failed", err))
					c.Abort()
					return
				}

				if !trunstileRes.Success {
					c.JSON(200, serializer.ErrWithDetails(c, serializer.CodeCaptchaError, "Captcha validation failed", err))
					c.Abort()
					return
				}

				break
			case setting.CaptchaCap:
				captchaSetting := settings.CapCaptcha(c)
				if captchaSetting.InstanceURL == "" || captchaSetting.SiteKey == "" || captchaSetting.SecretKey == "" {
					l.Warning("Cap verification failed: missing configuration")
					c.JSON(200, serializer.ErrWithDetails(c, serializer.CodeCaptchaError, "Captcha configuration error", nil))
					c.Abort()
					return
				}

				r := dep.RequestClient(
					request2.WithContext(c),
					request2.WithLogger(logging.FromContext(c)),
					request2.WithHeader(http.Header{"Content-Type": []string{"application/json"}}),
				)

				capEndpoint := strings.TrimSuffix(captchaSetting.InstanceURL, "/") + "/" + captchaSetting.SiteKey + "/siteverify"
				requestBody := map[string]string{
					"secret":   captchaSetting.SecretKey,
					"response": service.Ticket,
				}
				requestData, err := json.Marshal(requestBody)
				if err != nil {
					l.Warning("Cap verification failed: %s", err)
					c.JSON(200, serializer.ErrWithDetails(c, serializer.CodeCaptchaError, "Captcha validation failed", err))
					c.Abort()
					return
				}

				res, err := r.Request("POST", capEndpoint, strings.NewReader(string(requestData))).
					CheckHTTPResponse(http.StatusOK).
					GetResponse()
				if err != nil {
					l.Warning("Cap verification failed: %s", err)
					c.JSON(200, serializer.ErrWithDetails(c, serializer.CodeCaptchaError, "Captcha validation failed", err))
					c.Abort()
					return
				}

				var capRes capResponse
				err = json.Unmarshal([]byte(res), &capRes)
				if err != nil {
					l.Warning("Cap verification failed: %s", err)
					c.JSON(200, serializer.ErrWithDetails(c, serializer.CodeCaptchaError, "Captcha validation failed", err))
					c.Abort()
					return
				}

				if !capRes.Success {
					l.Warning("Cap verification failed: validation returned false")
					c.JSON(200, serializer.ErrWithDetails(c, serializer.CodeCaptchaError, "Captcha validation failed", nil))
					c.Abort()
					return
				}

				break
			}
		}
		c.Next()
	}
}
