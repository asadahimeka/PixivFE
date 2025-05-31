// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"

	"codeberg.org/pixivfe/pixivfe/audit"
)

// HTML is a type alias for template.HTML.
type HTML = template.HTML

// OptionalStrMap is a generic type that handles JSON responses that may return either
// a map object with string keysor an empty array.
//
// This type is necessary because the pixiv API can return [] instead of {}
// when no data is available.
//
// Example JSON responses that this type can handle:
//   - Valid map:   {"key": {"value": "data"}}
//   - Empty data:  []
type OptionalStrMap[V any] map[string]V

// UnmarshalJSON implements custom JSON unmarshaling for FlexibleMap.
//
// If an empty array is received, the map will be set to nil.
// Otherwise, the data will be unmarshaled as a standard map.
func (m *OptionalStrMap[V]) UnmarshalJSON(data []byte) error {
	// Handle empty array case
	if bytes.Equal(data, []byte("[]")) {
		*m = nil
		return nil
	}

	// Handle normal map case
	var nm map[string]V
	if err := json.Unmarshal(data, &nm); err != nil {
		return err
	}
	*m = OptionalStrMap[V](nm)
	return nil
}

// OptionalIntMap is a generic type that handles JSON responses that may return either
// a map object with integer keys or an empty array.
//
// This type is necessary because the pixiv APIs can return [] instead of {}
// when no data is available.
//
// Example JSON responses that this type can handle:
//   - Valid map:   {"123": {"value": "data"}}
//   - Empty data:  []
type OptionalIntMap[V any] map[int]V

// UnmarshalJSON implements custom JSON unmarshaling for OptionalIntMap.
//
// If an empty array is received, the map will be set to nil.
// Otherwise, the data will be unmarshaled as a standard map.
func (m *OptionalIntMap[V]) UnmarshalJSON(data []byte) error {
	// Handle empty array case
	if bytes.Equal(data, []byte("[]")) {
		*m = nil
		return nil
	}

	// Handle normal map case
	var nm map[int]V
	if err := json.Unmarshal(data, &nm); err != nil {
		return err
	}
	*m = OptionalIntMap[V](nm)
	return nil
}

// ExtractIDs is a method of OptionalIntMap to extract IDs and count from the map.
func (m OptionalIntMap[T]) ExtractIDs() ([]int, int) {
	ids := make([]int, 0, len(m))

	for k := range m {
		ids = append(ids, k)
	}

	return ids, len(m)
}

/*
pixiv returns tags in two distinct formats in the standard desktop API:
	- Array-based with metadata, enclosed inside a tags array (type 1)
  - Map-based with language translations, enclosed inside a tagTranslation object (type 2)
  - Array-based with no metadata, enclosed inside a body array (type 3)

Tags as type 1 can be unmarshalled directly as a []Tag, but code interfacing with endpoints
that return the type 2 format should call TagTranslationsToTag to allow for easier template
reuse in the frontend

Peak pixiv moment:
	tagTranslation will be returned as an empty *array* rather than
	an empty object when there are no tagTranslations

	this is handled by TagTranslationWrapper which uses a custom UnmarshalJSON method

	*do not* create a map[string]TagTranslations directly; Go will be unhappy when it
	hits this edge case

Response examples:

```json # type 1
{
  "tags": [
    {
      "tag": "漫画",
      "locked": true,
      "deletable": false,
      "userId": "67084033",
      "romaji": "mannga",
      "translation": {
        "en": "manga"
      },
      "userName": "postcard"
    }
  ]
}
```

```json # type 2
{
  "tagTranslation": {
    "漫画": {
      "en": "manga",
      "ko": "만화",
      "zh": "",
      "zh_tw": "漫畫",
      "romaji": "mannga"
    }
  }
}
```

```json # type 2 (no data)
{
	"tagTranslation": []
}
```

```json # type 3
{
"{
  "body": [
    {
      "tag": "足裏",
      "tag_translation": "sole"
    }
  ]
}
```
*/

// Type 1 format
type Tag struct {
	Name            string          `json:"tag"`         // Name of the tag
	Locked          bool            `json:"locked"`      // Whether the tag can be edited by other users
	Deletable       bool            `json:"deletable"`   // Whether the tag can be removed
	UserID          string          `json:"userId"`      // userId of the user that added the tag
	Romaji          string          `json:"romaji"`      // Japanese romanization of the tag
	TagTranslations TagTranslations `json:"translation"` //
	UserName        string          `json:"userName"`    // userName of the user that added the tag
}

