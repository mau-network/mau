package main

import (
	"strings"
	"time"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// buildFilters creates the filter UI components
func (tv *TimelineView) buildFilters() {
	filterBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	tv.addAuthorFilter(filterBox)
	tv.addDateRangeFilters(filterBox)
	tv.addApplyButton(filterBox)
	tv.page.Append(filterBox)
}

func (tv *TimelineView) addAuthorFilter(box *gtk.Box) {
	authorLabel := gtk.NewLabel("Author:")
	tv.filterAuthor = gtk.NewDropDown(nil, nil)
	box.Append(authorLabel)
	box.Append(tv.filterAuthor)
}

func (tv *TimelineView) addDateRangeFilters(box *gtk.Box) {
	tv.addStartDateFilter(box)
	tv.addEndDateFilter(box)
}

func (tv *TimelineView) addStartDateFilter(box *gtk.Box) {
	dateLabel := gtk.NewLabel("From:")
	tv.filterStart = gtk.NewEntry()
	tv.filterStart.SetPlaceholderText("YYYY-MM-DD")
	tv.filterStart.SetWidthChars(12)
	box.Append(dateLabel)
	box.Append(tv.filterStart)
}

func (tv *TimelineView) addEndDateFilter(box *gtk.Box) {
	toLabel := gtk.NewLabel("To:")
	tv.filterEnd = gtk.NewEntry()
	tv.filterEnd.SetPlaceholderText("YYYY-MM-DD")
	tv.filterEnd.SetWidthChars(12)
	box.Append(toLabel)
	box.Append(tv.filterEnd)
}

func (tv *TimelineView) addApplyButton(box *gtk.Box) {
	applyBtn := gtk.NewButton()
	applyBtn.SetLabel("Apply")
	applyBtn.ConnectClicked(func() {
		tv.Refresh()
	})
	box.Append(applyBtn)
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
