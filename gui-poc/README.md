# Mau GUI POC - GTK4/Adwaita Client

A proof-of-concept GUI client for Mau P2P social network using Go and GTK4/Libadwaita.

## Features

- ✅ **Auto Account Creation**: Creates account with dummy data on first run
- ✅ **Home View**: Compose and publish posts (SocialMediaPosting schema.org type)
- ✅ **Friends Management**: View and add friends to your network
- ✅ **Settings**: Update account name and email, view fingerprint
- ✅ **Modern UI**: Uses Libadwaita for GNOME-style modern interface

## Prerequisites

### System Dependencies

You need GTK4 and Libadwaita installed:

**Ubuntu/Debian:**
```bash
sudo apt install libgtk-4-dev libadwaita-1-dev
```

**Fedora:**
```bash
sudo dnf install gtk4-devel libadwaita-devel
```

**Arch Linux:**
```bash
sudo pacman -S gtk4 libadwaita
```

**macOS (Homebrew):**
```bash
brew install gtk4 libadwaita
```

### Go Dependencies

Go 1.26+ is required (already installed on this system).

## Building

```bash
cd gui-poc
go mod download
go build -o mau-gui
```

## Running

```bash
./mau-gui
```

On first run, it will:
1. Create `~/.mau-gui-poc/` directory
2. Generate a new Mau account with dummy data:
   - Name: "Demo User"
   - Email: "demo@mau.network"
3. Display the main window with three views

## Project Structure

```
gui-poc/
├── main.go          # Main application code
├── go.mod           # Go module definition
└── README.md        # This file
```

## Usage

### Home View
- Type your post content in the text area
- Click "Publish" to create a post (POC: logs to console)
- Uses schema.org SocialMediaPosting format

### Friends View
- View your friends list (currently placeholder)
- Click "Add Friend" to add a new friend (POC: shows dialog)

### Settings View
- Update your name and email
- View your account fingerprint
- Click "Save Changes" (POC: logs to console)

## POC Limitations

This is a proof-of-concept with the following limitations:

1. **No Persistence**: Posts and friend changes are logged but not actually saved
2. **No Crypto**: PGP signing/encryption not implemented in UI
3. **No Sync**: Peer-to-peer syncing not implemented
4. **Dummy Data**: Account created with hardcoded demo values
5. **No Friend Input**: Add friend dialog doesn't capture fingerprint input

## Next Steps for Full Implementation

1. Integrate Mau's PGP operations for signing/encrypting posts
2. Implement file writing for posts to `~/.mau-gui-poc/<fingerprint>/`
3. Add friend management with keyring operations
4. Implement sync daemon integration
5. Add post timeline/feed view
6. Add friend's post viewing
7. Implement settings persistence
8. Add avatar/profile picture support
9. Add notifications for new posts
10. Integrate with Mau server/peer discovery

## Development Notes

- Data directory: `~/.mau-gui-poc/`
- Account file: `~/.mau-gui-poc/.mau/account.pgp`
- Uses Mau library from parent directory via `replace` directive
- Modern Adwaita widgets for native GNOME look and feel

## License

Same as Mau project (GPLv3)
