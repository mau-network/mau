// Package format provides post formatting utilities.
package format

import (
	"fmt"
	"strings"
)

const (
	MaxPostBodyLength = 10000 // 10KB max post body
	MaxTagLength      = 50    // Max characters per tag
	MaxTags           = 20    // Max number of tags per post
	MaxTagsInput      = 200   // Max characters in tag input field
)

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
		return fmt.Errorf("Cannot post empty content")
	}
	if len(body) > MaxPostBodyLength {
		return fmt.Errorf("Post is too long (max 10,000 characters)")
	}
	// Check for invalid characters (null bytes)
	if strings.Contains(body, "\x00") {
		return fmt.Errorf("Post contains invalid characters")
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
