package testutil

import (
	"bytes"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
)

func NewGinContext(method, target string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	ctx.Request = req

	return ctx, recorder
}

func AddCookie(ctx *gin.Context, name, value string) {
	ctx.Request.AddCookie(&http.Cookie{Name: name, Value: value})
}
