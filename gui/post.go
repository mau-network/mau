package main

import (
	"encoding/json"
	"time"
)

// Post represents a social media post
type Post struct {
	Context     string       `json:"@context"`
	Type        string       `json:"@type"`
	Headline    string       `json:"headline"`
	Body        string       `json:"articleBody"`
	Published   time.Time    `json:"datePublished"`
	Author      Author       `json:"author"`
	Tags        []string     `json:"keywords,omitempty"`
	Attachments []Attachment `json:"attachment,omitempty"`
}

// Author represents post author
type Author struct {
	Type        string `json:"@type"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

// Attachment represents a file attachment
type Attachment struct {
	Type        string `json:"@type"`
	Name        string `json:"name"`
	ContentType string `json:"encodingFormat,omitempty"`
	Data        string `json:"contentUrl"`
}

// NewPost creates a new post
func NewPost(body string, author Author, tags []string) Post {
	return Post{
		Context:   "https://schema.org",
		Type:      "SocialMediaPosting",
		Headline:  "New Post",
		Body:      body,
		Published: time.Now(),
		Author:    author,
		Tags:      tags,
	}
}

// ToJSON serializes post to JSON
func (p Post) ToJSON() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

// PostFromJSON deserializes post from JSON
func PostFromJSON(data []byte) (Post, error) {
	var post Post
	err := json.Unmarshal(data, &post)
	return post, err
}
