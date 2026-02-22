# GUI Function Length Refactoring Plan

**Goal:** Reduce all functions to ≤20 lines (as per .golangci.yml funlen linter rule)

**Current Status:** 31 functions exceed the limit

## Priority Order (Largest First)

### Critical (>50 lines) - 5 functions
1. ✅ `app_ui.go:loadCSS()` - 68 lines → Already split in previous work
2. ✅ `home_view.go:Refresh()` - 65 lines → Already split  
3. ✅ `friends_view.go:showAddFriendDialog()` - 58 lines → Already split
4. `timeline_view.go:Build()` - 57 lines → NEEDS SPLIT
5. `app.go:buildUI()` - 54 lines → NEEDS SPLIT

### High Priority (40-50 lines) - 3 functions
6. `timeline_view_display.go:displayPage()` - 45 lines → NEEDS SPLIT
7. `settings_view.go:buildSyncSection()` - 43 lines → NEEDS SPLIT
8. `settings_view.go:buildServerSection()` - 42 lines → NEEDS SPLIT

### Medium Priority (30-40 lines) - 6 functions
9. `friends_view.go:Build()` - 37 lines → NEEDS SPLIT
10. `friends_view.go:Refresh()` - 35 lines → NEEDS SPLIT
11. `timeline_view_filters.go:buildFilters()` - 34 lines → NEEDS SPLIT
12. `timeline_view.go:Refresh()` - 33 lines → NEEDS SPLIT
13. `retry.go:RetryWithContext()` - 32 lines → NEEDS SPLIT
14. `timeline_view_display.go:loadPostsFromFriends()` - 32 lines → NEEDS SPLIT

### Low Priority (21-29 lines) - 17 functions
15. `post_utils.go:ParseTags()` - 28 lines → NEEDS SPLIT
16. `friends_view.go:validatePGPKey()` - 27 lines → NEEDS SPLIT
17. `retry.go:RetryOperation()` - 27 lines → NEEDS SPLIT
18. `app_sync.go:syncFriends()` - 26 lines → NEEDS SPLIT
19. `post_manager.go:Save()` - 26 lines → NEEDS SPLIT
20. `settings_view.go:buildAppearanceSection()` - 25 lines → NEEDS SPLIT
21. `settings_view.go:buildAccountSection()` - 25 lines → NEEDS SPLIT
22. `home_view.go:Build()` - 24 lines → NEEDS SPLIT
23. `config.go:Load()` - 24 lines → NEEDS SPLIT
24. `app_ui.go:processToastQueue()` - 23 lines → NEEDS SPLIT
25. `app_sync.go:performSync()` - 23 lines → NEEDS SPLIT
26. `post_manager.go:Load()` - 23 lines → NEEDS SPLIT
27. `home_view_composer.go:publishPost()` - 22 lines → NEEDS SPLIT
28. `home_view_composer.go:buildComposerTextView()` - 22 lines → NEEDS SPLIT
29. `settings_view.go:Build()` - 21 lines → NEEDS SPLIT
30. `app.go:activate()` - 50 lines → NEEDS SPLIT
31. (Additional functions may be discovered during refactoring)

## Refactoring Strategy

### For UI Building Functions (Build, buildXSection)
- Extract widget creation to helper methods
- One method per major UI component
- Target: 5-10 lines per helper

### For Business Logic (Save, Load, Refresh)
- Extract validation → separate function
- Extract error handling → separate function  
- Extract data transformation → separate function

### For Complex Workflows (displayPage, syncFriends)
- Break into preparation → execution → cleanup phases
- Each phase becomes a separate function

## Estimated Time
- Critical + High: 3-4 hours (8 functions)
- Medium: 2-3 hours (6 functions)
- Low: 2-3 hours (17 functions)
- **Total: 7-10 hours**

## Testing Strategy
After each file refactoring:
1. `go build` - ensure compilation
2. `go test ./...` - ensure tests pass
3. Commit with descriptive message

## Success Criteria
- All functions ≤20 lines
- All tests passing
- No functionality changes (pure refactoring)
- golangci-lint funlen passes for gui/
