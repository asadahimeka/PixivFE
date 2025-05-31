// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
Implementation of a thread-safe fixed size LRU cache.

Generics aren't used as we only need string keys.
*/
package requests

import (
	"container/list"
	"errors"
	"sync"
)

var ErrInvalidSize = errors.New("must provide a positive size")

// LRUCache implements a thread-safe fixed-size LRU cache using a combination of a doubly-linked list
// (to track the usage order) and a map (for O(1) lookups of items).
type LRUCache struct {
	size      int                      // Maximum capacity of the cache
	evictList *list.List               // A doubly-linked list to manage the eviction order
	items     map[string]*list.Element // Maps string keys to their corresponding linked-list elements
	lock      sync.RWMutex             // For thread-safe operations
}

// cacheEntry holds the key/value pair stored in each linked-list element.
type cacheEntry struct {
	key   string
	value any
}

// NewLRUCache creates a new LRU cache with the specified maximum size.
//
// It returns an error if the size is not a positive integer.
func NewLRUCache(size int) (*LRUCache, error) {
	if size <= 0 {
		return nil, ErrInvalidSize
	}

	return &LRUCache{
		size:      size,
		evictList: list.New(),
		items:     make(map[string]*list.Element),
	}, nil
}

// Add adds or updates a value to the cache.
//
// If the key already exists, its value is updated and the item is moved to the front (most recently used).
// If the key is new and the cache is at capacity, eviction of the oldest item occurs.
// The boolean return indicates whether an eviction actually happened.
func (c *LRUCache) Add(key string, value any) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	// If the item already exists, move it to the front as "most recently used" and update its value.
	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)

		if cacheEnt, ok := ent.Value.(*cacheEntry); ok {
			cacheEnt.value = value
		}

		return false
	}

	// Otherwise, create a new entry and place it at the front.
	ent := &cacheEntry{key, value}
	entry := c.evictList.PushFront(ent)
	c.items[key] = entry

	// If we've exceeded our capacity, remove the oldest item from the back of the list.
	evicted := c.evictList.Len() > c.size
	if evicted {
		c.removeOldest()
	}

	return evicted
}

// Get retrieves the value for a given key if it exists, and moves that item to the front
// (as it becomes the most recently used).
func (c *LRUCache) Get(key string) (any, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)

		if cacheEnt, ok := ent.Value.(*cacheEntry); ok {
			return cacheEnt.value, true
		}
	}

	return nil, false
}

// Peek retrieves the value for a given key without modifying the LRU order
// or moving the item to the front.
func (c *LRUCache) Peek(key string) (any, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if ent, ok := c.items[key]; ok {
		if cacheEnt, ok := ent.Value.(*cacheEntry); ok {
			return cacheEnt.value, true
		}
	}

	return nil, false
}

// Remove deletes the entry associated with the given key from the cache.
//
// It returns true if the key was found and removed, or false otherwise.
func (c *LRUCache) Remove(key string) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	if ent, ok := c.items[key]; ok {
		c.removeElement(ent)

		return true
	}

	return false
}

// Keys returns a slice of all keys in the cache, from the oldest to the newest.
//
// Iterating from the back of the list to the front ensures we start with the oldest first.
func (c *LRUCache) Keys() []string {
	c.lock.RLock()
	defer c.lock.RUnlock()

	keys := make([]string, len(c.items))
	index := 0

	// The back of the list is the oldest entry.
	for ent := c.evictList.Back(); ent != nil; ent = ent.Prev() {
		if cacheEnt, ok := ent.Value.(*cacheEntry); ok {
			keys[index] = cacheEnt.key
			index++
		}
	}

	return keys
}

// Len returns the current number of items in the cache.
func (c *LRUCache) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.evictList.Len()
}

// removeOldest removes the oldest item from both the linked list and the map.
func (c *LRUCache) removeOldest() {
	ent := c.evictList.Back()
	if ent != nil {
		c.removeElement(ent)
	}
}

// removeElement is a helper function that removes a specific list element
// from the eviction list and deletes it from the map.
//
// Used to remove a given LRU entry.
func (c *LRUCache) removeElement(e *list.Element) {
	c.evictList.Remove(e)

	if kv, ok := e.Value.(*cacheEntry); ok {
		delete(c.items, kv.key)
	}
}