// Type 2 format
// Can be standalone or embedded in type 1
//
//nolint:tagliatelle
type TagTranslations struct {
	En     string `json:"en"`     // English translation
	Ko     string `json:"ko"`     // Korean translation
	Zh     string `json:"zh"`     // Simplified Chinese translation
	ZhTw   string `json:"zh_tw"`  // Traditional Chinese translation
	Romaji string `json:"romaji"` // Japanese romanization, use Tag.Romaji instead of this field
}

// Type 3 format
//
//nolint:tagliatelle
type SimpleTag struct {
	Name        string `json:"tag"`
	Translation string `json:"tag_translation"`
}

type TagTranslationWrapper map[string]TagTranslations

func (t *TagTranslationWrapper) UnmarshalJSON(data []byte) error {
	// Try unmarshaling as map first
	var asMap map[string]TagTranslations
	if err := json.Unmarshal(data, &asMap); err == nil {
		*t = TagTranslationWrapper(asMap)
		return nil
	}

	// If that fails, try as array
	var asArray []any
	if err := json.Unmarshal(data, &asArray); err == nil {
		// If it's an array, return empty map
		*t = make(TagTranslationWrapper)
		return nil
	}

	// If both fail, try one more time as map to get original error
	var finalTry map[string]TagTranslations
	return json.Unmarshal(data, &finalTry)
}

// TagTranslationsToTags converts tag translations into a Tag slice.
//
// If tagNames is provided (non-nil and non-empty), the returned slice preserves that order and filters duplicates.
// Otherwise, the result is built by iterating over the keys of tagTranslations.
func TagTranslationsToTags(tagNames []string, tagTranslations TagTranslationWrapper) []Tag {
	// When tagNames is not provided, build []Tag from tagTranslations map keys.
	if len(tagNames) == 0 {
		result := make([]Tag, 0, len(tagTranslations))
		for name, translation := range tagTranslations {
			result = append(result, Tag{
				Name:            name,
				TagTranslations: translation,
				Romaji:          translation.Romaji,
			})
		}
		return result
	}

	// When tagNames is provided, iterate over it and ignore any duplicates.
	seen := make(map[string]bool, len(tagNames))
	result := make([]Tag, 0, len(tagNames))

	for _, name := range tagNames {
		if seen[name] {
			continue
		}
		seen[name] = true

		tag := Tag{Name: name}
		if translation, ok := tagTranslations[name]; ok {
			tag.TagTranslations = translation
			tag.Romaji = translation.Romaji
		}

		result = append(result, tag)
	}
	return result
}

// SimpleTagsToTag creates a []Tag from a []SimpleTag
func SimpleTagsToTag(simpleTags []SimpleTag) []Tag {
	result := make([]Tag, len(simpleTags))

	for i, simpleTag := range simpleTags {
		result[i] = Tag{
			Name: simpleTag.Name,
			TagTranslations: TagTranslations{
				En: simpleTag.Translation,
			},
		}
	}

	return result
}

// Thumbnails holds different size variants of an artwork image.
//
// Each variant is generated from the original image URL using specific dimension and path transformations.
//
// The thumbnail URLs follow this pattern:
// https://i.pximg.net/{size_quality}/img/YYYY/MM/DD/HH/MM/SS/{id}_{page}_{filename_suffix}.{file_extension}
//
// Size variants are generated by modifying the size-quality segment:
//   - Tiny:     		250x250_80_a2
//   - Small:    		360x360_70
//   - Medium:   		600x600
//   - Large:    		1200x1200
//   - Original: 		Full-size source image obtained by removing the size-quality segment, forcing `category` as `img-original`, and stripping any `filename_suffix`
//
// Parameters:
//   - {size_quality}: Optional, in the format `/c/{width}x{height}{quality}`
//   - {width} and {height}: integers
//   - {quality}: quality percentage (e.g. _90) and optional modifier (e.g. _a2, _webp)
//   - {category}: Required, can be one of the following: `custom-thumb`, `custom1200`, `img-master`, and `img-original`
//   - {id}: Required, the artwork ID
//   - {page}: Required, in the format `p{page_number}`
//   - {page_number}: zero-indexed integer that increments for each page
//   - {filename_suffix}: Optional, can be of the following: `square`, `custom`, or `master`
//   - {file_extension}: Required, either .png or .jpg for Original, but always .jpg otherwise (even webp images have a .jpg extension)
//
// Examples:
//   - MasterLarge:		/img-master/img/2025/02/08/23/08/44/127035726_p0_master1200.jpg 	{filename_suffix} is `master1200`, {file_extension} is .jpg
//   - OriginalJPG:		/img-original/img/2025/02/08/23/08/44/127035726_p0.jpg 						{category} is `img-original`, {file_extension} is .jpg
//   - OriginalPNG		/img-original/img/2023/08/20/06/04/20/110992799_p0.png 						{category} is `img-original`, {file_extension} is .png
type Thumbnails struct {
	Width           int        // Width of the Original image
	Height          int        // Height of the Original image
	Tiny            string     // 250x250 thumbnail
	Small           string     // 360x360 thumbnail
	Medium          string     // 600x600 thumbnail
	Large           string     // 1200x1200 thumbnail
	MasterLarge     string     // Original aspect ratio, width limited to 1200px
	MasterWebp_1200 string     // Original aspect ratio, width limited to 1200px, WebP format
	Original        string     // Full-size original artwork, JPG or PNG format, returned by the API
	OriginalJPG     string     // Full-size original artwork, JPG format
	OriginalPNG     string     // Full-size original artwork, PNG format
	Webp_360        string     // 360x360 thumbnail, quality 10, WebP format
	Webp_540        string     // 540x540 thumbnail, quality 10, WebP format
	Webp_1200       string     // 1200x1200 thumbnail, quality 90, WebP format
	Video           string     // Video URL for ugoira
	IllustType      IllustType // Artwork type
}

