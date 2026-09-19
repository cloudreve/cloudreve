package controllers

import (
	"context"
	"encoding/json"

	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// respondErr writes the standard serialized error body and aborts the chain
// when err is non-nil. Handlers use it as `if respondErr(c, err) { return }`.
func respondErr(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	c.JSON(200, serializer.Err(c, err))
	c.Abort()
	return true
}

// respond writes err as an error body, or ok as the success body.
func respond(c *gin.Context, err error, ok any) {
	if respondErr(c, err) {
		return
	}
	c.JSON(200, ok)
}

// ParamErrorMsg 根据Validator返回的错误信息给出错误提示
func ParamErrorMsg(filed string, tag string) string {
	// 未通过验证的表单域与中文对应
	fieldMap := map[string]string{
		"UserName": "Email",
		"Password": "Password",
		"Path":     "Path",
		"SourceID": "Source resource",
		"URL":      "URL",
		"Nick":     "Nickname",
	}
	// 未通过的规则与中文对应
	tagMap := map[string]string{
		"required": "cannot be empty",
		"min":      "too short",
		"max":      "too long",
		"email":    "format error",
	}
	fieldVal, findField := fieldMap[filed]
	if !findField {
		fieldVal = filed
	}
	tagVal, findTag := tagMap[tag]
	if findTag {
		// 返回拼接出来的错误信息
		return fieldVal + " " + tagVal
	}
	return ""
}

// ErrorResponse 返回错误消息
func ErrorResponse(c *gin.Context, err error) serializer.Response {
	// 处理 Validator 产生的错误
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, e := range ve {
			return serializer.ParamErr(
				c,
				ParamErrorMsg(e.Field(), e.Tag()),
				err,
			)
		}
	}

	if _, ok := err.(*json.UnmarshalTypeError); ok {
		return serializer.ParamErr(c, "JSON marshall error", err)
	}

	return serializer.ParamErr(c, "Parameter error", err)
}

// FromJSON Parse and validate JSON from request body
func FromJSON[T any](ctxKey any) gin.HandlerFunc {
	return func(c *gin.Context) {
		var service T
		if err := c.ShouldBindJSON(&service); err == nil {
			c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxKey, &service))
			c.Next()
		} else {
			c.JSON(200, ErrorResponse(c, err))
			c.Abort()
		}
	}
}

// FromQuery Parse and validate form from request query
func FromQuery[T any](ctxKey any) gin.HandlerFunc {
	return func(c *gin.Context) {
		var service T
		if err := c.ShouldBindQuery(&service); err == nil {
			c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxKey, &service))
			c.Next()
		} else {
			c.JSON(200, ErrorResponse(c, err))
			c.Abort()
		}
	}
}

func FromForm[T any](ctxKey any) gin.HandlerFunc {
	return func(c *gin.Context) {
		var service T
		if err := c.ShouldBind(&service); err == nil {
			c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxKey, &service))
			c.Next()
		} else {
			c.JSON(200, ErrorResponse(c, err))
			c.Abort()
		}
	}
}

// FromUri Parse and validate form from request uri
func FromUri[T any](ctxKey any) gin.HandlerFunc {
	return func(c *gin.Context) {
		var service T
		if err := c.ShouldBindUri(&service); err == nil {
			c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxKey, &service))
			c.Next()
		} else {
			c.JSON(200, ErrorResponse(c, err))
			c.Abort()
		}
	}
}

// ParametersFromContext retrieves request parameters from context
func ParametersFromContext[T any](c *gin.Context, ctxKey any) T {
	return c.Request.Context().Value(ctxKey).(T)
}
