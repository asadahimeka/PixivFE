// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"slices"
	"testing"
)

func TestGetRandomUserAgent(t *testing.T) {
	// Test that the function returns a non-empty string
	ua := GetRandomUserAgent()
	if ua == "" {
		t.Error("GetRandomUserAgent returned an empty string")
	}

	// Test that the returned user agent is in one of the lists
	inLinux := slices.Contains(chromeLinuxAgents, ua)
	inMac := slices.Contains(chromeMacAgents, ua)
	inWindows := slices.Contains(chromeWindowsAgents, ua)

	if !(inLinux || inMac || inWindows) {
		t.Error("GetRandomUserAgent returned a user agent not in any of the available lists")
	}
}
