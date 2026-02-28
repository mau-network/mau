// Package storage implements domain interfaces using Mau's file-based storage.
package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mau-network/mau"
	"github.com/mau-network/mau-gui-poc/internal/domain/post"
)

// PostStore implements post.Store using Mau's encrypted file storage
type PostStore struct {
	account *mau.Account
}

// NewPostStore creates a new post store
func NewPostStore(account *mau.Account) *PostStore {
	return &PostStore{account: account}
}

// Save persists a post as an encrypted file
func (s *PostStore) Save(p post.Post) error {
	jsonData, err := p.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize post: %w", err)
	}

	keyring, err := s.account.ListFriends()
	if err != nil {
		return fmt.Errorf("failed to list friends: %w", err)
	}

	recipients := keyring.FriendsSet()
	filename := fmt.Sprintf("posts/post-%d.json", time.Now().UnixNano())
	reader := bytes.NewReader(jsonData)

	_, err = s.account.AddFile(reader, filename, recipients)
	if err != nil {
		return fmt.Errorf("failed to add file: %w", err)
	}

	return nil
}

// Load retrieves a post from an encrypted file
func (s *PostStore) Load(file *mau.File) (post.Post, error) {
	reader, err := file.Reader(s.account)
	if err != nil {
		return post.Post{}, fmt.Errorf("failed to read file: %w", err)
	}

	var p post.Post
	if err := json.NewDecoder(reader).Decode(&p); err != nil {
		return post.Post{}, fmt.Errorf("failed to decode post: %w", err)
	}

	return p, nil
}

// List retrieves post files for a user
func (s *PostStore) List(fingerprint mau.Fingerprint, limit int) ([]*mau.File, error) {
	files := s.account.ListFiles(fingerprint, time.Time{}, uint(limit))

	var postFiles []*mau.File
	for _, f := range files {
		// Files are stored as posts/post-*.json.pgp (AddFile adds .pgp extension)
		if strings.HasPrefix(f.Name(), "posts/") && strings.Contains(f.Name(), ".json") {
			postFiles = append(postFiles, f)
		}
	}

	return postFiles, nil
}
