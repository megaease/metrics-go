package metricshub

import (
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	dto "github.com/prometheus/client_model/go"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// httpStatusUpdateInterval is the interval for updating HTTP status metrics.
	// It is set to 5 seconds to match the default interval used by go-metrics.
	// https://github.com/rcrowley/go-metrics/blob/3113b8401b8a98917cde58f8bbd42a1b1c03b1fd/ewma.go#L98-L99
	httpStatusUpdateInterval = 5 * time.Second

	// MergedLabelValue is the placeholder value for merged metrics.
	MergedLabelValue = "MERGED_LABEL"

	MetricTypeGaugeVec     MetricType = "GaugeVec"
	MetricTypeCounterVec   MetricType = "CounterVec"
	MetricTypeSummaryVec   MetricType = "SummaryVec"
	MetricTypeHistogramVec MetricType = "HistogramVec"

	golangType  = "golang"
	defaultType = "gpu-runtime"
)

var (
	defaultExcludedHttpPath = []string{"/metrics", "/actuator/health"}
)

// MetricsHub wraps Prometheus metrics for monitoring purposes.
type (
	MetricsHubConfig struct {
		// ServiceName is the name of the service. It is required.
		// The service name will be used as a label in the http metrics.
		// Other custom metrics will not add this label, should be added manually.
		ServiceName string `yaml:"serviceName" json:"serviceName"`
		// HostName is the hostname of the service.
		// The hostname will be used as a label in the http metrics.
		// Other custom metrics will not add this label, should be added manually.
		// +optional
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
		// EnableHostNameLabel is the flag to enable the hostname label in the http metrics.
		// Default is false.
		// If set to false, the hostname label will not be added to the http metrics.
		// If set to true, but the hostname is not set, the host_name label will be set to os.Hostname().
		// +optional
		EnableHostNameLabel bool `yaml:"enableHostNameLabel" json:"enableHostNameLabel"`

		// DisableFixedLabels is the flag to disable fixed labels in the http metrics.
		// Default is false.
		DisableFixedLabels bool `yaml:"disableFixedLabels" json:"disableFixedLabels"`

		// DisableDefaultExcludedHttpPath is the flag to disable default excluded http paths.
		// Default is false.
		DisableDefaultExcludedHttpPath bool `yaml:"disableDefaultExcludedHttpPath" json:"disableDefaultExcludedHttpPath"`

		// ExcludedHttpPath is the list of excluded http paths.
		// Default is ["/metrics", "/actuator/health"].
		// +optional
		ExcludedHttpPath []string `yaml:"excludedHttpPath" json:"excludedHttpPath"`
	}

	MetricsHub struct {
		config               *MetricsHubConfig
		registry             *prometheus.Registry
		metricsRegistrations map[string]*MetricRegistration
		httpMetrics          *httpRequestMetrics
		httpStats            map[httpStatsKey]*HTTPStat
		fixedLabels          prometheus.Labels
	}

	httpStatsKey struct {
		Method string
		Path   string
	}

	MetricType string

	MetricRegistration struct {
		// Name must be unique.
		Name string
		// Only support GaugeVec, CounterVec, SummaryVec, HistogramVec, and ObserverVec.
		Type MetricType
		// Help is the description of the metric.
		Help string
		// LabelKeys is the list of label keys.
		LabelKeys []string

		// Only used for HistogramVec.
		HistogramBuckets []float64
		// Only used for SummaryVec.
		SummaryObjectives map[float64]float64

		collector prometheus.Collector
	}

	mergeMetric struct {
		Labels map[string]string
		value  float64
	}
)

// NewMetricsHub initializes a new MetricsHub instance.
func NewMetricsHub(config *MetricsHubConfig) *MetricsHub {
	reg := prometheus.NewRegistry()
	prometheus.WrapRegistererWith(prometheus.Labels{
		"service_name": config.ServiceName,
		"type":         golangType,
	}, reg).MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	hub := &MetricsHub{
		config:               config,
		registry:             reg,
		metricsRegistrations: make(map[string]*MetricRegistration),
		httpStats:            make(map[httpStatsKey]*HTTPStat),
	}

	if !hub.config.DisableFixedLabels {
		hub.fixedLabels = hub.getFixedLabels()
	}
	if hub.config.ExcludedHttpPath == nil {
		hub.config.ExcludedHttpPath = make([]string, 0)
	}
	if !hub.config.DisableDefaultExcludedHttpPath {
		hub.config.ExcludedHttpPath = append(hub.config.ExcludedHttpPath, defaultExcludedHttpPath...)
	}
	hub.httpMetrics = hub.newHTTPMetrics()

	go hub.run()

	return hub
}

func (hub *MetricsHub) IsExcludedHttpPath(path string) bool {
	if hub.config.ExcludedHttpPath == nil {
		return false
	}
	return slices.Contains(hub.config.ExcludedHttpPath, path)
}

