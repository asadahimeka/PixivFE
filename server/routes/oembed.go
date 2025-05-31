// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package routes

import (
	"fmt"
	"net/http"
	"strconv"

	"codeberg.org/pixivfe/pixivfe/server/utils"
	"github.com/goccy/go-json"
)

// BaseOEmbed contains common fields for all oEmbed types.
//
//nolint:tagliatelle
type BaseOEmbed struct {
	Type         string `json:"type"`
	Title        string `json:"title"`
	Version      string `json:"version"`
	ProviderName string `json:"provider_name"`
	ProviderURL  string `json:"provider_url"`
}

// PhotoOEmbed represents the oEmbed response structure for the photo type.
//
//nolint:tagliatelle
type PhotoOEmbed struct {
	BaseOEmbed
	URL             string `json:"url"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	AuthorName      string `json:"author_name"`
	AuthorURL       string `json:"author_url"`
	ThumbnailURL    string `json:"thumbnail_url"`
	ThumbnailWidth  int    `json:"thumbnail_width"`
	ThumbnailHeight int    `json:"thumbnail_height"`
}

// LinkOEmbed represents the oEmbed response structure for the link type.
type LinkOEmbed struct {
	BaseOEmbed
}

func Oembed(w http.ResponseWriter, r *http.Request) error {
	providerURL := utils.Origin(r)
	oEmbedType := GetQueryParam(r, "type", "")
	title := GetQueryParam(r, "title", "")
	authorName := GetQueryParam(r, "author_name", "")
	authorURL := GetQueryParam(r, "author_url", "")
	thumbnailURL := GetQueryParam(r, "thumbnail_url", "")

	// Convert width and height strings to integers
	thumbnailWidthInt := 0

	if width := GetQueryParam(r, "thumbnail_width", ""); width != "" {
		if converted, err := strconv.Atoi(width); err == nil {
			thumbnailWidthInt = converted
		}
	}

	thumbnailHeightInt := 0

	if height := GetQueryParam(r, "thumbnail_height", ""); height != "" {
		if converted, err := strconv.Atoi(height); err == nil {
			thumbnailHeightInt = converted
		}
	}

	base := BaseOEmbed{
		Type:         oEmbedType,
		Title:        title,
		Version:      "1.0",
		ProviderName: "PixivFE",
		ProviderURL:  providerURL,
	}

	// Per the oEmbed spec, `application/json+oembed` is for discovery via <link> elements and Link headers
	//
	// The actual provider response should use the mime-type of `application/json`
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if oEmbedType == "photo" {
		photoData := PhotoOEmbed{
			BaseOEmbed:      base,
			URL:             thumbnailURL,
			Width:           thumbnailWidthInt,
			Height:          thumbnailHeightInt,
			AuthorName:      authorName,
			AuthorURL:       authorURL,
			ThumbnailURL:    thumbnailURL,
			ThumbnailWidth:  thumbnailWidthInt,
			ThumbnailHeight: thumbnailHeightInt,
		}
		if err := json.NewEncoder(w).Encode(photoData); err != nil {
			return fmt.Errorf("failed to encode photo oEmbed response: %w", err)
		}
	} else {
		linkData := LinkOEmbed{
			BaseOEmbed: base,
		}
		if err := json.NewEncoder(w).Encode(linkData); err != nil {
			return fmt.Errorf("failed to encode link oEmbed response: %w", err)
		}
	}
	return nil
}
