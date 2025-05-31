// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMatch(t *testing.T) {
	match := Match{
		File:   "test.go",
		Msg:    "Hello World",
		Line:   42,
		Offset: 100,
	}

	tests := []struct {
		field    string
		expected any
		actual   any
	}{
		{"File", "test.go", match.File},
		{"Msg", "Hello World", match.Msg},
		{"Line", 42, match.Line},
		{"Offset", 100, match.Offset},
	}

	for _, test := range tests {
		if test.expected != test.actual {
			t.Errorf("Expected %s to be %v, got %v", test.field, test.expected, test.actual)
		}
	}
}

func TestFindI18nCalls(t *testing.T) {
	testCode := `package test

import "i18n"

func example() {
	i18n.T("test message")
	i18n.Plural("count message", 5)
	notI18n.T("should not match")
	i18n.T(variable) // should not match - not a string literal
}
`

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")
	createFileInDir(t, tempDir, "test.go", testCode)

	matches, err := parseFileForI18n(testFile)
	if err != nil {
		t.Fatalf("Failed to parse test file: %v", err)
	}

	expectedMatches := []struct {
		msg  string
		line int
	}{
		{"test message", 6},
		{"count message", 7},
	}

	if len(matches) != len(expectedMatches) {
		t.Fatalf("Expected %d matches, got %d", len(expectedMatches), len(matches))
	}

	for i, expected := range expectedMatches {
		if matches[i].Msg != expected.msg {
			t.Errorf("Match %d: expected message '%s', got '%s'", i, expected.msg, matches[i].Msg)
		}
		if matches[i].Line != expected.line {
			t.Errorf("Match %d: expected line %d, got %d", i, expected.line, matches[i].Line)
		}
	}
}

func TestJSONRoundTrip(t *testing.T) {
	matches := []Match{
		{File: "file1.go", Msg: "message1", Line: 10, Offset: 50},
		{File: "file2.go", Msg: "message2", Line: 20, Offset: 100},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(matches); err != nil {
		t.Fatalf("Failed to encode JSON: %v", err)
	}

	var decoded []Match
	if err := json.NewDecoder(&buf).Decode(&decoded); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	if len(decoded) != len(matches) {
		t.Fatalf("Expected %d matches after JSON round-trip, got %d", len(matches), len(decoded))
	}

	for i, match := range matches {
		if decoded[i] != match {
			t.Errorf("Match %d doesn't match after JSON round-trip: expected %+v, got %+v", i, match, decoded[i])
		}
	}
}

func TestNonGoFilesIgnored(t *testing.T) {
	tempDir := t.TempDir()

	files := map[string]string{
		"test.go": `package test
func main() { 
	i18n.T("go message") 
}`,
		"test.txt":  `i18n.T("text message")`,
		"test.js":   `i18n.T("js message")`,
		"README.md": `# i18n.T("markdown message")`,
	}

	for filename, content := range files {
		createFileInDir(t, tempDir, filename, content)
	}

	matches, err := findI18nCalls(tempDir)
	if err != nil {
		t.Fatalf("Error walking directory: %v", err)
	}

	if len(matches) != 1 {
		t.Errorf("Expected 1 match (only from .go files), got %d", len(matches))
	}

	if len(matches) > 0 && matches[0].Msg != "go message" {
		t.Errorf("Expected match message to be 'go message', got '%s'", matches[0].Msg)
	}
}

func TestMainIntegration(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	testFiles := map[string]string{
		"file1.go": `package main
import "i18n"
func example1() {
	i18n.T("Hello World")
	i18n.Plural("items", count)
}`,
		"subdir/file2.go": `package subdir
func example2() {
	i18n.Format("Welcome %s", name)
}`,
	}

	for filename, content := range testFiles {
		createFileInDir(t, ".", filename, content)
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	main()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	var matches []Match
	if err := json.Unmarshal([]byte(output), &matches); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	expectedMessages := []string{"Hello World", "items", "Welcome %s"}
	if len(matches) != len(expectedMessages) {
		t.Errorf("Expected %d matches, got %d", len(expectedMessages), len(matches))
	}

	foundMessages := make(map[string]bool)
	for _, match := range matches {
		foundMessages[match.Msg] = true
	}

	for _, expectedMsg := range expectedMessages {
		if !foundMessages[expectedMsg] {
			t.Errorf("Expected to find message '%s' but didn't", expectedMsg)
		}
	}
}

func BenchmarkParseFileForI18n(b *testing.B) {
	testCode := `package test
import "i18n"
func example() {
	i18n.T("test message")
	i18n.Plural("count message", 5)
	i18n.Format("format message %s", arg)
}
`

	tempDir := b.TempDir()
	createFileInDir(b, tempDir, "test.go", testCode)
	testFile := filepath.Join(tempDir, "test.go")

	for b.Loop() {
		_, err := parseFileForI18n(testFile)
		if err != nil {
			b.Fatalf("Failed to parse test file: %v", err)
		}
	}
}

func createFileInDir(t testing.TB, dir, filename, content string) {
	filePath := filepath.Join(dir, filename)
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("Failed to create directory for %s: %v", filePath, err)
	}
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write file %s: %v", filePath, err)
	}
}
