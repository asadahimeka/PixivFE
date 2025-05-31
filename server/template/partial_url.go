// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
This file provides utilities for manipulating URL paths and query parameters.
*/
package template

import (
	"fmt"
	"strings"
)

// PartialURL represents a simplified URL with a path and query parameters.
type PartialURL struct {
	Path  string
	Query map[string]string
}

// LowercaseFirstChar returns a string with the first character lowercased.
func LowercaseFirstChar(s string) string {
	return strings.ToLower(s[0:1]) + s[1:]
}

// unfinishedQuery builds a URL string from a PartialURL by including all query parameters except a specified key,
// which it leaves empty and places at the end of the query string.
//
// Useful for preparing a URL where the value for the key is appended later (e.g. during query parameter replacement).
func unfinishedQuery(url PartialURL, key string) string {
	// Start building the URL with the path
	result := url.Path
	// Flag to check if we are adding the first query parameter
	firstQueryPair := true

	// Iterate over the query parameters, excluding the specified key
	for queryKey, queryValue := range url.Query {
		// Lowercase the first character of the key to standardize it
		queryKey = LowercaseFirstChar(queryKey)

		// Skip the specified key to handle it separately later
		if queryKey == key {
			continue
		}

		// Skip parameters with empty values to avoid cluttering the URL
		if queryValue == "" {
			continue
		}

		// Add '?' before the first query parameter, '&' before subsequent ones
		if firstQueryPair {
			result += "?"
			firstQueryPair = false
		} else {
			result += "&"
		}
		// Append the key-value pair to the result
		result += fmt.Sprintf("%s=%s", queryKey, queryValue)
	}

	// Append the specified key at the end with an empty value
	// This ensures the key appears last in the query string and can be easily modified
	var queryParamSeparator string
	if firstQueryPair {
		// No query parameters were added before, so use '?'
		queryParamSeparator = "?"
	} else {
		// Query parameters were added before, so use '&'
		queryParamSeparator = "&"
	}

	result += fmt.Sprintf("%s%s=", queryParamSeparator, key)

	return result
}

// replaceQuery constructs a URL string by replacing the value of the specified key in the query parameters.
//
// It uses UnfinishedQuery to build the base URL and appends the new value for the key at the end.
func replaceQuery(url PartialURL, key string, value string) string {
	// Build the unfinished query and append the new value for the key
	return unfinishedQuery(url, key) + value
}
