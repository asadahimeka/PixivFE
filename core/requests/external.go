// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
Public functions
*/
package requests

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/tidwall/gjson"
)

const (
	RandomToken string = "RandomToken" // Use a randomly generated token
	NoToken     string = "NoToken"     // Don't set PHPSESSID at all
)

var (
	ErrInvalidJSON              = errors.New("response contained invalid JSON")
	ErrAPIResponseError         = errors.New("API response indicated error")
	ErrIncompatibleRespBody     = errors.New("incompatible response body")
	ErrMissingRequiredPHPSESSID = errors.New("PHPSESSID cookie is required for POST requests")
)

// FetchJSON makes a GET request and validates the response is valid JSON.
//
// Returns the raw JSON response body as byte slice, or an error if the request
// fails or the response is not valid JSON.
func FetchJSON(
	ctx context.Context,
	url string,
	cookies map[string]string,
	incomingHeaders http.Header,
) ([]byte, error) {
	resp, err := PerformGET(ctx, url, cookies, incomingHeaders)
	if err != nil {
		return nil, err
	}

	if !gjson.ValidBytes(resp.Body) {
		return nil, fmt.Errorf("%w: %v", ErrInvalidJSON, resp.Body)
	}

	return resp.Body, nil
}

// FetchJSONBodyField makes a GET request and extracts the "body" field from the JSON response.
//
// Returns the JSON serialized as byte slice of the "body" field, which
// may be any JSON type (object, array, etc.).
//
// Returns an error if:
//   - The request fails
//   - The response contains invalid JSON
//   - The "error" field is a boolean true
//   - The "body" field is missing
func FetchJSONBodyField(
	ctx context.Context,
	url string,
	cookies map[string]string,
	incomingHeaders http.Header,
) ([]byte, error) {
	resp, err := PerformGET(ctx, url, cookies, incomingHeaders)
	if err != nil {
		return nil, err
	}

	if !gjson.ValidBytes(resp.Body) {
		return nil, fmt.Errorf("%w: %v", ErrInvalidJSON, resp.Body)
	}

	result := gjson.ParseBytes(resp.Body)

	if result.Get("error").Bool() {
		message := result.Get("message").String()

		return nil, fmt.Errorf("%w: %s", ErrAPIResponseError, message)
	}

	body := result.Get("body")

	if !body.Exists() {
		return nil, ErrIncompatibleRespBody
	}

	return []byte(body.Raw), nil
}

// PerformGET performs a GET request.
func PerformGET(
	ctx context.Context,
	url string,
	cookies map[string]string,
	incomingHeaders http.Header,
) (*SimpleHTTPResponse, error) {
	return handleRequest(ctx, RequestOptions{
		Method:          http.MethodGet,
		URL:             url,
		Cookies:         cookies,
		IncomingHeaders: incomingHeaders,
	})
}

// PerformGETReader performs a GET request and wraps the returned response.Body in a bytes.Reader.
func PerformGETReader(
	ctx context.Context,
	url string,
	cookies map[string]string,
	incomingHeaders http.Header,
) (io.Reader, error) {
	response, err := PerformGET(ctx, url, cookies, incomingHeaders)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(response.Body), nil
}

// PerformPOST performs a POST request.
func PerformPOST(
	ctx context.Context,
	url string,
	payload any,
	cookies map[string]string,
	csrf string,
	contentType string,
	incomingHeaders http.Header,
) (*SimpleHTTPResponse, error) {
	if cookies == nil || cookies["PHPSESSID"] == "" {
		return nil, ErrMissingRequiredPHPSESSID
	}

	return handleRequest(ctx, RequestOptions{
		Method:          http.MethodPost,
		URL:             url,
		IncomingHeaders: incomingHeaders,
		Payload:         payload,
		Cookies:         cookies,
		CSRF:            csrf,
		ContentType:     contentType,
	})
}

func ProxyHandler(w http.ResponseWriter, r *http.Request, baseURL string, headers map[string]string) error {
	targetURL := fmt.Sprintf("%s/%s?%s", baseURL, r.URL.Path, r.URL.Query().Encode())

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, targetURL, nil)
	if err != nil {
		if isContextError(err) {
			return nil
		}

		return fmt.Errorf("failed to create request for %s: %w", baseURL, err)
	}

	for key, value := range headers {
		req.Header.Add(key, value)
	}

	if err := proxyRequest(w, req); err != nil {
		if isContextError(err) {
			return nil
		}

		return fmt.Errorf("failed to proxy request to %s: %w", baseURL, err)
	}

	return nil
}
