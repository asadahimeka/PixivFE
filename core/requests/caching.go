// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
Upstream HTTP response caching.
*/
package requests

import (
	"crypto/rand"
	"encoding/binary"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"codeberg.org/pixivfe/pixivfe/audit"
	"codeberg.org/pixivfe/pixivfe/config"
	"github.com/zeebo/xxh3"
)

var (
	cacheSeed uint64
	cache     *LRUCache

	// excludedCachePaths lists API endpoints for which responses are never cached,
	// regardless of any other factors.
	excludedCachePaths = []string{
		"/ajax/discovery/artworks",
		"/ajax/discovery/novels",
		"/ajax/discovery/users",
		"/ajax/illust/new",
	}
)

// CachedItem represents a cached HTTP response along with its expiration time and original URL.
type CachedItem struct {
	Response  *SimpleHTTPResponse
	ExpiresAt time.Time
	URL       string
}

// CachePolicy defines the caching behavior for a request.
type CachePolicy struct {
	// Whether to attempt fetching from the cache
	// and store any OK response that we receieve.
	ShouldUseCache bool
	// The cached response if available and valid.
	CachedResponse *SimpleHTTPResponse
}

// Setup initializes the API response cache based on parameters in GlobalConfig.
//
// It sets up an LRU cache with a specified size and logs the cache parameters.
// If caching is disabled in the configuration, it skips initialization.
func Setup() {
	if !config.GlobalConfig.Cache.Enabled {
		audit.GlobalAuditor.Logger.Infow("Cache is disabled, skipping cache initialization")

		return
	}

	var err error

	// Initialize the LRU cache with the configured parameters.
	cache, err = NewLRUCache(config.GlobalConfig.Cache.Size)
	if err != nil {
		audit.GlobalAuditor.Logger.Panicf("Failed to create cache: %v", err)
	}

	// Create a byte slice to hold the random seed.
	var seedBytes [8]byte

	// Read 8 random bytes from the crypto/rand reader.
	_, err = rand.Read(seedBytes[:])
	if err != nil {
		audit.GlobalAuditor.Logger.Panicf("Failed to generate cache key seed: %v", err)
	}

	// Convert the byte slice to a uint64 seed using little endian.
	cacheSeed = binary.LittleEndian.Uint64(seedBytes[:])
}

// The `generateCacheKey` function securely binds cached responses to both the request URL and the full authenticated
// session token by combining them into a crypto-hashed identifier.
//
// Using only the user ID portion of the token for cache keys would expose two risks:
//  1. Validating a token's authenticity would require an API call, undermining cache efficiency;
//  2. Attackers could forge tokens containing valid user IDs (e.g., `123456_invalidSessionSecret`)
//     to access cached private data for arbitrary users.
//
// By hashing the *entire* userToken alongside the URL with a random seed, we ensure responses remain strictly scoped
// to the exact authentication session that originally requested them, preventing cache-poisoning via ID guessing
// while maintaining isolation between users.
//
// Seeded hashing adds further protection against precomputed key attacks, ensuring cache entries cannot be
// reverse-engineered from observed URL patterns.
func generateCacheKey(url, userToken string) string {
	combined := url + ":" + userToken

	hash := xxh3.HashStringSeed(combined, cacheSeed)

	return strconv.FormatUint(hash, 16)
}

