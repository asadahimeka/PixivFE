// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package routes

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"codeberg.org/pixivfe/pixivfe/audit"
	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/core"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"codeberg.org/pixivfe/pixivfe/server/template"
	"go.uber.org/zap"
)

// userPageData is an intermediate struct
type userPageData struct {
	user     core.User
	category core.UserWorkCategory
	page     int
	queries  template.PartialURL
}

func UserPage(w http.ResponseWriter, r *http.Request) error {
	data, err := fetchData(r, true)
	if err != nil {
		return err
	}

	audit.GlobalAuditor.Logger.Debug("Category details",
		zap.String("category", data.category.Value),
		zap.Int("pageLimit", data.category.PageLimit),
	)

	if session.GetUserToken(r) != "" {
		w.Header().Set("Cache-Control", "private, max-age=60")
	} else {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, stale-while-revalidate=%d",
			int(config.GlobalConfig.HTTPCache.MaxAge.Seconds()),
			int(config.GlobalConfig.HTTPCache.StaleWhileRevalidate.Seconds())))
	}

	// FIXME: call these methods from within Jet, not here
	personalFields := data.user.PersonalFields()
	workspaceItems := data.user.WorkspaceItems()

	return template.RenderHTML(w, r, Data_user{
		Title:          data.user.Name,
		User:           data.user,
		Category:       data.category,
		Page:           data.page,
		Queries:        data.queries,
		PersonalFields: personalFields,
		WorkspaceItems: workspaceItems,
		MetaImage:      data.user.BackgroundImage,
	})
}

func UserAtomFeed(w http.ResponseWriter, r *http.Request) error {
	data, err := fetchData(r, false)
	if err != nil {
		return err
	}

	// TODO: need to handle following status
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, stale-while-revalidate=%d",
		int(config.GlobalConfig.HTTPCache.MaxAge.Seconds()),
		int(config.GlobalConfig.HTTPCache.StaleWhileRevalidate.Seconds())))

	return template.RenderWithContentType(w, r, "application/atom+xml", Data_userAtom{
		URL:      r.RequestURI,
		Title:    data.user.Name,
		User:     data.user,
		Category: data.category,
		Updated:  time.Now().Format(time.RFC3339),
		Page:     data.page,
	})
}

func fetchData(r *http.Request, getTags bool) (userPageData, error) {
	id := GetPathVar(r, "id")
	if _, err := strconv.Atoi(id); err != nil {
		return userPageData{}, err
	}

	categoryValue := GetPathVar(r, "category", core.CategoryAny.Value)
	category := core.NewUserWorkCategory(categoryValue)
	err := category.Validate()
	if err != nil {
		return userPageData{}, err
	}

	page := GetQueryParam(r, "page", "1")
	pageInt, err := strconv.Atoi(page)
	if err != nil {
		return userPageData{}, err
	}

	mode := GetQueryParam(r, "mode", "show")

	user, err := core.GetUserProfile(r, id, &category, pageInt, getTags, mode)
	if err != nil {
		return userPageData{}, err
	}

	// NOTE: these values are now calculated in core/user, scoped per category
	// var worksPerPage float64
	// if category.value == core.CategoryBookmarks.value {
	// 	worksPerPage = 48.0
	// } else {
	// 	worksPerPage = 30.0
	// }

	// worksCount := user.CategoryItemCount
	// pageLimit := int(math.Ceil(float64(worksCount) / worksPerPage))
	// category.SetPageLimit(pageLimit)

	urlc := template.PartialURL{
		Path:  r.URL.Path, // NOTE: unsure whether not hardcoding a path here will cause issues with PartialURL
		Query: map[string]string{"mode": mode},
	}

	return userPageData{
		user:     user,
		category: category,
		page:     pageInt,
		queries:  urlc,
	}, nil
}
