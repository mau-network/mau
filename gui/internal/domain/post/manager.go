package post

import (
	"fmt"

	"github.com/mau-network/mau"
)

// Manager handles post operations with caching
type Manager struct {
	store Store
	cache Cache
}

// NewManager creates a post manager
func NewManager(store Store, cache Cache) *Manager {
	return &Manager{
		store: store,
		cache: cache,
	}
}

// Save saves a post to storage and cache
func (m *Manager) Save(post Post) error {
	if err := m.store.Save(post); err != nil {
		return fmt.Errorf("failed to save post: %w", err)
	}

	// Cache uses filename as key (will be set by store implementation)
	return nil
}

// Load retrieves a post, checking cache first
func (m *Manager) Load(file *mau.File) (Post, error) {
	// Try cache first
	cacheKey := file.Name()
	if cached, ok := m.cache.Get(cacheKey); ok {
		return cached, nil
	}

	// Cache miss - load from storage
	post, err := m.store.Load(file)
	if err != nil {
		return Post{}, fmt.Errorf("failed to load post: %w", err)
	}

	// Update cache
	m.cache.Set(cacheKey, post)

	return post, nil
}

// List retrieves posts for a user
func (m *Manager) List(fingerprint mau.Fingerprint, limit int) ([]*mau.File, error) {
	files, err := m.store.List(fingerprint, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list posts: %w", err)
	}

	return files, nil
}

// ClearCache clears the post cache
func (m *Manager) ClearCache() {
	m.cache.Clear()
}

// CacheStats returns cache statistics
func (m *Manager) CacheStats() (size int, capacity int) {
	return m.cache.Stats()
}
