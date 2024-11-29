package metricshub

import (
	"fmt"
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"

	"github.com/prometheus/client_golang/prometheus"
)

func printGaugeVec(m *prometheus.GaugeVec) {
	mfs := make(chan prometheus.Metric)
	go func() {
		m.Collect(mfs)
		close(mfs)
	}()

	for mf := range mfs {
		m := &dto.Metric{}
		err := mf.Write(m)
		if err != nil {
			return
		}

		fmt.Printf("%f ", m.GetGauge().GetValue())
		labels := m.GetLabel()
		for _, label := range labels {
			fmt.Printf("%s=%s ", label.GetName(), label.GetValue())
		}
		fmt.Println()
	}
}

func TestMergedMetrics1(t *testing.T) {
	metricsHub := NewMetricsHub(&MetricsHubConfig{
		ServiceName: "test",
		HostName:    "test",
		Labels:      nil,
	})

	metricsHub.RegisterMetric(&MetricRegistration{
		Name:               "example_gauge",
		Help:               "An example gauge vector",
		Type:               MetricTypeGaugeVec,
		LabelKeys:          []string{"cluster", "node", "spec"},
		AutoMerge:          true,
		AutoMergeLabelKeys: []string{"node"},
	})

	metricsHub.UpdateMetrics("example_gauge", 8.0, map[string]string{
		"cluster": "1001",
		"node":    "ds01",
		"spec":    "4090",
	})
	metricsHub.UpdateMetrics("example_gauge", 7.0, map[string]string{
		"cluster": "1001",
		"node":    "ds02",
		"spec":    "4060",
	})
	metricsHub.UpdateMetrics("example_gauge", 8.0, map[string]string{
		"cluster": "1001",
		"node":    "ds03",
		"spec":    "4060",
	})
	metricsHub.UpdateMetrics("example_gauge", 8.0, map[string]string{
		"cluster": "1002",
		"node":    "ds04",
		"spec":    "4060",
	})

	collector := metricsHub.GetCollector("example_gauge")

	printGaugeVec(collector.(*prometheus.GaugeVec))

	mfs := make(chan prometheus.Metric)
	go func() {
		collector.Collect(mfs)
		close(mfs)
	}()

	length := 0
	for mf := range mfs {
		m := &dto.Metric{}
		err := mf.Write(m)
		assert.NoError(t, err)
		length++
	}
	assert.Equal(t, 7, length)
}

func TestMergedMetrics2(t *testing.T) {
	metricsHub := NewMetricsHub(&MetricsHubConfig{
		ServiceName: "test",
		HostName:    "test",
		Labels:      nil,
	})

	metricsHub.RegisterMetric(&MetricRegistration{
		Name:               "example_gauge",
		Help:               "An example gauge vector",
		Type:               MetricTypeGaugeVec,
		LabelKeys:          []string{"cluster", "node", "spec"},
		AutoMerge:          true,
		AutoMergeLabelKeys: []string{"node", "cluster"},
	})

	metricsHub.UpdateMetrics("example_gauge", 8.0, map[string]string{
		"cluster": "1001",
		"node":    "ds01",
		"spec":    "4090",
	})
	metricsHub.UpdateMetrics("example_gauge", 7.0, map[string]string{
		"cluster": "1001",
		"node":    "ds02",
		"spec":    "4060",
	})
	metricsHub.UpdateMetrics("example_gauge", 8.0, map[string]string{
		"cluster": "1001",
		"node":    "ds03",
		"spec":    "4060",
	})
	metricsHub.UpdateMetrics("example_gauge", 8.0, map[string]string{
		"cluster": "1002",
		"node":    "ds04",
		"spec":    "4060",
	})

	collector := metricsHub.GetCollector("example_gauge")

	printGaugeVec(collector.(*prometheus.GaugeVec))

	mfs := make(chan prometheus.Metric)
	go func() {
		collector.Collect(mfs)
		close(mfs)
	}()

	length := 0
	for mf := range mfs {
		m := &dto.Metric{}
		err := mf.Write(m)
		assert.NoError(t, err)
		length++
	}
	assert.Equal(t, 6, length)
}
