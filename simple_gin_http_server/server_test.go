package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestTidMiddleware_Existing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()

	tid := "testtid"
	c, _ := gin.CreateTestContext(w)
	req, err := http.NewRequest(http.MethodGet, "ping", strings.NewReader(""))
	assert.Nil(t, err)
	
	c.Request = req
	c.Set(TidCtxKey, tid)
	TidMiddleware(c)
	assert.Equal(t, tid, c.Writer.Header().Get(TidHeaderKey))
}

func TestTidMiddleware_NotExisting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	req, err := http.NewRequest(http.MethodGet, "ping", strings.NewReader(""))
	assert.Nil(t, err)
	
	c.Request = req
	TidMiddleware(c)
	assert.NotNil(t, c.Writer.Header().Get(TidHeaderKey))
}
