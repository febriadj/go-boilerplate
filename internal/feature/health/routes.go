package health

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type response struct {
	Status  string    `json:"status"`
	Service string    `json:"service"`
	Version string    `json:"version"`
	Time    time.Time `json:"time"`
}

// Register mounts GET /healthz. A readiness check for the database is added with Postgres.
func Register(router gin.IRouter, version string) {
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, response{
			Status:  "ok",
			Service: "medpath-server",
			Version: version,
			Time:    time.Now().UTC(),
		})
	})
}
