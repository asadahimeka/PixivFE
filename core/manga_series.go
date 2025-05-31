// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"net/http"
	"strconv"
	"time"

	"codeberg.org/pixivfe/pixivfe/core/requests"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"github.com/goccy/go-json"
)

// MangaSeries represents a manga series page, which can contain one or more IllustSeries
type MangaSeries struct {
	TagTranslation TagTranslationWrapper `json:"tagTranslation"`
	Thumbnails     struct {
		Illust []ArtworkBrief `json:"illust"`
	} `json:"thumbnails"`
	IllustSeries []IllustSeries `json:"illustSeries"`
	Users        []User         `json:"users"`
	Page         struct {
		Series               []SeriesEntry `json:"series"`
		IsSetCover           bool          `json:"isSetCover"`
		SeriesID             int           `json:"seriesId"`
		OtherSeriesID        string        `json:"otherSeriesId"`
		RecentUpdatedWorkIds []int         `json:"recentUpdatedWorkIds"`
		Total                int           `json:"total"`
		IsWatched            bool          `json:"isWatched"`
		IsNotifying          bool          `json:"isNotifying"`
	} `json:"page"`
	Tags []Tag
}

// IllustSeries represents a specific manga series with its associated artworks.
//
// Analogous to ArtworkBrief.
type IllustSeries struct {
	ID             string         `json:"id"`
	UserID         string         `json:"userId"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	Caption        string         `json:"caption"`
	Total          int            `json:"total"`
	ContentOrder   any            `json:"content_order"`
	Thumbnail      string         `json:"url"`
	CoverImageSl   int            `json:"coverImageSl"`
	FirstIllustID  string         `json:"firstIllustId"`
	LatestIllustID string         `json:"latestIllustId"`
	CreateDate     time.Time      `json:"createDate"`
	UpdateDate     time.Time      `json:"updateDate"`
	WatchCount     any            `json:"watchCount"`
	IsWatched      bool           `json:"isWatched"`
	IsNotifying    bool           `json:"isNotifying"`
	List           []ArtworkBrief // Artworks specific to this series
}

// SeriesEntry represents an entry in the series page with ordering.
type SeriesEntry struct {
	WorkID string `json:"workId"`
	Order  int    `json:"order"`
}

// GetMangaSeriesByID retrieves the content of a manga series by its ID and page number.
func GetMangaSeriesByID(r *http.Request, id string, page int) (MangaSeries, error) {
	var mangaSeries MangaSeries

	url := GetMangaSeriesContentURL(id, page)

	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	rawResp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
	if err != nil {
		return mangaSeries, err
	}

	rawResp = RewriteContentURLs(r, rawResp)

	err = json.Unmarshal(rawResp, &mangaSeries)
	if err != nil {
		return mangaSeries, err
	}

	// Group artworks into their respective series
	illustSeriesMap := make(map[string]*IllustSeries)

	for i := range mangaSeries.IllustSeries {
		series := &mangaSeries.IllustSeries[i]
		series.List = make([]ArtworkBrief, 0)
		illustSeriesMap[series.ID] = series
	}

	for i := range mangaSeries.Thumbnails.Illust {
		// Populate thumbnails for each artwork
		if err := mangaSeries.Thumbnails.Illust[i].PopulateThumbnails(); err != nil {
			return mangaSeries, err
		}

		if series, exists := illustSeriesMap[mangaSeries.Thumbnails.Illust[i].SeriesID]; exists {
			series.List = append(series.List, mangaSeries.Thumbnails.Illust[i])
		}
	}

	// Order the main series according to Page.Series entries
	mainSeriesID := strconv.Itoa(mangaSeries.Page.SeriesID)
	if mainSeries, exists := illustSeriesMap[mainSeriesID]; exists {
		artworkMap := make(map[string]ArtworkBrief, len(mainSeries.List))
		for _, artwork := range mainSeries.List {
			artworkMap[artwork.ID] = artwork
		}

		orderedList := make([]ArtworkBrief, 0, len(mangaSeries.Page.Series))

		for _, entry := range mangaSeries.Page.Series {
			if artwork, ok := artworkMap[entry.WorkID]; ok {
				orderedList = append(orderedList, artwork)
			}
		}
		mainSeries.List = orderedList
	}

	// Collect all tags from all series artworks
	allTags := make([]string, 0)

	for _, series := range mangaSeries.IllustSeries {
		for _, artwork := range series.List {
			allTags = append(allTags, artwork.Tags...)
		}
	}

	// Convert tag translations to Tag objects
	mangaSeries.Tags = TagTranslationsToTags(allTags, mangaSeries.TagTranslation)

	return mangaSeries, nil
}
