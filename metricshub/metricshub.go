package metricshub

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// DefaultTimeTicker is the default time ticker for updating metrics.
	DefaultTimeTicker = 5 * time.Second
)

// MetricsHub wraps Prometheus metrics for monitoring purposes.
type (
	MetricsHubConfig struct {
		ServiceName string `yaml:"serviceName" json:"serviceName"`
		HostName    string `yaml:"hostName" json:"hostName"`
	}

	MetricsHub struct {
		config          *MetricsHubConfig
		registry        *prometheus.Registry
		metrics         map[string]prometheus.Collector
		internalMetrics *internalMetrics
		internalStats   map[internalStatsKey]*HTTPStat
	}

	internalStatsKey struct {
		Method string
		Path   string
	}

	MetricCollector struct {
		Collector prometheus.Collector
		Name      string
		Method    string
		Path      string
		HttpStat  *HTTPStat
	}
)

// NewMetricsHub initializes a new MetricsHub instance.
func NewMetricsHub(config *MetricsHubConfig) *MetricsHub {
	hub := &MetricsHub{
		config:        config,
		registry:      prometheus.DefaultRegisterer.(*prometheus.Registry),
		metrics:       make(map[string]prometheus.Collector),
		internalStats: make(map[internalStatsKey]*HTTPStat),
	}

	// inject the internal metrics
	hub.internalMetrics = hub.newInternalMetrics()

	go hub.run()
	return hub
}

func (hub *MetricsHub) run() {
	ticker := time.NewTicker(DefaultTimeTicker)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for key, stats := range hub.internalStats {
				status := stats.Status()
				hub.internalMetrics.exportPrometheusMetricsForTicker(status, key.Method, key.Path)
			}
		}
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

// HTTPHandler returns an HTTP handler for the metrics endpoint.
func (hub *MetricsHub) HTTPHandler() http.Handler {
	return promhttp.Handler()
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
		return nil // RequestMetric not found
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

func (hub *MetricsHub) UpdateInternalMetrics(requestMetric *RequestMetric, method, path string) {
	key := internalStatsKey{
		Method: method,
		Path:   path,
	}

	stat, exists := hub.internalStats[key]
	if !exists {
		stat = NewHTTPStat()
		hub.internalStats[key] = stat
	}

	if stat == nil {
		return
	}

	if hub.internalMetrics == nil {
		return
	}

	if requestMetric == nil {
		return
	}

	stat.Stat(requestMetric)
	hub.internalMetrics.exportPrometheusMetricsForRequestMetric(requestMetric, method, path)
}
