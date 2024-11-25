package metricshub

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsHub wraps Prometheus metrics for monitoring purposes.
type MetricsHub struct {
	registry *prometheus.Registry
	metrics  map[string]prometheus.Collector
}

// NewMetricsHub initializes a new MetricsHub instance.
func NewMetricsHub() *MetricsHub {
	return &MetricsHub{
		registry: prometheus.NewRegistry(),
		metrics:  make(map[string]prometheus.Collector),
	}
}

// RegisterMetric registers a new Prometheus metric with the hub.
func (hub *MetricsHub) RegisterMetric(name string, metric prometheus.Collector) error {
	if _, exists := hub.metrics[name]; exists {
		return nil // Already registered
	}
	hub.metrics[name] = metric
	return hub.registry.Register(metric)
}

// HTTPhandler returns an HTTP handler for the metrics endpoint.
func (hub *MetricsHub) HTTPhandler() http.Handler {
	return promhttp.HandlerFor(hub.registry, promhttp.HandlerOpts{})
}

// CurrentMetrics returns a snapshot of all registered metrics.
func (hub *MetricsHub) CurrentMetrics() []string {
	var metricNames []string
	for name := range hub.metrics {
		metricNames = append(metricNames, name)
	}
	return metricNames
}

// UpdateMetrics allows dynamic updates to a specific metric by its name.
func (hub *MetricsHub) UpdateMetrics(name string, value float64) error {
	metric, exists := hub.metrics[name]
	if !exists {
		return nil // Metric not found
	}

	// Check if the metric is a Gauge or Counter and update accordingly.
	switch m := metric.(type) {
	case prometheus.Gauge:
		m.Set(value)
	case prometheus.Counter:
		m.Add(value)
	}
	return nil
}
