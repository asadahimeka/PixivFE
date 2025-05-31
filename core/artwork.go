// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"codeberg.org/pixivfe/pixivfe/audit"
	"codeberg.org/pixivfe/pixivfe/core/requests"
	"codeberg.org/pixivfe/pixivfe/i18n"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"codeberg.org/pixivfe/pixivfe/server/utils"
	"github.com/goccy/go-json"
	"golang.org/x/sync/errgroup"
)

var (
	ErrInvalidXRestrict = errors.New("invalid XRestrict value")
	ErrInvalidAiType    = errors.New("invalid AiType value")
)

// Pixiv returns 0, 1, 2 to filter SFW and/or NSFW artworks.
// Those values are saved in `XRestrict`
// 0: Safe
// 1: R18
// 2: R18G
type XRestrict int

const (
	Safe XRestrict = 0
	R18  XRestrict = 1
	R18G XRestrict = 2
)

func (x XRestrict) String() (string, error) {
	switch x {
	case Safe:
		return i18n.Tr("Safe"), nil
	case R18:
		return i18n.Tr("R18"), nil
	case R18G:
		return i18n.Tr("R18G"), nil
	}

	return "", fmt.Errorf("%w: %d", ErrInvalidXRestrict, int(x))
}

// pixiv returns 0, 1, 2 to filter SFW and/or NSFW artworks.
// Those values are saved in `aiType`
// 0: Not rated / Unknown
// 1: Not AI-generated
// 2: AI-generated
type AiType int

const (
	Unrated AiType = 0
	NotAI   AiType = 1
	AI      AiType = 2 //nolint:varnamelen
)

func (x AiType) String() (string, error) {
	switch x {
	case Unrated:
		return i18n.Tr("Unrated"), nil
	case NotAI:
		return i18n.Tr("Not AI"), nil
	case AI:
		return i18n.Tr("AI"), nil
	}

	return "", fmt.Errorf("%w: %d", ErrInvalidAiType, int(x))
}

// pixiv returns 0, 1, 2 to indicate the type of illustration
// Those values are saved in `illustType`
// 0: Illustration
// 1: Manga
// 2: Ugoira
type IllustType int

const (
	Illustration IllustType = 0
	Manga        IllustType = 1
	Ugoira       IllustType = 2
)

// SanityLevel represents pixiv's content rating system for artworks.
// It is more reliable and granular for authorization control than XRestrict.
//
// SanityLevel values:
//
//	0: Unreviewed - Typically seen on newly uploaded works
//	2: Safe       - Reviewed and unrestricted content
//	4: R-15       - Reviewed, mild age restriction
//	6: R-18/R-18G - Reviewed, strict age restriction
//	                (Maps to XRestrict values 1 and 2 respectively)
//
// Notes:
//   - Content with SanityLevel > 4 requires user authorization, but
//     appear to be intermittently enforced by the API.
//   - Novel routes lack SanityLevel data.
type SanityLevel int

const (
	SLUnreviewed SanityLevel = 0
	SLSafe       SanityLevel = 2
	SLR15        SanityLevel = 4
	SLR18        SanityLevel = 6
)

type ImageResponse struct {
	Width  int               `json:"width"`
	Height int               `json:"height"`
	Urls   map[string]string `json:"urls"`
}

// BookmarkData is a custom type to handle the
// following API response formats:
//
// ```json # type 1, bookmarked
//
//	"bookmarkData": {
//	  "id": "1234",
//	  "private": false
//	},
//
// ```
//
// ```json # type 2, not bookmarked
// "bookmarkData": null
// ```
type BookmarkData struct {
	ID      string `json:"id"`
	Private bool   `json:"private"`
}

type ArtworkBrief struct {
	ID           string        `json:"id"`
	Title        string        `json:"title"`
	UserID       string        `json:"userId"`
	UserName     string        `json:"userName"`
	UserAvatar   string        `json:"profileImageUrl"`
	Thumbnail    string        `json:"url"`
	Pages        int           `json:"pageCount"`
	XRestrict    XRestrict     `json:"xRestrict"`
	SanityLevel  SanityLevel   `json:"sl"`
	AiType       AiType        `json:"aiType"`
	BookmarkData *BookmarkData `json:"bookmarkData"`
	IllustType   int           `json:"illustType"`
	Tags         []string      `json:"tags"`     // used by core/popular_search
	SeriesID     string        `json:"seriesId"` // used by core/mangaseries
	SeriesTitle  string        `json:"seriesTitle"`
	Thumbnails   Thumbnails
	Width        int
	Height       int
}

