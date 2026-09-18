package controllers

import (
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/service/oauth"
	"github.com/gin-gonic/gin"
)

func OpenIDConfiguration(c *gin.Context) {
	service := &oauth.DiscoveryService{}
	c.JSON(200, service.Get(c))
}

func OpenIDJWKS(c *gin.Context) {
	service := &oauth.JWKService{}
	res, err := service.Get(c)
	if err != nil {
		c.JSON(500, serializer.Err(c, err))
		c.Abort()
		return
	}

	c.JSON(200, res)
}

func GetAppRegistration(c *gin.Context) {
	service := ParametersFromContext[*oauth.GetAppRegistrationService](c, oauth.GetAppRegistrationParamCtx{})
	app, err := service.Get(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	c.JSON(200, serializer.Response{Data: app})
}

func GrantAppConsent(c *gin.Context) {
	service := ParametersFromContext[*oauth.GrantService](c, oauth.GrantParamCtx{})
	res, err := service.Get(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

type ExchangeErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	ErrorCodes       []int  `json:"error_codes"`
	CorrelationID    string `json:"correlation_id"`
}

func ExchangeToken(c *gin.Context) {
	service := ParametersFromContext[*oauth.ExchangeTokenService](c, oauth.ExchangeTokenParamCtx{})
	res, err := service.Exchange(c)
	if err != nil {
		errResp := serializer.Err(c, err)
		c.JSON(400, ExchangeErrorResponse{
			Error:            errResp.Msg,
			ErrorDescription: errResp.Error,
			ErrorCodes:       []int{errResp.Code},
			CorrelationID:    errResp.CorrelationID,
		})
		c.Abort()
		return
	}

	c.JSON(200, res)
}

func OpenIDUserInfo(c *gin.Context) {
	service := ParametersFromContext[*oauth.UserInfoService](c, oauth.UserInfoParamCtx{})
	res, err := service.GetUserInfo(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	c.JSON(200, res)
}

func DeleteOAuthGrant(c *gin.Context) {
	service := ParametersFromContext[*oauth.DeleteOAuthGrantService](c, oauth.DeleteOAuthGrantParamCtx{})
	err := service.Delete(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	c.JSON(200, serializer.Response{})
}
