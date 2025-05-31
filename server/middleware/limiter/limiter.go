// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
This file provides network-based rate limiting for HTTP requests.

Clients are grouped by their IP network, with shared rate limits
applied depending on the network's history.

clientHistory allows rate limits to be dynamically adjusted based
on the suspicious-to-normal client ratio from the network over time.

Note that the Client type handles assessing individual requests,
while limiterWrapper manages the actual rate limiting for IP networks.
*/
package limiter

import (
	"sync"
	"time"

	"codeberg.org/pixivfe/pixivfe/audit"
	"golang.org/x/time/rate"
)

// Note that the 40% suspiciousRatio gap between RestrictThreshold and RelaxThreshold exists
// to protect against flapping in limiterWrapper state.
const (
	RegularRate             = 2.0             // 120 tokens per minute (2 per second) for a normal network.
	RegularBurst            = 120             // Maximum tokens or a normal network.
	SuspiciousRate          = 0.1             // 6 tokens per minute (0.1 per second) for a suspicious network.
	SuspiciousBurst         = 90              // Maximum tokens for a suspicious network.
	LimiterExpiryDuration   = time.Hour       // How long to keep limiters in memory before cleanup.
	CleanupInterval         = 5 * time.Minute // Interval between limiter cleanup runs.
	MaxNetworkClientHistory = 60              // Max. number of client histories to track per network.
	RestrictThreshold       = 0.6             // Ratio of suspicious clients that triggers suspicious rate limits.
	RelaxThreshold          = 0.2             // Ratio of suspicious clients that triggers normal rate limits.
)

var (
	limiters sync.Map   // In-memory storage for rate limiters.
	timeNow  = time.Now // Wrapper for time.Now, which allows us to mock it in tests.
)

// clientHistory represents a circular buffer of client suspicious statuses.
type clientHistory struct {
	statuses   []bool // true = suspicious, false = not suspicious
	index      int    // Current index for insertion
	count      int    // Count of items in the buffer
	suspicious int    // Count of suspicious clients
}

// limiterWrapper holds a rate limiter and additional metadata.
//
// Limiters are associated with an IP network and persist in the limiters sync.Map.
type limiterWrapper struct {
	limiter      *rate.Limiter
	network      string        // Associated network identifier
	lastAccess   time.Time     // Last time limiter was accessed
	mu           sync.Mutex    // mutex for operations on this limiter
	history      clientHistory // History of client suspicious statuses
	isSuspicious bool          // Current limiter suspicious status
}

// checkRateLimit attempts to consume 1 token from the limiterWrapper.
//
// Returns an empty string if the request is allowed, or a non-empty string with
// the reason if the request is blocked due to rate limiting.
func checkRateLimit(limiter *limiterWrapper, networkStr string) string {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	// Update last access time
	limiter.lastAccess = timeNow()

	// Try to allow 1 request
	if !limiter.limiter.Allow() {
		audit.GlobalAuditor.Logger.Warnln("Rate limit exceeded",
			"ip", networkStr)

		return "Rate limit exceeded"
	}

	audit.GlobalAuditor.Logger.Debugln("Rate limits passed",
		"ip", networkStr)

	return ""
}

// getOrCreateLimiter returns a limiterWrapper for the given network.
//
// If a limiter already exists in memory, it is returned as-is (the suspicious parameter is ignored).
// Otherwise, if suspicious is true, a limiter with reduced rate/burst is created; if false,
// a regular rate limiter is created.
func getOrCreateLimiter(networkStr string, suspicious bool) *limiterWrapper {
	// Try to load existing limiter from memory
	if limWrapper, found := loadLimiterFromMemory(networkStr); found {
		return limWrapper
	}

	// Create new limiter with appropriate rate and burst
	var limWrapper *limiterWrapper
	if suspicious {
		limWrapper = newLimiterWrapper(SuspiciousRate, SuspiciousBurst, networkStr, true)
	} else {
		limWrapper = newLimiterWrapper(RegularRate, RegularBurst, networkStr, false)
	}

	// Store the new limiter
	limiters.Store(networkStr, limWrapper)

	return limWrapper
}

// loadLimiterFromMemory tries to load from memory a limiterWrapper
// for a given network.
//
// Returns the limiter wrapper if found and true, or nil and false if no data was found.
func loadLimiterFromMemory(network string) (*limiterWrapper, bool) {
	if value, ok := limiters.Load(network); ok {
		limWrapper, ok := value.(*limiterWrapper)
		if !ok {
			return nil, false
		}

		limWrapper.mu.Lock()
		limWrapper.lastAccess = timeNow()
		limWrapper.mu.Unlock()

		return limWrapper, true
	}

	return nil, false
}

