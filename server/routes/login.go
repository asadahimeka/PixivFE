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

func LoginPage(w http.ResponseWriter, r *http.Request) error {
	// Never cache the login page to be safe
	w.Header().Set("Cache-Control", "no-store")

	commonData := requestcontext.FromRequest(r).CommonData

	pageData := pages.LoginData{
		Title:            "Sign in",
		LoginReturnPath:  GetQueryParam(r, "loginReturnPath"),
		NoAuthReturnPath: GetQueryParam(r, "noAuthReturnPath"),
	}

	pageContent := pages.Login(pageData)

	ctxWithChildren := templ.WithChildren(r.Context(), pageContent)

	return layout.Default(commonData, pageData.Title, true).Render(ctxWithChildren, w)
}
