// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package routes

import (
	"net/http"

	"codeberg.org/pixivfe/pixivfe/core"
	"codeberg.org/pixivfe/pixivfe/server/template"
)

func DiscoveryPage(w http.ResponseWriter, r *http.Request) error {
	mode := GetQueryParam(r, "mode", "safe")
	urlc := template.PartialURL{
		Path: "/discovery",
		Query: map[string]string{
			"mode": mode,
		},
	}

	w.Header().Set("Cache-Control", "no-store")

	isHtmx := r.Header.Get("HX-Request") == "true"

	// For all HTMX requests, return async loaded page
	if isHtmx {
		return template.RenderHTML(w, r, Data_discovery{
			Title:             "Discovery",
			Queries:           urlc,
			RequiresAsyncLoad: true, // Indicates skeleton will be rendered
		})
	}

	// For all other requests, return actual content
	works, err := core.GetDiscoveryArtwork(r, mode)
	if err != nil {
		return err
	}

	return template.RenderHTML(w, r, Data_discovery{
		Title:             "Discovery",
		Artworks:          works,
		Queries:           urlc,
		RequiresAsyncLoad: false, // Indicates full content will be rendered
	})
}

func NovelDiscoveryPage(w http.ResponseWriter, r *http.Request) error {
	mode := GetQueryParam(r, "mode", "safe")

	works, err := core.GetDiscoveryNovels(r, mode)
	if err != nil {
		return err
	}

	urlc := template.PartialURL{
		Path:  "/discovery/novel",
		Query: map[string]string{"mode": mode},
	}

	return template.RenderHTML(w, r, Data_novelDiscovery{
		Novels:  works,
		Title:   "Discovery",
		Queries: urlc,
	})
}

func UserDiscoveryPage(w http.ResponseWriter, r *http.Request) error {
	users, err := core.GetDiscoveryUsers(r)
	if err != nil {
		return err
	}

	urlc := template.PartialURL{
		Path:  "/discovery/users",
		Query: map[string]string{},
	}

	return template.RenderHTML(w, r, Data_userDiscovery{
		Users:   users,
		Title:   "Discovery",
		Queries: urlc,
	})
}

// Discovery refresh route handlers - provides a redirect back to the
// discovery pages.
//
// Separate handlers used to avoid browser form resubmission issues.

func DiscoveryPageRefresh(w http.ResponseWriter, r *http.Request) error {
	// Check for HTMX request
	if r.Header.Get("HX-Request") == "true" {
		// Call DiscoveryPage directly instead of redirecting
		return DiscoveryPage(w, r)
	}

	// Default redirect behavior for non-HTMX requests
	redirectURL := "/discovery"
	if r.FormValue("reset") != "on" && r.URL.RawQuery != "" {
		redirectURL += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	return nil
}

func NovelDiscoveryPageRefresh(w http.ResponseWriter, r *http.Request) error {
	// Check for HTMX request
	if r.Header.Get("HX-Request") == "true" {
		// Call NovelDiscoveryPage directly instead of redirecting
		return NovelDiscoveryPage(w, r)
	}

	// Default redirect behavior for non-HTMX requests
	redirectURL := "/discovery/novel"
	if r.FormValue("reset") != "on" && r.URL.RawQuery != "" {
		redirectURL += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	return nil
}

func UserDiscoveryPageRefresh(w http.ResponseWriter, r *http.Request) error {
	// Check for HTMX request
	if r.Header.Get("HX-Request") == "true" {
		// Call UserDiscoveryPage directly instead of redirecting
		return UserDiscoveryPage(w, r)
	}

	// Default redirect behavior for non-HTMX requests
	redirectURL := "/discovery/users"
	if r.FormValue("reset") != "on" && r.URL.RawQuery != "" {
		redirectURL += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	return nil
}
