package apidocs

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"

	"medpath-server/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

const (
	specPath = "/docs/openapi.json"
	// Pinned so a new Scalar release cannot change the page without a code change.
	scalarVersion = "1.77.0"
)

// Register serves the generated spec, Scalar at /docs, and Swagger UI at /docs/swagger/.
// Scalar's AI agent, vendor toolbar, and hosted fonts stay off so the page calls no third party
// besides the pinned script on jsDelivr.
func Register(router gin.IRouter, version string) {
	docs.SwaggerInfo.Version = version

	router.GET(specPath, func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(docs.SwaggerInfo.ReadDoc()))
	})

	router.GET("/docs", func(c *gin.Context) {
		c.Header("Content-Security-Policy", scalarCSP)
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(scalarPage))
	})

	router.GET("/docs/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/docs/swagger/index.html")
	})
	swagger := router.Group("/docs/swagger", func(c *gin.Context) {
		c.Header("Content-Security-Policy", swaggerCSP)
	})
	swagger.GET("/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL(specPath),
		ginSwagger.DefaultModelsExpandDepth(1),
	))
}

// The inline script is allowed by its hash, so editing it updates the policy automatically.
const scalarScript = `
Scalar.createApiReference('#app', {
  url: '` + specPath + `',
  hideClientButton: true,
  showDeveloperTools: 'never',
  agent: { disabled: true },
  withDefaultFonts: false,
})
`

const scalarPage = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Medpath API</title>
</head>
<body>
<div id="app"></div>
<script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference@` + scalarVersion + `"></script>
<script>` + scalarScript + `</script>
</body>
</html>
`

var scalarCSP = "default-src 'self'; " +
	"script-src https://cdn.jsdelivr.net '" + scriptHash(scalarScript) + "'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data:; " +
	"font-src 'self' data:; " + // system fonts; withDefaultFonts is off
	"connect-src 'self'; " +
	"frame-ancestors 'none'; base-uri 'none'; form-action 'none'"

// Swagger UI ships its scripts from this server; it still uses inline style attributes.
const swaggerCSP = "default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data:; " +
	"frame-ancestors 'none'; base-uri 'none'; form-action 'none'"

func scriptHash(script string) string {
	sum := sha256.Sum256([]byte(script))
	return "sha256-" + base64.StdEncoding.EncodeToString(sum[:])
}
