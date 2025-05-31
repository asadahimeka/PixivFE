// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package limiter

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// resetTokenStorage is a helper to reset the global token storage between tests.
func resetTokenStorage() {
	tokenStorage = &linkTokenStorage{
		tokens: make(map[string]tokenEntry),
	}
}

func TestLinkTokenHandler(t *testing.T) {
	setupLimiterTest(t)

	resetTokenStorage()

	t.Run("Valid token", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/token.css", nil)
		r.Header.Set("X-Real-IP", "1.1.1.1")
		r.Header.Set("User-Agent", "test-agent")

		// Create a client to get the fingerprint
		client, err := newClient(r)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		// Create token with the calculated fingerprint
		css := []byte("/* test CSS */")
		token := tokenStorage.createToken(client.fingerprint, css)

		r = httptest.NewRequest(http.MethodGet, "/"+token+".css", nil)
		r.Header.Set("X-Real-IP", "1.1.1.1")
		r.Header.Set("User-Agent", "test-agent")
		r = mux.SetURLVars(r, map[string]string{"token": token})

		w := httptest.NewRecorder()
		if err := LinkTokenHandler(w, r); err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Got status %d, want %d", w.Code, http.StatusOK)
		}

		// Check that we got CSS and an empty body is not returned
		if ct := w.Header().Get("Content-Type"); ct != "text/css" {
			t.Errorf("Expected text/css content type, got %q", ct)
		}
		if len(w.Body.Bytes()) == 0 {
			t.Error("Expected CSS content, got empty response")
		}

		// After consumption, token should no longer exist (try consuming again)
		_, valid := tokenStorage.consumeToken(token, client.fingerprint)
		if valid {
			t.Error("Token should have been consumed and removed, but can still be consumed")
		}
	})

	t.Run("Invalid token", func(t *testing.T) {
		resetTokenStorage()
		token := "DOES_NOT_EXIST"

		r := httptest.NewRequest(http.MethodGet, "/"+token+".css", nil)
		r.Header.Set("X-Real-IP", "1.1.1.1")
		r.Header.Set("User-Agent", "test-agent")
		r = mux.SetURLVars(r, map[string]string{"token": token})

		w := httptest.NewRecorder()
		err := LinkTokenHandler(w, r)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404 for invalid token, got %d", w.Code)
		}
		if err == nil {
			t.Error("Expected an error for invalid token, got none")
		}
	})

	t.Run("Fingerprint mismatch", func(t *testing.T) {
		resetTokenStorage()
		// Create a token for one fingerprint
		fingerprint := "mismatch-fprint"
		token := tokenStorage.createToken(fingerprint, []byte("/* mismatch test */"))

		// But pass different IP/User-Agent so we generate a different fingerprint
		r := httptest.NewRequest(http.MethodGet, "/"+token+".css", nil)
		r.Header.Set("X-Real-IP", "2.2.2.2") // different IP => different fingerprint
		r.Header.Set("User-Agent", "test-agent")
		r = mux.SetURLVars(r, map[string]string{"token": token})

		w := httptest.NewRecorder()
		err := LinkTokenHandler(w, r)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404 for fingerprint mismatch, got %d", w.Code)
		}
		if err == nil {
			t.Error("Expected an error for fingerprint mismatch, got none")
		}
	})

	t.Run("Expired token", func(t *testing.T) {
		mockTime := setupLimiterTest(t)

		resetTokenStorage()

		fingerprint := "expire-fprint"
		token := tokenStorage.createToken(fingerprint, []byte("/* expiry test */"))

		// Wait until the token has expired
		mockTime.Sleep(61 * time.Second)

		// Attempt to consume
		r := httptest.NewRequest(http.MethodGet, "/"+token+".css", nil)
		r.Header.Set("X-Real-IP", "1.1.1.1")
		r.Header.Set("User-Agent", "test-agent")
		r = mux.SetURLVars(r, map[string]string{"token": token})

		w := httptest.NewRecorder()
		err := LinkTokenHandler(w, r)
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404 for expired token, got %d", w.Code)
		}
		if err == nil {
			t.Error("Expected error for expired token, got none")
		}
	})
}

func TestGetOrCreateLinkToken(t *testing.T) {
	resetTokenStorage()

	t.Run("Generate new token", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("X-Real-IP", "1.1.1.1")
		r.Header.Set("User-Agent", "test-agent")

		token, err := GetOrCreateLinkToken(r)
		if err != nil {
			t.Fatalf("Error generating token: %v", err)
		}
		if token == "" {
			t.Error("Expected non-empty token")
		}

		// Ensure token is in storage by trying to consume it
		client, err := newClient(r)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		_, valid := tokenStorage.consumeToken(token, client.fingerprint)
		if !valid {
			t.Error("Token should be valid and consumable")
		}
	})

	t.Run("Always generate fresh tokens", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("X-Real-IP", "1.1.1.1")
		r.Header.Set("User-Agent", "test-agent")

		token1, err := GetOrCreateLinkToken(r)
		if err != nil {
			t.Fatalf("Error generating first token: %v", err)
		}
		token2, err := GetOrCreateLinkToken(r)
		if err != nil {
			t.Fatalf("Error generating second token: %v", err)
		}
		if token1 == token2 {
			t.Error("Expected different tokens for each call, got the same value")
		}
	})
}

