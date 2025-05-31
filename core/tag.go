// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"fmt"
	"net/http"
	"strings"

	"codeberg.org/pixivfe/pixivfe/audit"
	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/core/requests"
	"codeberg.org/pixivfe/pixivfe/i18n"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"github.com/goccy/go-json"
)

const (
	usersPerPage int = 10 // The pixiv API returns 10 at a time
)

// TagSearchResult defines the API response structure for /ajax/search/tags
type TagSearchResult struct {
	Name            string `json:"tag"`
	AlternativeName string `json:"word"`
	Metadata        struct {
		Detail string      `json:"abstract"`
		Image  string      `json:"image"`
		Name   string      `json:"tag"`
		ID     json.Number `json:"id"`
	} `json:"pixpedia"`
	CoverArtwork Illust // Custom field to store extended info about the cover artwork
}

// ArtworkSearchResponse defines the API response structure for /ajax/search/[category]
//
// TODO: the name for this type is awful
type ArtworkSearchResponse struct {
	IllustManga   ArtworkResults `json:"illustManga"` // Populated when category is "artworks"
	Illustrations ArtworkResults `json:"illust"`      // Populated when category is "illustrations"
	Manga         ArtworkResults `json:"manga"`       // Populated when category is "manga"
	Novels        struct {
		Data     []NovelBrief `json:"data"`
		Total    int          `json:"total"`
		LastPage int          `json:"lastPage"`
	} `json:"novel"` // Populated when category is "novel"
	Users struct {
		Data     []User // custom
		Total    int    // custom
		LastPage int    // custom
	} // custom
	Popular struct {
		Permanent []ArtworkBrief `json:"permanent"`
		Recent    []ArtworkBrief `json:"recent"`
	} `json:"popular"`
	TagTranslation TagTranslationWrapper `json:"tagTranslation"`
	RawRelatedTags []string              `json:"relatedTags"`
	RelatedTags    []Tag
	Total          int // custom
	LastPage       int // custom
}

// TODO: come up with a better name for this type
type ArtworkResults struct {
	Data     []ArtworkBrief `json:"data"`
	Total    int            `json:"total"`
	LastPage int            `json:"lastPage"`
}

// UserSearchResponse defines the API response structure for /ajax/search/users
type UserSearchResponse struct {
	Data []any `json:"data"` // NOTE: this was an empty array in the response analyzed
	Page struct {
		UserIDs []int                 `json:"userIds"`
		WorkIDs map[string][]WorkItem `json:"workIds"` // Key is userID (string)
		Total   int                   `json:"total"`
	} `json:"page"` // Page holds user IDs and their associated work IDs.
	TagTranslation TagTranslationWrapper `json:"tagTranslation"`
	Thumbnails     struct {
		Illust      []ArtworkBrief `json:"illust"`
		Novel       []NovelBrief   `json:"novel"`
		NovelSeries []any          `json:"novelSeries"` // FIXME: our NovelSeries type might be correct, not sure
		NovelDraft  []any          `json:"novelDraft"`  // We don't have a type
		Collection  []any          `json:"collection"`  // We don't have a type
	} `json:"thumbnails"`
	IllustSeries []IllustSeries `json:"illustSeries"`
	Requests     []any          `json:"requests"` // We don't have a type
	Users        []User         `json:"users"`
	// ZoneConfig   any            `json:"zoneConfig"` // NOTE: not implemented, these are advertisements
}

// WorkItem represents a single work item (illust or novel) associated with a user.
type WorkItem struct {
	ID        string `json:"id"`
	Type      string `json:"type"`       // NOTE: observed values were "illust" and "novel"
	CreatedAt string `json:"created_at"` // NOTE: parseable as time.Time
}

type ArtworkSearchSettings struct {
	Name     string // Keywords to search for
	Word     string // Mirror of the Name field, required for multiple keywords to work
	Category string // Filter by type, could be illustrations or manga
	Order    string // Sort by date
	Mode     string // Safe, R18 or both
	Ratio    string // Landscape, portrait, or squared
	Page     string // Page number
	Smode    string // Exact match, partial match, or match with title
	Wlt      string // Minimum image width
	Wgt      string // Maximum image width
	Hlt      string // Minimum image height
	Hgt      string // Maximum image height
	Tool     string // Filter by production tools (e.g. Photoshop)
	Scd      string // After this date
	Ecd      string // Before this date
}

