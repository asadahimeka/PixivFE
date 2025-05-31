// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
Helpers
*/
package router

import (
	"net/http"
	"net/url"
	"strings"

	"codeberg.org/pixivfe/pixivfe/server/routes"
	"github.com/gorilla/mux"
)

// enPrefixPaths defines paths for which we should handle an "/en/" prefix.
var enPrefixPaths = []string{
	"users/",
	"artworks/",
	"novel/",
}

// handleStripPrefix is a utility function that combines path prefix matching with
// stripping the prefix from the request URL before passing it to the handler.
func handleStripPrefix(router *mux.Router, pathPrefix string, handler http.Handler) *mux.Route {
	return router.PathPrefix(pathPrefix).Handler(http.StripPrefix(pathPrefix, handler))
}

// hasTrailingSlash is a helper function to check for trailing slashes.
func hasTrailingSlash(r *http.Request, _ *mux.RouteMatch) bool {
	return r.URL.Path != "/" && strings.HasSuffix(r.URL.Path, "/")
}

// removeTrailingSlash is a helper function to remove trailing slash and redirect.
func removeTrailingSlash(w http.ResponseWriter, r *http.Request) {
	url := r.URL

	if len(url.Path) > 1 {
		url.Path = url.Path[:len(url.Path)-1]
	}

	// iacore: i think this won't have open redirect vuln
	http.Redirect(w, r, url.String(), http.StatusPermanentRedirect)
}

// legacyRedirect is a helper function to redirect requests to
// a target path while preserving the specified query parameter.
func legacyRedirect(targetPath string, preservedParam string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, targetPath+routes.GetQueryParam(r, preservedParam), http.StatusPermanentRedirect)
	}
}

// hasEnPrefix checks if a request path starts with /en/.
func hasEnPrefix(r *http.Request, _ *mux.RouteMatch) bool {
	path := r.URL.Path

	// Check if path starts with /en/
	if len(path) <= 4 || path[:4] != "/en/" {
		return false
	}

	// Check if what follows /en/ is one of enPrefixPaths
	for _, validPath := range enPrefixPaths {
		if strings.HasPrefix(path[4:], validPath) {
			return true
		}
	}

	return false
}

// removeEnPrefix removes the /en/ prefix from the request URL and forwards to the canonical path.
func removeEnPrefix(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	canonicalPath := path[3:] // Remove the '/en' part (keep the trailing slash)

	// Create new URL with the en prefix removed
	target, _ := url.Parse(r.URL.String())
	target.Path = canonicalPath

	http.Redirect(w, r, target.String(), http.StatusMovedPermanently)
}
