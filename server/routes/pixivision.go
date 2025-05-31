// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package routes

import (
	"fmt"
	"net/http"
	"strconv"

	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/core/pixivision"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"codeberg.org/pixivfe/pixivfe/server/template"
)

func PixivisionHomePage(w http.ResponseWriter, r *http.Request) error {
	page := GetQueryParam(r, "p", "1")
	data, err := pixivision.GetHomepage(r, page, "en")
	if err != nil {
		return err
	}

	pageint, err := strconv.Atoi(page)
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

	return template.RenderHTML(w, r, Data_pixivisionIndex{
		Data:  data,
		Page:  pageint,
		Title: "pixivision",
	})
}

func PixivisionArticlePage(w http.ResponseWriter, r *http.Request) error {
	id := GetPathVar(r, "id")
	lang := []string{"en"}

	data, doc, err := pixivision.ParseArticle(r, id, lang)
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

	if len(data.Items) == 0 {
		data, err := pixivision.ParseArticleFreeform(r, doc)
		if err != nil {
			return err
		}
	
		return template.RenderHTML(w, r, Data_pixivisionArticleFreeform{
			Children: data.Content,
			Title:    data.Title,
		})
	} else {
		return template.RenderHTML(w, r, Data_pixivisionArticle{
			Article: data,
			Title:   data.Title,
		})
	}
}

func PixivisionCategoryPage(w http.ResponseWriter, r *http.Request) error {
	page := GetQueryParam(r, "p", "1")
	id := GetPathVar(r, "id")
	data, err := pixivision.GetCategory(r, id, page, "en")
	if err != nil {
		return err
	}

	pageint, err := strconv.Atoi(page)
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

	return template.RenderHTML(w, r, Data_pixivisionCategory{
		Category: data,
		Page:     pageint,
		ID:       id,
		Title:    data.Title,
	})
}

func PixivisionTagPage(w http.ResponseWriter, r *http.Request) error {
	id := GetPathVar(r, "id")
	page := GetQueryParam(r, "p", "1")

	data, err := pixivision.GetTag(r, id, page, "en")
	if err != nil {
		return err
	}

	pageint, err := strconv.Atoi(page)
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

	return template.RenderHTML(w, r, Data_pixivisionTag{
		Tag:   data,
		Page:  pageint,
		ID:    id,
		Title: data.Title,
	})
}
