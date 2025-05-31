// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package pixivision

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"codeberg.org/pixivfe/pixivfe/core"
	"codeberg.org/pixivfe/pixivfe/core/requests"
	"github.com/PuerkitoBio/goquery"
)

// ParseArticle fetches and parses a single article on pixivision.
func ParseArticle(r *http.Request, id string, lang []string) (PixivisionArticle, *goquery.Document, error) {
	var (
		article      PixivisionArticle
		descParseErr error
	)

	doc, err := FetchArticle(r, id, lang)
	if err != nil {
		return PixivisionArticle{}, nil, fmt.Errorf("failed to fetch pixivision article for parsing: %w", err)
	}

	// Parse metadata
	article.Title = doc.Find("h1.am__title").Text()
	article.Category = doc.Find(".am__categoty-pr ._category-label").Text()
	// NOTE: parse error for time is intentionally ignored
	article.Date, _ = time.Parse(pixivDatetimeLayout, doc.Find("time._date").AttrOr("datetime", ""))

	// Extract the category ID from the href attribute
	// Example: /c/123/some-name -> 123
	categoryHref := doc.Find(".am__categoty-pr a").AttrOr("href", "")

	parts := strings.Split(categoryHref, "/c/")
	if len(parts) > 1 && len(parts[1]) > 0 {
		// parts[1] could be "ID" or "ID/foo". We need "ID".
		idAndRest := strings.Split(parts[1], "/")
		article.CategoryID = idAndRest[0] // strings.Split always returns at least one element
	}

	// Thumbnail processing involves multiple steps:
	// 1. Get original src.
	// 2. Apply special replacement for "embed.pixiv.net" URLs.
	// 3. Attempt WebP conversion on the (potentially modified) URL.
	// 4. Apply general proxy transformation to the final URL.
	currentThumbnailURL := doc.Find(".aie__image").AttrOr("src", "")
	currentThumbnailURL = strings.ReplaceAll(currentThumbnailURL, "https://embed.pixiv.net", "/proxy/embed.pixiv.net")

	if parsedURL, err := url.Parse(currentThumbnailURL); err == nil {
		// Note: If currentThumbnailURL became a relative path (e.g., "/proxy/..."),
		// parsedURL will reflect that. generateMasterWebpURL is called with this.
		currentThumbnailURL = core.GenerateMasterWebpURL(parsedURL, core.SizeQualityRe)
	}

	article.Thumbnail = string(core.RewriteContentURLsNoEscape(r, []byte(currentThumbnailURL)))

	// Parse description paragraphs
	doc.Find(".fab__paragraph p").EachWithBreak(func(_ int, pSelection *goquery.Selection) bool {
		// Check if the paragraph's text content (after stripping tags and trimming space) is empty.
		if strings.TrimSpace(pSelection.Text()) == "" {
			return true
		}

		// The paragraph has actual content; get its inner HTML to preserve formatting tags.
		innerHTML, err := pSelection.Html()
		if err != nil {
			descParseErr = err

			return false
		}

		article.Description = append(article.Description, innerHTML)

		return true
	})

	if descParseErr != nil {
		return PixivisionArticle{}, doc, fmt.Errorf("failed to render description paragraph: %w", descParseErr)
	}

	// Parse artworks featured in the article
	doc.Find("._feature-article-body__pixiv_illust").Each(func(i int, artworkSelection *goquery.Selection) {
		var item PixivisionArticleItem

		titleLinkSelection := artworkSelection.Find(".am__work__title a.inner-link")
		userLinkSelection := artworkSelection.Find(".am__work__user-name a.inner-link")

		item.Title = titleLinkSelection.Text()
		item.ID = parseIDFromPixivLink(titleLinkSelection.AttrOr("href", ""))
		item.Username = userLinkSelection.Text()
		item.UserID = parseIDFromPixivLink(userLinkSelection.AttrOr("href", ""))

		// NOTE: the "uesr" typo is per the source HTML
		avatarSrc := artworkSelection.Find(".am__work__user-icon-container img.am__work__uesr-icon").AttrOr("src", "")
		item.Avatar = string(core.RewriteContentURLsNoEscape(r, []byte(avatarSrc)))

		// Process images within the artwork item
		artworkSelection.Find("img.am__work__illust").Each(func(_ int, imageSelection *goquery.Selection) {
			imgSrc := imageSelection.AttrOr("src", "")
			finalImgSrc := processPixivisionImageURL(r, imgSrc, core.SizeQualityRe)
			item.Images = append(item.Images, finalImgSrc)
		})

		article.Items = append(article.Items, item)
	})

	// Parse tags associated with the article
	doc.Find("._tag-list a").Each(func(i int, tagSelection *goquery.Selection) {
		var tag PixivisionEmbedTag
		tag.ID = parseIDFromPixivLink(tagSelection.AttrOr("href", ""))
		tag.Name = tagSelection.AttrOr("data-gtm-label", "")
		article.Tags = append(article.Tags, tag)
	})

	doc.Find("div._related-articles[data-gtm-category='Related Article Latest']").Each(func(i int, section *goquery.Selection) {
		if section.Find("ul.rla__list-group").Length() > 0 {
			article.NewestTaggedArticles = parseRelatedArticleSection(section, r, core.SizeQualityRe)
		}
	})

	doc.Find("div._related-articles[data-gtm-category='Related Article Popular']").Each(func(i int, section *goquery.Selection) {
		if section.Find("ul.rla__list-group").Length() > 0 {
			article.PopularTaggedArticles = parseRelatedArticleSection(section, r, core.SizeQualityRe)
		}
	})

	doc.Find("div._related-articles[data-gtm-category='Article Latest']").Each(func(i int, section *goquery.Selection) {
		if section.Find("ul.rla__list-group").Length() > 0 {
			article.NewestCategoryArticles = parseRelatedArticleSection(section, r, core.SizeQualityRe)
		}
	})

	return article, doc, nil
}

