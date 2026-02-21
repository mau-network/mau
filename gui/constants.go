package main

const (
	appID      = "com.mau.Gui"
	appTitle   = "Mau - P2P Social Network"
	draftFile  = "draft.txt"
	configFile = "gui-config.json"

	// Validation limits
	maxPostBodyLength = 10000  // 10KB max post body
	maxTagLength      = 50     // Max characters per tag
	maxTags           = 20     // Max number of tags per post
	maxTagsInput      = 200    // Max characters in tag input field

	// Performance tuning
	draftSaveDelay    = 10     // Seconds to wait before auto-saving draft
	markdownDebounce  = 500    // Milliseconds to debounce markdown preview
	postLoadLimit     = 100    // Max posts to load at once
	friendPostLimit   = 50     // Max posts per friend in timeline
)
