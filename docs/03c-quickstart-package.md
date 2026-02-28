# Quick Start: Mau Go Package

This tutorial shows how to build custom Mau applications using the **Go package**. You'll create a simple social app from scratch.

**Time:** ~20 minutes  
**Prerequisites:**
- Go 1.21+ installed
- Basic Go knowledge
- Completed [CLI Tutorial](03b-quickstart-cli.md) (recommended)

## Why Use the Package?

The Mau Go package lets you:
- Build custom social applications
- Integrate Mau into existing Go projects
- Create specialized clients (mobile, desktop, web)
- Automate workflows and bots

## Step 1: Set Up Your Project

```bash
mkdir mau-app
cd mau-app
go mod init example.com/mau-app
go get github.com/mau-network/mau@latest
```

## Step 2: Create or Open an Account

Create `main.go`:

```go
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/mau-network/mau"
)

func main() {
	// Mau directory path
	mauDir := filepath.Join(os.Getenv("HOME"), ".mau-app")

	// Check if account exists
	accountFile := filepath.Join(mauDir, ".mau", "account.pgp")
	var account *mau.Account
	var err error

	if _, err := os.Stat(accountFile); os.IsNotExist(err) {
		// Create new account
		fmt.Println("Creating new account...")
		fmt.Print("Name: ")
		var name string
		fmt.Scanln(&name)
		
		fmt.Print("Email: ")
		var email string
		fmt.Scanln(&email)
		
		fmt.Print("Passphrase: ")
		var passphrase string
		fmt.Scanln(&passphrase)

		account, err = mau.NewAccount(mauDir, name, email, passphrase)
		if err != nil {
			log.Fatalf("Failed to create account: %v", err)
		}
		fmt.Println("✓ Account created!")
	} else {
		// Open existing account
		fmt.Print("Enter passphrase: ")
		var passphrase string
		fmt.Scanln(&passphrase)

		account, err = mau.OpenAccount(mauDir, passphrase)
		if err != nil {
			log.Fatalf("Failed to open account: %v", err)
		}
		fmt.Println("✓ Account opened!")
	}

	// Display account info
	fmt.Printf("\nName:        %s\n", account.Name())
	fmt.Printf("Email:       %s\n", account.Email())
	fmt.Printf("Fingerprint: %s\n", account.Fingerprint())
}
```

Run it:

```bash
go run main.go
```

Output:
```
Creating new account...
Name: Alice
Email: alice@example.com
Passphrase: [your-password]
✓ Account created!

Name:        Alice
Email:       alice@example.com
Fingerprint: 5D000B2F2C040A1675B49D7F0C7CB7DC36999D56
```

## Step 3: Create and Share a Post

Add a function to create posts:

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mau-network/mau"
)

// Post represents a social media posting
type Post struct {
	Context       string    `json:"@context"`
	Type          string    `json:"@type"`
	Headline      string    `json:"headline"`
	ArticleBody   string    `json:"articleBody,omitempty"`
	Author        Author    `json:"author"`
	DatePublished time.Time `json:"datePublished"`
}

type Author struct {
	Type       string `json:"@type"`
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
}

func createPost(account *mau.Account, headline, body string) error {
	post := Post{
		Context:       "https://schema.org",
		Type:          "SocialMediaPosting",
		Headline:      headline,
		ArticleBody:   body,
		Author: Author{
			Type:       "Person",
			Name:       account.Name(),
			Identifier: account.Fingerprint().String(),
		},
		DatePublished: time.Now(),
	}

	// Marshal to JSON
	postJSON, err := json.MarshalIndent(post, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal post: %w", err)
	}

	// Create file from JSON
	reader := bytes.NewReader(postJSON)
	filename := fmt.Sprintf("post-%d.json", time.Now().Unix())

	// Add file (encrypted for yourself = public post)
	file, err := account.AddFile(reader, filename, nil)
	if err != nil {
		return fmt.Errorf("failed to add file: %w", err)
	}

	fmt.Printf("✓ Post created: %s\n", file.Name())
	return nil
}

