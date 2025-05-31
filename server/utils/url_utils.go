// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package utils

import (
	"net/url"
	"strings"

	"codeberg.org/pixivfe/pixivfe/i18n"
)

// ValidateURL checks if the given URL is valid.
func ValidateURL(urlString string, urlType string) (*url.URL, error) {
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return nil, i18n.Errorf("failed to parse %s URL: %w", urlType, err)
	}

	// Ensure both scheme and host are present in the URL
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, i18n.Errorf(
			"%s URL is invalid: %s. Please specify a complete URL with scheme and host, e.g. https://example.com",
			urlType,
			urlString)
	}

	if strings.HasSuffix(parsedURL.Path, "/") {
		return nil, i18n.Errorf(
			"%s URL path (%s) cannot end in /: %s. PixivFE does not support this now",
			urlType,
			parsedURL.Path,
			urlString)
	}

	return parsedURL, nil
}
