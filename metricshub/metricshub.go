package metricshub

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/megaease/metrics-go/conf"
	"github.com/megaease/metrics-go/notify"
	dto "github.com/prometheus/client_model/go"

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

	// MergedLabelValue is the placeholder value for merged metrics.
	MergedLabelValue = "MERGED_LABEL"

	MetricTypeGaugeVec     MetricType = "GaugeVec"
	MetricTypeCounterVec   MetricType = "CounterVec"
	MetricTypeSummaryVec   MetricType = "SummaryVec"
	MetricTypeHistogramVec MetricType = "HistogramVec"
)

// MetricsHub wraps Prometheus metrics for monitoring purposes.
type (
	MetricsHubConfig conf.Config
	MetricsHub       struct {
		config               *MetricsHubConfig
		registry             *prometheus.Registry
		metricsRegistrations map[string]*MetricRegistration
		httpMetrics          *httpRequestMetrics
		httpStats            map[httpStatsKey]*HTTPStat
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
	hub := &MetricsHub{
		config:               config,
		registry:             prometheus.DefaultRegisterer.(*prometheus.Registry),
		metricsRegistrations: make(map[string]*MetricRegistration),
		httpStats:            make(map[httpStatsKey]*HTTPStat),
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

// RegisterMetric registers a new metric with the hub.
func (hub *MetricsHub) RegisterMetric(reg *MetricRegistration) error {
	if _, exists := hub.metricsRegistrations[reg.Name]; exists {
		return fmt.Errorf("metric %s already exists", reg.Name)
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
	return promhttp.Handler()
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
	return notify.NotifyMessage(hub.config.ToConfig(), msg)
}

// NotifyResult sends a result to backend, for now, we only support Slack.
// Use this to form the notification message nicely.
func (hub *MetricsHub) NotifyResult(result *notify.Result) error {
	return notify.NotifyResult(hub.config.ToConfig(), result)
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

func (cfg *MetricsHubConfig) ToConfig() *conf.Config {
	return (*conf.Config)(cfg)
}