func (hub *MetricsHub) getFixedLabels() prometheus.Labels {
	if hub.fixedLabels != nil {
		return hub.fixedLabels
	}

	labels := prometheus.Labels{
		"service_name": hub.config.ServiceName,
		"type":         defaultType,
	}

	if hub.config.EnableHostNameLabel {
		if hub.config.HostName == "" {
			hostname, _ := os.Hostname()
			labels["host_name"] = hostname
		} else {
			labels["host_name"] = hub.config.HostName
		}
	}

	if hub.config.Labels != nil {
		// override the default labels
		for k, v := range hub.config.Labels {
			labels[k] = v
		}
	}

	hub.fixedLabels = labels
	return labels
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

// RegisterMetric registers a new metric with the hub.
func (hub *MetricsHub) RegisterMetric(reg *MetricRegistration) error {
	if _, exists := hub.metricsRegistrations[reg.Name]; exists {
		return fmt.Errorf("metric %s already exists", reg.Name)
	}
	if !hub.config.DisableFixedLabels {
		for k := range hub.fixedLabels {
			if !slices.Contains(reg.LabelKeys, k) {
				reg.LabelKeys = append(reg.LabelKeys, k)
			}
		}
	}
	hub.metricsRegistrations[reg.Name] = reg

	var collector prometheus.Collector
	switch reg.Type {
	case MetricTypeGaugeVec:
		collector = hub.NewGaugeVec(reg.Name, reg.Help, reg.LabelKeys)
	case MetricTypeCounterVec:
		collector = hub.NewCounterVec(reg.Name, reg.Help, reg.LabelKeys)
	case MetricTypeHistogramVec:
		collector = hub.NewHistogramVec(reg.Name, reg.Help, reg.LabelKeys, reg.HistogramBuckets)
	case MetricTypeSummaryVec:
		collector = hub.NewSummaryVec(reg.Name, reg.Help, reg.LabelKeys, reg.SummaryObjectives)
	default:
		return fmt.Errorf("unsupported metric type: %s", reg.Type)
	}

	reg.collector = collector

	return nil
}

func (hub *MetricsHub) GetCollector(name string) prometheus.Collector {
	return hub.metricsRegistrations[name].collector
}

// HTTPHandler returns an HTTP handler for the metrics endpoint.
func (hub *MetricsHub) HTTPHandler() http.Handler {
	return promhttp.HandlerFor(hub.registry, promhttp.HandlerOpts{})
}

// CurrentMetrics returns a snapshot of all custom metrics registered with the hub.
func (hub *MetricsHub) CurrentMetrics() []string {
	var metricNames []string
	for name := range hub.metricsRegistrations {
		metricNames = append(metricNames, name)
	}
	return metricNames
}

// UpdateMetrics allows dynamic updates to a specific metric by its name.
// Labels are optional and only used for *Vec types.
func (hub *MetricsHub) UpdateMetrics(name string, value float64, labels map[string]string) error {
	metricReg, exists := hub.metricsRegistrations[name]
	if !exists {
		return nil // RequestMetric not found
	}
	if !hub.config.DisableFixedLabels {
		for k, v := range hub.fixedLabels {
			if _, exists := labels[k]; !exists {
				labels[k] = v
			}
		}
	}

	switch m := metricReg.collector.(type) {
	case *prometheus.GaugeVec:
		m.With(labels).Set(value)
	case *prometheus.CounterVec:
		if value > 1 {
			m.With(labels).Add(value)
		} else {
			m.With(labels).Inc()
		}
	case *prometheus.SummaryVec:
		m.With(labels).Observe(value)
	case *prometheus.HistogramVec:
		m.With(labels).Observe(value)
	default:
		return fmt.Errorf("BUG: unsupported metric type: %T", m)
	}

	return nil
}

// IncMetrics increments a metric by 1.
// It only works for GaugeVec and CounterVec, other types will return an error.
func (hub *MetricsHub) IncMetrics(name string, labels map[string]string) error {
	metricReg, exists := hub.metricsRegistrations[name]
	if !exists {
		return nil // RequestMetric not found
	}
	if !hub.config.DisableFixedLabels {
		for k, v := range hub.fixedLabels {
			if _, exists := labels[k]; !exists {
				labels[k] = v
			}
		}
	}

	switch m := metricReg.collector.(type) {
	case *prometheus.GaugeVec:
		m.With(labels).Inc()
	case *prometheus.CounterVec:
		m.With(labels).Inc()
	default:
		return fmt.Errorf("BUG: unsupported metric type for inc: %T", m)
	}

	return nil
}

// DecMetrics decrements a metric by 1.
// It only works for GaugeVec, other types will return an error.
func (hub *MetricsHub) DecMetrics(name string, labels map[string]string) error {
	metricReg, exists := hub.metricsRegistrations[name]
	if !exists {
		return nil // RequestMetric not found
	}
	if !hub.config.DisableFixedLabels {
		for k, v := range hub.fixedLabels {
			if _, exists := labels[k]; !exists {
				labels[k] = v
			}
		}
	}

	switch m := metricReg.collector.(type) {
	case *prometheus.GaugeVec:
		m.With(labels).Dec()
	default:
		return fmt.Errorf("BUG: unsupported metric type for dec: %T", m)
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

// NotifyMessage sends a message to backend, for now, we only support Slack.
// So be sure to set the webhook URL if you want to receive notifications.
func (hub *MetricsHub) NotifyMessage(msg string) error {
	return notifyMessage(hub.config, msg)
}

// NotifyResult sends a result to backend, for now, we only support Slack.
// Use this to form the notification message nicely.
func (hub *MetricsHub) NotifyResult(result *Result) error {
	return notifyResult(hub.config, result)
}

func (hub *MetricsHub) CollectMergedMetrics(name string, mergedLabels []string) error {
	reg, exists := hub.metricsRegistrations[name]
	if !exists {
		return errors.New("metric not found")
	}
	err := hub.mergeMetrics(reg, mergedLabels)
	return err
}

func (hub *MetricsHub) groupMetrics(reg *MetricRegistration, mergedLabels []string) ([]mergeMetric, error) {
	compositeLabelKeys := make([]string, 0)
	for _, labelKey := range reg.LabelKeys {
		isMergeKey := false
		for _, mergeLabelKey := range mergedLabels {
			if labelKey == mergeLabelKey {
				isMergeKey = true
				break
			}
		}

		if !isMergeKey {
			compositeLabelKeys = append(compositeLabelKeys, labelKey)
		}
	}

	mfs := make(chan prometheus.Metric)
	go func() {
		reg.collector.Collect(mfs)
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

		compositeKeyParts := make([]string, 0, len(compositeLabelKeys))
		metricLabels := m.GetLabel()
		mergedMetric := false
		for i := range metricLabels {
			if metricLabels[i].GetValue() == MergedLabelValue {
				mergedMetric = true
			}
		}
		if mergedMetric {
			continue
		}

		newLabels := make(map[string]string)
		for _, label := range compositeLabelKeys {
			for i := range metricLabels {
				if metricLabels[i].GetName() == label {
					compositeKeyParts = append(compositeKeyParts, metricLabels[i].GetValue())
					newLabels[label] = metricLabels[i].GetValue()
					break
				}
			}
		}
		compositeKey := strings.Join(compositeKeyParts, ",")

		value := hub.getMetricValue(m, reg.Type)
		if _, exists := groupedValues[compositeKey]; !exists {
			groupedValues[compositeKey] = value
			groupedLabels[compositeKey] = newLabels
		} else {
			groupedValues[compositeKey] += value
		}
	}

	var mergedMetrics []mergeMetric
	for key, groupLabels := range groupedLabels {
		for _, mergeLabel := range mergedLabels {
			groupLabels[mergeLabel] = MergedLabelValue
		}
		value := groupedValues[key]
		mergedMetrics = append(mergedMetrics, mergeMetric{
			Labels: groupLabels,
			value:  value,
		})
	}
	return mergedMetrics, nil
}

func (hub *MetricsHub) mergeMetrics(reg *MetricRegistration, mergedLabels []string) error {
	switch m := reg.collector.(type) {
	case *prometheus.GaugeVec:
		mergedMetrics, err := hub.groupMetrics(reg, mergedLabels)
		if err != nil {
			return err
		}
		for _, mergedMetric := range mergedMetrics {
			m.With(mergedMetric.Labels).Set(mergedMetric.value)
		}
	case *prometheus.CounterVec:
		mergedMetrics, err := hub.groupMetrics(reg, mergedLabels)
		if err != nil {
			return err
		}
		for _, mergedMetric := range mergedMetrics {
			m.With(mergedMetric.Labels).Add(mergedMetric.value)
		}
	case *prometheus.SummaryVec:
		mergedMetrics, err := hub.groupMetrics(reg, mergedLabels)
		if err != nil {
			return err
		}
		for _, mergedMetric := range mergedMetrics {
			m.With(mergedMetric.Labels).Observe(mergedMetric.value)
		}
	case *prometheus.HistogramVec:
		mergedMetrics, err := hub.groupMetrics(reg, mergedLabels)
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

func (hub *MetricsHub) getMetricValue(m *dto.Metric, metricType MetricType) float64 {
	switch metricType {
	case MetricTypeGaugeVec:
		return m.GetGauge().GetValue()
	case MetricTypeCounterVec:
		return m.GetCounter().GetValue()
	case MetricTypeSummaryVec:
		return m.GetSummary().GetSampleSum()
	case MetricTypeHistogramVec:
		return m.GetHistogram().GetSampleSum()
	default:
		return 0
	}
}
