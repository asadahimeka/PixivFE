// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"net/http"
	"time"

	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/core/requests"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"codeberg.org/pixivfe/pixivfe/server/utils"
	"github.com/goccy/go-json"
	"golang.org/x/sync/errgroup"
)

type CommentsData struct {
	Comments []Comment
	Count    int
}

type Comment struct {
	UserId          string    `json:"userId"`
	UserName        string    `json:"userName"`
	IsDeletedUser   bool      `json:"isDeletedUser"`
	Img             string    `json:"img"`
	Id              string    `json:"id"`
	Comment         string    `json:"comment"`
	StampId         string    `json:"stampId"`
	StampLink       string    // pixiv returns a stampLink JSON field for replies, but we don't use it
	CommentDate     time.Time `json:"commentDate"`
	CommentRootId   string    `json:"commentRootId"` // Only present for comments that are replies
	CommentParentId string    `json:"commentParentId"`
	CommentUserId   string    `json:"commentUserId"`
	ReplyToUserId   string    `json:"replyToUserId"` // Only present for comments that are replies
	ReplyToUserName string    `json:"replyToUserName"`
	Editable        bool      `json:"editable"`
	HasReplies      bool      `json:"hasReplies"`
	Replies         []Comment // Internal field to hold replies
	WorkUserId      string    // Internal field to hold the UserId of the main work's author
}

type CommentService struct {
	getCommentsURL func(id string, page int) string
	getRepliesURL  func(id string, page int) string
}

func NewCommentService(
	getCommentsURL func(id string, page int) string,
	getRepliesURL func(id string, page int) string,
) *CommentService {
	return &CommentService{
		getCommentsURL: getCommentsURL,
		getRepliesURL:  getRepliesURL,
	}
}

type ArtworkCommentsParams struct {
	ID          string
	UserID      string
	SanityLevel SanityLevel
}

type NovelCommentsParams struct {
	ID        string
	UserID    string
	XRestrict XRestrict
}

func (cs *CommentService) GetComments(r *http.Request, workID string, noToken bool, workUserID string) ([]Comment, []utils.Timing, error) {
	// Initialize timings slice
	timings := make([]utils.Timing, 0)

	// Track total time
	start := time.Now()
	defer func() {
		totalDuration := time.Since(start)
		timings = append(timings, utils.Timing{
			Name:        "comments-total",
			Duration:    totalDuration,
			Description: "Total comments fetch time",
		})
	}()

	var allComments []Comment
	page := 0
	hasNext := true

	// Fetch all top-level comments first
	for hasNext {
		pageStart := time.Now()

		var resp struct {
			Comments []Comment `json:"comments"`
			HasNext  bool      `json:"hasNext"`
		}

		url := cs.getCommentsURL(workID, page)

		cookies := make(map[string]string)
		if noToken {
			cookies["PHPSESSID"] = requests.RandomToken
		}

		rawResp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
		if err != nil {
			return nil, timings, err
		}

		rawResp = RewriteContentURLs(r, rawResp)

		err = json.Unmarshal(rawResp, &resp)
		if err != nil {
			return nil, timings, err
		}

		allComments = append(allComments, resp.Comments...)
		hasNext = resp.HasNext
		page++

		timings = append(timings, utils.Timing{
			Name:        "comments-root-fetch",
			Duration:    time.Since(pageStart),
			Description: "Root comments fetch",
		})
	}

	// Create errgroup for concurrent fetching
	errGroup, ctx := errgroup.WithContext(r.Context())

	// Pre-allocate replies slices to avoid race conditions
	for i := range allComments {
		if allComments[i].HasReplies {
			allComments[i].Replies = make([]Comment, 0)
		}
	}

	// Fetch replies concurrently for comments that have them
	repliesStart := time.Now()

	for i := range allComments {
		if allComments[i].HasReplies {
			index := i

			errGroup.Go(func() error {
				replies, err := cs.fetchReplies(r.WithContext(ctx), allComments[index].Id, noToken)
				if err != nil {
					return err
				}
				allComments[index].Replies = replies
				return nil
			})
		}
	}

	// Wait for all reply fetches to complete
	if err := errGroup.Wait(); err != nil {
		return nil, timings, err
	}

	timings = append(timings, utils.Timing{
		Name:        "comments-replies-fetch",
		Duration:    time.Since(repliesStart),
		Description: "Replies fetch time",
	})

	// Replies are returned in reverse chronological order, but should be displayed in normal chronological order
	// in the UI, so we reverse the order of replies for each comment
	for comment := range allComments {
		// Set workUserId and processStamp for top-level comment
		allComments[comment].WorkUserId = workUserID
		if err := allComments[comment].processStamp(r); err != nil {
			return nil, timings, err
		}

		if allComments[comment].HasReplies && len(allComments[comment].Replies) > 0 {
			// Set workUserId and processStamp for each reply
			for j := range allComments[comment].Replies {
				allComments[comment].Replies[j].WorkUserId = workUserID
				if err := allComments[comment].Replies[j].processStamp(r); err != nil {
					return nil, timings, err
				}
			}

			// Reverse the replies slice
			replies := allComments[comment].Replies
			for left, right := 0, len(replies)-1; left < right; left, right = left+1, right-1 {
				replies[left], replies[right] = replies[right], replies[left]
			}
		}
	}

	return allComments, timings, nil
}

