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

func GetNewestFromFollowing(r *http.Request, contentType, mode, page string) (NewestFromFollowingResponse, error) {
	var resp NewestFromFollowingResponse

	url := GetNewestFromFollowingURL(contentType, mode, page)

	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	rawResp, err := requests.FetchJSON(r.Context(), url, cookies, r.Header)
	if err != nil {
		return NewestFromFollowingResponse{}, err
	}

	rawResp = RewriteContentURLs(r, rawResp)

	err = json.Unmarshal(rawResp, &resp)
	if err != nil {
		return NewestFromFollowingResponse{}, err
	}

	// Convert tag translations to Tag objects
	resp.Body.Page.Tags = TagTranslationsToTags(nil, resp.Body.TagTranslation)

	// Populate thumbnails for each artwork and filter R-18 content if mode is "safe"
	//
	// We need to do this manually as the pixiv API doesn't
	// support this functionality natively
	filteredIllusts := make([]ArtworkBrief, 0, len(resp.Body.Thumbnails.Illust))

	for _, artwork := range resp.Body.Thumbnails.Illust {
		if mode == "safe" && artwork.XRestrict > 0 {
			continue
		}

		if err := artwork.PopulateThumbnails(); err != nil {
			audit.GlobalAuditor.Logger.Errorf("Failed to populate thumbnails for artwork ID %s: %v", artwork.ID, err)
			return NewestFromFollowingResponse{},
				fmt.Errorf("failed to populate thumbnails for artwork ID %s: %w", artwork.ID, err)
		}

		filteredIllusts = append(filteredIllusts, artwork)
	}
	resp.Body.Thumbnails.Illust = filteredIllusts

	return resp, nil
}
