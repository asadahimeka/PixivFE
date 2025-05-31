// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package template

import (
	"fmt"
	"html"
	"html/template"
	"io/fs"
	"log"
	"math"
	"net/url"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/core"
	"codeberg.org/pixivfe/pixivfe/i18n"
	"codeberg.org/pixivfe/pixivfe/server/assets"
	"codeberg.org/pixivfe/pixivfe/server/session"
)

// HTML is a type alias for template.HTML.
type HTML = template.HTML

// PageInfo represents information about a single page in pagination.
type PageInfo struct {
	Number int
	URL    string
}

// PaginationData contains all necessary information for rendering pagination controls.
type PaginationData struct {
	CurrentPage   int
	MaxPage       int
	Pages         []PageInfo
	HasPrevious   bool
	HasNext       bool
	PreviousURL   string
	NextURL       string
	FirstURL      string
	LastURL       string
	HasMaxPage    bool
	LastPage      int
	DropdownPages []PageInfo
}

// RelativeTimeData holds the numeric value and description for relative time.
type RelativeTimeData struct {
	Value       string
	Description string
	Time        string
}

// emojiList is a map of emoji shortcodes to their corresponding image IDs.
var emojiList = map[string]string{
	"normal":        "101",
	"surprise":      "102",
	"serious":       "103",
	"heaven":        "104",
	"happy":         "105",
	"excited":       "106",
	"sing":          "107",
	"cry":           "108",
	"normal2":       "201",
	"shame2":        "202",
	"love2":         "203",
	"interesting2":  "204",
	"blush2":        "205",
	"fire2":         "206",
	"angry2":        "207",
	"shine2":        "208",
	"panic2":        "209",
	"normal3":       "301",
	"satisfaction3": "302",
	"surprise3":     "303",
	"smile3":        "304",
	"shock3":        "305",
	"gaze3":         "306",
	"wink3":         "307",
	"happy3":        "308",
	"excited3":      "309",
	"love3":         "310",
	"normal4":       "401",
	"surprise4":     "402",
	"serious4":      "403",
	"love4":         "404",
	"shine4":        "405",
	"sweat4":        "406",
	"shame4":        "407",
	"sleep4":        "408",
	"heart":         "501",
	"teardrop":      "502",
	"star":          "503",
}

// regionList is a list of ISO 3166-1 alpha-2 codes (except South Sudan
// as it is not included in Pixiv's list)

