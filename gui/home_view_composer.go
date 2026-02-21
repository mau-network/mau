package main

import (
	"fmt"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// buildComposer builds the post composer UI
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

// updateCharCount updates the character counter label
func (hv *HomeView) updateCharCount() {
	buffer := hv.postEntry.Buffer()
	start := buffer.StartIter()
	end := buffer.EndIter()
	text := buffer.Text(start, end, false)
	count := len([]rune(text))
	hv.charCountLabel.SetText(fmt.Sprintf("%d characters", count))
}

// debouncedMarkdownPreview delays markdown rendering until typing stops
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

// updateMarkdownPreview renders markdown preview
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

// publishPost handles post publishing
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
