package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// HomeView handles the home view with composer and posts
type HomeView struct {
	app                   *MauApp
	page                  *gtk.Box
	postEntry             *gtk.TextView
	searchEntry           *gtk.SearchEntry
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

	// Search section
	hv.buildSearch()

	// Posts list
	hv.buildPostsList()

	// Load initial data
	hv.loadDraft()
	hv.Refresh()

	return hv.page
}

func (hv *HomeView) buildComposer() {
	postGroup := adw.NewPreferencesGroup()
	postGroup.SetTitle("Create a Post")
	postGroup.SetDescription("Share with your network (encrypted & signed)")

	// Header with markdown toggle and character counter
	headerBox := gtk.NewBox(gtk.OrientationHorizontal, 6)

	hv.markdownToggle = gtk.NewToggleButton()
	hv.markdownToggle.SetLabel("Preview")
	hv.markdownToggle.SetIconName("view-paged-symbolic")
	hv.markdownToggle.ConnectToggled(func() {
		hv.updateMarkdownPreview()
	})
	headerBox.Append(hv.markdownToggle)

	hv.charCountLabel = gtk.NewLabel("0 characters")
	hv.charCountLabel.AddCSSClass("char-counter")
	hv.charCountLabel.SetHExpand(true)
	hv.charCountLabel.SetHAlign(gtk.AlignEnd)
	headerBox.Append(hv.charCountLabel)

	// Text view
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetVExpand(false)
	scrolled.SetSizeRequest(-1, 150)

	hv.postEntry = gtk.NewTextView()
	hv.postEntry.SetWrapMode(gtk.WrapWord)
	hv.postEntry.SetMarginTop(6)
	hv.postEntry.SetMarginBottom(6)
	hv.postEntry.SetMarginStart(6)
	hv.postEntry.SetMarginEnd(6)

	// Enable undo/redo
	hv.postEntry.Buffer().SetEnableUndo(true)

	hv.postEntry.Buffer().ConnectChanged(func() {
		hv.updateCharCount()
		hv.debouncedMarkdownPreview()
		hv.saveDraftDelayed()
	})

	scrolled.SetChild(hv.postEntry)

	// Markdown preview
	hv.markdownPreview = gtk.NewLabel("")
	hv.markdownPreview.SetWrap(true)
	hv.markdownPreview.SetMarginTop(6)
	hv.markdownPreview.SetMarginBottom(6)
	hv.markdownPreview.SetMarginStart(6)
	hv.markdownPreview.SetMarginEnd(6)
	hv.markdownPreview.SetVisible(false)
	hv.markdownPreview.SetUseMarkup(true)
	hv.markdownPreview.AddCSSClass("preview-box")

	// Tags entry
	tagsBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	tagsLabel := gtk.NewLabel("Tags:")
	hv.tagEntry = gtk.NewEntry()
	hv.tagEntry.SetPlaceholderText(placeholderTags)
	hv.tagEntry.SetHExpand(true)
	tagsBox.Append(tagsLabel)
	tagsBox.Append(hv.tagEntry)

	// Buttons
	btnBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	btnBox.SetHAlign(gtk.AlignEnd)

	publishBtn := gtk.NewButton()
	publishBtn.SetLabel("Publish")
	publishBtn.AddCSSClass("suggested-action")
	publishBtn.ConnectClicked(func() {
		hv.publishPost()
	})
	btnBox.Append(publishBtn)

	// Compose everything
	composerBox := gtk.NewBox(gtk.OrientationVertical, 6)
	composerBox.Append(headerBox)
	composerBox.Append(scrolled)
	composerBox.Append(hv.markdownPreview)
	composerBox.Append(tagsBox)
	composerBox.Append(btnBox)

	postRow := adw.NewActionRow()
	postRow.SetChild(composerBox)
	postGroup.Add(postRow)

	hv.page.Append(postGroup)
}

