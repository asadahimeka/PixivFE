// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package requests_test

import (
	"testing"

	. "codeberg.org/pixivfe/pixivfe/core/requests" //nolint:revive
)

// TestNewLRUCache checks the creation of a new LRUCache with both valid and invalid sizes.
func TestNewLRUCache(t *testing.T) {
	t.Parallel()

	t.Run("ValidSize", func(t *testing.T) {
		t.Parallel()

		// Create an LRU cache of size 3, which is valid.
		cache, err := NewLRUCache(3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cache == nil {
			t.Fatal("expected cache to be initialized")
		}

		// Immediately after creation, the cache should be empty.
		if cache.Len() != 0 {
			t.Errorf("expected cache length to be 0, got %d", cache.Len())
		}
	})

	t.Run("InvalidSize", func(t *testing.T) {
		t.Parallel()

		// Create an LRU cache of size 0, which should fail.
		cache, err := NewLRUCache(0)
		if err == nil {
			t.Fatal("expected error when creating cache of size 0, got nil")
		}

		if cache != nil {
			t.Error("expected no cache to be returned on error")
		}
	})
}

// TestLRUCache_AddAndGet verifies that adding a key to the cache and retrieving it works correctly,
// and that eviction occurs once the capacity is reached.
func TestLRUCache_AddAndGet(t *testing.T) {
	t.Parallel()

	// Create a cache with capacity 2.
	cache, err := NewLRUCache(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Add first key; eviction should not occur yet.
	evicted := cache.Add("foo", "bar")
	if evicted {
		t.Error("eviction should not occur when the cache is not full")
	}

	// Retrieve the newly added key.
	value, ok := cache.Get("foo")
	if !ok {
		t.Error("expected to retrieve value for key 'foo'")
	}

	if value != "bar" {
		t.Errorf("expected 'bar', got %v", value)
	}

	// Add second key.
	cache.Add("hello", "world")

	// Ensure the cache length is now 2.
	if cache.Len() != 2 {
		t.Errorf("expected cache length 2, got %d", cache.Len())
	}

	// Adding a third key should cause eviction of the least recently used item.
	evicted = cache.Add("key3", "value3")
	if !evicted {
		t.Error("expected eviction when adding third key to size 2 cache")
	}

	// "foo" should be evicted because it was the oldest after the second key was used.
	_, ok = cache.Get("foo")
	if ok {
		t.Error("expected 'foo' to be evicted, but it still exists")
	}
}

// TestLRUCache_AddExistingKey ensures that adding a key that already exists
// updates the value and does not evict any item.
func TestLRUCache_AddExistingKey(t *testing.T) {
	t.Parallel()

	cache, _ := NewLRUCache(2)

	cache.Add("k1", "v1")
	cache.Add("k2", "v2")

	// Re-add k1 with new value; expect no eviction since the key already exists.
	evicted := cache.Add("k1", "v1-updated")
	if evicted {
		t.Error("re-adding an existing key should not evict anything")
	}

	// Verify the value was updated in the cache.
	val, ok := cache.Get("k1")
	if !ok {
		t.Error("expected to find updated key 'k1'")
	}

	if val != "v1-updated" {
		t.Errorf("expected 'v1-updated', got %v", val)
	}

	// Cache size should remain at 2 (no evictions).
	if cache.Len() != 2 {
		t.Errorf("expected cache length 2, got %d", cache.Len())
	}
}

// TestLRUCache_Peek checks that Peek returns the value without updating the item’s priority.
func TestLRUCache_Peek(t *testing.T) {
	t.Parallel()

	cache, _ := NewLRUCache(2)

	cache.Add("foo", "bar")
	cache.Add("baz", "qux")

	// Peek at "foo"; this should not move it to the front of the usage list.
	val, ok := cache.Peek("foo")
	if !ok {
		t.Error("expected to peek value for 'foo'")
	}

	if val != "bar" {
		t.Errorf("expected 'bar', got %v", val)
	}

	// Now add a third key to force eviction of the least recently used item.
	cache.Add("third", "value3")

	// If the peek didn't promote "foo", it remains the oldest and should be evicted.
	_, ok = cache.Get("foo")
	if ok {
		t.Error("expected 'foo' to be evicted after adding 'third'")
	}

	_, ok = cache.Get("baz")
	if !ok {
		t.Error("expected 'baz' to remain in the cache")
	}
}

// TestLRUCache_Remove confirms that removing a key explicitly works.
func TestLRUCache_Remove(t *testing.T) {
	t.Parallel()

	cache, _ := NewLRUCache(2)

	cache.Add("foo", "bar")
	cache.Add("key", "value")

	// Remove the key "foo" and verify it no longer exists.
	removed := cache.Remove("foo")
	if !removed {
		t.Error("expected to remove existing key 'foo'")
	}

	val, ok := cache.Get("foo")
	if ok || val != nil {
		t.Error("expected 'foo' to be removed from cache")
	}

	// Attempt to remove a key that doesn't exist.
	removed = cache.Remove("not-present")
	if removed {
		t.Error("expected false when removing a non-existent key, but got true")
	}
}

// TestLRUCache_Keys checks that Keys returns the slice of keys from oldest to newest.
func TestLRUCache_Keys(t *testing.T) {
	t.Parallel()

	cache, _ := NewLRUCache(3)
	cache.Add("first", 1)
	cache.Add("second", 2)
	cache.Add("third", 3)

	// Keys() returns oldest-to-newest, i.e., from the back of the list to the front.
	keys := cache.Keys()
	expected := []any{"first", "second", "third"}

	if len(keys) != len(expected) {
		t.Fatalf("expected %d keys, got %d", len(expected), len(keys))
	}

	for i, k := range keys {
		if k != expected[i] {
			t.Errorf("Keys() mismatch: expected %v at idx %d, got %v", expected[i], i, k)
		}
	}

	// Access "first", which should move it to the newest position.
	cache.Get("first")
	keys = cache.Keys()
	// Now the oldest is "second", next is "third", and the newest is "first".
	expected = []any{"second", "third", "first"}

	for i, k := range keys {
		if k != expected[i] {
			t.Errorf("After usage, Keys mismatch: expected %v at idx %d, got %v", expected[i], i, k)
		}
	}
}

// TestLRUCache_Len verifies the length of the cache under various operations.
func TestLRUCache_Len(t *testing.T) {
	t.Parallel()

	cache, _ := NewLRUCache(2)

	// Initially, the cache should be empty.
	if cache.Len() != 0 {
		t.Errorf("expected newly created cache size to be 0, got %d", cache.Len())
	}

	cache.Add("a", "b")

	// Now the cache should have exactly 1 item.
	if cache.Len() != 1 {
		t.Errorf("expected cache length to be 1, got %d", cache.Len())
	}

	cache.Add("c", "d")

	// Cache is now full (size 2).
	if cache.Len() != 2 {
		t.Errorf("expected cache length to be 2, got %d", cache.Len())
	}

	cache.Add("e", "f") // This will evict the oldest item.

	// The length should remain at 2 after the eviction.
	if cache.Len() != 2 {
		t.Errorf("expected cache length to remain 2 after eviction, got %d", cache.Len())
	}
}
