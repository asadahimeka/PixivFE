// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package routes

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/goccy/go-json"

	"codeberg.org/pixivfe/pixivfe/audit"
	"codeberg.org/pixivfe/pixivfe/core"
	"codeberg.org/pixivfe/pixivfe/core/requests"
	"codeberg.org/pixivfe/pixivfe/i18n"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"codeberg.org/pixivfe/pixivfe/server/template"
	"codeberg.org/pixivfe/pixivfe/server/utils"
)

const (
	returnPathFormat    string = "return_path"
	bookmarkCountFormat string = "bookmark_count"
	likeCountFormat     string = "like_count"
	artworkIDFormat     string = "artwork_id"
	bookmarkIDFormat    string = "bookmark_id"
	userIDFormat        string = "user_id"
	privateFormat       string = "private" // privacy setting for bookmarks and follows
)

/*
When handling htmx requests, we pass all bookmark-related illust data
to the opposite partial (add ↔ delete) to ensure state preservation
during client-side swaps.
*/

// AddBookmarkRoute handles both full‑page and HTMX fragment requests for adding
// a bookmark to an illustration.
//
// Quick‑action (thumbnail) requests are always rendered as HTMX fragments.
func AddBookmarkRoute(w http.ResponseWriter, r *http.Request) error {
	// Extract session & CSRF: if missing, show the unauthorized page.
	sessionID := session.GetUserToken(r)
	csrfToken := session.GetCookie(r, session.Cookie_CSRF)
	returnPath := r.FormValue(returnPathFormat)
	if sessionID == "" || csrfToken == "" {
		return UnauthorizedPage(w, r, returnPath, returnPath)
	}

	// Figure out if we're in a "quick" thumbnail context or a full‑button context.
	isQuick := r.Header.Get("Quick-Action") == "true"
	isHtmx := isQuick || r.Header.Get("HX-Request") == "true"

	// Parse current bookmark count for the non‑quick (full) button.
	bookmarkCount := 0
	if !isQuick {
		var err error
		bookmarkCount, err = strconv.Atoi(r.FormValue(bookmarkCountFormat))
		if err != nil {
			bookmarkCount = 0
			// FIXME: what is this?
			// return i18n.Error("Invalid bookmark count.")
		}
	}

	// Get the illustration ID from the URL.
	illustID := GetPathVar(r, artworkIDFormat)
	if illustID == "" {
		return i18n.Error("No illustration ID provided.")
	}

	// Build and send the pixiv API request to add the bookmark.
	// The `restrict` flag controls private vs public.
	// We check for "on" since this is what an <input type="checkbox"> provides.
	restrict := "0"
	if r.FormValue(privateFormat) == "on" {
		restrict = "1"
	}
	payload := fmt.Sprintf(
		`{"illust_id":"%s","restrict":%s,"comment":"","tags":[]}`,
		illustID, restrict,
	)
	apiURL := core.PostAddIllustBookmarkURL()
	resp, err := requests.PerformPOST(
		r.Context(),
		apiURL,
		payload,
		map[string]string{"PHPSESSID": sessionID},
		csrfToken,
		"application/json; charset=utf-8",
		r.Header,
	)
	if err != nil {
		return err
	}

	// Parse the response body to get the new bookmark ID.
	var addResp core.AddIllustBookmarkResponse
	if err := json.Unmarshal(resp.Body, &addResp); err != nil {
		return err
	}
	newBookmarkID := addResp.Body.LastBookmarkID

	// Invalidate the relevant cache entries:
	// 	- the user's bookmark-list ajax endpoint
	// 	- the illustration's ajax endpoint
	userID := strings.Split(sessionID, "_")[0]
	urls := []string{
		fmt.Sprintf("https://www.pixiv.net/ajax/user/%s/illusts/bookmarks", userID),
		fmt.Sprintf("https://www.pixiv.net/ajax/illust/%s", illustID),
	}
	cnt, invalidated := requests.InvalidateURLs(urls)
	audit.GlobalAuditor.Logger.Infow(
		"Invalidated cache entries",
		"userID", userID,
		"count", cnt,
		"requested", urls,
		"invalidated", invalidated,
	)

	// If this is HTMX (including all quick actions), render only the
	// appropriate partial. Otherwise fall back to a redirect.
	if isHtmx {
		if isQuick {
			// Quick thumbnail swap → render the small "delete" icon only.
			return template.RenderHTML(w, r, Data_quickDeleteBookmarkPartial{
				ID:           illustID,
				BookmarkData: &core.BookmarkData{ID: newBookmarkID},
			})
		}
		// Full‑button swap → render the larger button with updated count.
		return template.RenderHTML(w, r, Data_deleteBookmarkPartial{
			Illust: core.Illust{
				ID:           illustID,
				BookmarkData: &core.BookmarkData{ID: newBookmarkID},
				Bookmarks:    bookmarkCount + 1,
			},
		})
	}

	// Non‑HTMX fallback: redirect back to wherever we came from.
	utils.RedirectToWhenceYouCame(w, r, returnPath)

	return nil
}

