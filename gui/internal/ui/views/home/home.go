// Package home provides the home view with post composer and feed.
package home

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/mau-network/mau"
	"github.com/mau-network/mau-gui-poc/internal/adapters/notification"
	"github.com/mau-network/mau-gui-poc/internal/domain/account"
	"github.com/mau-network/mau-gui-poc/internal/domain/post"
	"github.com/mau-network/mau-gui-poc/internal/ui"
	"github.com/mau-network/mau-gui-poc/pkg/markdown"
)

const (
	draftFile       = "draft.txt"
	draftSaveDelay  = 10  // Seconds to wait before auto-saving draft
	markdownDebounce = 500 // Milliseconds to debounce markdown preview
)

// View handles the home view with composer and posts
type View struct {
	accountMgr *account.Manager
	postMgr    *post.Manager
	notifier   *notification.Notifier

	// UI components
	page                  *gtk.Box
	postEntry             *gtk.TextView
	charCountLabel        *gtk.Label
	markdownToggle        *gtk.ToggleButton
	markdownPreview       *gtk.Label
	tagEntry              *gtk.Entry
	postsList             *gtk.ListBox

	// State
	mdRenderer            *markdown.Renderer
	markdownDebounceTimer glib.SourceHandle
	draftSaveTimer        glib.SourceHandle
}

// Config holds view configuration
type Config struct {
	AccountMgr *account.Manager
	PostMgr    *post.Manager
	Notifier   *notification.Notifier
}

// New creates a new home view
func New(cfg Config) *View {
	v := &View{
		accountMgr: cfg.AccountMgr,
		postMgr:    cfg.PostMgr,
		notifier:   cfg.Notifier,
		mdRenderer: markdown.NewRenderer(),
	}
	v.buildUI()
	return v
}

// Widget returns the view's widget
func (v *View) Widget() *gtk.Box {
	return v.page
}

// Refresh reloads the posts list showing posts from self and friends
func (v *View) Refresh() {
	v.postsList.RemoveAll()

	// Get own posts
	fpr := v.accountMgr.Account().Fingerprint()
	ownFiles, err := v.postMgr.List(fpr, 100)
	if err != nil {
		v.notifier.ShowToast(fmt.Sprintf("Error loading posts: %v", err))
	}

	// Get friends' posts
	keyring, err := v.accountMgr.Account().ListFriends()
	var friendFiles []*mau.File
	if err == nil {
		friends := keyring.FriendsSet()
		for _, friend := range friends {
			friendFpr := friend.Fingerprint()
			files, _ := v.postMgr.List(friendFpr, 100)
			friendFiles = append(friendFiles, files...)
		}
	}

	// Combine all files
	allFiles := append(ownFiles, friendFiles...)

	if len(allFiles) == 0 {
		row := adw.NewActionRow()
		row.SetTitle("No posts yet")
		row.SetSubtitle("Create your first post above or add friends to see their posts")
		v.postsList.Append(row)
		return
	}

	// Sort by newest first (files are named with timestamp)
	sort.Slice(allFiles, func(i, j int) bool {
		return allFiles[i].Name() > allFiles[j].Name()
	})

	for _, file := range allFiles {
		p, err := v.postMgr.Load(file)
		if err != nil {
			continue
		}

		row := adw.NewActionRow()
		row.SetTitle(markdown.Truncate(p.Body, 80))

		// Add author name to subtitle
		authorInfo := p.Author.Name
		if p.Author.Name == v.accountMgr.Account().Name() {
			authorInfo = "You"
		}

		subtitle := authorInfo + " • " + p.Published.Format("2006-01-02 15:04")
		if len(p.Tags) > 0 {
			subtitle += " • " + markdown.FormatTags(p.Tags)
		}
		row.SetSubtitle(subtitle)

		icon := gtk.NewImageFromIconName("security-high-symbolic")
		row.AddPrefix(icon)

		v.postsList.Append(row)
	}
}

// buildUI creates and returns the view widget
func (v *View) buildUI() {
	v.page = gtk.NewBox(gtk.OrientationVertical, 12)
	v.page.SetMarginTop(12)
	v.page.SetMarginBottom(12)
	v.page.SetMarginStart(12)
	v.page.SetMarginEnd(12)

	// Welcome header
	welcomeLabel := gtk.NewLabel(fmt.Sprintf("Welcome, %s!", v.accountMgr.Account().Name()))
	welcomeLabel.AddCSSClass("title-1")
	v.page.Append(welcomeLabel)

	// Composer section
	v.buildComposer()

	// Posts list
	v.buildPostsList()

	// Load initial data
	v.loadDraft()
	v.Refresh()
}

