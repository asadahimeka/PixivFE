// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package audit

import (
	"os"
	"path"
	"slices"
	"strings"

	"codeberg.org/pixivfe/pixivfe/config"
	"github.com/oklog/ulid/v2"
	"go.uber.org/zap"
)

const (
	ComponentUpstream       string      = "Upstream"
	ComponentServer         string      = "Server"
	responseFilePermissions os.FileMode = 0o600
)

var (
	staticSkippedPathPrefixes = []string{"/img/", "/css/", "/js/", "/diagnostics"}
	devSkippedPathPrefixes    = []string{"/proxy/s.pximg.net/", "/proxy/i.pximg.net/"}
	staticSkippedHostnames    = []string{"s.pximg.net", "i.pximg.net"}
)

// RecordedRequestSpans stores a slice of Span objects.
var RecordedRequestSpans = []Span{}

// ShouldSkipServerLogging determines if a request should bypass the logging middleware.
func ShouldSkipServerLogging(path string) bool {
	for _, prefix := range staticSkippedPathPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	if config.GlobalConfig.Development.InDevelopment {
		for _, prefix := range devSkippedPathPrefixes {
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
	}

	return false
}

// ShouldSkipUpstreamLogging determines if requests to this host should skip logging.
func ShouldSkipUpstreamLogging(host string) bool {
	return slices.Contains(staticSkippedHostnames, host)
}

func (a *Auditor) LogAndRecord(span Span) {
	a.handleSaveResponseBody(&span)

	// Build log fields from Span
	logFields := []zap.Field{
		zap.String("component", span.Component),
		zap.String("method", span.Method),
		zap.String("url", span.URL),
		zap.Int("status_code", span.StatusCode),
		zap.Duration("duration", span.Duration),
		zap.String("request_id", span.RequestID),
	}

	// Add error field if non-nil
	if span.Error != nil {
		logFields = append(logFields, zap.Error(span.Error))
	}

	// Add response filename if present
	if span.responseFilename != "" {
		logFields = append(logFields, zap.String("response_filename", span.responseFilename))
	}

	// Obtain a desugared logger
	logger := a.Logger.Desugar()

	// Log at Error level if error exists OR status code >= 400,
	// otherwise log normally at Info level
	if span.Error != nil || span.StatusCode >= 400 {
		logger.Error("Failed request", logFields...)
	} else {
		logger.Info("Successful request", logFields...)
	}

	// Record the span if applicable
	if GlobalAuditor.MaxRecorded > 0 {
		if len(RecordedRequestSpans) >= GlobalAuditor.MaxRecorded {
			RecordedRequestSpans = RecordedRequestSpans[1:]
		}

		RecordedRequestSpans = append(RecordedRequestSpans, span)
	}
}

// handleSaveResponseBody saves the response body to a file.
func (a *Auditor) handleSaveResponseBody(span *Span) {
	if span.Component != ComponentUpstream {
		return
	}

	if len(span.Body) == 0 || !GlobalAuditor.SaveResponses {
		return
	}

	id := ulid.Make().String()
	filename := path.Join(config.GlobalConfig.Development.ResponseSaveLocation, id[len(id)-6:])

	if err := os.WriteFile(filename, span.Body, responseFilePermissions); err != nil {
		a.Logger.Error("Failed to save response", err, "request_id", span.RequestID)

		return
	}

	span.responseFilename = filename
}