// AddBookmarkRoute handles both full‑page and HTMX fragment requests for removing
// a bookmark from an illustration.
//
// Quick‑action (thumbnail) requests are always rendered as HTMX fragments.
func DeleteBookmarkRoute(w http.ResponseWriter, r *http.Request) error {
	// Extract session & CSRF: if missing, show the unauthorized page.
	sessionID := session.GetUserToken(r)
	csrfToken := session.GetCookie(r, session.Cookie_CSRF)
	returnPath := r.FormValue(returnPathFormat)
	if sessionID == "" || csrfToken == "" {
		return UnauthorizedPage(w, r, returnPath, returnPath)
	}

	// Figure out if we're in a "quick" thumbnail context or a full‑button context.
	isQuick := r.Header.Get("Quick-Action") == "true"
	isHtmx := isQuick || r.Header.Get("HX-Request") == "true"

	// Parse current bookmark count for the non‑quick (full) button.
	bookmarkCount := 0
	if !isQuick {
		var err error
		bookmarkCount, err = strconv.Atoi(r.FormValue(bookmarkCountFormat))
		if err != nil {
			return i18n.Error("Invalid bookmark count.")
		}
	}

	// Get the bookmark ID from the URL.
	bookmarkID := GetPathVar(r, bookmarkIDFormat)
	if bookmarkID == "" {
		return i18n.Error("No bookmark ID provided.")
	}

	// Build and send the pixiv API request to remove the bookmark.
	payload := fmt.Sprintf("bookmark_id=%s", bookmarkID)
	apiURL := core.PostDeleteIllustBookmarkURL()
	if _, err := requests.PerformPOST(
		r.Context(),
		apiURL,
		payload,
		map[string]string{"PHPSESSID": sessionID},
		csrfToken,
		"application/x-www-form-urlencoded; charset=utf-8",
		r.Header,
	); err != nil {
		return err
	}

	// We need the illustration ID back in the form so we can re‑render
	// the button (and also to invalidate its cache entry).
	illustID := r.FormValue(artworkIDFormat)

	// Invalidate the relevant cache entries:
	// 	- the user's bookmark-list ajax endpoint
	// 	- the illustration's ajax endpoint
	userID := strings.Split(sessionID, "_")[0]
	urls := []string{
		fmt.Sprintf("https://www.pixiv.net/ajax/user/%s/illusts/bookmarks", userID),
		fmt.Sprintf("https://www.pixiv.net/ajax/illust/%s", illustID),
	}
	cnt, invalidated := requests.InvalidateURLs(urls)
	audit.GlobalAuditor.Logger.Infow(
		"Invalidated cache entries",
		"userID", userID,
		"count", cnt,
		"requested", urls,
		"invalidated", invalidated,
	)

	// If this is HTMX (including all quick actions), render only the
	// appropriate partial. Otherwise fall back to a redirect.
	if isHtmx {
		if isQuick {
			// Quick thumbnail swap → render the small "add" icon only.
			return template.RenderHTML(w, r, Data_quickAddBookmarkPartial{
				ID: illustID,
			})
		}
		// Full‑button swap → render the larger "add" button with updated count.
		return template.RenderHTML(w, r, Data_addBookmarkPartial{
			Illust: core.Illust{
				ID:        illustID,
				Bookmarks: bookmarkCount - 1,
			},
		})
	}

	// Non‑HTMX fallback: redirect back to wherever we came from.
	utils.RedirectToWhenceYouCame(w, r, returnPath)

	return nil
}

