package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/megaease/metrics-go/metricshub"
)

// NewGinMetricsCollector creates a Gin middleware to collect HTTP request metrics.
func NewGinMetricsCollector(hub *metricshub.MetricsHub) gin.HandlerFunc {
	return func(c *gin.Context) {
		startAt := time.Now()

		// Process the next handler in the chain
		c.Next()

		// Calculate processing time and extract request details
		processTime := time.Since(startAt)
		routePath := c.FullPath() // Use the registered router path directly
		method := c.Request.Method
		statusCode := c.Writer.Status()
		bodyBytesReceived := c.Request.ContentLength
		bodyBytesSent := int64(c.Writer.Size())

		// Prepare the metric data
		requestMetric := &metricshub.RequestMetric{
			StatusCode: statusCode,
			Duration:   processTime,
			ReqSize:    uint64(bodyBytesReceived),
			RespSize:   uint64(bodyBytesSent),
		}

		// Update metrics in the MetricsHub
		hub.UpdateHTTPRequestMetrics(requestMetric, method, routePath)
	}
}
