// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
Types for core/user.go
*/
package core

import (
	"net/url"
	"slices"
	"sort"
	"strings"

	"codeberg.org/pixivfe/pixivfe/i18n"
)

// UserWorkCategory represents a unified structure for handling different work categories.
type UserWorkCategory struct {
	Value        string         // Category identifier (e.g., "illustrations", "manga")
	PageLimit    int            // Maximum number of pages for pagination
	FrequentTags []Tag          // Frequently used tags within the category
	IllustWorks  []ArtworkBrief // Populated if the work type requires ArtworkBrief
	NovelWorks   []NovelBrief   // Populated if the work type requires NovelBrief
	WorkIDs      string         // Concatenated string of work IDs
	WorkCount    int            // Number of works for the category
	MangaSeries  []IllustSeries // Populated for the "manga" category
	NovelSeries  []NovelSeries  // Populated for the "novels" category
	Users        []User         // Populated for the "following" and "followers" category
}

// Category constants define the possible work category types
const (
	CategoryValueEmpty         = ""
	CategoryValueArtworks      = "artworks"
	CategoryValueIllustrations = "illustrations"
	CategoryValueManga         = "manga"
	CategoryValueBookmarks     = "bookmarks"
	CategoryValueNovels        = "novels"
	CategoryValueFollowing     = "following"
	CategoryValueFollowers     = "followers"
)

// Pre-defined UserWorkCategory instances
var (
	CategoryAny          = &UserWorkCategory{Value: CategoryValueEmpty}
	CategoryAnyAlt       = &UserWorkCategory{Value: CategoryValueArtworks}
	CategoryIllustration = &UserWorkCategory{Value: CategoryValueIllustrations}
	CategoryManga        = &UserWorkCategory{Value: CategoryValueManga}
	CategoryBookmarks    = &UserWorkCategory{Value: CategoryValueBookmarks}
	CategoryNovels       = &UserWorkCategory{Value: CategoryValueNovels}
	CategoryFollowing    = &UserWorkCategory{Value: CategoryValueFollowing}
	CategoryFollowers    = &UserWorkCategory{Value: CategoryValueFollowers}
)

// NewUserWorkCategory creates a new UserWorkCategory with the given value.
func NewUserWorkCategory(value string) UserWorkCategory {
	return UserWorkCategory{
		Value:       value,
		IllustWorks: nil,
		WorkCount:   0,
	}
}

// Validate checks if the current work category is valid.
func (cat *UserWorkCategory) Validate() error {
	validValues := []string{
		CategoryValueEmpty,
		CategoryValueArtworks,
		CategoryValueIllustrations,
		CategoryValueManga,
		CategoryValueBookmarks,
		CategoryValueNovels,
		CategoryValueFollowing,
		/*
			NOTE: Implementing the Followers category is low priority given
						that you can only see your own followers, and thus is mostly
					 	only useful for users that actually post to the platform
		*/
		CategoryValueFollowers,
	}

	if slices.Contains(validValues, cat.Value) {
		return nil
	}
	return i18n.Errorf(`Invalid work category: %#v.`, cat.Value)
}

// SetPageLimit sets the maximum number of pages for the category.
func (cat *UserWorkCategory) SetPageLimit(limit int) {
	cat.PageLimit = limit
}

// SocialEntry represents a single social media entry with platform and URL.
type SocialEntry struct {
	Platform string
	URL      string
}

// PersonalField represents a key/value pair for a personal field.
type PersonalField struct {
	Key   string
	Value string
}

// WorkspaceItem represents a key/value pair for workspace details.
type WorkspaceItem struct {
	Key   string
	Value string
}

// User represents a user.
type User struct {
	ID             string                            `json:"userId"`
	Name           string                            `json:"name"`
	Image          string                            `json:"image"`    // Unimplemented
	Avatar         string                            `json:"imageBig"` // Higher resolution avatar
	AvatarOriginal string                            // Original resolution avatar
	Premium        bool                              `json:"premium"`    // Unimplemented
	IsFollowed     bool                              `json:"isFollowed"` // Whether the logged-in user is following this user
	IsMyPixiv      bool                              `json:"isMypixiv"`  // Unimplemented
	IsBlocking     bool                              `json:"isBlocking"` // Unimplemented
	Background     map[string]any                    `json:"background"`
	SketchLiveID   string                            `json:"sketchLiveId"` // Unimplemented
	Partial        int                               `json:"partial"`      // Unimplemented
	SketchLives    []any                             `json:"sketchLives"`  // Unimplemented
	Commission     any                               `json:"commission"`   // Unimplemented
	Following      int                               `json:"following"`
	MyPixiv        int                               `json:"mypixivCount"`
	FollowedBack   bool                              `json:"followedBack"` // Unimplemented
	Comment        string                            `json:"comment"`      // Biography
	CommentHTML    HTML                              `json:"commentHtml"`  // HTML-formatted biography
	Webpage        string                            `json:"webpage"`
	SocialRaw      OptionalStrMap[map[string]string] `json:"social"`
	CanSendMessage bool                              `json:"canSendMessage"` // Unimplemented
	Region         struct {
		Name             string
		Region           string
		Prefecture       string
		ParsedPrefecture string
		PrivacyLevel     string
	} `json:"region"`
	Age struct {
		Name         string // terrible naming, should be `Value`, not `Name`; pixiv moment
		PrivacyLevel string
	} `json:"age"`
	BirthDay struct {
		Name         string
		PrivacyLevel string
	} `json:"birthDay"`
	Gender struct {
		Name         string
		PrivacyLevel string
	} `json:"gender"`
	Job struct {
		Name         string
		PrivacyLevel string
	} `json:"job"`
	Workspace struct {
		UserWorkspacePc     string
		UserWorkspaceTool   string
		UserWorkspaceTablet string
		UserWorkspaceMouse  string
	} `json:"workspace"`
	Official bool `json:"official"`
	Group    any  `json:"group"`

	// The following fields are internal to PixivFE
	Social                  []SocialEntry
	BackgroundImage         string
	BackgroundImageOriginal string                       // Original resolution background image
	Categories              map[string]*UserWorkCategory // Holds work categories
	Artworks                []ArtworkBrief               // Populated on user discovery and following/follower pages
	Novels                  []NovelBrief                 // Populated on user discovery and following/follower pages
}

