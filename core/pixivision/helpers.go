// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
Helpers for core pixivision code
*/
package pixivision

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"codeberg.org/pixivfe/pixivfe/core"
)

const defaultLanguage = "en" // defaultLanguage defines the default language used for pixivision requests

// generatePixivisionURL creates a URL for pixivision based on the route and language
func generatePixivisionURL(route string, lang []string) string {
	template := "https://www.pixivision.net/%s/%s"
	language := defaultLanguage
	availableLangs := []string{"en", "zh", "ja", "zh-tw", "ko"}

	// Validate and set the language if provided
	if len(lang) > 0 {
		t := lang[0]

		for _, i := range availableLangs {
			if t == i {
				language = t
			}
		}
	}

	return fmt.Sprintf(template, language, route)
}

// re_lang is a regular expression to extract the language code from a URL
var re_lang = regexp.MustCompile(`.*\/\/.*?\/(.*?)\/`)

// determineUserLang determines the language to use for the user_lang cookie
func determineUserLang(url string, lang ...string) string {
	// Check if language is provided in parameters
	if len(lang) > 0 && lang[0] != "" {
		return lang[0]
	}

	// Try to extract language from URL
	matches := re_lang.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}

	// Fallback to default language
	return defaultLanguage
}

// re_findid is a regular expression to extract the ID from a Pixiv link
var re_findid = regexp.MustCompile(`.*\/(\d+)`)

// parseIDFromPixivLink extracts the numeric ID from a Pixiv URL
func parseIDFromPixivLink(link string) string {
	matches := re_findid.FindStringSubmatch(link)

	// Check if the regex found a match AND captured the group
	// (should always be 2 elements if matched)
	if len(matches) < 2 {
		return ""
	}

	// If we have at least 2 elements, index 1 contains the captured digits
	return matches[1]
}

// r_img is a regular expression to extract the image URL from a CSS background-image property
var r_img = regexp.MustCompile(`background-image:\s*url\(([^)]+)\)`)

// parseBackgroundImage extracts the image URL from a CSS background-image property
func parseBackgroundImage(link string) string {
	matches := r_img.FindStringSubmatch(link)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

// processPixivisionImageURL handles the common logic for processing image URLs from pixivision.
func processPixivisionImageURL(r *http.Request, imgSrc string, reSizeQuality *regexp.Regexp) string {
	var finalImgSrc string

	if strings.HasPrefix(imgSrc, "https://source.pixiv.net") {
		// Special handling for "source.pixiv.net": direct proxy, no WebP conversion.
		finalImgSrc = strings.ReplaceAll(imgSrc, "https://source.pixiv.net", "/proxy/source.pixiv.net")
	} else if parsedImgURL, err := url.Parse(imgSrc); err == nil {
		// For other parsable URLs (including empty strings, which url.Parse handles):
		// attempt WebP conversion, then apply general proxying.
		webpConvertedSrc := core.GenerateMasterWebpURL(parsedImgURL, reSizeQuality)
		finalImgSrc = string(core.RewriteContentURLsNoEscape(r, []byte(webpConvertedSrc)))
	} else {
		// Fallback for non-parsable URLs (e.g., URL with invalid characters): use the original src.
		finalImgSrc = imgSrc
	}
	return finalImgSrc
}

// Better than constructing href values in templates manually
func normalizeHeadingLink(href string) string {
	if href == "" {
		return ""
	}

	// Strip language prefix if present (e.g. "/en/..." -> "/...")
	if len(href) >= 4 && href[0] == '/' && href[3] == '/' {
		href = "/" + href[4:]
	}

	// Add pixivision prefix
	return "/pixivision" + href
}