// determineCachePolicy determines the caching policy for a given request.
//
// It returns a CachePolicy struct indicating whether to fetch from cache,
// whether to store the response in cache, and the cached response if available.
func determineCachePolicy(rawURL, userToken string, headers http.Header) CachePolicy {
	if !config.GlobalConfig.Cache.Enabled {
		return CachePolicy{}
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return CachePolicy{}
	}

	urlPath := path.Clean(parsedURL.Path)

	// Check if the path is excluded from caching
	for _, exclPath := range excludedCachePaths {
		if strings.HasPrefix(urlPath, exclPath) {
			return CachePolicy{}
		}
	}

	// Retrieve the Cache-Control header from the downstream request and check for "no-cache"
	cacheControl := headers.Get("Cache-Control")

	lowerCacheControl := strings.ToLower(cacheControl)
	if strings.Contains(lowerCacheControl, "no-cache") {
		return CachePolicy{}
	}

	cacheKey := generateCacheKey(rawURL, userToken)

	// Attempt to fetch from cache
	if cachedItem, found := cache.Get(cacheKey); found {
		item, ok := cachedItem.(CachedItem)
		if !ok {
			// The cached item is not of the expected type, remove the invalid entry
			cache.Remove(cacheKey)

			return CachePolicy{}
		}

		if time.Now().Before(item.ExpiresAt) {
			return CachePolicy{
				ShouldUseCache: true,
				CachedResponse: item.Response,
			}
		}
		// Cache expired
		cache.Remove(cacheKey)
	}

	// Determine if the API response should use the cache based on headers
	shouldUseCache := cacheControl == "" || !strings.Contains(lowerCacheControl, "no-store")

	return CachePolicy{
		ShouldUseCache: shouldUseCache,
	}
}

// manageCaching decides if a cached response is available, or if the new response should be stored and used.
func manageCaching(rawURL, userToken string, headers http.Header, freshResp *SimpleHTTPResponse) *SimpleHTTPResponse {
	policy := determineCachePolicy(rawURL, userToken, headers)

	// If there is a cached response and we're allowed to use it, return it immediately.
	if policy.ShouldUseCache && policy.CachedResponse != nil {
		return policy.CachedResponse
	}

	// Otherwise, if we should cache this fresh response, store it now.
	if policy.ShouldUseCache && freshResp != nil {
		ttl := config.GlobalConfig.Cache.TTL
		cache.Add(
			generateCacheKey(rawURL, userToken),
			CachedItem{
				Response:  freshResp,
				ExpiresAt: time.Now().Add(ttl),
				URL:       rawURL,
			},
		)
	}

	// Return the fresh response if no cached copy was used.
	return freshResp
}

// InvalidateURLs removes all cached items where the cached URL starts with any of the provided URL prefixes.
//
// Takes a slice of URL prefixes to invalidate and returns the number of cache entries removed and their full URLs.
// Safe to call even if caching is disabled. If urls slice is nil or empty, returns 0 and an empty string slice.
//
// cache.Contains isn't used as we don't actually know which cache key to look for due to how generateCacheKey() works
//
// Scoping invalidation to a specific user's context is possible (e.g. per user ID), but offers little benefit
// for the additional complexity it introduces: how many users with different auth states are realistically
// hitting similar endpoints for it to matter?
func InvalidateURLs(urlPrefixes []string) (int, []string) {
	invalidatedURLs := make([]string, 0)

	audit.GlobalAuditor.Logger.Debugf("invalidating URLs with prefixes: %v", urlPrefixes)

	if !config.GlobalConfig.Cache.Enabled || cache == nil || len(urlPrefixes) == 0 {
		audit.GlobalAuditor.Logger.Debug("skipping URL invalidation - cache disabled, nil or no prefixes provided")

		return 0, invalidatedURLs
	}

	keys := cache.Keys()
	invalidated := 0

	for _, key := range keys {
		item, ok := cache.Peek(key)
		if !ok {
			continue
		}

		cachedItem, ok := item.(CachedItem)
		if !ok {
			continue
		}

		url := cachedItem.URL

		var match bool

		for _, prefix := range urlPrefixes {
			if strings.HasPrefix(url, prefix) {
				match = true

				break
			}
		}

		if !match {
			continue
		}

		cache.Remove(key)

		invalidated++

		invalidatedURLs = append(invalidatedURLs, url)
	}

	audit.GlobalAuditor.Logger.Debugf("invalidated %d URLs: %v", invalidated, invalidatedURLs)

	return invalidated, invalidatedURLs
}