func TestUtilityFunctions(t *testing.T) {
	t.Run("generateCSSContent returns valid CSS", func(t *testing.T) {
		content, err := generateCSSContent()
		if err != nil {
			t.Fatalf("Failed to generate CSS content: %v", err)
		}
		if len(content) == 0 {
			t.Error("Expected non-empty CSS content")
		}

		// Check that it looks like CSS with a comment
		contentStr := string(content)
		if !strings.HasPrefix(contentStr, "/*") || !strings.HasSuffix(contentStr, "*/") {
			t.Errorf("Expected CSS comment format, got %q", contentStr)
		}
	})
}

func TestLinkTokenStorage(t *testing.T) {
	setupLimiterTest(t)
	resetTokenStorage()

	t.Run("Token storage and retrieval", func(t *testing.T) {
		fingerprint := "test-fingerprint"
		cssContent := []byte("/* test content */")

		// Create token
		token := tokenStorage.createToken(fingerprint, cssContent)
		if token == "" {
			t.Error("Expected non-empty token")
		}

		// Consume token
		retrievedCSS, valid := tokenStorage.consumeToken(token, fingerprint)
		if !valid {
			t.Error("Expected token to be valid")
		}
		if string(retrievedCSS) != string(cssContent) {
			t.Errorf("Expected CSS content %q, got %q", string(cssContent), string(retrievedCSS))
		}

		// Try to consume again - should fail
		_, valid = tokenStorage.consumeToken(token, fingerprint)
		if valid {
			t.Error("Token should be consumed and no longer valid")
		}
	})

	t.Run("Token expiry", func(t *testing.T) {
		mockTime := setupLimiterTest(t)
		resetTokenStorage()

		fingerprint := "expire-test"
		token := tokenStorage.createToken(fingerprint, []byte("/* expiry */"))

		// Advance time beyond expiry
		mockTime.Sleep(time.Duration(linkTokenLiveSeconds+1) * time.Second)

		// Should not be consumable
		_, valid := tokenStorage.consumeToken(token, fingerprint)
		if valid {
			t.Error("Expired token should not be valid")
		}
	})

	t.Run("Concurrent token operations", func(t *testing.T) {
		resetTokenStorage()

		// Test concurrent creation and consumption
		const numGoroutines = 10
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer func() { done <- true }()

				fingerprint := fmt.Sprintf("concurrent-test-%d", id)
				token := tokenStorage.createToken(fingerprint, []byte(fmt.Sprintf("/* content %d */", id)))

				// Immediately consume
				_, valid := tokenStorage.consumeToken(token, fingerprint)
				if !valid {
					t.Errorf("Token should be valid for goroutine %d", id)
				}
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})
}

func TestLinkTokenCleanup(t *testing.T) {
	mockTime := setupLimiterTest(t)
	resetTokenStorage()

	t.Run("Cleanup expired tokens", func(t *testing.T) {
		// Create some tokens
		fingerprint1 := "cleanup-test-1"
		fingerprint2 := "cleanup-test-2"

		token1 := tokenStorage.createToken(fingerprint1, []byte("/* content 1 */"))
		token2 := tokenStorage.createToken(fingerprint2, []byte("/* content 2 */"))

		// Verify both tokens exist
		_, valid1 := tokenStorage.consumeToken(token1, fingerprint1)
		if !valid1 {
			t.Error("Token 1 should be valid before expiry")
		}

		// Recreate token1 since it was consumed
		token1 = tokenStorage.createToken(fingerprint1, []byte("/* content 1 */"))

		// Advance time to expire tokens
		mockTime.Sleep(time.Duration(linkTokenLiveSeconds+1) * time.Second)

		// Run cleanup
		tokenStorage.cleanupExpiredTokens()

		// Both tokens should now be invalid
		_, valid1 = tokenStorage.consumeToken(token1, fingerprint1)
		_, valid2 := tokenStorage.consumeToken(token2, fingerprint2)

		if valid1 || valid2 {
			t.Error("Expired tokens should be cleaned up and invalid")
		}
	})
}

func TestLinkTokenEdgeCases(t *testing.T) {
	setupLimiterTest(t)
	resetTokenStorage()

	t.Run("Empty fingerprint", func(t *testing.T) {
		token := tokenStorage.createToken("", []byte("/* empty fingerprint */"))

		// Should still work with empty fingerprint
		_, valid := tokenStorage.consumeToken(token, "")
		if !valid {
			t.Error("Token with empty fingerprint should still be valid")
		}
	})

	t.Run("Wrong fingerprint", func(t *testing.T) {
		token := tokenStorage.createToken("correct-fingerprint", []byte("/* test */"))

		// Try to consume with wrong fingerprint
		_, valid := tokenStorage.consumeToken(token, "wrong-fingerprint")
		if valid {
			t.Error("Token should not be valid with wrong fingerprint")
		}

		// Should still be consumable with correct fingerprint
		_, valid = tokenStorage.consumeToken(token, "correct-fingerprint")
		if !valid {
			t.Error("Token should still be valid with correct fingerprint")
		}
	})

	t.Run("Non-existent token", func(t *testing.T) {
		_, valid := tokenStorage.consumeToken("non-existent-token", "any-fingerprint")
		if valid {
			t.Error("Non-existent token should not be valid")
		}
	})
}
