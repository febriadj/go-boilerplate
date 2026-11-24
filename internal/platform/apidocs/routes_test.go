package apidocs_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"medpath-server/internal/platform/apidocs"
)

func newRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apidocs.Register(router, "9.9.9")
	return router
}

func get(t *testing.T, router http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestSpecIsServedWithBuildVersion(t *testing.T) {
	rec := get(t, newRouter(), "/docs/openapi.json")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var spec struct {
		Swagger string                     `json:"swagger"`
		Info    struct{ Version string }   `json:"info"`
		Paths   map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &spec); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if spec.Swagger != "2.0" || spec.Info.Version != "9.9.9" {
		t.Fatalf("swagger %q, version %q", spec.Swagger, spec.Info.Version)
	}
	if _, ok := spec.Paths["/healthz"]; !ok {
		t.Fatalf("paths = %v, want /healthz", spec.Paths)
	}
}

func TestScalarPage(t *testing.T) {
	rec := get(t, newRouter(), "/docs")
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("status %d, content-type %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(body, "@scalar/api-reference@1.77.0") || !strings.Contains(body, "/docs/openapi.json") {
		t.Fatal("Scalar page does not load the pinned script or the spec URL")
	}
	if !strings.Contains(body, "agent: { disabled: true }") || !strings.Contains(body, "showDeveloperTools: 'never'") {
		t.Fatal("Scalar AI agent or developer toolbar is not disabled")
	}
}

func TestSwaggerUI(t *testing.T) {
	router := newRouter()
	if rec := get(t, router, "/docs/swagger"); rec.Code != http.StatusMovedPermanently {
		t.Fatalf("/docs/swagger status = %d, want 301", rec.Code)
	}
	rec := get(t, router, "/docs/swagger/index.html")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "swagger-ui") {
		t.Fatalf("index.html status = %d", rec.Code)
	}
	init := get(t, router, "/docs/swagger/swagger-initializer.js")
	if !strings.Contains(init.Body.String(), "/docs/openapi.json") {
		t.Fatal("Swagger UI does not point at /docs/openapi.json")
	}
}
