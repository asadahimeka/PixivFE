// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"fmt"
	"net/http"

	"github.com/goccy/go-json"
	"github.com/tidwall/gjson"

	"codeberg.org/pixivfe/pixivfe/audit"
	"codeberg.org/pixivfe/pixivfe/core/requests"
	"codeberg.org/pixivfe/pixivfe/server/session"
)

// Pages is a helper struct for parsing Pixiv's API
// Each field should represent a section of the index page
// Some fields' type can be a separate struct.
// Structs made specifically for this struct must all have the suffix "SN" to distinguish themselves from other structs
type Pages struct {
	Pixivision       []Pixivision       `json:"pixivision"`
	Follow           []int              `json:"follow"`
	Recommended      RecommendedSN      `json:"recommend"`
	RecommendedByTag []RecommendByTagSN `json:"recommendByTag"`
	RecommendUser    []RecommendUserSN  `json:"recommendUser"`
	TrendingTag      []TrendingTagSN    `json:"trendingTags"`
	Newest           []string           `json:"newPost"`

	// Commented out fields that aren't currently implemented in the frontend
	// EditorRecommended []any `json:"editorRecommend"`
	// RecommendedUsers   []RecommendedUser `json:"recommendUser"`
	// Commission        []any `json:"completeRequestIds"`
	// BoothFollows []BoothFollow `json:"boothFollowItemIds"`
	// OngoingContests []Contest `json:"contestOngoing"`
	// ContestResult []Contest `json:"contestResult"`
	// FavoriteTags []any `json:"myFavoriteTags"`
	// MyPixiv []any `json:"mypixiv"`
	//"ranking",
	//"sketchLiveFollowIds",
	//"sketchLivePopularIds",
	//"tags",
	//"trendingTags",
	//"userEventIds"

	// These aren't included as pages
	// "requests"
	//"illustSeries"
	// SketchLives []SketchLive `json:"sketchLives"`
	// BoothItems []BoothItem `json:"boothItems"`
}

type TrendingTagSN struct {
	Name         string `json:"tag"`
	TrendingRate int    `json:"trendingRate"`
	IDs          []int  `json:"ids"`
}

type RecommendedSN struct {
	IDs []string `json:"ids"`
}

type RecommendByTagSN struct {
	Name string   `json:"tag"`
	IDs  []string `json:"ids"`
}

type RequestSN struct {
	RequestID       string   `json:"requestId"`
	PlanID          string   `json:"planId"`
	CreatorUserID   string   `json:"creatorUserId"`
	RequestTags     []string `json:"requestTags"`
	RequestProposal struct {
		RequestProposalHTML string `json:"requestOriginalProposalHtml"`
	}
	PostWork struct {
		PostWorkID string `json:"postWorkId"`
	} `json:"postWork"`
}

// Pixivision represents a Pixivision article as returned by the landing page endpoint.
//
// Note that this type is less complete than the PixivisionArticle type.
type Pixivision struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Thumbnail string `json:"thumbnailUrl"`
	URL       string `json:"url"`
}

// RecommendUser represents a recommended user as returned by the landing page endpoint.
//
// This type is distinct from the User type.
type RecommendUserSN struct {
	ID        int      `json:"id"`
	IllustIDs []string `json:"illustIds"`
	NovelIDs  []string `json:"novelIds"` // Unimplemented
}

// RecommendedTags groups artworks under a specific tag recommendation.
type RecommendedTags struct {
	Name     string
	Artworks []ArtworkBrief
}

type PopularTag struct {
	Name         string
	TrendingRate int
	IDs          []int
	Artworks     []ArtworkBrief
}

type Request struct {
	Description string
	Tags        []string
	Artwork     ArtworkBrief
}

// LandingArtworks aggregates various categories of artworks and other related data
// to be displayed on the landing page.
type LandingArtworks struct {
	Commissions     []ArtworkBrief
	Following       []ArtworkBrief
	Recommended     []ArtworkBrief
	Newest          []ArtworkBrief
	Rankings        Ranking
	Users           []ArtworkBrief
	Pixivision      []Pixivision
	RecommendByTags []RecommendedTags
	RecommendUser   []User
	PopularTag      []PopularTag
	Requests        []Request
}

