package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// timelinePost represents a post with its author info
type timelinePost struct {
	post       Post
	friendName string
}

// TimelineView handles the timeline view
type TimelineView struct {
	app             *MauApp
	page            *gtk.Box
	timelineList    *gtk.ListBox
	filterAuthor    *gtk.DropDown
	filterStart     *gtk.Entry
	filterEnd       *gtk.Entry
	loadMoreBtn     *gtk.Button
	currentPage     int
	pageSize        int
	allPosts        []timelinePost
	hasMore         bool
}

// NewTimelineView creates a new timeline view
func NewTimelineView(app *MauApp) *TimelineView {
	return &TimelineView{
		app:         app,
		currentPage: 0,
		pageSize:    timelinePageSize, // Configurable page size
		allPosts:    []timelinePost{},
		hasMore:     false,
	}
}

// Build creates and returns the view widget
func (tv *TimelineView) Build() *gtk.Box {
	tv.page = gtk.NewBox(gtk.OrientationVertical, 12)
	tv.page.SetMarginTop(12)
	tv.page.SetMarginBottom(12)
	tv.page.SetMarginStart(12)
	tv.page.SetMarginEnd(12)

	// Header with refresh
	header := gtk.NewBox(gtk.OrientationHorizontal, 12)
	headerLabel := gtk.NewLabel("Network Timeline")
	headerLabel.AddCSSClass("title-1")
	header.Append(headerLabel)

	refreshBtn := gtk.NewButton()
	refreshBtn.SetIconName("view-refresh-symbolic")
	refreshBtn.SetTooltipText("Sync with friends (F5)")
	refreshBtn.ConnectClicked(func() {
		tv.app.syncFriends()
	})
	header.Append(refreshBtn)

	tv.page.Append(header)

	// Filters
	tv.buildFilters()

	// Timeline list
	timelineGroup := adw.NewPreferencesGroup()
	timelineGroup.SetTitle("Posts from Friends")

	tv.timelineList = NewBoxedListBox()

	timelineScrolled := gtk.NewScrolledWindow()
	timelineScrolled.SetVExpand(true)
	timelineScrolled.SetChild(tv.timelineList)

	tv.page.Append(timelineGroup)
	tv.page.Append(timelineScrolled)

	// Load More button
	tv.loadMoreBtn = gtk.NewButton()
	tv.loadMoreBtn.SetLabel("Load More Posts")
	tv.loadMoreBtn.AddCSSClass("suggested-action")
	tv.loadMoreBtn.SetVisible(false)
	tv.loadMoreBtn.ConnectClicked(func() {
		tv.loadMore()
	})

	loadMoreBox := gtk.NewBox(gtk.OrientationHorizontal, 0)
	loadMoreBox.SetHAlign(gtk.AlignCenter)
	loadMoreBox.Append(tv.loadMoreBtn)
	tv.page.Append(loadMoreBox)

	tv.Refresh()

	return tv.page
}

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

// Refresh reloads the timeline with filters applied
func (tv *TimelineView) Refresh() {
	// Reset pagination
	tv.currentPage = 0
	tv.timelineList.RemoveAll()

	keyring, err := tv.app.accountMgr.Account().ListFriends()
	if err != nil {
		row := adw.NewActionRow()
		row.SetTitle("Error loading friends")
		row.SetSubtitle(err.Error())
		tv.timelineList.Append(row)
		tv.loadMoreBtn.SetVisible(false)
		return
	}

	friends := keyring.FriendsSet()
	if len(friends) == 0 {
		row := adw.NewActionRow()
		row.SetTitle("No friends yet")
		row.SetSubtitle("Add friends to see their posts")
		tv.timelineList.Append(row)
		tv.loadMoreBtn.SetVisible(false)
		return
	}

	// Get filter values
	filterAuthorText := strings.TrimSpace(tv.filterStart.Text()) // Reusing start field for author
	filterStartDate := tv.parseDate(tv.filterStart.Text())
	filterEndDate := tv.parseDate(tv.filterEnd.Text())

	tv.allPosts = []timelinePost{}

	for _, friend := range friends {
		fpr := friend.Fingerprint()
		files, _ := tv.app.postMgr.List(fpr, friendPostLimit)

		for _, file := range files {
			post, err := tv.app.postMgr.Load(file)
			if err != nil {
				continue
			}

			// Apply filters
			if !tv.matchesFilters(post, friend.Name(), filterAuthorText, filterStartDate, filterEndDate) {
				continue
			}

			tv.allPosts = append(tv.allPosts, timelinePost{
				post:       post,
				friendName: friend.Name(),
			})
		}
	}

	if len(tv.allPosts) == 0 {
		row := adw.NewActionRow()
		if filterAuthorText != "" || !filterStartDate.IsZero() || !filterEndDate.IsZero() {
			row.SetTitle("No posts match filters")
			row.SetSubtitle("Try adjusting your filter criteria")
		} else {
			row.SetTitle("No posts from friends yet")
		}
		tv.timelineList.Append(row)
		tv.loadMoreBtn.SetVisible(false)
		return
	}

	// Sort by newest first
	sort.Slice(tv.allPosts, func(i, j int) bool {
		return tv.allPosts[i].post.Published.After(tv.allPosts[j].post.Published)
	})

	// Display first page
	tv.displayPage()
}

func (tv *TimelineView) loadMore() {
	tv.currentPage++
	tv.displayPage()
}

func (tv *TimelineView) displayPage() {
	start := tv.currentPage * tv.pageSize
	end := start + tv.pageSize

	if start >= len(tv.allPosts) {
		// No more posts
		tv.hasMore = false
		tv.loadMoreBtn.SetVisible(false)
		return
	}

	if end > len(tv.allPosts) {
		end = len(tv.allPosts)
	}

	// Display posts for this page
	for i := start; i < end; i++ {
		tp := tv.allPosts[i]
		row := adw.NewActionRow()
		row.SetTitle(Truncate(tp.post.Body, 80))

		subtitle := fmt.Sprintf("%s • %s", tp.friendName, tp.post.Published.Format("2006-01-02 15:04"))
		if len(tp.post.Tags) > 0 {
			subtitle += " • " + FormatTags(tp.post.Tags)
		}
		row.SetSubtitle(subtitle)

		icon := gtk.NewImageFromIconName("avatar-default-symbolic")
		row.AddPrefix(icon)

		verifiedIcon := gtk.NewImageFromIconName("emblem-ok-symbolic")
		row.AddSuffix(verifiedIcon)

		tv.timelineList.Append(row)
	}

	// Update Load More button visibility
	tv.hasMore = end < len(tv.allPosts)
	tv.loadMoreBtn.SetVisible(tv.hasMore)

	if tv.hasMore {
		remaining := len(tv.allPosts) - end
		tv.loadMoreBtn.SetLabel(fmt.Sprintf("Load More (%d remaining)", remaining))
	}
}

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
