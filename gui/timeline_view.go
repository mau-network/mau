package main

import (
	"sort"

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
	app          *MauApp
	page         *gtk.Box
	timelineList *gtk.ListBox
	filterAuthor *gtk.DropDown
	filterStart  *gtk.Entry
	filterEnd    *gtk.Entry
	loadMoreBtn  *gtk.Button
	currentPage  int
	pageSize     int
	allPosts     []timelinePost
	hasMore      bool
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

// Refresh reloads the timeline with filters applied
func (tv *TimelineView) Refresh() {
	// Reset pagination
	tv.currentPage = 0
	tv.timelineList.RemoveAll()

	keyring, err := tv.app.accountMgr.Account().ListFriends()
	if err != nil {
		tv.showError("Error loading friends", err.Error())
		return
	}

	friends := keyring.FriendsSet()
	if len(friends) == 0 {
		tv.showNoFriends()
		return
	}

	// Load and filter posts
	tv.allPosts = tv.loadPostsFromFriends(friends)

	if len(tv.allPosts) == 0 {
		tv.showNoPosts()
		return
	}

	// Sort by newest first
	sort.Slice(tv.allPosts, func(i, j int) bool {
		return tv.allPosts[i].post.Published.After(tv.allPosts[j].post.Published)
	})

	// Display first page
	tv.displayPage()
}
