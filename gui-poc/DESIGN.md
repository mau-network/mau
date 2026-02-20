# Mau GUI POC - UI Design & Implementation Notes

## Application Overview

```
┌─────────────────────────────────────────────────────────┐
│ 🛸 Mau - P2P Social Network                             │
│ ┌──────┬──────────┬──────────┐                          │
│ │ Home │ Friends  │ Settings │  ← View Switcher         │
│ └──────┴──────────┴──────────┘                          │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  [Current View Content]                                  │
│                                                          │
│                                                          │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

## Home View

```
┌─────────────────────────────────────────────────────────┐
│                    Welcome, Demo User!                   │
│                                                          │
│  ╔══════════════════════════════════════════════════╗  │
│  ║ Create a Post                                     ║  │
│  ║ Share something with your network                ║  │
│  ╠════════════════════════════════════════════════╦══╣  │
│  ║                                                 ║  ║  │
│  ║  What's on your mind?                           ║  ║  │
│  ║                                                 ║  ║  │
│  ║  [Multi-line text entry area]                   ║  ║  │
│  ║                                                 ║  ║  │
│  ║                                                 ║  ║  │
│  ╚════════════════════════════════════════════════╩══╝  │
│                                                          │
│                                      ┌──────────────┐   │
│                                      │  📤 Publish   │   │
│                                      └──────────────┘   │
└─────────────────────────────────────────────────────────┘
```

**Features:**
- Welcome message with user name
- Large text area for composing posts
- Publish button (suggested-action style - blue/green)
- Posts are schema.org SocialMediaPosting type

**Future Enhancements:**
- Attach files/images
- Privacy selector (public/friends/specific)
- Post preview
- Character/word count
- Draft saving

## Friends View

```
┌─────────────────────────────────────────────────────────┐
│  ╔══════════════════════════════════════════════════╗  │
│  ║ Friends                                           ║  │
│  ║ Manage your network                              ║  │
│  ╠════════════════════════════════════════════════╦══╣  │
│  ║ 👤 Alice                                        ║  ║  │
│  ║    alice@example.com                            ║  ║  │
│  ║    FPR: 5D000B2F2C040A1675B49D7F...            ║  ║  │
│  ╠════════════════════════════════════════════════╬══╣  │
│  ║ 👤 Bob                                          ║  ║  │
│  ║    bob@example.com                              ║  ║  │
│  ║    FPR: 3A112C4D5E060F8765C38E6D...            ║  ║  │
│  ╠════════════════════════════════════════════════╬══╣  │
│  ║ 👤 Charlie                                      ║  ║  │
│  ║    charlie@example.com                          ║  ║  │
│  ║    FPR: 7B223D5E6F071A9876D49F7E...            ║  ║  │
│  ╚════════════════════════════════════════════════╩══╝  │
│                                                          │
│                                      ┌──────────────┐   │
│                                      │ ➕ Add Friend │   │
│                                      └──────────────┘   │
└─────────────────────────────────────────────────────────┘
```

**Features:**
- List of friends with name, email, fingerprint
- Scrollable list
- Add friend button

**Add Friend Dialog:**
```
┌─────────────────────────────────────┐
│  Add Friend                          │
├─────────────────────────────────────┤
│  Enter friend's fingerprint or      │
│  address to add them to your        │
│  network.                            │
│                                      │
│  Fingerprint:                        │
│  ┌─────────────────────────────────┐│
│  │ 5D000B2F2C040A1675B49D7F...     ││
│  └─────────────────────────────────┘│
│                                      │
│           ┌────────┬────────┐       │
│           │ Cancel │  Add   │       │
│           └────────┴────────┘       │
└─────────────────────────────────────┘
```

**Future Enhancements:**
- Friend groups/categories
- Search/filter friends
- Friend status (online/offline)
- Last sync time
- Remove friend option
- Import from QR code
- Export own fingerprint as QR

## Settings View

```
┌─────────────────────────────────────────────────────────┐
│  ╔══════════════════════════════════════════════════╗  │
│  ║ Account Settings                                  ║  │
│  ║ Update your profile information                  ║  │
│  ╠════════════════════════════════════════════════╦══╣  │
│  ║ Name                    ┌────────────────────┐  ║  ║  │
│  ║                         │ Demo User          │  ║  ║  │
│  ║                         └────────────────────┘  ║  ║  │
│  ╠════════════════════════════════════════════════╬══╣  │
│  ║ Email                   ┌────────────────────┐  ║  ║  │
│  ║                         │ demo@mau.network   │  ║  ║  │
│  ║                         └────────────────────┘  ║  ║  │
│  ╚════════════════════════════════════════════════╩══╝  │
│                                                          │
│  ╔══════════════════════════════════════════════════╗  │
│  ║ Account Information                               ║  │
│  ╠════════════════════════════════════════════════╦══╣  │
│  ║ Fingerprint                                     ║  ║  │
│  ║ 5D000B2F2C040A1675B49D7F0C7CB7DC36999D56       ║  ║  │
│  ╚════════════════════════════════════════════════╩══╝  │
│                                                          │
│                                      ┌──────────────┐   │
│                                      │ Save Changes │   │
│                                      └──────────────┘   │
└─────────────────────────────────────────────────────────┘
```

**Features:**
- Editable name and email fields
- Read-only fingerprint display
- Save button

**Future Enhancements:**
- Change password
- Export/backup account
- Import account from backup
- Avatar/profile picture upload
- Data directory path
- Sync settings (interval, auto-sync)
- Server settings (port, TLS)
- Privacy settings
- Storage stats (disk usage)
- Delete account (with confirmation)

## Technical Implementation Details

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

### File Operations

**Creating a Post:**
1. User types content in TextView
2. Click "Publish"
3. Create schema.org SocialMediaPosting JSON object
4. Use `mau.Account.Sign()` to sign the JSON
5. Optionally encrypt with `mau.Account.Encrypt()`
6. Write to `~/.mau-gui-poc/<user-fpr>/<post-id>.pgp`
7. Create version directory if needed
8. Update UI with confirmation

**Adding a Friend:**
1. User enters friend fingerprint
2. Fetch friend's public key (via HTTP, mDNS, or manual import)
3. Verify key fingerprint
4. Use `mau.Account.AddFriend()` to add to keyring
5. Save key to `~/.mau-gui-poc/.mau/<group>/<friend-fpr>.pgp`
6. Create friend directory `~/.mau-gui-poc/<friend-fpr>/`
7. Update friends list UI

**Syncing:**
1. Background goroutine/timer
2. For each friend in keyring:
   - Discover friend's address (mDNS, Kademlia)
   - HTTP GET `/<friend-fpr>` with `If-Modified-Since`
   - Download new/modified files
   - Verify signatures
   - Decrypt if encrypted for us
   - Save to `~/.mau-gui-poc/<friend-fpr>/`
3. Update UI with new content notifications

### Mau Library Integration

```go
// Initialize account
account, err := mau.NewAccount(dataDir, name, email, password)

