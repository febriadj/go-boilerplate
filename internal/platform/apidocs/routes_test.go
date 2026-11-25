package apidocs_test

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"medpath-server/internal/platform/apidocs"

	"github.com/gin-gonic/gin"
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

func TestDocsPagesHaveTheirOwnCSP(t *testing.T) {
	router := newRouter()

	rec := get(t, router, "/docs")
	csp := rec.Header().Get("Content-Security-Policy")
	inline := regexp.MustCompile(`(?s)<script>(.*?)</script>`).FindStringSubmatch(rec.Body.String())
	if inline == nil {
		t.Fatal("no inline script on the Scalar page")
	}
	sum := sha256.Sum256([]byte(inline[1]))
	hash := "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
	if !strings.Contains(csp, hash) || !strings.Contains(csp, "https://cdn.jsdelivr.net") {
		t.Fatalf("Scalar CSP %q does not allow the inline script %s", csp, hash)
	}
	for _, directive := range strings.Split(csp, ";") {
		if strings.HasPrefix(strings.TrimSpace(directive), "script-src") && strings.Contains(directive, "'unsafe-inline'") {
			t.Fatalf("Scalar CSP allows any inline script: %q", directive)
		}
	}

	swaggerCSP := get(t, router, "/docs/swagger/index.html").Header().Get("Content-Security-Policy")
	if !strings.Contains(swaggerCSP, "script-src 'self'") || strings.Contains(swaggerCSP, "cdn.jsdelivr.net") {
		t.Fatalf("Swagger UI CSP = %q", swaggerCSP)
	}
}
