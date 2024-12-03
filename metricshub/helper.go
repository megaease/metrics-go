package metricshub

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	counterMap   = make(map[string]*prometheus.CounterVec)
	gaugeMap     = make(map[string]*prometheus.GaugeVec)
	histogramMap = make(map[string]*prometheus.HistogramVec)
	summaryMap   = make(map[string]*prometheus.SummaryVec)
	lock         = sync.Mutex{}
)

var (
	validMetric = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)
	validLabel  = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

// NewCounterVec creates a counter metric vec.
func (hub *MetricsHub) NewCounterVec(name string, help string, labels []string) *prometheus.CounterVec {
	lock.Lock()
	defer lock.Unlock()

	metricName, err := getAndValidate(name, labels)
	if err != nil {
		return nil
	}

	if m, find := counterMap[metricName]; find {
		return m
	}

	counterMap[metricName] = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: metricName,
			Help: help,
		},
		labels,
	)
	hub.registry.MustRegister(counterMap[metricName])

	return counterMap[metricName]
}

// NewGaugeVec creates a gauge metric vec.
func (hub *MetricsHub) NewGaugeVec(name string, help string, labels []string) *prometheus.GaugeVec {
	lock.Lock()
	defer lock.Unlock()

	metricName, err := getAndValidate(name, labels)
	if err != nil {
		return nil
	}

	if m, find := gaugeMap[metricName]; find {
		return m
	}
	gaugeMap[metricName] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: metricName,
			Help: help,
		},
		labels,
	)
	hub.registry.MustRegister(gaugeMap[metricName])

	return gaugeMap[metricName]
}

// NewHistogramVec creates a Histogram metric vec.
// Export more opts if needed in future.
func (hub *MetricsHub) NewHistogramVec(name, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
	lock.Lock()
	defer lock.Unlock()

	metricName, err := getAndValidate(name, labels)
	if err != nil {
		return nil
	}

	if m, find := histogramMap[metricName]; find {
		return m
	}
	histogramMap[metricName] = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    metricName,
			Help:    help,
			Buckets: buckets,
		},
		labels,
	)
	hub.registry.MustRegister(histogramMap[metricName])

	return histogramMap[metricName]
}

// NewSummaryVec creates a Summary metric vec.
// Export more opts if needed in future.
func (hub *MetricsHub) NewSummaryVec(name, help string, labels []string, objectives map[float64]float64) *prometheus.SummaryVec {
	lock.Lock()
	defer lock.Unlock()

	metricName, err := getAndValidate(name, labels)
	if err != nil {
		return nil
	}

	if m, find := summaryMap[metricName]; find {
		return m
	}
	summaryMap[metricName] = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       metricName,
			Help:       help,
			Objectives: objectives,
		},
		labels,
	)
	hub.registry.MustRegister(summaryMap[metricName])

	return summaryMap[metricName]
}

func getAndValidate(name string, labels []string) (string, error) {
	if !ValidateMetricName(name) {
		return "", fmt.Errorf("invalid metric name: %s", name)
	}

	for _, l := range labels {
		if !ValidateLabelName(l) {
			return "", fmt.Errorf("invalid label name: %s", l)
		}
	}
	return name, nil
}

// ValidateMetricName checks if the metric name is valid
func ValidateMetricName(name string) bool {
	return validMetric.MatchString(name)
}

// ValidateLabelName checks if the label name is valid
func ValidateLabelName(label string) bool {
	return validLabel.MatchString(label)
}

// DefaultDurationBuckets returns default duration buckets in milliseconds
func DefaultDurationBuckets() []float64 {
	return []float64{10, 50, 100, 200, 400, 800, 1000, 2000, 4000, 8000}
}

// DefaultBodySizeBuckets returns default body size buckets in bytes
func DefaultBodySizeBuckets() []float64 {
	return prometheus.ExponentialBucketsRange(200, 400000, 10)
}

// DefaultObjectives returns default summary objectives
func DefaultObjectives() map[float64]float64 {
	return map[float64]float64{
		0.25: 0.1,
		0.5:  0.05,
		0.75: 0.01,
		0.9:  0.005,
		0.95: 0.001,
		0.99: 0.0001,
	}
}
