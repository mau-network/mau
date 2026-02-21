package main

const (
	appID      = "com.mau.Gui"
	appTitle   = "Mau - P2P Social Network"
	draftFile  = "draft.txt"
	configFile = "gui-config.json"

	// Validation limits
	maxPostBodyLength = 10000 // 10KB max post body
	maxTagLength      = 50    // Max characters per tag
	maxTags           = 20    // Max number of tags per post
	maxTagsInput      = 200   // Max characters in tag input field

	// Performance tuning
	draftSaveDelay   = 10  // Seconds to wait before auto-saving draft
	markdownDebounce = 500 // Milliseconds to debounce markdown preview
	postLoadLimit    = 100 // Max posts to load at once
	friendPostLimit  = 50  // Max posts per friend in timeline
)

// UI Strings
const (
	// Common messages
	msgLoading          = "Loading..."
	msgNoData           = "No data available"
	msgError            = "Error"
	msgSuccess          = "Success"
	msgConfirm          = "Are you sure?"
	
	// Toast messages
	toastSyncStarted    = "Syncing with friends..."
	toastSyncComplete   = "Sync complete"
	toastSyncFailed     = "Sync failed"
	toastServerStarted  = "Server started"
	toastServerStopped  = "Server stopped"
	toastServerFailed   = "Failed to start server"
	toastPostPublished  = "Post published!"
	toastDraftSaved     = "Draft saved"
	toastFriendAdded    = "Friend added successfully"
	toastFriendFailed   = "Failed to add friend"
	toastValidationFail = "Validation failed"
	toastNoContent      = "Please enter some content"
	toastNoFriends      = "No friends to sync"
	
	// Dialog titles
	dialogServerError   = "Server Error"
	dialogValidateError = "Validation Error"
	dialogSaveError     = "Save Error"
	dialogConfirmDelete = "Confirm Deletion"
	dialogConfirmClear  = "Confirm Clear"
	
	// Placeholder text
	placeholderPostBody = "What's on your mind?"
	placeholderTags     = "tag1, tag2, tag3"
	placeholderDate     = "YYYY-MM-DD"
	placeholderAuthor   = "Author name"
	
	// View titles
	titleHome          = "Home"
	titleTimeline      = "Timeline"
	titleFriends       = "Friends"
	titleNetwork       = "Network"
	titleSettings      = "Settings"
	
	// Descriptions
	descNoFriends         = "No friends yet - Add friends to see their posts"
	descNoPostsFromFriends = "No posts from friends yet"
	descNoPostsMatching   = "No posts match filters"
	descAdjustFilters     = "Try adjusting your filter criteria"
	descAutoStartServer   = "Start P2P server on launch"
	descAutoSync          = "Automatically sync with friends"
	descDarkMode          = "Use dark color scheme"
	descServerPort        = "Port for P2P server (requires restart)"
	descSyncInterval      = "Minutes between automatic syncs"
)
