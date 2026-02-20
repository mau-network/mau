# Mau GUI POC - Quick Reference

## 🎯 What You Get

A complete proof-of-concept GTK4/Libadwaita GUI client for Mau with:
- **Auto account creation** with dummy data
- **Post composition** (schema.org format)
- **Friend management** UI
- **Settings** panel
- **~880 lines** of code and documentation

## 📂 Files Created

```
mau/gui-poc/
├── main.go          (391 lines) - Complete GTK4/Adwaita application
├── go.mod           - Go module with dependencies
├── README.md        (121 lines) - User guide and build instructions  
├── DESIGN.md        (368 lines) - Detailed UI/UX specifications
├── setup.sh         - Automated dependency installation
├── demo.sh          - Visual demonstration script
├── Makefile         - Build automation
└── .gitignore       - Git ignore rules
```

## ⚡ Quick Start

```bash
cd ~/.openclaw/workspace/mau/gui-poc

# Option 1: Automated setup (recommended)
./setup.sh
./mau-gui

# Option 2: Manual
sudo apt install libgtk-4-dev libadwaita-1-dev  # Ubuntu/Debian
go mod download
go build -o mau-gui
./mau-gui

# Option 3: Using Make
make install-deps
make run
```

## 🎨 Features

### Home View
- Welcome message with user name
- Multi-line text editor for posts
- Publish button (creates schema.org SocialMediaPosting)
- POC: Logs to console instead of persisting

### Friends View
- Scrollable friends list with name/email/fingerprint
- Add friend dialog with fingerprint entry
- POC: Shows placeholder data

### Settings View
- Editable name and email fields
- Read-only fingerprint display
- Save button
- POC: Logs changes instead of persisting

## 🔧 Technical Stack

- **Language:** Go 1.26+
- **GUI Framework:** GTK4 via gotk4
- **UI Library:** Libadwaita (GNOME's modern toolkit)
- **Integration:** Uses existing Mau library from parent directory
- **Platform:** Linux (GNOME desktop)

## 📋 Dependencies

### System (Ubuntu/Debian)
```bash
sudo apt install libgtk-4-dev libadwaita-1-dev pkg-config
```

### Go Modules
- `github.com/diamondburned/gotk4-adwaita` - Adwaita bindings
- `github.com/diamondburned/gotk4/pkg/gtk/v4` - GTK4 bindings
- `github.com/mau-network/mau` - Mau P2P library (local)

## 🚀 What It Does

1. **First Launch:**
   - Creates `~/.mau-gui-poc/` directory
   - Generates PGP account with dummy data:
     - Name: "Demo User"
     - Email: "demo@mau.network"
   - Shows main window with 3 tabs

2. **Post Creation:**
   - Type content in text area
   - Click "Publish"
   - Creates JSON-LD object (schema.org SocialMediaPosting)
   - Logs to console: `Publishing post: {...}`

3. **Friend Management:**
   - View friends list (placeholder in POC)
   - Click "Add Friend"
   - Enter fingerprint (dialog UI ready, validation pending)
   - Confirms with info dialog

4. **Settings:**
   - Edit name/email in text fields
   - View account fingerprint
   - Click "Save Changes"
   - Logs: `Saving settings: name=..., email=...`

## 🔮 POC vs Full Implementation

### ✅ Implemented (POC)
- Complete UI layout with 3 views
- Account initialization (creates Mau account)
- Post composition interface
- Friends list interface
- Settings interface
- Modern Adwaita styling
- Dialog interactions

### 🚧 Not Implemented (Next Steps)
- File persistence (posts written to disk)
- PGP signing/encryption of posts
- Friend keyring management (add/remove)
- HTTP server integration
- Peer discovery (mDNS, Kademlia)
- Post timeline/feed
- Sync daemon
- Notifications

## 📖 Documentation

### README.md
- Prerequisites and installation
- Building and running instructions
- Usage guide
- POC limitations
- Next steps for full implementation

### DESIGN.md
- ASCII art wireframes of all 3 views
- Data flow diagrams
- File operation details
- Mau library integration examples
- GTK4/Adwaita best practices
- 6-phase roadmap
- Testing strategy
- Security notes

### demo.sh
- Visual demonstration of features
- Build instructions
- Demo scenario walkthrough
- Code metrics
- Roadmap overview

## 💡 Integration with Mau

The POC uses Mau's existing Go library:

```go
// Account creation (implemented)
account, err := mau.NewAccount(dataDir, name, email, password)

// Account loading (implemented)
account, err := mau.LoadAccount(dataDir, password)

// Ready for integration (not yet wired):
// - account.Sign(data)
// - account.Encrypt(data, fingerprint)
// - account.AddFriend(key, group)
// - mau.NewServer(account, port)
// - client.Sync(friendFPR, address)
```

## 🎯 Use Cases

### Development
```bash
make debug   # Run with logging
```

### Testing
```bash
rm -rf ~/.mau-gui-poc  # Reset data
./mau-gui              # Fresh start
```

### Demo
```bash
./demo.sh    # Show feature overview
```

## 🛠️ Customization

### Change Dummy Data
Edit `main.go`:
```go
const (
    dummyEmail = "your@email.com"
    dummyName  = "Your Name"
)
```

### Change Data Directory
Edit `main.go`:
```go
dataDir: filepath.Join(os.Getenv("HOME"), ".your-dir"),
```

### Styling
GTK4/Adwaita uses CSS classes:
- `suggested-action` - Blue buttons
- `destructive-action` - Red buttons
- `title-1`, `title-2` - Headings
- `boxed-list` - Bordered lists

## 🐛 Troubleshooting

### "GTK4 not found"
```bash
sudo apt install libgtk-4-dev
pkg-config --modversion gtk4  # Verify
```

### "Libadwaita not found"
```bash
sudo apt install libadwaita-1-dev
pkg-config --modversion libadwaita-1  # Verify
```

### "go: command not found"
```bash
# Go should be installed at /usr/local/go/bin/go
export PATH=$PATH:/usr/local/go/bin
```

### Build errors with CGO
```bash
export CGO_ENABLED=1
go build -o mau-gui
```

## 📊 Metrics

- **Total Code:** 391 lines (main.go)
- **Documentation:** 489 lines (README + DESIGN)
- **Dependencies:** 3 main packages
- **Views:** 3 (Home, Friends, Settings)
- **Dialogs:** 2 (Add Friend, Info)
- **Build Time:** ~10 seconds (first build)
- **Binary Size:** ~15MB (with debug symbols)

## 🎓 Learning Resources

- **GTK4 Tutorial:** https://www.gtk.org/docs/getting-started/
- **Libadwaita Docs:** https://gnome.pages.gitlab.gnome.org/libadwaita/
- **gotk4 Examples:** https://github.com/diamondburned/gotk4-examples
- **Mau Spec:** `../README.md`

## 🤝 Next Steps

1. **Test the POC:**
   ```bash
   ./setup.sh && ./mau-gui
   ```

2. **Explore the code:**
   - `main.go` - Application logic
   - `DESIGN.md` - Implementation details

3. **Phase 2 - Add Persistence:**
   - Wire up `publishPost()` to write files
   - Implement `saveSettings()` persistence
   - Add friend keyring operations

4. **Phase 3 - Add Networking:**
   - Start HTTP server in background
   - Implement sync timer
   - Add peer discovery

5. **Phase 4 - Add Timeline:**
   - Create feed view
   - Load posts from friends
   - Display chronologically

---

**Status:** ✅ POC Ready  
**Next Phase:** Persistence Implementation  
**Timeline:** 2-3 days for full Phase 2
