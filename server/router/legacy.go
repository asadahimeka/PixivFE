// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
Legacy handlers
*/
package router

import (
	"net/http"
	"strings"

	"codeberg.org/pixivfe/pixivfe/i18n"
	"codeberg.org/pixivfe/pixivfe/server/routes"
)

// handleFollowPost handles legacy POST requests for both follow/unfollow actions.
func handleFollowPost(w http.ResponseWriter, r *http.Request) error {
	action := r.FormValue("action")

	// Convert POST to appropriate handler based on action
	switch strings.ToLower(action) {
	case "unfollow", "delete":
		// Copy form values to query parameters for consistency with DELETE handler
		query := r.URL.Query()
		if returnPath := r.FormValue("returnPath"); returnPath != "" {
			query.Set("returnPath", returnPath)
		}

		if userid := r.FormValue("userid"); userid != "" {
			query.Set("userid", userid)
		}

		r.URL.RawQuery = query.Encode()

		return routes.UnfollowRoute(w, r)

	case "follow", "add":
		return routes.FollowRoute(w, r)

	default:
		return i18n.Error("Invalid or missing action parameter")
	}
}