// GetArtworkComments fetches comments for a given artwork ID.
//
// Requires ArtworkCommentsParams to be provided.
//
// Returns []utils.Timing instead of writing timing headers
// directly to allow for safe concurrent use.
func GetArtworkComments(r *http.Request, params ArtworkCommentsParams) (CommentsData, []utils.Timing, error) {
	var commentsSection CommentsData

	artworkCommentService := NewCommentService(GetArtworkCommentsURL, GetArtworkCommentRepliesURL)

	noToken := true
	if params.SanityLevel > SLSafe {
		noToken = false
	}

	comments, commentTimings, err := artworkCommentService.GetComments(r, params.ID, noToken, params.UserID)
	if err != nil {
		return commentsSection, nil, err
	}

	commentsSection.Comments = comments
	commentsSection.Count = len(comments)

	return commentsSection, commentTimings, err
}

// GetNovelComments fetches comments for a given artwork ID.
//
// Requires NovelCommentsParams to be provided.
//
// Returns []utils.Timing instead of writing timing headers
// directly to allow for safe concurrent use.
func GetNovelComments(r *http.Request, params NovelCommentsParams) (CommentsData, []utils.Timing, error) {
	var commentsSection CommentsData

	novelCommentService := NewCommentService(GetNovelCommentsURL, GetNovelCommentRepliesURL)

	noToken := true
	if params.XRestrict >= 1 {
		noToken = false
	}

	comments, commentTimings, err := novelCommentService.GetComments(r, params.ID, noToken, params.UserID)
	if err != nil {
		return commentsSection, nil, err
	}

	commentsSection.Comments = comments
	commentsSection.Count = len(comments)

	return commentsSection, commentTimings, err
}

// fetchReplies is a helper function to fetch replies for a single comment.
func (cs *CommentService) fetchReplies(r *http.Request, commentID string, noToken bool) ([]Comment, error) {
	var allReplies []Comment
	page := 1
	hasNext := true

	for hasNext {
		var resp struct {
			Comments []Comment `json:"comments"`
			HasNext  bool      `json:"hasNext"`
		}

		url := cs.getRepliesURL(commentID, page)

		cookies := make(map[string]string)
		if noToken {
			cookies["PHPSESSID"] = requests.RandomToken
		}

		rawResp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
		if err != nil {
			return nil, err
		}

		rawResp = RewriteContentURLs(r, rawResp)

		err = json.Unmarshal(rawResp, &resp)
		if err != nil {
			return nil, err
		}

		allReplies = append(allReplies, resp.Comments...)
		hasNext = resp.HasNext
		page++
	}

	return allReplies, nil
}

func (c *Comment) processStamp(r *http.Request) error {
	if c.StampId != "" {
		proxy := session.GetProxyPrefix(session.GetStaticProxy(r))

		stampURL := proxy + "/common/images/stamp/generated-stamps/" + c.StampId + "_s.jpg"
		c.StampLink = `<img src="` + stampURL + `" alt="` + stampURL + `" class="stamp" loading="lazy" />`
		return nil
	}
	return nil
}

// UnmarshalJSON implements custom JSON unmarshaling for Comment.
func (c *Comment) UnmarshalJSON(data []byte) error {
	type Alias Comment // Create alias to avoid recursion
	aux := &struct {
		Date string `json:"commentDate"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// The pixiv API returns commentDate in the client's local TZ,
	// so we need to correct it back to UTC for relative dates to
	// display correctly in the front end
	loc, err := time.LoadLocation(config.GlobalConfig.Basic.TimeZone)
	if err != nil {
		// Fallback to UTC if timezone loading fails
		loc = time.UTC
	}

	parsedTime, err := time.ParseInLocation("2006-01-02 15:04", aux.Date, loc)
	if err != nil {
		// We intentionally ignore date parsing errors and just use zero time;
		// a failed time parse doesn't warrant a failed page render
		c.CommentDate = time.Time{}
		return nil //nolint:nilerr
	}

	// If the parsedTime is a future date, cap it to the current time
	if parsedTime.After(time.Now()) {
		parsedTime = time.Now()
	}

	c.CommentDate = parsedTime.UTC()
	return nil
}
