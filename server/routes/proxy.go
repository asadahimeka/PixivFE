// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package routes

import (
	"net/http"

	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/core/requests"
)

// SPximgProxy handles requests for static assets from s.pximg.net.
func SPximgProxy(w http.ResponseWriter, r *http.Request) error {
	return requests.ProxyHandler(w, r, "https://s.pximg.net", nil)
}

// IPximgProxy handles requests for image assets from i.pximg.net.
func IPximgProxy(w http.ResponseWriter, r *http.Request) error {
	headers := map[string]string{
		"Referer": "https://www.pixiv.net/",
	}
	return requests.ProxyHandler(w, r, "https://i.pximg.net", headers)
}

// UgoiraProxy handles requests for video assets from ugoira.com.
func UgoiraProxy(w http.ResponseWriter, r *http.Request) error {
	return requests.ProxyHandler(w, r, "https://ugoira.com/api/mp4", nil)
}

// For pixivision
func EmbedPixivProxy(w http.ResponseWriter, r *http.Request) error {
	headers := map[string]string{
		"User-Agent": config.GetRandomUserAgent(),
	}
	return requests.ProxyHandler(w, r, "https://embed.pixiv.net", headers)
}

// For pixivision
func SourcePixivProxy(w http.ResponseWriter, r *http.Request) error {
	headers := map[string]string{
		"User-Agent": config.GetRandomUserAgent(),
	}
	return requests.ProxyHandler(w, r, "https://source.pixiv.net", headers)
}
