// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package template

import (
	"reflect"
	"testing"
)

func TestParseDescriptionURLs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "pixiv novel redirect",
			input:    "/jump.php?https%3A%2F%2Fwww.pixiv.net%2Fnovel%2Fshow.php%3Fid%3D24560221",
			expected: "/novel/24560221",
		},
		{
			name:     "pixiv novel redirect with language prefix",
			input:    "/jump.php?https%3A%2F%2Fwww.pixiv.net%2Fen%2Fnovel%2Fshow.php%3Fid%3D24560221",
			expected: "/novel/24560221",
		},
		{
			name:     "pixiv novel redirect with additional parameters",
			input:    "/jump.php?https%3A%2F%2Fwww.pixiv.net%2Fnovel%2Fshow.php%3Fsomeparam%3Dvalue%26id%3D24560221",
			expected: "/novel/24560221",
		},
		{
			name:     "pixiv user redirect",
			input:    "/jump.php?https%3A%2F%2Fwww.pixiv.net%2Fusers%2F12345",
			expected: "/users/12345",
		},
		{
			name:     "pixiv artwork redirect",
			input:    "/jump.php?https%3A%2F%2Fwww.pixiv.net%2Fartworks%2F67890",
			expected: "/artworks/67890",
		},
		{
			name:     "pixiv user redirect with language prefix",
			input:    "/jump.php?https%3A%2F%2Fwww.pixiv.net%2Fen%2Fusers%2F12345",
			expected: "/users/12345",
		},
		{
			name:     "pixiv artwork redirect with language prefix",
			input:    "/jump.php?https%3A%2F%2Fwww.pixiv.net%2Fen%2Fartworks%2F67890",
			expected: "/artworks/67890",
		},
		{
			name:     "pixiv non-user non-artwork redirect (e.g. /home)",
			input:    "/jump.php?https%3A%2F%2Fwww.pixiv.net%2Fhome",
			expected: "https://www.pixiv.net/home",
		},
		{
			name:     "Non-pixiv redirect",
			input:    "/jump.php?https%3A%2F%2Fexample.com",
			expected: "https://example.com",
		},
		{
			name:     "URL with special characters (percent-encoded ampersand in query)",
			input:    "/jump.php?https%3A%2F%2Fexample.com%2Fpath%3Fquery%3Dvalue%26another%3Dvalue",
			expected: "https://example.com/path?query=value&another=value",
		},
		{
			name:     "No redirect pattern, just plain URL",
			input:    "https://example.com",
			expected: "https://example.com",
		},
		{
			name:     "Invalid percent encoding in jump.php",
			input:    "/jump.php?https%3A%2F%2Fexample.com%ZZ",
			expected: "/jump.php?https%3A%2F%2Fexample.com%ZZ",
		},
		{
			name:     "Non-http protocol (ftp) in jump.php",
			input:    "/jump.php?ftp%3A%2F%2Fexample.com",
			expected: "ftp://example.com",
		},
		{
			name:     "Mailto protocol in jump.php",
			input:    "/jump.php?mailto%3Atest%40example.com",
			expected: "mailto:test@example.com",
		},
		{
			name:     "Javascript URI in jump.php",
			input:    "/jump.php?javascript%3Aalert%281%29",
			expected: "",
		},
		{
			name:     "Non-URL text (no scheme) in jump.php",
			input:    "/jump.php?justsometext",
			expected: "justsometext",
		},
		{
			name:     "Empty string input",
			input:    "",
			expected: "",
		},
		{
			name:     "String with multiple jump.php links",
			input:    "First: /jump.php?https%3A%2F%2Fwww.pixiv.net%2Fusers%2F12345 Second: /jump.php?https%3A%2F%2Fexample.com",
			expected: "First: /users/12345 Second: https://example.com",
		},
		{
			name:     "Multiple eligible pixiv links in jump.php",
			input:    "User: /jump.php?https%3A%2F%2Fwww.pixiv.net%2Fusers%2F123 Art: /jump.php?https%3A%2F%2Fwww.pixiv.net%2Fartworks%2F456",
			expected: "User: /users/123 Art: /artworks/456",
		},
		{
			name:     "Mixed pixiv links in jump.php, some eligible for relative, some not",
			input:    "User: /jump.php?https%3A%2F%2Fwww.pixiv.net%2Fen%2Fusers%2F789 Home: /jump.php?https%3A%2F%2Fwww.pixiv.net%2Fdashboard",
			expected: "User: /users/789 Home: https://www.pixiv.net/dashboard",
		},
		{
			name:     "pixiv user redirect with http (not https)",
			input:    "/jump.php?http%3A%2F%2Fwww.pixiv.net%2Fusers%2F12345",
			expected: "/users/12345",
		},
		{
			name:     "jump.php with empty parameter",
			input:    "/jump.php?",
			expected: "/jump.php?",
		},
		{
			name:     "jump.php with parameter ending before ampersand (regex behavior)",
			input:    "/jump.php?https%3A%2F%2Fexample.com¶m=2",
			expected: "https://example.com¶m=2",
		},
		// Tests for standalone absolute pixiv URLs
		{
			name:     "Standalone pixiv user URL",
			input:    "Check this user: https://www.pixiv.net/users/12345",
			expected: "Check this user: /users/12345",
		},
		{
			name:     "Standalone pixiv artwork URL with lang",
			input:    "Artwork: https://www.pixiv.net/en/artworks/67890",
			expected: "Artwork: /artworks/67890",
		},
		{
			name:     "Standalone pixiv novel URL",
			input:    "Novel: https://www.pixiv.net/novel/show.php?id=24560221",
			expected: "Novel: /novel/24560221",
		},
		{
			name:     "Standalone pixiv home URL (should not change)",
			input:    "Home: https://www.pixiv.net/home",
			expected: "Home: https://www.pixiv.net/home",
		},
		{
			name:     "Standalone pixiv URL with www and http",
			input:    "Link: http://www.pixiv.net/users/987",
			expected: "Link: /users/987",
		},
		{
			name:     "Standalone pixiv URL without www",
			input:    "Link: https://pixiv.net/artworks/654",
			expected: "Link: /artworks/654",
		},
		{
			name:     "Standalone javascript URI (should not change, not targeted by absolutePixivLinkRegex)",
			input:    "Script: javascript:alert('danger')",
			expected: "Script: javascript:alert('danger')",
		},
		{
			name:     "Mixed jump.php and standalone absolute pixiv URLs",
			input:    "Jump: /jump.php?https%3A%2F%2Fwww.pixiv.net%2Fusers%2F123 Standalone: https://www.pixiv.net/artworks/456",
			expected: "Jump: /users/123 Standalone: /artworks/456",
		},
		{
			name:     "Mixed jump.php (non-pixiv) and standalone absolute pixiv URL (non-special)",
			input:    "Jump: /jump.php?https%3A%2F%2Fexample.com Standalone: https://www.pixiv.net/about",
			expected: "Jump: https://example.com Standalone: https://www.pixiv.net/about",
		},
		{
			name:     "Already relative pixiv URL (should not change, not matched by absolutePixivLinkRegex)",
			input:    "Link: /users/12345",
			expected: "Link: /users/12345",
		},
		{
			name:     "URL with trailing punctuation",
			input:    "Link: https://www.pixiv.net/users/12345.",
			expected: "Link: /users/12345.", // url.Parse includes trailing dot in path, current logic keeps it.
		},
		{
			name:  "URL within quotes",
			input: `Link: "https://www.pixiv.net/users/12345"`,
			// absolutePixivLinkRegex's [^\s<>"']* means it stops at the quote.
			expected: `Link: "/users/12345"`,
		},
		{
			name:     "URL within parentheses",
			input:    "Link: (https://www.pixiv.net/users/12345)",
			expected: "Link: (/users/12345)", // Similar to quotes, stops at ) due to [^\s<>"']*
		},
		{
			name:     "Path ending with slash /users/123/",
			input:    "/jump.php?https%3A%2F%2Fwww.pixiv.net%2Fusers%2F123%2F",
			expected: "/users/123", // TrimPrefix and Split handle this well for pathParts
		},
		{
			name:     "Standalone path ending with slash /users/123/",
			input:    "https://www.pixiv.net/users/123/",
			expected: "/users/123",
		},
		{
			name:  "Path like /users/ (no ID)",
			input: "/jump.php?https%3A%2F%2Fwww.pixiv.net%2Fusers%2F",
			// pathParts will be ["users", ""]. id == "" condition will make it not convert.
			expected: "https://www.pixiv.net/users/",
		},
		{
			name:     "Standalone path like /users/ (no ID)",
			input:    "https://www.pixiv.net/users/",
			expected: "https://www.pixiv.net/users/",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ParseDescriptionURLs(tc.input)
			if result != tc.expected {
				t.Errorf("parseDescriptionURLs(%q):\n got: %q\nwant: %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestCreatePaginator(t *testing.T) {
	tests := []struct {
		name           string
		base           string
		ending         string
		currentPage    int
		maxPage        int
		pageMargin     int
		dropdownOffset int
		expectError    bool
		expectedData   PaginationData
	}{
		{
			name:           "Normal case",
			base:           "/ranking?content=all&date=20240101&mode=daily&page=",
			ending:         "#checkpoint",
			currentPage:    3,
			maxPage:        10,
			pageMargin:     1,
			dropdownOffset: 2,
			expectError:    false,
			expectedData: PaginationData{
				CurrentPage: 3,
				MaxPage:     10,
				Pages: []PageInfo{
					{Number: 2, URL: "/ranking?content=all&date=20240101&mode=daily&page=2#checkpoint"},
					{Number: 3, URL: "/ranking?content=all&date=20240101&mode=daily&page=3#checkpoint"},
					{Number: 4, URL: "/ranking?content=all&date=20240101&mode=daily&page=4#checkpoint"},
				},
				HasPrevious: true,
				HasNext:     true,
				PreviousURL: "/ranking?content=all&date=20240101&mode=daily&page=2#checkpoint",
				NextURL:     "/ranking?content=all&date=20240101&mode=daily&page=4#checkpoint",
				FirstURL:    "/ranking?content=all&date=20240101&mode=daily&page=1#checkpoint",
				LastURL:     "/ranking?content=all&date=20240101&mode=daily&page=10#checkpoint",
				HasMaxPage:  true,
				LastPage:    4,
				DropdownPages: []PageInfo{
					{Number: 1, URL: "/ranking?content=all&date=20240101&mode=daily&page=1#checkpoint"},
					{Number: 2, URL: "/ranking?content=all&date=20240101&mode=daily&page=2#checkpoint"},
					{Number: 3, URL: "/ranking?content=all&date=20240101&mode=daily&page=3#checkpoint"},
					{Number: 4, URL: "/ranking?content=all&date=20240101&mode=daily&page=4#checkpoint"},
					{Number: 5, URL: "/ranking?content=all&date=20240101&mode=daily&page=5#checkpoint"},
				},
			},
		},
		{
			name:           "Unknown max page",
			base:           "/ranking?content=all&date=20240101&mode=daily&page=",
			ending:         "#checkpoint",
			currentPage:    3,
			maxPage:        -1,
			pageMargin:     1,
			dropdownOffset: 2,
			expectError:    false,
			expectedData: PaginationData{
				CurrentPage: 3,
				MaxPage:     -1,
				Pages: []PageInfo{
					{Number: 2, URL: "/ranking?content=all&date=20240101&mode=daily&page=2#checkpoint"},
					{Number: 3, URL: "/ranking?content=all&date=20240101&mode=daily&page=3#checkpoint"},
					{Number: 4, URL: "/ranking?content=all&date=20240101&mode=daily&page=4#checkpoint"},
				},
				HasPrevious: true,
				HasNext:     true,
				PreviousURL: "/ranking?content=all&date=20240101&mode=daily&page=2#checkpoint",
				NextURL:     "/ranking?content=all&date=20240101&mode=daily&page=4#checkpoint",
				FirstURL:    "/ranking?content=all&date=20240101&mode=daily&page=1#checkpoint",
				LastURL:     "/ranking?content=all&date=20240101&mode=daily&page=-1#checkpoint",
				HasMaxPage:  false,
				LastPage:    4,
				DropdownPages: []PageInfo{
					{Number: 1, URL: "/ranking?content=all&date=20240101&mode=daily&page=1#checkpoint"},
					{Number: 2, URL: "/ranking?content=all&date=20240101&mode=daily&page=2#checkpoint"},
					{Number: 3, URL: "/ranking?content=all&date=20240101&mode=daily&page=3#checkpoint"},
					{Number: 4, URL: "/ranking?content=all&date=20240101&mode=daily&page=4#checkpoint"},
					{Number: 5, URL: "/ranking?content=all&date=20240101&mode=daily&page=5#checkpoint"},
				},
			},
		},
		{
			name:           "First page",
			base:           "/ranking?content=all&date=20240101&mode=daily&page=",
			ending:         "#checkpoint",
			currentPage:    1,
			maxPage:        10,
			pageMargin:     1,
			dropdownOffset: 2,
			expectError:    false,
			expectedData: PaginationData{
				CurrentPage: 1,
				MaxPage:     10,
				Pages: []PageInfo{
					{Number: 1, URL: "/ranking?content=all&date=20240101&mode=daily&page=1#checkpoint"},
					{Number: 2, URL: "/ranking?content=all&date=20240101&mode=daily&page=2#checkpoint"},
				},
				HasPrevious: false,
				HasNext:     true,
				PreviousURL: "",
				NextURL:     "/ranking?content=all&date=20240101&mode=daily&page=2#checkpoint",
				FirstURL:    "/ranking?content=all&date=20240101&mode=daily&page=1#checkpoint",
				LastURL:     "/ranking?content=all&date=20240101&mode=daily&page=10#checkpoint",
				HasMaxPage:  true,
				LastPage:    2,
				DropdownPages: []PageInfo{
					{Number: 1, URL: "/ranking?content=all&date=20240101&mode=daily&page=1#checkpoint"},
					{Number: 2, URL: "/ranking?content=all&date=20240101&mode=daily&page=2#checkpoint"},
					{Number: 3, URL: "/ranking?content=all&date=20240101&mode=daily&page=3#checkpoint"},
				},
			},
		},
		{
			name:           "Invalid current page",
			base:           "/ranking?content=all&date=20240101&mode=daily&page=",
			ending:         "#checkpoint",
			currentPage:    0,
			maxPage:        10,
			pageMargin:     1,
			dropdownOffset: 2,
			expectError:    true,
		},
		{
			name:           "Invalid page margin",
			base:           "/ranking?content=all&date=20240101&mode=daily&page=",
			ending:         "#checkpoint",
			currentPage:    1,
			maxPage:        10,
			pageMargin:     -1,
			dropdownOffset: 2,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotData, err := CreatePaginator(tt.base, tt.ending, tt.currentPage, tt.maxPage, tt.pageMargin, tt.dropdownOffset)

			if tt.expectError {
				if err == nil {
					t.Errorf("CreatePaginator() error = nil, expected an error")
				}
			} else {
				if err != nil {
					t.Errorf("CreatePaginator() unexpected error = %v", err)
				}

				if !reflect.DeepEqual(gotData, tt.expectedData) {
					t.Errorf("CreatePaginator() gotData = %v, want %v", gotData, tt.expectedData)
				}
			}
		})
	}
}
