// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
Common types
*/
package requests

import (
	"net/http"
)

// SimpleHTTPResponse represents an HTTP response body and
// its associated status code.
type SimpleHTTPResponse struct {
	StatusCode int
	Body       []byte
}

// RequestOptions consolidates all parameters for handleRequest.
type RequestOptions struct {
	Method          string
	URL             string
	Cookies         map[string]string
	IncomingHeaders http.Header
	Payload         any
	CSRF            string
	ContentType     string
}