// Total: 248
// Magic pattern: id="(\w+)".*\[\[ISO.*\]\] \|\| \[\[(\w+ ?\w+ ?\w+ ?\w+ ?\w+ ?)
var regionList = [][]string{
	{"AF", "Afghanistan"},
	{"AL", "Albania"},
	{"DZ", "Algeria"},
	{"AS", "American Samoa"},
	{"AD", "Andorra"},
	{"AO", "Angola"},
	{"AI", "Anguilla"},
	{"AQ", "Antarctica"},
	{"AG", "Antigua and Barbuda"},
	{"AR", "Argentina"},
	{"AM", "Armenia"},
	{"AW", "Aruba"},
	{"AU", "Australia"},
	{"AT", "Austria"},
	{"AZ", "Azerbaijan"},
	{"BH", "Bahrain"},
	{"GG", "Bailiwick of Guernsey"},
	{"BD", "Bangladesh"},
	{"BB", "Barbados"},
	{"BY", "Belarus"},
	{"BE", "Belgium"},
	{"BZ", "Belize"},
	{"BJ", "Benin"},
	{"BM", "Bermuda"},
	{"BT", "Bhutan"},
	{"BO", "Bolivia"},
	{"BA", "Bosnia and Herzegovina"},
	{"BW", "Botswana"},
	{"BV", "Bouvet Island"},
	{"BR", "Brazil"},
	{"IO", "British Indian Ocean Territory"},
	{"VG", "British Virgin Islands"},
	{"BN", "Brunei"},
	{"BG", "Bulgaria"},
	{"BF", "Burkina Faso"},
	{"BI", "Burundi"},
	{"KH", "Cambodia"},
	{"CM", "Cameroon"},
	{"CA", "Canada"},
	{"CV", "Cape Verde"},
	{"BQ", "Caribbean Netherlands"},
	{"KY", "Cayman Islands"},
	{"CF", "Central African Republic"},
	{"CL", "Chile"},
	{"CN", "China"},
	{"CX", "Christmas Island"},
	{"CC", "Cocos "},
	{"MF", "Collectivity of Saint Martin"},
	{"CO", "Colombia"},
	{"KM", "Comoros"},
	{"CK", "Cook Islands"},
	{"CR", "Costa Rica"},
	{"HR", "Croatia"},
	{"CY", "Cyprus"},
	{"CZ", "Czech Republic"},
	{"CD", "Democratic Republic of the Congo"},
	{"DK", "Denmark"},
	{"DJ", "Djibouti"},
	{"DM", "Dominica"},
	{"DO", "Dominican Republic"},
	{"TL", "East Timor"},
	{"EC", "Ecuador"},
	{"EG", "Egypt"},
	{"SV", "El Salvador"},
	{"GQ", "Equatorial Guinea"},
	{"ER", "Eritrea"},
	{"EE", "Estonia"},
	{"SZ", "Eswatini"},
	{"ET", "Ethiopia"},
	{"FK", "Falkland Islands"},
	{"FO", "Faroe Islands"},
	{"FM", "Federated States of Micronesia"},
	{"FI", "Finland"},
	{"FR", "France"},
	{"GF", "French Guiana"},
	{"PF", "French Polynesia"},
	{"TF", "French Southern and Antarctic Lands"},
	{"GA", "Gabon"},
	{"GE", "Georgia "},
	{"DE", "Germany"},
	{"GH", "Ghana"},
	{"GI", "Gibraltar"},
	{"GR", "Greece"},
	{"GL", "Greenland"},
	{"GD", "Grenada"},
	{"GP", "Guadeloupe"},
	{"GT", "Guatemala"},
	{"GN", "Guinea"},
	{"GW", "Guinea"},
	{"GY", "Guyana"},
	{"HT", "Haiti"},
	{"HM", "Heard Island and McDonald Islands"},
	{"HN", "Honduras"},
	{"HK", "Hong Kong"},
	{"HU", "Hungary"},
	{"IS", "Iceland"},
	{"IN", "India"},
	{"ID", "Indonesia"},
	{"IM", "Isle of Man"},
	{"IL", "Israel"},
	{"IT", "Italy"},
	{"JM", "Jamaica"},
	{"JP", "Japan"},
	{"JE", "Jersey"},
	{"JO", "Jordan"},
	{"KZ", "Kazakhstan"},
	{"KE", "Kenya"},
	{"NL", "Kingdom of the Netherlands"},
	{"KI", "Kiribati"},
	{"KW", "Kuwait"},
	{"KG", "Kyrgyzstan"},
	{"LV", "Latvia"},
	{"LB", "Lebanon"},
	{"LS", "Lesotho"},
	{"LR", "Liberia"},
	{"LY", "Libya"},
	{"LI", "Liechtenstein"},
	{"LT", "Lithuania"},
	{"LU", "Luxembourg"},
	{"MO", "Macau"},
	{"MG", "Madagascar"},
	{"MW", "Malawi"},
	{"MY", "Malaysia"},
	{"MV", "Maldives"},
	{"MT", "Malta"},
	{"MH", "Marshall Islands"},
	{"MQ", "Martinique"},
	{"MR", "Mauritania"},
	{"MU", "Mauritius"},
	{"YT", "Mayotte"},
	{"MX", "Mexico"},
	{"MD", "Moldova"},
	{"MC", "Monaco"},
	{"MN", "Mongolia"},
	{"ME", "Montenegro"},
	{"MS", "Montserrat"},
	{"MA", "Morocco"},
	{"MZ", "Mozambique"},
	{"MM", "Myanmar"},
	{"NA", "Namibia"},
	{"NR", "Nauru"},
	{"NP", "Nepal"},
	{"NC", "New Caledonia"},
	{"NZ", "New Zealand"},
	{"NI", "Nicaragua"},
	{"NE", "Niger"},
	{"NG", "Nigeria"},
	{"NF", "Norfolk Island"},
	{"MP", "Northern Mariana Islands"},
	{"MK", "North Macedonia"},
	{"NO", "Norway"},
	{"PK", "Pakistan"},
	{"PW", "Palau"},
	{"PA", "Panama"},
	{"PG", "Papua New Guinea"},
	{"PY", "Paraguay"},
	{"PH", "Philippines"},
	{"PN", "Pitcairn Islands"},
	{"PL", "Poland"},
	{"PT", "Portugal"},
	{"PR", "Puerto Rico"},
	{"QA", "Qatar"},
	{"IE", "Republic of Ireland"},
	{"CG", "Republic of the Congo"},
	{"RO", "Romania"},
	{"RU", "Russia"},
	{"RW", "Rwanda"},
	{"BL", "Saint Barth"},
	{"SH", "Saint Helena"},
	{"KN", "Saint Kitts and Nevis"},
	{"LC", "Saint Lucia"},
	{"PM", "Saint Pierre and Miquelon"},
	{"VC", "Saint Vincent and the Grenadines"},
	{"WS", "Samoa"},
	{"SM", "San Marino"},
	{"SA", "Saudi Arabia"},
	{"SN", "Senegal"},
	{"RS", "Serbia"},
	{"SC", "Seychelles"},
	{"SL", "Sierra Leone"},
	{"SG", "Singapore"},
	{"SX", "Sint Maarten"},
	{"SK", "Slovakia"},
	{"SI", "Slovenia"},
	{"SB", "Solomon Islands"},
	{"SO", "Somalia"},
	{"ZA", "South Africa"},
	{"GS", "South Georgia and the South "},
	{"KR", "South Korea"},
	// {"SS", "South Sudan"},
	{"ES", "Spain"},
	{"LK", "Sri Lanka"},
	{"SD", "Sudan"},
	{"SR", "Suriname"},
	{"SJ", "Svalbard and Jan Mayen"},
	{"SE", "Sweden"},
	{"CH", "Switzerland"},
	{"SY", "Syria"},
	{"TW", "Taiwan"},
	{"TJ", "Tajikistan"},
	{"TZ", "Tanzania"},
	{"TH", "Thailand"},
	{"BS", "The Bahamas"},
	{"GM", "The Gambia"},
	{"TK", "Tokelau"},
	{"TO", "Tonga"},
	{"TT", "Trinidad and Tobago"},
	{"TN", "Tunisia"},
	{"TR", "Turkey"},
	{"TM", "Turkmenistan"},
	{"TC", "Turks and Caicos Islands"},
	{"TV", "Tuvalu"},
	{"UG", "Uganda"},
	{"UA", "Ukraine"},
	{"AE", "United Arab Emirates"},
	{"GB", "United Kingdom"},
	{"US", "United States"},
	{"UM", "United States Minor Outlying Islands"},
	{"VI", "United States Virgin Islands"},
	{"UY", "Uruguay"},
	{"UZ", "Uzbekistan"},
	{"VU", "Vanuatu"},
	{"VA", "Vatican City"},
	{"VE", "Venezuela"},
	{"VN", "Vietnam"},
	{"WF", "Wallis and Futuna"},
	{"EH", "Western Sahara"},
	{"YE", "Yemen"},
	{"ZM", "Zambia"},
	{"ZW", "Zimbabwe"},
}

