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

func IndexPage(w http.ResponseWriter, r *http.Request) error {
	mode := GetQueryParam(r, "mode", "all")
	isLoggedIn := session.GetUserToken(r) != ""

	works, err := core.GetLanding(r, mode, isLoggedIn)
	if err != nil {
		return err
	}

	urlc := template.PartialURL{
		Path:  "/",
		Query: map[string]string{"mode": mode},
	}

	if session.GetUserToken(r) != "" {
		w.Header().Set("Cache-Control", "private, max-age=60")
	} else {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, stale-while-revalidate=%d",
			int(config.GlobalConfig.HTTPCache.MaxAge.Seconds()),
			int(config.GlobalConfig.HTTPCache.StaleWhileRevalidate.Seconds())))
	}

	return template.RenderHTML(w, r, Data_index{
		Title:    "Landing",
		Data:     *works,
		LoggedIn: isLoggedIn,
		Queries:  urlc,
	})
}
