// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package pixivision

import (
	"time"
)

// PixivisionArticle represents a single article on pixivision
type PixivisionArticle struct {
	ID                     string
	Title                  string
	Description            []string
	Category               string // Friendly category name; isn't guaranteed to be a valid category page reference
	CategoryID             string // Valid category page reference
	Thumbnail              string
	Date                   time.Time
	Items                  []PixivisionArticleItem
	Tags                   []PixivisionEmbedTag
	NewestTaggedArticles   RelatedArticleGroup
	PopularTaggedArticles  RelatedArticleGroup
	NewestCategoryArticles RelatedArticleGroup
}

// PixivisionEmbedTag represents a tag associated with a pixivision article
type PixivisionEmbedTag struct {
	ID   string
	Name string
}

// PixivisionArticleItem represents an item (artwork) within a pixivision article
type PixivisionArticleItem struct {
	Username string
	UserID   string
	Title    string
	ID       string
	Avatar   string
	Images   []string
}

// RelatedArticleGroup represents a group of related articles with a common heading link.
type RelatedArticleGroup struct {
	HeadingLink string
	Articles    []RelatedPixivisionArticle
}

// RelatedPixivisionArticle represents a related article summary found in sections
// like "Newest articles tagged X" or "If you liked X, you will also love...".
type RelatedPixivisionArticle struct {
	ID        string // Article ID, e.g., "10584"
	Title     string // Title of the related article
	Category  string // Category, e.g., "Illustrations"
	Thumbnail string // URL of the thumbnail image
}

// PixivisionTag represents a tag page on pixivision
type PixivisionTag struct {
	Title       string
	ID          string
	Description string
	Thumbnail   string
	Articles    []PixivisionArticle
	Total       int // The total number of articles
}

// PixivisionCategory represents a category page on pixivision
type PixivisionCategory struct {
	Articles    []PixivisionArticle
	Thumbnail   string
	Title       string
	Description string
}
