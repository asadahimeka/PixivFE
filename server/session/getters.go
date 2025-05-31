// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package session

import (
	"fmt"
	"net/http"
	"net/url"

	"codeberg.org/pixivfe/pixivfe/config"
)

// GetUserToken retrieves an authentication token for
// the pixiv API from the request's 'pixivfe-Token' cookie.
func GetUserToken(r *http.Request) string {
	return GetCookie(r, Cookie_Token)
}

// GetImageProxy returns the content proxy URL for i.pximg.net content.
//
// The proxy URL is retrieved from cookies if available, otherwise falls back
// to the default configuration.
func GetImageProxy(r *http.Request) url.URL {
	return getProxy(r, Cookie_ImageProxy, config.GlobalConfig.ContentProxies.Image)
}

// GetStaticProxy returns the content proxy URL for s.pximg.net content.
//
// The proxy URL is retrieved from cookies if available, otherwise falls back
// to the default configuration.
func GetStaticProxy(r *http.Request) url.URL {
	return getProxy(r, Cookie_StaticProxy, config.GlobalConfig.ContentProxies.Static)
}

// GetUgoiraProxy returns the content proxy URL for ugoira.com content.
//
// The proxy URL is retrieved from cookies if available, otherwise falls back
// to the default configuration.
func GetUgoiraProxy(r *http.Request) url.URL {
	return getProxy(r, Cookie_UgoiraProxy, config.GlobalConfig.ContentProxies.Ugoira)
}

// GetOrigin extracts the scheme and host from a URL to form its origin.
//
// Examples:
// -	"https://example.com/path" returns "https://example.com"
//
// Returns an empty string if either scheme or host is missing.
func GetOrigin(u url.URL) string {
	if u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// getProxy retrieves a content proxy URL from a cookieName.
//
// If the cookie value is present but fails to parse, the provided
// defaultProxy is returned.
func getProxy(r *http.Request, cookieName CookieName, defaultProxy url.URL) url.URL {
	value := GetCookie(r, cookieName)

	if value == "" {
		return defaultProxy
	}

	proxyURL, err := url.Parse(value)
	if err != nil {
		return defaultProxy
	}

	return *proxyURL
}

// GetProxyPrefix constructs a complete proxy prefix from a URL object.
//
// It combines the URL's authority (scheme + host) with its path component.
// /
// Examples:
//   - URL "https://proxy.com/img" returns "https://proxy.com/img"
//   - URL with path only "/proxy" returns "/proxy"
//
// Returns an error if the URL format is invalid (has scheme but no host or vice versa).
func GetProxyPrefix(proxy url.URL) string {
	authority := urlAuthority(proxy)

	// If authority is already a path (i.e. a path-only proxy), just return as-is
	if proxy.Scheme == "" {
		return authority
	}

	return authority + proxy.Path
}

// urlAuthority extracts the authority component from a URL.
//
// The function enforces URL format consistency: a URL must either have
// both scheme and host components, or have neither.
//
// Examples:
//   - "https://example.com/path" returns "https://example.com"
//   - "/local/path" returns "/local/path"
//   - "https://" returns error (missing host)
//   - "//example.com" returns error (missing scheme)
//
// iacore: still cannot believe Go doesn't have this function built-in. if stability is their goal, they really don't have the incentive to add useful, crucial features
func urlAuthority(u url.URL) string {
	if (u.Scheme != "") != (u.Host != "") {
		panic(fmt.Errorf("url must have both scheme and authority or neither: %s\nplease correct this in your proxy list", u.String()))
	}
	if u.Scheme == "" { // Handle path-only proxies (e.g., /proxy/i.pximg.net)
		return u.Path
	}
	return u.Scheme + "://" + u.Host
}
