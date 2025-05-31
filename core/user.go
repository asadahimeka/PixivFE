// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"codeberg.org/pixivfe/pixivfe/audit"
	"codeberg.org/pixivfe/pixivfe/core/requests"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"github.com/goccy/go-json"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	userFollowingPerPage = 12 // API uses limit=12
)

// GetUserProfile retrieves the user profile, including counts, artworks/bookmarks, and social data.
//
// Goroutines are used to avoid blocking on network requests.
func GetUserProfile(r *http.Request, id string, currentCategory *UserWorkCategory, page int, getTags bool, mode string) (User, error) {
	audit.GlobalAuditor.Logger.Debug("GetUserProfile called",
		zap.String("userID", id),
		zap.String("currentCategory", currentCategory.Value),
		zap.Int("page", page),
		zap.Bool("getTags", getTags),
	)

	var (
		userInfo   User
		categories map[string]*UserWorkCategory
		errGroup   errgroup.Group
	)

	// Goroutine to fetch user information
	errGroup.Go(func() error {
		// Fetch basic user information
		var err error
		userInfo, err = GetUserBasicInformation(r, id)
		if err != nil {
			audit.GlobalAuditor.Logger.Error("Failed to fetch user information", zap.Error(err))
			return err
		}

		// Populate parsed prefecture if available
		if userInfo.Region.Prefecture != "" {
			if prefName, ok := prefectures[userInfo.Region.Prefecture]; ok {
				userInfo.Region.ParsedPrefecture = prefName
			}
		}

		// Set original avatar URL
		userInfo.AvatarOriginal = GetOriginalAvatarURL(userInfo.Avatar)

		// Parse social data
		audit.GlobalAuditor.Logger.Debug("Parsing social data")
		userInfo.parseSocial()
		audit.GlobalAuditor.Logger.Debug("Social data parsed", zap.Any("social", userInfo.Social))

		// Add webpage as social entry if available
		if webpageEntry := userInfo.webpageToSocialEntry(); webpageEntry != nil {
			// Check for duplicate platform
			isDuplicate := false

			for _, entry := range userInfo.Social {
				if entry.Platform == webpageEntry.Platform {
					isDuplicate = true

					break
				}
			}

			// Only append if not a duplicate
			if !isDuplicate {
				userInfo.Social = append(userInfo.Social, *webpageEntry)
				audit.GlobalAuditor.Logger.Debug("Added webpage as social entry", zap.Any("webpageEntry", webpageEntry))
			} else {
				audit.GlobalAuditor.Logger.Debug("Skipped adding duplicate webpage platform", "platform", webpageEntry.Platform)
			}
		}

		// Sort social entries
		userInfo.sortSocial()
		audit.GlobalAuditor.Logger.Debug("Sorted social entries", zap.Any("social", userInfo.Social))

		// Set background image if available
		if userInfo.Background == nil {
			// No background image to process
			audit.GlobalAuditor.Logger.Debug("No background image available")
		} else {
			backgroundURL, ok := userInfo.Background["url"].(string)
			if !ok {
				audit.GlobalAuditor.Logger.Warn("Background URL not found or invalid")
			} else {
				// We have a valid background URL
				userInfo.BackgroundImage = backgroundURL
				audit.GlobalAuditor.Logger.Debug("Background image set", zap.String("BackgroundImage", backgroundURL))

				// Default to current URL for original in case parsing fails
				userInfo.BackgroundImageOriginal = backgroundURL

				// Try to create the original URL by parsing
				parsedURL, err := url.Parse(backgroundURL)
				if err != nil {
					audit.GlobalAuditor.Logger.Error("Failed to parse background image URL", zap.Error(err))
				} else {
					// Remove the /c/.../ segment from the path
					originalPath := SizeQualityRe.ReplaceAllString(parsedURL.Path, "/")
					parsedURL.Path = originalPath
					userInfo.BackgroundImageOriginal = parsedURL.String()
				}
			}
		}

		return nil
	})

	// Goroutine to fetch works and populate categories
	errGroup.Go(func() error {
		var err error

		audit.GlobalAuditor.Logger.Debug("Getting populated works")
		categories, err = getPopulatedWorks(r, id, currentCategory, page, getTags, mode)
		if err != nil {
			audit.GlobalAuditor.Logger.Error("Failed to get populated works", zap.Error(err))
			return err
		}
		return nil
	})

	// Wait for both goroutines to finish
	if err := errGroup.Wait(); err != nil {
		return User{}, err
	}

	// Merge userInfo and categories into the final user struct
	user := userInfo
	user.Categories = categories

	return user, nil
}

