// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"regexp"
	"slices"
	"strings"
)

// Simple regex-based HTML parser that extracts text content
var (
	// Regex to match HTML tags
	htmlTagRegex = regexp.MustCompile(`<[^>]*>`)
	// Regex to match whitespace sequences
	whitespaceRegex = regexp.MustCompile(`\s+`)
	// Regex to match HTML comments
	htmlCommentRegex = regexp.MustCompile(`<!--[\s\S]*?-->`)
	// Regex to match template control structures that should be removed
	templateControlRegex = regexp.MustCompile(`\{\{\s*(?:range|end|if|else|block|extends|include|yield)\b[\s\S]*?\}\}`)
)

// removeTemplateControlStructures removes template control flow but preserves variables in text
func removeTemplateControlStructures(content string) string {
	// Remove template control structures like {{ range }}, {{ end }}, {{ if }}, etc.
	result := templateControlRegex.ReplaceAllString(content, "")
	return result
}

// findAllSimple scans through HTML content using regex to extract text blocks
func findAllSimple(filename string, content string, onMatch func(match Match)) {
	// Remove HTML comments first
	content = htmlCommentRegex.ReplaceAllString(content, "")

	// Remove template control structures but preserve template variables in text
	content = removeTemplateControlStructures(content)

	// Remove HTML tags to get text content, but keep some structure
	textContent := htmlTagRegex.ReplaceAllStringFunc(content, func(tag string) string {
		// Convert block-level tags to newlines to preserve text separation
		if isBlockTag(tag) {
			return "\n"
		}
		return " " // Inline tags become spaces
	})

	// Normalize whitespace but preserve line breaks
	lines := strings.Split(textContent, "\n")

	offset := 0
	for _, line := range lines {
		// Clean up the line
		line = strings.TrimSpace(line)
		line = whitespaceRegex.ReplaceAllString(line, " ")

		// Skip empty lines and lines that look like template syntax or comments
		if line != "" && !shouldIgnore(line) && !looksLikeTemplateOrComment(line) {
			onMatch(Match{
				File:   filename,
				Msg:    line,
				Offset: offset,
			})
		}
		offset += len(line) + 1
	}
}

// looksLikeTemplateOrComment checks if the text looks like template syntax or comments
func looksLikeTemplateOrComment(text string) bool {
	trimmed := strings.TrimSpace(text)

	// Check if the entire text is just template syntax (like "{{ .Name }}")
	if strings.HasPrefix(trimmed, "{{") && strings.HasSuffix(trimmed, "}}") {
		return true
	}

	// Check for incomplete template syntax (like "{{- if")
	if strings.HasPrefix(trimmed, "{{") && !strings.HasSuffix(trimmed, "}}") {
		return true
	}

	// Check if the entire text is just template control structure
	if templateControlRegex.MatchString(trimmed) && strings.Count(trimmed, "{{") == 1 {
		return true
	}

	// Check for HTML comments
	if strings.HasPrefix(text, "<!--") || strings.HasSuffix(text, "-->") {
		return true
	}
	if strings.HasPrefix(text, "/*") || strings.HasSuffix(text, "*/") {
		return true
	}

	// Check for lines that are mostly punctuation or symbols
	if strings.TrimSpace(strings.Trim(text, "-><!{}()[]\"'`/\\*+=#@$%^&|~;:,._")) == "" {
		return true
	}
	return false
}

// isBlockTag checks if an HTML tag is a block-level element
func isBlockTag(tag string) bool {
	// Extract tag name from full tag string
	tagName := strings.ToLower(tag)
	if strings.HasPrefix(tagName, "</") {
		tagName = tagName[2:]
	} else if strings.HasPrefix(tagName, "<") {
		tagName = tagName[1:]
	}

	// Remove attributes and closing >
	if idx := strings.IndexAny(tagName, " \t\n>"); idx != -1 {
		tagName = tagName[:idx]
	}

	blockTags := []string{
		"div", "p", "h1", "h2", "h3", "h4", "h5", "h6",
		"ul", "ol", "li", "section", "article", "header",
		"footer", "nav", "main", "aside", "blockquote",
		"pre", "hr", "br", "form", "fieldset", "table",
		"tr", "td", "th", "thead", "tbody", "tfoot",
	}

	return slices.Contains(blockTags, tagName)
}
