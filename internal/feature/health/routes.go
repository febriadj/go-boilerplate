package health

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Status  string    `json:"status" example:"ok"`
	Service string    `json:"service" example:"medpath-server"`
	Version string    `json:"version" example:"0.1.0"`
	Time    time.Time `json:"time" example:"2026-11-24T08:00:00Z"`
}

// Register mounts GET /healthz. A readiness check for the database is added with Postgres.
func Register(router gin.IRouter, version string) {
	router.GET("/healthz", healthz(version))
}

// healthz godoc
//
//	@Summary		Liveness check
//	@Description	Reports that the process is up. It does not check the database.
//	@Tags			health
//	@Produce		json
//	@Success		200	{object}	Response
//	@Router			/healthz [get]
func healthz(version string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, Response{
			Status:  "ok",
			Service: "medpath-server",
			Version: version,
			Time:    time.Now().UTC(),
		})
	}
}
