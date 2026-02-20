# Mau GUI - GTK4/Adwaita Client

A modern GUI client for the Mau P2P social network using Go, GTK4, and Libadwaita.

## Quick Start

```bash
cd gui
./setup.sh    # Install dependencies
make          # Build binary
./mau-gui     # Run
```

**First launch:** Creates `~/.mau-gui-poc/` and generates account with dummy data (Demo User / demo@mau.network).

---

## Features

### ✅ Phase 1 Complete (Current POC)
- **Auto Account Creation** - PGP account with dummy data on first run
- **Home View** - Compose posts (schema.org SocialMediaPosting format)
- **Friends View** - View and add friends UI
- **Settings** - Edit name/email, view fingerprint
- **Modern UI** - GTK4 4.20+ with Libadwaita 1.8+ styling

### 🚧 Limitations (POC)
- Posts/settings log to console (not persisted to disk)
- No PGP signing/encryption in UI
- No peer-to-peer syncing
- No timeline/feed view
- Dummy data only

---

## Building

### Prerequisites

**System Dependencies (Ubuntu/Debian):**
```bash
sudo apt install libgtk-4-dev libadwaita-1-dev pkg-config
```

**Other Platforms:**
- **Fedora:** `sudo dnf install gtk4-devel libadwaita-devel`
- **Arch:** `sudo pacman -S gtk4 libadwaita`
- **macOS:** `brew install gtk4 libadwaita`

**Go:** 1.24+ required (already installed on this system)

### Build Commands

```bash
# Automated
./setup.sh && make

# Manual
go mod download
go build -o mau-gui

# With Make
make install-deps
make build
make run
```

**Build time:** ~60s first build (CGO compilation), ~5-10s incremental

---

## Usage

### First Run
```bash
./mau-gui
```

Creates `~/.mau-gui-poc/` with account:
- **Name:** Demo User
- **Email:** demo@mau.network
- **Password:** "demo"

### Views

**Home (Post Composition):**
1. Type message in text area
2. Click "Publish"
3. Creates JSON-LD SocialMediaPosting
4. Logs to console (POC)

**Friends (Network Management):**
1. Click "Add Friend"
2. Enter fingerprint in dialog
3. Confirms with info message
4. List shows placeholder (POC)

**Settings (Profile):**
1. Edit name/email fields
2. View read-only fingerprint
3. Click "Save Changes"
4. Logs changes (POC)

---

## Architecture

```
MauApp
├── Adwaita Application
├── Mau Account (PGP)
├── ViewStack (3 views)
│   ├── Home (post composition)
│   ├── Friends (network management)
│   └── Settings (profile editor)
└── Event Handlers
    ├── publishPost()
    ├── showAddFriendDialog()
    └── saveSettings()
```

### Data Flow

```
User Input → UI Component → Mau Library → Filesystem
                                ↓
                          PGP Sign/Encrypt
                                ↓
                           JSON+LD File
                                ↓
                        ~/.mau-gui-poc/<FPR>/
```

### UI Design

**Home View:**
```
┌─────────────────────────────────────┐
│     Welcome, Demo User!             │
│                                      │
│  ╔════════════════════════════════╗ │
│  ║ Create a Post                   ║ │
│  ║ [Multi-line text area]          ║ │
│  ║                                 ║ │
│  ╚════════════════════════════════╝ │
│                     [📤 Publish]    │
└─────────────────────────────────────┘
```

**Friends View:**
```
┌─────────────────────────────────────┐
│  ╔════════════════════════════════╗ │
│  ║ 👤 Alice                        ║ │
│  ║    alice@example.com            ║ │
│  ║    FPR: 5D000B2F...             ║ │
│  ╠════════════════════════════════╣ │
│  ║ 👤 Bob                          ║ │
│  ║    bob@example.com              ║ │
│  ╚════════════════════════════════╝ │
│                   [➕ Add Friend]   │
└─────────────────────────────────────┘
```

**Settings View:**
```
┌─────────────────────────────────────┐
│  Name     [Demo User          ]     │
│  Email    [demo@mau.network   ]     │
│                                      │
│  Fingerprint                         │
│  5D000B2F2C040A1675B49D7F...        │
│                 [Save Changes]      │
└─────────────────────────────────────┘
```

---

## Roadmap

### Phase 2: File Persistence (Next)
**Goal:** Make data actually persist to disk

- [ ] Save posts to `~/.mau-gui-poc/<fingerprint>/`
- [ ] Sign posts with PGP key
- [ ] Encrypt posts for followers
- [ ] Save friends to keyring
- [ ] Persist settings changes
- [ ] Load and display user's own posts

### Phase 3: Networking
**Goal:** Enable P2P communication

- [ ] Start/stop HTTP server
- [ ] mDNS peer discovery
- [ ] Kademlia DHT integration
- [ ] Manual/auto sync with friends
- [ ] Server status in UI
- [ ] Progress indicators

### Phase 4: Timeline/Feed
**Goal:** View friends' posts

