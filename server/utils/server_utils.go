// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package utils

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Timing struct {
	Name        string
	Duration    time.Duration
	Description string
}

// Timings provides thread-safe access to a Timing slice.
//
// Only needed when timings from multiple goroutines is desired;
// use utils.AddServerTimingHeader directly otherwise.
type Timings struct {
	mu      sync.Mutex
	entries []Timing
}

func NewTimings() *Timings {
	return &Timings{
		entries: make([]Timing, 0),
	}
}

func (t *Timings) Append(name string, duration time.Duration, desc string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.entries = append(t.entries, Timing{
		Name:        name,
		Duration:    duration,
		Description: desc,
	})
}

func (t *Timings) WriteHeaders(w http.ResponseWriter) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, entry := range t.entries {
		w.Header().Add("Server-Timing", fmt.Sprintf(
			"%s;dur=%.0f;desc=\"%s\"",
			entry.Name,
			float64(entry.Duration.Milliseconds()),
			entry.Description,
		))
	}
}

// AddServerTimingHeader writes a Server-Timing header.
func AddServerTimingHeader(w http.ResponseWriter, name string, duration time.Duration, description string) {
	w.Header().Add("Server-Timing", fmt.Sprintf(
		"%s;dur=%.0f;desc=\"%s\"",
		name,
		float64(duration.Milliseconds()),
		description,
	))
}
