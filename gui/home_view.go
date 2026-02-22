package main

import (
	"fmt"
	"sort"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/mau-network/mau"
)

// HomeView handles the home view with composer and posts
type HomeView struct {
	app                   *MauApp
	page                  *gtk.Box
	postEntry             *gtk.TextView
	charCountLabel        *gtk.Label
	markdownToggle        *gtk.ToggleButton
	markdownPreview       *gtk.Label
	tagEntry              *gtk.Entry
	postsList             *gtk.ListBox
	markdownDebounceTimer glib.SourceHandle
}

// NewHomeView creates a new home view
func NewHomeView(app *MauApp) *HomeView {
	return &HomeView{app: app}
}

// Build creates and returns the view widget
func (hv *HomeView) Build() *gtk.Box {
	hv.initializePage()
	hv.addWelcomeHeader()
	hv.buildComposer()
	hv.buildPostsList()
	hv.loadDraft()
	hv.Refresh()
	return hv.page
}

func (hv *HomeView) initializePage() {
	hv.page = gtk.NewBox(gtk.OrientationVertical, 12)
	hv.page.SetMarginTop(12)
	hv.page.SetMarginBottom(12)
	hv.page.SetMarginStart(12)
	hv.page.SetMarginEnd(12)
}

func (hv *HomeView) addWelcomeHeader() {
	welcomeLabel := gtk.NewLabel(fmt.Sprintf("Welcome, %s!", hv.app.accountMgr.Account().Name()))
	welcomeLabel.AddCSSClass("title-1")
	hv.page.Append(welcomeLabel)
}

func (hv *HomeView) buildPostsList() {
	postsGroup := adw.NewPreferencesGroup()
	postsGroup.SetTitle("Your Posts")

	hv.postsList = NewBoxedListBox()

	postsScrolled := gtk.NewScrolledWindow()
	postsScrolled.SetVExpand(true)
	postsScrolled.SetChild(hv.postsList)

	hv.page.Append(postsGroup)
	hv.page.Append(postsScrolled)
}

// Refresh reloads the posts list showing posts from self and friends
func (hv *HomeView) Refresh() {
	hv.postsList.RemoveAll()
	ownFiles, friendFiles := hv.loadAllPosts()
	allFiles := append(ownFiles, friendFiles...)
	if len(allFiles) == 0 {
		hv.showEmptyState()
		return
	}
	hv.displaySortedPosts(allFiles)
}

func (hv *HomeView) loadAllPosts() ([]*mau.File, []*mau.File) {
	ownFiles := hv.loadOwnPosts()
	friendFiles := hv.loadFriendPosts()
	return ownFiles, friendFiles
}

func (hv *HomeView) loadOwnPosts() []*mau.File {
	fpr := hv.app.accountMgr.Account().Fingerprint()
	ownFiles, err := hv.app.postMgr.List(fpr, 100)
	if err != nil {
		hv.app.showToast(fmt.Sprintf("Error loading posts: %v", err))
	}
	return ownFiles
}

func (hv *HomeView) loadFriendPosts() []*mau.File {
	keyring, err := hv.app.accountMgr.Account().ListFriends()
	if err != nil {
		return nil
	}
	var friendFiles []*mau.File
	friends := keyring.FriendsSet()
	for _, friend := range friends {
		friendFpr := friend.Fingerprint()
		files, _ := hv.app.postMgr.List(friendFpr, 100)
		friendFiles = append(friendFiles, files...)
	}
	return friendFiles
}

func (hv *HomeView) showEmptyState() {
	row := adw.NewActionRow()
	row.SetTitle("No posts yet")
	row.SetSubtitle("Create your first post above or add friends to see their posts")
	hv.postsList.Append(row)
}

func (hv *HomeView) displaySortedPosts(allFiles []*mau.File) {
	sort.Slice(allFiles, func(i, j int) bool {
		return allFiles[i].Name() > allFiles[j].Name()
	})
	for _, file := range allFiles {
		if row := hv.createPostRow(file); row != nil {
			hv.postsList.Append(row)
		}
	}
}

func (hv *HomeView) createPostRow(file *mau.File) *adw.ActionRow {
	post, err := hv.app.postMgr.Load(file)
	if err != nil {
		return nil
	}
	row := adw.NewActionRow()
	row.SetTitle(Truncate(post.Body, 80))
	row.SetSubtitle(hv.formatPostSubtitle(post))
	icon := gtk.NewImageFromIconName("security-high-symbolic")
	row.AddPrefix(icon)
	return row
}

func (hv *HomeView) formatPostSubtitle(post Post) string {
	authorInfo := post.Author.Name
	if post.Author.Name == hv.app.accountMgr.Account().Name() {
		authorInfo = "You"
	}
	subtitle := authorInfo + " • " + post.Published.Format("2006-01-02 15:04")
	if len(post.Tags) > 0 {
		subtitle += " • " + FormatTags(post.Tags)
	}
	return subtitle
}