// # Returns
// Some fields may not exist, but m[key] will return "" anyway
func (s ArtworkSearchSettings) ReturnMap() map[string]string {
	m := map[string]string{}
	SetIfNotEmpty(m, "Name", s.Name)
	SetIfNotEmpty(m, "Category", s.Category)
	SetIfNotEmpty(m, "Order", s.Order)
	SetIfNotEmpty(m, "Mode", s.Mode)
	SetIfNotEmpty(m, "Ratio", s.Ratio)
	SetIfNotEmpty(m, "Smode", s.Smode)
	SetIfNotEmpty(m, "Wlt", s.Wlt)
	SetIfNotEmpty(m, "Wgt", s.Wgt)
	SetIfNotEmpty(m, "Hlt", s.Hlt)
	SetIfNotEmpty(m, "Hgt", s.Hgt)
	SetIfNotEmpty(m, "Scd", s.Scd)
	SetIfNotEmpty(m, "Ecd", s.Ecd)
	SetIfNotEmpty(m, "Tool", s.Tool)
	SetIfNotEmpty(m, "Page", s.Page)
	return m
}

// this is so that the query string won't be long af
func SetIfNotEmpty(m map[string]string, key, val string) {
	if val != "" {
		m[key] = val
	}
}

func GetTagData(r *http.Request, name string) (TagSearchResult, error) {
	var tag TagSearchResult

	url := GetTagDetailURL(name)

	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	rawResp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
	if err != nil {
		return tag, err
	}

	rawResp = RewriteContentURLs(r, rawResp)

	err = json.Unmarshal(rawResp, &tag)
	if err != nil {
		return tag, err
	}

	return tag, nil
}

// GetSearch delegates the search operation to either getPopularSearch or getStandardSearch based on settings.Order.
func GetSearch(r *http.Request, settings ArtworkSearchSettings) (*ArtworkSearchResponse, error) {
	if strings.ToLower(settings.Order) == "popular" {
		return getPopularSearch(r, settings)
	}
	return getStandardSearch(r, settings)
}

// getStandardSearch handles the standard search logic.
func getStandardSearch(r *http.Request, settings ArtworkSearchSettings) (*ArtworkSearchResponse, error) {
	url := GetArtworkSearchURL(settings.ReturnMap())

	// p_ab := session.GetCookie(r, session.Cookie_P_AB)
	// if p_ab == "" {
	p_ab := config.GlobalConfig.GetP_AB()
	//}

	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
		"p_ab_d_id": p_ab,
		"p_ab_id":   "8",
		"p_ab_id_2": "4",
	}

	resp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
	if err != nil {
		return nil, err
	}

	resp = RewriteContentURLs(r, resp)

	var result ArtworkSearchResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal search response: %w", err)
	}

	// Convert string tags to Tag objects with translations
	result.RelatedTags = TagTranslationsToTags(result.RawRelatedTags, result.TagTranslation)

	// Process thumbnails for popular artworks
	for i := range result.Popular.Permanent {
		if err := result.Popular.Permanent[i].PopulateThumbnails(); err != nil {
			return nil, fmt.Errorf("failed to populate thumbnails for popular permanent artwork %d: %w", i, err)
		}
	}

	for i := range result.Popular.Recent {
		if err := result.Popular.Recent[i].PopulateThumbnails(); err != nil {
			return nil, fmt.Errorf("failed to populate thumbnails for popular recent artwork %d: %w", i, err)
		}
	}

	// Process data based on category and set top-level Total and LastPage
	switch settings.Category {
	case "artworks":
		// Process thumbnails for artworks
		for i := range result.IllustManga.Data {
			if err := result.IllustManga.Data[i].PopulateThumbnails(); err != nil {
				return nil, fmt.Errorf("failed to populate thumbnails for artwork %d: %w", i, err)
			}
		}
		result.Total = result.IllustManga.Total
		result.LastPage = result.IllustManga.LastPage

	case "illustrations":
		// Process thumbnails for illustrations
		for i := range result.Illustrations.Data {
			if err := result.Illustrations.Data[i].PopulateThumbnails(); err != nil {
				return nil, fmt.Errorf("failed to populate thumbnails for illustration %d: %w", i, err)
			}
		}
		result.Total = result.Illustrations.Total
		result.LastPage = result.Illustrations.LastPage

	case "manga":
		// Process thumbnails for manga
		for i := range result.Manga.Data {
			if err := result.Manga.Data[i].PopulateThumbnails(); err != nil {
				return nil, fmt.Errorf("failed to populate thumbnails for manga %d: %w", i, err)
			}
		}
		result.Total = result.Manga.Total
		result.LastPage = result.Manga.LastPage

	case "novels":
		// No thumbnail processing needed for novels
		result.Total = result.Novels.Total
		result.LastPage = result.Novels.LastPage

	default:
		return nil, fmt.Errorf("invalid category: %s", settings.Category)
	}

	return &result, nil
}

