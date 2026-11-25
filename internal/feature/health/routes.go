package health

import (
	"context"
	"log/slog"
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

type ReadinessResponse struct {
	Status   string `json:"status" example:"ok" enums:"ok,unavailable"`
	Database string `json:"database" example:"ok" enums:"ok,unavailable"`
}

// Pinger is satisfied by *sql.DB.
type Pinger interface {
	PingContext(ctx context.Context) error
}

const readinessTimeout = 2 * time.Second

func Register(router gin.IRouter, version string, db Pinger, logger *slog.Logger) {
	router.GET("/healthz", healthz(version))
	router.GET("/readyz", readyz(db, logger))
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

// readyz godoc
//
//	@Summary		Readiness check
//	@Description	Reports whether the server can take traffic, which requires a reachable database.
//	@Tags			health
//	@Produce		json
//	@Success		200	{object}	ReadinessResponse
//	@Failure		503	{object}	ReadinessResponse
//	@Router			/readyz [get]
func readyz(db Pinger, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), readinessTimeout)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			// The reason stays in the server log; callers only learn that the database is down.
			logger.WarnContext(ctx, "readiness check failed", slog.String("error", err.Error()))
			c.JSON(http.StatusServiceUnavailable, ReadinessResponse{Status: "unavailable", Database: "unavailable"})
			return
		}
		c.JSON(http.StatusOK, ReadinessResponse{Status: "ok", Database: "ok"})
	}
}
