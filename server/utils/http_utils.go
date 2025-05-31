// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package utils

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/zeebo/xxh3"
)

const (
	GenerateETagBytes int = 8
)

// SendString writes a plain text response to the provided http.ResponseWriter.
// It sets the content type to "text/plain" and returns any error encountered during writing.
func SendString(w http.ResponseWriter, text string) error {
	w.Header().Set("Content-Type", "text/plain")

	_, err := w.Write([]byte(text))
	if err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	return nil
}

// RedirectTo performs a redirect to the specified path with optional query parameters.
// It uses HTTP status 303 (See Other) for the redirect.
func RedirectTo(w http.ResponseWriter, r *http.Request, path string, queryParams map[string]string) error {
	query := url.Values{}
	for k, v := range queryParams {
		query.Add(k, v)
	}

	http.Redirect(w, r, path+"?"+query.Encode(), http.StatusSeeOther)

	return nil
}

// RedirectToWhenceYouCame redirects the user back to the referring page if it's from the same origin.
//
// This helps prevent open redirects by checking the referrer against the current origin.
// If the referrer is not from the same origin, it responds with a 200 OK status.
//
// returnPath  Return to this URL. If empty, return to the referrer.
func RedirectToWhenceYouCame(w http.ResponseWriter, r *http.Request, returnPath string) {
	if returnPath == "" {
		referrer := r.Referer()
		if strings.HasPrefix(referrer, Origin(r)) {
			returnPath = referrer
		}
	}

	if returnPath == "" {
		w.WriteHeader(http.StatusOK)
	} else {
		http.Redirect(w, r, returnPath, http.StatusSeeOther)
	}
}

// Origin returns the origin (scheme + host) from an HTTP request.
// The scheme is determined by first checking the X-Forwarded-Proto header,
// then the TLS connection status, defaulting to "http" if neither is set.
// The result is returned in the format "scheme://host".
func Origin(r *http.Request) string {
	scheme := "http"

	// Check X-Forwarded-Proto header first
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if r.TLS != nil {
		scheme = "https"
	}

	return scheme + "://" + r.Host
}

// GenerateETag creates a strong ETag from content bytes.
func GenerateETag(content []byte) string {
	hash := xxh3.Hash(content)
	hashBytes := make([]byte, GenerateETagBytes)
	binary.LittleEndian.PutUint64(hashBytes, hash)

	return base64.RawURLEncoding.EncodeToString(hashBytes)
}
