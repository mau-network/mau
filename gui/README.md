# Mau GUI Architecture

A GTK4/Adwaita-based desktop client for the Mau P2P social network.

## Architecture Overview

This application follows **Clean Architecture** principles with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────────┐
│                     Presentation Layer                       │
│                  (GTK4 UI Components)                        │
│  cmd/mau-gui, internal/ui/{views,components,window,theme}   │
└─────────────────────────────────────────────────────────────┘
                              ▲
                              │ (uses)
┌─────────────────────────────────────────────────────────────┐
│                    Application Layer                         │
│              (Orchestration & Lifecycle)                     │
│                   internal/app/                              │
└─────────────────────────────────────────────────────────────┘
                              ▲
                              │ (uses)
┌─────────────────────────────────────────────────────────────┐
│                     Domain Layer                             │
│            (GTK-agnostic Business Logic)                     │
│    internal/domain/{post,config,account,server}             │
└─────────────────────────────────────────────────────────────┘
                              ▲
                              │ (implements)
┌─────────────────────────────────────────────────────────────┐
│                    Adapter Layer                             │
│            (External System Integration)                     │
│      internal/adapters/{storage,network,notification}        │
└─────────────────────────────────────────────────────────────┘
```

### Design Principles

1. **Dependency Inversion** - Core business logic depends on abstractions (interfaces), not implementations
2. **Single Responsibility** - Each package has ONE clear purpose
3. **Interface Segregation** - Small, focused interfaces
4. **Testability** - All layers mockable and testable in isolation
5. **GTK Independence** - Domain logic contains NO GTK dependencies

---

## Directory Structure

```
gui/
├── cmd/mau-gui/              # Application entry point
│   └── main.go               # Dependency injection & bootstrap
│
├── internal/                 # Private application code
│   ├── app/                  # Application orchestration
│   │   ├── app.go            # App lifecycle coordinator
│   │   └── services.go       # Service registry (future)
│   │
│   ├── domain/               # Business logic (GTK-agnostic)
│   │   ├── account/          # Account management
│   │   │   ├── interfaces.go # Store interface
│   │   │   └── manager.go    # Account operations
│   │   ├── config/           # Configuration
│   │   │   ├── config.go     # Config model
│   │   │   ├── interfaces.go # Store interface
│   │   │   └── manager.go    # Config operations
│   │   ├── post/             # Posts & social content
│   │   │   ├── cache.go      # LRU cache with TTL
│   │   │   ├── interfaces.go # Store & Cache interfaces
│   │   │   ├── manager.go    # Post operations
│   │   │   └── post.go       # Post model
│   │   └── server/           # P2P server lifecycle
│   │       ├── interfaces.go # Controller interface
│   │       └── manager.go    # Server operations
│   │
│   ├── ui/                   # GTK presentation layer
│   │   ├── components/       # Reusable widgets
│   │   │   ├── markdown_preview.go
│   │   │   ├── post_card.go
│   │   │   └── loading_spinner.go
│   │   ├── views/            # Full-screen views
│   │   │   ├── home/
│   │   │   │   ├── composer.go
│   │   │   │   ├── home.go
│   │   │   │   └── posts_list.go
│   │   │   ├── friends/
│   │   │   │   └── friends.go
│   │   │   ├── settings/
│   │   │   │   └── settings.go
│   │   │   └── timeline/
│   │   │       ├── display.go
│   │   │       ├── filters.go
│   │   │       └── timeline.go
│   │   ├── window/           # Main window
│   │   │   ├── header.go
│   │   │   └── window.go
│   │   └── theme/
│   │       └── theme.go      # Dark/light mode
│   │
│   └── adapters/             # External system integration
│       ├── storage/          # File system adapters
│       │   ├── account_store.go
│       │   ├── config_store.go
│       │   └── post_store.go
│       ├── network/          # Mau P2P server adapter
│       │   └── server_adapter.go
│       └── notification/     # Toast notifications
│           └── toast.go
│
├── pkg/                      # Public reusable packages
│   ├── retry/                # Exponential backoff retry
│   │   ├── retry.go
│   │   └── retry_test.go
│   └── markdown/             # Markdown rendering
│       └── renderer.go
│
├── docs/                     # Documentation
│   └── architecture.md       # Architecture decision records
│
├── go.mod
├── go.sum
└── README.md                 # This file
```

---

## Layer Responsibilities

### 1. `cmd/mau-gui` - Entry Point

**Purpose:** Bootstrap the application with dependency injection

**Responsibilities:**
- Create concrete implementations (stores, adapters)
- Wire up dependencies
- Start the application

**Example:**
```go
func main() {
    dataDir := getDataDir()
    
    // Create stores (adapters)
    configStore := storage.NewConfigStore(dataDir)
    accountStore := storage.NewAccountStore(dataDir)
    
    // Create domain managers
    configMgr := config.NewManager(configStore)
    accountMgr := account.NewManager(accountStore)
    
    // Initialize account
    if err := accountMgr.Init(); err != nil {
        log.Fatal(err)
    }
    
    // Create post infrastructure
    postStore := storage.NewPostStore(accountMgr.Account())
    postCache := post.NewCache(100, 30*time.Minute)
    postMgr := post.NewManager(postStore, postCache)
    
    // Create and run application
    app := app.New(app.Config{
        ConfigMgr:  configMgr,
        AccountMgr: accountMgr,
        PostMgr:    postMgr,
    })
    
    os.Exit(app.Run(os.Args))
}
```

**Key Point:** This is the ONLY place where concrete types are instantiated. Everything else depends on interfaces.

---

### 2. `internal/domain` - Business Logic

**Purpose:** Core application logic, independent of UI and infrastructure

**Characteristics:**
- ❌ NO GTK imports
- ❌ NO file system access
- ❌ NO network calls
- ✅ Pure Go logic
- ✅ Fully testable with mocks
- ✅ Reusable across different UIs

#### Example: Post Domain

**`post/post.go`** - Data model:
```go
type Post struct {
    Context     string    `json:"@context"`
    Type        string    `json:"@type"`
    Body        string    `json:"articleBody"`
    Published   time.Time `json:"datePublished"`
    Author      Author    `json:"author"`
    Tags        []string  `json:"keywords,omitempty"`
}