// Create and sign content
content := map[string]interface{}{
    "@context": "https://schema.org",
    "@type": "SocialMediaPosting",
    "headline": "My Post",
    "articleBody": "Content here",
}
jsonData, _ := json.Marshal(content)
signedData, _ := account.Sign(jsonData)

// Encrypt for recipient
encryptedData, _ := account.Encrypt(signedData, recipientFingerprint)

// Write to file
ioutil.WriteFile(filepath, encryptedData, 0600)

// Add friend
friendKey := /* fetch from network or import */
account.AddFriend(friendKey, "friends")

// Start server
server := mau.NewServer(account, ":8080")
go server.Start()

// Sync with friend
client := mau.NewClient(account)
client.Sync(friendFingerprint, friendAddress)
```

### GTK4/Adwaita Patterns

**Preferences Groups:**
- Use `adw.PreferencesGroup` for sections
- Use `adw.ActionRow` for list items
- Consistent spacing and margins

**Dialogs:**
- Use `adw.MessageDialog` for confirmations
- Use `adw.AlertDialog` for important alerts
- Always set response appearance (suggested, destructive)

**Lists:**
- Use `gtk.ListBox` with `boxed-list` CSS class
- Wrap in `gtk.ScrolledWindow` for long lists
- Use `adw.ActionRow` for items with title/subtitle

**Buttons:**
- `suggested-action` for primary actions (blue)
- `destructive-action` for dangerous actions (red)
- `flat` for secondary actions

## Build System

**Dependencies:**
- GTK4 4.x
- Libadwaita 1.x
- Go 1.26+
- pkg-config

**Build Tags:**
None required for basic build

**CGO Requirements:**
- Yes (required for GTK bindings)
- Set `CGO_ENABLED=1`

## Future Full Implementation Roadmap

1. **Phase 1: Core Functionality (This POC)** ✅
   - Account creation
   - Basic UI with 3 views
   - Post composition UI
   - Friends list UI
   - Settings UI

2. **Phase 2: Persistence**
   - Implement actual file writing
   - PGP signing/encryption integration
   - Settings persistence
   - Friends keyring management

3. **Phase 3: Networking**
   - Integrate Mau HTTP server
   - Implement sync daemon
   - mDNS discovery
   - Kademlia routing

4. **Phase 4: Feed**
   - Display posts from friends
   - Chronological timeline
   - Post filtering
   - Search

5. **Phase 5: Rich Content**
   - Image attachments
   - File sharing
   - Different post types (Recipe, Article, etc.)
   - Markdown support

6. **Phase 6: Polish**
   - Notifications
   - Dark mode
   - Keyboard shortcuts
   - Accessibility
   - Localization

## Testing Strategy

**Unit Tests:**
- UI component initialization
- Data model operations
- Mau library integration

**Integration Tests:**
- Full application startup
- Account creation flow
- Post publishing flow
- Friend management flow

**Manual Testing:**
- Run on different GNOME versions
- Test with real PGP keys
- Multi-account scenarios
- Network sync testing

## Known Limitations

1. **Single Account**: Currently only supports one account per data directory
2. **No Offline Mode**: Requires network for sync (future: queue operations)
3. **No Background Sync**: Must keep app open (future: systemd service)
4. **Limited Error Handling**: POC has minimal error recovery
5. **No Undo**: Operations are immediate (future: undo stack)
6. **English Only**: No i18n yet (future: gettext integration)

## Performance Considerations

- Use `gtk.ListBox` instead of `gtk.ListView` for small lists (<1000 items)
- Lazy load posts in timeline
- Cache rendered content
- Debounce search/filter operations
- Use background threads for crypto operations
- Limit memory usage with pagination

## Security Notes

- Never store password in memory longer than necessary
- Use secure memory for PGP operations
- Clear clipboard after copy operations
- Warn before following untrusted links
- Validate all friend fingerprints
- Check file permissions (0600 for private data)
- Verify signatures before displaying content
