package main

import (
	"log"
	"net/http"

	"github.com/megaease/metrics-go/metricshub"

	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	config := &metricshub.MetricsHubConfig{
		ServiceName: "vm-operator",
		HostName:    "sprite-run-serverless-01",
	}
	// Initialize MetricsHub
	mHub := metricshub.NewMetricsHub(config)

	// Create and register metrics
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "example_gauge",
		Help: "An example gauge metric",
		ConstLabels: prometheus.Labels{
			"label_1": "value_1",
		},
	})
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "example_counter",
		Help: "An example counter metric",
		ConstLabels: prometheus.Labels{
			"label_2": "value_2",
		},
	})

	mHub.RegisterMetric("example_gauge", gauge)
	mHub.RegisterMetric("example_counter", counter)

	// Update metrics dynamically
	mHub.UpdateMetrics("example_gauge", 42.5, nil)
	mHub.UpdateMetrics("example_counter", 1, nil)

	// Serve metrics
	http.Handle("/metrics", mHub.HTTPHandler())
	log.Println("Serving metrics at :8080/metrics")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
