package httpserver

import (
	"log/slog"
	"net/http"
	"time"

	"medpath-server/internal/platform/config"

	"github.com/gin-gonic/gin"
)

// NewRouter builds the Gin engine with the middleware every feature shares.
func NewRouter(cfg config.Config, logger *slog.Logger) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	// No proxy is trusted until the deployment says otherwise, so client IPs cannot be spoofed.
	_ = router.SetTrustedProxies(nil)
	router.Use(
		gin.CustomRecovery(func(c *gin.Context, recovered any) {
			logger.Error("panic recovered",
				slog.String("request_id", RequestIDFrom(c)),
				slog.Any("panic", recovered),
			)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}),
		RequestID(),
		AccessLog(logger),
		CORS(cfg.AllowedOrigins),
	)
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})
	return router
}

func NewServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
