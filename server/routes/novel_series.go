// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package routes

import (
	"fmt"
	"math"
	"net/http"
	"strconv"

	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/core"
	"codeberg.org/pixivfe/pixivfe/i18n"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"codeberg.org/pixivfe/pixivfe/server/template"
)

func NovelSeriesPage(w http.ResponseWriter, r *http.Request) error {
	id := GetPathVar(r, "id")
	if _, err := strconv.Atoi(id); err != nil {
		return i18n.Errorf("Invalid ID: %s", id)
	}

	series, err := core.GetNovelSeriesByID(r, id)
	if err != nil {
		return err
	}

	// Hard coded limit
	perPage := 30
	pageLimit := int(math.Ceil(float64(series.Total) / float64(perPage)))

	page := GetQueryParam(r, "p", "1")
	pageNum, err := strconv.Atoi(page)
	if err != nil || pageNum < 1 || pageNum > pageLimit {
		return i18n.Errorf("Invalid Page Number: %d", pageNum)
	}

	// TODO should use token only if R-18/R-18G
	seriesContents, err := core.GetNovelSeriesContentByID(r, id, pageNum, perPage)
	if err != nil {
		return err
	}

	user, err := core.GetUserBasicInformation(r, series.UserID)
	if err != nil {
		return err
	}

	title := fmt.Sprintf("%s | %s", series.Title, series.UserName)

	if session.GetUserToken(r) != "" {
		w.Header().Set("Cache-Control", "private, max-age=60")
	} else {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, stale-while-revalidate=%d",
			int(config.GlobalConfig.HTTPCache.MaxAge.Seconds()),
			int(config.GlobalConfig.HTTPCache.StaleWhileRevalidate.Seconds())))
	}

	return template.RenderHTML(w, r, Data_novelSeries{
		NovelSeries:         series,
		NovelSeriesContents: seriesContents,
		Title:               title,
		User:                user,
		Page:                pageNum,
		PageLimit:           pageLimit,
	})
}
