// Package ui provides UI constants shared across views.
package ui

const (
	// UI sizing
	DialogDefaultWidth  = 500
	DialogDefaultHeight = -1
	TextViewMinHeight   = 200

	// Validation limits
	MaxPGPKeySize = 50000 // 50KB max PGP key

	// UI Strings - Toast messages
	ToastSyncStarted    = "Syncing with friends..."
	ToastSyncComplete   = "Sync complete"
	ToastSyncFailed     = "Sync failed"
	ToastServerStarted  = "Server started"
	ToastServerStopped  = "Server stopped"
	ToastServerFailed   = "Failed to start server"
	ToastPostPublished  = "Post published!"
	ToastDraftSaved     = "Draft saved"
	ToastFriendAdded    = "Friend added successfully"
	ToastFriendFailed   = "Failed to add friend"
	ToastValidationFail = "Validation failed"
	ToastNoContent      = "Please enter some content"
	ToastNoFriends      = "No friends to sync"

	// Dialog titles
	DialogServerError   = "Server Error"
	DialogValidateError = "Validation Error"
	DialogSaveError     = "Save Error"
	DialogConfirmDelete = "Confirm Deletion"
	DialogConfirmClear  = "Confirm Clear"
	DialogNetworkError  = "Network Error"

	// Placeholder text
	PlaceholderPostBody = "What's on your mind?"
	PlaceholderTags     = "tag1, tag2, tag3"
	PlaceholderDate     = "YYYY-MM-DD"
	PlaceholderAuthor   = "Author name"

	// View titles
	TitleHome     = "Home"
	TitleTimeline = "Timeline"
	TitleFriends  = "Friends"
	TitleNetwork  = "Network"
	TitleSettings = "Settings"

	// Descriptions
	DescNoFriends          = "No friends yet - Add friends to see their posts"
	DescNoPostsFromFriends = "No posts from friends yet"
	DescNoPostsMatching    = "No posts match filters"
	DescAdjustFilters      = "Try adjusting your filter criteria"

	// Error messages
	ErrNetworkUnavailable = "Network connection unavailable. Please check your internet connection and try again."
	ErrServerNotRunning   = "P2P server is not running. Start the server in the Network tab."
	ErrPortInUse          = "The configured server port is already in use. Try changing the port in Settings."
	ErrConnectionTimeout  = "Connection timed out. The server may be unreachable or offline."

	// Friend errors
	ErrInvalidPGPKey      = "Invalid PGP key format. Please paste a valid PGP public key block."
	ErrPGPKeyTooShort     = "PGP key appears truncated (too short). Please paste the complete key."
	ErrPGPKeyTooLarge     = "PGP key is too large (max 50KB). Please verify you pasted only the public key."
	ErrMissingPGPHeaders  = "Missing PGP armor headers. The key must start with '-----BEGIN PGP PUBLIC KEY BLOCK-----'."
	ErrFriendAlreadyAdded = "This friend is already in your network."

	// Post errors
	ErrPostTooLong       = "Post is too long (max 10,000 characters). Please shorten your post."
	ErrTooManyTags       = "Too many tags (max 20). Please remove some tags."
	ErrTagTooLong        = "One or more tags are too long (max 50 characters each)."
	ErrInvalidCharacters = "Post contains invalid characters. Please remove null bytes or control characters."

	// Sync errors
	ErrSyncFailed      = "Failed to sync with friends. This may be a temporary network issue. Retrying..."
	ErrNoFriendsToSync = "You haven't added any friends yet. Add friends in the Friends tab to see their posts."
)