// newLimiterWrapper creates a new limiterWrapper with the given parameters.
func newLimiterWrapper(rateLim float64, burstLim int, network string, isSuspicious bool) *limiterWrapper {
	now := timeNow()

	return &limiterWrapper{
		limiter:      rate.NewLimiter(rate.Limit(rateLim), burstLim),
		network:      network,
		lastAccess:   now,
		isSuspicious: isSuspicious,
		history: clientHistory{
			statuses: make([]bool, MaxNetworkClientHistory),
		},
	}
}

// updateNetworkHistory adds a client's suspicious status to the limiterWrapper's history.
//
// The rate and burst of the associated limiter may be updated depending on the history.
func updateNetworkHistory(limiter *limiterWrapper, networkStr string, isSuspicious bool) {
	if limiter == nil {
		return
	}

	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	// Add this client's status to the history
	addClientToHistory(&limiter.history, isSuspicious)

	// Check if we need to upgrade or downgrade the limiter
	shouldUpgrade, shouldDowngrade := evaluateLimiterChange(limiter.history)

	if shouldUpgrade && limiter.isSuspicious {
		// Upgrade: change from suspicious to regular
		limiter.limiter.SetLimit(rate.Limit(RegularRate))
		limiter.limiter.SetBurst(RegularBurst)
		limiter.isSuspicious = false

		audit.GlobalAuditor.Logger.Infoln("Upgraded rate limiter for network",
			"network", networkStr)
	} else if shouldDowngrade && !limiter.isSuspicious {
		// Downgrade: change from regular to suspicious
		limiter.limiter.SetLimit(rate.Limit(SuspiciousRate))
		limiter.limiter.SetBurst(SuspiciousBurst)
		limiter.isSuspicious = true

		audit.GlobalAuditor.Logger.Infoln("Downgraded rate limiter for network",
			"network", networkStr)
	}
}

// addClientToHistory adds a client's suspicious status to the clientHistory.
func addClientToHistory(history *clientHistory, isSuspicious bool) {
	// Initialize history if needed
	if history.statuses == nil {
		history.statuses = make([]bool, MaxNetworkClientHistory)
	}

	// If we're replacing an existing entry, adjust suspicious count
	if history.count == MaxNetworkClientHistory {
		if history.statuses[history.index] {
			history.suspicious--
		}
	} else {
		history.count++
	}

	// Add new entry
	history.statuses[history.index] = isSuspicious
	if isSuspicious {
		history.suspicious++
	}

	// Move index for next insertion
	history.index = (history.index + 1) % MaxNetworkClientHistory
}

// evaluateLimiterChange determines if a limiter should be upgraded or downgraded
// based on the client history.
func evaluateLimiterChange(history clientHistory) (bool, bool) {
	// Only make decisions when the buffer is full
	if history.count < MaxNetworkClientHistory {
		// Keep the initial limiter configuration until we have enough data
		return false, false
	}

	suspiciousRatio := float64(history.suspicious) / float64(history.count)

	// Determine if we should upgrade or downgrade
	upgrade := suspiciousRatio <= RelaxThreshold
	downgrade := suspiciousRatio >= RestrictThreshold

	return upgrade, downgrade
}

// initLimiterCleanup starts a goroutine to periodically clean up expired limiters.
func initLimiterCleanup() bool {
	go func() {
		ticker := time.NewTicker(CleanupInterval)
		defer ticker.Stop()

		for range ticker.C {
			cleanupExpiredLimiters()
		}
	}()

	audit.GlobalAuditor.Logger.Infoln("Rate limiter cleanup initialized",
		"interval", CleanupInterval.String())

	return true
}

// cleanupExpiredLimiters removes limiters that haven't been accessed for the expiry duration.
func cleanupExpiredLimiters() {
	now := timeNow()

	var expiredCount int

	limiters.Range(func(key, value any) bool {
		limWrapper, ok := value.(*limiterWrapper)
		if !ok {
			audit.GlobalAuditor.Logger.Warnln("Found invalid limiter type in map", "key", key)

			limiters.Delete(key) // Remove the invalid entry

			return true
		}

		limWrapper.mu.Lock()
		defer limWrapper.mu.Unlock()

		if now.Sub(limWrapper.lastAccess) > LimiterExpiryDuration {
			limiters.Delete(key)

			expiredCount++
		}

		return true
	})

	if expiredCount > 0 {
		audit.GlobalAuditor.Logger.Debugln("Cleaned up expired rate limiters",
			"count", expiredCount)
	}
}
