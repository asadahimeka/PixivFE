// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
The ping cookie works alongside the link token to verify legitimate browsers.

When a real browser successfully fetches the CSS resource at /limiter/{token}.css, this cookie
is created and signed using HMAC-SHA256, which should then be attached to the client's future requests.

The cookie consists of a Unix timestamp and a client fingerprint.
*/
package limiter

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"codeberg.org/pixivfe/pixivfe/audit"
	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/server/session"
)

const (
	// pingCookieName is the name of the cookie used for storing ping status.
	pingCookieName string = "pixivfe-Ping"

	// pingCookieMaxAge is the max age of the ping cookie in seconds.
	pingCookieMaxAge int = 604800 // 7 days

	// pingCookiePath is the path where the cookie is valid.
	pingCookiePath string = "/"

	pingCookiePayloadParts int = 2
)

// pingCookieSecret is the HMAC secret used for signing cookies.
var pingCookieSecret = []byte(config.GlobalConfig.Limiter.PingHMAC)

// createPingCookie generates an opaque signed cookie.
func createPingCookie(r *http.Request) *http.Cookie {
	client, err := newClient(r)
	if err != nil {
		return nil
	}

	timestamp := time.Now().Unix()

	// Combine data and encode
	payload := fmt.Sprintf("%d:%s", timestamp, client.fingerprint)
	encodedPayload := base64.StdEncoding.EncodeToString([]byte(payload))

	// Sign the encoded payload
	signature := createSignature(encodedPayload)

	// Combine payload and signature into opaque token
	token := encodedPayload + "." + signature

	return &http.Cookie{
		Name:     pingCookieName,
		Value:    token,
		Path:     pingCookiePath,
		MaxAge:   pingCookieMaxAge,
		HttpOnly: true,
		Secure:   session.ShouldCookieBeSecure(r),
		SameSite: http.SameSiteStrictMode,
	}
}

// createSignature creates an HMAC signature for the provided payload.
func createSignature(payload string) string {
	mac := hmac.New(sha256.New, pingCookieSecret)
	mac.Write([]byte(payload))
	signature := mac.Sum(nil)

	return base64.StdEncoding.EncodeToString(signature)
}

// verifyPingCookie validates a ping cookie.
func verifyPingCookie(cookie *http.Cookie, r *http.Request) bool {
	if cookie == nil {
		return false
	}

	// Split token into payload and signature
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != pingCookiePayloadParts {
		audit.GlobalAuditor.Logger.Warnln("Invalid cookie format")

		return false
	}

	encodedPayload, signature := parts[0], parts[1]

	// Verify signature
	expectedSignature := createSignature(encodedPayload)
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		audit.GlobalAuditor.Logger.Warnln("Invalid cookie signature")

		return false
	}

	// Decode payload
	payloadBytes, err := base64.StdEncoding.DecodeString(encodedPayload)
	if err != nil {
		audit.GlobalAuditor.Logger.Warnln("Invalid payload encoding")

		return false
	}

	// Split decoded payload
	payloadParts := strings.Split(string(payloadBytes), ":")
	if len(payloadParts) != pingCookiePayloadParts {
		audit.GlobalAuditor.Logger.Warnln("Invalid payload format")

		return false
	}

	rawTimestamp, providedFingerprint := payloadParts[0], payloadParts[1]

	// Verify timestamp
	timestamp, err := strconv.ParseInt(rawTimestamp, 10, 64)
	if err != nil {
		audit.GlobalAuditor.Logger.Warnln("Invalid timestamp in cookie")

		return false
	}

	// Check if cookie is expired
	if time.Now().Unix()-timestamp > int64(pingCookieMaxAge) {
		audit.GlobalAuditor.Logger.Warnln("Expired cookie")

		return false
	}

	// Create a client and compare the fingerprint generated with the one provided in the cookie
	client, err := newClient(r)
	if err != nil {
		return false
	}

	if providedFingerprint != client.fingerprint {
		audit.GlobalAuditor.Logger.Warnln("Fingerprint mismatch")

		return false
	}

	return true
}
