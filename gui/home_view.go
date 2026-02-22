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
	hv.page = gtk.NewBox(gtk.OrientationVertical, 12)
	hv.page.SetMarginTop(12)
	hv.page.SetMarginBottom(12)
	hv.page.SetMarginStart(12)
	hv.page.SetMarginEnd(12)

	// Welcome header
	welcomeLabel := gtk.NewLabel(fmt.Sprintf("Welcome, %s!", hv.app.accountMgr.Account().Name()))
	welcomeLabel.AddCSSClass("title-1")
	hv.page.Append(welcomeLabel)

	// Composer section
	hv.buildComposer()

	// Posts list
	hv.buildPostsList()

	// Load initial data
	hv.loadDraft()
	hv.Refresh()

	return hv.page
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

	// Get own posts
	fpr := hv.app.accountMgr.Account().Fingerprint()
	ownFiles, err := hv.app.postMgr.List(fpr, 100)
	if err != nil {
		hv.app.showToast(fmt.Sprintf("Error loading posts: %v", err))
	}

	// Get friends' posts
	keyring, err := hv.app.accountMgr.Account().ListFriends()
	var friendFiles []*mau.File
	if err == nil {
		friends := keyring.FriendsSet()
		for _, friend := range friends {
			friendFpr := friend.Fingerprint()
			files, _ := hv.app.postMgr.List(friendFpr, 100)
			friendFiles = append(friendFiles, files...)
		}
	}

	// Combine all files
	allFiles := append(ownFiles, friendFiles...)

	if len(allFiles) == 0 {
		row := adw.NewActionRow()
		row.SetTitle("No posts yet")
		row.SetSubtitle("Create your first post above or add friends to see their posts")
		hv.postsList.Append(row)
		return
	}

	// Sort by newest first (files are named with timestamp)
	sort.Slice(allFiles, func(i, j int) bool {
		return allFiles[i].Name() > allFiles[j].Name()
	})

	for _, file := range allFiles {
		post, err := hv.app.postMgr.Load(file)
		if err != nil {
			continue
		}

		row := adw.NewActionRow()
		row.SetTitle(Truncate(post.Body, 80))

		// Add author name to subtitle
		authorInfo := post.Author.Name
		if post.Author.Name == hv.app.accountMgr.Account().Name() {
			authorInfo = "You"
		}

		subtitle := authorInfo + " • " + post.Published.Format("2006-01-02 15:04")
		if len(post.Tags) > 0 {
			subtitle += " • " + FormatTags(post.Tags)
		}
		row.SetSubtitle(subtitle)

		icon := gtk.NewImageFromIconName("security-high-symbolic")
		row.AddPrefix(icon)

		hv.postsList.Append(row)
	}
}
