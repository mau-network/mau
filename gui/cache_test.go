package main

import (
	"testing"
	"time"
)

func TestPostCache_GetSet(t *testing.T) {
	cache := NewPostCache(10, 1*time.Minute)

	post := Post{
		Body:      "Test post",
		Published: time.Now(),
	}

	// Cache miss
	if _, ok := cache.Get("key1"); ok {
		t.Error("Expected cache miss, got hit")
	}

	// Set and get
	cache.Set("key1", post)
	retrieved, ok := cache.Get("key1")
	if !ok {
		t.Fatal("Expected cache hit, got miss")
	}

	if retrieved.Body != post.Body {
		t.Errorf("Expected body %q, got %q", post.Body, retrieved.Body)
	}
}

func TestPostCache_Expiration(t *testing.T) {
	cache := NewPostCache(10, 50*time.Millisecond)

	post := Post{Body: "Test"}
	cache.Set("key1", post)

	// Should be cached
	if _, ok := cache.Get("key1"); !ok {
		t.Error("Expected cache hit before expiration")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	if _, ok := cache.Get("key1"); ok {
		t.Error("Expected cache miss after expiration")
	}
}

func TestPostCache_Eviction(t *testing.T) {
	cache := NewPostCache(3, 1*time.Minute)

	// Fill cache
	for i := 0; i < 3; i++ {
		key := string(rune('a' + i))
		cache.Set(key, Post{Body: key})
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Verify all cached
	for i := 0; i < 3; i++ {
		key := string(rune('a' + i))
		if _, ok := cache.Get(key); !ok {
			t.Errorf("Expected %q to be cached", key)
		}
	}

	// Add one more (should evict oldest)
	cache.Set("d", Post{Body: "d"})

	// First entry should be evicted
	if _, ok := cache.Get("a"); ok {
		t.Error("Expected oldest entry 'a' to be evicted")
	}

	// Others should still be cached
	for _, key := range []string{"b", "c", "d"} {
		if _, ok := cache.Get(key); !ok {
			t.Errorf("Expected %q to be cached", key)
		}
	}
}

func TestPostCache_Invalidate(t *testing.T) {
	cache := NewPostCache(10, 1*time.Minute)

	post := Post{Body: "Test"}
	cache.Set("key1", post)

	// Verify cached
	if _, ok := cache.Get("key1"); !ok {
		t.Error("Expected cache hit")
	}

	// Invalidate
	cache.Invalidate("key1")

	// Should be gone
	if _, ok := cache.Get("key1"); ok {
		t.Error("Expected cache miss after invalidation")
	}
}

func TestPostCache_Clear(t *testing.T) {
	cache := NewPostCache(10, 1*time.Minute)

	// Add multiple entries
	for i := 0; i < 5; i++ {
		key := string(rune('a' + i))
		cache.Set(key, Post{Body: key})
	}

	size, _ := cache.Stats()
	if size != 5 {
		t.Errorf("Expected cache size 5, got %d", size)
	}

	// Clear
	cache.Clear()

	size, _ = cache.Stats()
	if size != 0 {
		t.Errorf("Expected cache size 0 after clear, got %d", size)
	}

	// All entries should be gone
	for i := 0; i < 5; i++ {
		key := string(rune('a' + i))
		if _, ok := cache.Get(key); ok {
			t.Errorf("Expected %q to be cleared", key)
		}
	}
}

func TestPostCache_Stats(t *testing.T) {
	cache := NewPostCache(10, 1*time.Minute)

	size, capacity := cache.Stats()
	if size != 0 {
		t.Errorf("Expected initial size 0, got %d", size)
	}
	if capacity != 10 {
		t.Errorf("Expected capacity 10, got %d", capacity)
	}

	// Add entries
	for i := 0; i < 5; i++ {
		cache.Set(string(rune('a'+i)), Post{})
	}

	size, capacity = cache.Stats()
	if size != 5 {
		t.Errorf("Expected size 5, got %d", size)
	}
	if capacity != 10 {
		t.Errorf("Expected capacity 10, got %d", capacity)
	}
}
