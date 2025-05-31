// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package routes

import (
	"fmt"
	"net/http"

	"github.com/goccy/go-json"
	"github.com/soluble-ai/go-jnode"

	"codeberg.org/pixivfe/pixivfe/audit"
	"codeberg.org/pixivfe/pixivfe/server/template"
)

func Diagnostics(w http.ResponseWriter, r *http.Request) error {
	return template.RenderHTML(w, r, Data_diagnostics{})
}

func ResetDiagnosticsData(w http.ResponseWriter, r *http.Request) {
	audit.RecordedRequestSpans = audit.RecordedRequestSpans[:0]
	w.WriteHeader(http.StatusOK)
}

// formatSpanSummary creates a LogLine string from audit.Span
func formatSpanSummary(span audit.Span) string {
	durationSec := span.Duration.Seconds()
	return fmt.Sprintf("%s - %s - %d - %.3fs",
		span.Component,
		span.Method,
		span.StatusCode,
		durationSec,
	)
}

func DiagnosticsData(w http.ResponseWriter, _ *http.Request) error {
	data := jnode.NewArrayNode()
	for _, span := range audit.RecordedRequestSpans {
		bytes, err := json.Marshal(span)
		if err != nil {
			return err
		}
		obj, err := jnode.FromJSON(bytes)
		if err != nil {
			return err
		}
		// Replace error object with message string
		if span.Error != nil {
			obj.Put("Error", span.Error.Error())
		}
		obj.Put("LogLine", formatSpanSummary(span))
		data.Append(obj)
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(200)
	return json.NewEncoder(w).Encode(data)
}
