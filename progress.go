package main

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// progressBar is an io.Writer that counts bytes copied through it and renders
// a live progress bar on the given output.
type progressBar struct {
	out       io.Writer
	total     int64 // -1 when Content-Length is unknown
	current   int64
	start     time.Time
	lastDraw  time.Time
	barWidth  int
	lastWidth int // width of the previously drawn line, for clean overwrite
}

func newProgressBar(out io.Writer, total int64) *progressBar {
	t := time.Now()
	return &progressBar{
		out:      out,
		total:    total,
		start:    t,
		lastDraw: t.Add(-time.Second), // force an initial draw
		barWidth: 50,
	}
}

func (p *progressBar) Write(b []byte) (int, error) {
	n := len(b)
	p.current += int64(n)
	// Throttle redraws so the terminal is not flooded.
	if time.Since(p.lastDraw) >= 100*time.Millisecond {
		p.render(false)
		p.lastDraw = time.Now()
	}
	return n, nil
}

// render draws the current state. When final is true it draws the completed
// bar and moves to a new line.
func (p *progressBar) render(final bool) {
	elapsed := time.Since(p.start).Seconds()
	if elapsed <= 0 {
		elapsed = 1e-9
	}
	speed := float64(p.current) / elapsed

	var line string
	if p.total > 0 {
		ratio := float64(p.current) / float64(p.total)
		if ratio > 1 {
			ratio = 1
		}
		filled := int(ratio * float64(p.barWidth))
		bar := strings.Repeat("=", filled)
		if filled < p.barWidth {
			bar += ">" + strings.Repeat(" ", p.barWidth-filled-1)
		}

		var eta time.Duration
		if speed > 0 {
			eta = time.Duration(float64(p.total-p.current)/speed) * time.Second
		}

		line = fmt.Sprintf(" %s / %s [%s] %6.2f%% %s %s",
			humanBinary(float64(p.current)),
			humanBinary(float64(p.total)),
			bar,
			ratio*100,
			humanSpeed(speed),
			humanETA(eta),
		)
	} else {
		// Unknown length: no percentage or ETA.
		line = fmt.Sprintf(" %s [%s] %s",
			humanBinary(float64(p.current)),
			strings.Repeat("=", p.barWidth),
			humanSpeed(speed),
		)
	}

	// Pad with spaces to erase any leftovers from a longer previous line.
	pad := ""
	if diff := p.lastWidth - len(line); diff > 0 {
		pad = strings.Repeat(" ", diff)
	}
	p.lastWidth = len(line)

	fmt.Fprintf(p.out, "\r%s%s", line, pad)
	if final {
		fmt.Fprintln(p.out)
	}
}

// Finish draws the completed bar (current is forced to total when known).
func (p *progressBar) Finish() {
	if p.total > 0 {
		p.current = p.total
	}
	p.render(true)
}