func main() {
	// ... (previous account setup code) ...

	// Create a post
	fmt.Println("\n--- Create a Post ---")
	if err := createPost(account, "Hello from Go!", "This post was created programmatically."); err != nil {
		log.Fatalf("Failed to create post: %v", err)
	}
}
```

Run it:

```bash
go run main.go
```

Output:
```
✓ Account opened!
...
--- Create a Post ---
✓ Post created: post-1709107200.json
```

## Step 4: List Your Posts

Add a function to list files:

```go
func listPosts(account *mau.Account) error {
	// List all your files
	files := account.ListFiles(account.Fingerprint(), time.Time{}, 100)

	fmt.Printf("\n📝 Your posts (%d):\n\n", len(files))

	for _, file := range files {
		// Decrypt and parse
		content, err := file.Read()
		if err != nil {
			log.Printf("Error reading %s: %v", file.Name(), err)
			continue
		}

		var post Post
		if err := json.Unmarshal(content, &post); err != nil {
			log.Printf("Error parsing %s: %v", file.Name(), err)
			continue
		}

		fmt.Printf("🔹 %s\n", post.Headline)
		if post.ArticleBody != "" {
			fmt.Printf("   %s\n", post.ArticleBody)
		}
		fmt.Printf("   📅 %s\n\n", post.DatePublished.Format("Jan 2, 2006 15:04"))
	}

	return nil
}

func main() {
	// ... (previous code) ...

	// List posts
	if err := listPosts(account); err != nil {
		log.Fatalf("Failed to list posts: %v", err)
	}
}
```

## Step 5: Add a Friend

Add friend management:

```go
func addFriend(account *mau.Account, publicKeyPath string) error {
	// Open friend's public key file
	file, err := os.Open(publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to open key file: %w", err)
	}
	defer file.Close()

	// Add friend
	friend, err := account.AddFriend(file)
	if err != nil {
		return fmt.Errorf("failed to add friend: %w", err)
	}

	fmt.Printf("✓ Friend added: %s <%s>\n", friend.Name(), friend.Email())
	fmt.Printf("  Fingerprint: %s\n", friend.Fingerprint())

	return nil
}

func listFriends(account *mau.Account) error {
	keyring, err := account.ListFriends()
	if err != nil {
		return fmt.Errorf("failed to list friends: %w", err)
	}

	friends := keyring.All()
	fmt.Printf("\n👥 Friends (%d):\n\n", len(friends))

	for _, friend := range friends {
		fmt.Printf("• %s <%s>\n", friend.Name(), friend.Email())
		fmt.Printf("  Fingerprint: %s\n", friend.Fingerprint())
		
		// Check if following
		follows, _ := account.ListFollows()
		following := false
		for _, f := range follows {
			if f.Fingerprint() == friend.Fingerprint() {
				following = true
				break
			}
		}
		
		if following {
			fmt.Printf("  Status: Following ✓\n")
		} else {
			fmt.Printf("  Status: Not following\n")
		}
		fmt.Println()
	}

	return nil
}

func main() {
	// ... (previous code) ...

	// Example: Add a friend from file
	// friendKeyPath := "/path/to/friend-pubkey.pgp"
	// if err := addFriend(account, friendKeyPath); err != nil {
	//     log.Printf("Failed to add friend: %v", err)
	// }

	// List friends
	if err := listFriends(account); err != nil {
		log.Fatalf("Failed to list friends: %v", err)
	}
}
```

## Step 6: Follow a Friend and Sync

```go
func followFriend(account *mau.Account, fingerprint mau.Fingerprint) error {
	// Get friend from keyring
	keyring, err := account.ListFriends()
	if err != nil {
		return fmt.Errorf("failed to list friends: %w", err)
	}

	friend, err := keyring.Get(fingerprint)
	if err != nil {
		return fmt.Errorf("friend not found: %w", err)
	}

	// Follow
	if err := account.Follow(friend); err != nil {
		return fmt.Errorf("failed to follow: %w", err)
	}

	fmt.Printf("✓ Now following: %s\n", friend.Name())
	return nil
}

