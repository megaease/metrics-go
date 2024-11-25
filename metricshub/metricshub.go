package metricshub

import (
	"bytes"
	"errors"
	"fmt"
	dto "github.com/prometheus/client_model/go"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// httpStatusUpdateInterval is the interval for updating HTTP status metrics.
	// It is set to 5 seconds to match the default interval used by go-metrics.
	// https://github.com/rcrowley/go-metrics/blob/3113b8401b8a98917cde58f8bbd42a1b1c03b1fd/ewma.go#L98-L99
	httpStatusUpdateInterval = 5 * time.Second

	// defaultSlackWebhookURL is the default webhook URL for Slack notifications.
	// It will send the notifications to the #online-alert channel.
	defaultSlackWebhookURL = "https://hooks.slack.com/services/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"
)

// MetricsHub wraps Prometheus metrics for monitoring purposes.
type (
	MetricsHubConfig struct {
		// ServiceName is the name of the service. It is required.
		// The service name will be used as a label in the http metrics.
		// Other custom metrics will not add this label, should be added manually.
		ServiceName string `yaml:"serviceName" json:"serviceName"`
		// HostName is the hostname of the service. It is required.
		// The hostname will be used as a label in the http metrics.
		// Other custom metrics will not add this label, should be added manually.
		HostName string `yaml:"hostName" json:"hostName"`
		// Labels is the additional labels for the service.
		// This labels will be added to the http metrics.
		// Other custom metrics will not add this label, should be added manually.
		// +optional
		Labels map[string]string `yaml:"labels" json:"labels"`
		// SlackWebhookURL is the webhook URL for Slack notifications.
		// If not set, the default value will be used when sending notifications.
		// So be sure to set this value if you want to receive notifications.
		// +optional
		SlackWebhookURL string `yaml:"slackWebhookURL" json:"slackWebhookURL"`
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

	mergeMetric struct {
		Labels map[string]string
		value  float64
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
// If the metric is not unique, should use *Vec type, not *Gauge or *Counter.
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

// CurrentMetrics returns a snapshot of all custom metrics registered with the hub.
func (hub *MetricsHub) CurrentMetrics() []string {
	var metricNames []string
	for name := range hub.metrics {
		metricNames = append(metricNames, name)
	}
	return metricNames
}

// UpdateMetrics allows dynamic updates to a specific metric by its name.
// Labels are optional and only used for *Vec types.
func (hub *MetricsHub) UpdateMetrics(name string, value float64, selectLabels map[string]string) error {
	metric, exists := hub.metrics[name]
	if !exists {
		return nil // RequestMetric not found
	}

	// Check if the metric is a Gauge or Counter and update accordingly.
	switch m := metric.(type) {
	case *prometheus.GaugeVec:
		m.With(selectLabels).Set(value)
	case prometheus.Gauge:
		m.Set(value)
	case *prometheus.CounterVec:
		if value > 1 {
			m.With(selectLabels).Add(value)
		} else {
			m.With(selectLabels).Inc()
		}
	case prometheus.Counter:
		if value > 1 {
			m.Add(value)
		} else {
			m.Inc()
		}
	case *prometheus.SummaryVec:
		m.With(selectLabels).Observe(value)
	case prometheus.Summary:
		m.Observe(value)
	case *prometheus.HistogramVec:
		m.With(selectLabels).Observe(value)
	case prometheus.Histogram:
		m.Observe(value)
	case prometheus.ObserverVec:
		m.With(selectLabels).Observe(value)
	case prometheus.Observer:
		m.Observe(value)
	default:
		return errors.New("unsupported metric type")
	}
	return nil
}

// UpdateHTTPRequestMetrics updates the HTTP request metrics.
// Do not call this method directly, use the middleware instead.
// Or only when you need to call the third-party API, and statistics are needed.
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

// NotifySlack sends a message to the Slack webhook.
// If the webhook URL is not set, the default value will be used.
// So be sure to set the webhook URL if you want to receive notifications.
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

func (hub *MetricsHub) groupMetrics(metric prometheus.Collector, labels, mergedLabels []string, metricType string) ([]mergeMetric, error) {
	mfs := make(chan prometheus.Metric)
	go func() {
		metric.Collect(mfs)
		close(mfs)
	}()

	groupedValues := make(map[string]float64)
	groupedLabels := make(map[string]map[string]string)

	for mf := range mfs {
		m := &dto.Metric{}
		err := mf.Write(m)
		if err != nil {
			return nil, err
		}

		compositeKeyParts := make([]string, len(labels))
		metricLabels := m.GetLabel()
		mergedMetric := false
		for i := range metricLabels {
			if metricLabels[i].GetValue() == "merged" {
				mergedMetric = true
			}
		}
		if mergedMetric {
			continue
		}

		newLabels := make(map[string]string)
		for _, label := range labels {
			for i := range metricLabels {
				if metricLabels[i].GetName() == label {
					compositeKeyParts = append(compositeKeyParts, metricLabels[i].GetValue())
					newLabels[label] = metricLabels[i].GetValue()
					break
				}
			}
		}
		compositeKey := strings.Join(compositeKeyParts, "，")

		value := hub.getMetricValue(m, metricType)
		if _, exists := groupedValues[compositeKey]; !exists {
			groupedValues[compositeKey] = value
			groupedLabels[compositeKey] = make(map[string]string)
			groupedLabels[compositeKey] = newLabels
		} else {
			groupedValues[compositeKey] += value
		}
	}

	var mergedMetrics []mergeMetric
	for key, groupLabels := range groupedLabels {
		for _, label := range mergedLabels {
			groupLabels[label] = "merged"
		}
		value := groupedValues[key]
		mergedMetrics = append(mergedMetrics, mergeMetric{
			Labels: groupLabels,
			value:  value,
		})
	}
	return mergedMetrics, nil
}

func (hub *MetricsHub) MergeMetric(metric prometheus.Collector, labels, mergedLabels []string) error {
	switch m := metric.(type) {
	case *prometheus.GaugeVec:
		mergedMetrics, err := hub.groupMetrics(metric, labels, mergedLabels, "gauge")
		if err != nil {
			return err
		}
		for _, mergedMetric := range mergedMetrics {
			m.With(mergedMetric.Labels).Set(mergedMetric.value)
		}
	case *prometheus.CounterVec:
		mergedMetrics, err := hub.groupMetrics(metric, labels, mergedLabels, "counter")
		if err != nil {
			return err
		}
		for _, mergedMetric := range mergedMetrics {
			m.With(mergedMetric.Labels).Add(mergedMetric.value)
		}
	case *prometheus.SummaryVec:
		mergedMetrics, err := hub.groupMetrics(metric, labels, mergedLabels, "summary")
		if err != nil {
			return err
		}
		for _, mergedMetric := range mergedMetrics {
			m.With(mergedMetric.Labels).Observe(mergedMetric.value)
		}
	case *prometheus.HistogramVec:
		mergedMetrics, err := hub.groupMetrics(metric, labels, mergedLabels, "histogram")
		if err != nil {
			return err
		}
		for _, mergedMetric := range mergedMetrics {
			m.With(mergedMetric.Labels).Observe(mergedMetric.value)
		}
	default:
		return errors.New("unsupported metric type")
	}

	return nil
}

func (hub *MetricsHub) getMetricValue(m *dto.Metric, metricType string) float64 {
	switch metricType {
	case "gauge":
		return m.GetGauge().GetValue()
	case "counter":
		return m.GetCounter().GetValue()
	case "summary":
		return m.GetSummary().GetSampleSum()
	case "histogram":
		return m.GetHistogram().GetSampleSum()
	default:
		return 0
	}
}
