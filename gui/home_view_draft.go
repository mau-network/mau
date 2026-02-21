package main

import (
	"os"
	"path/filepath"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

// saveDraftDelayed saves the draft after a delay to avoid excessive writes
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

// saveDraft saves the current post content to disk
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

// loadDraft loads the saved draft from disk
func (hv *HomeView) loadDraft() {
	draftPath := filepath.Join(hv.app.dataDir, draftFile)
	data, err := os.ReadFile(draftPath)
	if err != nil {
		return
	}

	hv.postEntry.Buffer().SetText(string(data))
	hv.updateCharCount()
}

// clearDraft removes the draft file
func (hv *HomeView) clearDraft() {
	draftPath := filepath.Join(hv.app.dataDir, draftFile)
	os.Remove(draftPath)
}
