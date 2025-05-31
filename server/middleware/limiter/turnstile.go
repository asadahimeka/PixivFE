// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package limiter

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"codeberg.org/pixivfe/pixivfe/assets/components"
	"codeberg.org/pixivfe/pixivfe/audit"
	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/server/requestcontext"
	"codeberg.org/pixivfe/pixivfe/server/utils"
	"github.com/a-h/templ"
	"github.com/goccy/go-json"
)

// NOTE: ephemeral_id is unimplemented as it is Enterprise only.
type siteverifyResponse struct {
	Success     bool     `json:"success"`      // Whether the operation was successful or not
	ChallengeTS string   `json:"challenge_ts"` // ISO timestamp for the time the challenge was solved
	Hostname    string   `json:"hostname"`     // Hostname for which the challenge was served
	ErrorCodes  []string `json:"error-codes"`  // List of errors that occurred
	Action      string   `json:"action"`       // Customer widget identifier passed to the widget on the client side
	CData       string   `json:"cdata"`        // Customer data passed to the widget on the client side
}

// NOTE: idempotency_key is unimplemented as we don't retry failed requests.
type siteverifyRequest struct {
	Secret   string `json:"secret"`   // Widget's secret key
	Response string `json:"response"` // Response provided by the Turnstile client-side render
	RemoteIP string `json:"remoteip"` // Visitor's IP address
}

const siteverifyEndpoint = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

var (
	errFailedToSendVerificationRequest    = errors.New("failed to send turnstile verification request")
	errFailedToReadResponseBody           = errors.New("failed to read turnstile verification response body")
	errFailedToParseJSONResponse          = errors.New("failed to parse turnstile verification JSON response")
	errFailedToMarshalTurnstileRequest    = errors.New("failed to marshal turnstile request payload")
	errFailedToCreateTurnstileHTTPRequest = errors.New("failed to create turnstile HTTP request")
)

// HTTP handlers

// TurnstileActionHandler serves the initial Turnstile challenge or clears it if already verified.
func TurnstileActionHandler(w http.ResponseWriter, r *http.Request) error {
	client, err := newClient(r)
	if err != nil {
		return fmt.Errorf("failed to initialize client for turnstile: %w", err)
	}

	if client.validatePingCookie(r) {
		audit.GlobalAuditor.Logger.Infoln("Turnstile verification skipped, client already has valid ping cookie",
			"ip", client.ip.String(), "client_fingerprint", client.fingerprint)
		return components.TurnstileNoAction().Render(r.Context(), w)
	}

	sitekey := config.GlobalConfig.Limiter.TurnstileSitekey

	audit.GlobalAuditor.Logger.Infoln("Serving Turnstile challenge",
		"ip", client.ip.String(), "client_fingerprint", client.fingerprint)

	component := renderTurnstileInitialChallenge(sitekey)

	err = component.Render(r.Context(), w)
	if err != nil {
		audit.GlobalAuditor.Logger.Errorln("Failed to render turnstile challenge component", "error", err,
			"ip", client.ip.String(), "client_fingerprint", client.fingerprint)
		return fmt.Errorf("failed to render turnstile challenge component: %w", err)
	}
	return nil
}

