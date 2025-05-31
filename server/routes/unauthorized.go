// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package routes

import (
	"net/http"

	"codeberg.org/pixivfe/pixivfe/assets/components/layout"
	"codeberg.org/pixivfe/pixivfe/assets/components/pages"
	"codeberg.org/pixivfe/pixivfe/server/requestcontext"
	"github.com/a-h/templ"
)

// UnauthorizedPage renders a page that prompts the user to login.
//
// Route handlers should http.Redirect to this route if the user
// lacks authentication and attempts an action that requires a
// personal pixiv account.
func UnauthorizedPage(w http.ResponseWriter, r *http.Request, noAuthReturnPath, loginReturnPath string) error {
	w.WriteHeader(http.StatusUnauthorized)

	w.Header().Add("HX-Push-Url", "/unauthorized")

	commonData := requestcontext.FromRequest(r).CommonData

	pageData := pages.UnauthorizedData{
		Title:            "Unauthorized",
		NoAuthReturnPath: noAuthReturnPath, // Return path if the user exits the login flow at any stage
		LoginReturnPath:  loginReturnPath,  // Return path if the user completes the login flow
	}

	pageContent := pages.Unauthorized(pageData)

	ctxWithChildren := templ.WithChildren(r.Context(), pageContent)

	err := layout.Default(commonData, pageData.Title, true).Render(ctxWithChildren, w)
	if err != nil {
		return err
	}
	return nil
}
