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
	postGroup.SetDescription("Share with your network (encrypted &amp; signed)")

	composerBox := gtk.NewBox(gtk.OrientationVertical, 6)
	composerBox.Append(hv.buildComposerHeader())
	composerBox.Append(hv.buildComposerTextView())
	composerBox.Append(hv.buildComposerPreview())
	composerBox.Append(hv.buildComposerTags())
	composerBox.Append(hv.buildComposerButtons())

	postRow := adw.NewActionRow()
	postRow.SetChild(composerBox)
	postGroup.Add(postRow)
	hv.page.Append(postGroup)
}

// buildComposerHeader creates the header with toggle and counter
func (hv *HomeView) buildComposerHeader() *gtk.Box {
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

	return headerBox
}

// buildComposerTextView creates the text entry area
func (hv *HomeView) buildComposerTextView() *gtk.ScrolledWindow {
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetVExpand(false)
	scrolled.SetSizeRequest(-1, 150)

	hv.postEntry = gtk.NewTextView()
	hv.postEntry.SetWrapMode(gtk.WrapWord)
	hv.postEntry.SetMarginTop(6)
	hv.postEntry.SetMarginBottom(6)
	hv.postEntry.SetMarginStart(6)
	hv.postEntry.SetMarginEnd(6)
	hv.postEntry.Buffer().SetEnableUndo(true)

	hv.postEntry.Buffer().ConnectChanged(func() {
		hv.updateCharCount()
		hv.debouncedMarkdownPreview()
		hv.saveDraftDelayed()
	})

	scrolled.SetChild(hv.postEntry)
	return scrolled
}

// buildComposerPreview creates the markdown preview area
func (hv *HomeView) buildComposerPreview() *gtk.Label {
	hv.markdownPreview = gtk.NewLabel("")
	hv.markdownPreview.SetWrap(true)
	hv.markdownPreview.SetMarginTop(6)
	hv.markdownPreview.SetMarginBottom(6)
	hv.markdownPreview.SetMarginStart(6)
	hv.markdownPreview.SetMarginEnd(6)
	hv.markdownPreview.SetVisible(false)
	hv.markdownPreview.SetUseMarkup(true)
	hv.markdownPreview.AddCSSClass("preview-box")
	return hv.markdownPreview
}

// buildComposerTags creates the tags input area
func (hv *HomeView) buildComposerTags() *gtk.Box {
	tagsBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	tagsLabel := gtk.NewLabel("Tags:")
	hv.tagEntry = gtk.NewEntry()
	hv.tagEntry.SetPlaceholderText(placeholderTags)
	hv.tagEntry.SetHExpand(true)
	tagsBox.Append(tagsLabel)
	tagsBox.Append(hv.tagEntry)
	return tagsBox
}

// buildComposerButtons creates the action buttons
func (hv *HomeView) buildComposerButtons() *gtk.Box {
	btnBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	btnBox.SetHAlign(gtk.AlignEnd)

	publishBtn := gtk.NewButton()
	publishBtn.SetLabel("Publish")
	publishBtn.AddCSSClass("suggested-action")
	publishBtn.ConnectClicked(func() {
		hv.publishPost()
	})
	btnBox.Append(publishBtn)
	return btnBox
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

// debouncedMarkdownPreview delays markdown rendering
func (hv *HomeView) debouncedMarkdownPreview() {
	if hv.markdownDebounceTimer != 0 {
		glib.SourceRemove(hv.markdownDebounceTimer)
	}

	if !hv.markdownToggle.Active() {
		return
	}

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

	text := hv.getPostText()
	pango := hv.app.mdRenderer.ToPango(text)
	hv.markdownPreview.SetMarkup(pango)
	hv.markdownPreview.SetVisible(true)
	hv.postEntry.SetVisible(false)
}

// getPostText retrieves the current post text from buffer
func (hv *HomeView) getPostText() string {
	buffer := hv.postEntry.Buffer()
	start := buffer.StartIter()
	end := buffer.EndIter()
	return buffer.Text(start, end, false)
}

// publishPost handles post publishing
func (hv *HomeView) publishPost() {
	text := hv.getPostText()

	if text == "" {
		hv.app.showToast(toastNoContent)
		return
	}

	if err := hv.validateAndPublish(text); err != nil {
		hv.app.ShowError(dialogValidateError, err.Error())
		return
	}

	hv.clearComposer()
	hv.app.showToast(toastPostPublished)
	
	// Refresh after a brief delay to ensure file is on disk
	glib.IdleAdd(func() bool {
		hv.Refresh()
		return false
	})
}

// validateAndPublish validates and publishes a post
func (hv *HomeView) validateAndPublish(text string) error {
	if err := ValidatePostBody(text); err != nil {
		return err
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
	return hv.app.postMgr.Save(post)
}

// clearComposer clears the composer UI
func (hv *HomeView) clearComposer() {
	hv.postEntry.Buffer().SetText("")
	hv.tagEntry.SetText("")
	hv.clearDraft()
}
