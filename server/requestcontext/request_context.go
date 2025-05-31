// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
Package requestcontext provides per-request state management for HTTP handlers.

iacore: this package is separate because Go disallows cyclic import. pain
*/
package requestcontext

import (
	"context"
	"net/http"

	"codeberg.org/pixivfe/pixivfe/server/template/commondata"
	"github.com/oklog/ulid/v2"
)

// RequestContext carries request-scoped data through the middleware chain.
//
// This data survives the entire lifetime of a single HTTP request and is safe
// for concurrent access from multiple goroutines handling the same request.
type RequestContext struct {
	// ULID for tracing requests.
	RequestID string

	// Holds any critical error encountered during request processing.
	//
	// Automatically populated by middleware.CatchError when handlers return errors,
	// which interrupts normal response handling and renders an error page instead.
	RequestError error

	// HTTP status code to be sent in the response. Defaults to 200 OK.
	StatusCode int

	CommonData commondata.PageCommonData
}

// requestContextKeyType defines a unique type for a RequestContext key.
type requestContextKeyType struct{}

// requestContextKey is a unique key used to access RequestContext
// values from a context.Context.
var requestContextKey = requestContextKeyType{}

// WithRequestContext initializes a new request context and attaches it to
// the parent context.
//
// This is called once per request, first in the middleware chain (see main.go).
func WithRequestContext(ctx context.Context, r *http.Request, generateToken commondata.LinkTokenGenerator) context.Context {
	rc := RequestContext{
		RequestID:  ulid.Make().String(),
		StatusCode: http.StatusOK,
	}
	commondata.PopulatePageCommonData(r, &rc.CommonData, generateToken)

	return context.WithValue(ctx, requestContextKey, &rc)
}

// FromContext extracts the RequestContext from a context, always returning
// a valid pointer.
//
// If no context is found, returns a zero-value instance to prevent nil pointer
// dereferences in handlers.
// FromContext extracts the RequestContext from a context, always returning a valid pointer.
func FromContext(ctx context.Context) *RequestContext {
	if v := ctx.Value(requestContextKey); v != nil {
		if rc, ok := v.(*RequestContext); ok {
			return rc
		}
	}
	// Return a zero-value instance with an empty PageCommonData to prevent nil panics
	return &RequestContext{CommonData: commondata.PageCommonData{}}
}

// FromRequest is a convenience wrapper for extracting RequestContext
// directly from HTTP requests.
//
// Prefer this in handlers that have access to the *http.Request object.
func FromRequest(r *http.Request) *RequestContext {
	return FromContext(r.Context())
}
