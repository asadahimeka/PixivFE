// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package tokenmanager

import (
	"sync"
	"testing"
	"time"
)

// TestNewTokenManager verifies that the NewTokenManager function correctly initializes
// a TokenManager with the provided configuration.
func TestNewTokenManager(t *testing.T) {
	tokenValues := []string{"token1", "token2", "token3"}
	maxRetries := 5
	baseTimeout := 1000 * time.Millisecond
	maxBackoffTime := 32000 * time.Millisecond
	loadBalancingMethod := "round-robin"

	tm := NewTokenManager(tokenValues, maxRetries, baseTimeout, maxBackoffTime, loadBalancingMethod)

	// Check if the number of tokens matches the input
	if len(tm.tokens) != len(tokenValues) {
		t.Errorf("Expected %d tokens, got %d", len(tokenValues), len(tm.tokens))
	}

	// Verify that all configuration parameters are correctly set
	if tm.maxRetries != maxRetries {
		t.Errorf("Expected maxRetries to be %d, got %d", maxRetries, tm.maxRetries)
	}

	if tm.baseTimeout != baseTimeout {
		t.Errorf("Expected baseTimeout to be %v, got %v", baseTimeout, tm.baseTimeout)
	}

	if tm.maxBackoffTime != maxBackoffTime {
		t.Errorf("Expected maxBackoffTime to be %v, got %v", maxBackoffTime, tm.maxBackoffTime)
	}

	if tm.loadBalancingMethod != loadBalancingMethod {
		t.Errorf("Expected loadBalancingMethod to be %s, got %s", loadBalancingMethod, tm.loadBalancingMethod)
	}
}

// TestGetTokenAllMethods tests all implemented load balancing methods to ensure
// they behave as expected when selecting tokens.
func TestGetTokenAllMethods(t *testing.T) {
	tests := []struct {
		name                string
		loadBalancingMethod string
		expectedBehavior    func(*testing.T, *TokenManager)
	}{
		{
			name:                "Round Robin",
			loadBalancingMethod: "round-robin",
			expectedBehavior: func(t *testing.T, tm *TokenManager) {
				// Test if tokens are returned in a cyclic order
				for i := range len(tm.tokens) * 2 {
					token := tm.GetToken()
					expectedValue := tm.tokens[i%len(tm.tokens)].Value
					if token.Value != expectedValue {
						t.Errorf("Expected token value %s, got %s", expectedValue, token.Value)
					}
				}
			},
		},
		{
			name:                "Random",
			loadBalancingMethod: "random",
			expectedBehavior: func(t *testing.T, tm *TokenManager) {
				// Test if all tokens are used over multiple selections
				usedTokens := make(map[string]bool)
				for range len(tm.tokens) * 10 {
					token := tm.GetToken()
					usedTokens[token.Value] = true
				}
				if len(usedTokens) != len(tm.tokens) {
					t.Errorf("Random selection did not use all available tokens")
				}
			},
		},
		{
			name:                "Least Recently Used",
			loadBalancingMethod: "least-recently-used",
			expectedBehavior: func(t *testing.T, tm *TokenManager) {
				// Test if tokens are returned in order of least recent use
				firstToken := tm.GetToken()
				time.Sleep(10 * time.Millisecond)
				secondToken := tm.GetToken()
				time.Sleep(10 * time.Millisecond)
				thirdToken := tm.GetToken()

				if firstToken.Value == secondToken.Value || firstToken.Value == thirdToken.Value || secondToken.Value == thirdToken.Value {
					t.Errorf("Least-recently-used selection returned duplicate tokens")
				}
			},
		},
	}

	// Run tests for each load balancing method
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := NewTokenManager([]string{"token1", "token2", "token3"}, 5, 1000*time.Millisecond, 32000*time.Millisecond, tt.loadBalancingMethod)
			tt.expectedBehavior(t, tm)
		})
	}
}

