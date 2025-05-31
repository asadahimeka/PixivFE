// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
Search logic
*/
package routes

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/core"
	"codeberg.org/pixivfe/pixivfe/server/router/redirect"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"codeberg.org/pixivfe/pixivfe/server/template"
	"codeberg.org/pixivfe/pixivfe/server/utils"
	"golang.org/x/sync/errgroup"
)

func parsePixivURL(url string) (string, bool) {
	subpath, found := strings.CutPrefix(url, "https://pixiv.net/")
	if found {
		return redirect.RedirectFromPixivPath("/" + subpath), true
	}
	subpath, found = strings.CutPrefix(url, "https://www.pixiv.net/")
	if found {
		return redirect.RedirectFromPixivPath("/" + subpath), true
	}
	return "", false
}

func TagPage(w http.ResponseWriter, r *http.Request) error {
	// Extract search query from either path variable or query parameter
	pathParam := GetPathVar(r, "name")
	var searchQuery string

	if pathParam == "" {
		// No path variable, use query parameter instead
		searchQuery = GetQueryParam(r, "name")
	} else {
		// Path variable exists, needs to be URL-unescaped
		var err error
		searchQuery, err = url.PathUnescape(pathParam)
		if err != nil {
			return err
		}
	}

	subpath, ok := parsePixivURL(searchQuery)
	if ok {
		http.Redirect(w, r, subpath, http.StatusSeeOther)
		return nil
	}

	category := GetQueryParam(r, "category", "artworks")

	queries := core.ArtworkSearchSettings{
		Name:     searchQuery,
		Category: category,
		Order:    GetQueryParam(r, "order", "date_d"),
		Mode:     GetQueryParam(r, "mode", "safe"),
		Ratio:    GetQueryParam(r, "ratio", ""),
		Wlt:      GetQueryParam(r, "wlt", ""),
		Wgt:      GetQueryParam(r, "wgt", ""),
		Hlt:      GetQueryParam(r, "hlt", ""),
		Hgt:      GetQueryParam(r, "hgt", ""),
		Tool:     GetQueryParam(r, "tool", ""),
		Scd:      GetQueryParam(r, "scd", ""),
		Ecd:      GetQueryParam(r, "ecd", ""),
		Page:     GetQueryParam(r, "page", "1"),
	}

	switch category {
	case "users":
		return resultPageUser(w, r, queries)
	case "novels":
		return resultPageArtNovel(w, r, queries)
	default:
		return resultPageArtNovel(w, r, queries)
	}
}

func resultPageUser(w http.ResponseWriter, r *http.Request, queries core.ArtworkSearchSettings) error {
	name := queries.Name
	pageInt, err := strconv.Atoi(queries.Page)
	if err != nil {
		return err
	}

	result, err := core.GetUserSearch(r, queries)
	if err != nil {
		return err
	}

	setCacheControl(r, w)

	urlc := template.PartialURL{
		Path:  "/tags",
		Query: queries.ReturnMap(),
	}

	return template.RenderHTML(w, r, Data_tag{
		SearchQuery:    name,
		Title:          "Results for " + name,
		Data:           *result,
		Page:           pageInt,
		QueriesC:       urlc,
		ActiveCategory: "users",
	})
}

// artworks, illustrations, manga, novels
func resultPageArtNovel(w http.ResponseWriter, r *http.Request, queries core.ArtworkSearchSettings) error {
	name := queries.Name
	pageInt, err := strconv.Atoi(queries.Page)
	if err != nil {
		return err
	}

	var (
		tag    core.TagSearchResult
		result *core.ArtworkSearchResponse
	)

	g, _ := errgroup.WithContext(r.Context())

	// Fetch tag data and search results concurrently
	g.Go(func() error {
		t, err := core.GetTagData(r, name)
		if err != nil {
			return err
		}
		tag = t

		// Fetch cover artwork after tag data is available
		id := tag.Metadata.ID.String()
		if id != "" {
			var illust core.Illust

			if err := core.GetBasicArtwork(r, id, &illust); err != nil {
				return err
			}

			tag.CoverArtwork = illust
		}

		// PreloadImage(w, tag.Metadata.ImageMaster)
		PreloadImage(w, tag.CoverArtwork.Thumbnails.Webp_1200)

		if config.GlobalConfig.Response.EarlyHintsResponsesEnabled {
			w.WriteHeader(http.StatusEarlyHints)
		}

		return nil
	})

	g.Go(func() error {
		res, err := core.GetSearch(r, queries)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err := g.Wait(); err != nil {
		return err
	}

	urlc := template.PartialURL{
		Path:  "/tags",
		Query: queries.ReturnMap(),
	}

	setCacheControl(r, w)

	return template.RenderHTML(w, r, Data_tag{
		SearchQuery:          name,
		Title:                "Results for " + name,
		Tag:                  tag,
		Data:                 *result,
		QueriesC:             urlc,
		Page:                 pageInt,
		ActiveCategory:       queries.Category,
		ActiveOrder:          queries.Order,
		ActiveMode:           queries.Mode,
		ActiveRatio:          queries.Ratio,
		ActiveSearchMode:     GetQueryParam(r, "smode", ""),
		PopularSearchEnabled: config.GlobalConfig.Feature.PopularSearchEnabled,
	})
}

func setCacheControl(r *http.Request, w http.ResponseWriter) {
	if session.GetUserToken(r) != "" {
		w.Header().Set("Cache-Control", "private, max-age=60")
	} else {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, stale-while-revalidate=%d",
			int(config.GlobalConfig.HTTPCache.MaxAge.Seconds()),
			int(config.GlobalConfig.HTTPCache.StaleWhileRevalidate.Seconds())))
	}
}

func AdvancedTagPost(w http.ResponseWriter, r *http.Request) error {
	m := map[string]string{}
	core.SetIfNotEmpty(m, "name", GetQueryParam(r, "name", r.FormValue("name")))
	core.SetIfNotEmpty(m, "category", GetQueryParam(r, "category", "artworks"))
	core.SetIfNotEmpty(m, "order", GetQueryParam(r, "order", "date_d"))
	core.SetIfNotEmpty(m, "mode", GetQueryParam(r, "mode", "safe"))
	core.SetIfNotEmpty(m, "ratio", GetQueryParam(r, "ratio"))
	core.SetIfNotEmpty(m, "page", GetQueryParam(r, "page", "1"))
	core.SetIfNotEmpty(m, "wlt", GetQueryParam(r, "wlt", r.FormValue("wlt")))
	core.SetIfNotEmpty(m, "wgt", GetQueryParam(r, "wgt", r.FormValue("wgt")))
	core.SetIfNotEmpty(m, "hlt", GetQueryParam(r, "hlt", r.FormValue("hlt")))
	core.SetIfNotEmpty(m, "hgt", GetQueryParam(r, "hgt", r.FormValue("hgt")))
	core.SetIfNotEmpty(m, "tool", GetQueryParam(r, "tool", r.FormValue("tool")))
	core.SetIfNotEmpty(m, "scd", GetQueryParam(r, "scd", r.FormValue("scd")))
	core.SetIfNotEmpty(m, "ecd", GetQueryParam(r, "ecd", r.FormValue("ecd")))
	return utils.RedirectTo(w, r, "/tags", m)
}
