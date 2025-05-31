// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"codeberg.org/pixivfe/pixivfe/audit"
	"codeberg.org/pixivfe/pixivfe/core/requests"
	"codeberg.org/pixivfe/pixivfe/server/session"
)

const (
	resultsLimit    = 100
	fakeBookmarkTag = "虚偽users入りタグ" //nolint:gosmopolitan
)

// descendingSuffixes defines popularity thresholds in descending order
var descendingSuffixes = []string{"100000", "50000", "30000", "10000", "5000", "1000", "500", "100", "50"}

// SearchResponse represents the JSON structure returned by the Pixiv API for the search endpoint
type SearchResponse struct {
	IllustManga struct {
		Data []ArtworkBrief `json:"data"`
	} `json:"illustManga"`
}

// isValidResult checks if an artwork meets the search criteria
func isValidResult(item ArtworkBrief) bool {
	// Check for the fake bookmark count tag
	for _, tag := range item.Tags {
		trimmedTag := strings.TrimSpace(tag)
		if strings.EqualFold(trimmedTag, fakeBookmarkTag) {
			return false
		}
	}

	// Invalidate if AI-generated since most fake results are
	if item.AiType == 2 {
		return false
	}

	return true
}

// processSearchResults filters and adds valid results to the artworks slice
func processSearchResults(items []ArtworkBrief, artworks *[]ArtworkBrief) bool {
	for _, item := range items {
		if isValidResult(item) {
			*artworks = append(*artworks, item)
			if len(*artworks) >= resultsLimit {
				return true
			}
		}
	}
	return false
}

// fetchResultsForSuffix retrieves all results for a specific popularity threshold
func fetchResultsForSuffix(ctx context.Context, r *http.Request, settings ArtworkSearchSettings, suffix string, suffixArtworks *[]ArtworkBrief) error {
	page := 1

	for len(*suffixArtworks) < resultsLimit {
		// Copy settings to avoid modifying the original
		currentSettings := settings

		// Adjust settings.Name to include the suffix
		currentSettings.Name = fmt.Sprintf("%s%susers入り", settings.Name, suffix)

		// Set page number
		currentSettings.Page = strconv.Itoa(page)

		url := GetArtworkSearchURL(currentSettings.ReturnMap())

		cookies := map[string]string{
			"PHPSESSID": session.GetUserToken(r),
		}

		body, err := requests.FetchJSONBodyField(ctx, url, cookies, r.Header)
		if err != nil {
			audit.GlobalAuditor.Logger.Errorw("Error fetching results", "error", err, "suffix", suffix, "page", page)
			return err
		}

		rawResp := RewriteContentURLs(r, body)

		// Unmarshal the proxied JSON body into SearchResponse
		var resp SearchResponse
		err = json.Unmarshal(rawResp, &resp)
		if err != nil {
			audit.GlobalAuditor.Logger.Errorw("Failed to unmarshal JSON", "error", err, "body", rawResp)
			return err
		}

		if len(resp.IllustManga.Data) == 0 {
			audit.GlobalAuditor.Logger.Debugw("No more results found", "suffix", suffix, "page", page)

			break
		}

		reachedLimit := processSearchResults(resp.IllustManga.Data, suffixArtworks)
		if reachedLimit {
			return nil
		}

		page++
		audit.GlobalAuditor.Logger.Debugw("Moving to next page", "suffix", suffix, "page", page)
	}
	return nil
}