func (hv *HomeView) buildSearch() {
	searchBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	searchBox.SetMarginTop(12)

	searchLabel := gtk.NewLabel("Search:")
	hv.searchEntry = gtk.NewSearchEntry()
	hv.searchEntry.SetHExpand(true)
	hv.searchEntry.ConnectSearchChanged(func() {
		hv.Refresh()
	})

	searchBox.Append(searchLabel)
	searchBox.Append(hv.searchEntry)
	hv.page.Append(searchBox)
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

func (hv *HomeView) updateCharCount() {
	buffer := hv.postEntry.Buffer()
	start := buffer.StartIter()
	end := buffer.EndIter()
	text := buffer.Text(start, end, false)
	count := len([]rune(text))
	hv.charCountLabel.SetText(fmt.Sprintf("%d characters", count))
}

func (hv *HomeView) debouncedMarkdownPreview() {
	// Cancel existing timer
	if hv.markdownDebounceTimer != 0 {
		glib.SourceRemove(hv.markdownDebounceTimer)
	}

	// Only render if preview is active
	if !hv.markdownToggle.Active() {
		return
	}

	// Debounce markdown rendering (wait for typing to stop)
	hv.markdownDebounceTimer = glib.TimeoutAdd(markdownDebounce, func() bool {
		hv.updateMarkdownPreview()
		hv.markdownDebounceTimer = 0
		return false
	})
}

func (hv *HomeView) updateMarkdownPreview() {
	if !hv.markdownToggle.Active() {
		hv.markdownPreview.SetVisible(false)
		hv.postEntry.SetVisible(true)
		return
	}

	buffer := hv.postEntry.Buffer()
	start := buffer.StartIter()
	end := buffer.EndIter()
	text := buffer.Text(start, end, false)

	pango := hv.app.mdRenderer.ToPango(text)
	hv.markdownPreview.SetMarkup(pango)
	hv.markdownPreview.SetVisible(true)
	hv.postEntry.SetVisible(false)
}

func (hv *HomeView) saveDraftDelayed() {
	if hv.app.draftSaveTimer != 0 {
		glib.SourceRemove(hv.app.draftSaveTimer)
	}

	hv.app.draftSaveTimer = glib.TimeoutSecondsAdd(draftSaveDelay, func() bool {
		hv.saveDraft()
		hv.app.draftSaveTimer = 0
		return false
	})
}

func (hv *HomeView) saveDraft() {
	buffer := hv.postEntry.Buffer()
	start := buffer.StartIter()
	end := buffer.EndIter()
	text := buffer.Text(start, end, false)

	if text == "" {
		return
	}

	draftPath := filepath.Join(hv.app.dataDir, draftFile)
	if err := os.WriteFile(draftPath, []byte(text), 0600); err != nil {
		hv.app.showToast("Failed to save draft")
	}
}

func (hv *HomeView) loadDraft() {
	draftPath := filepath.Join(hv.app.dataDir, draftFile)
	data, err := os.ReadFile(draftPath)
	if err != nil {
		return
	}

	hv.postEntry.Buffer().SetText(string(data))
	hv.updateCharCount()
}

func (hv *HomeView) clearDraft() {
	draftPath := filepath.Join(hv.app.dataDir, draftFile)
	os.Remove(draftPath)
}

func (hv *HomeView) publishPost() {
	buffer := hv.postEntry.Buffer()
	start := buffer.StartIter()
	end := buffer.EndIter()
	text := buffer.Text(start, end, false)

	if text == "" {
		hv.app.showToast(toastNoContent)
		return
	}

	// Validate and sanitize post body
	if err := ValidatePostBody(text); err != nil {
		hv.app.ShowError(dialogValidateError, err.Error())
		return
	}
	text = SanitizePostBody(text)

	tags := ParseTags(hv.tagEntry.Text())

	author := Author{
		Type:        "Person",
		Name:        hv.app.accountMgr.Account().Name(),
		Email:       hv.app.accountMgr.Account().Email(),
		Fingerprint: hv.app.accountMgr.Account().Fingerprint().String(),
	}

	post := NewPost(text, author, tags)

	if err := hv.app.postMgr.Save(post); err != nil {
		hv.app.ShowError(dialogSaveError, fmt.Sprintf("Failed to save post: %v", err))
		return
	}

	hv.app.showToast(toastPostPublished)
	buffer.SetText("")
	hv.tagEntry.SetText("")
	hv.clearDraft()
	hv.Refresh()
}

// Refresh reloads the posts list
func (hv *HomeView) Refresh() {
	hv.postsList.RemoveAll()

	fpr := hv.app.accountMgr.Account().Fingerprint()
	files, err := hv.app.postMgr.List(fpr, 100)
	if err != nil || len(files) == 0 {
		row := adw.NewActionRow()
		row.SetTitle("No posts yet")
		row.SetSubtitle("Create your first post above")
		hv.postsList.Append(row)
		return
	}

	// Sort by newest first
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() > files[j].Name()
	})

	searchTerm := strings.ToLower(hv.searchEntry.Text())

	for _, file := range files {
		post, err := hv.app.postMgr.Load(file)
		if err != nil {
			continue
		}

		if searchTerm != "" && !strings.Contains(strings.ToLower(post.Body), searchTerm) {
			continue
		}

		row := adw.NewActionRow()
		row.SetTitle(Truncate(post.Body, 80))

		subtitle := post.Published.Format("2006-01-02 15:04")
		if len(post.Tags) > 0 {
			subtitle += " • " + FormatTags(post.Tags)
		}
		row.SetSubtitle(subtitle)

		icon := gtk.NewImageFromIconName("security-high-symbolic")
		row.AddPrefix(icon)

		hv.postsList.Append(row)
	}
}
