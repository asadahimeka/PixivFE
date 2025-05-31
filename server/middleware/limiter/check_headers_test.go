// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package limiter

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"codeberg.org/pixivfe/pixivfe/audit"
)

func TestBlockedByHeaders(t *testing.T) {
	audit.NewTestingLogger(t)

	tests := []struct {
		name        string
		path        string
		headers     map[string]string
		blockReason string
	}{
		{
			name: "Valid browser request",
			path: "/artworks/20",
			headers: map[string]string{
				"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:130.0) Gecko/20100101 Firefox/130.0",
				"Accept":          "text/html",
				"Accept-Encoding": "gzip, deflate",
				"Accept-Language": "en-US",
				"Sec-Fetch-Mode":  "navigate",
				"Sec-Fetch-Site":  "same-origin",
				"Sec-Fetch-Dest":  "document",
			},
			blockReason: "",
		},
		{
			name: "Missing User-Agent",
			path: "/artworks/20",
			headers: map[string]string{
				"Accept":          "text/html",
				"Accept-Encoding": "gzip, deflate",
				"Accept-Language": "en-US",
				"Sec-Fetch-Mode":  "navigate",
				"Sec-Fetch-Site":  "same-origin",
				"Sec-Fetch-Dest":  "document",
			},
			blockReason: "Blocked by user-agent check",
		},
		{
			name: "Bot User-Agent",
			path: "/artworks/20",
			headers: map[string]string{
				"User-Agent":      "Mozilla/5.0 (Linux; Android 6.0.1; Nexus 5X Build/MMB29P) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/41.0.2272.96 Mobile Safari/537.36 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
				"Accept":          "text/html",
				"Accept-Encoding": "gzip, deflate",
				"Accept-Language": "en-US",
				"Sec-Fetch-Mode":  "navigate",
				"Sec-Fetch-Site":  "same-origin",
				"Sec-Fetch-Dest":  "document",
			},
			blockReason: "Blocked by user-agent check",
		},
		{
			name: "Wrong Accept for JS file",
			path: "/js/htmx@2.0.4.min.js",
			headers: map[string]string{
				"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:130.0) Gecko/20100101 Firefox/130.0",
				"Accept":          "text/html",
				"Accept-Encoding": "gzip, deflate",
				"Accept-Language": "en-US",
				"Sec-Fetch-Mode":  "navigate",
				"Sec-Fetch-Site":  "same-origin",
				"Sec-Fetch-Dest":  "document",
			},
			blockReason: "Blocked by Accept header - JavaScript file requires JavaScript Accept type",
		},
		{
			name: "Missing Accept-Encoding",
			path: "/artworks/20",
			headers: map[string]string{
				"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:130.0) Gecko/20100101 Firefox/130.0",
				"Accept":          "text/html",
				"Accept-Language": "en-US",
				"Sec-Fetch-Mode":  "navigate",
				"Sec-Fetch-Site":  "same-origin",
				"Sec-Fetch-Dest":  "document",
			},
			blockReason: "Blocked by Accept-Encoding check",
		},
		{
			name: "Missing Accept-Language",
			path: "/artworks/20",
			headers: map[string]string{
				"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:130.0) Gecko/20100101 Firefox/130.0",
				"Accept":          "text/html",
				"Accept-Encoding": "gzip, deflate",
				"Sec-Fetch-Mode":  "navigate",
				"Sec-Fetch-Site":  "same-origin",
				"Sec-Fetch-Dest":  "document",
			},
			blockReason: "Blocked by Accept-Language check",
		},
		{
			name: "Missing all Sec-Fetch headers",
			path: "/artworks/20",
			headers: map[string]string{
				"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:130.0) Gecko/20100101 Firefox/130.0",
				"Accept":          "text/html",
				"Accept-Encoding": "gzip, deflate",
				"Accept-Language": "en-US",
			},
			blockReason: "Missing Sec-Fetch headers: Sec-Fetch-Dest, Sec-Fetch-Mode, Sec-Fetch-Site",
		},
		{
			name: "Missing multiple Sec-Fetch headers",
			path: "/artworks/20",
			headers: map[string]string{
				"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:130.0) Gecko/20100101 Firefox/130.0",
				"Accept":          "text/html",
				"Accept-Encoding": "gzip, deflate",
				"Accept-Language": "en-US",
				"Sec-Fetch-Mode":  "navigate",
			},
			blockReason: "Missing Sec-Fetch headers: Sec-Fetch-Dest, Sec-Fetch-Site",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tt.path, nil)

			// Set headers
			for key, value := range tt.headers {
				r.Header.Set(key, value)
			}

			// Add TLS context for tests that expect Sec-Fetch validation
			if strings.Contains(tt.blockReason, "Sec-Fetch") {
				r.TLS = &tls.ConnectionState{}
			}

			blocked := blockedByHeaders(r)
			if blocked != tt.blockReason {
				t.Errorf("blockedByHeaders() = %v, want %v", blocked, tt.blockReason)
			}
		})
	}
}