// GetUserBasicInformation retrieves basic information for a given user ID
func GetUserBasicInformation(r *http.Request, id string) (User, error) {
	var user User

	url := GetUserInformationURL(id, "1")

	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	rawResp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
	if err != nil {
		return user, err
	}
	rawResp = RewriteContentURLs(r, rawResp)

	err = json.Unmarshal(rawResp, &user)
	if err != nil {
		return user, err
	}

	return user, nil
}

// getPopulatedWorks fetches then populates information for a user's works, as well as handling their
// frequently used tags and series data.
//
// Goroutines are used to avoid blocking on network requests.
func getPopulatedWorks(r *http.Request, id string, currentCategory *UserWorkCategory, page int, getTags bool, mode string) (map[string]*UserWorkCategory, error) {
	audit.GlobalAuditor.Logger.Debug("getPopulatedWorks called",
		zap.String("userID", id),
		zap.String("currentCategory", currentCategory.Value),
		zap.Int("page", page),
		zap.Bool("getTags", getTags),
	)

	// Fetch work IDs and series data
	audit.GlobalAuditor.Logger.Debug("Fetching work IDs and series data for works")
	categoriesData, err := fetchWorkIDsAndSeriesData(r, id, currentCategory, page)
	if err != nil {
		audit.GlobalAuditor.Logger.Error("Failed to fetch work IDs and series data", zap.Error(err))
		return nil, err
	}

	// Initialize user-based categories if currentCategory is set accordingly
	if currentCategory != nil {
		currentCatValue := currentCategory.Value
		if currentCatValue == CategoryValueFollowing || currentCatValue == CategoryValueFollowers {
			if _, exists := categoriesData[currentCatValue]; !exists {
				categoriesData[currentCatValue] = &UserWorkCategory{
					Value: currentCatValue,
				}
			}
		}
	}

	// Create an errgroup
	var errGroup errgroup.Group

	// Fetch and populate works for each category in parallel
	for _, cat := range categoriesData {
		// Skip categories with no work IDs which can have them
		if cat.WorkIDs == "" &&
			cat.Value != CategoryValueBookmarks &&
			cat.Value != CategoryValueFollowing &&
			cat.Value != CategoryValueFollowers {
			continue
		}

		errGroup.Go(func() error {
			switch cat.Value {
			case "illustrations", "manga":
				audit.GlobalAuditor.Logger.Debug("Fetching and populating ArtworkBriefs", zap.String("category", cat.Value))
				artworks, err := populateArtworkIDs(r, id, cat.WorkIDs)
				if err != nil {
					audit.GlobalAuditor.Logger.Error("Failed to populate artwork IDs", zap.Error(err))
					return err
				}
				cat.IllustWorks = artworks

			case "novels":
				audit.GlobalAuditor.Logger.Debug("Fetching and populating NovelBriefs", zap.String("category", cat.Value))
				novels, err := populateNovelIDs(r, id, cat.WorkIDs)
				if err != nil {
					audit.GlobalAuditor.Logger.Error("Failed to populate novel IDs", zap.Error(err))
					return err
				}
				cat.NovelWorks = novels

			case "bookmarks":
				audit.GlobalAuditor.Logger.Debug("Fetching and populating Bookmarks", zap.String("category", cat.Value))
				bookmarks, count, err := populateIllustBookmarks(r, id, mode, page)
				if err != nil {
					audit.GlobalAuditor.Logger.Error("Failed to populate bookmarked work IDs", zap.Error(err))
					return err
				}
				cat.IllustWorks = bookmarks
				cat.WorkCount = count
				cat.PageLimit = int(math.Ceil(float64(count) / BookmarksPageSize))

			case "following":
				audit.GlobalAuditor.Logger.Debug("Fetching and populating Following", zap.String("category", cat.Value))
				users, total, err := populateUserFollowing(r, id, mode, page)
				if err != nil {
					return err
				}
				cat.Users = users
				cat.WorkCount = total
				cat.PageLimit = int(math.Ceil(float64(total) / float64(userFollowingPerPage)))
			}

			return nil
		})

		errGroup.Go(func() error {
			// Fetch frequent tags if requested
			if getTags && (cat.Value != "bookmarks" && cat.Value != "following" && cat.Value != "followers") {
				audit.GlobalAuditor.Logger.Debug("Fetching frequent tags for category", zap.String("category", cat.Value))
				tags, err := fetchFrequentTags(r, cat.WorkIDs, cat)
				if err != nil {
					audit.GlobalAuditor.Logger.Error("Failed to fetch frequent tags",
						"category", cat.Value,
						err)
					return err
				}
				cat.FrequentTags = tags
			}

			return nil
		})
	}

	// Wait for all goroutines to finish
	if err := errGroup.Wait(); err != nil {
		return nil, err
	}

	// Set PageLimit for the current category
	if currentCategory != nil {
		if cat, ok := categoriesData[currentCategory.Value]; ok {
			currentCategory.PageLimit = cat.PageLimit
		}
	}

	return categoriesData, nil
}

