// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package audit

import (
	"time"
)

// Span represents the logging format used by PixivFE.
type Span struct {
	Component  string
	Duration   time.Duration
	RequestID  string
	Method     string
	URL        string
	StatusCode int
	Error      error
	Body       []byte // Body is not logged as is; only for response saving

	responseFilename string // responseFilename logs the filename of a saved response
}