type Illust struct {
	ID               string                    `json:"id"`
	Title            string                    `json:"title"`
	Description      HTML                      `json:"description"`
	UserID           string                    `json:"userId"`
	UserName         string                    `json:"userName"`
	UserAccount      string                    `json:"userAccount"`
	RawRecentWorkIDs OptionalIntMap[*struct{}] `json:"userIllusts"` // We only want the IDs
	RecentWorkIDs    []int
	Date             time.Time `json:"uploadDate"`
	Images           []Thumbnails
	Tags             struct {
		AuthorID string `json:"authorId"`
		IsLocked bool   `json:"isLocked"`
		Tags     []Tag  `json:"tags"`
		Writable bool   `json:"writable"`
	} `json:"tags"`
	Pages         int           `json:"pageCount"`
	Bookmarks     int           `json:"bookmarkCount"`
	Likes         int           `json:"likeCount"`
	Comments      int           `json:"commentCount"`
	Views         int           `json:"viewCount"`
	CommentOff    int           `json:"commentOff"`
	SanityLevel   SanityLevel   `json:"sl"`
	XRestrict     XRestrict     `json:"xRestrict"`
	AiType        AiType        `json:"aiType"`
	BookmarkData  *BookmarkData `json:"bookmarkData"`
	Liked         bool          `json:"likeData"`
	SeriesNavData struct {
		SeriesType  string `json:"seriesType"`
		SeriesID    string `json:"seriesId"`
		Title       string `json:"title"`
		IsWatched   bool   `json:"isWatched"`
		IsNotifying bool   `json:"isNotifying"`
		Order       int    `json:"order"`
		Next        struct {
			Title string `json:"title"`
			Order int    `json:"order"`
			ID    string `json:"id"`
		} `json:"next"`
		Prev struct {
			Title string `json:"title"`
			Order int    `json:"order"`
			ID    string `json:"id"`
		} `json:"prev"`
	} `json:"seriesNavData"`
	User         User
	RecentWorks  []ArtworkBrief
	RelatedWorks []ArtworkBrief
	CommentsData CommentsData
	IllustType   IllustType `json:"illustType"`

	// The following are used on the /tags route only
	Urls struct {
		Mini     string `json:"mini"`
		Thumb    string `json:"thumb"`
		Small    string `json:"small"`
		Regular  string `json:"regular"`
		Original string `json:"original"`
	} `json:"urls"`
	Thumbnails Thumbnails
	Width      int `json:"width"`
	Height     int `json:"height"`
}

// FastIllustParams encapsulates basic artwork data required
// to call GetArtworkByIDFast, available through Artwork-*
// request headers.
type FastIllustParams struct {
	ID         string
	UserID     string
	IllustType IllustType
	Pages      int
	WebpURL    string
}

// GetArtwork retrieves information about a specific artwork.
func GetArtwork(w http.ResponseWriter, r *http.Request, artworkID string) (*Illust, error) {
	start := time.Now()
	timings := utils.NewTimings()

	var illust Illust

	// Fetch basic artwork data and process it
	basicFetchStart := time.Now()

	err := GetBasicArtwork(r, artworkID, &illust)
	if err != nil {
		return nil, fmt.Errorf("basic data fetch failed: %w", err)
	}

	timings.Append("artwork-basic-fetch", time.Since(basicFetchStart), "Basic artwork data fetch and process")

	var errGroup errgroup.Group

	// Fetch user basic information
	errGroup.Go(func() error {
		userStart := time.Now()
		userInfo, err := GetUserBasicInformation(r, illust.UserID)
		if err != nil {
			return err
		}
		illust.User = userInfo

		timings.Append("artwork-user-fetch",
			time.Since(userStart),
			"User info fetch",
		)

		return nil
	})

	// Fetch artwork images
	errGroup.Go(func() error {
		imagesStart := time.Now()
		images, err := getArtworkImages(r, artworkID, illust.IllustType)
		if err != nil {
			return err
		}
		illust.Images = images

		timings.Append("artwork-images-fetch",
			time.Since(imagesStart),
			"Images fetch",
		)

		return nil
	})

	// Fetch related artworks
	errGroup.Go(func() error {
		relatedStart := time.Now()
		related, err := GetArtworkRelated(r, artworkID, 180)
		if err != nil {
			return err
		}
		illust.RelatedWorks = related

		timings.Append("artwork-related-fetch",
			time.Since(relatedStart),
			"Related artworks fetch",
		)

		return nil
	})

	// Fetch recent works
	errGroup.Go(func() error {
		recentStart := time.Now()

		recent, err := PopulateArtworkRecent(r, illust.UserID, illust.RecentWorkIDs)
		if err != nil {
			return err
		}
		illust.RecentWorks = recent

		timings.Append("artwork-recent-fetch",
			time.Since(recentStart),
			"Recent works fetch",
		)

		return nil
	})

	// Fetch comments if comments are enabled
	if illust.CommentOff != 1 {
		errGroup.Go(func() error {
			params := ArtworkCommentsParams{
				ID:          artworkID,
				UserID:      illust.UserID,
				SanityLevel: illust.SanityLevel,
			}

			commentsData, commentTimings, err := GetArtworkComments(r, params)
			if err != nil {
				return err
			}

			illust.CommentsData = commentsData

			// Add each comment timing to the main timings
			for _, timing := range commentTimings {
				timings.Append(timing.Name, timing.Duration, timing.Description)
			}

			return nil
		})
	}

	// Wait for all goroutines to finish
	if err := errGroup.Wait(); err != nil {
		return nil, err
	}

	if illust.Images == nil {
		illust.Images = make([]Thumbnails, illust.Pages)
	}

	if illust.IllustType == Ugoira {
		proxy := session.GetProxyPrefix(session.GetUgoiraProxy(r))

		if len(illust.Images) > 0 {
			illust.Images[0].Video = proxy + "/ugoira/" + illust.ID
		}
	}

	// Write timing headers and total duration
	timings.WriteHeaders(w)
	utils.AddServerTimingHeader(
		w,
		"artwork-total",
		time.Since(start),
		"Total artwork fetch time",
	)

	return &illust, nil
}