func LikeRoute(w http.ResponseWriter, r *http.Request) error {
	token := session.GetUserToken(r)
	csrf := session.GetCookie(r, session.Cookie_CSRF)
	returnPath := r.FormValue(returnPathFormat)
	noAuthReturnPath := returnPath
	loginReturnPath := returnPath
	strLikeCount := r.FormValue(likeCountFormat)

	likeCount, err := strconv.Atoi(strLikeCount)
	if err != nil {
		return i18n.Error("Invalid bookmark count.")
	}

	artworkID := GetPathVar(r, artworkIDFormat)
	if artworkID == "" {
		return i18n.Error("No ID provided.")
	}

	if token == "" || csrf == "" {
		return UnauthorizedPage(w, r, noAuthReturnPath, loginReturnPath)
	}

	payload := fmt.Sprintf(`{"illust_id": "%s"}`, artworkID)
	contentType := "application/json; charset=utf-8"
	apiURL := core.PostIllustLikeURL()
	cookies := map[string]string{
		"PHPSESSID": token,
	}

	_, err = requests.PerformPOST(r.Context(), apiURL, payload, cookies, csrf, contentType, r.Header)
	if err != nil {
		return err
	}

	userID := strings.Split(token, "_")
	artworkURL := fmt.Sprintf("https://www.pixiv.net/ajax/illust/%s", artworkID)
	urlsToInvalidate := []string{artworkURL}

	invalidatedCount, invalidatedURLs := requests.InvalidateURLs(urlsToInvalidate)
	audit.GlobalAuditor.Logger.Infow("Invalidated cache entries",
		"userId", userID[0],
		"invalidatedCount", invalidatedCount,
		"requestedURLs", urlsToInvalidate,
		"invalidatedURLs", invalidatedURLs,
	)

	isHtmx := r.Header.Get("HX-Request") == "true"
	if isHtmx {
		return template.RenderHTML(w, r, Data_unlikePartial{
			Illust: core.Illust{
				Likes: likeCount + 1,
			},
		})
	}

	utils.RedirectToWhenceYouCame(w, r, returnPath)

	return nil
}

/*
NOTE: we're using the mobile API for FollowRoute and UnfollowRoute since it's an actual AJAX API
			instead of some weird php thing for the usual desktop routes (/bookmark_add.php and /rpc_group_setting.php)

			the desktop routes return HTML for the pixiv SPA when they feel like it and don't return helpful responses
			when you send a request that doesn't perfectly meet their specifications, making troubleshooting a nightmare

			for comparison, the mobile API worked first try without any issues

			interestingly enough, replicating the requests for the desktop routes via cURL worked fine but a Go implementation
			just refused to work
*/

