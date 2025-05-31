// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"testing"
)

func TestIsBlockTag(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected bool
	}{
		{
			name:     "div tag",
			tag:      "<div>",
			expected: true,
		},
		{
			name:     "p tag",
			tag:      "<p>",
			expected: true,
		},
		{
			name:     "span tag (inline)",
			tag:      "<span>",
			expected: false,
		},
		{
			name:     "a tag (inline)",
			tag:      "<a>",
			expected: false,
		},
		{
			name:     "end tag",
			tag:      "</div>",
			expected: true,
		},
		{
			name:     "tag with attributes",
			tag:      `<div class="test">`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBlockTag(tt.tag)
			if result != tt.expected {
				t.Errorf("isBlockTag(%q) = %v, want %v", tt.tag, result, tt.expected)
			}
		})
	}
}

func TestFindAllSimple(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		content  string
		expected []Match
	}{
		{
			name:     "simple text",
			filename: "test.html",
			content:  "<p>Hello world</p>",
			expected: []Match{
				{File: "test.html", Msg: "Hello world", Offset: 0},
			},
		},
		{
			name:     "multiple block elements",
			filename: "test.html",
			content:  "<div>First</div><p>Second</p>",
			expected: []Match{
				{File: "test.html", Msg: "First", Offset: 0},
				{File: "test.html", Msg: "Second", Offset: 6},
			},
		},
		{
			name:     "empty content",
			filename: "test.html",
			content:  "<div></div>",
			expected: []Match{},
		},
		{
			name:     "text with attributes",
			filename: "test.html",
			content:  `<div class="test">Hello</div>`,
			expected: []Match{
				{File: "test.html", Msg: "Hello", Offset: 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var matches []Match
			findAllSimple(tt.filename, tt.content, func(m Match) {
				matches = append(matches, m)
			})

			if len(matches) != len(tt.expected) {
				t.Errorf("findAllSimple() found %d matches, want %d", len(matches), len(tt.expected))
				return
			}

			for i, expected := range tt.expected {
				if matches[i].File != expected.File || matches[i].Msg != expected.Msg {
					t.Errorf("findAllSimple() match[%d] = {File: %q, Msg: %q}, want {File: %q, Msg: %q}",
						i, matches[i].File, matches[i].Msg, expected.File, expected.Msg)
				}
			}
		})
	}
}

func TestLooksLikeTemplateOrComment(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "normal text",
			text:     "Hello world",
			expected: false,
		},
		{
			name:     "template syntax",
			text:     "{{ .Name }}",
			expected: true,
		},
		{
			name:     "HTML comment",
			text:     "<!-- comment -->",
			expected: true,
		},
		{
			name:     "arrow symbols",
			text:     "-->",
			expected: true,
		},
		{
			name:     "mostly punctuation",
			text:     "{{- if",
			expected: true,
		},
		{
			name:     "text with some symbols",
			text:     "Hello: world",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := looksLikeTemplateOrComment(tt.text)
			if result != tt.expected {
				t.Errorf("looksLikeTemplateOrComment(%q) = %v, want %v", tt.text, result, tt.expected)
			}
		})
	}
}
