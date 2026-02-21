package main

import (
	"sync"
	"time"
)

// PostCache provides an in-memory cache for posts
type PostCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	maxSize int
	ttl     time.Duration
}

// cacheEntry represents a cached post with timestamp
type cacheEntry struct {
	post      Post
	timestamp time.Time
}

// NewPostCache creates a new post cache
func NewPostCache(maxSize int, ttl time.Duration) *PostCache {
	return &PostCache{
		entries: make(map[string]*cacheEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// Get retrieves a post from cache
func (c *PostCache) Get(key string) (Post, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return Post{}, false
	}

	// Check if expired
	if time.Since(entry.timestamp) > c.ttl {
		return Post{}, false
	}

	return entry.post, true
}

// Set stores a post in cache
func (c *PostCache) Set(key string, post Post) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict old entries if at capacity
	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	c.entries[key] = &cacheEntry{
		post:      post,
		timestamp: time.Now(),
	}
}

// Invalidate removes a post from cache
func (c *PostCache) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

// Clear removes all entries from cache
func (c *PostCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*cacheEntry)
}

// evictOldest removes the oldest entry (LRU-like)
func (c *PostCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, entry := range c.entries {
		if first || entry.timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.timestamp
			first = false
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

// Stats returns cache statistics
func (c *PostCache) Stats() (size int, capacity int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries), c.maxSize
}
