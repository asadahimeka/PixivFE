// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package routes

import (
	"fmt"
	"math"
	"net/http"
	"strconv"

	"golang.org/x/sync/errgroup"

	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/core"
	"codeberg.org/pixivfe/pixivfe/i18n"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"codeberg.org/pixivfe/pixivfe/server/template"
)

func MangaSeriesPage(w http.ResponseWriter, r *http.Request) error {
	user_id := GetPathVar(r, "user_id")
	if _, err := strconv.Atoi(user_id); err != nil {
		return i18n.Errorf("Invalid user ID: %s", user_id)
	}

	series_id := GetPathVar(r, "series_id")
	if _, err := strconv.Atoi(series_id); err != nil {
		return i18n.Errorf("Invalid series ID: %s", series_id)
	}

	// jackyzy823: No way to know total before the GetMangaSeriesByID request.
	pageStr := GetQueryParam(r, "page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		return i18n.Errorf("Invalid Page")
	}

	var mangaSeries core.MangaSeries
	var user core.User

	// Create errgroup to fetch MangaSeries and User data concurrently
	g := new(errgroup.Group)

	g.Go(func() error {
		var err error
		// jackyzy823: Must use token. because R-18 artwork could be included in an All-Age series.
		mangaSeries, err = core.GetMangaSeriesByID(r, series_id, page)
		return err
	})

	g.Go(func() error {
		var err error
		user, err = core.GetUserBasicInformation(r, user_id)
		return err
	})

	// Wait for both requests to complete
	if err := g.Wait(); err != nil {
		return err
	}

	// Handle redirect if user_id doesn't match
	//
	// jackyzy823: Pixiv auto redirects to correct uid when requesting the url, but we could only do following logic
	if user_id != mangaSeries.IllustSeries[0].UserID {
		redirectURL := fmt.Sprintf("/users/%s/series/%s", mangaSeries.IllustSeries[0].UserID, series_id)
		if page != 1 {
			redirectURL += fmt.Sprintf("?page=%d", page)
		}
		http.Redirect(w, r, redirectURL, http.StatusPermanentRedirect)
		return nil
	}

	// jackyzy823: Hard coded limit
	perPage := 12
	pageLimit := int(math.Ceil(float64(mangaSeries.IllustSeries[0].Total) / float64(perPage)))

	// jackyzy823: Pixiv display empty (not error page) if page id exceeds the total/12 +1
	if page > pageLimit {
		return i18n.Errorf("Invalid Page")
	}

	// Replace user data
	mangaSeries.Users[0] = user

	if session.GetUserToken(r) != "" {
		w.Header().Set("Cache-Control", "private, max-age=60")
	} else {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, stale-while-revalidate=%d",
			int(config.GlobalConfig.HTTPCache.MaxAge.Seconds()),
			int(config.GlobalConfig.HTTPCache.StaleWhileRevalidate.Seconds())))
	}

	return template.RenderHTML(w, r, Data_mangaSeries{
		MangaSeries: mangaSeries,
		Title:       mangaSeries.IllustSeries[0].Title,
		Page:        page,
		PageLimit:   pageLimit,
	})
}
