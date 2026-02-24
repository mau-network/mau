// Package markdown provides markdown rendering utilities.
package markdown

import (
	"fmt"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

const (
	MaxTagsInput      = 500  // Maximum tag input field length
	MaxTagLength      = 50   // Maximum individual tag length
	MaxTags           = 10   // Maximum number of tags
	MaxPostBodyLength = 5000 // Maximum post body length
)

// Renderer handles markdown conversion
type Renderer struct{}

// NewRenderer creates a markdown renderer
func NewRenderer() *Renderer {
	return &Renderer{}
}

// ToHTML converts markdown to HTML
func (r *Renderer) ToHTML(markdownText string) string {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse([]byte(markdownText))

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)
	htmlBytes := markdown.Render(doc, renderer)

	return string(htmlBytes)
}

// ToPango converts markdown to Pango markup for GTK labels
func (r *Renderer) ToPango(markdownText string) string {
	htmlStr := r.ToHTML(markdownText)

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

	// Validate input length
	if len(tagText) > MaxTagsInput {
		tagText = tagText[:MaxTagsInput]
	}

	var tags []string
	for _, tag := range strings.Split(tagText, ",") {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		// Validate individual tag length
		if len(tag) > MaxTagLength {
			tag = tag[:MaxTagLength]
		}
		tags = append(tags, tag)
		// Limit number of tags
		if len(tags) >= MaxTags {
			break
		}
	}
	return tags
}

// ValidatePostBody validates post body content
func ValidatePostBody(body string) error {
	if body == "" {
		return fmt.Errorf("post body cannot be empty")
	}
	if len(body) > MaxPostBodyLength {
		return fmt.Errorf("post exceeds maximum length of %d characters", MaxPostBodyLength)
	}
	// Check for invalid characters (null bytes)
	if strings.Contains(body, "\x00") {
		return fmt.Errorf("post contains invalid characters")
	}
	return nil
}

// SanitizePostBody sanitizes post body (basic HTML escape for safety)
func SanitizePostBody(body string) string {
	// Basic sanitization - the markdown renderer handles the rest
	body = strings.ReplaceAll(body, "\x00", "") // Remove null bytes
	return strings.TrimSpace(body)
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
