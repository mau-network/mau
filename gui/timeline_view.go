package main

import (
	"fmt"
	"sort"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// TimelineView handles the timeline view
type TimelineView struct {
	app          *MauApp
	page         *gtk.Box
	timelineList *gtk.ListBox
	filterAuthor *gtk.DropDown
	filterStart  *gtk.Entry
	filterEnd    *gtk.Entry
}

// NewTimelineView creates a new timeline view
func NewTimelineView(app *MauApp) *TimelineView {
	return &TimelineView{app: app}
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

	tv.timelineList = gtk.NewListBox()
	tv.timelineList.AddCSSClass("boxed-list")

	timelineScrolled := gtk.NewScrolledWindow()
	timelineScrolled.SetVExpand(true)
	timelineScrolled.SetChild(tv.timelineList)

	tv.page.Append(timelineGroup)
	tv.page.Append(timelineScrolled)

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

// Refresh reloads the timeline
func (tv *TimelineView) Refresh() {
	tv.timelineList.RemoveAll()

	keyring, err := tv.app.accountMgr.Account().ListFriends()
	if err != nil {
		row := adw.NewActionRow()
		row.SetTitle("Error loading friends")
		tv.timelineList.Append(row)
		return
	}

	friends := keyring.FriendsSet()
	if len(friends) == 0 {
		row := adw.NewActionRow()
		row.SetTitle("No friends yet")
		row.SetSubtitle("Add friends to see their posts")
		tv.timelineList.Append(row)
		return
	}

	type timelinePost struct {
		post       Post
		friendName string
	}
	var allPosts []timelinePost

	for _, friend := range friends {
		fpr := friend.Fingerprint()
		files, _ := tv.app.postMgr.List(fpr, 50)

		for _, file := range files {
			post, err := tv.app.postMgr.Load(file)
			if err != nil {
				continue
			}

			allPosts = append(allPosts, timelinePost{
				post:       post,
				friendName: friend.Name(),
			})
		}
	}

	if len(allPosts) == 0 {
		row := adw.NewActionRow()
		row.SetTitle("No posts from friends yet")
		tv.timelineList.Append(row)
		return
	}

	// Sort by newest first
	sort.Slice(allPosts, func(i, j int) bool {
		return allPosts[i].post.Published.After(allPosts[j].post.Published)
	})

	for _, tp := range allPosts {
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
}
