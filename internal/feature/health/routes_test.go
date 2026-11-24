package health_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"medpath-server/internal/feature/health"
	"medpath-server/internal/platform/config"
	"medpath-server/internal/platform/httpserver"
)

func newRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	cfg := config.Config{AllowedOrigins: []string{"http://localhost:3000"}}
	router := httpserver.NewRouter(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	health.Register(router, "test")
	return router
}

func TestHealthz(t *testing.T) {
	rec := httptest.NewRecorder()
	newRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body.Status != "ok" || body.Version != "test" {
		t.Fatalf("body = %+v", body)
	}
	if rec.Header().Get(httpserver.RequestIDHeader) == "" {
		t.Fatal("missing request ID header")
	}
}

func TestRequestIDIsEchoedOnlyWhenSafe(t *testing.T) {
	router := newRouter()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(httpserver.RequestIDHeader, "abc-123")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if got := rec.Header().Get(httpserver.RequestIDHeader); got != "abc-123" {
		t.Fatalf("request ID = %q, want abc-123", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(httpserver.RequestIDHeader, "bad id\nwith newline")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if got := rec.Header().Get(httpserver.RequestIDHeader); got == "" || got == "bad id\nwith newline" {
		t.Fatalf("unsafe request ID was echoed: %q", got)
	}
}

func TestCORS(t *testing.T) {
	router := newRouter()

	req := httptest.NewRequest(http.MethodOptions, "/healthz", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("preflight: status %d, allow-origin %q", rec.Code, rec.Header().Get("Access-Control-Allow-Origin"))
	}

	req = httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unknown origin was allowed: %q", got)
	}
}

func TestUnknownRouteReturnsJSON404(t *testing.T) {
	rec := httptest.NewRecorder()
	newRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/nope", nil))
	if rec.Code != http.StatusNotFound || rec.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("status %d, content-type %q", rec.Code, rec.Header().Get("Content-Type"))
	}
}
