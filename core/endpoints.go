// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"fmt"
	"log"
	"net/url"

	"github.com/goccy/go-json"
)

const (
	DiscoveryLimit      = 60
	DiscoveryNovelLimit = 24
	DiscoveryUserLimit  = 12

	BookmarksPageSize       = 48 // For both illustrations and novels
	UserFollowersPageSize   = 100
	ArtworkCommentsPageSize = 1000
	NovelCommentsPageSize   = 1000
)

// API responses

type NewestFromFollowingResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	Body    struct {
		Page struct {
			IDs        []int `json:"ids"`
			IsLastPage bool  `json:"isLastPage"`
			Tags       []Tag `json:"-"` // a "tags" field exists in the response, but isn't populated
		} `json:"page"`
		TagTranslation TagTranslationWrapper `json:"tagTranslation"`
		Thumbnails     struct {
			Illust      []ArtworkBrief `json:"illust"`
			Novel       []NovelBrief   `json:"novel"`
			NovelSeries []NovelSeries  `json:"novelSeries"`
			NovelDraft  []any          `json:"novelDraft"`
			Collection  []any          `json:"collection"`
		} `json:"thumbnails"`
		IllustSeries []IllustSeries `json:"illustSeries"`
		Requests     []any          `json:"requests"`
		Users        []User         `json:"users"`
	} `json:"body"`
}

// ArtworkRelatedResponse represents the API response for GetArtworkRelatedURL
type ArtworkRelatedResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	Body    struct {
		Illusts []ArtworkBrief `json:"illusts"`
		NextIds []string       `json:"nextIds"`
		Details OptionalStrMap[struct {
			Methods         []string      `json:"methods"`
			Score           float64       `json:"score"`
			SeedIllustIds   []json.Number `json:"seedIllustIds"`
			BanditInfo      string        `json:"banditInfo"`
			RecommendListId string        `json:"recommendListId"`
		}] `json:"details"`
	} `json:"body"`
}

// UserFollowingResponse represents the API response for GetUserFollowingURL
type UserFollowingResponse struct {
	Users []struct {
		UserId          string         `json:"userId"`
		UserName        string         `json:"userName"`
		ProfileImageUrl string         `json:"profileImageUrl"`
		UserComment     string         `json:"userComment"`
		Following       bool           `json:"following"`
		Followed        bool           `json:"followed"`
		IsBlocking      bool           `json:"isBlocking"`
		IsMypixiv       bool           `json:"isMypixiv"`
		Illusts         []ArtworkBrief `json:"illusts"`
	} `json:"users"`
	Total int `json:"total"`
}

// AddIllustBookmarkResponse represents the API response for PostAddIllustBookmarkURL
type AddIllustBookmarkResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	Body    struct {
		LastBookmarkID string `json:"last_bookmark_id"`
		StaccStatusID  string `json:"stacc_status_id"`
	} `json:"body"`
}

// FollowResponse represents the API response for a follow (mobile API)
type FollowResponse struct {
	IsSucceed bool `json:"isSucceed"`
}

// GET endpoints

func GetNewestArtworksURL(workType, r18, lastID string) string {
	base := "https://www.pixiv.net/ajax/illust/new?limit=30&type=%s&r18=%s&lastId=%s"
	return fmt.Sprintf(base, workType, r18, lastID)
}

func GetDiscoveryURL(mode string, limit int) string {
	base := "https://www.pixiv.net/ajax/discovery/artworks?mode=%s&limit=%d"
	return fmt.Sprintf(base, mode, limit)
}

func GetDiscoveryNovelURL(mode string, limit int) string {
	base := "https://www.pixiv.net/ajax/discovery/novels?mode=%s&limit=%d"
	return fmt.Sprintf(base, mode, limit)
}

func GetDiscoveryUserURL(limit int) string {
	base := "https://www.pixiv.net/ajax/discovery/users?limit=%d"
	return fmt.Sprintf(base, limit)
}

func GetRankingURL(mode, contentType, date, page string) string {
	params := url.Values{}
	params.Add("mode", mode)
	params.Add("type", contentType)
	params.Add("page", page)

	if date != "" {
		params.Add("date", date)
	}
	return fmt.Sprintf("https://www.pixiv.net/touch/ajax/ranking/illust?%s", params.Encode())
}

func GetIllustDetailsManyURL(illustIDs []string) string {
	params := url.Values{}

	for _, id := range illustIDs {
		params.Add("illust_ids[]", id)
	}

	return "https://www.pixiv.net/touch/ajax/illust/details/many?" + params.Encode()
}

func GetRankingCalendarURL(mode string, year, month int) string {
	base := "https://www.pixiv.net/ranking_log.php?mode=%s&date=%d%02d"

	return fmt.Sprintf(base, mode, year, month)
}

func GetUserInformationURL(userID, full string) string {
	base := "https://www.pixiv.net/ajax/user/%s?full=%s"

	return fmt.Sprintf(base, userID, full)
}

