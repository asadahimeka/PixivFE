// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package limiter

import (
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"codeberg.org/pixivfe/pixivfe/audit"
)

// blockedByHeaders checks a handful of typical "browser" headers, returning
// a non-empty string with the block reason if something is suspicious enough to block.
func blockedByHeaders(r *http.Request) string {
	// User-Agent check: if missing or in a known-bot list, block.
	if reason := checkUserAgent(r); reason != "" {
		return reason
	}

	// Accept check for various file types
	if reason := checkAcceptHeader(r.URL.Path, r.Header.Get("Accept")); reason != "" {
		audit.GlobalAuditor.Logger.Warnln("Blocked by Accept header - " + reason)

		return "Blocked by Accept header - " + reason
	}

	// Accept-Encoding check: must contain gzip or deflate
	if reason := checkAcceptEncoding(r); reason != "" {
		return reason
	}

	// Accept-Language check: cannot be empty
	if reason := checkAcceptLanguage(r); reason != "" {
		return reason
	}

	// Browsers may not set fetch metadata headers in an insecure context,
	// so don't check them in this case
	if r.TLS != nil {
		if reason := checkSecFetch(r); reason != "" {
			return reason
		}
	}

	// Connection check: block if "close"
	//
	// This check is actually quite useless as when the application is behind a reverse proxy,
	// all incoming connections will likely be over HTTP/1.1 with Connection: keep-alive anyway
	//
	// SearXNG disables this check for a different reason related to uSWGI behavior,
	// see https://github.com/searxng/searxng/issues/2892
	//
	// conn := r.Header.Get("Connection")
	// if strings.EqualFold(strings.TrimSpace(conn), "close") {
	// 	return "Blocked by Connection=close"
	// }

	audit.GlobalAuditor.Logger.Debugln("All header checks passed")

	return ""
}

func checkUserAgent(r *http.Request) string {
	userAgent := r.Header.Get("User-Agent")
	audit.GlobalAuditor.Logger.Debugln("Checking User-Agent header",
		"user_agent", userAgent)

	if userAgent == "" || isBotUserAgent(userAgent) {
		audit.GlobalAuditor.Logger.Warnln("Blocked by User-Agent check",
			"user_agent", userAgent)

		return "Blocked by user-agent check"
	}

	return ""
}

// isBotUserAgent checks if the user-agent matches known bots or suspicious patterns.
//
//nolint:funlen
func isBotUserAgent(userAgent string) bool {
	botSubstrings := []string{
		"abonti",
		"ahrefsbot",
		"archive.org_bot",
		"baiduspider",
		"bingbot",
		"blexbot",
		"bitlybot",
		"curl",
		"exabot",
		"farside/0.1.0",
		"feedfetcher",
		"go-http-client",
		"googlebot",
		"googleimageproxy",
		"headlesschrome",
		"httpclient",
		"jakarta",
		"james bot",
		"java",
		"javafx",
		"jersey",
		"libwww-perl",
		"linkdexbot",
		"mj12bot",
		"msnbot",
		"netvibes",
		"okhttp",
		"petalbot",
		"pixray",
		"python",
		"python-requests",
		"ruby",
		"scrapy",
		"semrushbot",
		"seznambot",
		"sogou",
		"spinn3r",
		"splash",
		"synhttpclient",
		"universalfeedparser",
		"unknown",
		"wget",
		"yahoo! slurp",
		"yacybot",
		"yandexbot",
		"yandexmobilebot",
		"zmeu",
	}

	s := strings.ToLower(userAgent)
	for _, sub := range botSubstrings {
		if strings.Contains(s, sub) {
			audit.GlobalAuditor.Logger.Warnln("Detected bot User-Agent",
				"pattern", sub,
				"user_agent", userAgent)

			return true
		}
	}

	return false
}