// GetArtworkFast returns an Illust quicker than GetArtworkByID, but
// requires FastIllustParams to be known beforehand and does not fetch
// related works, recent works, or comments.
func GetArtworkFast(w http.ResponseWriter, r *http.Request, params FastIllustParams) (*Illust, error) {
	start := time.Now()
	timings := utils.NewTimings()

	var illust Illust

	var errGroup errgroup.Group

	// Fetch basic artwork data concurrently
	errGroup.Go(func() error {
		basicFetchStart := time.Now()

		err := GetBasicArtwork(r, params.ID, &illust)
		if err != nil {
			return fmt.Errorf("basic data fetch failed: %w", err)
		}

		timings.Append(
			"artwork-basic-fetch",
			time.Since(basicFetchStart),
			"Basic artwork data fetch and process",
		)

		return nil
	})

	// User info fetch
	errGroup.Go(func() error {
		userStart := time.Now()
		userInfo, err := GetUserBasicInformation(r, params.UserID)
		if err != nil {
			return fmt.Errorf("user info fetch failed: %w", err)
		}
		illust.User = userInfo

		timings.Append(
			"artwork-user-fetch",
			time.Since(userStart),
			"User info fetch",
		)

		return nil
	})

	// Images fetch
	//
	// This goroutine only runs if the artwork has more than one image
	// since we already have dimension data for the first image from
	// the `Artwork-Width` and `Artwork-Height` request headers
	if params.Pages > 1 {
		errGroup.Go(func() error {
			imagesStart := time.Now()
			images, err := getArtworkImages(r, params.ID, params.IllustType)
			if err != nil {
				return fmt.Errorf("artwork images fetch failed: %w", err)
			}
			illust.Images = images

			timings.Append(
				"artwork-images-fetch",
				time.Since(imagesStart),
				"Images fetch",
			)

			return nil
		})
	}

	// Wait for all goroutines to finish
	if err := errGroup.Wait(); err != nil {
		return nil, err
	}

	// If we skipped the images fetch, populate data for Images[0]
	if illust.Images == nil {
		illust.Images = make([]Thumbnails, illust.Pages)

		thumbnails, err := PopulateThumbnailsFor(params.WebpURL)
		if err != nil {
			return nil, fmt.Errorf("failed to generate thumbnails for image: %w", err)
		}

		illust.Images[0] = thumbnails

		// Populate data we don't yet have from PopulateThumbnailsFor
		illust.Images[0].Width = illust.Width
		illust.Images[0].Height = illust.Height
		illust.Images[0].Original = illust.Urls.Original // URL for the Original image per the API, not our generated one
		illust.Images[0].IllustType = illust.IllustType
	}

	if illust.IllustType == Ugoira {
		proxy := session.GetProxyPrefix(session.GetUgoiraProxy(r))
		if len(illust.Images) > 0 {
			illust.Images[0].Video = proxy + "/ugoira/" + illust.ID
		}
	}

	// Add timing headers
	timings.WriteHeaders(w)
	utils.AddServerTimingHeader(
		w,
		"artwork-total",
		time.Since(start),
		"Total artwork fetch time",
	)

	return &illust, nil
}