func GetUserWorksURL(userID string) string {
	base := "https://www.pixiv.net/ajax/user/%s/profile/all"

	return fmt.Sprintf(base, userID)
}

func GetUserFullArtworkURL(userIDs, illustIDs string) string {
	base := "https://www.pixiv.net/ajax/user/%s/profile/illusts?work_category=illustManga&is_first_page=0&lang=en%s"

	return fmt.Sprintf(base, userIDs, illustIDs)
}

func GetUserFullNovelURL(userID, novelIDs string) string {
	base := "https://www.pixiv.net/ajax/user/%s/profile/novels?is_first_page=0&lang=en%s"

	return fmt.Sprintf(base, userID, novelIDs)
}

func GetUserIllustBookmarksURL(userID, mode string, page int) string {
	base := "https://www.pixiv.net/ajax/user/%s/illusts/bookmarks?tag=&offset=%d&limit=48&rest=%s"

	return fmt.Sprintf(base, userID, page*BookmarksPageSize, mode)
}

func GetUserNovelBookmarksURL(userID, mode string, page int) string {
	base := "https://www.pixiv.net/ajax/user/%s/novels/bookmarks?tag=&offset=%d&limit=48&rest=%s"

	return fmt.Sprintf(base, userID, page*BookmarksPageSize, mode)
}

func GetArtworkFrequentTagsURL(illustIDs string) string {
	base := "https://www.pixiv.net/ajax/tags/frequent/illust?%s"

	return fmt.Sprintf(base, illustIDs)
}

func GetNovelFrequentTagsURL(novelIDs string) string {
	base := "https://www.pixiv.net/ajax/tags/frequent/novel?%s"

	return fmt.Sprintf(base, novelIDs)
}

// Retrieves the users followed by a given user
//
// The mode parameter controls whether follows that are public ("show")
// or private ("hide") are retrieved
//
// Attempting to retrieve private follows for a user other than the one
// for which a PHPSESSID is provided returns their public follows instead
func GetUserFollowingURL(userID string, page int, limit int, mode string) string {
	base := "https://www.pixiv.net/ajax/user/%s/following?offset=%d&limit=%d&rest=%s"

	return fmt.Sprintf(base, userID, page*limit, limit, mode)
}

// Retrieves the users following a given user
//
// Attempting to retrieve followers for a user other than the one
// for which a PHPSESSID is provided returns HTTP 403 for the API request
func GetUserFollowersURL(userID string, page int) string {
	base := "https://www.pixiv.net/ajax/user/%s/followers?offset=%d&limit=100"

	return fmt.Sprintf(base, userID, page*UserFollowersPageSize)
}

func GetNewestFromFollowingURL(contentType, mode, page string) string {
	base := "https://www.pixiv.net/ajax/follow_latest/%s?mode=%s&p=%s"

	// TODO: Recheck this URL
	return fmt.Sprintf(base, contentType, mode, page)
}

func GetArtworkInformationURL(illustID string) string {
	base := "https://www.pixiv.net/ajax/illust/%s"

	return fmt.Sprintf(base, illustID)
}

func GetArtworkImagesURL(illustID string) string {
	base := "https://www.pixiv.net/ajax/illust/%s/pages"

	return fmt.Sprintf(base, illustID)
}

func GetArtworkRelatedURL(illustID string, limit int) string {
	base := "https://www.pixiv.net/ajax/illust/%s/recommend/init?limit=%d"

	return fmt.Sprintf(base, illustID, limit)
}

// Retrieves the comments for a given illustration ID
//
// Unlike other endpoints, the limit parameter doesn't seem to have a maximum
func GetArtworkCommentsURL(illustID string, page int) string {
	base := "https://www.pixiv.net/ajax/illusts/comments/roots?illust_id=%s&offset=%d&limit=1000"

	return fmt.Sprintf(base, illustID, page*ArtworkCommentsPageSize)
}

// Retrieves the replies for a given comment ID
//
// Unsure what the page parameter does given the lack of a limit parameter
func GetArtworkCommentRepliesURL(illustID string, page int) string {
	base := "https://www.pixiv.net/ajax/illusts/comments/replies?comment_id=%s&page=%d"

	return fmt.Sprintf(base, illustID, page)
}

// Retrieves the comments for a given novel ID
//
// Unlike other endpoints, the limit parameter doesn't seem to have a maximum
func GetNovelCommentsURL(novelID string, page int) string {
	base := "https://www.pixiv.net/ajax/novels/comments/roots?novel_id=%s&offset=%d&limit=1000"

	return fmt.Sprintf(base, novelID, page*NovelCommentsPageSize)
}

// Retrieves the replies for a given comment ID
//
// Unsure what the page parameter does given the lack of a limit parameter
func GetNovelCommentRepliesURL(novelID string, page int) string {
	base := "https://www.pixiv.net/ajax/novels/comments/replies?comment_id=%s&page=%d"

	return fmt.Sprintf(base, novelID, page)
}

