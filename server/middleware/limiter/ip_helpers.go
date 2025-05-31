// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package limiter

import (
	"net"
	"net/http"
	"strings"

	"codeberg.org/pixivfe/pixivfe/audit"
)

// IPv4 and IPv6 address bit lengths.
const (
	ipv4BitLength = 32  // Standard length of an IPv4 address in bits
	ipv6BitLength = 128 // Standard length of an IPv6 address in bits
)

// getClientIP extracts the client's IP address from an HTTP request with proxy awareness.
//
// Proxy headers (X-Forwarded-For, X-Real-IP) are only trusted when the connection
// comes from trusted sources (private/loopback networks).
func getClientIP(r *http.Request) string {
	audit.GlobalAuditor.Logger.Debugln("Resolving client IP",
		"remote_addr", r.RemoteAddr,
		"x_forwarded_for", r.Header.Get("X-Forwarded-For"),
		"x_real_ip", r.Header.Get("X-Real-IP"))

	// Extract IP from RemoteAddr by removing the port component
	remoteIP := r.RemoteAddr
	if ip, _, err := net.SplitHostPort(remoteIP); err == nil {
		remoteIP = ip
	}

	// Only trust proxy headers if request comes from a trusted network
	fromTrustedSource := false
	if ip := net.ParseIP(remoteIP); ip != nil {
		fromTrustedSource = ip.IsPrivate() || ip.IsLoopback()
	}

	if fromTrustedSource {
		// X-Real-IP takes precedence as it's typically the originating client IP
		// when set by a trusted proxy
		if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
			return realIP
		}

		// If X-Real-IP isn't available, use the last IP in X-Forwarded-For
		// This represents the client's IP in a chain of proxies
		if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
			parts := strings.Split(xff, ",")
			return strings.TrimSpace(parts[len(parts)-1])
		}
	} else {
		audit.GlobalAuditor.Logger.Warnln("Request from untrusted source, ignoring proxy headers",
			"remote_ip", remoteIP)
	}

	// Fallback to the direct connection IP when proxy headers aren't available
	// or the source isn't trusted
	if remoteIP != "" {
		return remoteIP
	}

	audit.GlobalAuditor.Logger.Errorln("Could not determine client IP")
	return ""
}

// ipMatchesList checks if an IP is within any of the provided CIDRs or matches them exactly.
func ipMatchesList(rawIP net.IP, cidrs []string) bool {
	audit.GlobalAuditor.Logger.Debugln("Checking IP against list",
		"ip", rawIP,
		"list_size", len(cidrs))

	// Store IP string representation to avoid repeated conversions
	ipStr := rawIP.String()

	for _, cidr := range cidrs {
		// Check for an exact match first
		if ipStr == cidr {
			audit.GlobalAuditor.Logger.Debugln("IP matched exactly",
				"ip", ipStr,
				"match", cidr)
			return true
		}

		// Try parsing as CIDR and check if IP is within subnet
		_, subnet, err := net.ParseCIDR(cidr)
		if err == nil && subnet.Contains(rawIP) {
			audit.GlobalAuditor.Logger.Debugln("IP matched CIDR",
				"ip", ipStr,
				"cidr", cidr)
			return true
		}
	}

	// No match found in any CIDR or exact comparison
	return false
}

func getNetwork(rawIP net.IP, ipv4Prefix, ipv6Prefix int) *net.IPNet {
	// Create mask based on IP version and configured prefix
	var mask net.IPMask
	if rawIP.To4() != nil {
		// IPv4
		mask = net.CIDRMask(ipv4Prefix, ipv4BitLength)
	} else {
		// IPv6
		mask = net.CIDRMask(ipv6Prefix, ipv6BitLength)
	}

	// Create network with the IP and determined mask
	return &net.IPNet{
		IP:   rawIP.Mask(mask),
		Mask: mask,
	}
}