func TestCheckUserAgent(t *testing.T) {
	audit.NewTestingLogger(t)

	tests := []struct {
		name        string
		userAgent   string
		blockReason string
	}{
		{
			name:        "Valid browser User-Agent",
			userAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:130.0) Gecko/20100101 Firefox/130.0",
			blockReason: "",
		},
		{
			name:        "Empty User-Agent",
			userAgent:   "",
			blockReason: "Blocked by user-agent check",
		},
		{
			name:        "Bot User-Agent",
			userAgent:   "Mozilla/5.0 (Linux; Android 6.0.1; Nexus 5X Build/MMB29P) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/41.0.2272.96 Mobile Safari/537.36 (compatible; Googlebot/2.1; +http://www.google.com/bot.html",
			blockReason: "Blocked by user-agent check",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("User-Agent", tt.userAgent)

			blocked := checkUserAgent(r)
			if blocked != tt.blockReason {
				t.Errorf("checkUserAgent() = %v, want %v", blocked, tt.blockReason)
			}
		})
	}
}

func TestIsBotUserAgent(t *testing.T) {
	audit.NewTestingLogger(t)

	testCases := []struct {
		name     string
		ua       string
		expected bool
	}{
		{
			name:     "Googlebot Smartphone",
			ua:       "Mozilla/5.0 (Linux; Android 6.0.1; Nexus 5X Build/MMB29P) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/41.0.2272.96 Mobile Safari/537.36 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
			expected: true,
		},
		{
			name:     "Firefox",
			ua:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:130.0) Gecko/20100101 Firefox/130.0",
			expected: false,
		},
		{
			name:     "Chrome",
			ua:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
			expected: false,
		},
		{
			name:     "Empty string",
			ua:       "",
			expected: false, // Empty string is blocked by checkUserAgent, not isBotUserAgent
		},
		{
			name:     "Python requests",
			ua:       "python-requests/2.28.2",
			expected: true,
		},
		{
			name:     "curl",
			ua:       "curl/7.88.1",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isBotUserAgent(tc.ua)
			if result != tc.expected {
				t.Errorf("isBotUserAgent(%q) = %v; want %v", tc.ua, result, tc.expected)
			}
		})
	}
}

func TestCheckAcceptHeader(t *testing.T) {
	audit.NewTestingLogger(t)

	tests := []struct {
		name        string
		path        string
		accept      string
		blockReason string
	}{
		{
			name:        "Valid wildcard accept",
			path:        "/proxy/i.pximg.net/c/1200x1200_90_webp/img-master/img/2007/09/09/22/14/07/20_p0_master1200.jpg",
			accept:      "*/*",
			blockReason: "",
		},
		{
			name:        "Valid JavaScript request",
			path:        "/js/htmx@2.0.4.min.js",
			accept:      "application/javascript",
			blockReason: "",
		},
		{
			name:        "Valid JavaScript alt content type",
			path:        "/js/htmx@2.0.4.min.js",
			accept:      "text/javascript",
			blockReason: "",
		},
		{
			name:        "Invalid JavaScript accept",
			path:        "/js/htmx@2.0.4.min.js",
			accept:      "text/html",
			blockReason: "JavaScript file requires JavaScript Accept type",
		},
		{
			name:        "Valid CSS request",
			path:        "/css/tailwind-style.css",
			accept:      "text/css",
			blockReason: "",
		},
		{
			name:        "Invalid CSS accept",
			path:        "/css/tailwind-style.css",
			accept:      "text/plain",
			blockReason: "CSS file requires text/css Accept type",
		},
		{
			name:        "Valid PNG image request",
			path:        "/img/ai.png",
			accept:      "image/png",
			blockReason: "",
		},
		{
			name:        "Valid JPEG image request",
			path:        "/img/createdwith.jpeg",
			accept:      "image/jpeg",
			blockReason: "",
		},
		{
			name:        "Invalid image accept",
			path:        "/img/ai.png",
			accept:      "text/plain",
			blockReason: "Image file requires image/* Accept type",
		},
		{
			name:        "Valid JSON request",
			path:        "/manifest.json",
			accept:      "application/json",
			blockReason: "",
		},
		{
			name:        "Invalid JSON accept",
			path:        "/manifest.json",
			accept:      "text/plain",
			blockReason: "JSON file requires application/json Accept type",
		},
		{
			name:        "Valid text file request",
			path:        "/robots.txt",
			accept:      "text/plain",
			blockReason: "",
		},
		{
			name:        "Invalid text accept",
			path:        "/robots.txt",
			accept:      "text/html",
			blockReason: "Text file requires text/plain Accept type",
		},
		{
			name:        "Valid HTML request",
			path:        "/artworks/20",
			accept:      "text/html",
			blockReason: "",
		},
		{
			name:        "Valid root path request",
			path:        "/",
			accept:      "text/html",
			blockReason: "",
		},
		{
			name:        "Invalid HTML accept",
			path:        "/artworks/20",
			accept:      "application/json",
			blockReason: "HTML file requires text/html Accept type",
		},
		{
			name:        "Multiple valid accept types",
			path:        "/js/htmx@2.0.4.min.js",
			accept:      "text/html,application/javascript",
			blockReason: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkAcceptHeader(tt.path, tt.accept)
			if (result == "") == (tt.blockReason != "") {
				t.Errorf("checkAcceptHeader() got %q, want %q", result, tt.blockReason)
			}
			if tt.blockReason != "" && !strings.Contains(result, tt.blockReason) {
				t.Errorf("checkAcceptHeader() got %q, want it to contain %q", result, tt.blockReason)
			}
		})
	}
}