func New(body string, author Author, tags []string) Post {
    return Post{
        Context:   "https://schema.org",
        Type:      "SocialMediaPosting",
        Body:      body,
        Published: time.Now(),
        Author:    author,
        Tags:      tags,
    }
}
```

**`post/interfaces.go`** - Abstractions:
```go
type Store interface {
    Save(post Post) error
    Load(file *mau.File) (Post, error)
    List(fingerprint mau.Fingerprint, limit int) ([]*mau.File, error)
}

type Cache interface {
    Get(key string) (Post, bool)
    Set(key string, post Post)
    Clear()
}
```

**`post/manager.go`** - Business logic:
```go
type Manager struct {
    store Store
    cache Cache
}

func (m *Manager) Save(post Post) error {
    if err := m.store.Save(post); err != nil {
        return fmt.Errorf("failed to save post: %w", err)
    }
    return nil
}

func (m *Manager) Load(file *mau.File) (Post, error) {
    // Check cache first
    if cached, ok := m.cache.Get(file.Name()); ok {
        return cached, nil
    }
    
    // Load from storage
    post, err := m.store.Load(file)
    if err != nil {
        return Post{}, err
    }
    
    // Update cache
    m.cache.Set(file.Name(), post)
    return post, nil
}
```

**Why this matters:**
- Unit tests don't need GTK or file system
- Could build a CLI tool reusing this logic
- Business rules centralized and explicit

---

### 3. `internal/adapters` - External Integration

**Purpose:** Implement domain interfaces for real systems

**Responsibilities:**
- File system operations
- Network calls
- External library integration
- Format conversions

#### Example: Storage Adapter

**`storage/post_store.go`**:
```go
type PostStore struct {
    account *mau.Account
}

// Implements post.Store interface
func (s *PostStore) Save(p post.Post) error {
    jsonData, _ := p.ToJSON()
    
    keyring, _ := s.account.ListFriends()
    recipients := keyring.FriendsSet()
    filename := fmt.Sprintf("posts/post-%d.json", time.Now().UnixNano())
    
    reader := bytes.NewReader(jsonData)
    _, err := s.account.AddFile(reader, filename, recipients)
    return err
}

