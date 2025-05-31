// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package routes

import (
	"codeberg.org/pixivfe/pixivfe/core"
	"codeberg.org/pixivfe/pixivfe/core/pixivision"
	"codeberg.org/pixivfe/pixivfe/server/template"
)

// Tutorial: adding new types in this file
// Whenever you add new types, update `TestTemplates` in render_test.go to include the type in the test
//
// Warnings:
// - Do not use pointer in Data_* struct. faker will insert nil.
// - Do not name template file a.b.jet.html or it won't be able to be used here, since Data_a.b is not a valid identifier.

type Data_about struct {
	Time           string
	Version        string
	DomainName     string
	RepoURL        string
	Revision       string
	RevisionHash   string
	ImageProxy     string
	AcceptLanguage string
}
type Data_artwork struct {
	Illust          core.Illust
	Title           string
	MetaDescription string
	MetaImage       string
	MetaImageWidth  int
	MetaImageHeight int
	MetaAuthor      string
	MetaAuthorID    string
}

type Data_artworkFast struct {
	Illust          core.Illust
	Title           string
	MetaDescription string
	MetaImage       string
	MetaImageWidth  int
	MetaImageHeight int
	MetaAuthor      string
	MetaAuthorID    string
}
type Data_ugoiraThumbnail struct {
	ID           string
	ThumbnailURL string
}
type Data_ugoiraPreview struct {
	ID string
}
type Data_artworkMulti struct {
	Artworks []core.Illust
	Title    string
}
type Data_comments struct {
	Illust   core.Illust
	Comments []core.Comment
	Title    string
}
type Data_discovery struct {
	Artworks          []core.ArtworkBrief
	Title             string
	Queries           template.PartialURL
	RequiresAsyncLoad bool // Determines if we should show loading skeleton and trigger async load
}
type Data_following struct {
	Title string
	Mode  string
	Data  core.NewestFromFollowingResponse
	Page  int
}
type Data_index struct {
	Title       string
	LoggedIn    bool
	Data        core.LandingArtworks
	NoTokenData core.Ranking
	Queries     template.PartialURL
}
type Data_newest struct {
	Items []core.ArtworkBrief
	Title string
}
type Data_novel struct {
	Novel                    core.Novel
	NovelRelated             []core.NovelBrief
	NovelSeriesContentTitles []core.NovelSeriesContentTitle
	NovelSeriesIDs           []string
	NovelSeriesTitles        []string
	User                     core.User
	Title                    string
	FontType                 string
	ViewMode                 string
	Language                 string
}
type Data_novelSeries struct {
	NovelSeries         core.NovelSeries
	NovelSeriesContents []core.NovelSeriesContent
	User                core.User
	Title               string
	Page                int
	PageLimit           int
}
type Data_novelDiscovery struct {
	Novels  []core.NovelBrief
	Title   string
	Queries template.PartialURL
}

type Data_pixivisionIndex struct {
	Data  []pixivision.PixivisionArticle
	Page  int
	Title string
}

type Data_pixivisionArticle struct {
	Article pixivision.PixivisionArticle
	Title   string
}

type Data_pixivisionArticleFreeform struct {
	Children string
	Title    string
}

type Data_pixivisionCategory struct {
	Category pixivision.PixivisionCategory
	Page     int
	ID       string
	Title    string
}

type Data_pixivisionTag struct {
	Tag   pixivision.PixivisionTag
	Page  int
	ID    string
	Title string
}

type Data_rank struct {
	Title     string
	Page      int
	PageLimit int
	Date      string
	Data      core.Ranking
}
type Data_rankingCalendar struct {
	Title       string
	Calendar    []core.DayCalendar
	Mode        string
	Year        int
	MonthBefore DateWrap
	MonthAfter  DateWrap
	ThisMonth   DateWrap
}
type Data_settings struct {
	SelfSettings       core.SettingsSelfResponse
	ProxyList          []string
	DefaultProxyServer string
}
type Data_tag struct {
	SearchQuery          string
	Title                string
	Tag                  core.TagSearchResult
	Data                 core.ArtworkSearchResponse
	QueriesC             template.PartialURL
	Page                 int
	ActiveCategory       string
	ActiveOrder          string
	ActiveMode           string
	ActiveRatio          string
	ActiveSearchMode     string
	PopularSearchEnabled bool
}
type Data_user struct {
	Title          string
	User           core.User
	PersonalFields []core.PersonalField
	WorkspaceItems []core.WorkspaceItem
	Category       core.UserWorkCategory
	Queries        template.PartialURL
	PageLimit      int
	Page           int
	MetaImage      string
}
type Data_userAtom struct {
	URL       string
	Title     string
	User      core.User
	Category  core.UserWorkCategory
	Updated   string
	PageLimit int
	Page      int
	// MetaImage string
}
type Data_userDiscovery struct {
	Users   []core.User
	Title   string
	Queries template.PartialURL
}
type Data_mangaSeries struct {
	MangaSeries core.MangaSeries
	Title       string
	Page        int
	PageLimit   int
}

type (
	Data_diagnostics struct{}
)

type Data_addBookmarkPartial struct {
	Illust core.Illust
}

type Data_deleteBookmarkPartial struct {
	Illust core.Illust
}

type Data_quickAddBookmarkPartial struct {
	ID string // Artwork ID
}

type Data_quickDeleteBookmarkPartial struct {
	ID           string // Artwork ID
	BookmarkData *core.BookmarkData
}

// Initiating a unlike from an like state
// is impossible so we don't need to worry
// about passing data to be able to render
// a like button in this state
//
// type Data_likePartial struct {
// 	Illust core.Illust
// }

type Data_unlikePartial struct {
	Illust core.Illust
}

type Data_addFollowPartial struct {
	User core.User
}

type Data_deleteFollowPartial struct {
	User core.User
}

type Data_artworkPartial struct {
	Illust core.Illust
}

type Data_recentPartial struct {
	Illust core.Illust
	Novel  core.Novel
}

type Data_relatedPartial struct {
	Illust core.Illust
	Novel  core.Novel
}

type Data_commentsPartial struct {
	CommentsData core.CommentsData
}

type Data_discoveryArtworkPartial struct {
	Artworks []core.ArtworkBrief
}
