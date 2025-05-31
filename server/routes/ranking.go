// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package routes

import (
	"fmt"
	"net/http"
	"strconv"

	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/core"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"codeberg.org/pixivfe/pixivfe/server/template"
)

func RankingPage(w http.ResponseWriter, r *http.Request) error {
	mode := GetQueryParam(r, "mode", "daily")
	content := GetQueryParam(r, "content", "all")
	date := GetQueryParam(r, "date", "")

	page := GetQueryParam(r, "page", "1")
	pageInt, err := strconv.Atoi(page)
	if err != nil {
		return err
	}

	works, err := core.GetRanking(r, mode, content, date, page)
	if err != nil {
		return err
	}

	if session.GetUserToken(r) != "" {
		w.Header().Set("Cache-Control", "private, max-age=60")
	} else {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, stale-while-revalidate=%d",
			int(config.GlobalConfig.HTTPCache.MaxAge.Seconds()),
			int(config.GlobalConfig.HTTPCache.StaleWhileRevalidate.Seconds())))
	}

	return template.RenderHTML(w, r, Data_rank{
		Title:     "Ranking",
		Page:      pageInt,
		PageLimit: 28,
		Date:      date,
		Data:      works,
	})
}
