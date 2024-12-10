package metricshub

import (
	"github.com/prometheus/client_golang/prometheus"
	"os"
)

type (
	// httpRequestMetrics is the statistics tool for HTTP traffic.
	httpRequestMetrics struct {
		TotalRequests               *prometheus.CounterVec
		TotalResponses              *prometheus.CounterVec
		TotalErrorRequests          *prometheus.CounterVec
		RequestsDuration            prometheus.ObserverVec
		RequestSizeBytes            prometheus.ObserverVec
		ResponseSizeBytes           prometheus.ObserverVec
		RequestsDurationPercentage  prometheus.ObserverVec
		RequestSizeBytesPercentage  prometheus.ObserverVec
		ResponseSizeBytesPercentage prometheus.ObserverVec

		M1            *prometheus.GaugeVec
		M5            *prometheus.GaugeVec
		M15           *prometheus.GaugeVec
		M1Err         *prometheus.GaugeVec
		M5Err         *prometheus.GaugeVec
		M15Err        *prometheus.GaugeVec
		M1ErrPercent  *prometheus.GaugeVec
		M5ErrPercent  *prometheus.GaugeVec
		M15ErrPercent *prometheus.GaugeVec
		Min           *prometheus.GaugeVec
		Max           *prometheus.GaugeVec
		Mean          *prometheus.GaugeVec
		P25           *prometheus.GaugeVec
		P50           *prometheus.GaugeVec
		P75           *prometheus.GaugeVec
		P95           *prometheus.GaugeVec
		P98           *prometheus.GaugeVec
		P99           *prometheus.GaugeVec
		P999          *prometheus.GaugeVec
		ReqSize       *prometheus.GaugeVec
		RespSize      *prometheus.GaugeVec
	}
)

// newHTTPMetrics create the HttpServerMetrics.
func (hub *MetricsHub) newHTTPMetrics() *httpRequestMetrics {
	commonLabels := prometheus.Labels{
		"service_name": hub.config.ServiceName,
	}
	httpserverLabels := []string{"service_name", "method", "path"}
	if hub.config.EnableHostNameLabel {
		httpserverLabels = append(httpserverLabels, "host_name")
		if hub.config.HostName == "" {
			hostname, _ := os.Hostname()
			hub.config.HostName = hostname
		}
		commonLabels["host_name"] = hub.config.HostName
	}
	if hub.config.Labels != nil {
		for k, v := range hub.config.Labels {
			commonLabels[k] = v
			httpserverLabels = append(httpserverLabels, k)
		}
	}

	return &httpRequestMetrics{
		TotalRequests: hub.NewCounterVec(
			"service_total_requests",
			"the total count of http requests",
			httpserverLabels).MustCurryWith(commonLabels),
		TotalResponses: hub.NewCounterVec(
			"service_total_responses",
			"the total count of http responses",
			httpserverLabels).MustCurryWith(commonLabels),
		TotalErrorRequests: hub.NewCounterVec(
			"service_total_error_requests",
			"the total count of http error requests",
			httpserverLabels).MustCurryWith(commonLabels),
		RequestsDuration: hub.NewHistogramVec(
			"service_requests_duration",
			"request processing duration histogram of a backend",
			httpserverLabels,
			DefaultDurationBuckets()).MustCurryWith(commonLabels),
		RequestSizeBytes: hub.NewHistogramVec(
			"service_requests_size_bytes",
			"a histogram of the total size of the request to a backend. Includes body",
			httpserverLabels,
			DefaultBodySizeBuckets()).MustCurryWith(commonLabels),
		ResponseSizeBytes: hub.NewHistogramVec(
			"service_responses_size_bytes",
			"a histogram of the total size of the returned response body from a backend",
			httpserverLabels,
			DefaultBodySizeBuckets()).MustCurryWith(commonLabels),
		RequestsDurationPercentage: hub.NewSummaryVec(
			"service_requests_duration_percentage",
			"request processing duration summary of a backend",
			httpserverLabels,
			DefaultObjectives()).MustCurryWith(commonLabels),
		RequestSizeBytesPercentage: hub.NewSummaryVec(
			"service_requests_size_bytes_percentage",
			"a summary of the total size of the request to a backend. Includes body",
			httpserverLabels,
			DefaultObjectives()).MustCurryWith(commonLabels),
		ResponseSizeBytesPercentage: hub.NewSummaryVec(
			"service_responses_size_bytes_percentage",
			"a summary of the total size of the returned response body from a backend",
			httpserverLabels,
			DefaultObjectives()).MustCurryWith(commonLabels),
		M1: hub.NewGaugeVec(
			"service_m1",
			"QPS (exponentially-weighted moving average) in last 1 minute",
			httpserverLabels).MustCurryWith(commonLabels),
		M5: hub.NewGaugeVec(
			"service_m5",
			"QPS (exponentially-weighted moving average) in last 5 minute",
			httpserverLabels).MustCurryWith(commonLabels),
		M15: hub.NewGaugeVec(
			"service_m15",
			"QPS (exponentially-weighted moving average) in last 15 minute",
			httpserverLabels).MustCurryWith(commonLabels),
		M1Err: hub.NewGaugeVec(
			"service_m1_err",
			"QPS (exponentially-weighted moving average) in last 1 minute",
			httpserverLabels).MustCurryWith(commonLabels),
		M5Err: hub.NewGaugeVec(
			"service_m5_err",
			"QPS (exponentially-weighted moving average) in last 5 minute",
			httpserverLabels).MustCurryWith(commonLabels),
		M15Err: hub.NewGaugeVec(
			"service_m15_err",
			"QPS (exponentially-weighted moving average) in last 15 minute",
			httpserverLabels).MustCurryWith(commonLabels),
		M1ErrPercent: hub.NewGaugeVec(
			"service_m1_err_percent",
			"error percentage in last 1 minute",
			httpserverLabels).MustCurryWith(commonLabels),
		M5ErrPercent: hub.NewGaugeVec(
			"service_m5_err_percent",
			"error percentage in last 5 minute",
			httpserverLabels).MustCurryWith(commonLabels),
		M15ErrPercent: hub.NewGaugeVec(
			"service_m15_err_percent",
			"error percentage in last 15 minute",
			httpserverLabels).MustCurryWith(commonLabels),
		Min: hub.NewGaugeVec(
			"service_min",
			"The http-request minimal execution duration in milliseconds",
			httpserverLabels).MustCurryWith(commonLabels),
		Max: hub.NewGaugeVec(
			"service_max",
			"The http-request maximal execution duration in milliseconds",
			httpserverLabels).MustCurryWith(commonLabels),
		Mean: hub.NewGaugeVec(
			"service_mean",
			"The http-request mean execution duration in milliseconds",
			httpserverLabels).MustCurryWith(commonLabels),
		P25: hub.NewGaugeVec(
			"service_p25",
			"TP25: The processing time for 25% of the requests, in milliseconds.",
			httpserverLabels).MustCurryWith(commonLabels),
		P50: hub.NewGaugeVec(
			"service_p50",
			"TP50: The processing time for 50% of the requests, in milliseconds.",
			httpserverLabels).MustCurryWith(commonLabels),
		P75: hub.NewGaugeVec(
			"service_p75",
			"TP75: The processing time for 75% of the requests, in milliseconds.",
			httpserverLabels).MustCurryWith(commonLabels),
		P95: hub.NewGaugeVec(
			"service_p95",
			"TP95: The processing time for 95% of the requests, in milliseconds.",
			httpserverLabels).MustCurryWith(commonLabels),
		P98: hub.NewGaugeVec(
			"service_p98",
			"TP98: The processing time for 98% of the requests, in milliseconds.",
			httpserverLabels).MustCurryWith(commonLabels),
		P99: hub.NewGaugeVec(
			"service_p99",
			"TP99: The processing time for 99% of the requests, in milliseconds.",
			httpserverLabels).MustCurryWith(commonLabels),
		P999: hub.NewGaugeVec(
			"service_p999",
			"TP999: The processing time for 99.9% of the requests, in milliseconds.",
			httpserverLabels).MustCurryWith(commonLabels),
		ReqSize: hub.NewGaugeVec(
			"service_req_size",
			"The total size of the http requests in this statistic window",
			httpserverLabels).MustCurryWith(commonLabels),
		RespSize: hub.NewGaugeVec(
			"service_resp_size",
			"The total size of the http responses in this statistic window",
			httpserverLabels).MustCurryWith(commonLabels),
	}
}

