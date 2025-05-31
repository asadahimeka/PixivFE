// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package routes

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"codeberg.org/pixivfe/pixivfe/audit"
	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/core"
	"codeberg.org/pixivfe/pixivfe/i18n"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"codeberg.org/pixivfe/pixivfe/server/template"
)

func ArtworkPage(w http.ResponseWriter, r *http.Request) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		w.Header().Add("Server-Timing", fmt.Sprintf("total;dur=%.0f;desc=\"Total Time\"", float64(duration.Milliseconds())))
	}()

	id := GetPathVar(r, "id")
	if _, err := strconv.Atoi(id); err != nil {
		return i18n.Errorf("Invalid ID: %s", id)
	}

	isHtmx := r.Header.Get("HX-Request") == "true"
	isFastRequest := r.Header.Get("Fast-Request") == "true"

	// For HTMX requests with Fast-Request, route to fast path render
	//
	// Explicitly opt-in to skip requests without the required Artwork-* headers
	if isHtmx && isFastRequest {
		return ArtworkPageFast(w, r)
	}

	illust, err := core.GetArtwork(w, r, id)
	if err != nil {
		return err
	}

	tagNames := make([]string, 0)

	for _, i := range illust.Tags.Tags {
		tagNames = append(tagNames, i.Name)
	}
	metaDescription := strings.Join(tagNames, ", ")

	// Preload assets and write HTTP 103 since we're waiting on an API response
	preloadArtworkAssets(w, r, *illust)

	if config.GlobalConfig.Response.EarlyHintsResponsesEnabled {
		w.WriteHeader(http.StatusEarlyHints)
	}

	if session.GetUserToken(r) != "" {
		w.Header().Set("Cache-Control", "private, max-age=60")
	} else {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, stale-while-revalidate=%d",
			int(config.GlobalConfig.HTTPCache.MaxAge.Seconds()),
			int(config.GlobalConfig.HTTPCache.StaleWhileRevalidate.Seconds())))
	}

	return template.RenderHTML(w, r, Data_artwork{
		Illust:          *illust,
		Title:           illust.Title,
		MetaDescription: metaDescription,
		MetaImage:       illust.Images[0].Original,
		MetaImageWidth:  illust.Images[0].Width,
		MetaImageHeight: illust.Images[0].Height,
		MetaAuthor:      illust.UserName,
		MetaAuthorID:    illust.UserID,
	})
}

// ArtworkPageFast handles a fast path artwork page render.
func ArtworkPageFast(w http.ResponseWriter, r *http.Request) error {
	// Load all header data first
	artworkID := r.Header.Get("Artwork-ID")
	userID := r.Header.Get("Artwork-User-ID")

	title, err := url.QueryUnescape(r.Header.Get("Artwork-Title"))
	if err != nil {
		return fmt.Errorf("failed to decode title: %w", err)
	}

	// Image dimension headers
	pages, err := strconv.Atoi(r.Header.Get("Artwork-Pages"))
	if err != nil {
		return fmt.Errorf("failed to parse Artwork-Pages header: %w", err)
	}

	width, err := strconv.Atoi(r.Header.Get("Artwork-Width"))
	if err != nil {
		return fmt.Errorf("failed to parse Artwork-Width header: %w", err)
	}

	height, err := strconv.Atoi(r.Header.Get("Artwork-Height"))
	if err != nil {
		return fmt.Errorf("failed to parse Artwork-Height header: %w", err)
	}

	illustType, err := strconv.Atoi(r.Header.Get("Artwork-IllustType"))
	if err != nil {
		return fmt.Errorf("failed to parse Artwork-IllustType header: %w", err)
	}

	// Image URL headers
	webpURL := r.Header.Get("Artwork-Master-Webp-1200-Url")
	originalURL := r.Header.Get("Artwork-Original-Url")

	// Initialize illust structure
	var illust core.Illust
	illust.ID = artworkID
	illust.UserID = userID
	illust.Title = title
	illust.Pages = pages
	illust.Width = width
	illust.Height = height
	illust.IllustType = core.IllustType(illustType)

	// Initialize images slice and populate thumbnails
	illust.Images = make([]core.Thumbnails, illust.Pages)

	// Populate fields for Images slice items
	for pageNum := range illust.Pages {
		// For first page, use the provided URL directly
		pageWebpURL := webpURL

		// For subsequent pages, modify the URLs to reflect the correct page number
		if pageNum > 0 {
			// Replace _p0_ with _p{pageNum}_ in the URLs
			pageWebpURL = strings.Replace(webpURL, "_p0_", fmt.Sprintf("_p%d_", pageNum), 1)
		}

		// Generate thumbnails for this page
		thumbnails, err := core.PopulateThumbnailsFor(pageWebpURL)
		if err != nil {
			return fmt.Errorf("failed to generate thumbnails for image on page %d: %w", pageNum, err)
		}

		// Assign the generated thumbnails
		illust.Images[pageNum] = thumbnails

		// Set additional fields for this page
		illust.Images[pageNum].MasterWebp_1200 = pageWebpURL
		illust.Images[pageNum].Original = "" // originalURL not reliable, leave empty

		// Set dimensions and illustration type
		if pageNum == 0 {
			// Use actual dimensions for the first page
			illust.Images[pageNum].Width = illust.Width
			illust.Images[pageNum].Height = illust.Height
		} else {
			// Set dummy dimensions for subsequent pages
			// This reduces layout shift if the user opens the lightbox
			// before the htmx request to ArtworkPartial completes
			illust.Images[pageNum].Width = 1000
			illust.Images[pageNum].Height = 1000
		}

		// Same IllustType for all pages
		illust.Images[pageNum].IllustType = illust.IllustType
	}

	// We skip validating the image URLs provided in request headers since:
	// 1. These URLs are only used for Link preload headers
	// 2. The actual URLs used by the frontend come from GetArtworkByIDFast
	//
	// The priority here is sending HTTP 200 with Link headers as quickly as possible.
	// Adding URL validation would create an extra network request in the critical path,
	// slowing down the initial page render.

	// Store original URL before any extension swapping
	originalExt := path.Ext(originalURL)

	if originalExt == ".jpg" {
		originalURL = strings.TrimSuffix(originalURL, ".jpg") + ".png"
		audit.GlobalAuditor.Logger.Infow("Swapped extension from jpg to png",
			"newUrl", originalURL)
	} else if originalExt == ".png" {
		originalURL = strings.TrimSuffix(originalURL, ".png") + ".jpg"
		audit.GlobalAuditor.Logger.Infow("Swapped extension from png to jpg",
			"newUrl", originalURL)
	}

	// Preload assets, but don't write HTTP 103 since we're not waiting on an API response
	preloadArtworkAssets(w, r, illust)

	// Set cache control headers
	if session.GetUserToken(r) != "" {
		w.Header().Set("Cache-Control", "private, max-age=60")
	} else {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, stale-while-revalidate=%d",
			int(config.GlobalConfig.HTTPCache.MaxAge.Seconds()),
			int(config.GlobalConfig.HTTPCache.StaleWhileRevalidate.Seconds())))
	}

	return template.RenderHTML(w, r, Data_artworkFast{
		Illust: illust,
		Title:  illust.Title,
	})
}
