package metricshub

import (
	"math"
	"sync"
	"sync/atomic"
	"time"

	metrics "github.com/rcrowley/go-metrics"

	"github.com/megaease/metrics-go/helper"
)

type (
	// HTTPStat is the statistics tool for HTTP traffic.
	HTTPStat struct {
		mutex sync.RWMutex

		count  uint64
		rate1  metrics.EWMA
		rate5  metrics.EWMA
		rate15 metrics.EWMA

		errCount  uint64
		errRate1  metrics.EWMA
		errRate5  metrics.EWMA
		errRate15 metrics.EWMA

		total uint64
		min   uint64
		max   uint64

		durationSampler *helper.DurationSampler

		reqSize  uint64
		respSize uint64

		cc *helper.HTTPStatusCodeCounter
	}

	// RequestMetric is the package of statistics at once.
	RequestMetric struct {
		StatusCode int
		Duration   time.Duration
		ReqSize    uint64
		RespSize   uint64
	}

	// StatisticsMetric contains request metrics.
	StatisticsMetric struct {
		Count uint64  `json:"count"`
		M1    float64 `json:"m1"`
		M5    float64 `json:"m5"`
		M15   float64 `json:"m15"`

		ErrCount uint64  `json:"errCount"`
		M1Err    float64 `json:"m1Err"`
		M5Err    float64 `json:"m5Err"`
		M15Err   float64 `json:"m15Err"`

		M1ErrPercent  float64 `json:"m1ErrPercent"`
		M5ErrPercent  float64 `json:"m5ErrPercent"`
		M15ErrPercent float64 `json:"m15ErrPercent"`

		Min  uint64 `json:"min"`
		Max  uint64 `json:"max"`
		Mean uint64 `json:"mean"`

		P25  float64 `json:"p25"`
		P50  float64 `json:"p50"`
		P75  float64 `json:"p75"`
		P95  float64 `json:"p95"`
		P98  float64 `json:"p98"`
		P99  float64 `json:"p99"`
		P999 float64 `json:"p999"`

		ReqSize  uint64 `json:"reqSize"`
		RespSize uint64 `json:"respSize"`
	}

	// StatusCodeMetric is the metrics of http status code.
	StatusCodeMetric struct {
		Code  int    `json:"code"`
		Count uint64 `json:"cnt"`
	}

	// Status contains all status generated by HTTPStat.
	Status struct {
		StatisticsMetric
		Codes map[int]uint64 `json:"codes"`
	}
)

func (m *RequestMetric) isErr() bool {
	return m.StatusCode >= 400
}

// NewHTTPStat creates an HTTPStat.
func NewHTTPStat() *HTTPStat {
	hs := &HTTPStat{
		rate1:  metrics.NewEWMA1(),
		rate5:  metrics.NewEWMA5(),
		rate15: metrics.NewEWMA15(),

		errRate1:  metrics.NewEWMA1(),
		errRate5:  metrics.NewEWMA5(),
		errRate15: metrics.NewEWMA15(),

		min:             math.MaxUint64,
		durationSampler: helper.NewDurationSampler(),

		cc: helper.New(),
	}

	return hs
}

// Stat stats the ctx.
func (hs *HTTPStat) Stat(m *RequestMetric) {
	// Note: although this is a data update operation, we are using the RLock here,
	// which means goroutines can execute this function concurrently, and contentions
	// are handled by the atomic operations for each item.
	//
	// This lock is only a mutex for the 'Status' function below.
	hs.mutex.RLock()
	defer hs.mutex.RUnlock()

	atomic.AddUint64(&hs.count, 1)
	hs.rate1.Update(1)
	hs.rate5.Update(1)
	hs.rate15.Update(1)

	if m.isErr() {
		atomic.AddUint64(&hs.errCount, 1)
		hs.errRate1.Update(1)
		hs.errRate5.Update(1)
		hs.errRate15.Update(1)
	}

	duration := uint64(m.Duration.Milliseconds())
	atomic.AddUint64(&hs.total, duration)
	for {
		minN := atomic.LoadUint64(&hs.min)
		if duration >= minN {
			break
		}
		if atomic.CompareAndSwapUint64(&hs.min, minN, duration) {
			break
		}
	}
	for {
		maxN := atomic.LoadUint64(&hs.max)
		if duration <= maxN {
			break
		}
		if atomic.CompareAndSwapUint64(&hs.max, maxN, duration) {
			break
		}
	}

	hs.durationSampler.Update(m.Duration)

	atomic.AddUint64(&hs.reqSize, m.ReqSize)
	atomic.AddUint64(&hs.respSize, m.RespSize)

	hs.cc.Count(m.StatusCode)
}

// Status returns HTTPStat Status, It assumes it is called every five seconds.
// https://github.com/rcrowley/go-metrics/blob/3113b8401b8a98917cde58f8bbd42a1b1c03b1fd/ewma.go#L98-L99
func (hs *HTTPStat) Status() *Status {
	hs.mutex.Lock()
	defer hs.mutex.Unlock()

	hs.rate1.Tick()
	hs.rate5.Tick()
	hs.rate15.Tick()
	hs.errRate1.Tick()
	hs.errRate5.Tick()
	hs.errRate15.Tick()

	m1, m5, m15 := hs.rate1.Rate(), hs.rate5.Rate(), hs.rate15.Rate()
	m1Err, m5Err, m15Err := hs.errRate1.Rate(), hs.errRate5.Rate(), hs.errRate15.Rate()
	m1ErrPercent, m5ErrPercent, m15ErrPercent := 0.0, 0.0, 0.0
	if m1 > 0 {
		m1ErrPercent = m1Err / m1
	}
	if m5 > 0 {
		m1ErrPercent = m5Err / m5
	}
	if m15 > 0 {
		m1ErrPercent = m15Err / m15
	}

	percentiles := hs.durationSampler.Percentiles()
	hs.durationSampler.Reset()

	codes := hs.cc.Codes()
	hs.cc.Reset()

	mean, minN := uint64(0), uint64(0)
	if hs.count > 0 {
		mean = hs.total / hs.count
		minN = hs.min
	}
	status := &Status{
		StatisticsMetric: StatisticsMetric{
			Count: hs.count,
			M1:    m1,
			M5:    m5,
			M15:   m15,

			ErrCount: hs.errCount,
			M1Err:    m1Err,
			M5Err:    m5Err,
			M15Err:   m15Err,

			M1ErrPercent:  m1ErrPercent,
			M5ErrPercent:  m5ErrPercent,
			M15ErrPercent: m15ErrPercent,

			Min:  minN,
			Mean: mean,
			Max:  hs.max,

			P25:  percentiles[0],
			P50:  percentiles[1],
			P75:  percentiles[2],
			P95:  percentiles[3],
			P98:  percentiles[4],
			P99:  percentiles[5],
			P999: percentiles[6],

			ReqSize:  hs.reqSize,
			RespSize: hs.respSize,
		},

		Codes: codes,
	}

	return status
}
