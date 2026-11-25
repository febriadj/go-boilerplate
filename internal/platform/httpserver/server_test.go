package httpserver_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"medpath-server/internal/platform/config"
	"medpath-server/internal/platform/httpserver"

	"github.com/gin-gonic/gin"
)

func newRouter(env config.Environment) *gin.Engine {
	gin.SetMode(gin.TestMode)
	cfg := config.Config{Env: env, AllowedOrigins: []string{"http://localhost:3000"}}
	router := httpserver.NewRouter(cfg, slog.New(slog.DiscardHandler))
	router.GET("/ok", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	router.GET("/panic", func(*gin.Context) { panic("boom") })
	return router
}

func serve(router http.Handler, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// Error responses carry the same headers, so a 404 or a recovered panic never leaks cacheable data.
func TestSecurityHeadersOnEveryResponse(t *testing.T) {
	want := map[string]string{
		"Content-Security-Policy":      httpserver.APIContentSecurityPolicy,
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "DENY",
		"Referrer-Policy":              "no-referrer",
		"Cross-Origin-Opener-Policy":   "same-origin",
		"Cross-Origin-Resource-Policy": "same-origin",
		"Cache-Control":                "no-store",
	}
	router := newRouter(config.Production)

	for path, status := range map[string]int{
		"/ok":      http.StatusOK,
		"/missing": http.StatusNotFound,
		"/panic":   http.StatusInternalServerError,
	} {
		rec := serve(router, http.MethodGet, path, nil)
		if rec.Code != status {
			t.Fatalf("%s: status = %d, want %d", path, rec.Code, status)
		}
		for name, value := range want {
			if got := rec.Header().Get(name); got != value {
				t.Errorf("%s: %s = %q, want %q", path, name, got, value)
			}
		}
	}
}

func TestHSTSOnlyOutsideDevelopment(t *testing.T) {
	for env, want := range map[config.Environment]bool{
		config.Development: false,
		config.Staging:     true,
		config.Production:  true,
	} {
		got := serve(newRouter(env), http.MethodGet, "/ok", nil).Header().Get("Strict-Transport-Security") != ""
		if got != want {
			t.Errorf("%s: HSTS present = %v, want %v", env, got, want)
		}
	}
}

func TestCORSAllowsOnlyConfiguredOrigins(t *testing.T) {
	router := newRouter(config.Development)

	rec := serve(router, http.MethodOptions, "/ok", map[string]string{"Origin": "http://localhost:3000"})
	if rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("preflight: status %d, allow-origin %q", rec.Code, rec.Header().Get("Access-Control-Allow-Origin"))
	}

	rec = serve(router, http.MethodGet, "/ok", map[string]string{"Origin": "https://evil.example"})
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unknown origin was allowed: %q", got)
	}
}

// A caller-supplied ID is logged, so only short plain values are kept.
func TestRequestIDIsEchoedOnlyWhenSafe(t *testing.T) {
	router := newRouter(config.Development)

	rec := serve(router, http.MethodGet, "/ok", map[string]string{httpserver.RequestIDHeader: "abc-123"})
	if got := rec.Header().Get(httpserver.RequestIDHeader); got != "abc-123" {
		t.Fatalf("request ID = %q, want abc-123", got)
	}

	rec = serve(router, http.MethodGet, "/ok", map[string]string{httpserver.RequestIDHeader: "bad id with spaces"})
	if got := rec.Header().Get(httpserver.RequestIDHeader); got == "" || got == "bad id with spaces" {
		t.Fatalf("unsafe request ID was echoed: %q", got)
	}
}