func (work *ArtworkBrief) PopulateThumbnails() error {
	thumbnails, err := PopulateThumbnailsFor(work.Thumbnail)
	if err != nil {
		return err
	}
	work.Thumbnails = thumbnails
	return nil
}

// Precompiled regex
var (
	// match the "/c/{parameters}/" segment
	SizeQualityRe = regexp.MustCompile(`/c/[^/]+/`)

	// match any suffix that starts with an underscore and is
	// followed by non-slash characters before the file extension
	filenameSuffixRe = regexp.MustCompile(`_[^/_]+\.(jpg|png|jpeg)$`)

	customThumbRe      = regexp.MustCompile(`/custom-thumb/`)
	masterFileSuffixRe = regexp.MustCompile(`_(square|custom|master)1200\.(jpg|png|jpeg)$`)
)

// PopulateThumbnailsFor is a helper function that populates all thumbnail sizes, including Original.
func PopulateThumbnailsFor(thumbnailURL string) (Thumbnails, error) {
	var thumbnails Thumbnails

	// auditor.SugaredLogger.Debugf("PopulateThumbnails called with Thumbnail URL: %s", thumbnailURL)

	// Parse the original Thumbnail URL to ensure it's valid
	parsedURL, err := url.Parse(thumbnailURL)
	if err != nil {
		return thumbnails, fmt.Errorf("invalid Thumbnail URL '%s': %w", thumbnailURL, err)
	}

	// Verify that the Thumbnail URL contains the expected pattern
	if !SizeQualityRe.MatchString(parsedURL.Path) {
		audit.GlobalAuditor.Logger.Warnf("Thumbnail URL does not match expected pattern: %s. Using original URL for all thumbnail sizes.", thumbnailURL)
		thumbnails.Tiny = thumbnailURL
		thumbnails.Small = thumbnailURL
		thumbnails.Medium = thumbnailURL
		thumbnails.Large = thumbnailURL
		thumbnails.MasterLarge = thumbnailURL
		thumbnails.OriginalJPG = thumbnailURL
		thumbnails.OriginalPNG = thumbnailURL
		return thumbnails, nil
	}

	// Define the desired sizes for the thumbnails along with corresponding fields
	thumbSizes := []struct {
		name  string
		size  string
		field *string
	}{
		{"Tiny", "250x250_80_a2", &thumbnails.Tiny},
		{"Small", "360x360_70", &thumbnails.Small},
		{"Medium", "600x600", &thumbnails.Medium},
		{"Large", "1200x1200", &thumbnails.Large},
		{"Webp_360", "360x360_10_webp", &thumbnails.Webp_360},
		{"Webp_540", "540x540_10_webp", &thumbnails.Webp_540},
		{"Webp_1200", "1200x1200_90_webp", &thumbnails.Webp_1200},
	}

	// Generate regular thumbnails
	for _, thumb := range thumbSizes {
		finalURL, err := generateThumbnailURL(parsedURL, SizeQualityRe, thumb.size)
		if err != nil {
			return thumbnails, fmt.Errorf("error generating thumbnail URL for size %s: %w", thumb.size, err)
		}
		*thumb.field = finalURL
	}

	// Generate MasterLarge URL
	thumbnails.MasterLarge = generateMasterLargeURL(parsedURL, SizeQualityRe)

	// Generate MasterWebp_1200 URL
	thumbnails.MasterWebp_1200 = GenerateMasterWebpURL(parsedURL, SizeQualityRe)

	// Generate illustration and manga original URLs
	thumbnails.OriginalJPG = generateOriginalURL(parsedURL, SizeQualityRe, "jpg")
	thumbnails.OriginalPNG = generateOriginalURL(parsedURL, SizeQualityRe, "png")

	return thumbnails, nil
}

