// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
For interacting with the pixiv touch AJAX API
*/
package core

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// IllustDetailsManyResponse encapsulates the API response structure of GetIllustDetailsManyURL.
//
//nolint:tagliatelle
type IllustDetailsManyResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	Body    struct {
		IllustDetails []TouchArtwork `json:"illust_details"`
	} `json:"body"`
}

// TouchArtwork encapsulates artwork information as returned by GetIllustDetailsManyURL.
//
// Note that certain fields are provided as strings in the JSON, but are converted to ints via a custom UnmarshalJSON.
//
//nolint:tagliatelle
type TouchArtwork struct {
	URL                     string   `json:"url"`
	Tags                    []string `json:"tags"`
	TitleCaptionTranslation struct {
		WorkTitle   *string `json:"work_title"`
		WorkCaption *string `json:"work_caption"`
	} `json:"title_caption_translation"`
	IsMypixiv         bool    `json:"is_mypixiv"`
	IsPrivate         bool    `json:"is_private"`
	IsHowto           bool    `json:"is_howto"`
	IsOriginal        bool    `json:"is_original"`
	Alt               string  `json:"alt"`
	URLS              *string `json:"url_s"`
	URLSm             *string `json:"url_sm"`
	URLW              *string `json:"url_w"`
	URLSs             *string `json:"url_ss"`
	URLBig            *string `json:"url_big"`
	URLPlaceholder    *string `json:"url_placeholder"`
	UploadTimestamp   int64   `json:"upload_timestamp"`
	LocationMask      bool    `json:"location_mask"`
	ID                string  `json:"id"`
	UserID            string  `json:"user_id"`
	Title             string  `json:"title"`
	Width             int
	Height            int
	Type              int
	BookStyle         string `json:"book_style"`
	PageCount         int
	CommentOffSetting int `json:"comment_off_setting"`
	Restrict          int
	XRestrict         XRestrict
	Sl                int     `json:"sl"`
	AiType            AiType  `json:"ai_type"`
	Comment           *string `json:"comment"`
	AuthorDetails     struct {
		UserID      string `json:"user_id"`
		UserName    string `json:"user_name"`
		UserAccount string `json:"user_account"`
		UserImage   string `json:"user_image"`
	} `json:"author_details"`
	// Internal fields to hold custom data.
	Thumbnails Thumbnails
	Rank       int
}

// UnmarshalJSON implements custom unmarshalling for TouchArtwork
// so that certain JSON string fields can be converted into int fields.
func (ta *TouchArtwork) UnmarshalJSON(data []byte) error {
	// alias avoids infinite recursion.
	type alias TouchArtwork

	// aux is a temporary structure with string fields for data that need conversion.
	aux := &struct {
		Width     string `json:"width"`
		Height    string `json:"height"`
		Restrict  string `json:"restrict"`
		XRestrict string `json:"x_restrict"`
		Type      string `json:"type"`
		PageCount string `json:"page_count"`
		*alias
	}{
		alias: (*alias)(ta),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// convField is a conversion structure to handle string-to-int conversion generically.
	type convField struct {
		name  string
		value string
		dst   *int
	}
	conversions := []convField{
		{"width", aux.Width, &ta.Width},
		{"height", aux.Height, &ta.Height},
		{"restrict", aux.Restrict, &ta.Restrict},
		{"x_restrict", aux.XRestrict, (*int)(&ta.XRestrict)},
		{"type", aux.Type, &ta.Type},
		{"page_count", aux.PageCount, &ta.PageCount},
	}

	for _, conv := range conversions {
		n, err := strconv.Atoi(conv.value)
		if err != nil {
			return fmt.Errorf("failed to convert %s %q: %w", conv.name, conv.value, err)
		}
		*conv.dst = n
	}

	return nil
}
