// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package pixivision

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"codeberg.org/pixivfe/pixivfe/core"
	"codeberg.org/pixivfe/pixivfe/core/requests"
	"github.com/PuerkitoBio/goquery"
)

// GetTag fetches and parses a tag page on pixivision.
func GetTag(r *http.Request, id string, page string, lang ...string) (PixivisionTag, error) {
	var tag PixivisionTag

	URL := generatePixivisionURL(fmt.Sprintf("t/%s/?p=%s", id, page), lang)

	userLang := determineUserLang(URL, lang...)

	cookies := map[string]string{
		"user_lang": userLang,
		"PHPSESSID": requests.NoToken,
	}

	resp, err := requests.PerformGETReader(r.Context(), URL, cookies, r.Header)
	if err != nil {
		return tag, err
	}

	doc, err := goquery.NewDocumentFromReader(resp)
	if err != nil {
		return tag, err
	}

	tag.Title = doc.Find(".tdc__header h1").Text()
	if tag.Title == "" {
		tag.Title = doc.Find("li.brc__list-item:nth-child(3)").Text()
	}

	tag.ID = id

	// Extract and process the description
	fullDescription := doc.Find(".tdc__description").Text()
	parts := strings.Split(fullDescription, "pixivision") // split on the "pixivision currently has ..." boilerplate
	tag.Description = strings.TrimSpace(parts[0])

	// Extract thumbnail
	tag.Thumbnail = parseBackgroundImage(doc.Find(".tdc__thumbnail").AttrOr("style", ""))
	if strings.HasPrefix(tag.Thumbnail, "https://source.pixiv.net") {
		tag.Thumbnail = strings.ReplaceAll(tag.Thumbnail, "https://source.pixiv.net", "/proxy/source.pixiv.net")
	} else {
		proxiedThumb := core.RewriteContentURLsNoEscape(r, []byte(tag.Thumbnail))
		tag.Thumbnail = string(proxiedThumb)
	}

	// Extract total number of articles if available
	if len(parts) > 1 {
		re := regexp.MustCompile(`(\d+)\s+article\(s\)`)
		matches := re.FindStringSubmatch(parts[1])
		if len(matches) > 1 {
			tag.Total, _ = strconv.Atoi(matches[1])
		}
	}

	// Parse each article in the tag page
	doc.Find("._article-card").Each(func(i int, s *goquery.Selection) {
		var article PixivisionArticle

		// article.ID = s.Find(".arc__title a").AttrOr("data-gtm-label", "")
		// article.Title = s.Find(".arc__title a").Text()

		article.ID = s.Find(`a[data-gtm-action="ClickTitle"]`).AttrOr("data-gtm-label", "")
		article.Title = s.Find(`a[data-gtm-action="ClickTitle"]`).Text()
		article.Category = s.Find(".arc__thumbnail-label").Text()
		article.Thumbnail = parseBackgroundImage(s.Find("._thumbnail").AttrOr("style", ""))

		date := s.Find("time._date").AttrOr("datetime", "")
		article.Date, _ = time.Parse(pixivDatetimeLayout, date)

		// Parse tags associated with the article
		s.Find("._tag-list a").Each(func(i int, t *goquery.Selection) {
			var ttag PixivisionEmbedTag
			ttag.ID = parseIDFromPixivLink(t.AttrOr("href", ""))
			ttag.Name = t.AttrOr("data-gtm-label", "")

			article.Tags = append(article.Tags, ttag)
		})

		// Proxy the thumbnail URL for the article
		proxied := core.RewriteContentURLsNoEscape(r, []byte(article.Thumbnail))
		article.Thumbnail = string(proxied)

		tag.Articles = append(tag.Articles, article)
	})

	return tag, nil
}
