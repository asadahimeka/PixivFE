// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"codeberg.org/pixivfe/pixivfe/core/requests"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"github.com/goccy/go-json"
	"github.com/tidwall/gjson"
)

type Novel struct {
	Bookmarks      int       `json:"bookmarkCount"`
	CommentCount   int       `json:"commentCount"`
	MarkerCount    int       `json:"markerCount"`
	CreateDate     time.Time `json:"createDate"`
	UploadDate     time.Time `json:"uploadDate"`
	Description    string    `json:"description"`
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Likes          int       `json:"likeCount"`
	Pages          int       `json:"pageCount"`
	UserID         string    `json:"userId"`
	UserName       string    `json:"userName"`
	Views          int       `json:"viewCount"`
	IsOriginal     bool      `json:"isOriginal"`
	IsBungei       bool      `json:"isBungei"`
	XRestrict      XRestrict `json:"xRestrict"`
	Restrict       int       `json:"restrict"`
	Content        string    `json:"content"`
	CoverURL       string    `json:"coverUrl"`
	IsBookmarkable bool      `json:"isBookmarkable"`
	BookmarkData   any       `json:"bookmarkData"`
	LikeData       bool      `json:"likeData"`
	PollData       any       `json:"pollData"`
	Marker         any       `json:"marker"`
	Tags           struct {
		AuthorID string `json:"authorId"`
		IsLocked bool   `json:"isLocked"`
		Tags     []struct {
			Name string `json:"tag"`
		} `json:"tags"`
		Writable bool `json:"writable"`
	} `json:"tags"`
	SeriesNavData struct {
		SeriesType    string `json:"seriesType"`
		SeriesID      int    `json:"seriesId"`
		Title         string `json:"title"`
		IsConcluded   bool   `json:"isConcluded"`
		IsReplaceable bool   `json:"isReplaceable"`
		IsWatched     bool   `json:"isWatched"`
		IsNotifying   bool   `json:"isNotifying"`
		Order         int    `json:"order"`
		Next          struct {
			Title     string `json:"title"`
			Order     int    `json:"order"`
			ID        string `json:"id"`
			Available bool   `json:"available"`
		} `json:"next"`
		Prev struct {
			Title     string `json:"title"`
			Order     int    `json:"order"`
			ID        string `json:"id"`
			Available bool   `json:"available"`
		} `json:"prev"`
	} `json:"seriesNavData"`
	HasGlossary bool `json:"hasGlossary"`
	IsUnlisted  bool `json:"isUnlisted"`
	// seen values: zh-cn, ja
	Language       string `json:"language"`
	CommentOff     int    `json:"commentOff"`
	CharacterCount int    `json:"characterCount"`
	WordCount      int    `json:"wordCount"`
	UseWordCount   bool   `json:"useWordCount"`
	ReadingTime    int    `json:"readingTime"`
	AiType         AiType `json:"aiType"`
	Genre          string `json:"genre"`
	Settings       struct {
		ViewMode int `json:"viewMode"`
		// ...
	} `json:"suggestedSettings"`
	TextEmbeddedImages map[string]struct {
		NovelImageId string `json:"novelImageId"`
		SanityLevel  string `json:"sl"`
		Urls         struct {
			Two40Mw     string `json:"240mw"`
			Four80Mw    string `json:"480mw"`
			One200X1200 string `json:"1200x1200"`
			One28X128   string `json:"128x128"`
			Original    string `json:"original"`
		} `json:"urls"`
	} `json:"textEmbeddedImages"`
	CommentsData CommentsData
	UserNovels   map[string]*NovelBrief `json:"userNovels"`
}

type NovelBrief struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	XRestrict      XRestrict `json:"xRestrict"`
	Restrict       int       `json:"restrict"`
	CoverURL       string    `json:"url"`
	Tags           []string  `json:"tags"`
	UserID         string    `json:"userId"`
	UserName       string    `json:"userName"`
	UserAvatar     string    `json:"profileImageUrl"`
	TextCount      int       `json:"textCount"`
	WordCount      int       `json:"wordCount"`
	ReadingTime    int       `json:"readingTime"`
	Description    string    `json:"description"`
	IsBookmarkable bool      `json:"isBookmarkable"`
	BookmarkData   any       `json:"bookmarkData"`
	Bookmarks      int       `json:"bookmarkCount"`
	IsOriginal     bool      `json:"isOriginal"`
	CreateDate     time.Time `json:"createDate"`
	UpdateDate     time.Time `json:"updateDate"`
	IsMasked       bool      `json:"isMasked"`
	SeriesID       string    `json:"seriesId"`
	SeriesTitle    string    `json:"seriesTitle"`
	IsUnlisted     bool      `json:"isUnlisted"`
	AiType         AiType    `json:"aiType"`
	Genre          string    `json:"genre"`
}

