package main

import (
	"fmt"
	"time"
)

const timeLayout = "2006-01-02 15:04:05"

// now formats the current time in the required "yyyy-mm-dd hh:mm:ss" layout.
func now() string {
	return time.Now().Format(timeLayout)
}

// humanDecimal formats a byte count into MB or GB using powers of 1000,
// matching the "[~0.06MB]" style shown in the project description.
func humanDecimal(n int64) string {
	const (
		mb = 1000.0 * 1000.0
		gb = mb * 1000.0
	)
	f := float64(n)
	if f >= gb {
		return fmt.Sprintf("~%.2fGB", f/gb)
	}
	return fmt.Sprintf("~%.2fMB", f/mb)
}

// humanBinary formats a byte count into a value + IEC unit (KiB/MiB/GiB)
// using powers of 1024, used by the progress bar.
func humanBinary(n float64) string {
	const (
		kib = 1024.0
		mib = kib * 1024.0
		gib = mib * 1024.0
	)
	switch {
	case n >= gib:
		return fmt.Sprintf("%.2f GiB", n/gib)
	case n >= mib:
		return fmt.Sprintf("%.2f MiB", n/mib)
	default:
		return fmt.Sprintf("%.2f KiB", n/kib)
	}
}

// humanSpeed formats bytes-per-second into a human readable rate.
func humanSpeed(bytesPerSec float64) string {
	const (
		kib = 1024.0
		mib = kib * 1024.0
		gib = mib * 1024.0
	)
	switch {
	case bytesPerSec >= gib:
		return fmt.Sprintf("%.2f GiB/s", bytesPerSec/gib)
	case bytesPerSec >= mib:
		return fmt.Sprintf("%.2f MiB/s", bytesPerSec/mib)
	case bytesPerSec >= kib:
		return fmt.Sprintf("%.2f KiB/s", bytesPerSec/kib)
	default:
		return fmt.Sprintf("%.0f B/s", bytesPerSec)
	}
}

// humanETA formats a remaining duration compactly, e.g. "0s", "42s", "1m03s".
func humanETA(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	secs := int(d.Round(time.Second).Seconds())
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	m := secs / 60
	s := secs % 60
	if m < 60 {
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	h := m / 60
	m = m % 60
	return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
}