func (m *httpRequestMetrics) exportPrometheusMetricsForTicker(status *Status, method, path string) {
	labels := prometheus.Labels{
		"method": method,
		"path":   path,
	}

	m.M1.With(labels).Set(status.M1)
	m.M5.With(labels).Set(status.M5)
	m.M15.With(labels).Set(status.M15)
	m.M1Err.With(labels).Set(status.M1Err)
	m.M5Err.With(labels).Set(status.M5Err)
	m.M15Err.With(labels).Set(status.M15Err)
	m.M1ErrPercent.With(labels).Set(status.M1ErrPercent)
	m.M5ErrPercent.With(labels).Set(status.M5ErrPercent)
	m.M15ErrPercent.With(labels).Set(status.M15ErrPercent)
	m.Min.With(labels).Set(float64(status.Min))
	m.Max.With(labels).Set(float64(status.Max))
	m.Mean.With(labels).Set(float64(status.Mean))
	m.P25.With(labels).Set(status.P25)
	m.P50.With(labels).Set(status.P50)
	m.P75.With(labels).Set(status.P75)
	m.P95.With(labels).Set(status.P95)
	m.P98.With(labels).Set(status.P98)
	m.P99.With(labels).Set(status.P99)
	m.P999.With(labels).Set(status.P999)
	m.ReqSize.With(labels).Set(float64(status.ReqSize))
	m.RespSize.With(labels).Set(float64(status.RespSize))
}

func (m *httpRequestMetrics) exportPrometheusMetricsForRequestMetric(stat *RequestMetric, method, path string) {
	labels := prometheus.Labels{
		"method": method,
		"path":   path,
	}

	m.TotalRequests.With(labels).Inc()
	m.TotalResponses.With(labels).Inc()
	if stat.StatusCode >= 400 {
		m.TotalErrorRequests.With(labels).Inc()
	}
	m.RequestsDuration.With(labels).Observe(float64(stat.Duration.Milliseconds()))
	m.RequestSizeBytes.With(labels).Observe(float64(stat.ReqSize))
	m.ResponseSizeBytes.With(labels).Observe(float64(stat.RespSize))
	m.RequestsDurationPercentage.With(labels).Observe(float64(stat.Duration.Milliseconds()))
	m.RequestSizeBytesPercentage.With(labels).Observe(float64(stat.ReqSize))
	m.ResponseSizeBytesPercentage.With(labels).Observe(float64(stat.RespSize))
}
