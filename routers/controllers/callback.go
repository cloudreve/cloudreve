package controllers

import (
	"fmt"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver/upyun"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/cloudreve/Cloudreve/v4/service/callback"
	"github.com/gin-gonic/gin"
	"github.com/qiniu/go-sdk/v7/auth/qbox"
)

// RemoteCallback process callback request to complete upload
func ProcessCallback(failedStatusCode int, generalResp bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := callback.ProcessCallback(c)
		if err != nil {
			if generalResp {
				c.JSON(failedStatusCode, serializer.GeneralUploadCallbackFailed{Error: err.Error()})
			} else {
				c.JSON(failedStatusCode, serializer.Err(c, err))
			}
			return
		}

		c.JSON(200, serializer.Response{})
	}
}

// QiniuCallbackAuth 七牛回调签名验证
func QiniuCallbackValidate(c *gin.Context) {
	session := c.MustGet(manager.UploadSessionCtx).(*fs.UploadSession)

	// 验证回调是否来自qiniu
	mac := qbox.NewMac(session.Policy.AccessKey, session.Policy.SecretKey)
	ok, err := mac.VerifyCallback(c.Request)
	if err != nil {
		util.Log().Debug("Failed to verify callback request: %s", err)
		c.JSON(401, serializer.GeneralUploadCallbackFailed{Error: "Failed to verify callback request."})
		c.Abort()
		return
	}

	if !ok {
		c.JSON(401, serializer.GeneralUploadCallbackFailed{Error: "Invalid signature."})
		c.Abort()
		return
	}

	c.Next()
}

// OSSCallbackValidate 阿里云OSS上传回调
func OSSCallbackValidate(c *gin.Context) {
	var callbackBody callback.UploadCallbackService
	if err := c.ShouldBindJSON(&callbackBody); err == nil {
		uploadSession := c.MustGet(manager.UploadSessionCtx).(*fs.UploadSession)
		if uploadSession.Props.Size != callbackBody.Size {
			l := logging.FromContext(c)
			l.Error("Callback validate failed: size mismatch, expected: %d, actual:%d", uploadSession.Props.Size, callbackBody.Size)
			c.JSON(401,
				serializer.GeneralUploadCallbackFailed{
					Error: fmt.Sprintf("size mismatch"),
				})
			c.Abort()
			return
		}

		c.Next()
	} else {
		c.JSON(401, ErrorResponse(c, err))
		c.Abort()
	}
}

// UpyunCallbackAuth 又拍云回调签名验证
func UpyunCallbackAuth(c *gin.Context) {
	uploadSession := c.MustGet(manager.UploadSessionCtx).(*fs.UploadSession)
	l := logging.FromContext(c)
	if err := upyun.ValidateCallback(c, uploadSession); err != nil {
		l.Error("Failed to verify callback request: %s", err)

		c.AbortWithStatusJSON(401, serializer.GeneralUploadCallbackFailed{Error: "Failed to verify callback request."})
		return
	}

	c.Next()
}

// OneDriveOAuth OneDrive 授权回调
func OneDriveOAuth(c *gin.Context) {
	//var callbackBody callback.OauthService
	//if err := c.ShouldBindQuery(&callbackBody); err == nil {
	//	res := callbackBody.OdAuth(c)
	//	redirect := model.GetSiteURL()
	//	redirect.Path = path.Join(redirect.Path, "/admin/policy")
	//	queries := redirect.Query()
	//	queries.Add("code", strconv.Itoa(res.Code))
	//	queries.Add("msg", res.Msg)
	//	queries.Add("err", res.Error)
	//	redirect.RawQuery = queries.Encode()
	//	c.Redirect(303, redirect.String())
	//} else {
	//	c.JSON(200, ErrorResponse(c, err))
	//}
}

// GoogleDriveOAuth Google Drive 授权回调
func GoogleDriveOAuth(c *gin.Context) {
	//var callbackBody callback.OauthService
	//if err := c.ShouldBindQuery(&callbackBody); err == nil {
	//	res := callbackBody.GDriveAuth(c)
	//	redirect := model.GetSiteURL()
	//	redirect.Path = path.Join(redirect.Path, "/admin/policy")
	//	queries := redirect.Query()
	//	queries.Add("code", strconv.Itoa(res.Code))
	//	queries.Add("msg", res.Msg)
	//	queries.Add("err", res.Error)
	//	redirect.RawQuery = queries.Encode()
	//	c.Redirect(303, redirect.String())
	//} else {
	//	c.JSON(200, ErrorResponse(c, err))
	//}
}