// buildComposer builds the post composer UI
func (v *View) buildComposer() {
	postGroup := adw.NewPreferencesGroup()
	postGroup.SetTitle("Create a Post")
	postGroup.SetDescription("Share with your network (encrypted & signed)")

	composerBox := gtk.NewBox(gtk.OrientationVertical, 6)
	composerBox.Append(v.buildComposerHeader())
	composerBox.Append(v.buildComposerTextView())
	composerBox.Append(v.buildComposerPreview())
	composerBox.Append(v.buildComposerTags())
	composerBox.Append(v.buildComposerButtons())

	postRow := adw.NewActionRow()
	postRow.SetChild(composerBox)
	postGroup.Add(postRow)
	v.page.Append(postGroup)
}

// buildComposerHeader creates the header with toggle and counter
func (v *View) buildComposerHeader() *gtk.Box {
	headerBox := gtk.NewBox(gtk.OrientationHorizontal, 6)

	v.markdownToggle = gtk.NewToggleButton()
	v.markdownToggle.SetLabel("Preview")
	v.markdownToggle.SetIconName("view-paged-symbolic")
	v.markdownToggle.ConnectToggled(func() {
		v.updateMarkdownPreview()
	})
	headerBox.Append(v.markdownToggle)

	v.charCountLabel = gtk.NewLabel("0 characters")
	v.charCountLabel.AddCSSClass("char-counter")
	v.charCountLabel.SetHExpand(true)
	v.charCountLabel.SetHAlign(gtk.AlignEnd)
	headerBox.Append(v.charCountLabel)

	return headerBox
}

// buildComposerTextView creates the text entry area
func (v *View) buildComposerTextView() *gtk.ScrolledWindow {
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetVExpand(false)
	scrolled.SetSizeRequest(-1, 150)

	v.postEntry = gtk.NewTextView()
	v.postEntry.SetWrapMode(gtk.WrapWord)
	v.postEntry.SetMarginTop(6)
	v.postEntry.SetMarginBottom(6)
	v.postEntry.SetMarginStart(6)
	v.postEntry.SetMarginEnd(6)
	v.postEntry.Buffer().SetEnableUndo(true)

	v.postEntry.Buffer().ConnectChanged(func() {
		v.updateCharCount()
		v.debouncedMarkdownPreview()
		v.saveDraftDelayed()
	})

	scrolled.SetChild(v.postEntry)
	return scrolled
}

// buildComposerPreview creates the markdown preview area
func (v *View) buildComposerPreview() *gtk.Label {
	v.markdownPreview = gtk.NewLabel("")
	v.markdownPreview.SetWrap(true)
	v.markdownPreview.SetMarginTop(6)
	v.markdownPreview.SetMarginBottom(6)
	v.markdownPreview.SetMarginStart(6)
	v.markdownPreview.SetMarginEnd(6)
	v.markdownPreview.SetVisible(false)
	v.markdownPreview.SetUseMarkup(true)
	v.markdownPreview.AddCSSClass("preview-box")
	return v.markdownPreview
}

// buildComposerTags creates the tags input area
func (v *View) buildComposerTags() *gtk.Box {
	tagsBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	tagsLabel := gtk.NewLabel("Tags:")
	v.tagEntry = gtk.NewEntry()
	v.tagEntry.SetPlaceholderText(ui.PlaceholderTags)
	v.tagEntry.SetHExpand(true)
	tagsBox.Append(tagsLabel)
	tagsBox.Append(v.tagEntry)
	return tagsBox
}

// buildComposerButtons creates the action buttons
func (v *View) buildComposerButtons() *gtk.Box {
	btnBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	btnBox.SetHAlign(gtk.AlignEnd)

	publishBtn := gtk.NewButton()
	publishBtn.SetLabel("Publish")
	publishBtn.AddCSSClass("suggested-action")
	publishBtn.ConnectClicked(func() {
		v.publishPost()
	})
	btnBox.Append(publishBtn)
	return btnBox
}

func (v *View) buildPostsList() {
	postsGroup := adw.NewPreferencesGroup()
	postsGroup.SetTitle("Your Posts")

	v.postsList = ui.NewBoxedListBox()

	postsScrolled := gtk.NewScrolledWindow()
	postsScrolled.SetVExpand(true)
	postsScrolled.SetChild(v.postsList)

	v.page.Append(postsGroup)
	v.page.Append(postsScrolled)
}

