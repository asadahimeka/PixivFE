// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package limiter

import (
	"math"
	"net/http"
	"strconv"

	"codeberg.org/pixivfe/pixivfe/audit"
	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/server/routes"
)

// Rate limiting header names.
//
// ref: https://www.ietf.org/archive/id/draft-polli-ratelimit-headers-02.html
const (
	HeaderRateLimitLimit     string = "RateLimit-Limit" // this is intended
	HeaderRateLimitRemaining string = "RateLimit-Remaining"
	HeaderRateLimitReset     string = "RateLimit-Reset"
	HeaderRateLimitStatus    string = "RateLimit-Status" // non-standard
)

// excludedPaths won't have traffic filtered by the limiter middleware.
var excludedPaths = []string{
	"/limiter/", // CSS token endpoint or Turnstile verification endpoint
	"/proxy/",   // Content proxy endpoints
	"/about",
	"/css/",
	"/fonts/",
	"/icons/",
	"/img/",
	"/js/",
	"/manifest.json",
	"/robots.txt",
}

// Setup initializes the limiter middleware.
//
// Responsible for starting long-lived goroutines that periodically
// clear stale data to prevent build up.
func Setup() {
	initLimiterCleanup()

	if config.GlobalConfig.Limiter.DetectionMethod == config.LinkTokenDetectionMethod {
		initLinkTokenCleanup()
	}
}

// Evaluate is the entrypoint to the limiter middleware.
//
// The logic was originally based on the reference SearXNG code in searxng/searx/botdetection.
//
// Implementation notes:
//   - In the original SearXNG implementation, the HTTP header checks only occur for /search requests,
//     but here we do them for all requests as we have far more endpoints to protect (/artworks, /users, etc.);
//     better to ennumerate goodness via excluded paths than badness
func Evaluate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Initialize client with request data
		client, err := newClient(r)
		if client == nil || err != nil {
			// newClient has already written an error response
			return
		}

		// 1: Fast-path exclusions - check if the path is completely exempt from filtering
		if client.isFullyExcludedPath(r) {
			audit.GlobalAuditor.Logger.Debugln("Request allowed - excluded path",
				"ip", client.ip.String(),
				"network", client.network.String(),
				"path", r.URL.Path)
			next.ServeHTTP(w, r)
			return
		}

		// 2: IP-based filtering - explicit allow/deny lists take precedence
		if allowed, blocked := client.checkIPLists(); allowed {
			audit.GlobalAuditor.Logger.Infoln("Request allowed - IP in pass-list",
				"ip", client.ip.String(),
				"network", client.network.String())
			next.ServeHTTP(w, r)
			return
		} else if blocked {
			audit.GlobalAuditor.Logger.Warnln("Request blocked - IP in block-list",
				"ip", client.ip.String(),
				"network", client.network.String())

			// requestcontext.FromRequest(r).StatusCode = http.StatusUnauthorized
			w.WriteHeader(http.StatusUnauthorized)

			routes.BlockPage(w, r, "IP is on BLOCKLIST")
			return
		}

		// 3: Local network filtering (optional based on configuration)
		if !config.GlobalConfig.Limiter.FilterLocal && client.isLocalLink() {
			audit.GlobalAuditor.Logger.Debugln("Request allowed - local network",
				"ip", client.ip.String(),
				"network", client.network.String())
			next.ServeHTTP(w, r)
			return
		}

		// 4: Check request headers (conditionally)
		if config.GlobalConfig.Limiter.CheckHeaders {
			if blockReason := client.blockedByHeaders(r); blockReason != "" {
				audit.GlobalAuditor.Logger.Warnln("Request blocked - headers",
					"ip", client.ip.String(),
					"network", client.network.String(),
					"reason", blockReason)

				// requestcontext.FromRequest(r).StatusCode = http.StatusUnauthorized
				w.WriteHeader(http.StatusUnauthorized)

				routes.BlockPage(w, r, blockReason)
				return
			}
		}

		// 5: DetectionMethod handling
		detectionMethod := config.GlobalConfig.Limiter.DetectionMethod

		switch detectionMethod {
		case config.LinkTokenDetectionMethod, config.TurnstileDetectionMethod:
			// For both link token and Turnstile, the client is expected to present a valid pixivfe-Ping cookie.
			if !client.validatePingCookie(r) {
				// Client failed cookie validation, marked as suspicious.
				audit.GlobalAuditor.Logger.Warnln("Client failed challenge verification, marked as suspicious",
					"ip", client.ip.String(),
					"network", client.network.String(),
					"method", detectionMethod)
			} else {
				// Client passed cookie validation, marked as not suspicious.
				audit.GlobalAuditor.Logger.Debugln("Client passed challenge verification",
					"ip", client.ip.String(),
					"network", client.network.String(),
					"method", detectionMethod)
			}
		case config.NoneDetectionMethod:
			// No specific challenge method. Always treat clients as non-suspicious.
			client.clearSuspiciousStatus()
			client.limiter = getOrCreateLimiter(client.network.String(), client.isSuspicious)
			updateNetworkHistory(client.limiter, client.network.String(), client.isSuspicious)
			audit.GlobalAuditor.Logger.Debugln("Client validation skipped (no detection method), treated as not suspicious",
				"ip", client.ip.String(),
				"network", client.network.String())
		}

		// At this point:
		// - client.isSuspicious is set according to the detection method outcome.
		// - client.limiter is initialized with rates corresponding to the suspicious status.

		// 6: Rate limiting - apply limits based on client's (suspicious) status.
		if blockReason := checkRateLimit(client.limiter, client.network.String()); blockReason != "" {
			audit.GlobalAuditor.Logger.Warnln("Request blocked - exceeded rate limit",
				"ip", client.ip.String(),
				"network", client.network.String(),
				"suspicious", client.isSuspicious,
				"reason", blockReason)
			addRateLimitHeaders(w, client)

			// requestcontext.FromRequest(r).StatusCode = http.StatusTooManyRequests
			w.WriteHeader(http.StatusTooManyRequests)

			routes.BlockPage(w, r, blockReason)
			return
		}

		// All checks passed - serve the request.
		audit.GlobalAuditor.Logger.Debugln("Request allowed - all checks passed",
			"ip", client.ip.String(),
			"network", client.network.String(),
			"suspicious", client.isSuspicious)
		addRateLimitHeaders(w, client)
		next.ServeHTTP(w, r)
	})
}