// TestMarkTokenStatus verifies that the MarkTokenStatus method correctly
// updates a token's status and handles failure counts.
func TestMarkTokenStatus(t *testing.T) {
	tm := NewTokenManager([]string{"token1"}, 5, 1000*time.Millisecond, 32000*time.Millisecond, "round-robin")
	token := tm.GetToken()

	// Test marking a token as TimedOut
	tm.MarkTokenStatus(token, TimedOut)

	if token.Status != TimedOut {
		t.Errorf("Expected token status to be TimedOut, got %v", token.Status)
	}

	if token.FailureCount != 1 {
		t.Errorf("Expected failure count to be 1, got %d", token.FailureCount)
	}

	// Test marking a token as Good (should reset failure count)
	tm.MarkTokenStatus(token, Good)

	if token.Status != Good {
		t.Errorf("Expected token status to be Good, got %v", token.Status)
	}

	if token.FailureCount != 0 {
		t.Errorf("Expected failure count to be reset to 0, got %d", token.FailureCount)
	}
}

// TestResetAllTokens checks if the ResetAllTokens method correctly
// resets all tokens to their initial good state.
func TestResetAllTokens(t *testing.T) {
	tm := NewTokenManager([]string{"token1", "token2"}, 5, 1000*time.Millisecond, 32000*time.Millisecond, "round-robin")

	// Mark all tokens as TimedOut
	for _, token := range tm.tokens {
		tm.MarkTokenStatus(token, TimedOut)
	}

	// Reset all tokens
	tm.ResetAllTokens()

	// Check if all tokens are reset to Good status with 0 failure count
	for _, token := range tm.tokens {
		if token.Status != Good {
			t.Errorf("Expected all tokens to have Good status, got %v", token.Status)
		}

		if token.FailureCount != 0 {
			t.Errorf("Expected all tokens to have FailureCount 0, got %d", token.FailureCount)
		}
	}
}

// TestGetFallbackToken verifies that when all tokens are timed out,
// the TokenManager correctly selects and resets a fallback token.
func TestGetFallbackToken(t *testing.T) {
	tm := NewTokenManager([]string{"token1", "token2"}, 5, 1000*time.Millisecond, 32000*time.Millisecond, "round-robin")

	// Mark all tokens as timed out
	for _, token := range tm.tokens {
		tm.MarkTokenStatus(token, TimedOut)
		token.TimeoutUntil = time.Now().Add(-1000 * time.Millisecond) // Set timeout in the past
	}

	// Get a token, which should reset and return a previously timed-out token
	token := tm.GetToken()
	if token == nil {
		t.Error("Expected a fallback token, got nil")
	} else if token.Status != Good {
		t.Errorf("Expected fallback token status to be Good, got %v", token.Status)
	}
}

// TestExponentialBackoff checks if the exponential backoff mechanism
// correctly increases the timeout duration for consecutive failures.
func TestExponentialBackoff(t *testing.T) {
	tm := NewTokenManager([]string{"token1"}, 5, 1000*time.Millisecond, 8000*time.Millisecond, "round-robin")
	token := tm.GetToken()

	expectedTimeouts := []time.Duration{
		1000 * time.Millisecond,
		2000 * time.Millisecond,
		4000 * time.Millisecond,
		8000 * time.Millisecond,
		8000 * time.Millisecond,
	}

	for i, expected := range expectedTimeouts {
		tm.MarkTokenStatus(token, TimedOut)

		if time.Until(token.TimeoutUntil).Round(time.Millisecond) != expected {
			t.Errorf("Iteration %d: Expected timeout duration %v, got %v", i, expected, time.Until(token.TimeoutUntil).Round(time.Millisecond))
		}
	}
}

// TestConcurrentAccess verifies that the TokenManager can handle
// concurrent access from multiple goroutines without race conditions.
func TestConcurrentAccess(t *testing.T) {
	tm := NewTokenManager([]string{"token1", "token2", "token3"}, 5, 1000*time.Millisecond, 32000*time.Millisecond, "round-robin")

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			token := tm.GetToken()
			tm.MarkTokenStatus(token, Good)
		}()
	}

	wg.Wait()
}

// TestEmptyTokenList checks if the TokenManager correctly handles
// the case when initialized with an empty list of tokens.
func TestEmptyTokenList(t *testing.T) {
	tm := NewTokenManager([]string{}, 5, 1000*time.Millisecond, 32000*time.Millisecond, "round-robin")

	token := tm.GetToken()
	if token != nil {
		t.Errorf("Expected nil token for empty token list, got %v", token)
	}
}
