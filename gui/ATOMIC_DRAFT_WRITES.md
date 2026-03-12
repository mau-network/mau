# Atomic Draft Writes Implementation

## Problem
The draft save functionality in `home_view_draft.go` used `os.WriteFile()` directly, which is not atomic. If the application crashes or the system loses power during a write, the draft file could be corrupted or truncated, losing user work.

## Solution
Implemented atomic file writes using the **write-to-temp-then-rename** pattern:

1. Write content to `draft.txt.tmp`
2. Rename (atomic operation on POSIX) to `draft.txt`
3. Clean up temp file on failure

This ensures the draft file is never in a partially-written state.

## Changes Made

### `gui/home_view_draft.go`
- Modified `saveDraft()` to use atomic writes
- Added temp file creation with `.tmp` suffix
- Added atomic rename operation
- Added cleanup on failure

### `gui/home_view_draft_test.go` (New)
- `TestAtomicDraftWrite`: Verifies atomic write pattern works correctly
- `TestDraftWriteNoCorruption`: Simulates interrupted write, ensures original file unchanged
- `TestDraftPermissions`: Verifies file permissions are secure (0600)

## Benefits
- **Data safety**: Draft cannot be corrupted by crashes
- **Atomic guarantee**: File is always either old version or new version, never partial
- **No data loss**: User work is protected during saves
- **POSIX standard**: `os.Rename()` is atomic on all UNIX-like systems

## Testing
```bash
cd gui
go test -v -run TestAtomicDraftWrite
go test -v -run TestDraftWriteNoCorruption  
go test -v -run TestDraftPermissions
```

## Related
This follows the same pattern already used in `config.go` (lines 84-101) for configuration persistence. Now both config and drafts use atomic writes for consistency.

## Impact
- ✅ Low risk: Change is isolated to draft save logic
- ✅ Backwards compatible: Works with existing draft files
- ✅ Performance: Negligible overhead (one extra rename syscall)
- ✅ Security: Maintains 0600 permissions for privacy