// Constants for calling GetNewestFromFollowing when len(followIDs) == 0
const (
	// FollowingMode = "all"
	FollowingPage = "1"
)

// landingToRankingMode maps landing page modes to their corresponding ranking modes.
var landingToRankingMode = map[string]string{
	"all": "daily",
	"r18": "daily_r18",
}

// GetLanding retrieves and organizes the landing page data based on the provided mode and user login status.
//
// It fetches raw data from the landing URL, parses the JSON response, and populates the LandingArtworks struct.
func GetLanding(r *http.Request, mode string, isLoggedIn bool) (*LandingArtworks, error) {
	url := GetLandingURL(mode)

	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	var pages Pages

	resp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
	if err != nil {
		return nil, err
	}

	resp = RewriteContentURLs(r, resp)

	artworks, err := parseArtworks(resp)
	if err != nil {
		return nil, err
	}

	users, err := parseUsers(resp)
	if err != nil {
		return nil, err
	}

	// Populate thumbnails for each artwork
	for id, artwork := range artworks {
		if err := artwork.PopulateThumbnails(); err != nil {
			audit.GlobalAuditor.Logger.Errorf("Failed to populate thumbnails for artwork ID %s: %v", id, err)
			return nil, fmt.Errorf("failed to populate thumbnails for artwork ID %s: %w", id, err)
		}
		artworks[id] = artwork
	}

	pagesResp := gjson.GetBytes(resp, "page").Raw

	if err := json.Unmarshal([]byte(pagesResp), &pages); err != nil {
		return nil, err
	}

	landing := LandingArtworks{
		Pixivision: pages.Pixivision,
	}

	// If the user is logged in, populate personalized sections
	if isLoggedIn {
		landing.Following, err = populateFollowing(r, mode, pages.Follow, artworks)
		if err != nil {
			return nil, err
		}

		landing.RecommendByTags = populateRecommendedByTags(pages.RecommendedByTag, artworks)
		landing.Recommended = populateArtworks(pages.Recommended.IDs, artworks)
		landing.RecommendUser = populateRecommendUsers(pages.RecommendUser, users, artworks)
		landing.PopularTag = populatePopularTags(pages.TrendingTag, artworks)

		landing.Requests, err = parsePixivRequests(resp, artworks)
		if err != nil {
			return nil, err
		}
	}

	landing.Rankings, err = fetchRankings(r, mode)
	// landing.Newest = populateArtworks(pages.Newest, artworks)
	if err != nil {
		return nil, err
	}

	return &landing, nil
}

// parseArtworks extracts artwork information from the "thumbnails.illust" field
// of the JSON response and maps them by their ID.
func parseArtworks(resp []byte) (map[string]ArtworkBrief, error) {
	artworks := make(map[string]ArtworkBrief)

	stuff := gjson.GetBytes(resp, "thumbnails.illust")

	// Iterate over each thumbnail entry and unmarshal it into the ArtworkBrief struct
	stuff.ForEach(func(_, value gjson.Result) bool {
		var artwork ArtworkBrief
		if err := json.Unmarshal([]byte(value.Raw), &artwork); err != nil {
			// Invalid artwork entries are simply skipped
			return false
		}

		// Map the artwork by its ID for easy access later
		if artwork.ID != "" {
			artworks[artwork.ID] = artwork
		}

		return true // Continue iteration
	})

	return artworks, nil
}

// parseUsers extracts user information from the "users" field
// of the JSON response and maps them by their ID.
func parseUsers(resp []byte) (map[string]User, error) {
	users := make(map[string]User)

	stuff := gjson.GetBytes(resp, "users")

	// Iterate over each user entry and unmarshal it into the User struct
	stuff.ForEach(func(_, value gjson.Result) bool {
		var user User
		if err := json.Unmarshal([]byte(value.String()), &user); err != nil {
			// Invalid user entries are simply skipped
			return false
		}

		// Map the user by its ID for easy access later
		if user.ID != "" {
			users[user.ID] = user
		}

		return true // Continue iteration
	})

	return users, nil
}

