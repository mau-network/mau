package main

import (
	"fmt"
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/mau-network/mau"
)

// showError displays an error message in the timeline
func (tv *TimelineView) showError(title, message string) {
	row := adw.NewActionRow()
	row.SetTitle(title)
	row.SetSubtitle(message)
	tv.timelineList.Append(row)
	tv.loadMoreBtn.SetVisible(false)
}

// showNoFriends displays a message when no friends exist
func (tv *TimelineView) showNoFriends() {
	row := adw.NewActionRow()
	row.SetTitle("No friends yet")
	row.SetSubtitle("Add friends to see their posts")
	tv.timelineList.Append(row)
	tv.loadMoreBtn.SetVisible(false)
}

// showNoPosts displays a message when no posts match filters
func (tv *TimelineView) showNoPosts() {
	row := adw.NewActionRow()
	filterAuthorText := strings.TrimSpace(tv.filterStart.Text())
	filterStartDate := tv.parseDate(tv.filterStart.Text())
	filterEndDate := tv.parseDate(tv.filterEnd.Text())

	if filterAuthorText != "" || !filterStartDate.IsZero() || !filterEndDate.IsZero() {
		row.SetTitle("No posts match filters")
		row.SetSubtitle("Try adjusting your filter criteria")
	} else {
		row.SetTitle("No posts from friends yet")
	}
	tv.timelineList.Append(row)
	tv.loadMoreBtn.SetVisible(false)
}

// loadPostsFromFriends loads posts from all friends with filters applied
func (tv *TimelineView) loadPostsFromFriends(friends []*mau.Friend) []timelinePost {
	// Get filter values
	filterAuthorText := strings.TrimSpace(tv.filterStart.Text())
	filterStartDate := tv.parseDate(tv.filterStart.Text())
	filterEndDate := tv.parseDate(tv.filterEnd.Text())

	var posts []timelinePost

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

			posts = append(posts, timelinePost{
				post:       post,
				friendName: friend.Name(),
			})
		}
	}

	return posts
}

// loadMore increments the page and displays the next page
func (tv *TimelineView) loadMore() {
	tv.currentPage++
	tv.displayPage()
}

// displayPage renders the current page of posts
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
