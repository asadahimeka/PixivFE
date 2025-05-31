// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package routes

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/core"
	"codeberg.org/pixivfe/pixivfe/i18n"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"codeberg.org/pixivfe/pixivfe/server/template"
)

func ArtworkMultiPage(w http.ResponseWriter, r *http.Request) error {
	ids_ := GetPathVar(r, "ids")
	ids := strings.Split(ids_, ",")

	artworks := make([]core.Illust, len(ids))

	wg := sync.WaitGroup{}
	// // gofiber/fasthttp's API is trash
	// // i can't replace r.Context() with this
	// // so i guess we will have to wait for network traffic to finish on error
	// ctx, cancel := context.WithCancel(r.Context())
	// defer cancel()
	// r.SetUserContext(ctx)
	var err_global error = nil
	for i, id := range ids {
		if _, err := strconv.Atoi(id); err != nil {
			err_global = i18n.Errorf("Invalid ID: %s", id)
			break
		}

		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()

			illust, err := core.GetArtwork(w, r, id)
			if err != nil {
				artworks[i] = core.Illust{
					Title: err.Error(), // this might be flaky
				}
				return
			}

			artworks[i] = *illust
		}(i, id)
	}
	// if err_global != nil {
	// 	cancel()
	// }
	wg.Wait()
	if err_global != nil {
		return err_global
	}

	for _, illust := range artworks {
		for _, img := range illust.Images {
			PreloadImage(w, img.Large)
		}
	}

	if session.GetUserToken(r) != "" {
		w.Header().Set("Cache-Control", "private, max-age=60")
	} else {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, stale-while-revalidate=%d",
			int(config.GlobalConfig.HTTPCache.MaxAge.Seconds()),
			int(config.GlobalConfig.HTTPCache.StaleWhileRevalidate.Seconds())))
	}

	return template.RenderHTML(w, r, Data_artworkMulti{
		Artworks: artworks,
		Title:    fmt.Sprintf("(%d images)", len(artworks)),
	})
}
