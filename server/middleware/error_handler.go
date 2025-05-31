// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package middleware

import (
	"maps"
	"net/http"
	"net/http/httptest"
	"time"

	"codeberg.org/pixivfe/pixivfe/audit"
	"codeberg.org/pixivfe/pixivfe/server/requestcontext"
	"codeberg.org/pixivfe/pixivfe/server/routes"
	"codeberg.org/pixivfe/pixivfe/server/utils"
)

// CatchError is a middleware that wraps HTTP handlers that return an error.
//
// It buffers the response using a httptest.ResponseRecorder. If the handler returns
// an error, the buffered response is discarded and the error is stored into the request
// context. Otherwise, the buffered response is copied to the real ResponseWriter.
//
// This pattern ensures that nothing is written to the client until we know the handler
// succeeded. It also avoids the complexity of manually backing up and restoring headers.
func CatchError(handler func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create a recorder to capture the handler's output.
		recorder := httptest.NewRecorder()

		// Execute the handler, capturing any error.
		if err := handler(recorder, r); err != nil {
			// On error, store it in the request context.
			requestcontext.FromRequest(r).RequestError = err
			// (Do not flush the buffered response to the client.)
			return
		}

		// If no error occurred, copy the buffered response headers, code, and body.
		maps.Copy(w.Header(), recorder.Header())
		w.WriteHeader(recorder.Code)

		if _, err := recorder.Body.WriteTo(w); err != nil {
			audit.GlobalAuditor.Logger.Errorln("Failed to write response body:", err)
		}
	}
}

// HandleError is a middleware that wraps an http.Handler.
//
// It handles both error processing and request logging. It uses httptest.NewRecorder
// to capture the response, handles any errors by rendering an error page, and logs
// all application responses via package audit.
//
// If requestcontext.FromRequest(r).RequestError is set (either by CatchError
// or by a specific handler like the NotFoundHandler), HandleError renders an
// error page.
//
// The HTTP status code for the error page is taken from
// requestcontext.FromRequest(r).StatusCode if it's already an error code (>=400);
// otherwise, it defaults to http.StatusInternalServerError.
//
// If no RequestError is set, the buffered response from the recorder is copied
// to the actual ResponseWriter.
func HandleError(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Start timing and create recorder to capture response
		start := time.Now()
		recorder := httptest.NewRecorder()

		// Execute the wrapped handler (e.g., router dispatch, CatchError, NotFoundHandler)
		// with the recorder.
		next.ServeHTTP(recorder, r)

		ctx := requestcontext.FromRequest(r)

		// Check if an error was signaled in the RequestContext.
		if ctx.RequestError != nil {
			// An error occurred. Determine the correct status code.
			// If the StatusCode in context is not already an error code (e.g., it's still 200 OK
			// despite RequestError being set), then it's an unhandled internal error.
			if ctx.StatusCode < http.StatusBadRequest {
				ctx.StatusCode = http.StatusInternalServerError
			}
			// Write the determined status code and render the generic error page.
			w.WriteHeader(ctx.StatusCode)
			routes.ErrorPage(w, r) // ErrorPage uses ctx.RequestError and ctx.StatusCode
		} else {
			// No error was signaled in RequestContext.
			// Assume the handler executed successfully and wrote its response to the recorder.
			// Copy the buffered response (headers, status code, body) to the real ResponseWriter.
			ctx.StatusCode = recorder.Code // Ensure ctx.StatusCode reflects the actual written code for logging.
			maps.Copy(w.Header(), recorder.Header())
			w.WriteHeader(recorder.Code)

			if _, err := recorder.Body.WriteTo(w); err != nil {
				audit.GlobalAuditor.Logger.Errorln("Failed to write response body:", err)
			}
		}

		// Log the application response if not excluded
		// ctx.StatusCode has been set to the final status code, either from recorder.Code
		// or from the error handling path.
		if !audit.ShouldSkipServerLogging(r.URL.Path) {
			span := audit.Span{
				Component:  audit.ComponentServer,
				Duration:   time.Since(start),
				RequestID:  ctx.RequestID,
				Method:     r.Method,
				URL:        utils.Origin(r) + r.URL.String(),
				StatusCode: ctx.StatusCode,
				Error:      ctx.RequestError,
			}

			audit.GlobalAuditor.LogAndRecord(span)
		}
	})
}
