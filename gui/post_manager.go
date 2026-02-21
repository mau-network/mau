package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mau-network/mau"
)

// PostManager handles post operations
type PostManager struct {
	account *mau.Account
	cache   *PostCache
}

// NewPostManager creates a post manager
func NewPostManager(account *mau.Account) *PostManager {
	return &PostManager{
		account: account,
		cache:   NewPostCache(cacheMaxSize, cacheEntryTTL*time.Minute), // Configurable cache settings
	}
}

// Save saves a post
func (pm *PostManager) Save(post Post) error {
	jsonData, err := post.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize post: %w", err)
	}

	keyring, err := pm.account.ListFriends()
	if err != nil {
		return fmt.Errorf("failed to list friends: %w", err)
	}

	recipients := keyring.FriendsSet()
	filename := fmt.Sprintf("posts/post-%d.json", time.Now().UnixNano())
	reader := bytes.NewReader(jsonData)

	file, err := pm.account.AddFile(reader, filename, recipients)
	if err != nil {
		return fmt.Errorf("failed to add file: %w", err)
	}

	// Cache the newly saved post with consistent key (just filename)
	pm.cache.Set(filename, post)

	_ = file
	return nil
}

// Load loads a post from a file
func (pm *PostManager) Load(file *mau.File) (Post, error) {
	// Try cache first - use file path as cache key
	cacheKey := file.Name()
	if cached, ok := pm.cache.Get(cacheKey); ok {
		return cached, nil
	}

	// Cache miss - load from disk
	reader, err := file.Reader(pm.account)
	if err != nil {
		return Post{}, fmt.Errorf("failed to read file: %w", err)
	}

	var post Post
	if err := json.NewDecoder(reader).Decode(&post); err != nil {
		return Post{}, fmt.Errorf("failed to decode post: %w", err)
	}

	// Store in cache
	pm.cache.Set(cacheKey, post)

	return post, nil
}

// List lists posts for a user
func (pm *PostManager) List(fingerprint mau.Fingerprint, limit int) ([]*mau.File, error) {
	files := pm.account.ListFiles(fingerprint, time.Time{}, uint(limit))

	var postFiles []*mau.File
	for _, f := range files {
		// Files are stored as posts/post-*.json.pgp (AddFile adds .pgp extension)
		if strings.HasPrefix(f.Name(), "posts/") && strings.Contains(f.Name(), ".json") {
			postFiles = append(postFiles, f)
		}
	}

	return postFiles, nil
}

// ClearCache clears the post cache
func (pm *PostManager) ClearCache() {
	pm.cache.Clear()
}

// CacheStats returns cache statistics
func (pm *PostManager) CacheStats() (size int, capacity int) {
	return pm.cache.Stats()
}
