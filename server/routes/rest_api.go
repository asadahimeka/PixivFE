// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
A true REST API

For use with htmx page renders
*/
package routes

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/core"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"codeberg.org/pixivfe/pixivfe/server/template"
)

// ArtworkParams is a type alias for core.FastIllustParams
type ArtworkParams core.FastIllustParams

// RecentType represents the content type for which to retrieve recent items
type RecentType string

const (
	RecentTypeArtwork RecentType = "artwork"
	RecentTypeNovel   RecentType = "novel"
)

type RecentParams struct {
	Type          RecentType
	UserID        string
	RecentWorkIDs []int
}

// RelatedType represents the content type for which to retrieve related items
type RelatedType string

const (
	RelatedTypeArtwork RelatedType = "artwork"
	RelatedTypeNovel   RelatedType = "novel"
)

type RelatedParams struct {
	Type  RelatedType
	ID    string
	Limit int
}

// CommentType represents the content type for which to retrieve comments
type CommentType string

const (
	CommentTypeArtwork CommentType = "artwork"
	CommentTypeNovel   CommentType = "novel"
)

type CommentsParams struct {
	Type        CommentType
	ID          string
	UserID      string
	SanityLevel core.SanityLevel // For type=artworks
	XRestrict   core.XRestrict   // For type=novel
}

func ArtworkPartial(w http.ResponseWriter, r *http.Request) error {
	var illust core.Illust

	id := GetQueryParam(r, "id")
	userid := GetQueryParam(r, "userid")
	illustType, err := strconv.Atoi(GetQueryParam(r, "illusttype"))
	if err != nil {
		return err
	}
	pages, err := strconv.Atoi(GetQueryParam(r, "pages"))
	if err != nil {
		return err
	}
	webpURL := GetQueryParam(r, "webpurl")

	params := ArtworkParams{
		ID:         id,
		UserID:     userid,
		IllustType: core.IllustType(illustType),
		Pages:      pages,
		WebpURL:    webpURL,
	}

	data, err := core.GetArtworkFast(w, r, core.FastIllustParams(params))
	if err != nil {
		return err
	}
	illust = *data

	if session.GetUserToken(r) != "" {
		w.Header().Set("Cache-Control", "private, max-age=60")
	} else {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, stale-while-revalidate=%d",
			int(config.GlobalConfig.HTTPCache.MaxAge.Seconds()),
			int(config.GlobalConfig.HTTPCache.StaleWhileRevalidate.Seconds())))
	}

	return template.RenderHTML(w, r, Data_artworkPartial{
		Illust: illust,
	})
}

func RecentPartial(w http.ResponseWriter, r *http.Request) error {
	var (
		illust core.Illust
		novel  core.Novel
	)

	recentType := RecentType(GetQueryParam(r, "type"))
	userid := GetQueryParam(r, "userid")
	recentWorkIDsStr := GetQueryParam(r, "recentworkids")

	// Parse the recent work IDs
	recentWorkIDs, err := parseWorkIDs(recentWorkIDsStr)
	if err != nil {
		return err
	}

	params := RecentParams{
		Type:          recentType,
		UserID:        userid,
		RecentWorkIDs: recentWorkIDs,
	}

	switch params.Type {
	case RecentTypeArtwork:
		related, err := core.PopulateArtworkRecent(r, params.UserID, params.RecentWorkIDs)
		if err != nil {
			return err
		}
		illust.RecentWorks = related

	case RecentTypeNovel:
		return fmt.Errorf("novel related content not yet supported")

	default:
		return fmt.Errorf("unsupported related type: %s", params.Type)
	}

	if session.GetUserToken(r) != "" {
		w.Header().Set("Cache-Control", "private, max-age=60")
	} else {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, stale-while-revalidate=%d",
			int(config.GlobalConfig.HTTPCache.MaxAge.Seconds()),
			int(config.GlobalConfig.HTTPCache.StaleWhileRevalidate.Seconds())))
	}

	return template.RenderHTML(w, r, Data_recentPartial{
		Illust: illust,
		Novel:  novel,
	})
}

func RelatedPartial(w http.ResponseWriter, r *http.Request) error {
	var (
		illust core.Illust
		novel  core.Novel
	)

	relatedType := RelatedType(GetQueryParam(r, "type"))
	id := GetQueryParam(r, "id")
	intLimit, err := strconv.Atoi(GetQueryParam(r, "limit"))
	if err != nil {
		return err
	}

	params := RelatedParams{
		Type:  relatedType,
		ID:    id,
		Limit: intLimit,
	}

	switch params.Type {
	case RelatedTypeArtwork:
		related, err := core.GetArtworkRelated(r, params.ID, params.Limit)
		if err != nil {
			return err
		}
		illust.RelatedWorks = related

	case RelatedTypeNovel:
		return fmt.Errorf("novel related content not yet supported")

	default:
		return fmt.Errorf("unsupported related type: %s", params.Type)
	}

	if session.GetUserToken(r) != "" {
		w.Header().Set("Cache-Control", "private, max-age=60")
	} else {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, stale-while-revalidate=%d",
			int(config.GlobalConfig.HTTPCache.MaxAge.Seconds()),
			int(config.GlobalConfig.HTTPCache.StaleWhileRevalidate.Seconds())))
	}

	return template.RenderHTML(w, r, Data_relatedPartial{
		Illust: illust,
		Novel:  novel,
	})
}

