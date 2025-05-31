// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package pixivision

import (
	"fmt"
	"net/http"
	"time"

	"codeberg.org/pixivfe/pixivfe/core"
	"codeberg.org/pixivfe/pixivfe/core/requests"
	"github.com/PuerkitoBio/goquery"
)

const pixivDatetimeLayout = "2006-01-02" // PixivDatetimeLayout defines the date format used by pixiv

// GetHomepage fetches and parses the pixivision homepage
func GetHomepage(r *http.Request, page string, lang ...string) ([]PixivisionArticle, error) {
	var articles []PixivisionArticle

	URL := generatePixivisionURL(fmt.Sprintf("?p=%s", page), lang)

	userLang := determineUserLang(URL, lang...)

	cookies := map[string]string{
		"user_lang": userLang,
		"PHPSESSID": requests.NoToken,
	}

	resp, err := requests.PerformGETReader(r.Context(), URL, cookies, r.Header)
	if err != nil {
		return nil, err
	}

	// if resp.StatusCode == 404 {
	// 	return articles, i18n.Error("We couldn't find the page you're looking for")
	// }

	doc, err := goquery.NewDocumentFromReader(resp)
	if err != nil {
		return nil, err
	}

	// Parse each article on the homepage
	doc.Find("article.spotlight").Each(func(i int, s *goquery.Selection) {
		var article PixivisionArticle

		date := s.Find("time._date").AttrOr("datetime", "")

		article.ID = s.Find(`a[data-gtm-action=ClickTitle]`).AttrOr("data-gtm-label", "")
		article.Title = s.Find(`a[data-gtm-action=ClickTitle]`).Text()
		article.Category = s.Find("._category-label").Text()
		article.Thumbnail = parseBackgroundImage(s.Find("._thumbnail").AttrOr("style", ""))
		article.Date, _ = time.Parse(pixivDatetimeLayout, date)

		// Parse tags associated with the article
		s.Find("._tag-list a").Each(func(i int, t *goquery.Selection) {
			var tag PixivisionEmbedTag
			tag.ID = parseIDFromPixivLink(t.AttrOr("href", ""))
			tag.Name = t.AttrOr("data-gtm-label", "")

			article.Tags = append(article.Tags, tag)
		})

		articles = append(articles, article)
	})

	// Proxy the Thumbnail URL for each article
	for i := range articles {
		proxied := core.RewriteContentURLsNoEscape(r, []byte(articles[i].Thumbnail))
		articles[i].Thumbnail = string(proxied)
	}

	return articles, nil
}
