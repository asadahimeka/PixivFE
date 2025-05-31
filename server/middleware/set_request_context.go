// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package middleware

import (
	"net/http"

	"codeberg.org/pixivfe/pixivfe/server/middleware/limiter"
	"codeberg.org/pixivfe/pixivfe/server/requestcontext"
)

// WithRequestContext is a middleware that attaches a RequestContext to each HTTP request.
func WithRequestContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a new context with RequestContext attached
		ctxWithRequest := requestcontext.WithRequestContext(r.Context(), r, limiter.GetOrCreateLinkToken)

		// Create new request with the enhanced context and pass to the next handler
		next.ServeHTTP(w, r.WithContext(ctxWithRequest))
	})
}
