package middleware

import (
	"net/http"
	"strings"
	"testing"

	"identity-manager/internal/testutil"

	"github.com/gin-gonic/gin"
)

func TestRequireInternalToken_MissingConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, rec := testutil.NewGinContext(http.MethodPost, "/v1/internal/identity/resolve", nil)
	RequireInternalToken("")(ctx)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"code":"INTERNAL_AUTH_NOT_CONFIGURED"`) {
		t.Fatalf("expected INTERNAL_AUTH_NOT_CONFIGURED code, got %s", rec.Body.String())
	}
}

func TestRequireInternalToken_MissingBearer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, rec := testutil.NewGinContext(http.MethodPost, "/v1/internal/identity/resolve", nil)
	RequireInternalToken("secret")(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"code":"UNAUTHORIZED"`) {
		t.Fatalf("expected UNAUTHORIZED code, got %s", rec.Body.String())
	}
}

func TestRequireInternalToken_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, rec := testutil.NewGinContext(http.MethodPost, "/v1/internal/identity/resolve", nil)
	ctx.Request.Header.Set("Authorization", "Bearer wrong")
	RequireInternalToken("secret")(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRequireInternalToken_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, rec := testutil.NewGinContext(http.MethodPost, "/v1/internal/identity/resolve", nil)
	ctx.Request.Header.Set("Authorization", "Bearer secret")
	RequireInternalToken("secret")(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