// populateArtworks is a generic helper function that maps a slice of string IDs
// to their corresponding ArtworkBrief objects.
func populateArtworks(ids []string, artworks map[string]ArtworkBrief) []ArtworkBrief {
	populated := make([]ArtworkBrief, 0, len(ids))

	for _, id := range ids {
		if artwork, exists := artworks[id]; exists {
			populated = append(populated, artwork)
		}
	}

	return populated
}

// populateFollowing converts int IDs to strings and uses the generic helper.
//
// If followIDs is empty, it attempts population by calling GetNewestFromFollowing instead (pixiv moment).
func populateFollowing(r *http.Request, mode string, followIDs []int, artworks map[string]ArtworkBrief) ([]ArtworkBrief, error) {
	if len(followIDs) == 0 {
		data, err := GetNewestFromFollowing(r, "illust", mode, FollowingPage)
		if err != nil {
			return nil, err
		}

		works := data.Body.Thumbnails.Illust

		return works, nil
	}

	stringIDs := make([]string, len(followIDs))
	for i, id := range followIDs {
		stringIDs[i] = fmt.Sprint(id)
	}

	return populateArtworks(stringIDs, artworks), nil
}

// populateRecommendedByTags uses the generic helper for each tag's IDs.
func populateRecommendedByTags(recommends []RecommendByTagSN, artworks map[string]ArtworkBrief) []RecommendedTags {
	recommendByTags := make([]RecommendedTags, 0, len(recommends))

	for _, recommend := range recommends {
		artworksList := populateArtworks(recommend.IDs, artworks)
		recommendByTags = append(recommendByTags, RecommendedTags{
			Name:     recommend.Name,
			Artworks: artworksList,
		})
	}

	return recommendByTags
}

// populateRecommendUsers is a generic helper function.
func populateRecommendUsers(recommendUsers []RecommendUserSN, users map[string]User, artworks map[string]ArtworkBrief) []User {
	populated := make([]User, 0, len(recommendUsers))

	for _, recommendUser := range recommendUsers {
		idStr := fmt.Sprint(recommendUser.ID)
		if user, exists := users[idStr]; exists {
			// Populate the user's artworks using their illustIds
			user.Artworks = populateArtworks(recommendUser.IllustIDs, artworks)
			populated = append(populated, user)
		}
	}

	return populated
}

func populatePopularTags(tags []TrendingTagSN, artworks map[string]ArtworkBrief) []PopularTag {
	populated := make([]PopularTag, 0, len(tags))

	for _, tag := range tags {
		ids := make([]string, 0, 3)
		for _, workID := range tag.IDs {
			ids = append(ids, fmt.Sprint(workID))
		}

		artworksList := populateArtworks(ids, artworks)
		populated = append(populated, PopularTag{
			Name:         tag.Name,
			TrendingRate: tag.TrendingRate,
			Artworks:     artworksList,
		})
	}

	return populated
}

func parsePixivRequests(resp []byte, artworks map[string]ArtworkBrief) ([]Request, error) {
	var requests []RequestSN
	pixivRequests := gjson.GetBytes(resp, "requests").Raw

	if err := json.Unmarshal([]byte(pixivRequests), &requests); err != nil {
		return nil, err
	}

	populated := make([]Request, 0, len(requests))

	for _, request := range requests {
		desc := request.RequestProposal.RequestProposalHTML
		tags := request.RequestTags
		artwork := artworks[request.PostWork.PostWorkID]

		populated = append(populated, Request{
			Description: desc,
			Tags:        tags,
			Artwork:     artwork,
		})
	}

	return populated, nil
}

// fetchRankings retrieves the current rankings based on the selected mode.
//
// It maps the landing page mode to the appropriate ranking mode and fetches the ranking data.
func fetchRankings(r *http.Request, mode string) (Ranking, error) {
	rankingMode, exists := landingToRankingMode[mode]
	if !exists {
		rankingMode = "daily"
	}
	return GetRanking(r, rankingMode, "illust", "", "1")
}
