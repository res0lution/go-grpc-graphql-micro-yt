package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"identity-manager/internal/testutil"
)

type pingerMock struct {
	pingFn func(context.Context) error
}

func (m *pingerMock) Ping(ctx context.Context) error {
	return m.pingFn(ctx)
}

func TestHealthHandler_Up(t *testing.T) {
	h := NewHealthHandler(&pingerMock{pingFn: func(context.Context) error { return nil }})
	ctx, rec := testutil.NewGinContext(http.MethodGet, "/health", nil)
	h.Health(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHealthHandler_Down(t *testing.T) {
	h := NewHealthHandler(&pingerMock{pingFn: func(context.Context) error { return errors.New("db down") }})
	ctx, rec := testutil.NewGinContext(http.MethodGet, "/health", nil)
	h.Health(ctx)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"code":"SERVICE_UNAVAILABLE"`) {
		t.Fatalf("expected SERVICE_UNAVAILABLE code in response: %s", rec.Body.String())
	}
}

func TestHealthHandler_Live(t *testing.T) {
	h := NewHealthHandler(&pingerMock{pingFn: func(context.Context) error { return nil }})
	ctx, rec := testutil.NewGinContext(http.MethodGet, "/live", nil)
	h.Live(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
