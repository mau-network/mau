// Package timeline provides the network timeline view for viewing friends' posts.
package timeline

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/mau-network/mau"
	"github.com/mau-network/mau-gui-poc/internal/adapters/notification"
	"github.com/mau-network/mau-gui-poc/internal/domain/account"
	"github.com/mau-network/mau-gui-poc/internal/domain/post"
	"github.com/mau-network/mau-gui-poc/internal/ui"
	"github.com/mau-network/mau-gui-poc/pkg/markdown"
)

const (
	pageSize        = 20 // Posts per page
	friendPostLimit = 50 // Max posts per friend
)

// timelinePost represents a post with its author info
type timelinePost struct {
	post       post.Post
	friendName string
}

// View handles the timeline view
type View struct {
	accountMgr *account.Manager
	postMgr    *post.Manager
	notifier   *notification.Notifier

	// Callbacks
	onSyncRequested func()

	// UI components
	page         *gtk.Box
	timelineList *gtk.ListBox
	filterAuthor *gtk.Entry
	filterStart  *gtk.Entry
	filterEnd    *gtk.Entry
	loadMoreBtn  *gtk.Button

	// State
	currentPage int
	allPosts    []timelinePost
	hasMore     bool
}

// Config holds view configuration
type Config struct {
	AccountMgr      *account.Manager
	PostMgr         *post.Manager
	Notifier        *notification.Notifier
	OnSyncRequested func()
}

// New creates a new timeline view
func New(cfg Config) *View {
	v := &View{
		accountMgr:      cfg.AccountMgr,
		postMgr:         cfg.PostMgr,
		notifier:        cfg.Notifier,
		onSyncRequested: cfg.OnSyncRequested,
		currentPage:     0,
		allPosts:        []timelinePost{},
		hasMore:         false,
	}
	v.buildUI()
	return v
}

// Widget returns the view's widget
func (v *View) Widget() *gtk.Box {
	return v.page
}

// Refresh reloads the timeline with filters applied
func (v *View) Refresh() {
	// Reset pagination
	v.currentPage = 0
	v.timelineList.RemoveAll()

	keyring, err := v.accountMgr.Account().ListFriends()
	if err != nil {
		v.showError("Error loading friends", err.Error())
		return
	}

	friends := keyring.FriendsSet()
	if len(friends) == 0 {
		v.showNoFriends()
		return
	}

	// Load and filter posts
	v.allPosts = v.loadPostsFromFriends(friends)

	if len(v.allPosts) == 0 {
		v.showNoPosts()
		return
	}

	// Sort by newest first
	sort.Slice(v.allPosts, func(i, j int) bool {
		return v.allPosts[i].post.Published.After(v.allPosts[j].post.Published)
	})

	// Display first page
	v.displayPage()
}

// buildUI creates and returns the view widget
func (v *View) buildUI() {
	v.page = gtk.NewBox(gtk.OrientationVertical, 12)
	v.page.SetMarginTop(12)
	v.page.SetMarginBottom(12)
	v.page.SetMarginStart(12)
	v.page.SetMarginEnd(12)

	// Header with refresh
	header := gtk.NewBox(gtk.OrientationHorizontal, 12)
	headerLabel := gtk.NewLabel("Network Timeline")
	headerLabel.AddCSSClass("title-1")
	header.Append(headerLabel)

	refreshBtn := gtk.NewButton()
	refreshBtn.SetIconName("view-refresh-symbolic")
	refreshBtn.SetTooltipText("Sync with friends (F5)")
	refreshBtn.ConnectClicked(func() {
		if v.onSyncRequested != nil {
			v.onSyncRequested()
		}
	})
	header.Append(refreshBtn)

	v.page.Append(header)

	// Filters
	v.buildFilters()

	// Timeline list
	timelineGroup := adw.NewPreferencesGroup()
	timelineGroup.SetTitle("Posts from Friends")

	v.timelineList = ui.NewBoxedListBox()

	timelineScrolled := gtk.NewScrolledWindow()
	timelineScrolled.SetVExpand(true)
	timelineScrolled.SetChild(v.timelineList)

	v.page.Append(timelineGroup)
	v.page.Append(timelineScrolled)

	// Load More button
	v.loadMoreBtn = gtk.NewButton()
	v.loadMoreBtn.SetLabel("Load More Posts")
	v.loadMoreBtn.AddCSSClass("suggested-action")
	v.loadMoreBtn.SetVisible(false)
	v.loadMoreBtn.ConnectClicked(func() {
		v.loadMore()
	})

	loadMoreBox := gtk.NewBox(gtk.OrientationHorizontal, 0)
	loadMoreBox.SetHAlign(gtk.AlignCenter)
	loadMoreBox.Append(v.loadMoreBtn)
	v.page.Append(loadMoreBox)

	v.Refresh()
}

