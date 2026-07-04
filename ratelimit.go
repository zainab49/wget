package main

import (
	"io"
	"time"
)

// rateLimitedReader wraps a reader and caps throughput to limit bytes/second.
// It works by handing out a small "bucket" of bytes per time slice and
// sleeping when the caller drains it too quickly.
type rateLimitedReader struct {
	r         io.Reader
	limit     int64 // bytes per second
	allowance float64
	last      time.Time
}

func newRateLimitedReader(r io.Reader, limit int64) io.Reader {
	if limit <= 0 {
		return r
	}
	return &rateLimitedReader{
		r:         r,
		limit:     limit,
		allowance: float64(limit),
		last:      time.Now(),
	}
}

func (rl *rateLimitedReader) Read(p []byte) (int, error) {
	// Never read more than roughly a tenth of a second's worth at a time so
	// the rate stays smooth rather than bursty.
	maxChunk := rl.limit / 10
	if maxChunk < 1 {
		maxChunk = 1
	}
	if int64(len(p)) > maxChunk {
		p = p[:maxChunk]
	}

	// Refill the allowance based on elapsed time.
	now := time.Now()
	elapsed := now.Sub(rl.last).Seconds()
	rl.last = now
	rl.allowance += elapsed * float64(rl.limit)
	if rl.allowance > float64(rl.limit) {
		rl.allowance = float64(rl.limit)
	}

	// If we cannot afford a single byte yet, sleep until we can.
	if rl.allowance < 1 {
		sleep := time.Duration((1 - rl.allowance) / float64(rl.limit) * float64(time.Second))
		time.Sleep(sleep)
		rl.allowance = 1
		rl.last = time.Now()
	}

	if int64(len(p)) > int64(rl.allowance) {
		p = p[:int64(rl.allowance)]
	}

	n, err := rl.r.Read(p)
	rl.allowance -= float64(n)
	return n, err
}
