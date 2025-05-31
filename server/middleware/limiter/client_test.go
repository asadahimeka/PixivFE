// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package limiter

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestNewClient checks basic creation of a Client object.
func TestNewClient(t *testing.T) {
	setupLimiterTest(t)

	// Good IP
	r, _ := http.NewRequest("GET", "http://localhost/", nil)
	r.RemoteAddr = "192.168.0.1:9999"

	c, err := newClient(r)
	if err != nil {
		t.Fatalf("newClient returned error: %v", err)
	}
	if c.ip.String() != "192.168.0.1" {
		t.Errorf("Expected IP to be 192.168.0.1, got %s", c.ip.String())
	}
	if c.network.String() != "192.168.0.0/24" {
		t.Errorf("Expected network to be 192.168.0.0/24, got %s", c.network.String())
	}

	// Bad IP
	r, _ = http.NewRequest("GET", "http://localhost/", nil)
	r.RemoteAddr = "999.999.999.999:1234" // Invalid
	c, err = newClient(r)
	if err == nil {
		t.Error("Expected error for invalid IP, got nil")
	}
}

// TestCheckIPLists tests checkIPLists method, ensuring pass list and block list detection.
func TestCheckIPLists(t *testing.T) {
	setupLimiterTest(t)

	tests := []struct {
		ipStr       string
		expectPass  bool
		expectBlock bool
	}{
		{"127.0.0.1", true, false},
		{"10.0.0.1", false, true},
		{"192.168.0.1", false, false},
	}

	for _, tst := range tests {
		r := httptest.NewRequest("GET", "http://localhost/", nil)
		r.RemoteAddr = tst.ipStr + ":9999"

		clientObj, err := newClient(r)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		pass, block := clientObj.checkIPLists()
		if pass != tst.expectPass {
			t.Errorf("IP %s: wanted pass=%v, got %v", tst.ipStr, tst.expectPass, pass)
		}
		if block != tst.expectBlock {
			t.Errorf("IP %s: wanted block=%v, got %v", tst.ipStr, tst.expectBlock, block)
		}
	}
}

// TestValidateLinkToken checks the logic for linking token / suspicious IP logic via cookie checks.
func TestValidateLinkToken(t *testing.T) {
	setupLimiterTest(t)

	r := httptest.NewRequest("GET", "http://localhost/", nil)
	r.RemoteAddr = "192.168.0.1:9999"

	c, err := newClient(r)
	if err != nil {
		t.Fatalf("newClient returned error: %v", err)
	}

	// Case 1: No ping cookie
	result := c.validatePingCookie(r)
	if result {
		t.Error("Expected validateLinkToken to return false when no ping cookie exists")
	}
	if !c.isSuspicious {
		t.Error("Expected client to be marked suspicious when no ping cookie exists")
	}
	if c.limiter == nil {
		t.Error("Expected rate limiter to be created")
	}

	// Case 2: Provide a valid ping cookie
	cookie := createTestPingCookie(r)
	r.AddCookie(cookie)

	result = c.validatePingCookie(r)
	if !result {
		t.Error("Expected validateLinkToken to return true when a valid ping cookie exists")
	}
	if c.isSuspicious {
		t.Error("Expected client to not be suspicious with a valid ping cookie")
	}
	if c.limiter == nil {
		t.Error("Expected rate limiter to be created")
	}
}

// TestIsFullyExcludedPath verifies if fullyExcludedPaths is checked correctly.
func TestIsFullyExcludedPath(t *testing.T) {
	setupLimiterTest(t)

	tests := []struct {
		path         string
		expectResult bool
	}{
		{"/limiter/", true},
		{"/robots.txt", true},
		{"/some/other/path", false},
	}

	for _, tst := range tests {
		r := httptest.NewRequest("GET", "http://localhost"+tst.path, nil)
		r.RemoteAddr = "127.0.0.1:9999"

		c, err := newClient(r)
		if err != nil {
			t.Fatalf("newClient error: %v", err)
		}
		got := c.isFullyExcludedPath(r)
		if got != tst.expectResult {
			t.Errorf("Path %s: expected isFullyExcludedPath=%v, got %v", tst.path, tst.expectResult, got)
		}
	}
}