func GetTagDetailURL(unescapedTag string) string {
	base := "https://www.pixiv.net/ajax/search/tags/%s"

	unescapedTag = url.PathEscape(unescapedTag)

	return fmt.Sprintf(base, unescapedTag)
}

//nolint:cyclop,wsl
func GetArtworkSearchURL(params map[string]string) string {
	// Long.
	base := "https://www.pixiv.net/ajax/search/%s/%s"

	// URL-encode the category and name
	category := params["Category"]
	name := url.PathEscape(params["Name"])

	// Base URL
	baseURL := fmt.Sprintf(base, category, name)

	// Build the query parameters
	values := url.Values{}
	if params["Order"] != "" {
		values.Add("order", params["Order"])
	}
	if params["Mode"] != "" {
		values.Add("mode", params["Mode"])
	}
	if params["Ratio"] != "" {
		values.Add("ratio", params["Ratio"])
	}
	if params["Smode"] != "" {
		values.Add("s_mode", params["Smode"])
	}
	if params["Wlt"] != "" {
		values.Add("wlt", params["Wlt"])
	}
	if params["Wgt"] != "" {
		values.Add("wgt", params["Wgt"])
	}
	if params["Hlt"] != "" {
		values.Add("hlt", params["Hlt"])
	}
	if params["Hgt"] != "" {
		values.Add("hgt", params["Hgt"])
	}
	if params["Tool"] != "" {
		values.Add("tool", params["Tool"])
	}
	if params["Scd"] != "" {
		values.Add("scd", params["Scd"])
	}
	if params["Ecd"] != "" {
		values.Add("ecd", params["Ecd"])
	}
	if params["Page"] != "" {
		values.Add("p", params["Page"])
	}

	encodedQuery := values.Encode()
	// Add word parameter with URL encoding
	if params["Name"] != "" {
		if encodedQuery != "" {
			encodedQuery += "&"
		}
		// encodedQuery += "word=" + url.PathEscape(params["Name"])
		log.Println(params["Name"])
		encodedQuery += "word=" + params["Name"]
	}

	return baseURL + "?" + encodedQuery
}

// TODO: i=1 is Creator accounts only. i=0 returns all accounts
func GetUserSearchURL(query string, page string) string {
	baseURL, _ := url.Parse("https://www.pixiv.net/ajax/search/users")

	params := fmt.Sprintf("nick=%s&s_mode=s_usr&i=0&lang=en", url.QueryEscape(query))

	// Add page parameter if it exists
	if page != "" {
		params += fmt.Sprintf("&p=%s", url.QueryEscape(page))
	}

	// Set the raw query string
	baseURL.RawQuery = params

	return baseURL.String()
}

func GetLandingURL(mode string) string {
	base := "https://www.pixiv.net/ajax/top/illust?mode=%s"

	return fmt.Sprintf(base, mode)
}

func GetNovelURL(novelID string) string {
	base := "https://www.pixiv.net/ajax/novel/%s"

	return fmt.Sprintf(base, novelID)
}

func GetNovelRelatedURL(novelID string, limit int) string {
	base := "https://www.pixiv.net/ajax/novel/%s/recommend/init?limit=%d"

	return fmt.Sprintf(base, novelID, limit)
}

func GetNovelSeriesURL(seriesID string) string {
	base := "https://www.pixiv.net/ajax/novel/series/%s"

	return fmt.Sprintf(base, seriesID)
}

func GetNovelSeriesContentURL(seriesID string, page int, perPage int) string {
	base := "https://www.pixiv.net/ajax/novel/series_content/%s?limit=%d&last_order=%d&order_by=asc"

	return fmt.Sprintf(base, seriesID, perPage, perPage*(page-1))
}

func GetNovelSeriesContentTitlesURL(seriesID int) string {
	base := "https://www.pixiv.net/ajax/novel/series/%d/content_titles"

	return fmt.Sprintf(base, seriesID)
}

func GetInsertIllustURL(novelID, id string) string {
	base := "https://www.pixiv.net/ajax/novel/%s/insert_illusts?id[]=%s"

	return fmt.Sprintf(base, novelID, id)
}

func GetMangaSeriesContentURL(seriesID string, page int) string {
	base := "https://www.pixiv.net/ajax/series/%s?p=%d"

	return fmt.Sprintf(base, seriesID, page)
}

func GetPixivSettingsURL() string {
	base := "https://www.pixiv.net/ajax/settings"

	return base
}

func GetSettingsSelfURL() string {
	base := "https://www.pixiv.net/ajax/settings/self"

	return base
}

// POST endpoints

func PostAddIllustBookmarkURL() string {
	return "https://www.pixiv.net/ajax/illusts/bookmarks/add"
}

func PostDeleteIllustBookmarkURL() string {
	return "https://www.pixiv.net/ajax/illusts/bookmarks/delete"
}

func PostIllustLikeURL() string {
	return "https://www.pixiv.net/ajax/illusts/like"
}

func PostTouchAPI() string {
	return "https://www.pixiv.net/touch/ajax_api/ajax_api.php"
}
