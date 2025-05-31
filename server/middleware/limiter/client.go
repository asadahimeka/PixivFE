// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package limiter

import (
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"

	"codeberg.org/pixivfe/pixivfe/audit"
	"codeberg.org/pixivfe/pixivfe/config"
	"github.com/zeebo/xxh3"
)

var (
	errMissingClientIP = errors.New("missing client IP")
	errInvalidIPFormat = errors.New("invalid IP format")
	errNilNetwork      = errors.New("could not determine network")
)

// Client represents an HTTP request with associated network and rate limiting information.
//
// Instances are ephemeral and exist only for the duration of a single HTTP request lifecycle.
type Client struct {
	ip           net.IP
	network      net.IPNet
	fingerprint  string
	isSuspicious bool
	limiter      *limiterWrapper
}

// newClient constructs a Client from an HTTP request, resolving IP and
// network, but does not run any checks yet, leaving the isSuspicious and limiter fields uninitialized.
func newClient(r *http.Request) (*Client, error) {
	var client Client

	realIP := getClientIP(r)
	if realIP == "" {
		return nil, errMissingClientIP
	}

	parsedIP := net.ParseIP(realIP)
	if parsedIP == nil {
		return nil, errInvalidIPFormat
	}

	network := getNetwork(parsedIP, config.GlobalConfig.Limiter.IPv4Prefix, config.GlobalConfig.Limiter.IPv6Prefix)
	if network == nil {
		return nil, errNilNetwork
	}

	return &Client{
		ip:          parsedIP,
		network:     *network,
		fingerprint: client.generateClientFingerprint(r),
	}, nil
}

// generateClientFingerprint creates a hash from client attributes to act as a fingerprint.
//
// Relatively low entropy, but good enough for our purposes.
func (c *Client) generateClientFingerprint(r *http.Request) string {
	// Combine network with user agent and language then hash it for a unique fingerprint
	data := c.network.String() + r.Header.Get("Accept-Language") + r.Header.Get("User-Agent")

	return strconv.FormatUint(xxh3.HashString(data), 10)
}

// checkIPLists checks if the client's IP is on the pass or block list.
//
// Returns (allowed, blocked) as a tuple - at most one can be true.
func (c *Client) checkIPLists() (bool, bool) {
	if c.isPassListed() {
		return true, false
	}

	if c.isBlockListed() {
		return false, true
	}

	return false, false
}

// validatePingCookie checks pingCookieName, sets an appropriate client.isSuspicious value,
// then assigns a client.limiter with appropriate rate limits and updates the history
// for the client's network.
//
// Returns a bool with the result (true if not suspicious).
func (c *Client) validatePingCookie(r *http.Request) bool {
	if cookie, err := r.Cookie(pingCookieName); err == nil {
		if verifyPingCookie(cookie, r) {
			audit.GlobalAuditor.Logger.Debugln("Valid ping cookie found",
				"ip", c.ip.String())
			c.clearSuspiciousStatus()
			c.limiter = getOrCreateLimiter(c.network.String(), c.isSuspicious)
			updateNetworkHistory(c.limiter, c.network.String(), c.isSuspicious)

			return true
		}

		audit.GlobalAuditor.Logger.Warnln("Invalid ping cookie",
			"ip", c.ip.String())
	}

	// No valid cookie or invalid cookie, client is suspicious
	c.markSuspicious()
	c.limiter = getOrCreateLimiter(c.network.String(), c.isSuspicious)
	updateNetworkHistory(c.limiter, c.network.String(), c.isSuspicious)

	return false
}

// isFullyExcludedPath returns true if the request path matches any
// of the fullyExcludedPaths (skip all checks).
func (c *Client) isFullyExcludedPath(r *http.Request) bool {
	path := r.URL.Path
	for _, p := range excludedPaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}

	return false
}

// isPassListed returns true if c.IP is in the configured pass list.
func (c *Client) isPassListed() bool {
	return ipMatchesList(c.ip, config.GlobalConfig.Limiter.PassIPs)
}

// isBlockListed returns true if c.IP is in the configured block list.
func (c *Client) isBlockListed() bool {
	return ipMatchesList(c.ip, config.GlobalConfig.Limiter.BlockIPs)
}

// isLocalLink returns true if c.IP is a link-local address.
//
// Supports both IPv4 and IPv6.
func (c *Client) isLocalLink() bool {
	if c.ip.To4() != nil {
		// IPv4
		return c.ip.IsLinkLocalUnicast() // covers 169.254.0.0/16
	}
	// IPv6
	return c.ip.IsLinkLocalUnicast() // covers fe80::/10
}

// blockedByHeaders checks for suspicious request headers.
//
// Returns a non-empty string with the block reason if blocked.
func (c *Client) blockedByHeaders(r *http.Request) string {
	return blockedByHeaders(r)
}

// clearSuspiciousStatus sets the IsSuspicious field for this client
// to false, effectively giving it a clean slate.
func (c *Client) clearSuspiciousStatus() {
	c.isSuspicious = false
}

// markSuspicious sets the IsSuspicious field of the client to true.
func (c *Client) markSuspicious() {
	c.isSuspicious = true
}
