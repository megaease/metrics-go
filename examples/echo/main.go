package main

import (
	"fmt"
	"log"

	"github.com/labstack/echo/v4"
	"github.com/megaease/metrics-go/metricshub"
	"github.com/megaease/metrics-go/middleware"
)

func main() {
	app := echo.New()

	app.HideBanner = true
	app.HidePort = true

	config := &metricshub.MetricsHubConfig{
		ServiceName: "vm-operator-echo",
		HostName:    "sprite-run-serverless-01",
	}
	// Initialize MetricsHub
	mHub := metricshub.NewMetricsHub(config)
	app.Use(middleware.NewEchoMetricsCollector(mHub))

	app.GET("/metrics", echo.WrapHandler(mHub.HTTPHandler()))
	app.GET("/health/:component", func(c echo.Context) error {
		component := c.Param("component")
		if component == "" {
			return c.JSON(400, "component is required")
		}
		log.Printf("health check for component: %s", component)
		return c.JSON(200, "ok")
	})

	log.Printf("Serving metrics at :8080/metrics")
	err := app.Start(fmt.Sprintf(":%d", 8080))
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