// generateThumbnailURL constructs a thumbnail URL for a given size.
func generateThumbnailURL(parsedURL *url.URL, re *regexp.Regexp, size string) (string, error) {
	newPath := re.ReplaceAllString(parsedURL.Path, fmt.Sprintf("/c/%s/", size))

	// If the new path is the same as the original, it means either:
	// - The URL is already at the desired size.
	// - The regex did not match, which has been handled earlier in PopulateThumbnailsFor.
	// In both cases, returning the original URL is acceptable.
	if newPath == parsedURL.Path {
		return parsedURL.String(), nil
	}

	updatedURL := *parsedURL // Create a copy of the original URL
	updatedURL.Path = newPath
	return updatedURL.String(), nil
}

func generateMasterLargeURL(parsedURL *url.URL, re *regexp.Regexp) string {
	// Remove the size/quality segment
	newPath := re.ReplaceAllString(parsedURL.Path, "/")

	// Ensure we're using img-master
	newPath = customThumbRe.ReplaceAllString(newPath, "/img-master/")
	if !strings.Contains(newPath, "img-master") {
		newPath = strings.Replace(newPath, "/img/", "/img-master/img/", 1)
	}

	// Replace the filename suffix with master1200
	newPath = masterFileSuffixRe.ReplaceAllString(newPath, "_master1200.$2")

	// Clean up the path
	newPath = path.Clean(newPath)

	updatedURL := *parsedURL
	updatedURL.Path = newPath
	return updatedURL.String()
}

func GenerateMasterWebpURL(parsedURL *url.URL, re *regexp.Regexp) string {
	// Apply WebP size/quality parameters
	newPath := re.ReplaceAllString(parsedURL.Path, "/c/1200x1200_90_webp/")

	// Ensure we're using img-master
	newPath = customThumbRe.ReplaceAllString(newPath, "/img-master/")
	if !strings.Contains(newPath, "img-master") {
		newPath = strings.Replace(newPath, "/img/", "/img-master/img/", 1)
	}

	// Replace the filename suffix with master1200
	newPath = masterFileSuffixRe.ReplaceAllString(newPath, "_master1200.$2")

	// Clean up the path
	newPath = path.Clean(newPath)

	updatedURL := *parsedURL
	updatedURL.Path = newPath
	return updatedURL.String()
}

// generateOriginalURL constructs the original image URL.
func generateOriginalURL(parsedURL *url.URL, re *regexp.Regexp, extension string) string {
	clonedURL := *parsedURL

	// Remove size/quality segment (e.g., /c/250x250_80_a2/... -> /...)
	clonedURL.Path = re.ReplaceAllString(clonedURL.Path, "/")

	// Convert either '/custom-thumb/' or '/img-master/' to '/img-original/'
	//
	// A thumbnail URL can only have one or the other, not both
	if strings.Contains(clonedURL.Path, "/custom-thumb/") {
		clonedURL.Path = strings.Replace(clonedURL.Path, "/custom-thumb/", "/img-original/", 1)
	} else if strings.Contains(clonedURL.Path, "/img-master/") {
		clonedURL.Path = strings.Replace(clonedURL.Path, "/img-master/", "/img-original/", 1)
	}

	// Remove filename suffix and force specified extension
	clonedURL.Path = filenameSuffixRe.ReplaceAllString(clonedURL.Path, "."+extension)

	// Clean the path to resolve redundant elements (e.g., '//' -> '/')
	clonedURL.Path = path.Clean(clonedURL.Path)

	return clonedURL.String()
}

func GetOriginalAvatarURL(avatarURL string) string {
	if avatarURL == "" {
		return ""
	}

	// Find the last underscore and number segment before file extension
	lastDotIndex := strings.LastIndex(avatarURL, ".")
	if lastDotIndex == -1 {
		return avatarURL
	}

	lastUnderscoreIndex := strings.LastIndex(avatarURL[:lastDotIndex], "_")
	if lastUnderscoreIndex == -1 {
		return avatarURL
	}

	// Check if characters between underscore and dot are numeric
	numericPart := avatarURL[lastUnderscoreIndex+1 : lastDotIndex]
	if _, err := strconv.Atoi(numericPart); err != nil {
		return avatarURL
	}

	// Remove the _* segment
	return avatarURL[:lastUnderscoreIndex] + avatarURL[lastDotIndex:]
}
