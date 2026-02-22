package main

import (
	"sort"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/mau-network/mau"
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
	tv.initializePage()
	tv.page.Append(tv.buildHeader())
	tv.buildFilters()
	tv.page.Append(tv.buildTimelineGroup())
	tv.page.Append(tv.buildTimelineScrolled())
	tv.page.Append(tv.buildLoadMoreButton())
	tv.Refresh()
	return tv.page
}

func (tv *TimelineView) initializePage() {
	tv.page = gtk.NewBox(gtk.OrientationVertical, 12)
	tv.page.SetMarginTop(12)
	tv.page.SetMarginBottom(12)
	tv.page.SetMarginStart(12)
	tv.page.SetMarginEnd(12)
}

func (tv *TimelineView) buildHeader() *gtk.Box {
	header := gtk.NewBox(gtk.OrientationHorizontal, 12)
	headerLabel := gtk.NewLabel("Network Timeline")
	headerLabel.AddCSSClass("title-1")
	header.Append(headerLabel)
	header.Append(tv.buildRefreshButton())
	return header
}

func (tv *TimelineView) buildRefreshButton() *gtk.Button {
	refreshBtn := gtk.NewButton()
	refreshBtn.SetIconName("view-refresh-symbolic")
	refreshBtn.SetTooltipText("Sync with friends (F5)")
	refreshBtn.ConnectClicked(func() {
		tv.app.syncFriends()
	})
	return refreshBtn
}

func (tv *TimelineView) buildTimelineGroup() *adw.PreferencesGroup {
	timelineGroup := adw.NewPreferencesGroup()
	timelineGroup.SetTitle("Posts from Friends")
	return timelineGroup
}

func (tv *TimelineView) buildTimelineScrolled() *gtk.ScrolledWindow {
	tv.timelineList = NewBoxedListBox()
	timelineScrolled := gtk.NewScrolledWindow()
	timelineScrolled.SetVExpand(true)
	timelineScrolled.SetChild(tv.timelineList)
	return timelineScrolled
}

func (tv *TimelineView) buildLoadMoreButton() *gtk.Box {
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
	return loadMoreBox
}

// Refresh reloads the timeline with filters applied
func (tv *TimelineView) Refresh() {
	tv.resetPagination()
	keyring, err := tv.loadFriendKeyring()
	if err != nil {
		tv.showError("Error loading friends", err.Error())
		return
	}
	tv.refreshWithFriends(keyring.FriendsSet())
}

func (tv *TimelineView) resetPagination() {
	tv.currentPage = 0
	tv.timelineList.RemoveAll()
}

func (tv *TimelineView) loadFriendKeyring() (*mau.Keyring, error) {
	return tv.app.accountMgr.Account().ListFriends()
}

func (tv *TimelineView) refreshWithFriends(friends []*mau.Friend) {
	if len(friends) == 0 {
		tv.showNoFriends()
		return
	}
	tv.allPosts = tv.loadPostsFromFriends(friends)
	if len(tv.allPosts) == 0 {
		tv.showNoPosts()
		return
	}
	tv.sortPostsByNewest()
	tv.displayPage()
}

func (tv *TimelineView) sortPostsByNewest() {
	sort.Slice(tv.allPosts, func(i, j int) bool {
		return tv.allPosts[i].post.Published.After(tv.allPosts[j].post.Published)
	})
}
