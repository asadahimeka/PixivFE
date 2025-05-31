// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStripJetComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple comment",
			input:    "before {* comment *} after",
			expected: "before  after",
		},
		{
			name:     "multiline comment",
			input:    "before {* multi\nline\ncomment *} after",
			expected: "before  after",
		},
		{
			name:     "multiple comments",
			input:    "start {* first *} middle {* second *} end",
			expected: "start  middle  end",
		},
		{
			name:     "no comments",
			input:    "no comments here",
			expected: "no comments here",
		},
		{
			name:     "comment with special chars",
			input:    "before {* comment with {{ }} and ** *} after",
			expected: "before  after",
		},
		{
			name:     "nested braces in comment",
			input:    "before {* { nested } braces *} after",
			expected: "before  after",
		},
		{
			name:     "empty comment",
			input:    "before {**} after",
			expected: "before  after",
		},
		{
			name:     "comment at start",
			input:    "{* comment *} content",
			expected: " content",
		},
		{
			name:     "comment at end",
			input:    "content {* comment *}",
			expected: "content ",
		},
		{
			name:     "only comment",
			input:    "{* only comment *}",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripJetComments(tt.input)
			if result != tt.expected {
				t.Errorf("stripJetComments(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestReCommandFullmatch(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "simple command",
			input:    "{{ .Title }}",
			expected: true,
		},
		{
			name:     "command with whitespace",
			input:    "  {{ .Title }}  ",
			expected: true,
		},
		{
			name:     "command with newlines",
			input:    "\n{{ .Title }}\n",
			expected: true,
		},
		{
			name:     "multiple commands",
			input:    "{{ .Title }} {{ .Content }}",
			expected: true,
		},
		{
			name:     "extends command",
			input:    `{{- extends "layout/default" }}`,
			expected: true,
		},
		{
			name:     "block command",
			input:    `{{- block body() }}`,
			expected: true,
		},
		{
			name: "complex template",
			input: `{{- extends "layout/default" }}
{{- block body() }}`,
			expected: true,
		},
		{
			name:     "mixed content",
			input:    "Hello {{ .Name }}",
			expected: false,
		},
		{
			name:     "plain text",
			input:    "Hello World",
			expected: false,
		},
		{
			name:     "command with text after",
			input:    "{{ .Title }} some text",
			expected: false,
		},
		{
			name:     "command with text before",
			input:    "some text {{ .Title }}",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: true,
		},
		{
			name:     "only whitespace",
			input:    "   \n\t   ",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := re_command_fullmatch.MatchString(tt.input)
			if result != tt.expected {
				t.Errorf("re_command_fullmatch.MatchString(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestProcessFileWithMockFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "crawler_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name         string
		fileContent  string
		expectedMsgs []string
	}{
		{
			name: "simple HTML with text",
			fileContent: `<html>
<body>
<div>Hello World</div>
<p>Another paragraph</p>
</body>
</html>`,
			expectedMsgs: []string{"Hello World", "Another paragraph"},
		},
		{
			name: "HTML with Jet comments",
			fileContent: `<html>
<body>
{* This is a comment *}
<div>Visible text</div>
{* Another comment *}
<p>More visible text</p>
</body>
</html>`,
			expectedMsgs: []string{"Visible text", "More visible text"},
		},
		{
			name: "HTML with template commands only",
			fileContent: `{{- extends "layout/default" }}
{{- block body() }}
{{ .Title }}
{{- end }}`,
			expectedMsgs: []string{},
		},
		{
			name: "HTML with mixed content",
			fileContent: `<html>
<body>
<div>Regular text</div>
{{ .TemplateVar }}
<p>More text</p>
{* comment *}
<span>Final text</span>
</body>
</html>`,
			expectedMsgs: []string{"Regular text", "More text", "Final text"},
		},
		{
			name: "HTML with ignored strings",
			fileContent: `<html>
<body>
<div>PixivFE</div>
<p>»</p>
<span>Valid content</span>
</body>
</html>`,
			expectedMsgs: []string{"Valid content"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file
			testFile := filepath.Join(tempDir, "test.html")
			err := os.WriteFile(testFile, []byte(tt.fileContent), 0o644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Process the file
			var result []Match
			processFile(testFile, &result)

			// Extract messages from results
			var actualMsgs []string
			for _, match := range result {
				actualMsgs = append(actualMsgs, match.Msg)
			}

			// Compare results
			if len(actualMsgs) != len(tt.expectedMsgs) {
				t.Errorf("Expected %d messages, got %d", len(tt.expectedMsgs), len(actualMsgs))
				t.Errorf("Expected: %v", tt.expectedMsgs)
				t.Errorf("Got: %v", actualMsgs)
				return
			}

			for i, expected := range tt.expectedMsgs {
				if i >= len(actualMsgs) {
					t.Errorf("Missing message %d: %q", i, expected)
					continue
				}
				if actualMsgs[i] != expected {
					t.Errorf("Message %d: expected %q, got %q", i, expected, actualMsgs[i])
				}
			}

			// Verify all matches have the correct file path
			for _, match := range result {
				if !strings.HasSuffix(match.File, "test.html") {
					t.Errorf("Expected file path to end with 'test.html', got %q", match.File)
				}
				if match.Offset < 0 {
					t.Errorf("Expected non-negative offset, got %d", match.Offset)
				}
			}
		})
	}
}

// Additional tests for better coverage
func TestMatchJSONSerialization(t *testing.T) {
	match := Match{
		File:   "test.html",
		Msg:    "Hello World",
		Offset: 42,
	}

	expected := `{"file":"test.html","msg":"Hello World","offset":42}`

	jsonData, err := json.Marshal(match)
	if err != nil {
		t.Fatalf("Failed to marshal Match to JSON: %v", err)
	}

	if string(jsonData) != expected {
		t.Errorf("JSON serialization failed. Expected %s, got %s", expected, string(jsonData))
	}

	// Test deserialization
	var unmarshaled Match
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON to Match: %v", err)
	}

	if unmarshaled.File != match.File {
		t.Errorf("File mismatch after unmarshaling: expected %s, got %s", match.File, unmarshaled.File)
	}
	if unmarshaled.Msg != match.Msg {
		t.Errorf("Msg mismatch after unmarshaling: expected %s, got %s", match.Msg, unmarshaled.Msg)
	}
	if unmarshaled.Offset != match.Offset {
		t.Errorf("Offset mismatch after unmarshaling: expected %d, got %d", match.Offset, unmarshaled.Offset)
	}
}

func TestProcessFileErrorHandling(t *testing.T) {
	// Test processing a non-existent file
	var result []Match

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when processing non-existent file, but didn't panic")
		}
	}()

	processFile("/non/existent/file.html", &result)
}

func TestComplexHTMLStructures(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "crawler_complex_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name         string
		fileContent  string
		expectedMsgs []string
	}{
		{
			name: "deeply nested HTML",
			fileContent: `<html>
<body>
<div class="container">
	<header>
		<h1>Main Title</h1>
		<nav>
			<ul>
				<li><a href="#">Home</a></li>
				<li><a href="#">About</a></li>
			</ul>
		</nav>
	</header>
	<main>
		<article>
			<h2>Article Title</h2>
			<p>First paragraph with <em>emphasis</em> and <strong>strong</strong> text.</p>
			<p>Second paragraph.</p>
		</article>
	</main>
</div>
</body>
</html>`,
			expectedMsgs: []string{
				"Main Title",
				"Home",
				"About",
				"Article Title",
				"First paragraph with emphasis and strong text.",
				"Second paragraph.",
			},
		},
		{
			name: "HTML with tables",
			fileContent: `<table>
<thead>
	<tr>
		<th>Header 1</th>
		<th>Header 2</th>
	</tr>
</thead>
<tbody>
	<tr>
		<td>Cell 1</td>
		<td>Cell 2</td>
	</tr>
</tbody>
</table>`,
			expectedMsgs: []string{"Header 1", "Header 2", "Cell 1", "Cell 2"},
		},
		{
			name: "HTML with forms",
			fileContent: `<form>
<label for="name">Name:</label>
<input type="text" id="name" name="name">
<label for="email">Email:</label>
<input type="email" id="email" name="email">
<button type="submit">Submit</button>
</form>`,
			expectedMsgs: []string{"Name:", "Email:", "Submit"},
		},
		{
			name: "mixed Jet template syntax",
			fileContent: `<div>
{{ range .Items }}
	<p>Item: {{ .Name }}</p>
{{ end }}
<span>Total: {{ .Count }}</span>
</div>
<footer>Copyright 2023</footer>`,
			expectedMsgs: []string{"Item: {{ .Name }}", "Total: {{ .Count }}", "Copyright 2023"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file
			testFile := filepath.Join(tempDir, "test_complex.html")
			err := os.WriteFile(testFile, []byte(tt.fileContent), 0o644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Process the file
			var result []Match
			processFile(testFile, &result)

			// Extract messages from results
			var actualMsgs []string
			for _, match := range result {
				actualMsgs = append(actualMsgs, match.Msg)
			}

			// Compare results
			if len(actualMsgs) != len(tt.expectedMsgs) {
				t.Errorf("Expected %d messages, got %d", len(tt.expectedMsgs), len(actualMsgs))
				t.Errorf("Expected: %v", tt.expectedMsgs)
				t.Errorf("Got: %v", actualMsgs)
				return
			}

			for i, expected := range tt.expectedMsgs {
				if i >= len(actualMsgs) {
					t.Errorf("Missing message %d: %q", i, expected)
					continue
				}
				if actualMsgs[i] != expected {
					t.Errorf("Message %d: expected %q, got %q", i, expected, actualMsgs[i])
				}
			}
		})
	}
}