// iconCache holds all of our SVGs keyed by filename (without the “.svg” suffix).
var iconCache = make(map[string]string)

// LoadIcons scans the given directory for “.svg” files, reads each file into
// memory, wraps its bytes in HTML, and stores it in iconCache.
//
// If any operation fails, LoadIcons returns an error for the caller to handle.
func LoadIcons(dir string) error {
	// ReadDir returns a list of directory entries (files + subdirectories).
	entries, err := fs.ReadDir(assets.FS, dir)
	if err != nil {
		return fmt.Errorf("reading icons directory %q: %w", dir, err)
	}

	// Pre-allocate map capacity to avoid repeated growth.
	iconCache = make(map[string]string, len(entries))

	for _, entry := range entries {
		// Skip subdirectories — we only want files.
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Fast path: only care about .svg files.
		if !strings.HasSuffix(name, ".svg") {
			continue
		}

		// Build full path and slurp file in one call.
		fullPath := filepath.Join(dir, name)
		content, err := fs.ReadFile(assets.FS, fullPath)
		if err != nil {
			return fmt.Errorf("reading icon %q: %w", fullPath, err)
		}

		// Strip off the “.svg” extension to form the map key.
		key := strings.TrimSuffix(name, ".svg")

		// Wrap the raw bytes as HTML and store in the cache.
		iconCache[key] = string(content)
	}

	return nil
}

// ParseEmojis replaces emoji shortcodes in a string with corresponding image tags.
func ParseEmojis(s string) HTML {
	// Regular expression to match emoji shortcodes
	regex := regexp.MustCompile(`\(([^)]+)\)`)

	// Replace shortcodes with corresponding image tags
	parsedString := regex.ReplaceAllStringFunc(s, func(s string) string {
		s = s[1 : len(s)-1] // Get the string inside parentheses

		emojiID, found := emojiList[s] // Check if the string has a corresponding emoji ID
		if !found {
			return fmt.Sprintf("(%s)", s) // No replacement, return the original text
		}

		// FIXME: this ignores any user-set static image proxy, but you
		// can't pass *http.Request from within a Jet template
		//
		// code needs to be reworked so that this is called before RenderHTML
		proxy := session.GetProxyPrefix(config.GlobalConfig.ContentProxies.Static)

		return fmt.Sprintf(`<img src="%s/common/images/emoji/%s.png" alt="(%s)" class="emoji" />`, proxy, emojiID, s)
	})

	return HTML(parsedString)
}

// pixivRedirectRegex matches Pixiv redirect URLs (`/jump.php?...`).
//
// It captures the target URL component (the part after `?` up to a delimiter).
var pixivRedirectRegex = regexp.MustCompile(`\/jump\.php\?([^"&\s]+)`)

// absolutePixivLinkRegex matches absolute pixiv.net URLs.
//
// Used to identify standalone pixiv links for potential conversion to relative paths,
// distinct from those found within /jump.php redirects.
var absolutePixivLinkRegex = regexp.MustCompile(`https?://(?:[a-zA-Z0-9\-]+\.)*pixiv\.net[^\s<>"']*`)

// Constants defining the expected number of path segments for pixiv URLs,
// used when parsing paths for relativization.
const (
	// pathSegmentsCountDirect is for paths like "/users/ID".
	pathSegmentsCountDirect int = 2

	// pathSegmentsCountWithLang is for paths with a language code like "/en/users/ID".
	pathSegmentsCountWithLang int = 3
)

// tryMakePixivURLRelative attempts to convert a full pixiv.net URL string to a relative path
// if it matches specific patterns (users, artworks, novels).
//
// It returns the new URL string and a boolean indicating if a conversion occurred.
//
// If the input is not an HTTP/HTTPS pixiv.net URL eligible for conversion,
// it returns the original string and false.
func tryMakePixivURLRelative(fullURLString string) (string, bool) {
	parsedTargetURL, err := url.Parse(fullURLString)
	if err != nil {
		// Malformed URLs cannot be processed.
		return fullURLString, false
	}

	// Only http/https schemes are considered for relativization.
	if parsedTargetURL.Scheme != "http" && parsedTargetURL.Scheme != "https" {
		return fullURLString, false
	}

	if !strings.Contains(parsedTargetURL.Host, "pixiv.net") {
		// Not a pixiv.net URL.
		return fullURLString, false
	}

	// At this point, it's a valid http/https pixiv.net URL.
	// We also clean the path to handle any extraneous slashes.
	cleanedPath := path.Clean(parsedTargetURL.Path)
	query := parsedTargetURL.Query()

	// Handle novel URLs: /novel/show.php?id=... or /lang/novel/show.php?id=...
	// These are converted to /novels/ID.
	if strings.Contains(cleanedPath, "/novel/show.php") {
		if id := query.Get("id"); id != "" {
			return "/novel/" + id, true
		}
	}

	// Handle user/artwork paths: /users/ID, /artworks/ID, /lang/users/ID, /lang/artworks/ID.
	// These are converted to /users/ID or /artworks/ID.
	trimmedPath := strings.TrimPrefix(cleanedPath, "/")
	pathParts := strings.Split(trimmedPath, "/")

	// After cleaning and trimming, if original path was "/" or empty,
	// trimmedPath might be empty or ".", leading to pathParts like [""] or ["."].
	// These won't match lengths 2 or 3, correctly falling through.

	if len(pathParts) == pathSegmentsCountDirect { // e.g., "users", "123"
		key := pathParts[0]
		id := pathParts[1]
		if (key == "users" || key == "artworks") && id != "" { // Ensure ID is present.
			return "/" + key + "/" + id, true
		}
	}

	if len(pathParts) == pathSegmentsCountWithLang { // e.g., "en", "users", "123"
		// pathParts[0] is lang code (e.g., "en"), pathParts[1] is key, pathParts[2] is ID.
		key := pathParts[1]
		id := pathParts[2]
		if (key == "users" || key == "artworks") && id != "" { // Ensure ID is present.
			return "/" + key + "/" + id, true // Standardize to relative path, dropping lang code.
		}
	}

	// Matched a pixiv.net URL, but not a pattern we convert to relative
	// (e.g., pixiv.net/home, pixiv.net/ranking.php).
	return fullURLString, false
}

