// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package middleware

import (
	"net/http"

	"codeberg.org/pixivfe/pixivfe/i18n"
	"codeberg.org/pixivfe/pixivfe/server/session"
)

// SetLocaleFromCookie is a middleware that extracts the user's locale preference
// from a cookie and sets it in the i18n package.
func SetLocaleFromCookie(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		locale := session.GetCookie(r, session.Cookie_Locale)
		i18n.SetLocale(locale)

		next.ServeHTTP(w, r)
	})
}
