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
	recipients, err := pm.getRecipients()
	if err != nil {
		return err
	}
	filename := pm.generateFilename()
	if err := pm.saveFile(jsonData, filename, recipients); err != nil {
		return err
	}
	pm.cache.Set(filename, post)
	return nil
}

func (pm *PostManager) getRecipients() ([]*mau.Friend, error) {
	keyring, err := pm.account.ListFriends()
	if err != nil {
		return nil, fmt.Errorf("failed to list friends: %w", err)
	}
	return keyring.FriendsSet(), nil
}

func (pm *PostManager) generateFilename() string {
	return fmt.Sprintf("posts/post-%d.json", time.Now().UnixNano())
}

func (pm *PostManager) saveFile(data []byte, filename string, recipients []*mau.Friend) error {
	reader := bytes.NewReader(data)
	_, err := pm.account.AddFile(reader, filename, recipients)
	if err != nil {
		return fmt.Errorf("failed to add file: %w", err)
	}
	return nil
}

// Load loads a post from a file
func (pm *PostManager) Load(file *mau.File) (Post, error) {
	cacheKey := file.Name()
	if cached, ok := pm.cache.Get(cacheKey); ok {
		return cached, nil
	}
	return pm.loadFromDisk(file, cacheKey)
}

func (pm *PostManager) loadFromDisk(file *mau.File, cacheKey string) (Post, error) {
	reader, err := file.Reader(pm.account)
	if err != nil {
		return Post{}, fmt.Errorf("failed to read file: %w", err)
	}
	var post Post
	if err := json.NewDecoder(reader).Decode(&post); err != nil {
		return Post{}, fmt.Errorf("failed to decode post: %w", err)
	}
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