// fetchWorkIDsAndSeriesData retrieves artwork IDs and series information for a user.
//
// It populates UserWorkCategory structs for each category.
func fetchWorkIDsAndSeriesData(r *http.Request, id string, currentCategory *UserWorkCategory, page int) (map[string]*UserWorkCategory, error) {
	audit.GlobalAuditor.Logger.Debug("fetchWorkIDsAndSeriesData called",
		zap.String("userID", id),
		zap.String("currentCategory", currentCategory.Value),
		zap.Int("page", page),
	)
	url := GetUserWorksURL(id)

	audit.GlobalAuditor.Logger.Debug("Fetching user works", zap.String("URL", url))

	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	rawResp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
	if err != nil {
		audit.GlobalAuditor.Logger.Error("requests.API_GET_UnwrapJson failed", zap.Error(err))
		return nil, err
	}

	audit.GlobalAuditor.Logger.Debug("User works fetched", "response", rawResp)

	rawResp = RewriteContentURLs(r, rawResp)

	var resp struct {
		Illusts     OptionalIntMap[*struct{}] `json:"illusts"`
		Manga       OptionalIntMap[*struct{}] `json:"manga"`
		MangaSeries []IllustSeries            `json:"mangaSeries"`
		Novels      OptionalIntMap[*struct{}] `json:"novels"`
		NovelSeries []NovelSeries             `json:"novelSeries"`
	}

	err = json.Unmarshal(rawResp, &resp)
	if err != nil {
		audit.GlobalAuditor.Logger.Error("Failed to unmarshal user works response", zap.Error(err))
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	audit.GlobalAuditor.Logger.Debug("User works response unmarshalled")

	categories := make(map[string]*UserWorkCategory)

	// Process illustrations
	illustCat := &UserWorkCategory{Value: "illustrations"}
	illustIDs, count := resp.Illusts.ExtractIDs()
	illustCat.WorkCount = count
	illustCat.WorkIDs = buildIDString(illustIDs, page, currentCategory, illustCat)
	categories["illustrations"] = illustCat

	// Process manga
	mangaCat := &UserWorkCategory{Value: "manga"}
	mangaIDs, count := resp.Manga.ExtractIDs()
	mangaCat.WorkCount = count
	mangaCat.WorkIDs = buildIDString(mangaIDs, page, currentCategory, mangaCat)

	// Process novels
	novelCat := &UserWorkCategory{Value: "novels"}
	novelIDs, count := resp.Novels.ExtractIDs()
	novelCat.WorkCount = count
	novelCat.WorkIDs = buildIDString(novelIDs, page, currentCategory, novelCat)

	// Handle MangaSeries
	mangaCat.MangaSeries = resp.MangaSeries
	categories["manga"] = mangaCat

	// Handle NovelSeries
	novelCat.NovelSeries = resp.NovelSeries
	categories["novels"] = novelCat

	// Create an empty bookmarks category
	//
	// We don't build an ID string here as fetchBookmarks populates IllustWorks without the need for a WorkIDs string
	// (which is also why we can't call fetchFrequentTags for the "bookmarks" category, though extracting the IDs from
	// ArtworkBrief would be possible)
	categories["bookmarks"] = &UserWorkCategory{Value: "bookmarks"}

	return categories, nil
}

// buildIDString builds the ID string for API requests and sets the PageLimit.
func buildIDString(ids []int, page int, currentCategory, cat *UserWorkCategory) string {
	sort.Sort(sort.Reverse(sort.IntSlice(ids)))
	worksPerPage := 30.0
	totalItems := len(ids)

	// We only use the actual page number for the current category being viewed
	// and default the page to 1 for other categories.
	//
	// This is so that we don't attempt to paginate categories that don't have enough
	// items, which would raise a spurious error from computeSliceBounds regarding an
	// invalid page number.
	//
	// NOTE: A different approach will be needed if we require pageLimits for inactive categories.
	effectivePage := page
	if currentCategory.Value != cat.Value {
		effectivePage = 1
	}

	start, end, pageLimit, err := computeSliceBounds(effectivePage, worksPerPage, totalItems)
	if err != nil {
		audit.GlobalAuditor.Logger.Error("Error computing slice bounds", zap.Error(err))
		return ""
	}

	if currentCategory.Value == cat.Value {
		cat.SetPageLimit(pageLimit)
	}

	idsToUse := ids[start:end]

	var idsBuilder strings.Builder
	for _, k := range idsToUse {
		idsBuilder.WriteString(fmt.Sprintf("&ids[]=%d", k))
	}
	return idsBuilder.String()
}

// fetchFrequentTags fetches a user's frequently used tags, based on category.
func fetchFrequentTags(r *http.Request, ids string, workCategory *UserWorkCategory) ([]Tag, error) {
	var simpleTags []SimpleTag
	var url string

	switch workCategory.Value {
	case "illustrations", "manga":
		url = GetArtworkFrequentTagsURL(ids)
	case "novels":
		url = GetNovelFrequentTagsURL(ids)
	default:
		return nil, fmt.Errorf("unsupported category: %s", workCategory.Value)
	}

	// Return early if there are no IDs
	//
	// NOTE: theoretically, this check should never evaluate as true since fetchUserWorks doesn't call
	// getUserFrequentTags when len(tagsIDs) == 0, instead returning an empty []Tag
	if ids == "" {
		return nil, nil
	}

	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	rawResp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(rawResp, &simpleTags)
	if err != nil {
		return nil, err
	}

	// Convert SimpleTag to Tag
	return SimpleTagsToTag(simpleTags), nil
}

// populateUserFollowing retrieves and populates the users followed by a given user ID
func populateUserFollowing(r *http.Request, id, mode string, page int) ([]User, int, error) {
	var resp UserFollowingResponse

	page--

	audit.GlobalAuditor.Logger.Debug("populateFollowingUsers called",
		zap.String("userID", id),
		zap.String("mode", mode),
		zap.Int("page", page),
	)

	if mode == "all" {
		mode = "show"
	}

	url := GetUserFollowingURL(id, page, userFollowingPerPage, mode)

	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	rawResp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
	if err != nil {
		return nil, 0, err
	}

	rawResp = RewriteContentURLs(r, rawResp)

	if err := json.Unmarshal(rawResp, &resp); err != nil {
		return nil, 0, err
	}

	users := make([]User, len(resp.Users))
	for i, apiUser := range resp.Users {
		users[i] = User{
			ID:             apiUser.UserId,
			Name:           apiUser.UserName,
			Avatar:         apiUser.ProfileImageUrl,
			AvatarOriginal: GetOriginalAvatarURL(apiUser.ProfileImageUrl),
			Comment:        apiUser.UserComment,
			IsFollowed:     apiUser.Following,
			FollowedBack:   apiUser.Followed,
			IsBlocking:     apiUser.IsBlocking,
			IsMyPixiv:      apiUser.IsMypixiv,
			Artworks:       apiUser.Illusts,
		}

		// Populate thumbnails for each artwork
		for j := range users[i].Artworks {
			if err := users[i].Artworks[j].PopulateThumbnails(); err != nil {
				return nil, 0, err
			}
		}
	}

	return users, resp.Total, nil
}

// populateWorkIDs populates a slice of type T using
// the "works" field of the JSON response from the provided URL.
//
// The URL should include work IDs in the format `&ids[]=123456`.
func populateWorkIDs[T ArtworkBrief | NovelBrief](r *http.Request, url string) ([]T, error) {
	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	rawResp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
	if err != nil {
		return nil, err
	}

	rawResp = RewriteContentURLs(r, rawResp)

	var resp struct {
		Works map[int]T `json:"works"`
	}

	if err := json.Unmarshal(rawResp, &resp); err != nil {
		return nil, err
	}

	works := make([]T, 0, len(resp.Works))
	for _, work := range resp.Works {
		works = append(works, work)
	}

	return works, nil
}

// populateArtworkIDs populates a []ArtworkBrief for a given set of artwork IDs.
func populateArtworkIDs(r *http.Request, id, ids string) ([]ArtworkBrief, error) {
	works, err := populateWorkIDs[ArtworkBrief](r, GetUserFullArtworkURL(id, ids))
	if err != nil {
		return nil, err
	}

	// Sort the works based on ID in descending order
	sort.Slice(works, func(i, j int) bool {
		return numberGreaterThan(works[i].ID, works[j].ID)
	})

	// Populate thumbnails for each artwork
	for idx := range works {
		artwork := &works[idx]
		if err := artwork.PopulateThumbnails(); err != nil {
			audit.GlobalAuditor.Logger.Errorf("Failed to populate thumbnails for artwork ID %s: %v", artwork.ID, err)
			return nil, fmt.Errorf("failed to populate thumbnails for artwork ID %s: %w", artwork.ID, err)
		}
	}

	return works, nil
}

// populateNovelIDs populates a []NovelBrief for a given set of novel IDs.
func populateNovelIDs(r *http.Request, id, ids string) ([]NovelBrief, error) {
	// Fetch the novels using the existing populateWorkIDs function
	works, err := populateWorkIDs[NovelBrief](r, GetUserFullNovelURL(id, ids))
	if err != nil {
		return nil, err
	}

	// Sort the works based on ID in descending order
	sort.Slice(works, func(i, j int) bool {
		return numberGreaterThan(works[i].ID, works[j].ID)
	})

	return works, nil
}

// populateIllustBookmarks populates a []ArtworkBrief for a given set of bookmarked work IDs.
//
// This function cannot be neatly refactored to use getWorkIDs due to having
// a different API response structure.
func populateIllustBookmarks(r *http.Request, id, mode string, page int) ([]ArtworkBrief, int, error) {
	page--

	if mode == "all" {
		mode = "show"
	}

	url := GetUserIllustBookmarksURL(id, mode, page)

	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	rawResp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
	if err != nil {
		return nil, -1, err
	}

	rawResp = RewriteContentURLs(r, rawResp)

	var resp struct {
		Artworks []json.RawMessage `json:"works"`
		Total    int               `json:"total"`
	}

	err = json.Unmarshal(rawResp, &resp)
	if err != nil {
		return nil, -1, err
	}

	artworks := make([]ArtworkBrief, len(resp.Artworks))

	for index, rawResp := range resp.Artworks {
		var artwork ArtworkBrief

		err = json.Unmarshal(rawResp, &artwork)
		if err != nil {
			artworks[index] = ArtworkBrief{
				ID:        "#",
				Title:     "Deleted or private",
				UserName:  "Deleted or private",
				Thumbnail: "https://s.pximg.net/common/images/limit_unknown_360.png",
			}

			continue
		}

		// Populate thumbnails
		if err := artwork.PopulateThumbnails(); err != nil {
			audit.GlobalAuditor.Logger.Errorf("Failed to populate thumbnails for artwork ID %s: %v", id, err)
			return nil, -1, fmt.Errorf("failed to populate thumbnails for artwork ID %s: %w", id, err)
		}
		artworks[index] = artwork
	}

	return artworks, resp.Total, nil
}

// computeSliceBounds is a utility function to compute slice bounds safely.
//
// It calculates the start and end indices for slicing based on pagination parameters.
//
// Parameters:
//   - page: The current page number (1-based)
//   - worksPerPage: Number of items to display per page
//   - totalItems: Total number of items available
//
// Returns:
//   - int: Start index for slicing (inclusive)
//   - int: End index for slicing (exclusive)
//   - int: Total number of pages available
//   - error: Error if page number is invalid (less than 1 or greater than page limit)
//     or nil if calculation succeeds
//
// If totalItems is 0, returns (0, 0, 0, nil) indicating no items to slice.
func computeSliceBounds(page int, worksPerPage float64, totalItems int) (int, int, int, error) {
	if totalItems == 0 {
		return 0, 0, 0, nil
	}

	pageLimit := int(math.Ceil(float64(totalItems) / worksPerPage))

	if page < 1 || page > pageLimit {
		return 0, 0, 0, fmt.Errorf("invalid page number")
	}

	start := (page - 1) * int(worksPerPage)
	end := min(start+int(worksPerPage), totalItems)

	return start, end, pageLimit, nil
}

func numberGreaterThan(l, r string) bool {
	if len(l) > len(r) {
		return true
	}

	if len(l) < len(r) {
		return false
	}
	return l > r
}
