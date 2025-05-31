// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
The link token is embedded in HTML pages as /limiter/{token}.css, which is fetched by real browsers
as a simple proof that they are not naive bots (i.e., bots that can pass our basic HTTP request
header checks, but cannot fetch CSS resources linked to by the HTML and attach the generated
ping cookie to their future requests).

A link token is:
  - client-specific (tied to a fingerprint);
  - single-use;
  - and short-lived.

After a token is used to generate a ping cookie, it is immediately invalidated to prevent replay attacks.

This approach obviously won't prevent any bots that are remotely sophisticated that can properly mimic browser
behavior (think properly tuned playwright or selenium), but should stop some dude flooding an instance using
python-requests; if this actually becomes relevant in our threat model, we could use Cloudflare Turnstile for
more robust challenges, but this of course disadvantages users without JavaScript enabled.
*/
package limiter

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"codeberg.org/pixivfe/pixivfe/audit"
	"github.com/gorilla/mux"
)

const (
	// linkTokenLiveSeconds is the TTL for the link token.
	//
	// Short-lived as a link token should be consumed soon after being generated.
	linkTokenLiveSeconds = 60

	// tokenCleanupInterval is the interval between token cleanup runs.
	tokenCleanupInterval = 5 * time.Minute

	tokenBytesLength         int = 24
	randomCommentBytesLength int = 16
)

// tokenStorage is a global in-memory token storage.
var tokenStorage = &linkTokenStorage{
	tokens: make(map[string]tokenEntry),
}

var errInvalidOrExpiredToken = errors.New("invalid or expired token")

// tokenEntry represents a single token with its metadata.
type tokenEntry struct {
	token             string    // The actual token string
	expiresAt         time.Time // When this token expires
	clientFingerprint string    // Associated client fingerprint
	cssContent        []byte    // The CSS content to serve; used for SRI
}

// linkTokenStorage now holds multiple tokens in memory, each associated with a client fingerprint.
type linkTokenStorage struct {
	tokens map[string]tokenEntry // Map of token to token entries
	mu     sync.RWMutex
}

// createToken creates a new token for a specific client fingerprint.
func (s *linkTokenStorage) createToken(fingerprint string, cssContent []byte) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokenBytes := make([]byte, tokenBytesLength)

	_, err := rand.Read(tokenBytes)
	if err != nil {
		audit.GlobalAuditor.Logger.Errorln("Failed to generate secure token", "error", err)

		tokenBytes = fmt.Appendf(tokenBytes, "%d-%s", timeNow().UnixNano(), fingerprint)
	}

	// Encode as URL-safe base64
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Store the new token
	s.tokens[token] = tokenEntry{
		token:             token,
		expiresAt:         timeNow().Add(time.Second * time.Duration(linkTokenLiveSeconds)),
		clientFingerprint: fingerprint,
		cssContent:        cssContent,
	}

	return token
}

// consumeToken validates and removes a token if it matches the provided fingerprint.
func (s *linkTokenStorage) consumeToken(token string, fingerprint string) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.tokens[token]
	if !exists || timeNow().After(entry.expiresAt) {
		delete(s.tokens, token)
		return nil, false
	}

	// Check if the fingerprint matches
	if entry.clientFingerprint != fingerprint {
		audit.GlobalAuditor.Logger.Warnln(
			"Fingerprint mismatch for token",
			"token", token,
			"expected", entry.clientFingerprint,
			"received", fingerprint,
		)
		return nil, false
	}

	cssContent := entry.cssContent

	// Token is valid and matches fingerprint - consume it by removing
	delete(s.tokens, token)

	return cssContent, true
}

// cleanupExpiredTokens removes all expired tokens from storage.
func (s *linkTokenStorage) cleanupExpiredTokens() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := timeNow()
	expiredCount := 0

	for token, entry := range s.tokens {
		if now.After(entry.expiresAt) {
			delete(s.tokens, token)

			expiredCount++
		}
	}

	if expiredCount > 0 {
		audit.GlobalAuditor.Logger.Infoln("Cleaned up expired tokens", "count", expiredCount, "remaining", len(s.tokens))
	}
}

// LinkTokenHandler is the route that sets a signed cookie if the token is valid.
//
// This is analogous to SearXNG's "/client{token}.css" endpoint in botdetection/link_token.py.
func LinkTokenHandler(w http.ResponseWriter, r *http.Request) error {
	client, err := newClient(r)
	if err != nil {
		return err
	}

	// If the client already has a valid ping cookie, skip
	// the rest of the logic and return HTTP 204 No Content.
	if client.validatePingCookie(r) {
		w.Header().Set("Content-Type", "text/css")
		w.WriteHeader(http.StatusNoContent)
		return nil
	}

	tokenParam := mux.Vars(r)["token"]
	audit.GlobalAuditor.Logger.Debugln("Validating link token", "token", tokenParam)

	// Consume the token for the client fingerprint
	cssContent, valid := tokenStorage.consumeToken(tokenParam, client.fingerprint)
	if !valid {
		audit.GlobalAuditor.Logger.Warnln("Invalid or expired token", "token", tokenParam)
		w.WriteHeader(http.StatusNotFound)
		return fmt.Errorf("%w: %q", errInvalidOrExpiredToken, tokenParam)
	}

	// Valid token => create and set a signed cookie
	cookie := createPingCookie(r)
	http.SetCookie(w, cookie)

	w.Header().Set("Content-Type", "text/css")
	w.Header().Set("Cache-Control", "no-store") // clients should not cache this resource
	_, _ = w.Write(cssContent)
	return nil
}

// GetOrCreateLinkToken creates a new client-specific link token and returns the token plus an integrity attribute.
//
// A fresh token is always generated for each request, scoped to the client's fingerprint.
func GetOrCreateLinkToken(r *http.Request) (string, error) {
	client, err := newClient(r)
	if err != nil {
		return "", fmt.Errorf("failed to create client: %w", err)
	}

	cssContent, err := generateCSSContent()
	if err != nil {
		return "", fmt.Errorf("failed to generate CSS content: %w", err)
	}

	token := tokenStorage.createToken(client.fingerprint, cssContent)

	// Return the new token
	return token, nil
}

// generateCSSContent creates CSS content with a random comment.
func generateCSSContent() ([]byte, error) {
	// Generate random bytes for the CSS comment
	randomBytes := make([]byte, randomCommentBytesLength)

	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	randomComment := base64.StdEncoding.EncodeToString(randomBytes)

	cssContent := fmt.Appendf([]byte{}, "/* %s */", randomComment)

	return cssContent, nil
}

// initLinkTokenCleanup starts a goroutine to periodically clean up expired link tokens.
func initLinkTokenCleanup() {
	go func() {
		ticker := time.NewTicker(tokenCleanupInterval)
		defer ticker.Stop()

		for range ticker.C {
			tokenStorage.cleanupExpiredTokens()
		}
	}()

	audit.GlobalAuditor.Logger.Infoln("Link token cleanup initialized",
		"interval", tokenCleanupInterval.String())
}
