// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/core/requests"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"github.com/goccy/go-json"
)

type Ranking struct {
	Contents    []TouchArtwork
	Title       string
	Mode        string
	Content     string
	Page        int
	RankTotal   int
	CurrentDate string
	PrevDate    string
	NextDate    string
}

type RankingResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	Body    struct {
		RankingDate string `json:"rankingDate"`
		Ranking     []struct {
			IllustID string `json:"illustId"`
			Rank     int    `json:"rank"`
		} `json:"ranking"`
	} `json:"body"`
}

func GetRanking(r *http.Request, mode, contentType, date, page string) (Ranking, error) {
	var ranking Ranking

	pageInt, _ := strconv.Atoi(page)

	location := config.GlobalConfig.Basic.TimeLocation

	var (
		currentDate, prevDate, nextDate string
		rankingURL                      string
	)

	if date == "" {
		rankingURL = GetRankingURL(mode, contentType, "", page)
	} else {
		var err error
		currentDate, prevDate, nextDate, err = getDateRange(date, location)
		if err != nil {
			return ranking, fmt.Errorf("get date range: %w", err)
		}
		rankingURL = GetRankingURL(mode, contentType, currentDate, page)
	}

	var rankingResp RankingResponse

	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	rawRankingResp, err := requests.FetchJSON(r.Context(), rankingURL, cookies, r.Header)
	if err != nil {
		return ranking, fmt.Errorf("failed to fetch ranking: %v", err)
	}

	rawRankingResp = RewriteContentURLs(r, rawRankingResp)

	if err := json.Unmarshal(rawRankingResp, &rankingResp); err != nil {
		return ranking, fmt.Errorf("failed to unmarshal ranking data: %v", err)
	}

	if rankingResp.Error {
		return ranking, fmt.Errorf("ranking API error: %s", rankingResp.Message)
	}

	// Handle currentDate when date parameter is empty
	if date == "" {
		currentDate = rankingResp.Body.RankingDate
		parsedCurrentDate, err := time.Parse("2006-01-02", currentDate)
		if err != nil {
			return ranking, fmt.Errorf("failed to parse ranking date %s: %v", currentDate, err)
		}
		prevDate = parsedCurrentDate.AddDate(0, 0, -1).Format("2006-01-02")
		nextDate = parsedCurrentDate.AddDate(0, 0, 1).Format("2006-01-02")
	}

	// Prepare illust IDs and rank map
	illustIDs := make([]string, 0, len(rankingResp.Body.Ranking))
	rankMap := make(map[string]int)
	for _, item := range rankingResp.Body.Ranking {
		illustIDs = append(illustIDs, item.IllustID)
		rankMap[item.IllustID] = item.Rank
	}

	if len(illustIDs) == 0 {
		return ranking, fmt.Errorf("no illusts found in ranking")
	}

	// Fetch details
	detailsURL := GetIllustDetailsManyURL(illustIDs)
	var detailsResp IllustDetailsManyResponse

	rawDetailsResp, err := requests.FetchJSON(r.Context(), detailsURL, cookies, r.Header)
	if err != nil {
		return ranking, fmt.Errorf("failed to fetch details: %v", err)
	}

	rawDetailsResp = RewriteContentURLs(r, rawDetailsResp)

	if err := json.Unmarshal(rawDetailsResp, &detailsResp); err != nil {
		return ranking, fmt.Errorf("failed to unmarshal details data: %v", err)
	}

	if detailsResp.Error {
		return ranking, fmt.Errorf("details API error: %s", detailsResp.Message)
	}

	// Merge ranks.
	detailsMap := make(map[string]TouchArtwork)
	for _, detail := range detailsResp.Body.IllustDetails {
		detailsMap[detail.ID] = detail
	}

	// Combine ranking and details.
	var orderedContents []TouchArtwork

	for _, rankingItem := range rankingResp.Body.Ranking {
		if detail, exists := detailsMap[rankingItem.IllustID]; exists {
			// Set the rank from the ranking response.
			detail.Rank = rankingItem.Rank

			// Populate thumbnails for the artwork.
			thumbnails, err := PopulateThumbnailsFor(detail.URL)
			if err != nil {
				return Ranking{}, err
			}
			detail.Thumbnails = thumbnails

			orderedContents = append(orderedContents, detail)
		}
	}

	title := generateRankingTitle(mode, contentType)

	return Ranking{
		Contents:    orderedContents,
		Title:       title,
		Mode:        mode,
		Content:     contentType,
		Page:        pageInt,
		RankTotal:   len(rankingResp.Body.Ranking),
		CurrentDate: currentDate,
		PrevDate:    prevDate,
		NextDate:    nextDate,
	}, nil
}

func getDateRange(date string, loc *time.Location) (current, prev, next string, err error) {
	const dateFormat = "2006-01-02"

	if date == "" {
		now := time.Now().In(loc).UTC()
		return now.AddDate(0, 0, -1).Format(dateFormat),
			now.AddDate(0, 0, -2).Format(dateFormat),
			now.Format(dateFormat),
			nil
	}

	parsedDate, err := time.Parse(dateFormat, date)
	if err != nil {
		return "", "", "", fmt.Errorf("parse date: %w", err)
	}

	return date,
		parsedDate.AddDate(0, 0, -1).Format(dateFormat),
		parsedDate.AddDate(0, 0, 1).Format(dateFormat),
		nil
}

// generateRankingTitle generates a human-readable title based on mode and content type.
func generateRankingTitle(mode, contentType string) string {
	modeDisplay := map[string]string{
		"daily":        "Daily",
		"weekly":       "Weekly",
		"monthly":      "Monthly",
		"rookie":       "Weekly rookie",
		"original":     "Weekly original",
		"daily_ai":     "Daily AI-generated",
		"male":         "Popular among males",
		"female":       "Popular among females",
		"daily_r18":    "Daily R-18",
		"weekly_r18":   "Weekly R-18",
		"r18g":         "Weekly R-18G",
		"daily_r18_ai": "Daily R-18 AI-generated",
		"male_r18":     "Popular among males - R-18",
		"female_r18":   "Popular among females - R-18",
	}

	typeDisplay := map[string]string{
		"all":    "",
		"illust": "illustration",
		"manga":  "manga",
		"ugoira": "ugoira",
	}

	modePart, ok := modeDisplay[mode]
	if !ok {
		modePart = mode
	}

	typePart := typeDisplay[contentType]

	if typePart == "" {
		return fmt.Sprintf("%s rankings", modePart)
	}
	return fmt.Sprintf("%s %s rankings", modePart, typePart)
}