// TurnstileVerifyHandler handles the verification of the Turnstile token submitted by the client.
//
//nolint:funlen
func TurnstileVerifyHandler(w http.ResponseWriter, r *http.Request) error {
	client, err := newClient(r)
	if err != nil {
		audit.GlobalAuditor.Logger.Errorln("Failed to initialize client for turnstile verification", "error", err)
		w.WriteHeader(http.StatusInternalServerError)

		component := renderTurnstileErrorGeneric(config.GlobalConfig.Limiter.TurnstileSitekey)
		return component.Render(r.Context(), w)
	}

	currentSiteKey := config.GlobalConfig.Limiter.TurnstileSitekey

	if err := r.ParseForm(); err != nil {
		audit.GlobalAuditor.Logger.Warnln("Failed to parse form",
			"error", err,
			"ip", client.ip.String(),
			"client_fingerprint", client.fingerprint)
		w.WriteHeader(http.StatusBadRequest)

		component := renderTurnstileErrorGeneric(currentSiteKey)
		return component.Render(r.Context(), w)
	}

	turnstileToken := r.FormValue("cf-turnstile-response")
	if turnstileToken == "" {
		audit.GlobalAuditor.Logger.Warnln("Missing turnstile token in form",
			"ip", client.ip.String(),
			"client_fingerprint", client.fingerprint)
		w.WriteHeader(http.StatusBadRequest)

		component := renderTurnstileErrorGeneric(currentSiteKey)
		return component.Render(r.Context(), w)
	}

	clientIPStr := client.ip.String()

	verified, err := verifyTurnstileToken(r, turnstileToken, clientIPStr)
	if err != nil {
		audit.GlobalAuditor.Logger.Errorln("Failed to verify turnstile token with Cloudflare",
			"error", err,
			"ip", clientIPStr,
			"client_fingerprint", client.fingerprint)

		var component templ.Component

		if errors.Is(err, errFailedToSendVerificationRequest) {
			w.WriteHeader(http.StatusServiceUnavailable)

			component = renderTurnstileErrorUpstream(currentSiteKey)
		} else {
			w.WriteHeader(http.StatusInternalServerError)

			component = renderTurnstileErrorGeneric(currentSiteKey)
		}
		return component.Render(r.Context(), w)
	}

	if !verified {
		audit.GlobalAuditor.Logger.Warnln("Invalid turnstile token reported by Cloudflare",
			"ip", clientIPStr,
			"client_fingerprint", client.fingerprint)
		w.WriteHeader(http.StatusBadRequest)

		component := renderTurnstileErrorGeneric(currentSiteKey)
		return component.Render(r.Context(), w)
	}

	cookie := createPingCookie(r)
	http.SetCookie(w, cookie)

	audit.GlobalAuditor.Logger.Infoln("Turnstile verification successful, ping cookie set",
		"ip", clientIPStr,
		"client_fingerprint", client.fingerprint)
	// Render success state, which will then auto-clear via hx-get to /limiter/turnstile/clear
	component := renderTurnstileSuccess(currentSiteKey)
	return component.Render(r.Context(), w)
}

// TurnstileClearHandler serves an empty div to clear the Turnstile UI after success display timeout.
func TurnstileClearHandler(w http.ResponseWriter, r *http.Request) error {
	return components.TurnstileNoAction().Render(r.Context(), w)
}

func verifyTurnstileToken(r *http.Request, token, remoteIP string) (bool, error) {
	start := time.Now()
	secretKey := config.GlobalConfig.Limiter.TurnstileSecretKey

	requestPayload := siteverifyRequest{Secret: secretKey, Response: token, RemoteIP: remoteIP}

	jsonData, err := json.Marshal(requestPayload)
	if err != nil {
		return false, fmt.Errorf("%w: %w", errFailedToMarshalTurnstileRequest, err)
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, siteverifyEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return false, fmt.Errorf("%w: %w", errFailedToCreateTurnstileHTTPRequest, err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := utils.HTTPClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("%w: %w", errFailedToSendVerificationRequest, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("%w: %w", errFailedToReadResponseBody, err)
	}

	var result siteverifyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("%w: %w", errFailedToParseJSONResponse, err)
	}

	span := audit.Span{
		Component:  audit.ComponentUpstream,
		Duration:   time.Since(start),
		RequestID:  requestcontext.FromContext(r.Context()).RequestID,
		Method:     req.Method,
		URL:        siteverifyEndpoint,
		StatusCode: resp.StatusCode,
		Error:      nil,
		Body:       body,
	}
	audit.GlobalAuditor.LogAndRecord(span)

	if !result.Success {
		audit.GlobalAuditor.Logger.Warnln("Turnstile verification failed as per Cloudflare API",
			"errors", result.ErrorCodes, "hostname", result.Hostname, "action", result.Action, "cdata", result.CData)
	}
	return result.Success, nil
}

// Helper functions for rendering Turnstile states.
func renderTurnstileInitialChallenge(sitekey string) templ.Component {
	messages := []string{
		"Verifying your browser...",
		"Please wait a moment. This process is automatic.",
	}
	return components.TurnstileChallenge(sitekey, messages, false, false)
}

func renderTurnstileSuccess(sitekey string) templ.Component {
	messages := []string{
		"Verification successful",
		"You can now continue.",
	}
	return components.TurnstileChallenge(sitekey, messages, false, true)
}

func renderTurnstileErrorGeneric(sitekey string) templ.Component {
	messages := []string{
		"Verification failed",
		"Your request could not be verified. Please refresh and try again.",
	}
	return components.TurnstileChallenge(sitekey, messages, true, false)
}

func renderTurnstileErrorUpstream(sitekey string) templ.Component {
	messages := []string{
		"Verification error",
		"Could not reach the verification service. Please refresh and try again shortly.",
	}
	return components.TurnstileChallenge(sitekey, messages, true, false)
}
