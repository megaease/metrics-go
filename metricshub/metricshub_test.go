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
	gaugeVec := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "example_counter",
			Help: "An example counter vector",
		},
		[]string{"cluster", "node", "spec"},
	)

	gaugeVec.WithLabelValues("1001", "ds01", "4090").Set(8)
	gaugeVec.WithLabelValues("1001", "ds02", "4060").Set(7)
	gaugeVec.WithLabelValues("1001", "ds03", "4060").Set(8)
	gaugeVec.WithLabelValues("1002", "ds04", "4060").Set(8)

	metricsHub := NewMetricsHub(&MetricsHubConfig{
		ServiceName: "test",
		HostName:    "test",
		Labels:      nil,
	})

	printGaugeVec(gaugeVec)

	err := metricsHub.MergeMetric(gaugeVec, []string{"cluster", "spec"}, []string{"node"})
	if err != nil {
		t.Errorf("merge metric failed: %v", err)
	}

	fmt.Printf("--- after merge ---\n")

	printGaugeVec(gaugeVec)

	mfs := make(chan prometheus.Metric)
	go func() {
		gaugeVec.Collect(mfs)
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
	gaugeVec := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "example_counter",
			Help: "An example counter vector",
		},
		[]string{"cluster", "node", "spec"},
	)

	gaugeVec.WithLabelValues("1001", "ds01", "4090").Set(8)
	gaugeVec.WithLabelValues("1001", "ds02", "4060").Set(7)
	gaugeVec.WithLabelValues("1001", "ds03", "4060").Set(8)
	gaugeVec.WithLabelValues("1001", "ds04", "4090").Set(8)

	metricsHub := NewMetricsHub(&MetricsHubConfig{
		ServiceName: "test",
		HostName:    "test",
		Labels:      nil,
	})

	printGaugeVec(gaugeVec)

	err := metricsHub.MergeMetric(gaugeVec, []string{"spec"}, []string{"node", "cluster"})
	if err != nil {
		t.Errorf("merge metric failed: %v", err)
	}

	fmt.Printf("--- after merge ---\n")

	printGaugeVec(gaugeVec)

	mfs := make(chan prometheus.Metric)
	go func() {
		gaugeVec.Collect(mfs)
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