// FetchArticle fetches the pixivision article page and returns it as a goquery.Document.
func FetchArticle(r *http.Request, id string, lang []string) (*goquery.Document, error) {
	URL := generatePixivisionURL("a/"+id, lang)
	userLang := determineUserLang(URL, lang...)

	cookies := map[string]string{
		"user_lang": userLang,
		"PHPSESSID": requests.NoToken,
	}

	// Fetch the article page
	resp, err := requests.PerformGETReader(r.Context(), URL, cookies, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch article page: %w", err)
	}

	// Parse HTML response
	doc, err := goquery.NewDocumentFromReader(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	return doc, nil
}

// parseRelatedArticleSection parses a list of related articles from a div._related-articles selection.
//
// sectionSelection should be a goquery.Selection for a 'div._related-articles' element.
func parseRelatedArticleSection(
	sectionSelection *goquery.Selection,
	r *http.Request,
	reSizeQuality *regexp.Regexp,
) RelatedArticleGroup {
	var group RelatedArticleGroup
	group.HeadingLink = normalizeHeadingLink(sectionSelection.Find("h3.rla__heading a").AttrOr("href", ""))

	// Find article items within the ul.rla__list-group inside the current sectionSelection
	sectionSelection.Find("ul.rla__list-group li.rla__list-item article._article-summary-card-related").Each(func(i int, s *goquery.Selection) {
		var item RelatedPixivisionArticle

		thumbLinkSelection := s.Find("a.ascr__thumbnail-container")
		item.ID = thumbLinkSelection.AttrOr("data-gtm-label", "")

		styleAttr := thumbLinkSelection.Find("div._thumbnail").AttrOr("style", "")
		rawThumbnailURL := parseBackgroundImage(styleAttr)

		if rawThumbnailURL != "" {
			item.Thumbnail = processPixivisionImageURL(r, rawThumbnailURL, reSizeQuality)
		} else {
			item.Thumbnail = ""
		}

		item.Category = s.Find("div.ascr__category-pr a span._category-label").Text()
		titleElement := s.Find("div.ascr__title-container a h4.ascr__title")
		item.Title = titleElement.Text()

		if item.ID == "" {
			titleHref := titleElement.Parent().AttrOr("href", "")
			if titleHref != "" {
				item.ID = parseIDFromPixivLink(titleHref)
			}
		}

		group.Articles = append(group.Articles, item)
	})

	return group
}

type FreeformArticle struct {
	Title   string
	Content string
}

func ParseArticleFreeform(r *http.Request, doc *goquery.Document) (FreeformArticle, error) {
	outerHTML, err := goquery.OuterHtml(doc.Find(".am__body"))
	if err != nil {
		return FreeformArticle{}, fmt.Errorf("failed to extract article body: %w", err)
	}

	content := core.RewriteContentURLsNoEscape(r, []byte(outerHTML))
	title := doc.Find("h1.am__title").Text()

	return FreeformArticle{Title: title, Content: string(content)}, nil
}