func syncFromFriend(account *mau.Account, friendURL string, friendFPR mau.Fingerprint) error {
	// Create HTTP client for the friend
	// Note: This is simplified; real implementation needs DNS names from discovery
	client, err := account.Client(friendFPR, []string{friendURL})
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Get last sync time
	lastSync := account.GetLastSyncTime(friendFPR)
	fmt.Printf("📥 Syncing from %s (since %v)...\n", friendFPR.String()[:8], lastSync)

	// Sync files
	if err := client.Sync(lastSync); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	fmt.Println("✓ Sync complete!")
	return nil
}
```

## Step 7: Start an HTTP Server

```go
func startServer(account *mau.Account, port string) error {
	// Create server instance
	// knownNodes can be empty for local-only operation
	server, err := account.Server(nil)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	fmt.Printf("🚀 Starting server on :%s\n", port)
	fmt.Printf("   Fingerprint: %s\n", account.Fingerprint())
	
	// Start server (blocking)
	if err := server.Listen(":" + port); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func main() {
	// ... (previous code) ...

	// Start server (comment out for non-server mode)
	// if err := startServer(account, "8080"); err != nil {
	//     log.Fatalf("Server error: %v", err)
	// }
}
```

## Step 8: Build a Simple Feed Reader

Complete example application:

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mau-network/mau"
)

type FeedItem struct {
	Headline      string    `json:"headline"`
	Body          string    `json:"articleBody"`
	Author        string    `json:"-"`
	AuthorFPR     string    `json:"-"`
	DatePublished time.Time `json:"datePublished"`
	Filename      string    `json:"-"`
}

func getFeed(account *mau.Account) ([]FeedItem, error) {
	var feed []FeedItem

	// Get your own posts
	myFiles := account.ListFiles(account.Fingerprint(), time.Time{}, 50)
	for _, file := range myFiles {
		item, err := parseFeedItem(file, account.Name(), account.Fingerprint().String())
		if err == nil {
			feed = append(feed, item)
		}
	}

	// Get friends' posts
	follows, err := account.ListFollows()
	if err != nil {
		return nil, err
	}

	for _, friend := range follows {
		friendFiles := account.ListFiles(friend.Fingerprint(), time.Time{}, 50)
		for _, file := range friendFiles {
			item, err := parseFeedItem(file, friend.Name(), friend.Fingerprint().String())
			if err == nil {
				feed = append(feed, item)
			}
		}
	}

	// Sort by date (newest first)
	// (Implement sorting logic here)

	return feed, nil
}

func parseFeedItem(file *mau.File, authorName, authorFPR string) (FeedItem, error) {
	content, err := file.Read()
	if err != nil {
		return FeedItem{}, err
	}

	var post struct {
		Type          string    `json:"@type"`
		Headline      string    `json:"headline"`
		ArticleBody   string    `json:"articleBody"`
		DatePublished time.Time `json:"datePublished"`
	}

	if err := json.Unmarshal(content, &post); err != nil {
		return FeedItem{}, err
	}

	// Only include social media postings
	if post.Type != "SocialMediaPosting" {
		return FeedItem{}, fmt.Errorf("not a social media posting")
	}

	return FeedItem{
		Headline:      post.Headline,
		Body:          post.ArticleBody,
		Author:        authorName,
		AuthorFPR:     authorFPR,
		DatePublished: post.DatePublished,
		Filename:      file.Name(),
	}, nil
}

func displayFeed(account *mau.Account) error {
	feed, err := getFeed(account)
	if err != nil {
		return err
	}

	fmt.Printf("\n📰 Your Feed (%d posts)\n", len(feed))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for _, item := range feed {
		fmt.Printf("\n👤 %s (%s...)\n", item.Author, item.AuthorFPR[:8])
		fmt.Printf("🔹 %s\n", item.Headline)
		if item.Body != "" {
			fmt.Printf("   %s\n", item.Body)
		}
		fmt.Printf("📅 %s\n", item.DatePublished.Format("Jan 2, 2006 15:04"))
	}

	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	return nil
}

func main() {
	mauDir := filepath.Join(os.Getenv("HOME"), ".mau-app")
	
	fmt.Print("Enter passphrase: ")
	var passphrase string
	fmt.Scanln(&passphrase)

	account, err := mau.OpenAccount(mauDir, passphrase)
	if err != nil {
		log.Fatalf("Failed to open account: %v", err)
	}

	// Display feed
	if err := displayFeed(account); err != nil {
		log.Fatalf("Failed to display feed: %v", err)
	}
}
```

Build and run:

```bash
go build -o mau-reader .
./mau-reader
```

## Step 9: Send Private Messages

```go
type Message struct {
	Context   string    `json:"@context"`
	Type      string    `json:"@type"`
	Text      string    `json:"text"`
	Sender    Author    `json:"sender"`
	Recipient Author    `json:"recipient"`
	DateSent  time.Time `json:"dateSent"`
}

func sendMessage(account *mau.Account, recipientFPR mau.Fingerprint, text string) error {
	// Get recipient friend
	keyring, err := account.ListFriends()
	if err != nil {
		return err
	}

	recipient, err := keyring.Get(recipientFPR)
	if err != nil {
		return fmt.Errorf("recipient not found: %w", err)
	}

	msg := Message{
		Context: "https://schema.org",
		Type:    "Message",
		Text:    text,
		Sender: Author{
			Type:       "Person",
			Identifier: account.Fingerprint().String(),
		},
		Recipient: Author{
			Type:       "Person",
			Identifier: recipientFPR.String(),
		},
		DateSent: time.Now(),
	}

	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Create file encrypted only for recipient
	reader := bytes.NewReader(msgJSON)
	filename := fmt.Sprintf("msg-to-%s-%d.json", recipientFPR.String()[:8], time.Now().Unix())

	_, err = account.AddFile(reader, filename, []*mau.Friend{recipient})
	if err != nil {
		return err
	}

	fmt.Printf("✓ Message sent to %s\n", recipient.Name())
	return nil
}
```

## Step 10: Error Handling Best Practices

```go
package main

import (
	"errors"
	"fmt"
	"github.com/mau-network/mau"
)

func robustAccountOpen(mauDir, passphrase string) (*mau.Account, error) {
	account, err := mau.OpenAccount(mauDir, passphrase)
	if err != nil {
		// Check for specific errors
		if errors.Is(err, mau.ErrIncorrectPassphrase) {
			return nil, fmt.Errorf("incorrect passphrase, please try again")
		}
		if errors.Is(err, mau.ErrNoIdentity) {
			return nil, fmt.Errorf("no account found at %s, run 'init' first", mauDir)
		}
		return nil, fmt.Errorf("failed to open account: %w", err)
	}
	return account, nil
}

func robustAddFriend(account *mau.Account, keyPath string) error {
	file, err := os.Open(keyPath)
	if err != nil {
		return fmt.Errorf("cannot open key file: %w", err)
	}
	defer file.Close()

	friend, err := account.AddFriend(file)
	if err != nil {
		return fmt.Errorf("failed to import friend key: %w", err)
	}

	fmt.Printf("✓ Added: %s\n", friend.Name())
	return nil
}
```

## Complete Application Structure

Recommended project layout:

```
mau-app/
├── main.go           # Entry point
├── account.go        # Account management
├── posts.go          # Post creation/reading
├── friends.go        # Friend management
├── sync.go           # Syncing logic
├── server.go         # HTTP server
├── types.go          # JSON-LD type definitions
└── go.mod
```

## What You've Learned

✅ Create and open Mau accounts programmatically  
✅ Create and share posts using Go structs  
✅ List and read encrypted files  
✅ Manage friends and follows  
✅ Build custom sync logic  
✅ Start HTTP servers  
✅ Parse Schema.org JSON-LD content  
✅ Send private messages  
✅ Build complete social applications  

## Key Package APIs

| Function | Purpose |
|----------|---------|
| `mau.NewAccount(dir, name, email, pass)` | Create new account |
| `mau.OpenAccount(dir, pass)` | Open existing account |
| `account.AddFile(reader, name, recipients)` | Create encrypted file |
| `account.GetFile(fpr, name)` | Retrieve file |
| `account.ListFiles(fpr, after, limit)` | List files |
| `account.AddFriend(keyReader)` | Import friend's key |
| `account.Follow(friend)` | Start following |
| `account.Client(fpr, addresses)` | Create sync client |
| `account.Server(peers)` | Create HTTP server |
| `file.Read()` | Decrypt and read content |

## Comparison

| Level | Best For | Complexity |
|-------|----------|-----------|
| **Raw GPG** | Understanding primitives | High |
| **Mau CLI** | End users, scripts | Low |
| **Mau Package** | Custom apps, integration | Medium |

Use the package when you need:
- Custom UI (mobile, desktop, web)
- Integration with existing systems
- Automation and bots
- Specialized social apps

## Next Steps

- **[Building Social Apps](08-building-social-apps.md)** - Design patterns
- **[API Reference](11-api-reference.md)** - Complete package documentation
- **[Networking Guide](06-networking.md)** - Peer discovery and Kademlia
- **Study the GUI:** `gui/` directory has a full implementation

---

**Tip:** Check out the test files (`*_test.go`) in the Mau repository for more usage examples!
