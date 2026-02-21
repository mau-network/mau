package main

import (
	"strings"
	"time"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// buildFilters creates the filter UI components
func (tv *TimelineView) buildFilters() {
	filterBox := gtk.NewBox(gtk.OrientationHorizontal, 6)

	// Author filter
	authorLabel := gtk.NewLabel("Author:")
	tv.filterAuthor = gtk.NewDropDown(nil, nil)
	filterBox.Append(authorLabel)
	filterBox.Append(tv.filterAuthor)

	// Date range
	dateLabel := gtk.NewLabel("From:")
	tv.filterStart = gtk.NewEntry()
	tv.filterStart.SetPlaceholderText("YYYY-MM-DD")
	tv.filterStart.SetWidthChars(12)
	filterBox.Append(dateLabel)
	filterBox.Append(tv.filterStart)

	toLabel := gtk.NewLabel("To:")
	tv.filterEnd = gtk.NewEntry()
	tv.filterEnd.SetPlaceholderText("YYYY-MM-DD")
	tv.filterEnd.SetWidthChars(12)
	filterBox.Append(toLabel)
	filterBox.Append(tv.filterEnd)

	// Apply button
	applyBtn := gtk.NewButton()
	applyBtn.SetLabel("Apply")
	applyBtn.ConnectClicked(func() {
		tv.Refresh()
	})
	filterBox.Append(applyBtn)

	tv.page.Append(filterBox)
}

// matchesFilters checks if a post matches the current filter criteria
func (tv *TimelineView) matchesFilters(post Post, authorName, filterAuthor string, startDate, endDate time.Time) bool {
	// Author filter (case-insensitive substring match)
	if filterAuthor != "" {
		if !strings.Contains(strings.ToLower(authorName), strings.ToLower(filterAuthor)) {
			return false
		}
	}

	// Date range filters
	if !startDate.IsZero() && post.Published.Before(startDate) {
		return false
	}

	if !endDate.IsZero() && post.Published.After(endDate.Add(24*time.Hour-time.Second)) {
		return false
	}

	return true
}

// parseDate parses a date string in YYYY-MM-DD format
func (tv *TimelineView) parseDate(dateStr string) time.Time {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return time.Time{}
	}

	// Try YYYY-MM-DD format
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}
	}

	return t
}
