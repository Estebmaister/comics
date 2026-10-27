package middleware

import (
	"time"

	"comics/internal/metrics"

	"github.com/gin-gonic/gin"
)

// CatalogMetricsMiddleware records stable-label HTTP latency for catalog routes.
func CatalogMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}
		outcome := "success"
		switch {
		case c.Writer.Status() >= 500:
			outcome = "error"
		case c.Writer.Status() >= 400:
			outcome = "client_error"
		}
		metrics.ObserveCatalogHTTP(route, outcome, time.Since(start))
	}
}