// getPopularSearch handles the popular search logic.
func getPopularSearch(r *http.Request, settings ArtworkSearchSettings) (*ArtworkSearchResponse, error) {
	// Check if popular search is enabled
	if !config.GlobalConfig.Feature.PopularSearchEnabled {
		return nil, i18n.Errorf("Popular search is disabled by server configuration.")
	}

	// Perform popular search
	searchArtworks, err := searchPopular(r.Context(), r, settings)
	if err != nil {
		return nil, err
	}

	// Create SearchResult
	result := &ArtworkSearchResponse{
		IllustManga: searchArtworks,
		Total:       searchArtworks.Total,
		LastPage:    searchArtworks.LastPage,
		// TODO: populate Popular (the regular one) and RelatedTags
	}

	// Populate thumbnails for each artwork
	for id, artwork := range result.IllustManga.Data {
		if err := artwork.PopulateThumbnails(); err != nil {
			audit.GlobalAuditor.Logger.Errorf("Failed to populate thumbnails for artwork ID %d: %v", id, err)
			return nil, fmt.Errorf("failed to populate thumbnails for artwork ID %d: %w", id, err)
		}
		result.IllustManga.Data[id] = artwork
	}

	return result, nil
}

// GetUserSearch retrieves user search results and converts to ArtworkSearchResponse format.
func GetUserSearch(r *http.Request, settings ArtworkSearchSettings) (*ArtworkSearchResponse, error) {
	url := GetUserSearchURL(settings.Name, settings.Page)

	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	resp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
	if err != nil {
		return nil, err
	}

	resp = RewriteContentURLs(r, resp)

	var userResult UserSearchResponse
	if err := json.Unmarshal(resp, &userResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user search response: %w", err)
	}

	// Process thumbnails for users' works
	for i := range userResult.Thumbnails.Illust {
		if err := userResult.Thumbnails.Illust[i].PopulateThumbnails(); err != nil {
			return nil, fmt.Errorf("failed to populate thumbnails for user artwork %d: %w", i, err)
		}
	}

	// Create users array
	users := make([]User, len(userResult.Users))
	copy(users, userResult.Users)

	// Associate artworks and novels with their respective users
	AssociateContentWithUsers(&users, userResult.Thumbnails.Illust, userResult.Thumbnails.Novel)

	// Calculate last page based on total count
	lastPage := (userResult.Page.Total + 9) / usersPerPage

	// Create the ArtworkSearchResponse
	result := &ArtworkSearchResponse{
		Users: struct {
			Data     []User
			Total    int
			LastPage int
		}{
			Data:     users,
			Total:    userResult.Page.Total,
			LastPage: lastPage,
		},
		TagTranslation: userResult.TagTranslation,
		Total:          userResult.Page.Total,
		LastPage:       lastPage,
	}

	return result, nil
}
