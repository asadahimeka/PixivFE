// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package limiter

import (
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// clearLimiters removes all stored rate limiters from memory
func clearLimiters() {
	limiters = sync.Map{}
}

// TestCheckRateLimit ensures tokens are consumed correctly.
func TestCheckRateLimit(t *testing.T) {
	clearLimiters()

	_ = setupLimiterTest(t)

	r := httptest.NewRequest("GET", "http://localhost", nil)
	r.RemoteAddr = "192.168.0.1:9999"

	c, err := newClient(r)
	if err != nil {
		t.Fatalf("newClient error: %v", err)
	}

	// Create non-suspicious limiter
	c.limiter = getOrCreateLimiter(c.network.String(), false)
	if c.limiter == nil {
		t.Fatal("Limiter should not be nil")
	}

	// Consume tokens up to capacity
	// For a regular limiter, we expect burst=120
	for i := range RegularBurst {
		if reason := checkRateLimit(c.limiter, c.network.String()); reason != "" {
			t.Errorf("Expected successful consume %d, got blocked reason: %s", i+1, reason)
		}
	}

	// Next token should be blocked (limiter exhausted)
	if reason := checkRateLimit(c.limiter, c.network.String()); !strings.Contains(reason, "Rate limit exceeded") {
		t.Errorf("Expected 'Rate limit exceeded', got: %s", reason)
	}

	// Move time forward to refill some tokens (1 second = RegularRate tokens)
	//
	// NOTE: we need to pass actual time due to time/rate using the standard
	// time.Now() internally; 500ms is enough to restore one token for a regular
	// limiter, which should produce a rate-limit pass
	time.Sleep(500 * time.Millisecond)
	if reason := checkRateLimit(c.limiter, c.network.String()); reason != "" {
		t.Errorf("Expected rate-limit pass after refill, got: %s", reason)
	}
}

// TestGetOrCreateLimiter verifies creation or retrieval of a rate limiter.
func TestGetOrCreateLimiter(t *testing.T) {
	setupLimiterTest(t)

	r := httptest.NewRequest("GET", "http://localhost", nil)
	r.RemoteAddr = "192.168.0.1:9999"

	c, err := newClient(r)
	if err != nil {
		t.Fatalf("newClient error: %v", err)
	}

	l1 := getOrCreateLimiter(c.network.String(), false)
	if l1 == nil {
		t.Fatal("Expected a non-nil limiter wrapper")
	}
	// calling again should return the same object
	l2 := getOrCreateLimiter(c.network.String(), false)
	if l2 == nil || l1 != l2 {
		t.Error("Expected same limiter wrapper instance for repeated calls")
	}
}

func TestLoadLimiterFromMemory(t *testing.T) {
	clearLimiters()
	mockTime := setupLimiterTest(t)

	networkStr := "192.168.1.0/24"

	// Create and store a limiter
	originalLimiter := newLimiterWrapper(RegularRate, RegularBurst, networkStr, false)
	limiters.Store(networkStr, originalLimiter)

	initialTime := originalLimiter.lastAccess

	// Advance mock time
	mockTime.Sleep(time.Second)

	// Try to load it from memory
	loadedLimiter, found := loadLimiterFromMemory(networkStr)

	// Verify it was found and is the same instance
	if !found {
		t.Fatal("Expected to find limiter in memory")
	}

	if loadedLimiter != originalLimiter {
		t.Error("Loaded limiter is not the same instance as the original")
	}

	// Verify the last access time was updated
	if loadedLimiter.lastAccess == initialTime {
		t.Error("Last access time should have been updated")
	}

	// Try loading a non-existent limiter
	nonExistentLimiter, found := loadLimiterFromMemory("10.0.0.0/8")

	if found {
		t.Fatal("Should not find non-existent limiter")
	}

	if nonExistentLimiter != nil {
		t.Error("Should return nil for non-existent limiter")
	}

	// Test with invalid value type in map
	limiters.Store("invalid-type", "not-a-limiter")
	invalidLimiter, found := loadLimiterFromMemory("invalid-type")
	if found || invalidLimiter != nil {
		t.Error("Should return not found for invalid type in map")
	}
}

func TestNewLimiterWrapper(t *testing.T) {
	mockTime := setupLimiterTest(t)
	now := mockTime.Now()

	networkStr := "192.168.1.0/24"

	// Test creating a regular limiter
	regularLimiter := newLimiterWrapper(RegularRate, RegularBurst, networkStr, false)

	if regularLimiter == nil {
		t.Fatal("Regular limiter should not be nil")
	}

	if regularLimiter.network != networkStr {
		t.Errorf("Expected network %s, got %s", networkStr, regularLimiter.network)
	}

	if regularLimiter.isSuspicious {
		t.Error("Regular limiter should not be marked as suspicious")
	}

	if regularLimiter.history.statuses == nil {
		t.Error("History statuses array should be initialized")
	}

	if len(regularLimiter.history.statuses) != MaxNetworkClientHistory {
		t.Errorf("Expected history size %d, got %d", MaxNetworkClientHistory, len(regularLimiter.history.statuses))
	}

	if !regularLimiter.lastAccess.Equal(now) {
		t.Error("Last access time should be set to current time")
	}

	// Test creating a suspicious limiter
	suspiciousLimiter := newLimiterWrapper(SuspiciousRate, SuspiciousBurst, networkStr, true)

	if suspiciousLimiter == nil {
		t.Fatal("Suspicious limiter should not be nil")
	}

	if !suspiciousLimiter.isSuspicious {
		t.Error("Suspicious limiter should be marked as suspicious")
	}

	// Verify the limiter parameters
	if float64(regularLimiter.limiter.Limit()) != RegularRate {
		t.Errorf("Expected regular rate %f, got %f", RegularRate, float64(regularLimiter.limiter.Limit()))
	}

	if regularLimiter.limiter.Burst() != RegularBurst {
		t.Errorf("Expected regular burst %d, got %d", RegularBurst, regularLimiter.limiter.Burst())
	}

	if float64(suspiciousLimiter.limiter.Limit()) != SuspiciousRate {
		t.Errorf("Expected suspicious rate %f, got %f", SuspiciousRate, float64(suspiciousLimiter.limiter.Limit()))
	}

	if suspiciousLimiter.limiter.Burst() != SuspiciousBurst {
		t.Errorf("Expected suspicious burst %d, got %d", SuspiciousBurst, suspiciousLimiter.limiter.Burst())
	}
}

func TestUpdateNetworkHistory(t *testing.T) {
	clearLimiters()
	_ = setupLimiterTest(t)

	networkStr := "192.168.1.0/24"

	t.Run("nil limiter should not panic", func(t *testing.T) {
		// Test with nil limiter (should not panic)
		updateNetworkHistory(nil, networkStr, false)
	})

	t.Run("downgrade regular to suspicious", func(t *testing.T) {
		// Create a regular (non-suspicious) limiter
		limiter := newLimiterWrapper(RegularRate, RegularBurst, networkStr, false)
		originalLimit := limiter.limiter.Limit()
		originalBurst := limiter.limiter.Burst()

		// Verify initial state
		if limiter.isSuspicious {
			t.Fatal("Limiter should start as non-suspicious")
		}

		// Add enough suspicious clients to trigger downgrade
		// We need MaxNetworkClientHistory * DowngradeThreshold suspicious clients
		suspiciousNeeded := int(float64(MaxNetworkClientHistory) * RestrictThreshold)
		for range suspiciousNeeded {
			updateNetworkHistory(limiter, networkStr, true)
		}

		// Add remaining non-suspicious clients to fill the buffer
		nonSuspiciousToAdd := MaxNetworkClientHistory - suspiciousNeeded
		for range nonSuspiciousToAdd {
			updateNetworkHistory(limiter, networkStr, false)
		}

		// Check if limiter was downgraded
		if !limiter.isSuspicious {
			t.Error("Limiter should have been downgraded to suspicious")
		}

		// Verify rate and burst were changed
		if limiter.limiter.Limit() == originalLimit {
			t.Error("Limiter rate should have changed")
		}

		if limiter.limiter.Burst() == originalBurst {
			t.Error("Limiter burst should have changed")
		}

		if float64(limiter.limiter.Limit()) != SuspiciousRate {
			t.Errorf("Expected suspicious rate %f, got %f", SuspiciousRate, float64(limiter.limiter.Limit()))
		}

		if limiter.limiter.Burst() != SuspiciousBurst {
			t.Errorf("Expected suspicious burst %d, got %d", SuspiciousBurst, limiter.limiter.Burst())
		}
	})

	t.Run("upgrade suspicious to regular", func(t *testing.T) {
		// Create a suspicious limiter
		limiter := newLimiterWrapper(SuspiciousRate, SuspiciousBurst, networkStr, true)
		originalLimit := limiter.limiter.Limit()
		originalBurst := limiter.limiter.Burst()

		// Verify initial state
		if !limiter.isSuspicious {
			t.Fatal("Limiter should start as suspicious")
		}

		// Add enough non-suspicious clients to exceed the upgrade threshold
		// We need MaxNetworkClientHistory * UpgradeThreshold non-suspicious clients
		nonSuspiciousNeeded := int(float64(MaxNetworkClientHistory) * (1.0 - RelaxThreshold))
		for range nonSuspiciousNeeded {
			updateNetworkHistory(limiter, networkStr, false)
		}

		// Add remaining suspicious clients to fill the buffer
		suspiciousToAdd := MaxNetworkClientHistory - nonSuspiciousNeeded
		for range suspiciousToAdd {
			updateNetworkHistory(limiter, networkStr, true)
		}

		// Check if limiter was upgraded
		if limiter.isSuspicious {
			t.Error("Limiter should have been upgraded to regular")
		}

		// Verify rate and burst were changed
		if limiter.limiter.Limit() == originalLimit {
			t.Error("Limiter rate should have changed")
		}

		if limiter.limiter.Burst() == originalBurst {
			t.Error("Limiter burst should have changed")
		}

		if float64(limiter.limiter.Limit()) != RegularRate {
			t.Errorf("Expected regular rate %f, got %f", RegularRate, float64(limiter.limiter.Limit()))
		}

		if limiter.limiter.Burst() != RegularBurst {
			t.Errorf("Expected regular burst %d, got %d", RegularBurst, limiter.limiter.Burst())
		}
	})

	t.Run("no change when history not full", func(t *testing.T) {
		// Create a regular limiter
		limiter := newLimiterWrapper(RegularRate, RegularBurst, networkStr, false)
		originalLimit := limiter.limiter.Limit()
		originalBurst := limiter.limiter.Burst()

		// Add some clients but not enough to fill the buffer
		updateNetworkHistory(limiter, networkStr, true)
		updateNetworkHistory(limiter, networkStr, false)

		// We need MaxNetworkClientHistory - 2 more entries to fill the buffer
		for range MaxNetworkClientHistory - 3 {
			updateNetworkHistory(limiter, networkStr, true)
		}

		// Verify no change occurred even with mostly suspicious clients
		if limiter.isSuspicious {
			t.Error("Limiter should not have changed status until buffer is full")
		}

		if limiter.limiter.Limit() != originalLimit {
			t.Error("Limiter rate should not have changed")
		}

		if limiter.limiter.Burst() != originalBurst {
			t.Error("Limiter burst should not have changed")
		}

		// One more to fill the buffer and trigger the change
		updateNetworkHistory(limiter, networkStr, true)

		// Now it should have changed to suspicious
		if !limiter.isSuspicious {
			t.Error("Limiter should have changed to suspicious after buffer filled")
		}
	})
}

func TestAddClientToHistory(t *testing.T) {
	t.Run("Initialize empty history", func(t *testing.T) {
		h := clientHistory{}
		addClientToHistory(&h, false)

		if h.statuses == nil {
			t.Error("statuses should be initialized")
		}
		if h.count != 1 {
			t.Errorf("count should be 1, got %d", h.count)
		}
		if h.index != 1 {
			t.Errorf("index should be 1, got %d", h.index)
		}
		if h.suspicious != 0 {
			t.Errorf("suspicious should be 0, got %d", h.suspicious)
		}
	})

	t.Run("Add suspicious client", func(t *testing.T) {
		h := clientHistory{}
		addClientToHistory(&h, true)

		if h.suspicious != 1 {
			t.Errorf("suspicious should be 1, got %d", h.suspicious)
		}
		if !h.statuses[0] {
			t.Error("status should be true (suspicious)")
		}
	})

	t.Run("Fill buffer and verify count", func(t *testing.T) {
		h := clientHistory{}

		// First fill the history exactly to capacity with a known pattern
		// Add 7 suspicious clients (true)
		for range 7 {
			addClientToHistory(&h, true)
		}
		// Add 8 non-suspicious clients (false)
		for range 8 {
			addClientToHistory(&h, false)
		}

		// At this point we should have exactly MaxNetworkClientHistory entries
		// with exactly 7 suspicious entries
		if h.suspicious != 7 {
			t.Errorf("suspicious should be 7, got %d", h.suspicious)
		}
	})

	t.Run("Circular buffer overwrite", func(t *testing.T) {
		h := clientHistory{}

		// Fill the buffer with suspicious clients
		for range MaxNetworkClientHistory {
			addClientToHistory(&h, true)
		}

		if h.suspicious != MaxNetworkClientHistory {
			t.Errorf("suspicious should be %d, got %d", MaxNetworkClientHistory, h.suspicious)
		}

		// Now overwrite with non-suspicious client
		addClientToHistory(&h, false)

		if h.suspicious != MaxNetworkClientHistory-1 {
			t.Errorf("suspicious should be %d, got %d", MaxNetworkClientHistory-1, h.suspicious)
		}

		// Check that index wrapped correctly
		if h.index != 1 {
			t.Errorf("index should be 1, got %d", h.index)
		}
	})
}

func TestEvaluateLimiterChange(t *testing.T) {
	t.Run("Not enough data", func(t *testing.T) {
		h := clientHistory{
			statuses:   make([]bool, MaxNetworkClientHistory),
			count:      MaxNetworkClientHistory - 1,
			suspicious: 0,
		}

		upgrade, downgrade := evaluateLimiterChange(h)
		if upgrade || downgrade {
			t.Errorf("Should not recommend changes when buffer not full, got upgrade=%v, downgrade=%v",
				upgrade, downgrade)
		}
	})

	t.Run("Clear upgrade case", func(t *testing.T) {
		h := clientHistory{
			statuses:   make([]bool, MaxNetworkClientHistory),
			count:      MaxNetworkClientHistory,
			suspicious: int(float64(MaxNetworkClientHistory)*RelaxThreshold - 1), // Just below upgrade threshold
		}

		upgrade, downgrade := evaluateLimiterChange(h)
		if !upgrade {
			t.Error("Should recommend upgrade")
		}
		if downgrade {
			t.Error("Should not recommend downgrade")
		}
	})

	t.Run("Clear downgrade case", func(t *testing.T) {
		h := clientHistory{
			statuses:   make([]bool, MaxNetworkClientHistory),
			count:      MaxNetworkClientHistory,
			suspicious: int(float64(MaxNetworkClientHistory)*RestrictThreshold + 1), // Just above downgrade threshold
		}

		upgrade, downgrade := evaluateLimiterChange(h)
		if upgrade {
			t.Error("Should not recommend upgrade")
		}
		if !downgrade {
			t.Error("Should recommend downgrade")
		}
	})

	t.Run("Neither upgrade nor downgrade", func(t *testing.T) {
		midPoint := int(float64(MaxNetworkClientHistory) * 0.5) // Between thresholds
		h := clientHistory{
			statuses:   make([]bool, MaxNetworkClientHistory),
			count:      MaxNetworkClientHistory,
			suspicious: midPoint,
		}

		upgrade, downgrade := evaluateLimiterChange(h)
		if upgrade {
			t.Error("Should not recommend upgrade")
		}
		if downgrade {
			t.Error("Should not recommend downgrade")
		}
	})

	t.Run("Edge case at thresholds", func(t *testing.T) {
		// Test at exact upgrade threshold
		h1 := clientHistory{
			statuses:   make([]bool, MaxNetworkClientHistory),
			count:      MaxNetworkClientHistory,
			suspicious: int(float64(MaxNetworkClientHistory) * RelaxThreshold), // Exactly at upgrade threshold
		}

		upgrade, _ := evaluateLimiterChange(h1)
		if !upgrade {
			t.Error("Should recommend upgrade at exact threshold")
		}

		// Test at exact downgrade threshold
		h2 := clientHistory{
			statuses:   make([]bool, MaxNetworkClientHistory),
			count:      MaxNetworkClientHistory,
			suspicious: int(float64(MaxNetworkClientHistory) * RestrictThreshold), // Exactly at downgrade threshold
		}

		_, downgrade := evaluateLimiterChange(h2)
		if !downgrade {
			t.Error("Should recommend downgrade at exact threshold")
		}
	})
}

// TestLimiterCleanup verifies that expired limiters are properly cleaned up
func TestLimiterCleanup(t *testing.T) {
	mockTime := setupLimiterTest(t)

	// Create a client and its limiter
	r := httptest.NewRequest("GET", "http://localhost", nil)
	r.RemoteAddr = "192.168.0.1:9999"
	c, _ := newClient(r)
	getOrCreateLimiter(c.network.String(), false)

	// Verify the limiter exists
	network := c.network.String()
	if _, found := limiters.Load(network); !found {
		t.Fatal("Expected limiter to exist")
	}

	// Move time forward beyond expiry duration
	mockTime.Sleep(LimiterExpiryDuration + time.Second)

	// Run cleanup
	cleanupExpiredLimiters()

	// Verify the limiter was removed
	if _, found := limiters.Load(network); found {
		t.Fatal("Expected limiter to be removed after cleanup")
	}
}

// TestNetworkHistoryUpgradeDowngrade ensures that the limiter changes
// from regular to suspicious and vice versa after enough data is collected.
func TestNetworkHistoryUpgradeDowngrade(t *testing.T) {
	clearLimiters()
	setupLimiterTest(t)

	r := httptest.NewRequest("GET", "http://localhost", nil)
	r.RemoteAddr = "192.168.0.1:9999"
	c, err := newClient(r)
	if err != nil {
		t.Fatalf("newClient error: %v", err)
	}

	// Start as regular (not suspicious)
	c.limiter = getOrCreateLimiter(c.network.String(), false)
	if c.limiter == nil {
		t.Fatal("Limiter should not be nil")
	}
	if c.limiter.isSuspicious {
		t.Error("Limiter should start as non-suspicious")
	}

	// Fill up exactly MaxNetworkClientHistory entries with suspicious clients
	for range MaxNetworkClientHistory {
		c.markSuspicious()
		updateNetworkHistory(c.limiter, c.network.String(), c.isSuspicious)
	}

	if !c.limiter.isSuspicious {
		t.Error("Limiter should have downgraded to suspicious after enough suspicious clients")
	}

	// Now feed in enough non-suspicious entries to see if upgrade is triggered
	// We need suspicious ratio <= 0.2 to upgrade (given UpgradeThreshold=0.8).
	//
	// Add MaxNetworkClientHistory non-suspicious entries to overwrite all suspicious ones
	for range MaxNetworkClientHistory {
		c.clearSuspiciousStatus()
		updateNetworkHistory(c.limiter, c.network.String(), c.isSuspicious)
	}

	if c.limiter.isSuspicious {
		t.Error("Limiter should have upgraded to regular after all suspicious clients were replaced")
	}
}

// TestRateLimitDifferences checks that suspicious clients get lower limits
func TestRateLimitDifferences(t *testing.T) {
	clearLimiters()
	setupLimiterTest(t)

	r := httptest.NewRequest("GET", "http://localhost", nil)
	r.RemoteAddr = "192.168.0.1:9999"
	c, _ := newClient(r)

	// Get regular limiter
	regularLimiter := getOrCreateLimiter(c.network.String(), false)

	// Clear limiters and create a suspicious one
	clearLimiters()
	suspiciousLimiter := getOrCreateLimiter(c.network.String(), true)

	// Convert from rate.Limit to float64 for comparison
	regularRate := float64(regularLimiter.limiter.Limit())
	suspiciousRate := float64(suspiciousLimiter.limiter.Limit())

	if regularRate <= suspiciousRate {
		t.Errorf("Expected regular rate (%f) to be higher than suspicious rate (%f)",
			regularRate, suspiciousRate)
	}

	// Check burst values
	regularBurst := regularLimiter.limiter.Burst()
	suspiciousBurst := suspiciousLimiter.limiter.Burst()

	if regularBurst != RegularBurst {
		t.Errorf("Expected regular burst to be %d, got %d", RegularBurst, regularBurst)
	}

	if suspiciousBurst != SuspiciousBurst {
		t.Errorf("Expected suspicious burst to be %d, got %d", SuspiciousBurst, suspiciousBurst)
	}
}
