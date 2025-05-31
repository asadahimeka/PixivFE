// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"fmt"
	"net/http"

	"codeberg.org/pixivfe/pixivfe/audit"
	"codeberg.org/pixivfe/pixivfe/core/requests"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"github.com/goccy/go-json"
)

func GetNewestArtworks(r *http.Request, worktype string, r18 string) ([]ArtworkBrief, error) {
	var body struct {
		Artworks []ArtworkBrief `json:"illusts"`
		// LastId string
	}

	url := GetNewestArtworksURL(worktype, r18, "0")

	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	rawResp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
	if err != nil {
		return nil, err
	}
	rawResp = RewriteContentURLs(r, rawResp)

	err = json.Unmarshal(rawResp, &body)
	if err != nil {
		return nil, err
	}

	// Populate thumbnails for each artwork
	for id, artwork := range body.Artworks {
		if err := artwork.PopulateThumbnails(); err != nil {
			audit.GlobalAuditor.Logger.Errorf("Failed to populate thumbnails for artwork ID %s: %v", id, err)
			return nil, fmt.Errorf("failed to populate thumbnails for artwork ID %d: %w", id, err)
		}
		body.Artworks[id] = artwork
	}

	return body.Artworks, nil
}
