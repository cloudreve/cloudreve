package controllers

import (
	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/service/basic"
	"github.com/gin-gonic/gin"
)

// SiteConfig 获取站点全局配置
func SiteConfig(c *gin.Context) {
	service := ParametersFromContext[*basic.GetSettingService](c, basic.GetSettingParamCtx{})

	resp, err := service.GetSiteConfig(c)
	if respondErr(c, err) {
		return
	}

	c.JSON(200, serializer.Response{
		Data: resp,
	})
}

// Ping 状态检查页面
func Ping(c *gin.Context) {
	version := constants.BackendVersion
	if constants.IsProBool {
		version += "-pro"
	}

	c.JSON(200, serializer.Response{
		Code: 0,
		Data: version,
	})
}

// Captcha 获取验证码
func Captcha(c *gin.Context) {
	c.JSON(200, serializer.Response{
		Code: 0,
		Data: basic.GetCaptchaImage(c),
	})
}

// Manifest 获取manifest.json
func Manifest(c *gin.Context) {
	settingClient := dependency.FromContext(c).SettingProvider()
	siteOpts := settingClient.SiteBasic(c)
	pwaOpts := settingClient.PWA(c)
	c.Header("Cache-Control", "public, no-cache")
	c.JSON(200, map[string]interface{}{
		"short_name": siteOpts.Name,
		"name":       siteOpts.Name,
		"icons": []map[string]string{
			{
				"src":   pwaOpts.SmallIcon,
				"sizes": "64x64 32x32 24x24 16x16",
				"type":  "image/x-icon",
			},
			{
				"src":   pwaOpts.MediumIcon,
				"type":  "image/png",
				"sizes": "192x192",
			},
			{
				"src":   pwaOpts.LargeIcon,
				"type":  "image/png",
				"sizes": "512x512",
			},
		},
		"start_url":        ".",
		"display":          pwaOpts.Display,
		"theme_color":      pwaOpts.ThemeColor,
		"background_color": pwaOpts.BackgroundColor,
	})
}