// addRateLimitHeaders adds rate limiting information to the response headers.
func addRateLimitHeaders(w http.ResponseWriter, client *Client) {
	if client == nil || client.limiter == nil {
		return
	}

	client.limiter.mu.Lock()
	defer client.limiter.mu.Unlock()

	limiter := client.limiter.limiter

	// Get current tokens and limit info
	currentTokens := limiter.Tokens()
	burst := limiter.Burst()
	limit := limiter.Limit()

	// Calculate tokens remaining (can't exceed burst)
	remaining := int(math.Min(float64(burst), currentTokens))

	// Calculate seconds until full bucket replenishment (if not already full)
	var resetTime int64

	if currentTokens < float64(burst) {
		tokenDeficit := float64(burst) - currentTokens
		if limit > 0 {
			resetTime = int64(math.Ceil(tokenDeficit / float64(limit)))
		}
	}

	// Add Rate-Limit headers
	burstStr := strconv.Itoa(burst)
	remainingStr := strconv.Itoa(remaining)
	resetStr := strconv.FormatInt(resetTime, 10)

	w.Header().Set(HeaderRateLimitLimit, burstStr)
	w.Header().Set(HeaderRateLimitRemaining, remainingStr)
	w.Header().Set(HeaderRateLimitReset, resetStr)

	// Add Retry-After header if rate limited (remaining = 0)
	// Retry-After should be seconds.
	if remaining <= 0 && resetTime >= 0 {
		w.Header().Set("Retry-After", strconv.FormatInt(resetTime, 10))
	}

	// Add status headers
	var statusValue string
	if client.isSuspicious {
		statusValue = "Suspicious"
	} else {
		statusValue = "Normal"
	}

	w.Header().Set(HeaderRateLimitStatus, statusValue)
}
