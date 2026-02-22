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
	maxPGPKeySize     = 50000 // 50KB max PGP key (friends_view.go:191)

	// UI sizing
	dialogDefaultWidth  = 500 // Default width for dialogs
	dialogDefaultHeight = -1  // Auto-height for dialogs
	textViewMinHeight   = 200 // Minimum height for text views

	// Performance tuning
	draftSaveDelay    = 10  // Seconds to wait before auto-saving draft
	markdownDebounce  = 500 // Milliseconds to debounce markdown preview
	postLoadLimit     = 100 // Max posts to load at once
	friendPostLimit   = 50  // Max posts per friend in timeline
	serverStartupWait = 2   // Seconds to wait for server startup
	retryDelay        = 1   // Seconds to wait before retrying operations
	retryInitialDelay = 2   // Seconds for first retry attempt
	retryMaxDelay     = 10  // Seconds maximum delay between retries
	toastDisplayTime  = 4   // Seconds to display each toast (3s timeout + 1s buffer)
	toastTimeout      = 3   // Seconds for toast auto-dismiss
	cacheEntryTTL     = 5   // Minutes before cache entries expire
	cacheMaxSize      = 500 // Maximum number of cached posts
	timelinePageSize  = 20  // Posts per page in timeline
)

// UI Strings
const (
	// Common messages
	msgLoading = "Loading..."
	msgNoData  = "No data available"
	msgError   = "Error"
	msgSuccess = "Success"
	msgConfirm = "Are you sure?"

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
	dialogNetworkError  = "Network Error"

	// Placeholder text
	placeholderPostBody = "What's on your mind?"
	placeholderTags     = "tag1, tag2, tag3"
	placeholderDate     = "YYYY-MM-DD"
	placeholderAuthor   = "Author name"

	// View titles
	titleHome     = "Home"
	titleTimeline = "Timeline"
	titleFriends  = "Friends"
	titleNetwork  = "Network"
	titleSettings = "Settings"

	// Descriptions
	descNoFriends          = "No friends yet - Add friends to see their posts"
	descNoPostsFromFriends = "No posts from friends yet"
	descNoPostsMatching    = "No posts match filters"
	descAdjustFilters      = "Try adjusting your filter criteria"
	descAutoStartServer    = "Start P2P server on launch"
	descAutoSync           = "Automatically sync with friends"
	descDarkMode           = "Use dark color scheme"
	descServerPort         = "Port for P2P server (requires restart)"
	descSyncInterval       = "Minutes between automatic syncs"
)

// User-Friendly Error Messages
const (
	// Network errors
	errNetworkUnavailable = "Network connection unavailable. Please check your internet connection and try again."
	errServerNotRunning   = "P2P server is not running. Start the server in the Network tab."
	errPortInUse          = "The configured server port is already in use. Try changing the port in Settings."
	errConnectionTimeout  = "Connection timed out. The server may be unreachable or offline."

	// Friend errors
	errInvalidPGPKey      = "Invalid PGP key format. Please paste a valid PGP public key block."
	errPGPKeyTooShort     = "PGP key appears truncated (too short). Please paste the complete key."
	errPGPKeyTooLarge     = "PGP key is too large (max 50KB). Please verify you pasted only the public key."
	errMissingPGPHeaders  = "Missing PGP armor headers. The key must start with '-----BEGIN PGP PUBLIC KEY BLOCK-----'."
	errFriendAlreadyAdded = "This friend is already in your network."

	// Post errors
	errPostTooLong       = "Post is too long (max 10,000 characters). Please shorten your post."
	errTooManyTags       = "Too many tags (max 20). Please remove some tags."
	errTagTooLong        = "One or more tags are too long (max 50 characters each)."
	errInvalidCharacters = "Post contains invalid characters. Please remove null bytes or control characters."

	// Sync errors
	errSyncFailed      = "Failed to sync with friends. This may be a temporary network issue. Retrying..."
	errNoFriendsToSync = "You haven't added any friends yet. Add friends in the Friends tab to see their posts."

	// Config errors
	errConfigLoadFailed = "Failed to load configuration. Using default settings."
	errConfigSaveFailed = "Failed to save configuration. Your changes may not persist."
	errInvalidPort      = "Invalid port number. Please choose a port between 1024 and 65535."

	// File errors
	errFileNotFound     = "File not found. It may have been moved or deleted."
	errPermissionDenied = "Permission denied. Check file permissions in ~/.mau-gui directory."
	errDiskFull         = "Disk is full. Free up space and try again."

	// General errors
	errUnknown            = "An unexpected error occurred. Please try again."
	errOperationCancelled = "Operation was cancelled."
)