// processPixivRedirectMatch is a helper for pixivRedirectRegex.ReplaceAllStringFunc.
//
// It extracts the target URL from a /jump.php? redirect, URL-decodes it,
// attempts to convert known pixiv URLs to relative paths (via tryMakePixivURLRelative),
// and sanitizes `javascript:` URIs to an empty string. Other URLs are returned decoded.
func processPixivRedirectMatch(match string) string {
	// The match must be like "/jump.php?URL_PARAMS". Ensure query part exists.
	if len(match) <= 10 || !strings.HasPrefix(match, "/jump.php?") {
		// Not a valid jump link structure or empty query; return original match.
		return match
	}

	encodedURL := match[10:] // Extract the part after "/jump.php?"

	decodedURL, err := url.QueryUnescape(encodedURL)
	if err != nil {
		// Unescape failed; return original jump.php match as a fallback.
		return match
	}

	// Attempt to make it a relative pixiv URL if applicable.
	if relativeURL, converted := tryMakePixivURLRelative(decodedURL); converted {
		return relativeURL
	}

	// If not converted by tryMakePixivURLRelative, the decodedURL could be:
	// - An external HTTP/S link.
	// - A pixiv URL not matching relativization patterns (e.g., pixiv.net/home).
	// - A URL with a non-HTTP/S scheme (e.g., mailto:, javascript:).
	// - A malformed URL.
	// We need to parse it to specifically check for and sanitize `javascript:` URIs.
	parsedTargetURL, err := url.Parse(decodedURL)
	if err != nil {
		// If decodedURL itself is malformed and cannot be parsed (e.g., "http://["),
		// return the decoded string. It couldn't be made relative or sanitized by scheme.
		return decodedURL
	}

	if strings.ToLower(parsedTargetURL.Scheme) == "javascript" {
		return "" // Sanitize javascript: URIs by returning an empty string.
	}

	// For all other cases (non-convertible pixiv URLs, external links, mailto:, ftp:, etc.),
	// return the decoded URL.
	return decodedURL
}

// ParseDescriptionURLs resolves pixiv's /jump.php? redirect URLs to their targets
// and converts specific pixiv.net URLs to relative paths.
//
// Relativization occurs for both resolved redirect targets and standalone
// absolute pixiv.net URLs found in the string.
//
// `javascript:` URIs from /jump.php? redirects are sanitized to an empty string for security.
//
// Other URLs from redirects (e.g., external links, mailto:) are URL-decoded and preserved.
//
// Standalone URLs not matching absolute pixiv.net patterns for relativization, or those that are
// not HTTP/HTTPS, remain unchanged by this function's direct processing.
//
// Returns the processed string.
func ParseDescriptionURLs(description string) string {
	// First, process /jump.php? redirect URLs.
	description = pixivRedirectRegex.ReplaceAllStringFunc(description, processPixivRedirectMatch)

	// Second, process standalone absolute pixiv.net URLs that might not have been in redirects.
	// This ensures direct pixiv links (not via jump.php) are also considered for relativization.
	description = absolutePixivLinkRegex.ReplaceAllStringFunc(description, func(match string) string {
		// tryMakePixivURLRelative attempts the conversion based on defined patterns.
		if relativeURL, converted := tryMakePixivURLRelative(match); converted {
			return relativeURL
		}
		// If not converted (e.g., it's https://www.pixiv.net/home), keep the original absolute URL.
		return match
	})

	return description
}

// EscapeString escapes a string for use in a URL query.
func EscapeString(s string) string {
	escaped := url.QueryEscape(s)

	return escaped
}

// UnescapeHTMLString unescapes all HTML entities to literal characters.
func UnescapeHTMLString(s string) string {
	return html.UnescapeString(s)
}

// ParseTime formats a time.Time value as a string in the format "2006-01-02 15:04".
func ParseTime(date time.Time) string {
	return date.Format("2006-01-02 15:04")
}

// ParseTimeCustomFormat formats a time.Time value as a string using a custom format.
func ParseTimeCustomFormat(date time.Time, format string) string {
	return date.Format(format)
}

// NaturalTime formats a time.Time value as a natural language string.
//
// TODO: tailor the format per locale.
func NaturalTime(date time.Time) string {
	return date.Format("Monday, 2 January 2006, at 3:04 PM")
}