// checkAcceptHeader validates that the provided Accept header (accept)
// is acceptable for the file at the given path (path).
//
// If none of the required MIME types (or substring for images)
// is found in the Accept header, an appropriate error message is returned.
func checkAcceptHeader(path, accept string) string {
	ext := strings.ToLower(filepath.Ext(path))

	audit.GlobalAuditor.Logger.Debugln("Checking Accept header",
		"accept", accept,
		"path", path,
		"extension", ext)

	// If the Accept header includes "*/*", it accepts every type so we can skip further checks.
	if strings.Contains(accept, "*/*") {
		return ""
	}

	// mimeRule holds MIME requirements and the associated error message.
	type mimeRule struct {
		required     []string // List of MIME type substrings that will be searched in Accept header.
		errorMsg     string   // Error message template if none of the required types are present.
		shouldFormat bool     // Whether or not errorMsg needs the list of MIME types inserted.
	}

	// Set rules based on the file's extension.
	var rule mimeRule

	switch ext {
	case ".js":
		rule = mimeRule{
			required:     []string{"application/javascript", "text/javascript"},
			errorMsg:     "JavaScript file requires JavaScript Accept type",
			shouldFormat: false,
		}
	case ".css":
		rule = mimeRule{
			required:     []string{"text/css"},
			errorMsg:     "CSS file requires text/css Accept type",
			shouldFormat: false,
		}
	case ".png", ".jpeg", ".jpg", ".gif", ".svg":
		// For image files, we only check for "image/" anywhere in the Accept header.
		rule = mimeRule{
			required:     []string{"image/"},
			errorMsg:     "Image file requires image/* Accept type",
			shouldFormat: false, // errorMsg is already complete; no formatting needed.
		}
	case ".json":
		rule = mimeRule{
			required:     []string{"application/json"},
			errorMsg:     "JSON file requires application/json Accept type",
			shouldFormat: false,
		}
	case ".txt", ".map", ".scss":
		rule = mimeRule{
			required:     []string{"text/plain"},
			errorMsg:     "Text file requires text/plain Accept type",
			shouldFormat: false,
		}
	case ".woff2":
		rule = mimeRule{
			required:     []string{"application/font-woff2", "application/font-woff", "font/woff"},
			errorMsg:     "WOFF2 font file requires Accept type: %s",
			shouldFormat: true,
		}
	default:
		// Default to HTML if no specific extension matches.
		rule = mimeRule{
			required:     []string{"text/html"},
			errorMsg:     "HTML file requires text/html Accept type",
			shouldFormat: false,
		}
	}

	// Iterate over the required MIME substrings.
	// If any of them is present in the Accept header, the request is valid.
	for _, r := range rule.required {
		if strings.Contains(accept, r) {
			return ""
		}
	}

	// If the Accept header did not include a required MIME type, return an error message.
	// If rule.formatted is true, insert the list of expected types; otherwise, the message is ready.
	if rule.shouldFormat {
		return fmt.Sprintf(rule.errorMsg, strings.Join(rule.required, " or "))
	}

	return rule.errorMsg
}

func checkAcceptEncoding(r *http.Request) string {
	enc := strings.ToLower(r.Header.Get("Accept-Encoding"))

	audit.GlobalAuditor.Logger.Debugln("Checking Accept-Encoding header",
		"encoding", enc)

	acceptedEncodings := []string{"identity", "gzip", "deflate"}

	// Check if any of the accepted encodings are present
	valid := false

	for _, acceptedEnc := range acceptedEncodings {
		if strings.Contains(enc, acceptedEnc) {
			valid = true

			break
		}
	}

	if !valid {
		audit.GlobalAuditor.Logger.Warnln("Blocked by Accept-Encoding check")

		return "Blocked by Accept-Encoding check"
	}

	return ""
}

func checkAcceptLanguage(r *http.Request) string {
	lang := r.Header.Get("Accept-Language")
	audit.GlobalAuditor.Logger.Debugln("Checking Accept-Language header",
		"lang", lang)

	if strings.TrimSpace(lang) == "" {
		audit.GlobalAuditor.Logger.Warnln("Blocked by Accept-Language check")

		return "Blocked by Accept-Language check"
	}

	return ""
}

// checkSecFetch checks whether the fetch metadata headers for
// a request are present and non-empty.
//
// Note that SearXNG doesn't actually check these headers
// for browser compatibility reasons, see https://github.com/searxng/searxng/pull/3965
func checkSecFetch(r *http.Request) string {
	headers := map[string]string{
		"Sec-Fetch-Mode": r.Header.Get("Sec-Fetch-Mode"),
		"Sec-Fetch-Site": r.Header.Get("Sec-Fetch-Site"),
		"Sec-Fetch-Dest": r.Header.Get("Sec-Fetch-Dest"),
	}

	audit.GlobalAuditor.Logger.Debug("Checking Sec-Fetch headers",
		"mode", headers["Sec-Fetch-Mode"],
		"site", headers["Sec-Fetch-Site"],
		"dest", headers["Sec-Fetch-Dest"])

	var missingHeaders []string

	for headerName, headerValue := range headers {
		if len(headerValue) == 0 {
			audit.GlobalAuditor.Logger.Warnln("Missing " + headerName + " header")
			missingHeaders = append(missingHeaders, headerName)
		}
	}

	switch len(missingHeaders) {
	case 0:
		return ""
	case 1:
		return "Missing " + missingHeaders[0] + " header"
	default:
		sort.Strings(missingHeaders)

		return "Missing Sec-Fetch headers: " + strings.Join(missingHeaders, ", ")
	}
}
