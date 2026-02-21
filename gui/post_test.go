package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewPost(t *testing.T) {
	author := Author{
		Type:        "Person",
		Name:        "Alice",
		Email:       "alice@example.com",
		Fingerprint: "FPR123",
	}

	tags := []string{"golang", "testing"}
	body := "Hello, world!"

	post := NewPost(body, author, tags)

	if post.Body != body {
		t.Errorf("Expected body=%s, got %s", body, post.Body)
	}
	if post.Author.Name != "Alice" {
		t.Errorf("Expected author=Alice, got %s", post.Author.Name)
	}
	if len(post.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(post.Tags))
	}
	if post.Context != "https://schema.org" {
		t.Error("Context not set correctly")
	}
	if post.Type != "SocialMediaPosting" {
		t.Error("Type not set correctly")
	}
}

func TestPost_ToJSON(t *testing.T) {
	author := Author{
		Type:  "Person",
		Name:  "Bob",
		Email: "bob@example.com",
	}

	post := NewPost("Test post", author, nil)
	data, err := post.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Verify it's valid JSON
	var decoded Post
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Generated invalid JSON: %v", err)
	}

	if decoded.Body != "Test post" {
		t.Errorf("Expected body='Test post', got %s", decoded.Body)
	}
	if decoded.Author.Name != "Bob" {
		t.Errorf("Expected author=Bob, got %s", decoded.Author.Name)
	}
}

func TestPostFromJSON(t *testing.T) {
	jsonData := []byte(`{
		"@context": "https://schema.org",
		"@type": "SocialMediaPosting",
		"headline": "Test",
		"articleBody": "Hello",
		"datePublished": "2026-02-21T12:00:00Z",
		"author": {
			"@type": "Person",
			"name": "Charlie",
			"email": "charlie@example.com"
		},
		"keywords": ["test", "demo"]
	}`)

	post, err := PostFromJSON(jsonData)
	if err != nil {
		t.Fatalf("PostFromJSON failed: %v", err)
	}

	if post.Body != "Hello" {
		t.Errorf("Expected body='Hello', got %s", post.Body)
	}
	if post.Author.Name != "Charlie" {
		t.Errorf("Expected author=Charlie, got %s", post.Author.Name)
	}
	if len(post.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(post.Tags))
	}
}

func TestPostFromJSON_InvalidJSON(t *testing.T) {
	_, err := PostFromJSON([]byte("{invalid"))
	if err == nil {
		t.Error("Expected error on invalid JSON, got nil")
	}
}

func TestPost_RoundTrip(t *testing.T) {
	author := Author{
		Type:        "Person",
		Name:        "Dave",
		Email:       "dave@example.com",
		Fingerprint: "FPR456",
	}

	original := NewPost("Round trip test", author, []string{"test"})

	// Serialize
	data, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Deserialize
	decoded, err := PostFromJSON(data)
	if err != nil {
		t.Fatalf("PostFromJSON failed: %v", err)
	}

	// Compare
	if decoded.Body != original.Body {
		t.Errorf("Body mismatch: expected %s, got %s", original.Body, decoded.Body)
	}
	if decoded.Author.Name != original.Author.Name {
		t.Errorf("Author mismatch: expected %s, got %s", original.Author.Name, decoded.Author.Name)
	}
	if len(decoded.Tags) != len(original.Tags) {
		t.Errorf("Tags length mismatch: expected %d, got %d", len(original.Tags), len(decoded.Tags))
	}
}

func TestMarkdownRenderer_ToHTML(t *testing.T) {
	mr := NewMarkdownRenderer()

	tests := []struct {
		name     string
		markdown string
		contains string
	}{
		{"Bold", "**bold**", "<strong>bold</strong>"},
		{"Italic", "*italic*", "<em>italic</em>"},
		{"Header", "# Header", "<h1"},
		{"Link", "[link](http://example.com)", "<a href"},
		{"Code", "`code`", "<code>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := mr.ToHTML(tt.markdown)
			if !strings.Contains(html, tt.contains) {
				t.Errorf("Expected HTML to contain %s, got %s", tt.contains, html)
			}
		})
	}
}

func TestMarkdownRenderer_ToPango(t *testing.T) {
	mr := NewMarkdownRenderer()

	tests := []struct {
		name     string
		markdown string
		contains string
	}{
		{"Bold", "**bold**", "<b>bold</b>"},
		{"Italic", "*italic*", "<i>italic</i>"},
		{"Code", "`code`", "<tt>code</tt>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pango := mr.ToPango(tt.markdown)
			if !strings.Contains(pango, tt.contains) {
				t.Errorf("Expected Pango to contain %s, got %s", tt.contains, pango)
			}
		})
	}
}

func TestParseTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"Empty", "", nil},
		{"Single", "test", []string{"test"}},
		{"Multiple", "tag1, tag2, tag3", []string{"tag1", "tag2", "tag3"}},
		{"Spaces", "  tag1  ,  tag2  ", []string{"tag1", "tag2"}},
		{"Mixed", "golang,testing, demo", []string{"golang", "testing", "demo"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseTags(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d tags, got %d", len(tt.expected), len(result))
				return
			}
			for i, tag := range result {
				if tag != tt.expected[i] {
					t.Errorf("Tag %d: expected %s, got %s", i, tt.expected[i], tag)
				}
			}
		})
	}
}

func TestFormatTags(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{"Empty", []string{}, ""},
		{"Single", []string{"test"}, "test"},
		{"Multiple", []string{"tag1", "tag2", "tag3"}, "tag1, tag2, tag3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTags(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"Short", "hello", 10, "hello"},
		{"Exact", "hello", 5, "hello"},
		{"Long", "hello world", 8, "hello..."},
		{"VeryLong", "this is a very long string", 10, "this is..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
			if len(result) > tt.maxLen {
				t.Errorf("Result length %d exceeds max %d", len(result), tt.maxLen)
			}
		})
	}
}

func TestPost_WithAttachments(t *testing.T) {
	author := Author{Type: "Person", Name: "Eve", Email: "eve@example.com"}
	post := NewPost("Test", author, nil)

	post.Attachments = []Attachment{
		{
			Type:        "ImageObject",
			Name:        "test.png",
			ContentType: "image/png",
			Data:        "base64data",
		},
	}

	data, err := post.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	decoded, err := PostFromJSON(data)
	if err != nil {
		t.Fatalf("PostFromJSON failed: %v", err)
	}

	if len(decoded.Attachments) != 1 {
		t.Fatalf("Expected 1 attachment, got %d", len(decoded.Attachments))
	}

	att := decoded.Attachments[0]
	if att.Name != "test.png" {
		t.Errorf("Expected name='test.png', got %s", att.Name)
	}
	if att.ContentType != "image/png" {
		t.Errorf("Expected type='image/png', got %s", att.ContentType)
	}
}

func TestPost_TimeFormat(t *testing.T) {
	author := Author{Type: "Person", Name: "Test", Email: "test@test"}
	post := NewPost("Test", author, nil)

	// Set specific time
	testTime := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	post.Published = testTime

	data, err := post.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	decoded, err := PostFromJSON(data)
	if err != nil {
		t.Fatalf("PostFromJSON failed: %v", err)
	}

	// Times should match
	if !decoded.Published.Equal(testTime) {
		t.Errorf("Time mismatch: expected %v, got %v", testTime, decoded.Published)
	}
}