func FollowRoute(w http.ResponseWriter, r *http.Request) error {
	token := session.GetUserToken(r)
	csrf := session.GetCookie(r, session.Cookie_CSRF)
	returnPath := r.FormValue(returnPathFormat)
	noAuthReturnPath := returnPath
	loginReturnPath := returnPath

	followUserID := r.FormValue(userIDFormat)
	if followUserID == "" {
		return i18n.Error("No user ID provided.")
	}

	privateVal := r.FormValue(privateFormat)
	isPrivate := privateVal == "true" || privateVal == "on"

	restrict := "0"
	if isPrivate {
		restrict = "1"
	}

	if token == "" || csrf == "" {
		return UnauthorizedPage(w, r, noAuthReturnPath, loginReturnPath)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("mode", "add_bookmark_user")
	writer.WriteField("restrict", restrict)
	writer.WriteField("user_id", followUserID)
	writer.Close()

	fields := map[string]string{
		"mode":     "add_bookmark_user",
		"restrict": restrict,
		"user_id":  followUserID,
	}
	url := core.PostTouchAPI()
	cookies := map[string]string{
		"PHPSESSID": token,
	}

	rawResp, err := requests.PerformPOST(r.Context(), url, fields, cookies, csrf, "", r.Header)
	if err != nil {
		return err
	}

	// Parse response
	var resp core.FollowResponse
	if err := json.Unmarshal(rawResp.Body, &resp); err != nil {
		return fmt.Errorf("failed to parse follow response: %w", err)
	}

	// Check if the follow operation was successful
	if !resp.IsSucceed {
		return fmt.Errorf("failed to follow user")
	}

	userID := strings.Split(token, "_")
	followerFollowingListURL := fmt.Sprintf("https://www.pixiv.net/ajax/user/%s/following", userID[0])
	followerProfileURL := fmt.Sprintf("https://www.pixiv.net/ajax/user/%s?full=", userID[0])
	followedProfileURL := fmt.Sprintf("https://www.pixiv.net/ajax/user/%s?full=", followUserID)
	urlsToInvalidate := []string{followerFollowingListURL, followerProfileURL, followedProfileURL}

	invalidatedCount, invalidatedURLs := requests.InvalidateURLs(urlsToInvalidate)
	audit.GlobalAuditor.Logger.Infow("Invalidated cache entries",
		"userId", userID[0],
		"followUserId", followUserID,
		"invalidatedCount", invalidatedCount,
		"requestedURLs", urlsToInvalidate,
		"invalidatedURLs", invalidatedURLs,
	)

	isHtmx := r.Header.Get("HX-Request") == "true"
	if isHtmx {
		return template.RenderHTML(w, r, Data_deleteFollowPartial{
			User: core.User{
				ID:         followUserID,
				IsFollowed: true,
			},
		})
	}

	utils.RedirectToWhenceYouCame(w, r, returnPath)

	return nil
}

func UnfollowRoute(w http.ResponseWriter, r *http.Request) error {
	token := session.GetUserToken(r)
	csrf := session.GetCookie(r, session.Cookie_CSRF)
	returnPath := r.FormValue(returnPathFormat)
	noAuthReturnPath := returnPath
	loginReturnPath := returnPath

	followUserID := GetQueryParam(r, userIDFormat)
	if followUserID == "" {
		followUserID = r.FormValue(userIDFormat)
	}
	if followUserID == "" {
		return i18n.Error("No user ID provided.")
	}

	if token == "" || csrf == "" {
		return UnauthorizedPage(w, r, noAuthReturnPath, loginReturnPath)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("mode", "delete_bookmark_user")
	writer.WriteField("user_id", followUserID)
	writer.Close()

	fields := map[string]string{
		"mode":    "delete_bookmark_user",
		"user_id": followUserID,
	}
	url := core.PostTouchAPI()
	cookies := map[string]string{
		"PHPSESSID": token,
	}

	rawResp, err := requests.PerformPOST(r.Context(), url, fields, cookies, csrf, "", r.Header)
	if err != nil {
		return err
	}

	// Parse response
	var resp core.FollowResponse
	if err := json.Unmarshal(rawResp.Body, &resp); err != nil {
		return fmt.Errorf("failed to parse follow response: %w", err)
	}

	// Check if the follow operation was successful
	if !resp.IsSucceed {
		return fmt.Errorf("failed to unfollow user")
	}

	userID := strings.Split(token, "_")
	followerFollowingListURL := fmt.Sprintf("https://www.pixiv.net/ajax/user/%s/following", userID[0])
	followerProfileURL := fmt.Sprintf("https://www.pixiv.net/ajax/user/%s?full=", userID[0])
	followedProfileURL := fmt.Sprintf("https://www.pixiv.net/ajax/user/%s?full=", followUserID)
	urlsToInvalidate := []string{followerFollowingListURL, followerProfileURL, followedProfileURL}

	invalidatedCount, invalidatedURLs := requests.InvalidateURLs(urlsToInvalidate)
	audit.GlobalAuditor.Logger.Infow("Invalidated cache entries",
		"userId", userID[0],
		"followUserId", followUserID,
		"invalidatedCount", invalidatedCount,
		"requestedURLs", urlsToInvalidate,
		"invalidatedURLs", invalidatedURLs,
	)

	isHtmx := r.Header.Get("HX-Request") == "true"
	if isHtmx {
		return template.RenderHTML(w, r, Data_addFollowPartial{
			User: core.User{
				ID:         followUserID,
				IsFollowed: false,
			},
		})
	}

	utils.RedirectToWhenceYouCame(w, r, returnPath)

	return nil
}