// PersonalFields returns a slice of personal fields for a user.
func (u *User) PersonalFields() []PersonalField {
	return []PersonalField{
		{"Age", u.Age.Name},
		{"Birthday", u.BirthDay.Name}, // Renamed to "Birthday" for the UI
		{"Gender", u.Gender.Name},
		{"Job", u.Job.Name},
	}
}

// WorkspaceItems returns a slice of workspace items for a user.
func (u *User) WorkspaceItems() []WorkspaceItem {
	return []WorkspaceItem{
		{"PC", u.Workspace.UserWorkspacePc},
		{"Tool", u.Workspace.UserWorkspaceTool},
		{"Tablet", u.Workspace.UserWorkspaceTablet},
		{"Mouse", u.Workspace.UserWorkspaceMouse},
	}
}

// parseSocial parses the social data for a user.
func (u *User) parseSocial() {
	// Convert to sorted slice
	u.Social = make([]SocialEntry, 0, len(u.SocialRaw))
	for platform, data := range u.SocialRaw {
		u.Social = append(u.Social, SocialEntry{
			Platform: platform,
			URL:      data["url"],
		})
	}
}

// sortSocial sorts the social data for a user.
func (u *User) sortSocial() {
	sort.Slice(u.Social, func(i, j int) bool {
		return u.Social[i].Platform < u.Social[j].Platform
	})
}

// webpageToSocialEntry converts a webpage URL to a SocialEntry by extracting
// the second-level domain as the platform name.
func (u *User) webpageToSocialEntry() *SocialEntry {
	if u.Webpage == "" {
		return nil
	}

	// Ensure URL has protocol prefix.
	urlStr := u.Webpage
	if !strings.HasPrefix(strings.ToLower(urlStr), "http://") &&
		!strings.HasPrefix(strings.ToLower(urlStr), "https://") {
		urlStr = "https://" + urlStr
	}

	// Parse the URL.
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil
	}

	// Split the host into parts.
	parts := strings.Split(parsedURL.Host, ".")
	if len(parts) < 2 {
		return nil
	}

	// Extract second-level domain (last two parts).
	domainIndex := len(parts) - 2
	if domainIndex < 0 {
		return nil
	}

	// Create and return SocialEntry.
	return &SocialEntry{
		Platform: parts[domainIndex], // Use second-level domain.
		URL:      u.Webpage,          // Use original URL.
	}
}

var prefectures = map[string]string{
	"1":  "Hokkaido",
	"2":  "Aomori",
	"3":  "Iwate",
	"4":  "Miyagi",
	"5":  "Akita",
	"6":  "Yamagata",
	"7":  "Fukushima",
	"8":  "Ibaraki",
	"9":  "Tochigi",
	"10": "Gunma",
	"11": "Saitama",
	"12": "Chiba",
	"13": "Tokyo",
	"14": "Kanagawa",
	"15": "Niigata",
	"16": "Toyama",
	"17": "Ishikawa",
	"18": "Fukui",
	"19": "Yamanashi",
	"20": "Nagano",
	"21": "Gifu",
	"22": "Shizuoka",
	"23": "Aichi",
	"24": "Mie",
	"25": "Shiga",
	"26": "Kyoto",
	"27": "Osaka",
	"28": "Hyogo",
	"29": "Nara",
	"30": "Wakayama",
	"31": "Tottori",
	"32": "Shimane",
	"33": "Okayama",
	"34": "Hiroshima",
	"35": "Yamaguchi",
	"36": "Tokushima",
	"37": "Kagawa",
	"38": "Ehime",
	"39": "Kochi",
	"40": "Fukuoka",
	"41": "Saga",
	"42": "Nagasaki",
	"43": "Kumamoto",
	"44": "Oita",
	"45": "Miyazaki",
	"46": "Kagoshima",
	"47": "Okinawa",
}