// updateCharCount updates the character counter label
func (v *View) updateCharCount() {
	buffer := v.postEntry.Buffer()
	start := buffer.StartIter()
	end := buffer.EndIter()
	text := buffer.Text(start, end, false)
	count := len([]rune(text))
	v.charCountLabel.SetText(fmt.Sprintf("%d characters", count))
}

// debouncedMarkdownPreview delays markdown rendering
func (v *View) debouncedMarkdownPreview() {
	if v.markdownDebounceTimer != 0 {
		glib.SourceRemove(v.markdownDebounceTimer)
	}

	if !v.markdownToggle.Active() {
		return
	}

	v.markdownDebounceTimer = glib.TimeoutAdd(markdownDebounce, func() bool {
		v.updateMarkdownPreview()
		v.markdownDebounceTimer = 0
		return false
	})
}

// updateMarkdownPreview renders markdown preview
func (v *View) updateMarkdownPreview() {
	if !v.markdownToggle.Active() {
		v.markdownPreview.SetVisible(false)
		v.postEntry.SetVisible(true)
		return
	}

	text := v.getPostText()
	pango := v.mdRenderer.ToPango(text)
	v.markdownPreview.SetMarkup(pango)
	v.markdownPreview.SetVisible(true)
	v.postEntry.SetVisible(false)
}

// getPostText retrieves the current post text from buffer
func (v *View) getPostText() string {
	buffer := v.postEntry.Buffer()
	start := buffer.StartIter()
	end := buffer.EndIter()
	return buffer.Text(start, end, false)
}

// publishPost handles post publishing
func (v *View) publishPost() {
	text := v.getPostText()

	if text == "" {
		v.notifier.ShowToast(ui.ToastNoContent)
		return
	}

	if err := v.validateAndPublish(text); err != nil {
		v.notifier.ShowError(ui.DialogValidateError, err.Error())
		return
	}

	v.clearComposer()
	v.notifier.ShowToast(ui.ToastPostPublished)

	// Refresh after a brief delay to ensure file is on disk
	glib.IdleAdd(func() bool {
		v.Refresh()
		return false
	})
}

// validateAndPublish validates and publishes a post
func (v *View) validateAndPublish(text string) error {
	if err := markdown.ValidatePostBody(text); err != nil {
		return err
	}
	text = markdown.SanitizePostBody(text)
	tags := markdown.ParseTags(v.tagEntry.Text())

	author := post.Author{
		Type:        "Person",
		Name:        v.accountMgr.Account().Name(),
		Email:       v.accountMgr.Account().Email(),
		Fingerprint: v.accountMgr.Account().Fingerprint().String(),
	}

	p := post.New(text, author, tags)
	return v.postMgr.Save(p)
}

// clearComposer clears the composer UI
func (v *View) clearComposer() {
	v.postEntry.Buffer().SetText("")
	v.tagEntry.SetText("")
	v.clearDraft()
}

// saveDraftDelayed saves the draft after a delay to avoid excessive writes
func (v *View) saveDraftDelayed() {
	if v.draftSaveTimer != 0 {
		glib.SourceRemove(v.draftSaveTimer)
	}

	v.draftSaveTimer = glib.TimeoutSecondsAdd(draftSaveDelay, func() bool {
		v.saveDraft()
		v.draftSaveTimer = 0
		return false
	})
}

// saveDraft saves the current post content to disk
func (v *View) saveDraft() {
	buffer := v.postEntry.Buffer()
	start := buffer.StartIter()
	end := buffer.EndIter()
	text := buffer.Text(start, end, false)

	if text == "" {
		return
	}

	dataDir := v.accountMgr.DataDir()
	draftPath := filepath.Join(dataDir, draftFile)
	if err := os.WriteFile(draftPath, []byte(text), 0600); err != nil {
		v.notifier.ShowToast("Failed to save draft")
	}
}

// loadDraft loads the saved draft from disk
func (v *View) loadDraft() {
	dataDir := v.accountMgr.DataDir()
	draftPath := filepath.Join(dataDir, draftFile)
	data, err := os.ReadFile(draftPath)
	if err != nil {
		return
	}

	v.postEntry.Buffer().SetText(string(data))
	v.updateCharCount()
}

// clearDraft removes the draft file
func (v *View) clearDraft() {
	dataDir := v.accountMgr.DataDir()
	draftPath := filepath.Join(dataDir, draftFile)
	os.Remove(draftPath)
}
