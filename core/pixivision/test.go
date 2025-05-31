// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package pixivision

import (
	"testing"
)

//nolint:lll
func TestParseBackgroundImage(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Valid background-image",
			input:    `<div class="tdc__thumbnail" style="background-image: url(https://i.pximg.net/c/384x280_80_a2_g2/img-master/img/2021/03/09/00/39/09/88315740_p0_master1200.jpg);"></div>`,
			expected: "https://i.pximg.net/c/384x280_80_a2_g2/img-master/img/2021/03/09/00/39/09/88315740_p0_master1200.jpg",
		},
		{
			name:     "Missing background-image property",
			input:    `<div style="background: url(http://example.com/image.jpg);"></div>`,
			expected: "",
		},
		{
			name:     "No URL value (using 'none')",
			input:    `<div style="background-image: none;"></div>`,
			expected: "",
		},
		{
			name:     "Missing closing parenthesis",
			input:    `<div style="background-image: url(http://example.com/image.jpg"></div>`,
			expected: "",
		},
		{
			name:     "Empty URL",
			input:    `<div style="background-image: url()"></div>`,
			expected: "",
		},
		{
			name:     "Empty input",
			input:    ``,
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseBackgroundImage(tc.input)
			if result != tc.expected {
				t.Errorf("For input %q: expected %q, got %q", tc.input, tc.expected, result)
			}
		})
	}
}