func CommentsPartial(w http.ResponseWriter, r *http.Request) error {
	var commentsData core.CommentsData

	commentType := CommentType(GetQueryParam(r, "type"))
	id := GetQueryParam(r, "id")
	userid := GetQueryParam(r, "userid")

	// use most relaxed constraint by default
	sanityLevel := core.SLR18
	if sl := GetQueryParam(r, "sanitylevel"); sl != "" {
		x, err := strconv.Atoi(sl)
		if err != nil {
			return err
		} else {
			sanityLevel = core.SanityLevel(x)
		}
	}

	// use most relaxed constraint by default
	xRestrict := core.R18G
	if xr := GetQueryParam(r, "xrestrict"); xr != "" {
		x, err := strconv.Atoi(xr)
		if err != nil {
			return err
		} else {
			xRestrict = core.XRestrict(x)
		}
	}

	params := CommentsParams{
		Type:        commentType,
		ID:          id,
		UserID:      userid,
		SanityLevel: sanityLevel,
		XRestrict:   xRestrict,
	}

	switch params.Type {
	case CommentTypeArtwork:
		artworkParams := core.ArtworkCommentsParams{
			ID:          params.ID,
			UserID:      params.UserID,
			SanityLevel: params.SanityLevel,
		}

		data, _, err := core.GetArtworkComments(r, artworkParams)
		if err != nil {
			return err
		}

		commentsData = data
	case CommentTypeNovel:
		artworkParams := core.NovelCommentsParams{
			ID:        params.ID,
			UserID:    params.UserID,
			XRestrict: params.XRestrict,
		}

		data, _, err := core.GetNovelComments(r, artworkParams)
		if err != nil {
			return err
		}

		commentsData = data
	default:
		return fmt.Errorf("unsupported comment type: %s", params.Type)
	}

	if session.GetUserToken(r) != "" {
		w.Header().Set("Cache-Control", "private, max-age=60")
	} else {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, stale-while-revalidate=%d",
			int(config.GlobalConfig.HTTPCache.MaxAge.Seconds()),
			int(config.GlobalConfig.HTTPCache.StaleWhileRevalidate.Seconds())))
	}

	return template.RenderHTML(w, r, Data_commentsPartial{
		CommentsData: commentsData,
	})
}

func DiscoveryPartial(w http.ResponseWriter, r *http.Request) error {
	mode := GetQueryParam(r, "mode", "safe")

	data, err := core.GetDiscoveryArtwork(r, mode)
	if err != nil {
		return err
	}

	w.Header().Set("Cache-Control", "no-store")

	return template.RenderHTML(w, r, Data_discoveryArtworkPartial{
		Artworks: data,
	})
}

func parseWorkIDs(s string) ([]int, error) {
	// Handle empty string
	if strings.TrimSpace(s) == "" {
		return []int{}, nil
	}

	// Split string on commas
	strIDs := strings.Split(s, ",")

	// Create result slice
	ids := make([]int, 0, len(strIDs))

	// Convert each string to int
	for _, str := range strIDs {
		// Trim any whitespace
		str = strings.TrimSpace(str)

		id, err := strconv.Atoi(str)
		if err != nil {
			return nil, fmt.Errorf("invalid work ID '%s': %w", str, err)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// func UgoiraPreview(w http.ResponseWriter, r *http.Request) error {
// 	id := GetPathVar(r, "id")
// 	if _, err := strconv.Atoi(id); err != nil {
// 		return i18n.Errorf("Invalid ID: %s", id)
// 	}

// 	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, stale-while-revalidate=%d",
// 		int(config.GlobalConfig.CacheControlMaxAge.Seconds()),
// 		int(config.GlobalConfig.CacheControlStaleWhileRevalidate.Seconds())))

// 	// If trigger starts with thumbnail-restore, always show thumbnail regardless of target (mouseleave case)
// 	trigger := r.Header.Get("HX-Trigger")
// 	if strings.HasPrefix(trigger, "thumbnail-restore") {
// 		thumbnailURL := r.Header.Get("Thumbnail-Url")
// 		return template.RenderHTML(w, r, Data_ugoiraThumbnail{
// 			ID:           id,
// 			ThumbnailURL: thumbnailURL,
// 		})
// 	}

// 	target := r.Header.Get("HX-Target")

// 	// If target starts with swapped-media-video, show thumbnail (mouseleave case)
// 	if strings.HasPrefix(target, "swapped-media-video") {
// 		thumbnailURL := r.Header.Get("Thumbnail-Url")
// 		return template.RenderHTML(w, r, Data_ugoiraThumbnail{
// 			ID:           id,
// 			ThumbnailURL: thumbnailURL,
// 		})
// 	}

// 	// If target starts with original-media or swapped-media-thumbnail, show video (mouseenter case)
// 	// This also handles the default case
// 	return template.RenderHTML(w, r, Data_ugoiraPreview{
// 		ID: id,
// 	})
// }
