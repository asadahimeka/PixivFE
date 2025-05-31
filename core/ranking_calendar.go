// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"codeberg.org/pixivfe/pixivfe/core/requests"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// DayCalendar represents the data for a single day in the ranking calendar.
type DayCalendar struct {
	DayNumber   int        // The day of the month
	DateString  string     // Pixiv-compatible string that represents the date (format: YYYYMMDD)
	ImageURL    string     // Proxy URL to the image (optional, can be empty when no image is available)
	ArtworkLink string     // The link to the artwork page for this day
	Thumbnails  Thumbnails // Image links derived from ImageURL
}

// Precompiled CSS selector and regex.
var (
	artworkIDRegex = regexp.MustCompile(`/(\d+)_p0_(custom|square)1200\.jpg`)
)

// GetRankingCalendar retrieves and processes the ranking calendar data from Pixiv.
// It returns a slice of DayCalendar structs and any error encountered.
//
// iacore: so the funny thing about Pixiv is that they will return this month's data for a request of a future date. is it a bug or a feature?
func GetRankingCalendar(r *http.Request, mode string, year, month int) ([]DayCalendar, error) {
	url := GetRankingCalendarURL(mode, year, month)

	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	resp, err := requests.PerformGET(r.Context(), url, cookies, r.Header)
	if err != nil {
		return nil, err
	}

	doc, err := html.Parse(bytes.NewReader(resp.Body))
	if err != nil {
		return nil, err
	}

	links, err := extractImageLinks(r, doc)
	if err != nil {
		return nil, err
	}

	calendar := []DayCalendar{}
	dayCount := 0

	// Get the first day of the month
	firstDayOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)

	// Add empty days before the first day of the month
	calendar, dayCount = addEmptyDaysBefore(calendar, firstDayOfMonth, dayCount)

	// Get the number of days in the month
	numDays := daysIn(time.Month(month), year)

	// Add days of the month
	calendar, dayCount = addDaysOfMonth(calendar, links, dayCount, numDays, year, month)

	// Add empty days after the last day to complete the week
	calendar, dayCount = addEmptyDaysAfter(calendar, dayCount)

	_ = dayCount

	// Populate Thumbnails field
	for i := range calendar {
		thumbnails, err := PopulateThumbnailsFor(calendar[i].ImageURL)
		if err != nil {
			return calendar, nil
		}

		calendar[i].Thumbnails = thumbnails
	}

	return calendar, nil
}

// extractImageLinks extracts image links from the parsed HTML document.
func extractImageLinks(r *http.Request, doc *html.Node) ([]string, error) {
	var links []string

	for _, sel := range goquery.NewDocumentFromNode(doc).Find("img").EachIter() {
		node := sel.Nodes[0]
		var src string
		// Find data-src attribute in current node
		for _, attr := range node.Attr {
			if attr.Key == "data-src" {
				src = attr.Val

				break // Only need first data-src per node
			}
		}

		if src == "" {
			continue // Skip nodes without data-src
		}

		// Process found URL
		url := RewriteContentURLsNoEscape(r, []byte(src))

		links = append(links, string(url))
	}
	return links, nil
}

// addEmptyDaysBefore adds empty days to the calendar before the first day of the month.
func addEmptyDaysBefore(calendar []DayCalendar, firstDay time.Time, dayCount int) ([]DayCalendar, int) {
	emptyDays := int(firstDay.Weekday())
	for range emptyDays {
		calendar = append(calendar, DayCalendar{DayNumber: 0})
		dayCount++
	}
	return calendar, dayCount
}

// addDaysOfMonth adds the actual days of the month to the calendar.
func addDaysOfMonth(calendar []DayCalendar, links []string, dayCount, numDays, year, month int) ([]DayCalendar, int) {
	for i := range numDays {
		var imageURL string
		if i < len(links) {
			imageURL = links[i]
		}

		var artworkLink string
		if artworkID := extractArtworkID(imageURL); artworkID != "" {
			artworkLink = fmt.Sprintf("/artworks/%s", artworkID)
		}

		day := DayCalendar{
			DayNumber:   i + 1,
			ImageURL:    imageURL,
			ArtworkLink: artworkLink,
			DateString:  fmt.Sprintf("%d-%02d-%02d", year, month, i+1),
		}

		calendar = append(calendar, day)
		dayCount++
	}
	return calendar, dayCount
}

// addEmptyDaysAfter adds empty days to the calendar after the last day of the month to complete the week.
func addEmptyDaysAfter(calendar []DayCalendar, dayCount int) ([]DayCalendar, int) {
	for dayCount%7 != 0 {
		calendar = append(calendar, DayCalendar{DayNumber: 0})
		dayCount++
	}
	return calendar, dayCount
}

// daysIn returns the number of days in a given month and year.
func daysIn(month time.Month, year int) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// extractArtworkID extracts the artwork ID from the image URL.
func extractArtworkID(imageURL string) string {
	matches := artworkIDRegex.FindStringSubmatch(imageURL)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
