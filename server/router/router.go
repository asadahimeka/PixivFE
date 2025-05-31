// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package router

import (
	"fmt"
	"io/fs"
	"net/http"

	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/i18n"
	"codeberg.org/pixivfe/pixivfe/server/assets"
	"codeberg.org/pixivfe/pixivfe/server/middleware"
	"codeberg.org/pixivfe/pixivfe/server/middleware/limiter"
	"codeberg.org/pixivfe/pixivfe/server/requestcontext"
	"codeberg.org/pixivfe/pixivfe/server/routes"
	"github.com/gorilla/mux"
)

// DefineRoutes sets up all the routes for the application.
//
// It returns a configured mux.Router with all paths and their corresponding handlers.
//
//nolint:funlen,lll // Inherently long.
func DefineRoutes() *mux.Router {
	router := mux.NewRouter()

	// By default, mux.Router does not distinguish between "%2F" and "/"
	// For example, "/tags/Fate%2FGrand%00Order" will be parsed as "/tags/Fate/Grand Order"
	// This method fixes that
	router.UseEncodedPath()

	// Tutorial: Adding new routes
	// 1. Use router.HandleFunc to define the path and handler
	// 2. Wrap the handler function, defined in package routes, with middleware.CatchError for error handling
	// 3. Specify the HTTP method(s) using .Methods()
	// 4. For URL parameters, use curly braces in the path, e.g., "/users/{id}"
	// 5. Group similar routes together for better organization
	//
	// Example:
	// router.HandleFunc("/new/route/{param}", middleware.CatchError(routes.NewHandler)).Methods("HEAD", "GET", "POST")

	// Handle trailing slashes
	router.MatcherFunc(hasTrailingSlash).HandlerFunc(removeTrailingSlash)

	// Handle /en/ language prefix in URLs
	router.MatcherFunc(hasEnPrefix).HandlerFunc(removeEnPrefix)

	// Serve static files from embedded assets.
	// assets.FS is embedded with "all:assets", meaning it contains an 'assets' directory at its root.
	// We need to serve files from this 'assets' subdirectory.
	staticContentFS, err := fs.Sub(assets.FS, "assets")
	if err != nil {
		// This panic will occur at startup if the "assets" directory is not found in the embed.FS.
		// This indicates a problem with the //go:embed directive or the project structure.
		panic(fmt.Errorf("failed to create sub-filesystem for embedded 'assets' directory: %w", err))
	}

	// http.FileServer serves files from an http.FileSystem.
	// http.FS converts an io/fs.FS (like our staticContentFS) to an http.FileSystem.
	fileServer := http.FileServer(http.FS(staticContentFS))

	// Serve specific files from the root of the 'assets' subdirectory (e.g., "assets/manifest.json").
	router.Path("/manifest.json").Handler(fileServer)
	router.Path("/robots.txt").Handler(fileServer)

	// Serve files from subdirectories within 'assets' (e.g., "assets/css/", "assets/img/").
	router.PathPrefix("/img/").Handler(fileServer)
	router.PathPrefix("/css/").Handler(fileServer)
	router.PathPrefix("/js/").Handler(fileServer)
	router.PathPrefix("/fonts/").Handler(fileServer)

	// Proxy routes
	handleStripPrefix(router, "/proxy/i.pximg.net/", middleware.CatchError(routes.IPximgProxy)).Methods("HEAD", "GET")
	handleStripPrefix(router, "/proxy/embed.pixiv.net/", middleware.CatchError(routes.EmbedPixivProxy)).Methods("HEAD", "GET")
	handleStripPrefix(router, "/proxy/s.pximg.net/", middleware.CatchError(routes.SPximgProxy)).Methods("HEAD", "GET")
	handleStripPrefix(router, "/proxy/source.pixiv.net/", middleware.CatchError(routes.SourcePixivProxy)).Methods("HEAD", "GET")
	// /ugoira/ segment for compatibility with external ugoira proxies that
	// can only reverse proxy the t-hk.ugoira.com domain directly (e.g. caddy)
	handleStripPrefix(router, "/proxy/ugoira.com/ugoira/", middleware.CatchError(routes.UgoiraProxy)).Methods("HEAD", "GET")

	// Main application routes
	router.HandleFunc("/", middleware.CatchError(routes.IndexPage)).Methods("HEAD", "GET")
	router.HandleFunc("/about", middleware.CatchError(routes.AboutPage)).Methods("HEAD", "GET")
	router.HandleFunc("/newest", middleware.CatchError(routes.NewestPage)).Methods("HEAD", "GET")
	router.HandleFunc("/discovery", middleware.CatchError(routes.DiscoveryPage)).Methods("HEAD", "GET")
	router.HandleFunc("/discovery/novel", middleware.CatchError(routes.NovelDiscoveryPage)).Methods("HEAD", "GET")
	router.HandleFunc("/discovery/users", middleware.CatchError(routes.UserDiscoveryPage)).Methods("HEAD", "GET")

	// Refresh routes for discovery routes
	router.HandleFunc("/discovery", middleware.CatchError(routes.DiscoveryPageRefresh)).Methods("POST")
	router.HandleFunc("/discovery/novel", middleware.CatchError(routes.NovelDiscoveryPageRefresh)).Methods("POST")
	router.HandleFunc("/discovery/users", middleware.CatchError(routes.UserDiscoveryPageRefresh)).Methods("POST")

	// Ranking routes
	router.HandleFunc("/ranking", middleware.CatchError(routes.RankingPage)).Methods("HEAD", "GET")
	router.HandleFunc("/rankingCalendar", middleware.CatchError(routes.RankingCalendarPage)).Methods("HEAD", "GET")
	router.HandleFunc("/rankingCalendar", middleware.CatchError(routes.RankingCalendarPicker)).Methods("POST")

	// User routes
	router.HandleFunc("/users/{id}.atom.xml", middleware.CatchError(routes.UserAtomFeed)).Methods("HEAD", "GET")
	router.HandleFunc("/users/{id}/{category}.atom.xml", middleware.CatchError(routes.UserAtomFeed)).Methods("HEAD", "GET")
	router.HandleFunc("/users/{id}", middleware.CatchError(routes.UserPage)).Methods("HEAD", "GET")
	router.HandleFunc("/users/{id}/{category}", middleware.CatchError(routes.UserPage)).Methods("HEAD", "GET")
	router.HandleFunc("/member.php", legacyRedirect("/users/", "id"))

	// Artwork routes
	router.HandleFunc("/artworks/{id}", middleware.CatchError(routes.ArtworkPage)).Methods("HEAD", "GET")
	router.HandleFunc("/artworks-multi/{ids}", middleware.CatchError(routes.ArtworkMultiPage)).Methods("HEAD", "GET")
	router.HandleFunc("/member_illust.php", legacyRedirect("/artworks/", "illust_id"))

	// Manga routes
	router.HandleFunc("/users/{user_id}/series/{series_id}", middleware.CatchError(routes.MangaSeriesPage)).Methods("HEAD", "GET")

	// Novel routes
	router.HandleFunc("/novel/show.php", legacyRedirect("/novel/", "id"))
	router.HandleFunc("/novel/{id}", middleware.CatchError(routes.NovelPage)).Methods("HEAD", "GET")
	router.HandleFunc("/novel/series/{id}", middleware.CatchError(routes.NovelSeriesPage)).Methods("HEAD", "GET")

	// Pixivision routes
	router.HandleFunc("/pixivision", middleware.CatchError(routes.PixivisionHomePage)).Methods("HEAD", "GET")
	router.HandleFunc("/pixivision/a/{id}", middleware.CatchError(routes.PixivisionArticlePage)).Methods("HEAD", "GET")
	router.HandleFunc("/pixivision/c/{id}", middleware.CatchError(routes.PixivisionCategoryPage)).Methods("HEAD", "GET")
	router.HandleFunc("/pixivision/t/{id}", middleware.CatchError(routes.PixivisionTagPage)).Methods("HEAD", "GET")

	// Settings routes
	router.HandleFunc("/settings", middleware.CatchError(routes.SettingsPage)).Methods("HEAD", "GET")
	router.HandleFunc("/settings/{type}", middleware.CatchError(routes.SettingsPost)).Methods("POST")

	// User action routes
	router.HandleFunc("/self", middleware.CatchError(routes.SelfUserPage)).Methods("HEAD", "GET")
	router.HandleFunc("/self/followingUsers", middleware.CatchError(routes.SelfFollowingUsersPage)).Methods("HEAD", "GET")
	router.HandleFunc("/self/followingWorks", middleware.CatchError(routes.SelfFollowingWorksPage)).Methods("HEAD", "GET")
	router.HandleFunc("/self/bookmarks", middleware.CatchError(routes.SelfBookmarksPage)).Methods("HEAD", "GET")
	router.HandleFunc("/self/addBookmark/{artwork_id}", middleware.CatchError(routes.AddBookmarkRoute)).Methods("POST")
	router.HandleFunc("/self/deleteBookmark/{bookmark_id}", middleware.CatchError(routes.DeleteBookmarkRoute)).Methods("POST")
	router.HandleFunc("/self/like/{artwork_id}", middleware.CatchError(routes.LikeRoute)).Methods("POST")
	router.HandleFunc("/self/login", middleware.CatchError(routes.LoginPage)).Methods("HEAD", "GET")

	// oEmbed endpoint
	router.HandleFunc("/oembed", middleware.CatchError(routes.Oembed)).Methods("HEAD", "GET")

	// Tag routes
	router.HandleFunc("/tags/{name}", middleware.CatchError(routes.TagPage)).Methods("HEAD", "GET")
	router.HandleFunc("/tags/{name}/", middleware.CatchError(routes.TagPage)).Methods("POST")
	router.HandleFunc("/tags", middleware.CatchError(routes.TagPage)).Methods("HEAD", "GET")
	router.HandleFunc("/tags", middleware.CatchError(routes.AdvancedTagPost)).Methods("POST")

	// REST API routes (for htmx)
	// safe methods
	router.HandleFunc("/api/v1/artwork", middleware.CatchError(routes.ArtworkPartial)).Methods("HEAD", "GET")
	router.HandleFunc("/api/v1/recent", middleware.CatchError(routes.RecentPartial)).Methods("HEAD", "GET")
	router.HandleFunc("/api/v1/related", middleware.CatchError(routes.RelatedPartial)).Methods("HEAD", "GET")
	router.HandleFunc("/api/v1/comments", middleware.CatchError(routes.CommentsPartial)).Methods("HEAD", "GET")
	router.HandleFunc("/api/v1/discovery", middleware.CatchError(routes.DiscoveryPartial)).Methods("HEAD", "GET")
	// non-safe methods
	router.HandleFunc("/api/v1/follow", middleware.CatchError(routes.FollowRoute)).Methods("PUT")
	router.HandleFunc("/api/v1/follow", middleware.CatchError(routes.UnfollowRoute)).Methods("DELETE")
	// POST routes for backward compatibility
	router.HandleFunc("/api/v1/follow", middleware.CatchError(handleFollowPost)).Methods("POST")

	// Diagnostic routes
	if config.GlobalConfig.Development.InDevelopment {
		router.HandleFunc("/diagnostics", middleware.CatchError(routes.Diagnostics)).Methods("HEAD", "GET")
		router.HandleFunc("/diagnostics/spans.json", middleware.CatchError(routes.DiagnosticsData)).Methods("HEAD", "GET")
		router.HandleFunc("/diagnostics/reset", routes.ResetDiagnosticsData)
	}

	// Link token route
	if config.GlobalConfig.Limiter.DetectionMethod == config.LinkTokenDetectionMethod {
		router.HandleFunc("/limiter/{token}.css", middleware.CatchError(limiter.LinkTokenHandler)).Methods("GET")
	}

	// Turnstile verification route
	if config.GlobalConfig.Limiter.DetectionMethod == config.TurnstileDetectionMethod {
		router.HandleFunc("/limiter/turnstile/action", middleware.CatchError(limiter.TurnstileActionHandler)).Methods("GET")
		router.HandleFunc("/limiter/turnstile/verify", middleware.CatchError(limiter.TurnstileVerifyHandler)).Methods("POST")
		router.HandleFunc("/limiter/turnstile/clear", middleware.CatchError(limiter.TurnstileClearHandler)).Methods("GET")
	}

	// Handle non-existent routes
	router.NewRoute().HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Signal to the HandleError middleware that a route was not found.
		// We don't write to ResponseWriter or call ErrorPage here
		// as the HandleError middleware will render the appropriate error page with a 404 status.
		ctx := requestcontext.FromRequest(r)
		ctx.RequestError = i18n.Error("Route not found")
		ctx.StatusCode = http.StatusNotFound
	})

	return router
}
