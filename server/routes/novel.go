// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package routes

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/core"
	"codeberg.org/pixivfe/pixivfe/i18n"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"codeberg.org/pixivfe/pixivfe/server/template"
	"codeberg.org/pixivfe/pixivfe/server/utils"
)

func NovelPage(w http.ResponseWriter, r *http.Request) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		w.Header().Add("Server-Timing", fmt.Sprintf("total;dur=%.0f;desc=\"Total Time\"", float64(duration.Milliseconds())))
	}()

	timings := utils.NewTimings()

	id := GetPathVar(r, "id")
	if _, err := strconv.Atoi(id); err != nil {
		return i18n.Errorf("Invalid ID: %s", id)
	}

	// Fetch main novel data
	novelFetchStart := time.Now()
	novel, err := core.GetNovelByID(r, id)
	if err != nil {
		return err
	}
	timings.Append(
		"novel-fetch",
		time.Since(novelFetchStart),
		"Basic novel data fetch",
	)

	// Fetch related novels
	relatedFetchStart := time.Now()
	related, err := core.GetNovelRelated(r, id)
	if err != nil {
		return err
	}
	timings.Append(
		"novel-related-fetch",
		time.Since(relatedFetchStart),
		"Related novels fetch",
	)

	var contentTitles []core.NovelSeriesContentTitle
	if novel.SeriesNavData.SeriesID != 0 {
		// Fetch series content titles
		seriesFetchStart := time.Now()
		// Must use token, because we can't determine Series' XRestrict via Novel API here
		// and All-age post could also appears in R-18 series.
		contentTitles, err = core.GetNovelSeriesContentTitlesByID(r, novel.SeriesNavData.SeriesID)
		if err != nil {
			return err
		}

		timings.Append(
			"novel-series-data-fetch",
			time.Since(seriesFetchStart),
			"Novel series data fetch",
		)
	}

	// Fetch comments if they are not disabled
	if novel.CommentOff != 1 {
		params := core.NovelCommentsParams{
			ID:        id,
			UserID:    novel.UserID,
			XRestrict: novel.XRestrict,
		}

		comments, commentTimings, err := core.GetNovelComments(r, params)
		if err != nil {
			return err
		}
		novel.CommentsData = comments

		// Add each comment timing to the main timings
		for _, timing := range commentTimings {
			timings.Append(timing.Name, timing.Duration, timing.Description)
		}
	}

	// Fetch user information
	userFetchStart := time.Now()
	user, err := core.GetUserBasicInformation(r, novel.UserID)
	if err != nil {
		return err
	}
	timings.Append(
		"user-data-fetch",
		time.Since(userFetchStart),
		"Basic user data fetch",
	)

	fontType := session.GetCookie(r, session.Cookie_NovelFontType)
	if fontType == "" {
		fontType = "gothic"
	}
	viewMode := session.GetCookie(r, session.Cookie_NovelViewMode)
	if viewMode == "" {
		viewMode = strconv.Itoa(novel.Settings.ViewMode)
	}

	title := novel.Title
	if novel.SeriesNavData.SeriesID != 0 {
		title = fmt.Sprintf("#%d %s | %s", novel.SeriesNavData.Order, novel.Title, novel.SeriesNavData.Title)
	}

	novelSeriesIDs := make([]string, len(contentTitles))
	novelSeriesTitles := make([]string, len(contentTitles))
	for i, ct := range contentTitles {
		novelSeriesIDs[i] = ct.ID
		novelSeriesTitles[i] = fmt.Sprintf("#%d %s", i+1, ct.Title)
	}

	if session.GetUserToken(r) != "" {
		w.Header().Set("Cache-Control", "private, max-age=60")
	} else {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, stale-while-revalidate=%d",
			int(config.GlobalConfig.HTTPCache.MaxAge.Seconds()),
			int(config.GlobalConfig.HTTPCache.StaleWhileRevalidate.Seconds())))
	}

	timings.WriteHeaders(w)

	return template.RenderHTML(w, r, Data_novel{
		Novel:                    novel,
		NovelRelated:             related,
		User:                     user,
		NovelSeriesContentTitles: contentTitles,
		NovelSeriesIDs:           novelSeriesIDs,
		NovelSeriesTitles:        novelSeriesTitles,
		Title:                    title,
		FontType:                 fontType,
		ViewMode:                 viewMode,
		Language:                 strings.ToLower(novel.Language),
	})
}
