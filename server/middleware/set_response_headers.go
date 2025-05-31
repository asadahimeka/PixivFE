// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package middleware

import (
	"net/http"
	"strings"

	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/server/session"
)

const cloudflareChallengesURL string = "https://challenges.cloudflare.com"

// SetResponseHeaders adds default headers to HTTP responses.
func SetResponseHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract origins from content proxy URLs
		imageOrigin := session.GetOrigin(session.GetImageProxy(r))
		staticOrigin := session.GetOrigin(session.GetStaticProxy(r))
		ugoiraOrigin := session.GetOrigin(session.GetUgoiraProxy(r))

		// Prepare CSP origins
		// TrimSpace() allows us simple string concatenation for imgOrigins
		imgOrigins := strings.TrimSpace(imageOrigin + " " + staticOrigin)

		// mediaOrigins will only ever be ugoiraOrigin or empty, so just assign directly
		mediaOrigins := ugoiraOrigin

		setCacheHeaders(w.Header(), r.URL.Path)
		setSecurityHeaders(w.Header(), imgOrigins, mediaOrigins, r.URL.Path)

		next.ServeHTTP(w, r)
	})
}

// setCacheHeaders sets appropriate cache control headers for static assets.
func setCacheHeaders(headers http.Header, path string) {
	// Default cache duration of 1 week
	cacheDuration := "max-age=604800"

	// Longer caching for fonts and icons (1 month)
	if strings.HasPrefix(path, "/fonts/") || strings.HasPrefix(path, "/icons/") {
		cacheDuration = "max-age=2592000"
	}

	// JavaScript and CSS get a moderate cache time (1 week)
	if strings.HasPrefix(path, "/js/") || strings.HasPrefix(path, "/css/") {
		cacheDuration = "max-age=604800"
	}

	// Images can be cached for 2 weeks
	if strings.HasPrefix(path, "/img/") {
		cacheDuration = "max-age=1209600"
	}

	headers.Set("Cache-Control", cacheDuration)
}

// setSecurityHeaders sets security headers.
//
// We don't set an HSTS header to avoid conflicts with reverse proxies.
// Generally better for reverse proxies to manage HSTS instead.
func setSecurityHeaders(headers http.Header, imgOrigins, mediaOrigins, path string) {
	// Standard security headers
	headers.Set("Referrer-Policy", "no-referrer")
	headers.Set("X-Frame-Options", "DENY")
	headers.Set("X-Content-Type-Options", "nosniff")
	// Setting CORP headers by default breaks compatibility with most external
	// image proxy servers, which don't set the required CORP cross-origin header
	// headers.Set("Cross-Origin-Embedder-Policy", "require-corp")
	// headers.Set("Cross-Origin-Opener-Policy", "same-origin")
	// headers.Set("Cross-Origin-Resource-Policy", "same-site")

	// Instance information
	headers.Set("Pixivfe-Version", config.GlobalConfig.Instance.Version)
	headers.Set("Pixivfe-Revision", config.GlobalConfig.Instance.Revision)
	headers.Set("X-Powered-By", "hatsune miku")

	setPermissionsPolicy(headers)
	setContentSecurityPolicy(headers, imgOrigins, mediaOrigins, path)
}

func setPermissionsPolicy(headers http.Header) {
	features := []string{
		"accelerometer=()",
		"ambient-light-sensor=()",
		"battery=()",
		"camera=()",
		"display-capture=()",
		"document-domain=()",
		"encrypted-media=()",
		"execution-while-not-rendered=()",
		"execution-while-out-of-viewport=()",
		"geolocation=()",
		"gyroscope=()",
		"magnetometer=()",
		"microphone=()",
		"midi=()",
		"navigation-override=()",
		"payment=()",
		"publickey-credentials-get=()",
		"screen-wake-lock=()",
		"sync-xhr=()",
		"usb=()",
		"web-share=()",
		"xr-spatial-tracking=()",
	}
	headers.Set("Permissions-Policy", strings.Join(features, ", "))
}

func setContentSecurityPolicy(headers http.Header, imgOrigins, mediaOrigins, path string) {
	if strings.HasPrefix(path, "/diagnostics") {
		return
	}

	// Build CSP directives that can have a dynamic number of origins
	imgDirective := "img-src 'self' data:"
	if imgOrigins != "" {
		imgDirective += " " + imgOrigins
	}

	mediaDirective := "media-src 'self'"
	if mediaOrigins != "" {
		mediaDirective += " " + mediaOrigins
	}

	scriptSrcDirective := "script-src 'self'"
	if config.GlobalConfig.Limiter.DetectionMethod == config.TurnstileDetectionMethod {
		scriptSrcDirective += " " + cloudflareChallengesURL
	}

	// Adding challenges.cloudflare.com to script-src-elem isn't noted in the official docs but it's required
	scriptSrcElemDirective := "script-src-elem 'self'"
	if config.GlobalConfig.Limiter.DetectionMethod == config.TurnstileDetectionMethod {
		scriptSrcElemDirective += " " + cloudflareChallengesURL
	}

	frameSrcDirective := "frame-src 'self'"
	if config.GlobalConfig.Limiter.DetectionMethod == config.TurnstileDetectionMethod {
		frameSrcDirective += " " + cloudflareChallengesURL
	}

	directives := []string{
		"base-uri 'self'",
		"default-src 'self'",
		scriptSrcDirective,
		scriptSrcElemDirective,
		"style-src 'self' 'unsafe-inline'",
		imgDirective,
		mediaDirective,
		"font-src 'self'",
		"connect-src 'self'",
		"form-action 'self'",
		frameSrcDirective,
		"frame-ancestors 'none'",
	}

	csp := strings.Join(directives, "; ") + ";"
	headers.Set("Content-Security-Policy", csp)
}
