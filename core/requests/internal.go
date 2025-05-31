// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
Internal logic
*/
package requests

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"math/rand"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"codeberg.org/pixivfe/pixivfe/audit"
	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/i18n"
	"codeberg.org/pixivfe/pixivfe/server/requestcontext"
	"codeberg.org/pixivfe/pixivfe/server/tokenmanager"
	"codeberg.org/pixivfe/pixivfe/server/utils"
)

var ErrUnsupportedPayloadType = errors.New("unsupported payload type")

// handleRequest handles HTTP requests using the provided RequestOptions.
func handleRequest(ctx context.Context, opts RequestOptions) (*SimpleHTTPResponse, error) {
	var (
		req               *http.Request
		reqBody           io.Reader
		contentTypeHeader string
	)

	tokenManager := config.GlobalConfig.TokenManager.TokenManager

	userToken := opts.Cookies["PHPSESSID"]

	token, err := retrieveToken(tokenManager, userToken)

	// Determine special cases for token
	if userToken == RandomToken {
		token = createRandomToken()
	} else if err != nil {
		return nil, err
	}

	// For GET requests, determine if caching should be used
	if opts.Method == http.MethodGet {
		policy := determineCachePolicy(opts.URL, userToken, opts.IncomingHeaders)
		if policy.ShouldUseCache && policy.CachedResponse != nil {
			return policy.CachedResponse, nil
		}
	}

	if opts.Method == http.MethodPost {
		switch v := opts.Payload.(type) {
		case string:
			reqBody = bytes.NewBufferString(v)
			contentTypeHeader = opts.ContentType
		case map[string]string:
			body, formContentType, err := createMultipartFormData(v)
			if err != nil {
				return nil, err
			}

			reqBody = body
			contentTypeHeader = formContentType
		default:
			return nil, ErrUnsupportedPayloadType
		}
	}

	req, err = http.NewRequestWithContext(ctx, opts.Method, opts.URL, reqBody)
	if err != nil {
		return nil, i18n.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("User-Agent", config.GetRandomUserAgent())
	req.Header.Add("Accept-Language", config.GlobalConfig.Request.AcceptLanguage)

	// Add PHPSESSID only if userToken is not NoToken
	if userToken != NoToken {
		req.AddCookie(&http.Cookie{
			Name:  "PHPSESSID",
			Value: token.Value,
		})
	}

	// Add any other cookies
	for k, v := range opts.Cookies {
		if k != "PHPSESSID" || userToken != NoToken {
			req.AddCookie(&http.Cookie{Name: k, Value: v})
		}
	}

	if opts.Method == http.MethodPost {
		req.Header.Add("X-Csrf-Token", opts.CSRF)

		if contentTypeHeader != "" {
			req.Header.Set("Content-Type", contentTypeHeader)
		}
	}

	resp, err := makeRequest(ctx, req, opts.URL)
	if err != nil {
		// If making the request itself failed, don't mark the token provided by tokenManager as timed out
		return nil, err
	}

	// Handle the response based on the status code
	if resp.StatusCode == http.StatusOK {
		// Mark the token as good if the response is OK
		tokenManager.MarkTokenStatus(token, tokenmanager.Good)

		if opts.Method == http.MethodGet {
			// manageCaching will either return a cached response (if available)
			// or the new response after optionally storing it.
			return manageCaching(opts.URL, userToken, opts.IncomingHeaders, resp), nil
		}

		return resp, nil
	}

	// Handle non-OK status codes
	err = i18n.Errorf("HTTP status code: %d", resp.StatusCode)

	// Mark the token provided by tokenManager as timed out if the
	// request succeeded, but returned a non-OK response
	tokenManager.MarkTokenStatus(token, tokenmanager.TimedOut)

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context canceled: %w", ctx.Err())
	default:
		return nil, err
	}
}

// makeRequest executes the HTTP request and processes the response.
func makeRequest(
	ctx context.Context,
	req *http.Request,
	url string,
) (*SimpleHTTPResponse, error) {
	start := time.Now()

	resp, err := utils.HTTPClient.Do(req)
	if err != nil {
		return nil, i18n.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, i18n.Errorf("failed to read response body: %w", err)
	}

	span := audit.Span{
		Component:  audit.ComponentUpstream,
		Duration:   time.Since(start),
		RequestID:  requestcontext.FromContext(ctx).RequestID,
		Method:     req.Method,
		URL:        url,
		StatusCode: resp.StatusCode,
		Error:      err,
		Body:       body,
	}
	audit.GlobalAuditor.LogAndRecord(span)

	return &SimpleHTTPResponse{
		StatusCode: resp.StatusCode,
		Body:       body,
	}, nil
}

// retrieveToken obtains a valid token for the request.
func retrieveToken(tokenManager *tokenmanager.TokenManager, userToken string) (*tokenmanager.Token, error) {
	if userToken != "" {
		return &tokenmanager.Token{Value: userToken}, nil
	}

	token := tokenManager.GetToken()
	if token == nil {
		tokenManager.ResetAllTokens()

		return nil, i18n.Errorf(
			`All tokens (%d) are timed out, resetting all tokens to their initial good state.
Consider providing additional tokens in PIXIVFE_TOKEN or reviewing token management configuration.
Please refer the following documentation for additional information:
- https://pixivfe-docs.pages.dev/hosting/api-authentication/`,
			len(config.GlobalConfig.Basic.Token),
		)
	}

	return token, nil
}

// createRandomToken generates an arbitrary Token with
// a random 33-character lowercase string value.
//
// ref: https://codeberg.org/kita/px-api-docs/src/commit/92a71331bb/README.md#authorization
//
// #nosec:G404 - the generated token doesn't need to be cryptographically secure.
func createRandomToken() *tokenmanager.Token {
	const (
		letters = "abcdefghijklmnopqrstuvwxyz"
		length  = 33
	)

	builder := strings.Builder{}
	builder.Grow(length)

	for range length {
		builder.WriteByte(letters[rand.Intn(len(letters))])
	}

	return &tokenmanager.Token{
		Value: builder.String(),
	}
}

// createMultipartFormData constructs multipart form data from a map of fields.
//
// It is used to prepare data for POST requests that require multipart encoding.
func createMultipartFormData(fields map[string]string) (*bytes.Buffer, string, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	defer writer.Close()

	for k, v := range fields {
		if err := writer.WriteField(k, v); err != nil {
			return nil, "", fmt.Errorf("failed to write multipart form field %q: %w", k, err)
		}
	}

	return body, writer.FormDataContentType(), nil
}

func proxyRequest(w http.ResponseWriter, r *http.Request) error {
	start := time.Now()

	resp, err := utils.HTTPClient.Do(r)
	if err != nil {
		return i18n.Errorf("failed to proxy request: %w", err)
	}
	defer resp.Body.Close()

	maps.Copy(w.Header(), resp.Header)

	w.WriteHeader(resp.StatusCode)

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		return i18n.Errorf("failed to copy response body: %w", err)
	}

	if !audit.ShouldSkipUpstreamLogging(r.URL.Hostname()) {
		span := audit.Span{
			Component:  audit.ComponentUpstream,
			Duration:   time.Since(start),
			RequestID:  requestcontext.FromContext(r.Context()).RequestID,
			Method:     r.Method,
			URL:        r.URL.String(),
			StatusCode: resp.StatusCode,
			Error:      err,
			Body:       make([]byte, 0),
		}
		audit.GlobalAuditor.LogAndRecord(span)
	}

	return nil
}

// isContextError checks if an error is due to context cancellation or deadline exceeded.
func isContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