// RelativeTime returns a RelativeTimeData struct with a relative description based on the given date.
//
// The "Yesterday" case is special and is triggered when the date matches exactly the previous
// calendar day (i.e., same year, month, and previous day), regardless of the exact number of
// hours that have elapsed.
//
// TODO: tailor the format per locale.
func RelativeTime(date time.Time) RelativeTimeData {
	now := time.Now()
	duration := now.Sub(date)

	// local helper function to choose the correct singular/plural unit.
	pluralize := func(value int, singular, plural string) string {
		if value == 1 {
			return singular
		}

		return plural
	}

	// For future dates, simply show the day and full date/time formatting.
	if duration < 0 {
		return RelativeTimeData{
			Value:       date.Format("2"),
			Description: date.Format("January 2006 3:04 PM"),
		}
	}

	// Less than one minute ago.
	if duration < time.Minute {
		return RelativeTimeData{
			Value: "Just now",
		}
	}

	// Less than one hour: display minutes.
	if duration < time.Hour {
		minutes := int(duration.Minutes())

		return RelativeTimeData{
			Value:       fmt.Sprintf("%d %s", minutes, pluralize(minutes, "minute", "minutes")),
			Description: "ago",
		}
	}

	// Less than one day: display hours.
	if duration < 24*time.Hour {
		hours := int(duration.Hours())

		return RelativeTimeData{
			Value:       fmt.Sprintf("%d %s", hours, pluralize(hours, "hour", "hours")),
			Description: "ago",
		}
	}

	// Check if the date corresponds to 'yesterday'
	yesterday := now.AddDate(0, 0, -1)
	if date.Year() == yesterday.Year() && date.Month() == yesterday.Month() && date.Day() == yesterday.Day() {
		return RelativeTimeData{
			Value:       "Yesterday",
			Description: "at",
			Time:        date.Format("3:04 PM"),
		}
	}

	// Less than one week: display days.
	if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)

		return RelativeTimeData{
			Value:       fmt.Sprintf("%d %s", days, pluralize(days, "day", "days")),
			Description: "ago",
		}
	}

	// Less than one month (using a 31-day threshold): display weeks.
	if duration < 31*24*time.Hour {
		weeks := int(duration.Hours() / (24 * 7))

		return RelativeTimeData{
			Value:       fmt.Sprintf("%d %s", weeks, pluralize(weeks, "week", "weeks")),
			Description: "ago",
		}
	}

	// Calculate total month difference (ignoring day differences for simplicity).
	yearDiff := now.Year() - date.Year()
	monthDiff := int(now.Month()) - int(date.Month())
	months := yearDiff*12 + monthDiff

	// Less than one year: display months.
	if months < 12 {
		return RelativeTimeData{
			Value:       fmt.Sprintf("%d %s", months, pluralize(months, "month", "months")),
			Description: "ago",
		}
	}

	// Otherwise, show years.
	years := months / 12

	return RelativeTimeData{
		Value:       fmt.Sprintf("%d %s", years, pluralize(years, "year", "years")),
		Description: "ago",
	}
}

// PrettyNumber pretty prints an integer with commas as thousands separators.
//
// Negative numbers are handled by first setting aside the sign.
func PrettyNumber(n int) string {
	// Determine sign
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n // work with an absolute value
	}

	// Convert the integer to a string using strconv.Itoa.
	numStr := strconv.Itoa(n)

	// If there are no more than three digits, no comma is needed.
	if len(numStr) <= 3 {
		return sign + numStr
	}

	// Build digit groups from right to left.
	var groups []string
	for len(numStr) > 3 {
		// Extract the last three digits.
		groups = append([]string{numStr[len(numStr)-3:]}, groups...)
		// Remove the processed group.
		numStr = numStr[:len(numStr)-3]
	}
	// Prepend any remaining digits.
	if len(numStr) > 0 {
		groups = append([]string{numStr}, groups...)
	}

	// Join the groups with commas, and add a preceding negative sign if needed.
	return sign + strings.Join(groups, ",")
}

// OrdinalNumeral returns an integer as its ordinal form in a string.
func OrdinalNumeral(num int) string {
	// Handle special cases for 11, 12, 13
	lastTwo := num % 100
	if lastTwo == 11 || lastTwo == 12 || lastTwo == 13 {
		return strconv.Itoa(num) + "th"
	}

	// Handle cases based on last digit
	lastDigit := num % 10
	switch lastDigit {
	case 1:
		return strconv.Itoa(num) + "st"
	case 2:
		return strconv.Itoa(num) + "nd"
	case 3:
		return strconv.Itoa(num) + "rd"
	default:
		return strconv.Itoa(num) + "th"
	}
}

