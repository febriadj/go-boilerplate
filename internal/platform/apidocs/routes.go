package apidocs

import (
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
// Scalar's AI agent and vendor toolbar stay off so the spec is never sent to a third party.
func Register(router gin.IRouter, version string) {
	docs.SwaggerInfo.Version = version

	router.GET(specPath, func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(docs.SwaggerInfo.ReadDoc()))
	})

	router.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(scalarPage))
	})

	router.GET("/docs/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/docs/swagger/index.html")
	})
	router.GET("/docs/swagger/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL(specPath),
		ginSwagger.DefaultModelsExpandDepth(1),
	))
}

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
<script>
Scalar.createApiReference('#app', {
  url: '` + specPath + `',
  hideClientButton: true,
  showDeveloperTools: 'never',
  agent: { disabled: true },
})
</script>
</body>
</html>
`