func TestCheckAcceptEncoding(t *testing.T) {
	audit.NewTestingLogger(t)

	tests := []struct {
		name        string
		encoding    string
		blockReason string
	}{
		{
			name:        "Valid gzip",
			encoding:    "gzip",
			blockReason: "",
		},
		{
			name:        "Valid deflate",
			encoding:    "deflate",
			blockReason: "",
		},
		{
			name:        "Valid multiple",
			encoding:    "gzip, deflate, br",
			blockReason: "",
		},
		{
			name:        "Empty encoding",
			encoding:    "",
			blockReason: "Blocked by Accept-Encoding check",
		},
		{
			name:        "Invalid encoding",
			encoding:    "br",
			blockReason: "Blocked by Accept-Encoding check",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Accept-Encoding", tt.encoding)

			result := checkAcceptEncoding(r)
			if result != tt.blockReason {
				t.Errorf("checkAcceptEncoding() = %v, want %v", result, tt.blockReason)
			}
		})
	}
}

func TestCheckAcceptLanguage(t *testing.T) {
	audit.NewTestingLogger(t)

	tests := []struct {
		name        string
		language    string
		blockReason string
	}{
		{
			name:        "Valid language",
			language:    "en-US",
			blockReason: "",
		},
		{
			name:        "Empty language",
			language:    "",
			blockReason: "Blocked by Accept-Language check",
		},
		{
			name:        "Only whitespace",
			language:    "   ",
			blockReason: "Blocked by Accept-Language check",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Accept-Language", tt.language)

			result := checkAcceptLanguage(r)
			if result != tt.blockReason {
				t.Errorf("checkAcceptLanguage() = %v, want %v", result, tt.blockReason)
			}
		})
	}
}

func TestCheckSecFetch(t *testing.T) {
	audit.NewTestingLogger(t)

	tests := []struct {
		name        string
		headers     map[string]string
		blockReason string
	}{
		{
			name: "Valid headers",
			headers: map[string]string{
				"Sec-Fetch-Mode": "navigate",
				"Sec-Fetch-Site": "same-origin",
				"Sec-Fetch-Dest": "document",
			},
			blockReason: "",
		},
		{
			name: "No Sec-Fetch-Mode header",
			headers: map[string]string{
				"Sec-Fetch-Site": "same-origin",
				"Sec-Fetch-Dest": "document",
			},
			blockReason: "Missing Sec-Fetch-Mode header",
		},
		{
			name: "Empty Sec-Fetch-Mode header",
			headers: map[string]string{
				"Sec-Fetch-Mode": "",
				"Sec-Fetch-Site": "same-origin",
				"Sec-Fetch-Dest": "document",
			},
			blockReason: "Missing Sec-Fetch-Mode header",
		},
		{
			name: "No Sec-Fetch-Site header",
			headers: map[string]string{
				"Sec-Fetch-Mode": "navigate",
				"Sec-Fetch-Dest": "document",
			},
			blockReason: "Missing Sec-Fetch-Site header",
		},
		{
			name: "Empty Sec-Fetch-Site header",
			headers: map[string]string{
				"Sec-Fetch-Mode": "navigate",
				"Sec-Fetch-Site": "",
				"Sec-Fetch-Dest": "document",
			},
			blockReason: "Missing Sec-Fetch-Site header",
		},
		{
			name: "No Sec-Fetch-Dest header",
			headers: map[string]string{
				"Sec-Fetch-Mode": "navigate",
				"Sec-Fetch-Site": "same-origin",
			},
			blockReason: "Missing Sec-Fetch-Dest header",
		},
		{
			name: "Empty Sec-Fetch-Dest header",
			headers: map[string]string{
				"Sec-Fetch-Mode": "navigate",
				"Sec-Fetch-Site": "same-origin",
				"Sec-Fetch-Dest": "",
			},
			blockReason: "Missing Sec-Fetch-Dest header",
		},
		{
			name:        "All headers missing",
			headers:     map[string]string{},
			blockReason: "Missing Sec-Fetch headers: Sec-Fetch-Dest, Sec-Fetch-Mode, Sec-Fetch-Site",
		},
		{
			name: "Multiple headers missing",
			headers: map[string]string{
				"Sec-Fetch-Mode": "navigate",
			},
			blockReason: "Missing Sec-Fetch headers: Sec-Fetch-Dest, Sec-Fetch-Site",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			// Set headers
			for key, value := range tt.headers {
				r.Header.Set(key, value)
			}

			result := checkSecFetch(r)
			if result != tt.blockReason {
				t.Errorf("checkSecFetch() = %v, want %v", result, tt.blockReason)
			}
		})
	}
}