// CreatePaginator generates pagination data based on the current page and maximum number of pages.
// It returns a PaginationData struct containing all necessary information for rendering pagination controls.
//
// Parameters:
// - base: The base part of the pagination URL.
// - ending: The ending part of the pagination URL.
// - current_page: The current page being displayed.
// - max_page: Maximum number of pages (-1 if unknown).
// - page_margin: Number of pages to display before and after the current page.
// - dropdown_offset: Number of pages to include in dropdown before and after current page.
func CreatePaginator(base, ending string, current_page, max_page, page_margin, dropdown_offset int) (PaginationData, error) {
	// Validate input parameters
	if current_page < 1 {
		return PaginationData{}, fmt.Errorf("current_page must be greater than or equal to 1, got %d", current_page)
	}

	if page_margin < 0 {
		return PaginationData{}, fmt.Errorf("page_margin must be non-negative, got %d", page_margin)
	}

	if dropdown_offset < 0 {
		return PaginationData{}, fmt.Errorf("dropdown_offset must be non-negative, got %d", dropdown_offset)
	}

	// Validation for users that don't have any artworks
	// NOTE: the following breaks the current max_page implementation, commenting it out for now
	// if max_page < 1 {
	// 	max_page = 1
	// }

	// Validation for max_page in relation to current_page
	// if max_page < current_page {
	// 	return PaginationData{}, fmt.Errorf("max_page (%d) must be greater than or equal to current_page (%d) when specified", max_page, current_page)
	// }

	hasMaxPage := max_page != -1

	pageURL := func(page int) string {
		return fmt.Sprintf(`%s%d%s`, base, page, ending)
	}

	// Helper function to generate a range of pages
	generatePageRange := func(start, end int) []PageInfo {
		if start > end {
			return []PageInfo{}
		}

		pages := make([]PageInfo, 0, end-start+1)
		for i := start; i <= end; i++ {
			pages = append(pages, PageInfo{Number: i, URL: pageURL(i)})
		}

		return pages
	}

	// Calculate the range of pages to display
	start := max(1, current_page-page_margin)

	end := current_page + page_margin
	if hasMaxPage {
		end = min(max_page, end)
	}

	// Generate page information for the range
	pages := generatePageRange(start, end)

	var lastPage int
	if len(pages) > 0 {
		lastPage = pages[len(pages)-1].Number
	} else {
		lastPage = current_page
	}

	// Generate dropdown pages
	dropdownStart := max(1, current_page-dropdown_offset)

	dropdownEnd := current_page + dropdown_offset
	if hasMaxPage {
		dropdownEnd = min(max_page, dropdownEnd)
	}

	dropdownPages := generatePageRange(dropdownStart, dropdownEnd)

	// Calculate previous and next URLs
	var previousURL, nextURL string
	if current_page > 1 {
		previousURL = pageURL(current_page - 1)
	}

	if !hasMaxPage || current_page < max_page {
		nextURL = pageURL(current_page + 1)
	}

	// Create and return the PaginationData struct
	return PaginationData{
		CurrentPage:   current_page,
		MaxPage:       max_page,
		Pages:         pages,
		HasPrevious:   current_page > 1,
		HasNext:       !hasMaxPage || current_page < max_page,
		PreviousURL:   previousURL,
		NextURL:       nextURL,
		FirstURL:      pageURL(1),
		LastURL:       pageURL(max_page),
		HasMaxPage:    hasMaxPage,
		LastPage:      lastPage,
		DropdownPages: dropdownPages,
	}, nil
}

var genreMap = map[string]string{
	"1":  "Romance",
	"2":  "Isekai fantasy",
	"3":  "Contemporary fantasy",
	"4":  "Mystery",
	"5":  "Horror",
	"6":  "Sci-fi",
	"7":  "Literature",
	"8":  "Drama",
	"9":  "Historical pieces",
	"10": "BL (yaoi)",
	"11": "Yuri",
	"12": "For kids",
	"13": "Poetry",
	"14": "Essays/non-fiction",
	"15": "Screenplays/scripts",
	"16": "Reviews/opinion pieces",
	"17": "Other",
}

// GetNovelGenre returns the genre name for a given genre ID.
func GetNovelGenre(s string) string {
	if genre, ok := genreMap[s]; ok {
		return i18n.Tr(genre)
	}
	return i18n.Sprintf("(Unknown Genre: %s)", s)
}

// IsFirstPathPart checks if the first part of the current path matches the given path.
func IsFirstPathPart(currentPath, pathToCheck string) bool {
	// Trim any trailing slashes from both paths
	currentPath = strings.TrimRight(currentPath, "/")
	pathToCheck = strings.TrimRight(pathToCheck, "/")

	// Split the current path into parts
	parts := strings.SplitN(currentPath, "/", 3)

	// Check if we have at least two parts (empty string and the first path part)
	if len(parts) < 2 {
		return false
	}

	// Compare the first path part with the pathToCheck
	return "/"+parts[1] == pathToCheck
}

// IsLastPathPart checks if the last part of the current path matches the given path.
func IsLastPathPart(currentPath, pathToCheck string) bool {
	// Parse the currentPath to remove query parameters
	u, err := url.Parse(currentPath)
	if err == nil {
		currentPath = u.Path
	}

	// Trim any trailing slashes from both paths
	currentPath = strings.TrimRight(currentPath, "/")
	pathToCheck = strings.TrimRight(pathToCheck, "/")

	// Split the current path into parts
	parts := strings.Split(currentPath, "/")

	// Check if there is at least one part
	if len(parts) < 1 {
		return false
	}

	// Get the last part
	lastPart := parts[len(parts)-1]

	// Compare the last path part with the pathToCheck
	return "/"+lastPart == pathToCheck
}

// IsFullPath checks if the entire current path matches the given path,
// excluding query parameters and fragments.
func IsFullPath(currentPath, pathToCheck string) bool {
	// Parse the currentPath to remove query parameters and fragment
	u, err := url.Parse(currentPath)
	if err == nil {
		currentPath = u.Path
	}

	// Trim any trailing slashes from both paths
	currentPath = strings.TrimRight(currentPath, "/")
	pathToCheck = strings.TrimRight(pathToCheck, "/")

	// Compare the cleaned paths
	return currentPath == pathToCheck
}

const (
	PIXIVISION_CATEGORY_EXPLORE = iota
	PIXIVISION_CATEGORY_CREATE
	PIXIVISION_CATEGORY_DISCOVER
)

type PixivisionCategory struct {
	Type int
	ID   string
}