// GetBasicArtwork fetches and processes basic artwork data.
func GetBasicArtwork(r *http.Request, artworkID string, illust *Illust) error {
	url := GetArtworkInformationURL(artworkID)

	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	rawResp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
	if err != nil {
		return err
	}

	rawResp = RewriteContentURLs(r, rawResp)

	if err = json.Unmarshal(rawResp, illust); err != nil {
		return err
	}

	thumbnails, err := PopulateThumbnailsFor(illust.Urls.Small)
	if err != nil {
		return err
	}
	illust.Thumbnails = thumbnails

	// Process RawRecentWorkIDs into RecentWorkIDs
	recentWorkIDs, _ := illust.RawRecentWorkIDs.ExtractIDs()

	// Sort in reverse order
	sort.Sort(sort.Reverse(sort.IntSlice(recentWorkIDs)))

	// Limit to 20 items
	if len(recentWorkIDs) > 20 {
		recentWorkIDs = recentWorkIDs[:20]
	}

	illust.RecentWorkIDs = recentWorkIDs

	return nil
}

// getArtworkImages retrieves the images for an artwork.
//
// Required to retrieve dimension information when an artwork has
// multiple associated images.
func getArtworkImages(r *http.Request, workID string, illustType IllustType) ([]Thumbnails, error) {
	var (
		imageResp       []ImageResponse
		thumbnailsSlice []Thumbnails
	)

	url := GetArtworkImagesURL(workID)

	rawResp, err := requests.FetchJSONBodyField(r.Context(), url, nil, r.Header)
	if err != nil {
		return nil, err
	}
	rawResp = RewriteContentURLs(r, rawResp)

	err = json.Unmarshal(rawResp, &imageResp)
	if err != nil {
		return nil, err
	}

	// Extract and proxy every image
	for _, imageRaw := range imageResp {
		// Use the "small" URL to generate thumbnails
		smallURL := imageRaw.Urls["small"]
		thumbnails, err := PopulateThumbnailsFor(smallURL)
		if err != nil {
			return nil, fmt.Errorf("failed to generate thumbnails for image: %w", err)
		}
		// Set the actual original URL and dimensions
		thumbnails.Original = imageRaw.Urls["original"]
		thumbnails.Width = imageRaw.Width
		thumbnails.Height = imageRaw.Height
		thumbnails.IllustType = illustType
		thumbnailsSlice = append(thumbnailsSlice, thumbnails)
	}

	return thumbnailsSlice, nil
}

func GetArtworkRelated(r *http.Request, artworkID string, limit int) ([]ArtworkBrief, error) {
	var resp ArtworkRelatedResponse

	url := GetArtworkRelatedURL(artworkID, limit)

	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	responseBody, err := requests.FetchJSON(r.Context(), url, cookies, r.Header)
	if err != nil {
		return nil, err
	}

	rawResp := RewriteContentURLs(r, responseBody)

	if err := json.Unmarshal(rawResp, &resp); err != nil {
		return nil, err
	}

	if resp.Error {
		return nil, fmt.Errorf(
			"API error fetching related artworks (artworkID=%s, limit=%d, upstreamURL=%s): %s",
			artworkID,
			limit,
			url,
			resp.Message,
		)
	}

	for i, artwork := range resp.Body.Illusts {
		if err := artwork.PopulateThumbnails(); err != nil {
			audit.GlobalAuditor.Logger.Errorf(
				"Failed to populate thumbnails for artwork ID %s: %v",
				artwork.ID,
				err,
			)

			return nil, fmt.Errorf("failed to populate thumbnails for artwork ID %s: %w", artwork.ID, err)
		}
		resp.Body.Illusts[i] = artwork
	}

	return resp.Body.Illusts, nil
}

func PopulateArtworkRecent(r *http.Request, userID string, recentWorkIDs []int) ([]ArtworkBrief, error) {
	idsString := ""
	for _, id := range recentWorkIDs {
		idsString += fmt.Sprintf("&ids[]=%d", id)
	}

	recent, err := populateArtworkIDs(r, userID, idsString)
	if err != nil {
		return nil, err
	}

	// Populate thumbnails for each artwork
	for i, artwork := range recent {
		if err := artwork.PopulateThumbnails(); err != nil {
			audit.GlobalAuditor.Logger.Errorf("Failed to populate thumbnails for artwork ID %s: %v", artwork.ID, err)

			return nil, fmt.Errorf("failed to populate thumbnails for artwork ID %s: %w", artwork.ID, err)
		}
		artwork.Thumbnail = artwork.Thumbnails.Medium
		recent[i] = artwork
	}

	sort.Slice(recent, func(i, j int) bool {
		return numberGreaterThan(recent[i].ID, recent[j].ID)
	})

	return recent, nil
}