// buildFilters creates the filter UI components
func (v *View) buildFilters() {
	filterBox := gtk.NewBox(gtk.OrientationHorizontal, 6)

	// Author filter
	authorLabel := gtk.NewLabel("Author:")
	v.filterAuthor = gtk.NewEntry()
	v.filterAuthor.SetPlaceholderText("Filter by name...")
	v.filterAuthor.SetWidthChars(15)
	filterBox.Append(authorLabel)
	filterBox.Append(v.filterAuthor)

	// Date range
	dateLabel := gtk.NewLabel("From:")
	v.filterStart = gtk.NewEntry()
	v.filterStart.SetPlaceholderText("YYYY-MM-DD")
	v.filterStart.SetWidthChars(12)
	filterBox.Append(dateLabel)
	filterBox.Append(v.filterStart)

	toLabel := gtk.NewLabel("To:")
	v.filterEnd = gtk.NewEntry()
	v.filterEnd.SetPlaceholderText("YYYY-MM-DD")
	v.filterEnd.SetWidthChars(12)
	filterBox.Append(toLabel)
	filterBox.Append(v.filterEnd)

	// Apply button
	applyBtn := gtk.NewButton()
	applyBtn.SetLabel("Apply")
	applyBtn.ConnectClicked(func() {
		v.Refresh()
	})
	filterBox.Append(applyBtn)

	v.page.Append(filterBox)
}

// showError displays an error message in the timeline
func (v *View) showError(title, message string) {
	row := adw.NewActionRow()
	row.SetTitle(title)
	row.SetSubtitle(message)
	v.timelineList.Append(row)
	v.loadMoreBtn.SetVisible(false)
}

// showNoFriends displays a message when no friends exist
func (v *View) showNoFriends() {
	row := adw.NewActionRow()
	row.SetTitle("No friends yet")
	row.SetSubtitle("Add friends to see their posts")
	v.timelineList.Append(row)
	v.loadMoreBtn.SetVisible(false)
}

// showNoPosts displays a message when no posts match filters
func (v *View) showNoPosts() {
	row := adw.NewActionRow()
	filterAuthorText := strings.TrimSpace(v.filterAuthor.Text())
	filterStartDate := v.parseDate(v.filterStart.Text())
	filterEndDate := v.parseDate(v.filterEnd.Text())

	if filterAuthorText != "" || !filterStartDate.IsZero() || !filterEndDate.IsZero() {
		row.SetTitle("No posts match filters")
		row.SetSubtitle("Try adjusting your filter criteria")
	} else {
		row.SetTitle("No posts from friends yet")
	}
	v.timelineList.Append(row)
	v.loadMoreBtn.SetVisible(false)
}

// loadPostsFromFriends loads posts from all friends with filters applied
func (v *View) loadPostsFromFriends(friends []*mau.Friend) []timelinePost {
	// Get filter values
	filterAuthorText := strings.TrimSpace(v.filterAuthor.Text())
	filterStartDate := v.parseDate(v.filterStart.Text())
	filterEndDate := v.parseDate(v.filterEnd.Text())

	var posts []timelinePost

	for _, friend := range friends {
		fpr := friend.Fingerprint()
		files, _ := v.postMgr.List(fpr, friendPostLimit)

		for _, file := range files {
			p, err := v.postMgr.Load(file)
			if err != nil {
				continue
			}

			// Apply filters
			if !v.matchesFilters(p, friend.Name(), filterAuthorText, filterStartDate, filterEndDate) {
				continue
			}

			posts = append(posts, timelinePost{
				post:       p,
				friendName: friend.Name(),
			})
		}
	}

	return posts
}

// matchesFilters checks if a post matches the current filter criteria
func (v *View) matchesFilters(p post.Post, authorName, filterAuthor string, startDate, endDate time.Time) bool {
	// Author filter (case-insensitive substring match)
	if filterAuthor != "" {
		if !strings.Contains(strings.ToLower(authorName), strings.ToLower(filterAuthor)) {
			return false
		}
	}

	// Date range filters
	if !startDate.IsZero() && p.Published.Before(startDate) {
		return false
	}

	if !endDate.IsZero() && p.Published.After(endDate.Add(24*time.Hour-time.Second)) {
		return false
	}

	return true
}

// parseDate parses a date string in YYYY-MM-DD format
func (v *View) parseDate(dateStr string) time.Time {
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

// loadMore increments the page and displays the next page
func (v *View) loadMore() {
	v.currentPage++
	v.displayPage()
}

// displayPage renders the current page of posts
func (v *View) displayPage() {
	start := v.currentPage * pageSize
	end := start + pageSize

	if start >= len(v.allPosts) {
		// No more posts
		v.hasMore = false
		v.loadMoreBtn.SetVisible(false)
		return
	}

	if end > len(v.allPosts) {
		end = len(v.allPosts)
	}

	// Display posts for this page
	for i := start; i < end; i++ {
		tp := v.allPosts[i]
		row := adw.NewActionRow()
		row.SetTitle(markdown.Truncate(tp.post.Body, 80))

		subtitle := fmt.Sprintf("%s • %s", tp.friendName, tp.post.Published.Format("2006-01-02 15:04"))
		if len(tp.post.Tags) > 0 {
			subtitle += " • " + markdown.FormatTags(tp.post.Tags)
		}
		row.SetSubtitle(subtitle)

		icon := gtk.NewImageFromIconName("avatar-default-symbolic")
		row.AddPrefix(icon)

		verifiedIcon := gtk.NewImageFromIconName("emblem-ok-symbolic")
		row.AddSuffix(verifiedIcon)

		v.timelineList.Append(row)
	}

	// Update Load More button visibility
	v.hasMore = end < len(v.allPosts)
	v.loadMoreBtn.SetVisible(v.hasMore)

	if v.hasMore {
		remaining := len(v.allPosts) - end
		v.loadMoreBtn.SetLabel(fmt.Sprintf("Load More (%d remaining)", remaining))
	}
}
