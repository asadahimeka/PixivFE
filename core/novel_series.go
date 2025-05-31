// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"net/http"
	"time"

	"codeberg.org/pixivfe/pixivfe/core/requests"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"github.com/goccy/go-json"
)

type NovelSeries struct {
	ID                            string    `json:"id"`
	UserID                        string    `json:"userId"`
	UserName                      string    `json:"userName"`
	UserAvatar                    string    `json:"profileImageUrl"`
	XRestrict                     XRestrict `json:"xRestrict"`
	IsOriginal                    bool      `json:"isOriginal"`
	IsConcluded                   bool      `json:"isConcluded"`
	GenreID                       string    `json:"genreId"`
	Title                         string    `json:"title"`
	Caption                       string    `json:"caption"`
	Language                      string    `json:"language"`
	Tags                          []string  `json:"tags"`
	PublishedContentCount         int       `json:"publishedContentCount"`
	PublishedTotalCharacterCount  int       `json:"publishedTotalCharacterCount"`
	PublishedTotalWordCount       int       `json:"publishedTotalWordCount"`
	PublishedReadingTime          int       `json:"publishedReadingTime"`
	UseWordCount                  bool      `json:"useWordCount"`
	LastPublishedContentTimestamp int       `json:"lastPublishedContentTimestamp"`
	CreatedTimestamp              int       `json:"createdTimestamp"`
	UpdatedTimestamp              int       `json:"updatedTimestamp"`
	CreateDate                    time.Time `json:"createDate"`
	UpdateDate                    time.Time `json:"updateDate"`
	FirstNovelID                  string    `json:"firstNovelId"`
	LatestNovelID                 string    `json:"latestNovelId"`
	DisplaySeriesContentCount     int       `json:"displaySeriesContentCount"`
	ShareText                     string    `json:"shareText"`
	Total                         int       `json:"total"`
	FirstEpisode                  struct {
		URL string `json:"url"`
	} `json:"firstEpisode"`
	WatchCount   any `json:"watchCount"`
	MaxXRestrict any `json:"maxXRestrict"`
	Cover        struct {
		Urls struct {
			Two40Mw     string `json:"240mw"`
			Four80Mw    string `json:"480mw"`
			One200X1200 string `json:"1200x1200"`
			One28X128   string `json:"128x128"`
			Original    string `json:"original"`
		} `json:"urls"`
	} `json:"cover"`
	CoverSettingData any    `json:"coverSettingData"`
	IsWatched        bool   `json:"isWatched"`
	IsNotifying      bool   `json:"isNotifying"`
	AiType           AiType `json:"aiType"`
	HasGlossary      bool   `json:"hasGlossary"`
}

type NovelSeriesContent struct {
	ID     string `json:"id"`
	UserID string `json:"userId"`
	Series struct {
		ID           int `json:"id"`
		ViewableType int `json:"viewableType"`
		ContentOrder int `json:"contentOrder"`
	} `json:"series"`
	Title             string    `json:"title"`
	CommentHTML       string    `json:"commentHtml"`
	Tags              []string  `json:"tags"`
	Restrict          int       `json:"restrict"`
	XRestrict         XRestrict `json:"xRestrict"`
	IsOriginal        bool      `json:"isOriginal"`
	TextLength        int       `json:"textLength"`
	CharacterCount    int       `json:"characterCount"`
	WordCount         int       `json:"wordCount"`
	UseWordCount      bool      `json:"useWordCount"`
	ReadingTime       int       `json:"readingTime"`
	Bookmarks         int       `json:"bookmarkCount"`
	CoverURL          string    `json:"url"`
	UploadTimestamp   int       `json:"uploadTimestamp"`
	ReuploadTimestamp int       `json:"reuploadTimestamp"`
	IsBookmarkable    bool      `json:"isBookmarkable"`
	BookmarkData      any       `json:"bookmarkData"`
	AiType            AiType    `json:"aiType"`
	// Merge the data of `thumbnails.novel`
	Brief NovelBrief
}

type NovelSeriesContentTitle struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Available bool   `json:"available"`
}

func GetNovelSeriesByID(r *http.Request, id string) (NovelSeries, error) {
	var series NovelSeries

	url := GetNovelSeriesURL(id)

	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	rawResp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
	if err != nil {
		return series, err
	}

	rawResp = RewriteContentURLs(r, rawResp)

	err = json.Unmarshal(rawResp, &series)
	if err != nil {
		return series, err
	}

	return series, nil
}

func GetNovelSeriesContentByID(r *http.Request, seriesID string, page int, perPage int) ([]NovelSeriesContent, error) {
	var novels struct {
		Thumbnails struct {
			List []NovelBrief `json:"novel"`
		} `json:"thumbnails"`
		Page struct {
			SeriesContents []NovelSeriesContent `json:"seriesContents"`
		} `json:"page"`
	}

	url := GetNovelSeriesContentURL(seriesID, page, perPage)

	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	rawResp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
	if err != nil {
		return novels.Page.SeriesContents, err
	}

	rawResp = RewriteContentURLs(r, rawResp)

	err = json.Unmarshal(rawResp, &novels)
	if err != nil {
		return novels.Page.SeriesContents, err
	}

	seriesContent := novels.Page.SeriesContents

	// TODO theoretically, series_content should have same order as with novels.Thumbnails.
	// Just ranging over and `series_content[idx].Brief = novels.Thumbnails.List[idx]` should be enough.
	for idx, content := range seriesContent {
		for _, item := range novels.Thumbnails.List {
			if item.ID == content.ID {
				seriesContent[idx].Brief = item

				break
			}
		}
	}

	return seriesContent, nil
}

func GetNovelSeriesContentTitlesByID(r *http.Request, id int) ([]NovelSeriesContentTitle, error) {
	var contentTitles []NovelSeriesContentTitle

	url := GetNovelSeriesContentTitlesURL(id)

	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	rawResp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
	if err != nil {
		return contentTitles, err
	}

	err = json.Unmarshal(rawResp, &contentTitles)
	if err != nil {
		return contentTitles, err
	}

	return contentTitles, nil
}