// PixivisionCategoryID returns the type of category and the ID of the
// category from its name.
func PixivisionCategoryID(name string) PixivisionCategory {
	switch name {
	case "Illustrations":
		return PixivisionCategory{Type: PIXIVISION_CATEGORY_EXPLORE, ID: "illustration"}
	case "Manga":
		return PixivisionCategory{Type: PIXIVISION_CATEGORY_EXPLORE, ID: "manga"}
	case "Novels":
		return PixivisionCategory{Type: PIXIVISION_CATEGORY_EXPLORE, ID: "novels"}
	case "Cosplay":
		return PixivisionCategory{Type: PIXIVISION_CATEGORY_EXPLORE, ID: "cosplay"}
	case "Music":
		return PixivisionCategory{Type: PIXIVISION_CATEGORY_EXPLORE, ID: "music"}
	case "Goods":
		return PixivisionCategory{Type: PIXIVISION_CATEGORY_EXPLORE, ID: "goods"}
	case "Tutorials":
		return PixivisionCategory{Type: PIXIVISION_CATEGORY_CREATE, ID: "how-to-draw"}
	case "Behind the Art":
		return PixivisionCategory{Type: PIXIVISION_CATEGORY_CREATE, ID: "draw-step-by-step"}
	case "Materials":
		return PixivisionCategory{Type: PIXIVISION_CATEGORY_CREATE, ID: "textures"}
	case "References":
		return PixivisionCategory{Type: PIXIVISION_CATEGORY_CREATE, ID: "art-references"}
	case "Other Guides":
		return PixivisionCategory{Type: PIXIVISION_CATEGORY_CREATE, ID: "how-to-make"}
	case "Featured":
		return PixivisionCategory{Type: PIXIVISION_CATEGORY_DISCOVER, ID: "recommend"}
	case "Interviews":
		return PixivisionCategory{Type: PIXIVISION_CATEGORY_DISCOVER, ID: "interview"}
	case "Columns":
		return PixivisionCategory{Type: PIXIVISION_CATEGORY_DISCOVER, ID: "column"}
	case "News":
		return PixivisionCategory{Type: PIXIVISION_CATEGORY_DISCOVER, ID: "news"}
	case "Deskwatch":
		return PixivisionCategory{Type: PIXIVISION_CATEGORY_DISCOVER, ID: "deskwatch"}
	case "We Tried It!":
		return PixivisionCategory{Type: PIXIVISION_CATEGORY_DISCOVER, ID: "try-out"}
	}

	return PixivisionCategory{Type: PIXIVISION_CATEGORY_EXPLORE, ID: "illustration"}
}

// FormatWorkIDs formats an integer slice of work IDs
// into a comma-separated string.
//
// Used to correctly format work IDs for the
// recent works API call in artworkFast.jet.html.
func FormatWorkIDs(ids []int) string {
	if len(ids) == 0 {
		return ""
	}

	// Convert ints to strings
	strIDs := make([]string, len(ids))
	for i, id := range ids {
		strIDs[i] = strconv.Itoa(id)
	}

	// Join with commas
	return strings.Join(strIDs, ",")
}

var (
	furiganaPattern = regexp.MustCompile(`\[\[rb:\s*(.+?)\s*>\s*(.+?)\s*\]\]`)
	chapterPattern  = regexp.MustCompile(`\[chapter:\s*(.+?)\s*\]`)
	jumpURIPattern  = regexp.MustCompile(`\[\[jumpuri:\s*(.+?)\s*>\s*(.+?)\s*\]\]`)
	jumpPagePattern = regexp.MustCompile(`\[jump:\s*(\d+?)\s*\]`)
	newPagePattern  = regexp.MustCompile(`\s*\[newpage\]\s*`)
)

func ParseNovelContent(s string) HTML {
	// Replace furigana markup with HTML ruby tags
	furiganaTemplate := `<ruby>$1<rp>(</rp><rt>$2</rt><rp>)</rp></ruby>`
	s = furiganaPattern.ReplaceAllString(s, furiganaTemplate)

	// Replace chapter markup with HTML h2 tags
	chapterTemplate := `<h2>$1</h2>`
	s = chapterPattern.ReplaceAllString(s, chapterTemplate)

	// Replace jump URI markup with HTML anchor tags
	jumpURITemplate := `<a href="$2" target="_blank">$1</a>`
	s = jumpURIPattern.ReplaceAllString(s, jumpURITemplate)

	// Replace jump page markup with HTML anchor tags
	jumpPageTemplate := `<a href="#$1">To page $1</a>`
	s = jumpPagePattern.ReplaceAllString(s, jumpPageTemplate)

	// Handle newpage markup
	if strings.Contains(s, "[newpage]") {
		// Prepend <hr id="1"/> to the page if [newpage] is present
		s = `<hr id="1"/>` + s
		pageIdx := 1

		// Create a slice of all matches
		matches := newPagePattern.FindAllString(s, -1)

		// Replace each match one by one
		for _, match := range matches {
			replacement := fmt.Sprintf(`<br /><hr id="%d"/>`, pageIdx+1)
			s = strings.Replace(s, match, replacement, 1)
			pageIdx++
		}
	}

	// Replace newlines with HTML line breaks
	s = strings.ReplaceAll(s, "\n", "<br />")

	return HTML(s)
}

// RenderIcon returns an SVG (as HTML), optionally
// injecting a trusted CSS class into the <svg> tag.
//
// iconName is developer‑controlled, and iconCache only ever
// contains vetted SVG blobs; directly returning HTML
// is therefore safe.
//
// If iconName is not found, a simple text placeholder is returned.
//
// The optional classes parameter can be used to inject a single
// CSS class into the <svg> tag. Only the first string in classes
// is used.
func RenderIcon(iconName string, classes ...string) string {
	raw, ok := iconCache[iconName]
	if !ok {
		// iconName is guaranteed safe, so we can inline it directly
		return "[missing icon: " + iconName + "]"
	}

	svg := string(raw)

	// if a class was provided, inject it into the <svg> tag
	if len(classes) > 0 && classes[0] != "" {
		// classes[0] is also guaranteed safe
		svg = strings.Replace(svg, "<svg", `<svg class="`+classes[0]+`"`, 1)
	}

	return svg
}

