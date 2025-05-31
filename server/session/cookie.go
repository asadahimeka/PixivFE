// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
User settings (Using HTTP cookies)
*/
package session

import (
	"net"
	"net/http"
	"net/url"
	"time"
)

// the __Host thing force it to be secure and same-origin (no subdomain)
//
// ref: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie
const (
	Cookie_Token                   CookieName = "pixivfe-Token"
	Cookie_CSRF                    CookieName = "pixivfe-CSRF"
	Cookie_P_AB                    CookieName = "pixivfe-P_AB"
	Cookie_Username                CookieName = "pixivfe-Username"
	Cookie_UserID                  CookieName = "pixivfe-UserID"
	Cookie_UserAvatar              CookieName = "pixivfe-UserAvatar"
	Cookie_ImageProxy              CookieName = "pixivfe-ImageProxy"
	Cookie_StaticProxy             CookieName = "pixivfe-StaticProxy"
	Cookie_UgoiraProxy             CookieName = "pixivfe-UgoiraProxy"
	Cookie_TZ                      CookieName = "pixivfe-TZ"
	Cookie_NovelFontType           CookieName = "pixivfe-NovelFontType"
	Cookie_NovelViewMode           CookieName = "pixivfe-NovelViewMode"
	Cookie_ThumbnailToNewTab       CookieName = "pixivfe-ThumbnailToNewTab"
	Cookie_ArtworkPreview          CookieName = "pixivfe-ArtworkPreview"
	Cookie_VisualEffectsEnabled    CookieName = "pixivfe-VisualEffectsEnabled"
	Cookie_SeasonalEffectsDisabled CookieName = "pixivfe-SeasonalEffectsDisabled"
	Cookie_VisibilityArtR18        CookieName = "pixivfe-VisibilityArtR18"
	Cookie_VisibilityArtR18G       CookieName = "pixivfe-VisibilityArtR18G"
	Cookie_VisibilityArtAI         CookieName = "pixivfe-VisibilityArtAI"
	Cookie_CensorArtAI             CookieName = "pixivfe-CensorArtAI"
	Cookie_HideArtAI               CookieName = "pixivfe-HideArtAI"
	Cookie_Locale                  CookieName = "pixivfe-Locale"
	Cookie_LogoStyle               CookieName = "pixivfe-LogoStyle"
	Cookie_BlacklistArtist         CookieName = "pixivfe-BlacklistArtist"
	Cookie_BlacklistTag            CookieName = "pixivfe-BlacklistTag"
)

// Go can't make this a const...
var (
	AllCookieNames = []CookieName{
		Cookie_Token,
		Cookie_CSRF,
		Cookie_P_AB,
		Cookie_Username,
		Cookie_UserID,
		Cookie_UserAvatar,
		Cookie_ImageProxy,
		Cookie_StaticProxy,
		Cookie_UgoiraProxy,
		Cookie_TZ,
		Cookie_NovelFontType,
		Cookie_NovelViewMode,
		Cookie_ThumbnailToNewTab,
		Cookie_ArtworkPreview,
		Cookie_VisibilityArtR18,
		Cookie_VisibilityArtR18G,
		Cookie_VisibilityArtAI,
		Cookie_CensorArtAI,
		Cookie_HideArtAI,
		Cookie_VisualEffectsEnabled,
		Cookie_SeasonalEffectsDisabled,
		Cookie_Locale,
		Cookie_LogoStyle,
		Cookie_BlacklistArtist,
		Cookie_BlacklistTag,
	}

	// Cookies will expire in 30 days from when they are set.
	cookieMaxAge = 30 * 24 * time.Hour

	cookieExpireDelete = time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
)

type CookieName string

func GetCookie(r *http.Request, name CookieName) string {
	cookie, err := r.Cookie(string(name))
	if err != nil {
		return ""
	}

	value, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		return ""
	}

	return value
}

func SetCookie(w http.ResponseWriter, r *http.Request, name CookieName, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     string(name),
		Value:    url.QueryEscape(value),
		Path:     "/",
		Expires:  time.Now().Add(cookieMaxAge),
		HttpOnly: true,
		Secure:   ShouldCookieBeSecure(r),
		SameSite: http.SameSiteStrictMode, // bye-bye cross site forgery
	})
}

func ClearCookie(w http.ResponseWriter, r *http.Request, name CookieName) {
	http.SetCookie(w, &http.Cookie{
		Name:     string(name),
		Value:    "",
		Path:     "/",
		Expires:  cookieExpireDelete,
		HttpOnly: true,
		Secure:   ShouldCookieBeSecure(r),
		SameSite: http.SameSiteStrictMode,
	})
}

func ClearAllCookies(w http.ResponseWriter, r *http.Request) {
	for _, name := range AllCookieNames {
		ClearCookie(w, r, name)
	}
}

func ValidateTimezone(tz string) (*time.Location, error) {
	if tz == "" {
		return time.UTC, nil
	}

	location, err := time.LoadLocation(tz)
	if err != nil {
		return time.UTC, err
	}

	return location, nil
}

func GetUserTimezone(r *http.Request) *time.Location {
	tz := GetCookie(r, Cookie_TZ)

	location, err := ValidateTimezone(tz)
	if err != nil {
		return time.UTC
	}

	return location
}

// ShouldCookieBeSecure determines if a cookie should have the Secure attribute.
//
// Target environments are (containerized and bare metal):
//   - Internet -> reverse proxy (e.g. cloudflare) -> reverse proxy -> application
//   - Internet -> reverse proxy -> application
//   - LAN -> reverse proxy -> application
//   - LAN -> application
//   - localhost -> application
//
// This function will incorrectly return false if the last reverse proxy
// in the chain has a public IP address, but this is expected to be a small minority
// of deployments.
func ShouldCookieBeSecure(r *http.Request) bool {
	// Always secure if directly using TLS
	if r.TLS != nil {
		return true
	}

	// Parse IP from RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false // Can't determine if it's secure
	}

	parsedIP := net.ParseIP(host)
	if parsedIP == nil {
		return false // Invalid IP
	}

	// Only trust X-Forwarded-Proto from private IPs
	if parsedIP.IsPrivate() && r.Header.Get("X-Forwarded-Proto") == "https" {
		return true
	}

	return false
}
