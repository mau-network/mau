package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/mau-network/mau"
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

// FromJSON deserializes post from JSON
func PostFromJSON(data []byte) (Post, error) {
	var post Post
	err := json.Unmarshal(data, &post)
	return post, err
}

// PostManager handles post operations
type PostManager struct {
	account *mau.Account
}

// NewPostManager creates a post manager
func NewPostManager(account *mau.Account) *PostManager {
	return &PostManager{account: account}
}

// Save saves a post
func (pm *PostManager) Save(post Post) error {
	jsonData, err := post.ToJSON()
	if err != nil {
		return err
	}

	keyring, err := pm.account.ListFriends()
	if err != nil {
		return err
	}

	recipients := keyring.FriendsSet()
	filename := fmt.Sprintf("posts/post-%d.json", time.Now().UnixNano())
	reader := bytes.NewReader(jsonData)

	_, err = pm.account.AddFile(reader, filename, recipients)
	return err
}

// Load loads a post from a file
func (pm *PostManager) Load(file *mau.File) (Post, error) {
	reader, err := file.Reader(pm.account)
	if err != nil {
		return Post{}, err
	}

	var post Post
	if err := json.NewDecoder(reader).Decode(&post); err != nil {
		return Post{}, err
	}

	return post, nil
}

// List lists posts for a user
func (pm *PostManager) List(fingerprint mau.Fingerprint, limit int) ([]*mau.File, error) {
	files := pm.account.ListFiles(fingerprint, time.Time{}, uint(limit))

	var postFiles []*mau.File
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "posts/") && strings.HasSuffix(f.Name(), ".json") {
			postFiles = append(postFiles, f)
		}
	}

	return postFiles, nil
}

// MarkdownRenderer handles markdown conversion
type MarkdownRenderer struct{}

// NewMarkdownRenderer creates a markdown renderer
func NewMarkdownRenderer() *MarkdownRenderer {
	return &MarkdownRenderer{}
}

// ToHTML converts markdown to HTML
func (mr *MarkdownRenderer) ToHTML(markdownText string) string {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse([]byte(markdownText))

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)
	htmlBytes := markdown.Render(doc, renderer)

	return string(htmlBytes)
}

// ToPango converts markdown to Pango markup
func (mr *MarkdownRenderer) ToPango(markdownText string) string {
	htmlStr := mr.ToHTML(markdownText)

	// Simplified HTML to Pango conversion
	pango := htmlStr
	pango = strings.ReplaceAll(pango, "<p>", "")
	pango = strings.ReplaceAll(pango, "</p>", "\n")
	pango = strings.ReplaceAll(pango, "<strong>", "<b>")
	pango = strings.ReplaceAll(pango, "</strong>", "</b>")
	pango = strings.ReplaceAll(pango, "<em>", "<i>")
	pango = strings.ReplaceAll(pango, "</em>", "</i>")
	pango = strings.ReplaceAll(pango, "<code>", "<tt>")
	pango = strings.ReplaceAll(pango, "</code>", "</tt>")

	return pango
}

// ParseTags parses comma-separated tags
func ParseTags(tagText string) []string {
	if tagText == "" {
		return nil
	}

	var tags []string
	for _, tag := range strings.Split(tagText, ",") {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}

// FormatTags formats tags as comma-separated string
func FormatTags(tags []string) string {
	return strings.Join(tags, ", ")
}

// Truncate truncates a string to maxLen
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