- [ ] Timeline view (4th tab)
- [ ] Load posts from friends
- [ ] Sort by date (newest first)
- [ ] Post cards with author info
- [ ] Signature verification display
- [ ] Pagination/infinite scroll

### Phase 5: Rich Content
**Goal:** Multimedia support

- [ ] Markdown preview in composer
- [ ] Markdown rendering in feed
- [ ] Image attachments
- [ ] File attachments
- [ ] Character counter
- [ ] Draft saving

### Phase 6: Polish
**Goal:** Production-ready UX

- [ ] Dark mode (auto-detect from system)
- [ ] Keyboard shortcuts
- [ ] Desktop notifications (libnotify)
- [ ] Toast messages
- [ ] Better error dialogs
- [ ] Auto-start server option
- [ ] Auto-sync configuration

### Phase 7: Advanced Features
**Goal:** Power user features

- [ ] Search posts by content
- [ ] Filter by author/date
- [ ] Tag posts with topics
- [ ] Group friends by tags
- [ ] Multi-account support
- [ ] Account switcher

### Technical Debt
- [ ] Split main.go into modules (views/, widgets/, models/)
- [ ] Add comprehensive error handling
- [ ] Structured logging (slog)
- [ ] Configuration file (TOML)
- [ ] Unit tests
- [ ] Integration tests
- [ ] Debian/Flatpak packages

---

## Mau Library Integration

### Account Operations
```go
// Create account
account, err := mau.NewAccount(dataDir, name, email, password)

// Sign content
signedData, err := account.Sign(jsonData)

// Encrypt for recipient
encryptedData, err := account.Encrypt(signedData, recipientFPR)

// Add friend
account.AddFriend(friendKey, "friends")
```

### Networking
```go
// Start server
server := mau.NewServer(account, ":8080")
go server.Start()

// Sync with friend
client := mau.NewClient(account)
client.Sync(friendFingerprint, friendAddress)
```

---

## Development

### Project Structure
```
gui/
├── main.go          # Application logic (391 lines)
├── go.mod           # Dependencies
├── README.md        # This file
├── Makefile         # Build automation
├── setup.sh         # Dependency installer
├── test.sh          # Build verification
└── demo.sh          # Feature demo
```

### Dependencies
- `github.com/diamondburned/gotk4-adwaita` - Adwaita bindings
- `github.com/diamondburned/gotk4/pkg/gtk/v4` - GTK4 bindings
- `github.com/mau-network/mau` - Mau library (local via replace directive)

### Build System
- **CGO:** Required (GTK bindings)
- **Build time:** ~60s initial, ~5-10s incremental
- **Binary size:** ~23MB (with debug symbols)

### GTK4/Adwaita Patterns
- Use `adw.PreferencesGroup` for sections
- Use `adw.ActionRow` for list items
- Use `adw.MessageDialog` for dialogs
- Button styles: `suggested-action` (blue), `destructive-action` (red)
- Lists: `gtk.ListBox` with `boxed-list` CSS class

---

## Testing

### Automated Verification
```bash
./test.sh
```

Checks:
- ✅ Binary exists and is executable
- ✅ Help flag works
- ✅ GTK4 version detected
- ✅ Libadwaita version detected
- ✅ GObject Introspection present

### Manual Testing
```bash
# Reset data
rm -rf ~/.mau-gui-poc

# Fresh start
./mau-gui
```

---

## Troubleshooting

**"GTK4 not found":**
```bash
sudo apt install libgtk-4-dev
pkg-config --modversion gtk4
```

**"Libadwaita not found":**
```bash
sudo apt install libadwaita-1-dev
pkg-config --modversion libadwaita-1
```

**CGO build errors:**
```bash
export CGO_ENABLED=1
go build -o mau-gui
```

**"no DISPLAY" error:**
Requires X11/Wayland. For headless testing, use Xvfb.

---

## Known Issues

### CGO Warnings (Non-Critical)
```
warning: conflicting types for built-in function 'free'
```
These are expected from GTK bindings and don't affect functionality.

### API Quirks
- `mau.LoadAccount()` doesn't exist - use `mau.NewAccount()` for both create and load
- Used local gotk4-adwaita clone due to version resolution issues
- GTK4 `Widgetter` doesn't have `NextSibling()` - use `ListBox.RemoveAll()`

---

## Performance

- Use `gtk.ListBox` for small lists (<1000 items)
- Lazy load posts in timeline
- Cache rendered content
- Debounce search/filter
- Background threads for crypto operations
- Pagination for memory efficiency

---

## Security

- Never store password in memory longer than necessary
- Use secure memory for PGP operations
- Clear clipboard after copy operations
- Warn before following untrusted links
- Validate all friend fingerprints
- File permissions: 0600 for private data
- Verify signatures before displaying content

---

## License

Same as Mau project (check parent directory)

---

## Status

**Phase 1:** ✅ Complete (POC)  
**Next:** Phase 2 (File Persistence)  
**Timeline:** 2-3 days for full Phase 2 implementation
