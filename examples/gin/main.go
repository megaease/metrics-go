package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/megaease/metrics-go/metricshub"
	"github.com/megaease/metrics-go/middleware"
)

func main() {
	// Create a Gin router
	router := gin.Default()

	// MetricsHub configuration
	config := &metricshub.MetricsHubConfig{
		ServiceName: "vm-operator-gion",
		HostName:    "sprite-run-serverless-01",
	}
	mHub := metricshub.NewMetricsHub(config)

	router.Use(middleware.NewGinMetricsCollector(mHub))

	// Route to serve metrics
	router.GET("/metrics", gin.WrapH(mHub.HTTPHandler()))

	// Health check route
	router.GET("/health/:component", func(c *gin.Context) {
		component := c.Param("component")
		if component == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "component is required"})
			return
		}
		time.Sleep(1 * time.Second)
		log.Printf("health check for component: %s", component)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Start the server
	port := 8080
	log.Printf("Serving metrics at :%d/metrics", port)
	err := router.Run(fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
