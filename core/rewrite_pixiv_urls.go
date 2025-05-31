// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"bytes"
	"net/http"
	"net/url"

	"codeberg.org/pixivfe/pixivfe/server/session"
)

// RewriteContentURLs replaces content URLs with their proxied equivalents.
//
// It handles pre-escaped URL patterns (with escaped forward slashes).
//
// Parameters:
//   - r: The HTTP request containing proxy configuration in cookies
//   - s: The byte slice containing URLs to be proxied
//
// Returns the processed byte slice.
func RewriteContentURLs(r *http.Request, data []byte) []byte {
	replacements := map[string]url.URL{
		`https:\/\/i.pximg.net`: session.GetImageProxy(r),  // Pre-escaped pattern
		`https:\/\/s.pximg.net`: session.GetStaticProxy(r), // Pre-escaped pattern
	}

	result := data

	for pattern, proxy := range replacements {
		proxyPrefix := session.GetProxyPrefix(proxy)
		result = bytes.ReplaceAll(result, []byte(pattern), []byte(proxyPrefix))
	}

	return result
}

// RewriteContentURLsNoEscape replaces content URLs with their proxied equivalents.
//
// Unlike ProxyContentURLs, it handles non-escaped URL patterns.
//
// Parameters:
//   - r: The HTTP request containing proxy configuration in cookies
//   - s: The byte slice containing URLs to be proxied
//
// Returns the processed byte slice.
func RewriteContentURLsNoEscape(r *http.Request, data []byte) []byte {
	replacements := map[string]url.URL{
		"https://i.pximg.net": session.GetImageProxy(r),  // Non-escaped pattern
		"https://s.pximg.net": session.GetStaticProxy(r), // Non-escaped pattern
	}

	result := data

	for pattern, proxy := range replacements {
		proxyPrefix := session.GetProxyPrefix(proxy)
		result = bytes.ReplaceAll(result, []byte(pattern), []byte(proxyPrefix))
	}

	return result
}
