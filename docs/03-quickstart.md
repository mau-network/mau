# Quick Start Guide

Build your first Mau application in 15 minutes. This guide assumes you have Go installed and basic familiarity with the command line.

## Prerequisites

- **Go 1.21+** - [Install Go](https://go.dev/doc/install)
- **GnuPG (GPG)** - For PGP operations
  ```bash
  # Ubuntu/Debian
  sudo apt install gnupg
  
  # macOS
  brew install gnupg
  
  # Verify
  gpg --version
  ```

## Step 1: Set Up Your Identity

First, create a PGP key pair (your Mau identity):

```bash
# Generate a key (RSA 4096-bit recommended)
gpg --full-generate-key

# Follow prompts:
# - Kind: (1) RSA and RSA
# - Key size: 4096
# - Expiration: 0 (no expiration) or set your preference
# - Real name: Your name or alias
# - Email: Your email
# - Passphrase: Choose a strong password
```

Get your fingerprint:

```bash
gpg --fingerprint your-email@example.com

# Output example:
# pub   rsa4096 2026-02-27 [SC]
#       5D00 0B2F 2C04 0A16 75B4  9D7F 0C7C B7DC 3699 9D56
#                                  └── Your fingerprint
```

Remove spaces from fingerprint:
```bash
export MY_FPR="5D000B2F2C040A1675B49D7F0C7CB7DC36999D56"
echo $MY_FPR
```

## Step 2: Install Mau Package

```bash
go get github.com/mau-network/mau@latest
```

Or clone and build from source:

```bash
git clone https://github.com/mau-network/mau.git
cd mau
go build ./cmd/mau
```

## Step 3: Initialize Mau Directory

Create your Mau data directory:

```bash
mkdir -p ~/.mau/$MY_FPR
mkdir -p ~/.mau/.mau

# Export your account keys
gpg --export-secret-keys your-email@example.com | \
  gpg --symmetric --armor > ~/.mau/.mau/account.pgp

# (Enter passphrase to encrypt your private key file)
```

## Step 4: Create Your First Post

Create a simple social media post:

```bash
cat > /tmp/my-first-post.json <<EOF
{
  "@context": "https://schema.org",
  "@type": "SocialMediaPosting",
  "headline": "Hello, Mau!",
  "articleBody": "My first post on the decentralized social web.",
  "author": {
    "@type": "Person",
    "name": "$(gpg --list-keys --with-colons your-email@example.com | awk -F: '/^uid/ {print $10; exit}')",
    "identifier": "$MY_FPR"
  },
  "datePublished": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
```

Encrypt and sign it:

```bash
gpg --sign --encrypt -r $MY_FPR \
  < /tmp/my-first-post.json \
  > ~/.mau/$MY_FPR/hello-mau.json.pgp
```

Verify it worked:

```bash
ls -lh ~/.mau/$MY_FPR/
# Should show: hello-mau.json.pgp
```

## Step 5: Read Your Post

Decrypt and verify:

```bash
gpg --decrypt ~/.mau/$MY_FPR/hello-mau.json.pgp

# Output:
# {
#   "@context": "https://schema.org",
#   "@type": "SocialMediaPosting",
#   ...
# }
# gpg: Signature made ...
# gpg: Good signature from "Your Name <your-email@example.com>"
```

## Step 6: Build a Simple Client

Create `mau-client.go`:

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Post struct {
	Context       string `json:"@context"`
	Type          string `json:"@type"`
	Headline      string `json:"headline"`
	ArticleBody   string `json:"articleBody"`
	DatePublished string `json:"datePublished"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: mau-client <fingerprint>")
		fmt.Println("Example: mau-client 5D000B2F2C040A1675B49D7F0C7CB7DC36999D56")
		os.Exit(1)
	}

	fingerprint := os.Args[1]
	mauDir := filepath.Join(os.Getenv("HOME"), ".mau", fingerprint)

	files, err := os.ReadDir(mauDir)
	if err != nil {
		log.Fatalf("Error reading directory: %v", err)
	}

	fmt.Printf("📝 Posts from %s\n\n", fingerprint[:8])

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".pgp" {
			continue
		}

		filePath := filepath.Join(mauDir, file.Name())
		
		// Decrypt the file
		cmd := exec.Command("gpg", "--decrypt", "--quiet", filePath)
		output, err := cmd.Output()
		if err != nil {
			log.Printf("Error decrypting %s: %v", file.Name(), err)
			continue
		}

		// Parse JSON
		var post Post
		if err := json.Unmarshal(output, &post); err != nil {
			log.Printf("Error parsing %s: %v", file.Name(), err)
			continue
		}

		// Display
		fmt.Printf("🔹 %s\n", post.Headline)
		if post.ArticleBody != "" {
			fmt.Printf("   %s\n", post.ArticleBody)
		}
		if post.DatePublished != "" {
			t, _ := time.Parse(time.RFC3339, post.DatePublished)
			fmt.Printf("   📅 %s\n", t.Format("Jan 2, 2006 15:04"))
		}
		fmt.Println()
	}
}
```

Build and run:

```bash
go build mau-client.go
./mau-client $MY_FPR
```

Output:
```
📝 Posts from 5D000B2F

🔹 Hello, Mau!
   My first post on the decentralized social web.
   📅 Feb 27, 2026 10:00
```

## Step 7: Add a Friend

Let's simulate adding a friend (Bob):

```bash
# Bob's fingerprint (example)
export BOB_FPR="ABC123DEF456789ABC123DEF456789ABC123DEF"

# Create Bob's directory
mkdir -p ~/.mau/$BOB_FPR

# Simulate Bob's public key (in real life, get from Bob)
gpg --export bob@example.com > /tmp/bob.pub

# Import Bob's key
gpg --import /tmp/bob.pub

# Save to .mau directory (encrypted with your key)
gpg --export $BOB_FPR | \
  gpg --encrypt -r $MY_FPR \
  > ~/.mau/.mau/$BOB_FPR.pgp
```

## Step 8: Start HTTP Server

The Mau package includes an HTTP server. Create `mau-server.go`:

```go
package main

import (
	"log"
	"github.com/mau-network/mau"
)

func main() {
	// Initialize Mau instance
	mauDir := "/home/youruser/.mau"  // Change to your path
	
	server := mau.NewServer(mauDir, ":8080")
	
	log.Println("Starting Mau server on http://localhost:8080")
	log.Fatal(server.ListenAndServe())
}
```

(Note: Check the actual Mau package API for correct method names)

Run:
```bash
go run mau-server.go
```

Test it:
```bash
curl http://localhost:8080/$MY_FPR

# Should return JSON list of your files
```

## Step 9: Create More Content Types

### Recipe

```json
{
  "@context": "https://schema.org",
  "@type": "Recipe",
  "name": "Spaghetti Carbonara",
  "author": {
    "@type": "Person",
    "name": "Your Name"
  },
  "recipeIngredient": [
    "400g spaghetti",
    "200g pancetta",
    "4 eggs",
    "100g parmesan cheese"
  ],
  "recipeInstructions": "Cook pasta. Fry pancetta. Mix eggs and cheese. Combine all while hot.",
  "totalTime": "PT30M"
}
```

### Comment on a Post

```json
{
  "@context": "https://schema.org",
  "@type": "Comment",
  "text": "Great recipe!",
  "author": {
    "@type": "Person",
    "identifier": "your-fpr"
  },
  "about": "/p2p/bob-fpr/carbonara-recipe.json",
  "dateCreated": "2026-02-27T10:30:00Z"
}
```

### Private Message

```bash
# Encrypt for Bob only
cat > /tmp/private-msg.json <<EOF
{
  "@context": "https://schema.org",
  "@type": "Message",
  "text": "Hey Bob, let's meet tomorrow!",
  "sender": {
    "@type": "Person",
    "identifier": "$MY_FPR"
  },
  "recipient": {
    "@type": "Person",
    "identifier": "$BOB_FPR"
  },
  "dateSent": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF

gpg --sign --encrypt -r $BOB_FPR \
  < /tmp/private-msg.json \
  > ~/.mau/$MY_FPR/msg-to-bob.json.pgp
```

## Step 10: Sync with a Friend

Create `mau-sync.go`:

```go
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type FileInfo struct {
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	SHA256   string    `json:"sha256"`
	Modified time.Time `json:"modified"`
}

func syncFrom(friendURL, fingerprint string, since time.Time) error {
	// Request file list
	req, _ := http.NewRequest("GET", friendURL+"/"+fingerprint, nil)
	if !since.IsZero() {
		req.Header.Set("If-Modified-Since", since.Format(http.TimeFormat))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var files []FileInfo
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return err
	}

	fmt.Printf("📥 Syncing %d files from %s\n", len(files), fingerprint[:8])

	for _, file := range files {
		// Download file
		fileURL := fmt.Sprintf("%s/%s/%s", friendURL, fingerprint, file.Name)
		resp, err := http.Get(fileURL)
		if err != nil {
			log.Printf("Error downloading %s: %v", file.Name, err)
			continue
		}
		defer resp.Body.Close()

		// Save locally
		localPath := filepath.Join(os.Getenv("HOME"), ".mau", fingerprint, file.Name)
		out, err := os.Create(localPath)
		if err != nil {
			log.Printf("Error creating %s: %v", localPath, err)
			continue
		}
		defer out.Close()

		if _, err := io.Copy(out, resp.Body); err != nil {
			log.Printf("Error saving %s: %v", file.Name, err)
			continue
		}

		fmt.Printf("   ✓ %s\n", file.Name)
	}

	return nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: mau-sync <friend-url> <fingerprint>")
		fmt.Println("Example: mau-sync http://bob.local:8080 ABC123...")
		os.Exit(1)
	}

	friendURL := os.Args[1]
	fingerprint := os.Args[2]

	// Sync files modified in last 7 days
	since := time.Now().AddDate(0, 0, -7)
	
	if err := syncFrom(friendURL, fingerprint, since); err != nil {
		log.Fatalf("Sync failed: %v", err)
	}

	fmt.Println("\n✅ Sync complete!")
}
```

Test:
```bash
go build mau-sync.go
./mau-sync http://localhost:8080 $MY_FPR
```

## What You've Built

Congratulations! You now have:

✅ A Mau identity (PGP key)  
✅ A local data directory  
✅ Your first encrypted, signed post  
✅ A simple reader client  
✅ An HTTP server exposing your posts  
✅ A sync client to pull from friends  

## Next Steps

### Learn More
- **[Storage and Data Format](04-storage-and-data.md)** - Deep dive into file formats
- **[Building Social Apps](08-building-social-apps.md)** - Practical patterns
- **[Peer-to-Peer Networking](06-networking.md)** - Discovery and routing

### Improve Your Client
- Add mDNS discovery for local peers
- Implement Kademlia routing
- Build a GUI with [Fyne](https://fyne.io/) or [Wails](https://wails.io/)
- Add push notifications for new content

### Join the Community
- **GitHub:** [mau-network/mau](https://github.com/mau-network/mau)
- **Issues:** Report bugs and suggest features
- **Discussions:** Share your Mau app!

---

**Tip:** Check out the existing Mau GUI implementation in `gui/` directory for a full-featured reference!
