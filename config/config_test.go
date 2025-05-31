// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"io"
	"log"
	"os"
	"testing"

	"codeberg.org/pixivfe/pixivfe/server/utils"
)

/*
No test case for GetToken yet due to token_manager dependency
Mocking token_manager isn't fun

TestLoadConfig focuses on verifying main functionality (e.g. fallback when invalid input),
and *shouldn't* need exhaustive scenarios
*/

// setupTestLogger sets up a logger that discards output and returns a function to restore the original logger
func setupTestLogger() func() {
	originalLogger := log.Default()
	log.SetOutput(io.Discard)
	return func() {
		log.SetOutput(originalLogger.Writer())
	}
}

// TestParseRevision is a test function that verifies the behavior of the parseRevision function.
func TestParseRevision(t *testing.T) {
	tests := []struct {
		name          string // Description of the test case
		revision      string // Input revision string
		expectedDate  string // Expected date output
		expectedHash  string // Expected hash output
		expectedDirty bool   // Expected isDirty output
	}{
		{"Valid revision", "2024.09.24-18d6874", "2024.09.24", "18d6874", false},
		{"Dirty revision", "2024.09.24-18d6874+dirty", "2024.09.24", "18d6874", true},
		{"Empty revision", "", "unknown", "unknown", false},
		{"Only hash", "18d6874", "unknown", "18d6874", false},
		{"Only hash dirty", "18d6874+dirty", "unknown", "18d6874", true},
		{"Invalid format", "2024.09.24-18d6874-extra", "unknown", "2024.09.24-18d6874-extra", false},
		{"Invalid format dirty", "2024.09.24-18d6874-extra+dirty", "unknown", "2024.09.24-18d6874-extra", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDate, gotHash, gotDirty := parseRevision(tt.revision)

			if gotDate != tt.expectedDate {
				t.Errorf("parseRevision() gotDate = %v, want %v", gotDate, tt.expectedDate)
			}
			if gotHash != tt.expectedHash {
				t.Errorf("parseRevision() gotHash = %v, want %v", gotHash, tt.expectedHash)
			}
			if gotDirty != tt.expectedDirty {
				t.Errorf("parseRevision() gotDirty = %v, want %v", gotDirty, tt.expectedDirty)
			}
		})
	}
}

// TestValidateURL is a test function that verifies the behavior of the validateURL function.
func TestValidateURL(t *testing.T) {
	tests := []struct {
		name     string // Description of the test case
		urlStr   string // Input URL string to be validated
		urlType  string // Type of URL (used for error messaging)
		wantErr  bool   // Whether an error is expected
		expected string // Expected output URL string
	}{
		{"Valid URL", "https://example.com", "Test", false, "https://example.com"},
		{"Valid URL with path", "https://example.com/path", "Test", false, "https://example.com/path"},
		{"Missing scheme", "example.com", "Test", true, ""},
		{"Missing host", "https://", "Test", true, ""},
		{"Trailing slash", "https://example.com/", "Test", true, ""},
		{"Empty URL", "", "Test", true, ""},
		{"URL with query params", "https://example.com/path?q=test", "Test", false, "https://example.com/path?q=test"},
		{"URL with fragment", "https://example.com/path#fragment", "Test", false, "https://example.com/path#fragment"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := utils.ValidateURL(tt.urlStr, tt.urlType)

			if (err != nil) != tt.wantErr {
				t.Errorf("utils.ValidateURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.String() != tt.expected {
					t.Errorf("utils.ValidateURL() got = %v, want %v", got, tt.expected)
				}
			}
		})
	}
}

// TestLoadConfig is a test function that verifies the behavior of the LoadConfig function.
func TestLoadConfig(t *testing.T) {
	// Helper function to set environment variables
	setEnv := func(env map[string]string) {
		for k, v := range env {
			os.Setenv(k, v)
		}
	}

	// Helper function to unset environment variables
	unsetEnv := func(env map[string]string) {
		for k := range env {
			os.Unsetenv(k)
		}
	}

	tests := []struct {
		name    string            // Description of the test case
		env     map[string]string // Name of the environment variable and its value
		wantErr bool              // Whether an error is expected
	}{
		{
			name: "Valid configuration",
			env: map[string]string{
				"PIXIVFE_HOST":  "localhost",
				"PIXIVFE_PORT":  "8282",
				"PIXIVFE_TOKEN": "token1,token2",
			},
			wantErr: false,
		},
		{
			name: "Missing required PIXIVFE_TOKEN",
			env: map[string]string{
				"PIXIVFE_HOST":       "localhost",
				"PIXIVFE_PORT":       "8282",
				"PIXIVFE_IMAGEPROXY": "https://imageproxy.test",
			},
			wantErr: true,
		},
		{
			name: "Invalid PIXIVFE_IMAGEPROXY",
			env: map[string]string{
				"PIXIVFE_HOST":       "localhost",
				"PIXIVFE_PORT":       "8282",
				"PIXIVFE_TOKEN":      "token1,token2",
				"PIXIVFE_IMAGEPROXY": "invalidimageproxy-test",
			},
			wantErr: true,
		},
		{
			name: "Invalid PIXIVFE_TOKEN_LOAD_BALANCING",
			env: map[string]string{
				"PIXIVFE_HOST":                 "localhost",
				"PIXIVFE_PORT":                 "8282",
				"PIXIVFE_TOKEN":                "token1,token2",
				"PIXIVFE_TOKEN_LOAD_BALANCING": "invalid-load-balancing-method",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up test logger
			restoreLogger := setupTestLogger()
			defer restoreLogger()

			// Set up environment
			setEnv(tt.env)
			defer unsetEnv(tt.env)

			// Create a new ServerConfig instance
			config := &ServerConfig{}

			// Call LoadConfig
			err := config.LoadConfig()

			// Check for errors
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Test whether config fields were set correctly
				if config.Basic.Host != tt.env["PIXIVFE_HOST"] {
					t.Errorf("LoadConfig() Host = %v, want %v", config.Basic.Host, tt.env["PIXIVFE_HOST"])
				}

				if config.Basic.Port != tt.env["PIXIVFE_PORT"] {
					t.Errorf("LoadConfig() Port = %v, want %v", config.Basic.Port, tt.env["PIXIVFE_PORT"])
				}

				if len(config.Basic.Token) != 2 && tt.env["PIXIVFE_TOKEN"] == "token1,token2" {
					t.Errorf("LoadConfig() Token count = %v, want 2", len(config.Basic.Token))
				}

				if config.TokenManager.TokenManager == nil {
					t.Error("LoadConfig() TokenManager is nil")
				}

				if config.ContentProxies.Image.String() == "" {
					t.Error("LoadConfig() ProxyServer is empty")
				}

				if config.TokenManager.LoadBalancing == "" {
					t.Error("LoadConfig() TokenLoadBalancing is empty")
				}
			}
		})
	}
}
