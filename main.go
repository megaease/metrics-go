package main

import (
	"log"
	"net/http"

	"github.com/megaease/metrics-go/metricshub"
)

func main() {
	config := &metricshub.MetricsHubConfig{
		ServiceName: "vm-operator",
		HostName:    "sprite-run-serverless-01",
	}
	// Initialize MetricsHub
	mHub := metricshub.NewMetricsHub(config)

	mHub.RegisterMetric(&metricshub.MetricRegistration{
		Name:      "example_gauge",
		Help:      "An example gauge metric",
		Type:      metricshub.MetricTypeGaugeVec,
		LabelKeys: []string{"label_1"},
	})

	mHub.RegisterMetric(&metricshub.MetricRegistration{
		Name:      "example_counter",
		Help:      "An example counter metric",
		Type:      metricshub.MetricTypeCounterVec,
		LabelKeys: []string{"label_2"},
	})

	// Update metrics dynamically
	mHub.UpdateMetrics("example_gauge", 42.5, nil)
	mHub.UpdateMetrics("example_counter", 1, nil)

	// Serve metrics
	http.Handle("/metrics", mHub.HTTPHandler())
	log.Println("Serving metrics at :8080/metrics")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
