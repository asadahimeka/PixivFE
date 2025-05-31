// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package routes

import (
	"net/http"
	"strconv"
	"strings"

	"codeberg.org/pixivfe/pixivfe/core"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"codeberg.org/pixivfe/pixivfe/server/template"
)

func SelfUserPage(w http.ResponseWriter, r *http.Request) error {
	noAuthReturnPath := GetQueryParam(r, "noAuthReturnPath")
	loginReturnPath := "/self"
	token := session.GetUserToken(r)

	if token == "" {
		return UnauthorizedPage(w, r, noAuthReturnPath, loginReturnPath)
	}

	// The left part of the token is the member ID
	userID := strings.Split(token, "_")

	http.Redirect(w, r, "/users/"+userID[0], http.StatusSeeOther)
	return nil
}

func SelfBookmarksPage(w http.ResponseWriter, r *http.Request) error {
	noAuthReturnPath := GetQueryParam(r, "noAuthReturnPath")
	loginReturnPath := "/self/bookmarks"
	token := session.GetUserToken(r)

	if token == "" {
		return UnauthorizedPage(w, r, noAuthReturnPath, loginReturnPath)
	}

	// The left part of the token is the member ID
	userID := strings.Split(token, "_")

	http.Redirect(w, r, "/users/"+userID[0]+"/bookmarks", http.StatusSeeOther)
	return nil
}

func SelfFollowingUsersPage(w http.ResponseWriter, r *http.Request) error {
	noAuthReturnPath := GetQueryParam(r, "noAuthReturnPath")
	loginReturnPath := "/self/followingUsers"
	token := session.GetUserToken(r)

	if token == "" {
		return UnauthorizedPage(w, r, noAuthReturnPath, loginReturnPath)
	}

	// The left part of the token is the member ID
	userID := strings.Split(token, "_")

	http.Redirect(w, r, "/users/"+userID[0]+"/following", http.StatusSeeOther)
	return nil
}

func SelfFollowingWorksPage(w http.ResponseWriter, r *http.Request) error {
	noAuthReturnPath := GetQueryParam(r, "noAuthReturnPath")
	loginReturnPath := "/self/followingWorks"
	token := session.GetUserToken(r)

	if token == "" {
		return UnauthorizedPage(w, r, noAuthReturnPath, loginReturnPath)
	}

	mode := GetQueryParam(r, "mode", "safe")
	page := GetQueryParam(r, "page", "1")

	pageInt, err := strconv.Atoi(page)
	if err != nil {
		return err
	}

	data, err := core.GetNewestFromFollowing(r, "illust", mode, page)
	if err != nil {
		return err
	}

	return template.RenderHTML(w, r, Data_following{
		Title: "Latest by followed users",
		Mode:  mode,
		Data:  data,
		Page:  pageInt,
	})
}
