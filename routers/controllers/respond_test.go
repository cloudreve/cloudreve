package controllers

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRespondErr(t *testing.T) {
	gin.SetMode(gin.TestMode)
	a := assert.New(t)

	// nil err: no write, no abort, returns false.
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	a.False(respondErr(c, nil))
	a.False(c.IsAborted())
	a.Equal(200, w.Code)
	a.Empty(w.Body.String())

	// non-nil err: error body written, chain aborted, returns true.
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	a.True(respondErr(c, errors.New("boom")))
	a.True(c.IsAborted())
	a.Contains(w.Body.String(), "boom")
}

func TestRespond(t *testing.T) {
	gin.SetMode(gin.TestMode)
	a := assert.New(t)

	// error path writes the error body, not the success body.
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	respond(c, errors.New("nope"), serializer.Response{})
	a.Contains(w.Body.String(), "nope")

	// success path writes the success body.
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	respond(c, nil, serializer.Response{})
	a.Equal(200, w.Code)
	a.NotContains(w.Body.String(), "nope")
}
