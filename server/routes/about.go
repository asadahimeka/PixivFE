// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package routes

import (
	"fmt"
	"net/http"

	"codeberg.org/pixivfe/pixivfe/assets/components/layout"
	"codeberg.org/pixivfe/pixivfe/assets/components/pages"
	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/server/requestcontext"
	"github.com/a-h/templ"
)

// AboutPage is the handler for the /about page.
func AboutPage(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, stale-while-revalidate=%d",
		int(config.GlobalConfig.HTTPCache.MaxAge.Seconds()),
		int(config.GlobalConfig.HTTPCache.StaleWhileRevalidate.Seconds())))

	commonData := requestcontext.FromRequest(r).CommonData

	pageData := pages.AboutData{
		Title:          "About",
		Time:           config.GlobalConfig.Instance.StartingTime,
		Version:        config.GlobalConfig.Instance.Version,
		ImageProxy:     config.GlobalConfig.ContentProxies.Image.String(),
		AcceptLanguage: config.GlobalConfig.Request.AcceptLanguage,
	}

	// Create the actual content component (the layout "child")
	pageContent := pages.About(commonData, pageData)

	// Create a new context that includes aboutContent as children.
	// This is how the { children... } placeholder in layout.Default will be populated.
	ctxWithChildren := templ.WithChildren(r.Context(), pageContent)

	// Render the Default layout, passing the common data it needs and the context with children.
	// The layout.Default component will then render aboutContent in its { children... } slot
	return layout.Default(commonData, pageData.Title, false).Render(ctxWithChildren, w)
}
