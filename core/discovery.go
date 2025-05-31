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
	"github.com/tidwall/gjson"
)

func GetDiscoveryArtwork(r *http.Request, mode string) ([]ArtworkBrief, error) {
	// While we can technically fetch up to 100 artworks at a time,
	// such a large number can produce poor UX due to scroll fatigue
	//
	// Plus it's divisible by the grid-cols values used
	// in the frontend (1, 2, 3, 4, and 5)
	url := GetDiscoveryURL(mode, DiscoveryLimit)

	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	var artworks []ArtworkBrief

	rawResp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
	if err != nil {
		return nil, err
	}

	rawResp = RewriteContentURLs(r, rawResp)

	// We only want the "thumbnails.illust" field
	thumbnailsResp := gjson.GetBytes(rawResp, "thumbnails.illust").Raw

	err = json.Unmarshal([]byte(thumbnailsResp), &artworks)
	if err != nil {
		return nil, err
	}

	// Populate thumbnails for each artwork
	for id, artwork := range artworks {
		if err := artwork.PopulateThumbnails(); err != nil {
			audit.GlobalAuditor.Logger.Errorf("Failed to populate thumbnails for artwork ID %s: %v", id, err)
			return nil, fmt.Errorf("failed to populate thumbnails for artwork ID %d: %w", id, err)
		}
		artworks[id] = artwork
	}

	return artworks, nil
}

func GetDiscoveryNovels(r *http.Request, mode string) ([]NovelBrief, error) {
	url := GetDiscoveryNovelURL(mode, DiscoveryNovelLimit)

	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	var novels []NovelBrief

	rawResp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
	if err != nil {
		return nil, err
	}

	rawResp = RewriteContentURLs(r, rawResp)

	// We only want the "thumbnails.novel" field
	thumbnailsResp := gjson.GetBytes(rawResp, "thumbnails.novel").Raw

	err = json.Unmarshal([]byte(thumbnailsResp), &novels)
	if err != nil {
		return nil, err
	}

	return novels, nil
}

// GetDiscoveryUsers retrieves users on the discovery page along with their associated artworks and novels.
func GetDiscoveryUsers(r *http.Request) ([]User, error) {
	var users []User

	url := GetDiscoveryUserURL(DiscoveryUserLimit)

	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	var (
		artworks []ArtworkBrief
		novels   []NovelBrief
	)

	resp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
	if err != nil {
		return nil, err
	}

	resp = RewriteContentURLs(r, resp)

	// Extract and unmarshal users
	userData := gjson.GetBytes(resp, "users").Raw
	if err = json.Unmarshal([]byte(userData), &users); err != nil {
		return nil, fmt.Errorf("error unmarshalling users in GetDiscoveryUser: %w", err)
	}

	// Extract and unmarshal artworks
	artworkData := gjson.GetBytes(resp, "thumbnails.illust").Raw
	if err = json.Unmarshal([]byte(artworkData), &artworks); err != nil {
		return nil, fmt.Errorf("error unmarshalling artworks in GetDiscoveryUser: %w", err)
	}

	// Populate thumbnails for each artwork
	for id, artwork := range artworks {
		if err := artwork.PopulateThumbnails(); err != nil {
			audit.GlobalAuditor.Logger.Errorf("Failed to populate thumbnails for artwork ID %s: %v", id, err)
			return nil, fmt.Errorf("failed to populate thumbnails for artwork ID %d: %w", id, err)
		}
		artworks[id] = artwork
	}

	// Extract and unmarshal novels
	novelData := gjson.GetBytes(resp, "thumbnails.novel").Raw
	if err = json.Unmarshal([]byte(novelData), &novels); err != nil {
		return nil, fmt.Errorf("error unmarshalling novels in GetDiscoveryUser: %w", err)
	}

	// Associate artworks and novels with users
	AssociateContentWithUsers(&users, artworks, novels)

	return users, nil
}