// confirmed: distinct int is still int
// func init() {
// 	panic(reflect.TypeFor[core.AiType]().Kind())
// }

func get[T any](r reflect.Value, fieldName string) (T, bool) {
	rT := reflect.TypeFor[T]()
	var zero T
	// log.Println("get", r.Type(), targetType)
	field := r.FieldByName(fieldName)
	if !field.IsValid() {
		return zero, false
	}
	if field.Type() != rT {
		return zero, false
	}
	// log.Println("field", r.Type(), field.Interface().(T), fieldName, field.Type())
	return field.Interface().(T), true
}

func lines(s string) []string {
	res := []string{}
	for line := range strings.SplitSeq(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			res = append(res, line)
		}
	}
	return res
}

// returns ok, reason
func ShouldHide(CookieList map[string]string, work_ any) (bool, string) {
	work := reflect.ValueOf(work_)
	if work.Kind() == reflect.Ptr {
		work = work.Elem()
	}
	if !(work.Type().Kind() == reflect.Struct) {
		log.Panicf("shouldHide: invalid type: %#v", reflect.TypeOf(work))
	}

	AiType, hasAiType := get[core.AiType](work, "AiType")
	XRestrict, hasXRestrict := get[core.XRestrict](work, "XRestrict")
	SanityLevel, hasSanityLevel := get[core.SanityLevel](work, "SanityLevel")

	hideArtAI := CookieList[string(session.Cookie_VisibilityArtAI)] == "hide"
	hideArtR18 := CookieList[string(session.Cookie_VisibilityArtR18)] == "hide"
	hideArtR18G := CookieList[string(session.Cookie_VisibilityArtR18G)] == "hide"

	if hasAiType && hideArtAI && AiType == core.AI {
		return true, "AI Art"
	}
	if hasXRestrict && hideArtR18 && XRestrict == core.R18 {
		return true, "R18"
	}
	if hasXRestrict && hideArtR18G && XRestrict == core.R18G {
		return true, "R18G"
	}

	// todo: filter by sanity level?
	_, _ = SanityLevel, hasSanityLevel

	hideArtists := lines(CookieList[string(session.Cookie_BlacklistArtist)])
	hideTags := lines(CookieList[string(session.Cookie_BlacklistTag)])

	UserID, hasUserID := get[string](work, "UserID")
	Tags, hasTags := get[[]string](work, "Tags")

	if hasUserID && slices.Contains(hideArtists, UserID) {
		return true, "UserID"
	}
	if hasTags {
		for _, tag := range Tags {
			if slices.Contains(hideTags, tag) {
				return true, fmt.Sprintf("Tag: %s", tag)
			}
		}
	}

	return false, ""
}

// getTemplateFunctions returns a map of custom template functions for use in HTML templates.
func getTemplateFunctions() template.FuncMap {
	return template.FuncMap{
		"icon": RenderIcon,
		"parseEmojis": func(s string) HTML {
			return ParseEmojis(s)
		},
		// TODO: update the name for this function call in templates
		"parsePixivRedirect": func(s string) string {
			return ParseDescriptionURLs(s)
		},
		"escapeString": func(s string) string {
			return EscapeString(s)
		},
		"unescapeHTMLString": func(s string) string {
			return UnescapeHTMLString(s)
		},
		"isEmphasize": func(s string) bool {
			switch s {
			case "R-18", "R-18G":
				return true
			}

			return false
		},
		"getSpecialEffects": func(s string) string {
			switch s {
			case "pixivSakuraEffect":
				return "/proxy/source.pixiv.net/special/seasonal-effect-tag/pixiv-sakura-effect/effect.png"
			}

			return ""
		},
		"parseTime": func(date time.Time) string {
			return ParseTime(date)
		},
		"parseTimeCustomFormat": func(date time.Time, format string) string {
			return ParseTimeCustomFormat(date, format)
		},
		"naturalTime": func(date time.Time) string {
			return NaturalTime(date)
		},
		"relativeTime": func(date time.Time) RelativeTimeData {
			return RelativeTime(date)
		},
		"prettyNumber": func(n int) string {
			return PrettyNumber(n)
		},
		"createPaginator": func(base, ending string, current_page, max_page, page_margin, dropdown_offset int) PaginationData {
			paginationData, err := CreatePaginator(base, ending, current_page, max_page, page_margin, dropdown_offset)
			if err != nil {
				fmt.Printf("Error creating paginator: %v", err)
				// Return an empty PaginationData in case of error
				return PaginationData{}
			}
			return paginationData
		},
		"parseNovelContent": func(s string) HTML {
			return ParseNovelContent(s)
		},
		"getNovelGenre": GetNovelGenre,
		"floor": func(i float64) int {
			return int(math.Floor(i))
		},
		"pixivisionCategoryID": PixivisionCategoryID,
		"ordinalNumeral":       OrdinalNumeral,
		"unfinishedQuery":      unfinishedQuery,
		"replaceQuery":         replaceQuery,
		"isFirstPathPart":      IsFirstPathPart,
		"isLastPathPart":       IsLastPathPart,
		"IsFullPath":           IsFullPath,
		"shouldHide": func(x map[string]string, y any) bool {
			a, reason := ShouldHide(x, y)
			_ = reason
			// jet doesn't support multiple return values
			// if we want to show the reason (why an artwork is hidden) to the user, we need to modify this func to take *string
			return a
		},
		"getRegionList": func() [][]string {
			return regionList
		},

		"FormatWorkIDs": FormatWorkIDs,
	}
}
