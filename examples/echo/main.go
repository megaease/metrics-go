package main

import (
	"fmt"
	"log"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/megaease/metrics-go/metricshub"
	"github.com/megaease/metrics-go/middleware"
)

func main() {
	app := echo.New()

	app.HideBanner = true
	app.HidePort = true

	config := &metricshub.MetricsHubConfig{
		ServiceName: "vm-operator",
		HostName:    "sprite-run-serverless-01",
	}
	// Initialize MetricsHub
	mHub := metricshub.NewMetricsHub(config)
	app.Use(middleware.NewMetricsCollector(mHub))

	app.GET("/metrics", echo.WrapHandler(mHub.HTTPHandler()))
	app.GET("/health/:component", func(c echo.Context) error {
		component := c.Param("component")
		if component == "" {
			return c.JSON(400, "component is required")
		}
		time.Sleep(1 * time.Second)
		log.Printf("health check for component: %s", component)
		return c.JSON(200, "ok")
	})

	err := app.Start(fmt.Sprintf(":%d", 8080))
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
