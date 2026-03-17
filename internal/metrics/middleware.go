package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		endpoint := c.FullPath()
		if endpoint == "" {
			endpoint = "unknown"
		}

		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method

		HTTPRequests.WithLabelValues(method, endpoint, status).Inc()
		HTTPDuration.WithLabelValues(method, endpoint).
			Observe(time.Since(start).Seconds())
	}
}
