# TODO - Mau GUI

## Phase 2: File Persistence & Core Functionality

### Post Management
- [ ] Save posts to disk (encrypted files)
- [ ] Sign posts with PGP key
- [ ] Encrypt posts for followers
- [ ] Load and display user's own posts
- [ ] Delete posts functionality

### Friend Management
- [ ] Import friend public keys from file
- [ ] Save friends to keyring
- [ ] Display friend list from account
- [ ] Remove friends functionality
- [ ] Follow/unfollow friends

### Settings Persistence
- [ ] Actually save name/email changes to account
- [ ] Reload account on settings change
- [ ] Validate input fields

## Phase 3: Networking

### HTTP Server
- [ ] Start/stop HTTP server for sharing posts
- [ ] Configure port in settings
- [ ] Display server status in UI
- [ ] Server controls (start/stop button)

### Peer Discovery
- [ ] Implement mDNS browser for local peers
- [ ] Display discovered peers in Friends view
- [ ] Add "Connect to peer" functionality
- [ ] Kademlia DHT integration

### Sync
- [ ] Download posts from friends (manual trigger)
- [ ] Auto-sync on interval
- [ ] Sync status indicator
- [ ] Progress bar for downloads

## Phase 4: Timeline/Feed

### Post Display
- [ ] Timeline view showing all posts (yours + friends)
- [ ] Sort by date (newest first)
- [ ] Post cards with author info
- [ ] Pagination or infinite scroll

### Post Interaction
- [ ] Click to view full post
- [ ] Author fingerprint display
- [ ] Timestamp display
- [ ] Verified signature indicator

## Phase 5: Rich Content

### Post Composer
- [ ] Markdown preview
- [ ] Image attachment support
- [ ] File attachment support
- [ ] Character counter
- [ ] Draft saving

### Post Renderer
- [ ] Markdown rendering in timeline
- [ ] Image display (inline or gallery)
- [ ] File download links
- [ ] Code syntax highlighting

## Phase 6: Polish

### UI/UX
- [ ] Dark mode support (auto-detect from system)
- [ ] Keyboard shortcuts
- [ ] Notifications (libnotify integration)
- [ ] Toast messages for actions
- [ ] Error dialogs with details

### Settings
- [ ] Auto-start server on launch
- [ ] Auto-sync interval configuration
- [ ] Theme selection
- [ ] Data directory selection
- [ ] Export account backup

### Performance
- [ ] Lazy loading for timeline
- [ ] Background sync thread
- [ ] Cache friend keys in memory
- [ ] Optimize PGP operations

## Phase 7: Advanced Features

### Search
- [ ] Search posts by content
- [ ] Search friends by name/email
- [ ] Filter timeline by author
- [ ] Date range filters

### Groups/Tags
- [ ] Tag posts with topics
- [ ] Filter by tags
- [ ] Group friends by tags
- [ ] Share posts with specific groups

### Multi-Account
- [ ] Switch between accounts
- [ ] Account selector in header
- [ ] Separate data directories per account

## Technical Debt

### Code Quality
- [ ] Split main.go into multiple files (views/, widgets/, models/)
- [ ] Add error handling for all PGP operations
- [ ] Add logging framework (slog)
- [ ] Add configuration file (TOML/YAML)

### Testing
- [ ] Unit tests for data models
- [ ] Integration tests for file operations
- [ ] UI tests (if possible with GTK4)
- [ ] E2E tests for sync workflow

### Build & Distribution
- [ ] GitHub Actions workflow for releases
- [ ] Debian package (.deb)
- [ ] Flatpak package
- [ ] AppImage
- [ ] Windows build (if feasible)
- [ ] macOS build (if feasible)

### Documentation
- [ ] User manual (markdown + HTML)
- [ ] Screenshots for README
- [ ] Video demo
- [ ] Contribution guide
- [ ] Architecture documentation

---

## Priority Order

1. **Phase 2** - Without persistence, nothing is usable
2. **Phase 4** - Timeline is the main feature (reading posts)
3. **Phase 3** - Networking enables the P2P aspect
4. **Phase 5** - Rich content makes it useful
5. **Phase 6** - Polish makes it pleasant
6. **Phase 7** - Advanced features for power users

## Current Status

✅ **Phase 1: POC Complete**
- UI skeleton with 3 views
- Account creation
- Dialog interactions
- Basic navigation

🚧 **Next Up: Phase 2** - Make it actually save data!
