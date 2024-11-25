package helper

import "sync/atomic"

// HTTPStatusCodeCounter is the goroutine unsafe HTTP status code counter.
// It is designed for counting http status code which is 1XX - 5XX,
// So the code range are limited to [0, 999]
type HTTPStatusCodeCounter struct {
	counter []uint64
}

// New creates a HTTPStatusCodeCounter.
func New() *HTTPStatusCodeCounter {
	return &HTTPStatusCodeCounter{
		counter: make([]uint64, 1000),
	}
}

// Count counts a new code.
func (cc *HTTPStatusCodeCounter) Count(code int) {
	if code < 0 || code >= len(cc.counter) {
		// TODO: log? panic?
		return
	}
	atomic.AddUint64(&cc.counter[code], 1)
}

// Reset resets counters of all codes to zero
func (cc *HTTPStatusCodeCounter) Reset() {
	for i := 0; i < len(cc.counter); i++ {
		cc.counter[i] = 0
	}
}

// Codes returns the codes.
func (cc *HTTPStatusCodeCounter) Codes() map[int]uint64 {
	codes := make(map[int]uint64)
	for i, count := range cc.counter {
		if count > 0 {
			codes[i] = count
		}
	}
	return codes
}
