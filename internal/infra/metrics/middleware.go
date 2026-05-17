package metrics

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		endpoint := c.FullPath()
		if endpoint == "/metrics" || strings.HasPrefix(endpoint, "/swagger") {
			c.Next()
			return
		}

		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method

		HTTPRequests.WithLabelValues(method, endpoint, status).Inc()
		HTTPDuration.WithLabelValues(method, endpoint).
			Observe(time.Since(start).Seconds())
	}
}
