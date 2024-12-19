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
		Labels: map[string]string{
			"env": "dev",
		},
		EnableHostNameLabel: true,
	}
	// Initialize MetricsHub
	mHub := metricshub.NewMetricsHub(config)
	err := mHub.RegisterMetric(&metricshub.MetricRegistration{
		Name:      "total_stocks",
		Help:      "total stocks",
		Type:      metricshub.MetricTypeGaugeVec,
		LabelKeys: []string{"cluster_id", "dataCenter_id", "spec_name", "node_name"},
	})
	if err != nil {
		log.Fatalf("register total_stocks metric failed: %v", err)
	}

	err = mHub.IncMetrics("total_stocks", map[string]string{
		"cluster_id":    "cluster-01",
		"dataCenter_id": "dc-01",
		"spec_name":     "spec-01",
		"node_name":     "node-01",
	})
	if err != nil {
		log.Fatalf("inc total_stocks metric failed: %v", err)
	}

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
	err = app.Start(fmt.Sprintf(":%d", 8080))
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
