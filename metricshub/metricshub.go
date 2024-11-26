package metricshub

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// httpStatusUpdateInterval is the interval for updating HTTP status metrics.
	// It is set to 5 seconds to match the default interval used by go-metrics.
	// https://github.com/rcrowley/go-metrics/blob/3113b8401b8a98917cde58f8bbd42a1b1c03b1fd/ewma.go#L98-L99
	httpStatusUpdateInterval = 5 * time.Second

	defaultSlackWebhookURL = "https://hooks.slack.com/services/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"
)

// MetricsHub wraps Prometheus metrics for monitoring purposes.
type (
	MetricsHubConfig struct {
		ServiceName     string            `yaml:"serviceName" json:"serviceName"`
		HostName        string            `yaml:"hostName" json:"hostName"`
		Labels          map[string]string `yaml:"labels" json:"labels"`
		SlackWebhookURL string            `yaml:"slackWebhookURL" json:"slackWebhookURL"`
	}

	MetricsHub struct {
		config      *MetricsHubConfig
		registry    *prometheus.Registry
		metrics     map[string]prometheus.Collector
		httpMetrics *httpRequestMetrics
		httpStats   map[httpStatsKey]*HTTPStat
	}

	httpStatsKey struct {
		Method string
		Path   string
	}

	MetricCollector struct {
		Collector  prometheus.Collector
		Name       string
		Method     string
		Path       string
		HTTPStatus *HTTPStat
	}
)

// NewMetricsHub initializes a new MetricsHub instance.
func NewMetricsHub(config *MetricsHubConfig) *MetricsHub {
	hub := &MetricsHub{
		config:    config,
		registry:  prometheus.DefaultRegisterer.(*prometheus.Registry),
		metrics:   make(map[string]prometheus.Collector),
		httpStats: make(map[httpStatsKey]*HTTPStat),
	}

	hub.httpMetrics = hub.newHTTPMetrics()

	go hub.run()

	return hub
}

func (hub *MetricsHub) run() {
	ticker := time.NewTicker(httpStatusUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for key, stats := range hub.httpStats {
				status := stats.Status()
				hub.httpMetrics.exportPrometheusMetricsForTicker(status, key.Method, key.Path)
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
func (hub *MetricsHub) UpdateMetrics(name string, value float64, labels map[string]string) error {
	metric, exists := hub.metrics[name]
	if !exists {
		return nil // RequestMetric not found
	}

	// Check if the metric is a Gauge or Counter and update accordingly.
	switch m := metric.(type) {
	case prometheus.GaugeVec:
		m.With(labels).Set(value)
	case prometheus.Gauge:
		m.Set(value)
	case prometheus.CounterVec:
		if value > 1 {
			m.With(labels).Add(value)
		} else {
			m.With(labels).Inc()
		}
	case prometheus.Counter:
		if value > 1 {
			m.Add(value)
		} else {
			m.Inc()
		}
	case prometheus.SummaryVec:
		m.With(labels).Observe(value)
	case prometheus.Summary:
		m.Observe(value)
	case prometheus.HistogramVec:
		m.With(labels).Observe(value)
	case prometheus.Histogram:
		m.Observe(value)
	case prometheus.ObserverVec:
		m.With(labels).Observe(value)
	case prometheus.Observer:
		m.Observe(value)
	default:
		return errors.New("unsupported metric type")
	}
	return nil
}

func (hub *MetricsHub) UpdateHTTPRequestMetrics(requestMetric *RequestMetric, method, path string) {
	key := httpStatsKey{
		Method: method,
		Path:   path,
	}

	stat, exists := hub.httpStats[key]
	if !exists {
		stat = NewHTTPStat()
		hub.httpStats[key] = stat
	}

	if stat == nil {
		return
	}

	if hub.httpMetrics == nil {
		return
	}

	if requestMetric == nil {
		return
	}

	stat.Stat(requestMetric)
	hub.httpMetrics.exportPrometheusMetricsForRequestMetric(requestMetric, method, path)
}

func (hub *MetricsHub) NotifySlack(msg string) error {
	if hub.config.SlackWebhookURL == "" {
		hub.config.SlackWebhookURL = defaultSlackWebhookURL
	}

	req, err := http.NewRequest(http.MethodPost, hub.config.SlackWebhookURL, bytes.NewBuffer([]byte(msg)))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Close = true

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("error response from Slack - code [%d] - msg [%s]", resp.StatusCode, string(buf))
	}
	return nil
}