func (s *PostStore) Load(file *mau.File) (post.Post, error) {
    reader, _ := file.Reader(s.account)
    var p post.Post
    json.NewDecoder(reader).Decode(&p)
    return p, nil
}
```

**Benefits:**
- Domain layer doesn't know about `mau.Account` internals
- Could swap to SQLite/PostgreSQL by creating new adapter
- Adapter handles format conversions

---

### 4. `internal/ui` - Presentation Layer

**Purpose:** GTK widgets and user interaction

**Characteristics:**
- ✅ GTK imports allowed
- ✅ Widget construction
- ✅ Event handling
- ❌ Minimal business logic (delegate to domain)

#### Component Structure

**Reusable Components** (`internal/ui/components/`):
- Small, focused widgets
- Self-contained
- Accept data via constructors
- Example: `PostCard`, `MarkdownPreview`, `LoadingSpinner`

**Views** (`internal/ui/views/*/`):
- Full-screen views
- Compose multiple components
- Coordinate user interactions
- Delegate logic to domain managers

#### Example: Home View Composer

**`ui/views/home/composer.go`**:
```go
type Composer struct {
    postMgr *post.Manager
    
    // GTK widgets
    textView   *gtk.TextView
    sendButton *gtk.Button
    tagEntry   *gtk.Entry
}

func NewComposer(postMgr *post.Manager) *Composer {
    c := &Composer{postMgr: postMgr}
    c.buildUI()
    return c
}

func (c *Composer) buildUI() {
    c.textView = gtk.NewTextView()
    c.sendButton = gtk.NewButtonWithLabel("Post")
    c.sendButton.ConnectClicked(c.handleSend)
    // ... build widgets
}

func (c *Composer) handleSend() {
    content := c.getTextContent()
    tags := c.parseTags()
    
    // Delegate to domain layer - NO business logic here
    post := post.New(content, c.getAuthor(), tags)
    if err := c.postMgr.Save(post); err != nil {
        c.showError(err)
        return
    }
    
    c.clearForm()
}
```

**Key Principle:** Views are "dumb" - they build widgets and delegate logic.

---

### 5. `internal/app` - Orchestration

**Purpose:** Lightweight coordinator that wires everything together

**Responsibilities:**
- Initialize domain managers
- Create UI with injected dependencies
- Coordinate application lifecycle
- Handle cross-cutting concerns (startup, shutdown)

**Anti-pattern:** God object with all methods  
**Pattern:** Thin coordinator (~100-150 lines)

#### Example

**`app/app.go`**:
```go
type App struct {
    gtkApp *adw.Application
    
    // Domain services (injected)
    configMgr  *config.Manager
    accountMgr *account.Manager
    postMgr    *post.Manager
    
    // UI
    mainWindow *window.Window
}

type Config struct {
    ConfigMgr  *config.Manager
    AccountMgr *account.Manager
    PostMgr    *post.Manager
}

func New(cfg Config) *App {
    gtkApp := adw.NewApplication("com.mau.gui", 0)
    
    app := &App{
        gtkApp:     gtkApp,
        configMgr:  cfg.ConfigMgr,
        accountMgr: cfg.AccountMgr,
        postMgr:    cfg.PostMgr,
    }
    
    gtkApp.ConnectActivate(app.activate)
    return app
}

func (a *App) activate() {
    // Build UI with injected dependencies
    a.mainWindow = window.New(window.Config{
        App:        a.gtkApp,
        ConfigMgr:  a.configMgr,
        AccountMgr: a.accountMgr,
        PostMgr:    a.postMgr,
    })
    
    a.mainWindow.Show()
}

func (a *App) Run(args []string) int {
    return a.gtkApp.Run(args)
}
```

---

## Dependency Flow

```
main.go (cmd/mau-gui)
  ├─> Creates: ConfigStore, AccountStore (adapters/storage)
  ├─> Creates: ConfigManager, AccountManager (domain)
  ├─> Creates: PostStore (adapters/storage)
  ├─> Creates: PostCache, PostManager (domain)
  └─> Creates: App (app layer)
       ├─> Creates: Window (ui/window)
       │    └─> Creates: Views (ui/views/*)
       │         └─> Uses: Domain Managers (via interfaces)
       └─> Runs: GTK Application
```

**Key Insight:** Dependencies point INWARD - outer layers depend on inner layers, never the reverse.

---

## Testing Strategy

### Domain Layer Tests

**Pure unit tests** with mock implementations:

```go
// post/manager_test.go
type mockStore struct {
    savedPost Post
}

func (m *mockStore) Save(p Post) error {
    m.savedPost = p
    return nil
}

func TestManager_Save(t *testing.T) {
    store := &mockStore{}
    cache := post.NewCache(10, time.Minute)
    mgr := post.NewManager(store, cache)
    
    post := post.New("Hello", post.Author{Name: "Alice"}, nil)
    err := mgr.Save(post)
    
    assert.NoError(t, err)
    assert.Equal(t, "Hello", store.savedPost.Body)
}
```

**Benefits:**
- No GTK dependencies
- Fast (no I/O)
- Easy to test edge cases

### Adapter Layer Tests

**Integration tests** with real dependencies:

```go
// adapters/storage/post_store_test.go
func TestPostStore_SaveAndLoad(t *testing.T) {
    tmpDir := t.TempDir()
    account := createTestAccount(t, tmpDir)
    store := storage.NewPostStore(account)
    
    original := post.New("Test content", post.Author{Name: "Bob"}, nil)
    err := store.Save(original)
    assert.NoError(t, err)
    
    // Verify it's persisted
    files, _ := store.List(account.Fingerprint(), 10)
    assert.Len(t, files, 1)
    
    loaded, _ := store.Load(files[0])
    assert.Equal(t, "Test content", loaded.Body)
}
```

### UI Layer Tests

**Interaction tests** with mocked domain managers:

```go
// ui/views/home/composer_test.go
type mockPostManager struct {
    lastSaved Post
}

func (m *mockPostManager) Save(p Post) error {
    m.lastSaved = p
    return nil
}

func TestComposer_Send(t *testing.T) {
    mockMgr := &mockPostManager{}
    composer := NewComposer(mockMgr)
    
    // Simulate user input
    composer.textView.SetText("My post")
    composer.tagEntry.SetText("golang,testing")
    composer.sendButton.Emit("clicked")
    
    assert.Equal(t, "My post", mockMgr.lastSaved.Body)
    assert.Equal(t, []string{"golang", "testing"}, mockMgr.lastSaved.Tags)
}
```

### End-to-End Tests

Keep existing `integration_test.go` for full workflow testing.

---

## Common Patterns

### 1. Dependency Injection via Constructors

```go
// BAD: Global state
var globalConfig *config.Manager

// GOOD: Injected dependency
type HomeView struct {
    configMgr *config.Manager
}

func NewHomeView(configMgr *config.Manager) *HomeView {
    return &HomeView{configMgr: configMgr}
}
```

### 2. Interface-Based Design

```go
// Define interface in domain layer
type PostStore interface {
    Save(Post) error
}

// Implement in adapter layer
type FilePostStore struct { ... }
func (s *FilePostStore) Save(p Post) error { ... }

// Use via interface in domain layer
type Manager struct {
    store PostStore  // Not *FilePostStore!
}
```

### 3. Error Wrapping

```go
// Add context at each layer
func (m *Manager) Save(post Post) error {
    if err := m.store.Save(post); err != nil {
        return fmt.Errorf("failed to save post: %w", err)
    }
    return nil
}
```

### 4. Configuration via Structs

```go
// Avoid long parameter lists
func NewWindow(cfg window.Config) *Window {
    return &Window{
        app:        cfg.App,
        configMgr:  cfg.ConfigMgr,
        accountMgr: cfg.AccountMgr,
        postMgr:    cfg.PostMgr,
    }
}
```

---

## Migration Status

### ✅ Completed

- [x] Directory structure created
- [x] Post domain extracted
- [x] Config domain extracted
- [x] Account domain extracted
- [x] Storage adapters implemented
- [x] Retry package moved to `pkg/`

### 🚧 In Progress

- [ ] Server domain (move from `app_server.go`)
- [ ] UI views reorganization
- [ ] Component extraction
- [ ] App layer slim down
- [ ] Entry point (`cmd/mau-gui/main.go`)

### 📋 Planned

- [ ] Notification adapter
- [ ] Network adapter (server lifecycle)
- [ ] Markdown package extraction
- [ ] Theme management
- [ ] Complete test coverage
- [ ] Remove old files from root `gui/`

---

## Design Decisions

### Why Clean Architecture?

**Problems with previous structure:**
- `MauApp` had 33 methods (god object)
- All code in `main` package (no reusability)
- GTK code mixed with business logic
- Hard to test in isolation

**Benefits of new structure:**
- Domain logic reusable (CLI, web, tests)
- Each package has single responsibility
- Easy to mock and test
- Clear dependency flow

### Why Not MVC?

MVC doesn't map cleanly to event-driven GTK applications - controllers become god objects. Clean Architecture provides better separation.

### Why Internal Packages?

- Enforces API boundaries
- Prevents external code from importing internals
- Clear public API in `pkg/`

---

## Building & Running

```bash
# From gui/ directory
go build -o mau-gui ./cmd/mau-gui

# Run
./mau-gui

# Run tests
go test ./...

# Run only domain tests (fast, no GTK)
go test ./internal/domain/...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## Adding New Features

### Example: Adding Comment Support

1. **Domain Layer** (`internal/domain/comment/`):
   ```go
   // comment.go - model
   type Comment struct {
       PostID  string
       Author  Author
       Body    string
       Created time.Time
   }
   
   // interfaces.go
   type Store interface {
       Save(Comment) error
       ListForPost(postID string) ([]Comment, error)
   }
   
   // manager.go
   type Manager struct {
       store Store
   }
   ```

2. **Adapter** (`internal/adapters/storage/comment_store.go`):
   ```go
   type CommentStore struct {
       account *mau.Account
   }
   
   func (s *CommentStore) Save(c comment.Comment) error {
       // Implementation using Mau files
   }
   ```

3. **UI** (`internal/ui/components/comment_list.go`):
   ```go
   type CommentList struct {
       commentMgr *comment.Manager
       listBox    *gtk.ListBox
   }
   
   func (cl *CommentList) Refresh(postID string) {
       comments, _ := cl.commentMgr.ListForPost(postID)
       // Build UI from comments
   }
   ```

4. **Wire Up** (`cmd/mau-gui/main.go`):
   ```go
   commentStore := storage.NewCommentStore(accountMgr.Account())
   commentMgr := comment.NewManager(commentStore)
   
   app := app.New(app.Config{
       // ... existing
       CommentMgr: commentMgr,
   })
   ```

---

## Resources

- **Mau Protocol:** https://github.com/mau-network/mau
- **GTK4 Go Bindings:** https://github.com/diamondburned/gotk4
- **Clean Architecture:** Robert C. Martin (Uncle Bob)
- **Dependency Injection:** https://go.dev/blog/wire

---

## Contributing

When adding new code:

1. **Identify the layer** - Where does this belong?
   - Pure logic → `internal/domain/`
   - External integration → `internal/adapters/`
   - UI widget → `internal/ui/`
   - Orchestration → `internal/app/`

2. **Define interfaces first** - What abstraction is needed?

3. **Write tests** - Especially for domain layer

4. **Keep layers pure** - No GTK in domain, no business logic in UI

5. **Document decisions** - Update this README or `docs/architecture.md`

---

## FAQ

**Q: Why can't I just add a method to `MauApp`?**  
A: That recreates the god object anti-pattern. Ask: which manager should own this logic?

**Q: Where do I put GTK utility functions?**  
A: If reusable → `internal/ui/components/`. If view-specific → that view's package.

**Q: Can domain code import `mau` library?**  
A: Yes, for types (`mau.Fingerprint`, `mau.File`). No, for operations - those go in adapters.

**Q: How do I test GTK widgets?**  
A: Minimize GTK logic. Test domain managers with mocks. For widgets, integration tests are OK.

**Q: What if I need to share state between views?**  
A: Use domain managers as the source of truth. Views fetch latest state when shown.

---

**Last Updated:** 2026-02-24  
**Maintainer:** Emad Elsaid (emad.elsaid.hamed@gmail.com)  
**Status:** Architecture refactoring in progress