// TestIsPassListed checks if an IP is recognized as pass-listed.
func TestIsPassListed(t *testing.T) {
	setupLimiterTest(t)

	// Configure test-specific pass list
	setTestPassIPs([]string{"192.168.0.1"})

	r := httptest.NewRequest("GET", "http://localhost", nil)
	r.RemoteAddr = "192.168.0.1:9999"

	c, err := newClient(r)
	if err != nil {
		t.Fatalf("newClient error: %v", err)
	}
	if !c.isPassListed() {
		t.Error("Expected IP 192.168.0.1 to be pass-listed")
	}
}

// TestIsBlockListed checks if an IP is recognized as block-listed.
func TestIsBlockListed(t *testing.T) {
	setupLimiterTest(t)

	// Configure test-specific block list
	setTestBlockIPs([]string{"192.168.0.1"})

	r := httptest.NewRequest("GET", "http://localhost", nil)
	r.RemoteAddr = "192.168.0.1:9999"

	c, err := newClient(r)
	if err != nil {
		t.Fatalf("newClient error: %v", err)
	}
	if !c.isBlockListed() {
		t.Error("Expected IP 192.168.0.1 to be block-listed")
	}
}

// TestIsLocalLink checks detection of link-local addresses.
func TestIsLocalLink(t *testing.T) {
	setupLimiterTest(t)

	// IPv4 link local: 169.254.x.x
	r := httptest.NewRequest("GET", "http://localhost", nil)
	r.RemoteAddr = "169.254.100.50:9999"
	c, _ := newClient(r)
	if !c.isLocalLink() {
		t.Error("Expected link-local (IPv4) to be flagged as local link")
	}

	// Non link-local
	r = httptest.NewRequest("GET", "http://localhost", nil)
	r.RemoteAddr = "192.168.0.1:9999"
	c, _ = newClient(r)
	if c.isLocalLink() {
		t.Error("192.168.0.1 is not link-local, but flagged as local link")
	}

	// IPv6 link-local: fe80::/10
	r = httptest.NewRequest("GET", "http://localhost", nil)
	r.RemoteAddr = "[fe80::1]:9999"
	c, _ = newClient(r)
	if !c.isLocalLink() {
		t.Error("Expected IPv6 fe80:: to be flagged as local link")
	}
}

// BlockedByHeaders tests live in check_headers_test.go.

// TestClearSuspiciousStatus ensures that clearing suspicious status works.
func TestClearSuspiciousStatus(t *testing.T) {
	setupLimiterTest(t)

	r := httptest.NewRequest("GET", "http://localhost", nil)
	r.RemoteAddr = "192.168.0.1:9999"

	c, err := newClient(r)
	if err != nil {
		t.Fatalf("newClient error: %v", err)
	}
	if c.isSuspicious {
		t.Error("New client should not be suspicious initially")
	}
	c.markSuspicious()
	if !c.isSuspicious {
		t.Error("Expected Suspicious to be true after markSuspicious()")
	}
	c.clearSuspiciousStatus()
	if c.isSuspicious {
		t.Error("Expected Suspicious to be false after clearSuspiciousStatus()")
	}
}

// TestMarkSuspicious checks that markSuspicious toggles the suspicious boolean.
func TestMarkSuspicious(t *testing.T) {
	setupLimiterTest(t)

	r := httptest.NewRequest("GET", "http://localhost", nil)
	r.RemoteAddr = "192.168.0.1:9999"

	c, err := newClient(r)
	if err != nil {
		t.Fatalf("newClient error: %v", err)
	}
	if c.isSuspicious {
		t.Error("New client should not be suspicious initially")
	}
	c.markSuspicious()
	if !c.isSuspicious {
		t.Error("Expected Suspicious to be true after markSuspicious()")
	}
}
