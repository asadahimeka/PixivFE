// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"testing"
)

func TestShouldIgnore(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: true,
		},
		{
			name:     "right arrow",
			input:    "»",
			expected: true,
		},
		{
			name:     "play symbol",
			input:    "▶",
			expected: true,
		},
		{
			name:     "pages template",
			input:    "⧉ {{ .Pages }}",
			expected: true,
		},
		{
			name:     "PixivFE brand name",
			input:    "PixivFE",
			expected: true,
		},
		{
			name:     "pixiv URL template",
			input:    "pixiv.net/i/{{ .ID }}",
			expected: true,
		},
		{
			name:     "normal text should not be ignored",
			input:    "Hello World",
			expected: false,
		},
		{
			name:     "case sensitive - pixivfe lowercase",
			input:    "pixivfe",
			expected: false,
		},
		{
			name:     "different template",
			input:    "{{ .Title }}",
			expected: false,
		},
		{
			name:     "partial match should not be ignored",
			input:    "PixivFE is awesome",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldIgnore(tt.input)
			if result != tt.expected {
				t.Errorf("shouldIgnore(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIgnoreTheseStringsMap(t *testing.T) {
	// Test that all expected strings are in the ignore map
	expectedIgnored := []string{
		"",
		"»",
		"▶",
		"⧉ {{ .Pages }}",
		"PixivFE",
		"pixiv.net/i/{{ .ID }}",
	}

	for _, str := range expectedIgnored {
		if !IgnoreTheseStrings[str] {
			t.Errorf("String %q should be in IgnoreTheseStrings map but is not", str)
		}
	}

	// Test that the map doesn't contain unexpected entries (at least some basic ones)
	unexpectedStrings := []string{
		"Hello",
		"World",
		"test",
		"content",
	}

	for _, str := range unexpectedStrings {
		if IgnoreTheseStrings[str] {
			t.Errorf("String %q should not be in IgnoreTheseStrings map but is present", str)
		}
	}
}
