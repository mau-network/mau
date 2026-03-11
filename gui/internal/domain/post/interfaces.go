package post

import "github.com/mau-network/mau"

// Store defines the interface for post persistence.
// Implementations handle the actual storage mechanism (files, database, etc.)
type Store interface {
	// Save persists a post
	Save(post Post) error

	// Load retrieves a post from a file reference
	Load(file *mau.File) (Post, error)

	// List retrieves posts for a given user, up to limit
	List(fingerprint mau.Fingerprint, limit int) ([]*mau.File, error)
}

// Cache defines the interface for post caching
type Cache interface {
	// Get retrieves a cached post by key
	Get(key string) (Post, bool)

	// Set stores a post in cache
	Set(key string, post Post)

	// Clear removes all cached posts
	Clear()

	// Stats returns cache statistics (size, capacity)
	Stats() (size int, capacity int)
}
