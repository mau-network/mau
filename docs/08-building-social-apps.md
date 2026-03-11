# Building Social Applications with Mau

This guide provides practical patterns and examples for building decentralized social applications with Mau. Learn how to structure your application, handle common scenarios, and leverage Mau's features effectively.

## Table of Contents

- [Application Architecture](#application-architecture)
- [Common Patterns](#common-patterns)
- [Content Types](#content-types)
- [User Interactions](#user-interactions)
- [Data Synchronization](#data-synchronization)
- [Example Applications](#example-applications)
- [Best Practices](#best-practices)

## Application Architecture

### Basic Structure

A typical Mau social application consists of:

```
your-app/
├── main.go              # Application entry point
├── client/              # Mau client initialization
├── handlers/            # HTTP request handlers
├── ui/                  # Frontend (web, desktop, mobile)
└── storage/             # Local data cache and indexing
```

### Initialization Pattern

```go
package main

import (
    "log"
    "os"
    "path/filepath"
    
    "github.com/mau-network/mau"
)

type App struct {
    client *mau.Client
    config Config
}

func NewApp(dataDir string) (*App, error) {
    // Ensure data directory exists
    if err := os.MkdirAll(dataDir, 0700); err != nil {
        return nil, err
    }
    
    // Initialize Mau client
    client, err := mau.NewClient(dataDir)
    if err != nil {
        return nil, err
    }
    
    return &App{
        client: client,
        config: LoadConfig(),
    }, nil
}

func (a *App) Start() error {
    // Start HTTP server for peer communication
    go a.client.StartServer(":8080")
    
    // Start DHT for peer discovery
    go a.client.StartDHT()
    
    // Start syncing with known peers
    go a.client.StartSync()
    
    log.Println("Application started successfully")
    return nil
}
```

## Common Patterns

### 1. Post-and-Sync Pattern

The most common pattern: create content and automatically share it with peers.

```go
func (a *App) CreatePost(author, content string) error {
    post := map[string]interface{}{
        "@context": "https://schema.org",
        "@type":    "SocialMediaPosting",
        "headline": content,
        "author": map[string]interface{}{
            "@type": "Person",
            "name":  author,
        },
        "datePublished": time.Now().Format(time.RFC3339),
    }
    
    // Generate unique filename
    filename := fmt.Sprintf("post-%d.json", time.Now().Unix())
    
    // Save encrypted and signed
    if err := a.client.Save(filename, post); err != nil {
        return fmt.Errorf("save post: %w", err)
    }
    
    // Automatically synced with peers
    return nil
}
```

### 2. Follow-and-Fetch Pattern

Follow users and fetch their content periodically.

```go
func (a *App) FollowUser(fingerprint string) error {
    // Add to follow list
    if err := a.client.Follow(fingerprint); err != nil {
        return err
    }
    
    // Fetch their content immediately
    return a.fetchUserContent(fingerprint)
}

func (a *App) fetchUserContent(fingerprint string) error {
    // Discover peer address
    addr, err := a.client.FindPeer(fingerprint)
    if err != nil {
        return fmt.Errorf("find peer: %w", err)
    }
    
    // Fetch their public files
    files, err := a.client.FetchFiles(addr, fingerprint)
    if err != nil {
        return fmt.Errorf("fetch files: %w", err)
    }
    
    // Process each file
    for _, file := range files {
        if err := a.processFile(file); err != nil {
            log.Printf("process file %s: %v", file.Name, err)
        }
    }
    
    return nil
}
```

### 3. Timeline Aggregation Pattern

Combine content from multiple sources into a unified timeline.

```go
type TimelineItem struct {
    Content   map[string]interface{}
    Author    string
    Timestamp time.Time
    Verified  bool
}

func (a *App) BuildTimeline(limit int) ([]TimelineItem, error) {
    var items []TimelineItem
    
    // Get followed users
    following := a.client.Following()
    
    for _, fingerprint := range following {
        // Fetch recent posts from this user
        posts, err := a.fetchRecentPosts(fingerprint, 10)
        if err != nil {
            log.Printf("fetch posts for %s: %v", fingerprint, err)
            continue
        }
        
        for _, post := range posts {
            items = append(items, TimelineItem{
                Content:   post,
                Author:    fingerprint,
                Timestamp: extractTimestamp(post),
                Verified:  true, // Mau verifies signatures
            })
        }
    }
    
    // Sort by timestamp (newest first)
    sort.Slice(items, func(i, j int) bool {
        return items[i].Timestamp.After(items[j].Timestamp)
    })
    
    // Limit results
    if len(items) > limit {
        items = items[:limit]
    }
    
    return items, nil
}
```

### 4. Private Messaging Pattern

Send encrypted direct messages between users.

```go
func (a *App) SendPrivateMessage(recipientFingerprint, message string) error {
    dm := map[string]interface{}{
        "@context": "https://schema.org",
        "@type":    "Message",
        "text":     message,
        "sender": map[string]interface{}{
            "@type": "Person",
            "identifier": a.client.Fingerprint(),
        },
        "recipient": map[string]interface{}{
            "@type": "Person",
            "identifier": recipientFingerprint,
        },
        "dateSent": time.Now().Format(time.RFC3339),
    }
    
    filename := fmt.Sprintf("dm-%s-%d.json", 
        recipientFingerprint[:8], 
        time.Now().Unix())
    
    // Encrypt specifically for recipient
    if err := a.client.SaveEncrypted(filename, dm, recipientFingerprint); err != nil {
        return fmt.Errorf("save message: %w", err)
    }
    
    // Send directly to recipient's server
    if err := a.client.SendTo(recipientFingerprint, filename); err != nil {
        return fmt.Errorf("send message: %w", err)
    }
    
    return nil
}

func (a *App) ReadPrivateMessages() ([]map[string]interface{}, error) {
    files, err := a.client.List("dm-")
    if err != nil {
        return nil, err
    }
    
    var messages []map[string]interface{}
    for _, filename := range files {
        var msg map[string]interface{}
        if err := a.client.Load(filename, &msg); err != nil {
            log.Printf("load message %s: %v", filename, err)
            continue
        }
        messages = append(messages, msg)
    }
    
    return messages, nil
}
```

## Content Types

### Schema.org Types for Social Apps

Mau uses [Schema.org](https://schema.org) vocabulary for structured content. Here are common types:

#### Social Media Post

```go
post := map[string]interface{}{
    "@context": "https://schema.org",
    "@type":    "SocialMediaPosting",
    "headline": "Post title",
    "articleBody": "Full post content with markdown support",
    "author": map[string]interface{}{
        "@type": "Person",
        "name":  "Alice",
        "identifier": fingerprint,
    },
    "datePublished": "2026-03-11T02:00:00Z",
    "keywords": []string{"mau", "decentralized", "social"},
}
```

#### Comment/Reply

```go
comment := map[string]interface{}{
    "@context": "https://schema.org",
    "@type":    "Comment",
    "text":     "Great post!",
    "author": map[string]interface{}{
        "@type": "Person",
        "name":  "Bob",
    },
    "dateCreated": "2026-03-11T02:05:00Z",
    "parentItem": map[string]interface{}{
        "@type": "SocialMediaPosting",
        "identifier": "original-post-id",
    },
}
```

#### Photo/Image Post

```go
photo := map[string]interface{}{
    "@context": "https://schema.org",
    "@type":    "ImageObject",
    "name":     "Sunset in Berlin",
    "contentUrl": "photos/sunset-2026-03-11.jpg",
    "encodingFormat": "image/jpeg",
    "author": map[string]interface{}{
        "@type": "Person",
        "name":  "Alice",
    },
    "datePublished": "2026-03-11T18:30:00Z",
    "caption": "Beautiful sunset from my balcony",
}
```

#### User Profile

```go
profile := map[string]interface{}{
    "@context": "https://schema.org",
    "@type":    "Person",
    "name":     "Alice",
    "identifier": fingerprint,
    "description": "Software engineer interested in decentralization",
    "image": "avatar.jpg",
    "url": "https://alice.example.com",
    "sameAs": []string{
        "https://github.com/alice",
        "https://twitter.com/alice",
    },
}
```

#### Event

```go
event := map[string]interface{}{
    "@context": "https://schema.org",
    "@type":    "Event",
    "name":     "Mau Developer Meetup",
    "description": "Monthly gathering of Mau developers",
    "startDate": "2026-03-15T18:00:00Z",
    "endDate": "2026-03-15T20:00:00Z",
    "location": map[string]interface{}{
        "@type": "Place",
        "name":  "Berlin Tech Hub",
        "address": map[string]interface{}{
            "@type": "PostalAddress",
            "streetAddress": "Tech Street 123",
            "addressLocality": "Berlin",
            "addressCountry": "DE",
        },
    },
    "organizer": map[string]interface{}{
        "@type": "Person",
        "name":  "Alice",
    },
}
```

## User Interactions

### Likes and Reactions

```go
func (a *App) LikePost(postID, authorFingerprint string) error {
    like := map[string]interface{}{
        "@context": "https://schema.org",
        "@type":    "LikeAction",
        "agent": map[string]interface{}{
            "@type": "Person",
            "identifier": a.client.Fingerprint(),
        },
        "object": map[string]interface{}{
            "@type": "SocialMediaPosting",
            "identifier": postID,
            "author": map[string]interface{}{
                "@type": "Person",
                "identifier": authorFingerprint,
            },
        },
        "startTime": time.Now().Format(time.RFC3339),
    }
    
    filename := fmt.Sprintf("like-%s-%d.json", postID, time.Now().Unix())
    return a.client.Save(filename, like)
}
```

### Shares/Reposts

```go
func (a *App) SharePost(originalPost map[string]interface{}) error {
    share := map[string]interface{}{
        "@context": "https://schema.org",
        "@type":    "ShareAction",
        "agent": map[string]interface{}{
            "@type": "Person",
            "identifier": a.client.Fingerprint(),
        },
        "object": originalPost,
        "startTime": time.Now().Format(time.RFC3339),
    }
    
    filename := fmt.Sprintf("share-%d.json", time.Now().Unix())
    return a.client.Save(filename, share)
}
```

### Following/Unfollowing

```go
func (a *App) ToggleFollow(fingerprint string) error {
    following := a.client.Following()
    
    // Check if already following
    for _, fp := range following {
        if fp == fingerprint {
            return a.client.Unfollow(fingerprint)
        }
    }
    
    return a.client.Follow(fingerprint)
}
```

## Data Synchronization

### Periodic Sync

```go
func (a *App) StartPeriodicSync(interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    
    for range ticker.C {
        if err := a.syncAll(); err != nil {
            log.Printf("sync error: %v", err)
        }
    }
}

func (a *App) syncAll() error {
    following := a.client.Following()
    
    for _, fingerprint := range following {
        if err := a.fetchUserContent(fingerprint); err != nil {
            log.Printf("sync %s: %v", fingerprint, err)
            // Continue with other users
        }
    }
    
    return nil
}
```

### Real-time Notifications

```go
func (a *App) WatchForNewContent(callback func(filename string)) error {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return err
    }
    defer watcher.Close()
    
    // Watch the Mau data directory
    if err := watcher.Add(a.client.DataDir()); err != nil {
        return err
    }
    
    for {
        select {
        case event := <-watcher.Events:
            if event.Op&fsnotify.Create == fsnotify.Create {
                callback(event.Name)
            }
        case err := <-watcher.Errors:
            log.Printf("watcher error: %v", err)
        }
    }
}
```

### Selective Sync

```go
func (a *App) SyncRecentContent(fingerprint string, since time.Time) error {
    files, err := a.client.FetchFiles(fingerprint)
    if err != nil {
        return err
    }
    
    for _, file := range files {
        // Skip old files
        if file.ModTime.Before(since) {
            continue
        }
        
        if err := a.processFile(file); err != nil {
            log.Printf("process file %s: %v", file.Name, err)
        }
    }
    
    return nil
}
```

## Example Applications

### 1. Microblogging Platform (Twitter-like)

```go
package main

import (
    "github.com/mau-network/mau"
)

type MicroBlog struct {
    client *mau.Client
}

func (m *MicroBlog) Post(text string) error {
    post := map[string]interface{}{
        "@context": "https://schema.org",
        "@type":    "SocialMediaPosting",
        "headline": text,
        "author": map[string]interface{}{
            "@type": "Person",
            "identifier": m.client.Fingerprint(),
        },
        "datePublished": time.Now().Format(time.RFC3339),
    }
    
    filename := fmt.Sprintf("post-%d.json", time.Now().Unix())
    return m.client.Save(filename, post)
}

func (m *MicroBlog) Timeline(limit int) ([]map[string]interface{}, error) {
    // Aggregate posts from followed users
    // (implementation similar to BuildTimeline above)
}

func (m *MicroBlog) Search(query string) ([]map[string]interface{}, error) {
    // Search across cached posts
    // (requires local indexing)
}
```

### 2. Photo Sharing App (Instagram-like)

```go
type PhotoApp struct {
    client *mau.Client
}

func (p *PhotoApp) UploadPhoto(filepath, caption string) error {
    // Read and encode photo
    data, err := os.ReadFile(filepath)
    if err != nil {
        return err
    }
    
    // Save photo file
    photoFilename := fmt.Sprintf("photo-%d.jpg", time.Now().Unix())
    if err := p.client.SaveBinary(photoFilename, data); err != nil {
        return err
    }
    
    // Create metadata
    metadata := map[string]interface{}{
        "@context": "https://schema.org",
        "@type":    "ImageObject",
        "contentUrl": photoFilename,
        "caption": caption,
        "datePublished": time.Now().Format(time.RFC3339),
    }
    
    metaFilename := fmt.Sprintf("photo-%d.json", time.Now().Unix())
    return p.client.Save(metaFilename, metadata)
}

func (p *PhotoApp) Gallery(fingerprint string) ([]map[string]interface{}, error) {
    // Fetch all ImageObject posts from user
}
```

### 3. Discussion Forum

```go
type Forum struct {
    client *mau.Client
}

func (f *Forum) CreateThread(title, content string) error {
    thread := map[string]interface{}{
        "@context": "https://schema.org",
        "@type":    "DiscussionForumPosting",
        "headline": title,
        "articleBody": content,
        "dateCreated": time.Now().Format(time.RFC3339),
    }
    
    filename := fmt.Sprintf("thread-%d.json", time.Now().Unix())
    return f.client.Save(filename, thread)
}

func (f *Forum) Reply(threadID, content string) error {
    reply := map[string]interface{}{
        "@context": "https://schema.org",
        "@type":    "Comment",
        "text":     content,
        "parentItem": map[string]interface{}{
            "@type": "DiscussionForumPosting",
            "identifier": threadID,
        },
        "dateCreated": time.Now().Format(time.RFC3339),
    }
    
    filename := fmt.Sprintf("reply-%s-%d.json", threadID, time.Now().Unix())
    return f.client.Save(filename, reply)
}
```

## Best Practices

### 1. Content Naming

Use descriptive, sortable filenames:

```go
// Good: chronological and descriptive
"post-2026-03-11-1234567890.json"
"photo-sunset-2026-03-11.jpg"
"dm-abc123-2026-03-11.json"

// Avoid: ambiguous or non-sortable
"post.json"
"image.jpg"
"message-123.json"
```

### 2. Error Handling

Always handle errors gracefully, especially for network operations:

```go
func (a *App) SafeFetch(fingerprint string) error {
    // Set timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Attempt fetch with retry
    for attempt := 0; attempt < 3; attempt++ {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            if err := a.client.FetchFiles(fingerprint); err == nil {
                return nil
            }
            time.Sleep(time.Second * time.Duration(attempt+1))
        }
    }
    
    return fmt.Errorf("fetch failed after 3 attempts")
}
```

### 3. Local Caching

Cache remote content to reduce network usage:

```go
type Cache struct {
    store map[string]CachedItem
    mutex sync.RWMutex
}

type CachedItem struct {
    Data      map[string]interface{}
    FetchedAt time.Time
    TTL       time.Duration
}

func (c *Cache) Get(key string) (map[string]interface{}, bool) {
    c.mutex.RLock()
    defer c.mutex.RUnlock()
    
    item, exists := c.store[key]
    if !exists {
        return nil, false
    }
    
    // Check if expired
    if time.Since(item.FetchedAt) > item.TTL {
        return nil, false
    }
    
    return item.Data, true
}
```

### 4. Incremental Loading

Load content progressively for better UX:

```go
func (a *App) LoadTimeline(offset, limit int) ([]TimelineItem, error) {
    // Fetch a page of results
    allItems := a.getCachedTimeline()
    
    start := offset
    end := offset + limit
    
    if start >= len(allItems) {
        return []TimelineItem{}, nil
    }
    
    if end > len(allItems) {
        end = len(allItems)
    }
    
    return allItems[start:end], nil
}
```

### 5. Privacy Considerations

Always respect privacy settings:

```go
func (a *App) CanView(content map[string]interface{}, viewer string) bool {
    // Check if content is public
    if isPublic(content) {
        return true
    }
    
    // Check if viewer is in recipient list
    recipients := extractRecipients(content)
    for _, recipient := range recipients {
        if recipient == viewer {
            return true
        }
    }
    
    // Check if viewer is followed by author
    author := extractAuthor(content)
    following := a.client.FollowedBy(author)
    for _, fp := range following {
        if fp == viewer {
            return true
        }
    }
    
    return false
}
```

### 6. Content Validation

Validate content before processing:

```go
func validatePost(post map[string]interface{}) error {
    // Check required fields
    if post["@type"] == nil {
        return errors.New("missing @type field")
    }
    
    // Validate Schema.org type
    validTypes := []string{
        "SocialMediaPosting",
        "ImageObject",
        "Comment",
        "Message",
    }
    
    postType := post["@type"].(string)
    valid := false
    for _, t := range validTypes {
        if postType == t {
            valid = true
            break
        }
    }
    
    if !valid {
        return fmt.Errorf("invalid @type: %s", postType)
    }
    
    return nil
}
```

### 7. Performance Optimization

Index content for fast retrieval:

```go
type Index struct {
    byAuthor    map[string][]string // fingerprint -> filenames
    byTimestamp map[int64][]string  // timestamp -> filenames
    byType      map[string][]string // @type -> filenames
}

func (idx *Index) Add(filename string, content map[string]interface{}) {
    // Index by author
    if author := extractAuthor(content); author != "" {
        idx.byAuthor[author] = append(idx.byAuthor[author], filename)
    }
    
    // Index by timestamp
    if ts := extractTimestamp(content); !ts.IsZero() {
        idx.byTimestamp[ts.Unix()] = append(idx.byTimestamp[ts.Unix()], filename)
    }
    
    // Index by type
    if t, ok := content["@type"].(string); ok {
        idx.byType[t] = append(idx.byType[t], filename)
    }
}

func (idx *Index) FindByAuthor(fingerprint string) []string {
    return idx.byAuthor[fingerprint]
}
```

## Next Steps

- Read [Privacy & Security](09-privacy-security.md) for best practices
- Explore [Performance & Optimization](10-performance.md) for scaling tips
- Check [API Reference](11-api-reference.md) for complete documentation
- Review [Schema.org Types](12-schema-types.md) for more content types

## Additional Resources

- [Schema.org Documentation](https://schema.org)
- [ActivityStreams Vocabulary](https://www.w3.org/TR/activitystreams-vocabulary/)
- [JSON-LD Specification](https://json-ld.org)
- [Mau Example Apps](https://github.com/mau-network/examples)

---

**Need Help?** Join the discussion at [github.com/mau-network/mau/discussions](https://github.com/mau-network/mau/discussions)
