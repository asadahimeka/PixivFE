// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"testing"
)

func TestSplit(t *testing.T) {
	tests := []struct {
		name     string
		input    Match
		expected []Match
	}{
		{
			name: "single line text",
			input: Match{
				File:   "test.html",
				Msg:    "Hello World",
				Offset: 100,
			},
			expected: []Match{
				{File: "test.html", Msg: "Hello World", Offset: 100},
			},
		},
		{
			name: "multiline text",
			input: Match{
				File:   "test.html",
				Msg:    "First line\nSecond line",
				Offset: 100,
			},
			expected: []Match{
				{File: "test.html", Msg: "First line", Offset: 100},
				{File: "test.html", Msg: "Second line", Offset: 111},
			},
		},
		{
			name: "multiline with empty lines",
			input: Match{
				File:   "test.html",
				Msg:    "First line\n\nThird line",
				Offset: 50,
			},
			expected: []Match{
				{File: "test.html", Msg: "First line", Offset: 50},
				{File: "test.html", Msg: "Third line", Offset: 62},
			},
		},
		{
			name: "text with leading/trailing whitespace",
			input: Match{
				File:   "test.html",
				Msg:    "  First line  \n  Second line  ",
				Offset: 10,
			},
			expected: []Match{
				{File: "test.html", Msg: "First line", Offset: 12},
				{File: "test.html", Msg: "Second line", Offset: 27},
			},
		},
		{
			name: "only whitespace lines",
			input: Match{
				File:   "test.html",
				Msg:    "   \n\t\n   ",
				Offset: 0,
			},
			expected: []Match{},
		},
		{
			name: "mixed content with empty lines",
			input: Match{
				File:   "test.html",
				Msg:    "Line 1\n\n  \nLine 4\n\nLine 6",
				Offset: 200,
			},
			expected: []Match{
				{File: "test.html", Msg: "Line 1", Offset: 200},
				{File: "test.html", Msg: "Line 4", Offset: 211},
				{File: "test.html", Msg: "Line 6", Offset: 219},
			},
		},
		{
			name: "single empty string",
			input: Match{
				File:   "test.html",
				Msg:    "",
				Offset: 0,
			},
			expected: []Match{},
		},
		{
			name: "newline only",
			input: Match{
				File:   "test.html",
				Msg:    "\n",
				Offset: 5,
			},
			expected: []Match{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := split(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d matches, got %d", len(tt.expected), len(result))
				t.Errorf("Expected: %+v", tt.expected)
				t.Errorf("Got: %+v", result)
				return
			}

			for i, expected := range tt.expected {
				if i >= len(result) {
					t.Errorf("Missing result %d: expected %+v", i, expected)
					continue
				}

				actual := result[i]
				if actual.File != expected.File {
					t.Errorf("Result %d: expected file %q, got %q", i, expected.File, actual.File)
				}
				if actual.Msg != expected.Msg {
					t.Errorf("Result %d: expected message %q, got %q", i, expected.Msg, actual.Msg)
				}
				if actual.Offset != expected.Offset {
					t.Errorf("Result %d: expected offset %d, got %d", i, expected.Offset, actual.Offset)
				}
			}
		})
	}
}