// searchPopular performs a search for popular artworks using a descending popularity threshold strategy.
//
// The search strategy is as follows:
//
// 1. Starts by checking the highest popularity threshold (100000 users).
//   - If results are found at the highest threshold, it stores these results.
//   - Continues checking lower thresholds in descending order (e.g., 50000, 30000, etc.),
//     adding results until the resultsLimit is reached.
//
// 2. If no results are found at the highest threshold:
//
//   - Temporarily stores results from the lowest threshold (50 users).
//
//   - Then, checks all intermediate thresholds (50000 down to 100 users).
//
//   - Uses the temporarily stored lowest threshold results only if the resultsLimit
//     cannot be reached with results from higher thresholds.
//
//     3. Assembles the final list of results in descending order of popularity thresholds,
//     stopping when the total number of results reaches resultsLimit.
func searchPopular(ctx context.Context, r *http.Request, settings ArtworkSearchSettings) (ArtworkResults, error) {
	audit.GlobalAuditor.Logger.Debugw("Starting popular search", "query", settings.Name, "mode", settings.Category)

	artworksPerSuffix := make(map[string][]ArtworkBrief)
	var totalArtworks int

	// Start with highest suffix
	highestSuffix := descendingSuffixes[0]
	audit.GlobalAuditor.Logger.Debugw("Searching with highest suffix", "suffix", highestSuffix)
	var highestSuffixArtworks []ArtworkBrief
	err := fetchResultsForSuffix(ctx, r, settings, highestSuffix, &highestSuffixArtworks)
	if err != nil {
		audit.GlobalAuditor.Logger.Errorw("Error fetching results for highest suffix", "suffix", highestSuffix, "error", err)
		return ArtworkResults{}, err
	}

	if len(highestSuffixArtworks) > 0 {
		// Found results with highest suffix, continue with lower thresholds
		artworksPerSuffix[highestSuffix] = highestSuffixArtworks
		totalArtworks += len(highestSuffixArtworks)

		// Iterate over the remaining suffixes except the last one (lowest threshold)
		for i := 1; i < len(descendingSuffixes)-1; i++ {
			if totalArtworks >= resultsLimit {
				break
			}

			suffix := descendingSuffixes[i]
			var suffixArtworks []ArtworkBrief
			err := fetchResultsForSuffix(ctx, r, settings, suffix, &suffixArtworks)
			if err != nil {
				audit.GlobalAuditor.Logger.Errorw("Error fetching results for suffix", "suffix", suffix, "error", err)

				continue
			}
			artworksPerSuffix[suffix] = suffixArtworks
			totalArtworks += len(suffixArtworks)
		}
	} else {
		// No results with highest suffix, check all other thresholds before using lowest
		var lowestSuffixArtworks []ArtworkBrief
		lowestSuffix := descendingSuffixes[len(descendingSuffixes)-1]

		// Store lowest threshold results temporarily
		err := fetchResultsForSuffix(ctx, r, settings, lowestSuffix, &lowestSuffixArtworks)
		if err != nil {
			audit.GlobalAuditor.Logger.Errorw("Error fetching results for lowest suffix", "suffix", lowestSuffix, "error", err)
			return ArtworkResults{}, err
		}

		// If no results are found from the lowest threshold, exit early
		if len(lowestSuffixArtworks) == 0 {
			audit.GlobalAuditor.Logger.Debugw("No results found for the lowest suffix. Exiting early.", "suffix", lowestSuffix)
			return ArtworkResults{
				Data:  []ArtworkBrief{},
				Total: 0,
			}, nil
		}

		// Check intermediate thresholds (50000 down to 100)
		for i := 1; i < len(descendingSuffixes)-1; i++ {
			suffix := descendingSuffixes[i]
			var suffixArtworks []ArtworkBrief
			err := fetchResultsForSuffix(ctx, r, settings, suffix, &suffixArtworks)
			if err != nil {
				audit.GlobalAuditor.Logger.Errorw("Error fetching results for suffix", "suffix", suffix, "error", err)

				continue
			}
			artworksPerSuffix[suffix] = suffixArtworks
			totalArtworks += len(suffixArtworks)

			if totalArtworks >= resultsLimit {
				break
			}
		}

		// Only use lowest threshold results if we haven't reached resultsLimit
		if totalArtworks < resultsLimit && len(lowestSuffixArtworks) > 0 {
			artworksPerSuffix[lowestSuffix] = lowestSuffixArtworks
			totalArtworks += len(lowestSuffixArtworks)
		}
	}

	// Assemble final results in descending order of popularity
	var artworks []ArtworkBrief

	for _, suffix := range descendingSuffixes {
		arts := artworksPerSuffix[suffix]
		if len(artworks)+len(arts) >= resultsLimit {
			artworks = append(artworks, arts[:resultsLimit-len(artworks)]...)

			break
		}

		artworks = append(artworks, arts...)
	}

	total := len(artworks)
	audit.GlobalAuditor.Logger.Debugw("Completed popular search",
		"query", settings.Name,
		"mode", settings.Category,
		"resultsCount", total)

	// // Sort by ID in descending order
	// sort.Slice(artworks, func(i, j int) bool {
	// 	return artworks[i].ID > artworks[j].ID
	// })

	return ArtworkResults{
		Data:  artworks,
		Total: total,
	}, nil
}