// Novel embedded illusts
var (
	re_r  = regexp.MustCompile(`\[pixivimage:\d+(-\d+)?\]`)
	re_d  = regexp.MustCompile(`\d+(-\d+)?`)
	re_u  = regexp.MustCompile(`\[uploadedimage:(\d+)\]`)
	re_id = regexp.MustCompile(`\d+`)
)

func GetNovelByID(r *http.Request, id string) (Novel, error) {
	var novel Novel

	url := GetNovelURL(id)
	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	resp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
	if err != nil {
		return novel, err
	}

	resp = RewriteContentURLs(r, resp)

	err = json.Unmarshal(resp, &novel)
	if err != nil {
		return novel, err
	}

	// Clean up UserNovels map by removing null entries
	if novel.UserNovels != nil {
		cleanedUserNovels := make(map[string]*NovelBrief)

		for id, novelBrief := range novel.UserNovels {
			if novelBrief != nil {
				cleanedUserNovels[id] = novelBrief
			}
		}
		novel.UserNovels = cleanedUserNovels
	}

	// Get view mode
	viewMode := determineViewMode(r, novel.Settings.ViewMode)

	// Process the novel content
	novel.Content = processNovelContent(r, novel, viewMode)

	return novel, nil
}

func GetNovelRelated(r *http.Request, id string) ([]NovelBrief, error) {
	var novels struct {
		List []NovelBrief `json:"novels"`
	}

	// hard-coded value, may change
	url := GetNovelRelatedURL(id, 180)

	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	rawResp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
	if err != nil {
		return novels.List, err
	}

	rawResp = RewriteContentURLs(r, rawResp)

	err = json.Unmarshal(rawResp, &novels)
	if err != nil {
		return novels.List, err
	}

	return novels.List, nil
}

// determineViewMode determines the view mode based on novel settings and cookie
func determineViewMode(r *http.Request, defaultViewMode int) int {
	viewMode := defaultViewMode

	// Check for Cookie_NovelViewMode override
	if cookieViewMode := session.GetCookie(r, session.Cookie_NovelViewMode); cookieViewMode != "" {
		if vmInt, err := strconv.Atoi(cookieViewMode); err == nil {
			viewMode = vmInt
		}
	}

	return viewMode
}

// processNovelContent handles the replacement of [pixivimage:...] and [uploadedimage:...] tags
func processNovelContent(r *http.Request, novel Novel, viewMode int) string {
	content := novel.Content

	// Replace [pixivimage:...] tags with actual images
	content = re_r.ReplaceAllStringFunc(content, func(s string) string {
		illustid := re_d.FindString(s)
		url := GetInsertIllustURL(novel.ID, illustid)

		cookies := map[string]string{
			"PHPSESSID": session.GetUserToken(r),
		}

		resp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
		if err != nil {
			return "Cannot insert illust" + illustid
		}

		imgURL := gjson.GetBytes(resp, "original").String()
		if imgURL == "" {
			return "Invalid image data for " + illustid
		}

		imgBytes := RewriteContentURLs(r, []byte(imgURL))

		imgURL = string(imgBytes)

		// make [pixivimage:illustid-index] jump to anchor
		link := fmt.Sprintf("/artworks/%s", strings.ReplaceAll(illustid, "-", "#"))

		return createImageHTML(imgURL, s, link, viewMode)
	})

	// Replace [uploadedimage:...] tags with actual images from TextEmbeddedImages
	content = re_u.ReplaceAllStringFunc(content, func(s string) string {
		imageID := re_id.FindString(s)
		if val, ok := novel.TextEmbeddedImages[imageID]; ok {
			return createImageHTML(val.Urls.Original, s, "", viewMode)
		}
		return s
	})

	return content
}

// createImageHTML generates the HTML for embedding an image based on view mode
func createImageHTML(imgURL, alt, link string, viewMode int) string {
	// Styling for horizontal text (default for viewMode 0 and 1)
	imgClass := "w-full md:w-6/12 rounded drop-shadow my-3"

	// Styling for vertical text
	if viewMode == 2 {
		imgClass = "size-full rounded drop-shadow ms-3 me-3"
	}

	if link != "" {
		return fmt.Sprintf(`<a href="%s"><img src=%s alt="%s" class="%s"/></a>`,
			link, imgURL, alt, imgClass)
	}

	return fmt.Sprintf(`<img src=%s alt="%s" class="%s"/>`,
		imgURL, alt, imgClass)
}
