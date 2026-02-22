package main

import (
	"fmt"
	"strings"
	"time"

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
	filterAuthorText := strings.TrimSpace(tv.filterStart.Text())
	filterStartDate := tv.parseDate(tv.filterStart.Text())
	filterEndDate := tv.parseDate(tv.filterEnd.Text())
	return tv.collectFilteredPosts(friends, filterAuthorText, filterStartDate, filterEndDate)
}

func (tv *TimelineView) collectFilteredPosts(friends []*mau.Friend, author string, start, end time.Time) []timelinePost {
	var posts []timelinePost
	for _, friend := range friends {
		posts = append(posts, tv.loadFriendPosts(friend, author, start, end)...)
	}
	return posts
}

func (tv *TimelineView) loadFriendPosts(friend *mau.Friend, author string, start, end time.Time) []timelinePost {
	fpr := friend.Fingerprint()
	files, _ := tv.app.postMgr.List(fpr, friendPostLimit)
	var posts []timelinePost
	for _, file := range files {
		if post := tv.loadAndFilterPost(file, friend, author, start, end); post != nil {
			posts = append(posts, *post)
		}
	}
	return posts
}

func (tv *TimelineView) loadAndFilterPost(file *mau.File, friend *mau.Friend, author string, start, end time.Time) *timelinePost {
	post, err := tv.app.postMgr.Load(file)
	if err != nil {
		return nil
	}
	if !tv.matchesFilters(post, friend.Name(), author, start, end) {
		return nil
	}
	return &timelinePost{
		post:       post,
		friendName: friend.Name(),
	}
}

// loadMore increments the page and displays the next page
func (tv *TimelineView) loadMore() {
	tv.currentPage++
	tv.displayPage()
}

// displayPage renders the current page of posts
func (tv *TimelineView) displayPage() {
	start, end := tv.calculatePageBounds()
	if start >= len(tv.allPosts) {
		tv.hideLoadMore()
		return
	}
	tv.renderPagePosts(start, end)
	tv.updateLoadMoreButton(end)
}

func (tv *TimelineView) calculatePageBounds() (int, int) {
	start := tv.currentPage * tv.pageSize
	end := start + tv.pageSize
	if end > len(tv.allPosts) {
		end = len(tv.allPosts)
	}
	return start, end
}

func (tv *TimelineView) hideLoadMore() {
	tv.hasMore = false
	tv.loadMoreBtn.SetVisible(false)
}

func (tv *TimelineView) renderPagePosts(start, end int) {
	for i := start; i < end; i++ {
		tv.timelineList.Append(tv.createPostRow(tv.allPosts[i]))
	}
}

func (tv *TimelineView) createPostRow(tp timelinePost) *adw.ActionRow {
	row := adw.NewActionRow()
	row.SetTitle(Truncate(tp.post.Body, 80))
	row.SetSubtitle(tv.formatPostSubtitle(tp))
	tv.addPostIcons(row)
	return row
}

func (tv *TimelineView) formatPostSubtitle(tp timelinePost) string {
	subtitle := fmt.Sprintf("%s • %s", tp.friendName, tp.post.Published.Format("2006-01-02 15:04"))
	if len(tp.post.Tags) > 0 {
		subtitle += " • " + FormatTags(tp.post.Tags)
	}
	return subtitle
}

func (tv *TimelineView) addPostIcons(row *adw.ActionRow) {
	icon := gtk.NewImageFromIconName("avatar-default-symbolic")
	row.AddPrefix(icon)
	verifiedIcon := gtk.NewImageFromIconName("emblem-ok-symbolic")
	row.AddSuffix(verifiedIcon)
}

func (tv *TimelineView) updateLoadMoreButton(end int) {
	tv.hasMore = end < len(tv.allPosts)
	tv.loadMoreBtn.SetVisible(tv.hasMore)
	if tv.hasMore {
		remaining := len(tv.allPosts) - end
		tv.loadMoreBtn.SetLabel(fmt.Sprintf("Load More (%d remaining)", remaining))
	}
}
