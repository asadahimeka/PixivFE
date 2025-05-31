// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
Helpers used for testing
*/
package limiter

import (
	"net/http"
	"testing"
	"time"

	"codeberg.org/pixivfe/pixivfe/audit"
	"codeberg.org/pixivfe/pixivfe/config"
)

// mockTimeProvider implements the timeProvider interface
// and maintains a controllable current time for testing.
type mockTimeProvider struct {
	currentTime time.Time
}

// newMockTimeProvider creates a new mockTimeProvider initialized with the given time.
//
// This allows tests to start from a specific point in time.
func newMockTimeProvider(initialTime time.Time) *mockTimeProvider {
	return &mockTimeProvider{
		currentTime: initialTime,
	}
}

// Now returns the current mock time.
func (m *mockTimeProvider) Now() time.Time {
	return m.currentTime
}

// Sleep advances the mock current time by the specified duration.
func (m *mockTimeProvider) Sleep(d time.Duration) {
	m.currentTime = m.currentTime.Add(d)
}

// setupLimiterTest prepares a test environment with a mock time provider
// and properly configured limiter settings for testing.
//
// It returns the mock time provider.
//
// The original time function and config are restored when the test completes.
func setupLimiterTest(t *testing.T) *mockTimeProvider {
	audit.NewTestingLogger(t)

	mockTime := newMockTimeProvider(time.Now())

	// Hook the mock time provider to the token bucket's timeNow
	origTimeNow := timeNow // Save original for restoring after tests
	timeNow = func() time.Time {
		return mockTime.Now()
	}

	// Set up test configuration with proper defaults
	setupTestConfig(t)

	// Register cleanup to restore the original timeNow function and config
	t.Cleanup(func() {
		timeNow = origTimeNow
	})

	return mockTime
}

// setupTestConfig configures the global config with test-appropriate values
// for the limiter functionality.
func setupTestConfig(t *testing.T) {
	// Save original config
	origConfig := config.GlobalConfig

	// Set up test configuration with basic defaults
	config.GlobalConfig.Limiter.Enabled = true
	config.GlobalConfig.Limiter.IPv4Prefix = 24                 // Default /24 for IPv4
	config.GlobalConfig.Limiter.IPv6Prefix = 64                 // Default /64 for IPv6
	config.GlobalConfig.Limiter.PassIPs = []string{"127.0.0.1"} // Basic pass IP for other tests
	config.GlobalConfig.Limiter.BlockIPs = []string{"10.0.0.1"} // Basic block IP for other tests
	config.GlobalConfig.Limiter.FilterLocal = false
	config.GlobalConfig.Limiter.CheckHeaders = true
	config.GlobalConfig.Limiter.PingHMAC = "test-secret-key-for-ping-cookie-signing-that-is-long-enough"

	// Restore original config on cleanup
	t.Cleanup(func() {
		config.GlobalConfig = origConfig
	})
}

// setTestPassIPs is a helper to set pass IPs for specific tests
func setTestPassIPs(ips []string) {
	config.GlobalConfig.Limiter.PassIPs = ips
}

// setTestBlockIPs is a helper to set block IPs for specific tests
func setTestBlockIPs(ips []string) {
	config.GlobalConfig.Limiter.BlockIPs = ips
}

// createTestPingCookie creates a valid ping cookie for testing purposes
func createTestPingCookie(r *http.Request) *http.Cookie {
	return createPingCookie(r)
}
