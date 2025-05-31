// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package routes

import (
	"fmt"
	"net/http"

	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/core"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"codeberg.org/pixivfe/pixivfe/server/template"
)

func NewestPage(w http.ResponseWriter, r *http.Request) error {
	worktype := GetQueryParam(r, "type", "illust")

	r18 := GetQueryParam(r, "r18", "false")

	works, err := core.GetNewestArtworks(r, worktype, r18)
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

	return template.RenderHTML(w, r, Data_newest{
		Items: works,
		Title: "Newest works",
	})
}
