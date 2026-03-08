# Storage and Data Format

This document explains how Mau stores and structures data on your filesystem, how JSON-LD works with Schema.org vocabulary, and how files are organized for peer-to-peer synchronization.

## Table of Contents
1. [Overview](#overview)
2. [Directory Structure](#directory-structure)
3. [File Naming and Content IDs](#file-naming-and-content-ids)
4. [JSON-LD and Schema.org](#json-ld-and-schemaorg)
5. [Content Versioning](#content-versioning)
6. [File Encryption and Signing](#file-encryption-and-signing)
7. [Working with Files](#working-with-files)
8. [Best Practices](#best-practices)

---

## Overview

### Everything is a File

Mau adopts the Unix philosophy: **everything is a file**. Every piece of content you create—a status update, a blog post, a photo, a comment—is stored as a file on your filesystem.

**Benefits of file-based storage:**

- **Simple backups** - Just `tar` your directory
- **Version control** - Use git, rsync, or any backup tool
- **Universal access** - Any program can read/write (CLI, GUI, text editor extension)
- **User ownership** - Data lives on YOUR disk, not a company's server
- **Easy deletion** - `rm` a file, it's gone
- **Network filesystem support** - Store on NAS, remote mounts, etc.

### Storage Philosophy

```
┌─────────────────────────────────────────────────────────┐
│                    Your Filesystem                      │
│  ┌────────────────────────────────────────────────┐    │
│  │  Mau Directory (e.g., ~/.mau/)                 │    │
│  │  ├── 5D000B2F.../ (your public key FPR)       │    │
│  │  │   ├── hello-world.json.pgp                 │    │
│  │  │   ├── recipe.json.pgp                      │    │
│  │  │   └── photo-2026.json.pgp                  │    │
│  │  ├── A1234567.../ (friend's FPR)              │    │
│  │  │   ├── comment.json.pgp                     │    │
│  │  │   └── like.json.pgp                        │    │
│  │  └── .mau/ (metadata)                          │    │
│  │      ├── A1234567....pgp (friend's public key)│    │
│  │      └── config.json                           │    │
│  └────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────┘
```

Files are:
- **Encrypted** with PGP (`.pgp` extension)
- **Signed** to prevent tampering
- **JSON-LD formatted** (structured content)
- **Schema.org compliant** (shared vocabulary)

---

## Directory Structure

### Root Directory Layout

Mau stores all data in a single root directory (commonly `~/.mau/`):

```
~/.mau/
├── <your-fingerprint>/        # Your content
├── <friend1-fingerprint>/     # Friend's synced content
├── <friend2-fingerprint>/     # Another friend's content
├── .<blocked-fingerprint>/    # Blocked user (hidden, not synced)
└── .mau/                       # Mau metadata directory
```

### User Content Directories

Each user (you and your contacts) has a directory named by their **PGP key fingerprint** (40-character hex, lowercase):

```
5d000b2f2c040a1675b49d7f0c7cb7dc36999d56/
├── hello-world.json.pgp
├── recipe-pasta.json.pgp
├── recipe-pasta.json.pgp.versions/
│   ├── a7f3b9...  (old version)
│   └── d2c1e8...  (older version)
├── photo-vacation.json.pgp
└── draft-blog.json.pgp
```

**Key points:**
- One directory per user
- Files are encrypted with `.pgp` extension
- Versioned files have `.versions/` subdirectories
- You control your own directory's files
- Friends' directories are read-only (synced from their peers)

### Hidden Directories (Blocked Users)

Prefix a directory with `.` to block a user:

```
.a1234567890abcdef1234567890abcdef12345678/
```

- Hidden from listings
- Not synced to peers
- Local-only (you can still read if needed)
- Use for blocking/muting users

### Metadata Directory (`.mau/`)

The `.mau/` directory stores system metadata:

```
.mau/
├── <friend1-fingerprint>.pgp   # Friend's public key
├── <friend2-fingerprint>.pgp   # Another friend's public key
├── config.json                  # Local configuration
├── peers.json                   # Known peer addresses
└── sync-state.json              # Sync timestamps
```

**Contents:**
- **Public keys** - PGP keys of contacts (`.pgp` files)
- **Configuration** - Local settings (port, paths, preferences)
- **Peer addresses** - Known network locations of contacts
- **Sync state** - Last sync times for each peer

---

## File Naming and Content IDs

### File Names as IDs

In Mau, **the filename IS the content ID**, including the `.pgp` extension:

```
hello-world.json.pgp
└── Content ID: "hello-world.json.pgp"
```

**The `.pgp` extension is required everywhere:**

- **On disk:** `hello-world.json.pgp`
- **Content ID:** `hello-world.json.pgp` (includes `.pgp`)
- **In URLs:** `/p2p/<fingerprint>/hello-world.json.pgp`
- **In JSON-LD @id:** `/p2p/<fingerprint>/hello-world.json.pgp`

When you call `AddFile(reader, "hello.txt", recipients)`, Mau automatically appends `.pgp` if not present, storing it as `hello.txt.pgp`. All references to the file must include the `.pgp` extension.

**Rules:**
- Filenames must be unique within a user's directory (including `.pgp`)
- IDs are scoped by user (two users can use the same filename)
- Updating the file (same name) creates a new version
- Deleting the file deletes the content
- No database, no internal IDs—just filenames

### Full Content Address

A content's full address combines the user fingerprint and filename (with `.pgp`):

```
/p2p/<user-fingerprint>/<filename.pgp>
```

**Examples:**
```
/p2p/5d000b2f2c040a1675b49d7f0c7cb7dc36999d56/hello-world.json.pgp
/p2p/5d000b2f2c040a1675b49d7f0c7cb7dc36999d56/recipe-pasta.json.pgp
```

### Naming Conventions

**Good filenames (before `.pgp` is added automatically):**
```
hello-world.json           → stored as hello-world.json.pgp
post-2026-03-08.json       → stored as post-2026-03-08.json.pgp
recipe-pasta.json          → stored as recipe-pasta.json.pgp
photo-vacation.json        → stored as photo-vacation.json.pgp
```

**If you include `.pgp` manually:**
```
hello-world.json.pgp       → stored as hello-world.json.pgp (no double .pgp)
```

Mau's `AddFile()` checks if the name ends with `.pgp` and adds it only if missing. Either way, the final filename always includes `.pgp`.

**Avoid:**
- Spaces (use hyphens or underscores)
- Special characters (`/`, `\`, `*`, `?`, etc.)
- Very long names (filesystem limits)
- Non-ASCII characters (for maximum compatibility)

**Tip:** Use descriptive names. They act as both IDs and human-readable labels.

---

## JSON-LD and Schema.org

### Why JSON-LD?

Mau uses **JSON-LD** (JSON for Linked Data) to structure content. JSON-LD extends JSON with:

- **@context** - Defines the vocabulary (usually Schema.org)
- **@type** - Specifies the content type (e.g., `SocialMediaPosting`)
- **Linked data** - References to other content/users

### Basic JSON-LD Structure

Every Mau file (before encryption) looks like this:

```json
{
  "@context": "https://schema.org",
  "@type": "SocialMediaPosting",
  "headline": "Hello, decentralized world!",
  "author": {
    "@type": "Person",
    "name": "Alice",
    "identifier": "5d000b2f2c040a1675b49d7f0c7cb7dc36999d56"
  },
  "datePublished": "2026-03-08T07:00:00Z",
  "text": "This is my first Mau post!"
}
```

**Required fields:**
- `@context` - Usually `"https://schema.org"`
- `@type` - Schema.org type (see below)

**Common fields:**
- `datePublished` - ISO 8601 timestamp
- `author` - Person object (usually with fingerprint)
- `headline` / `text` / `name` - Content text

### Schema.org Types

Schema.org provides a shared vocabulary for structured content. Here are common types for social applications:

#### Social Media

**SocialMediaPosting** - Status updates, tweets, posts
```json
{
  "@context": "https://schema.org",
  "@type": "SocialMediaPosting",
  "headline": "Quick update",
  "text": "Just implemented a cool feature!",
  "datePublished": "2026-03-08T10:30:00Z"
}
```

**Comment** - Comments on content
```json
{
  "@context": "https://schema.org",
  "@type": "Comment",
  "text": "Great post!",
  "dateCreated": "2026-03-08T10:45:00Z",
  "parentItem": {
    "@type": "SocialMediaPosting",
    "@id": "/p2p/5d000b.../hello-world.json.pgp"
  }
}
```

**Message** - Private messages
```json
{
  "@context": "https://schema.org",
  "@type": "Message",
  "text": "Hey, want to grab coffee?",
  "dateReceived": "2026-03-08T11:00:00Z",
  "sender": { "@type": "Person", "identifier": "a12345..." },
  "recipient": { "@type": "Person", "identifier": "5d000b..." }
}
```

#### Content Types

**Article** - Blog posts, long-form content
```json
{
  "@context": "https://schema.org",
  "@type": "Article",
  "headline": "Building P2P Apps with Mau",
  "articleBody": "In this post, I'll show you...",
  "author": { "@type": "Person", "name": "Alice" },
  "datePublished": "2026-03-08T12:00:00Z"
}
```

**ImageObject** - Photos, images
```json
{
  "@context": "https://schema.org",
  "@type": "ImageObject",
  "name": "Sunset at the beach",
  "contentUrl": "file:///home/user/.mau/photos/sunset.jpg",
  "encodingFormat": "image/jpeg",
  "dateCreated": "2026-03-07T18:30:00Z"
}
```

**VideoObject** - Videos
```json
{
  "@context": "https://schema.org",
  "@type": "VideoObject",
  "name": "Mau Tutorial",
  "contentUrl": "file:///home/user/.mau/videos/tutorial.mp4",
  "duration": "PT10M30S",
  "uploadDate": "2026-03-08T09:00:00Z"
}
```

**Recipe** - Cooking recipes
```json
{
  "@context": "https://schema.org",
  "@type": "Recipe",
  "name": "Pasta Carbonara",
  "recipeIngredient": ["200g pasta", "100g guanciale", "2 eggs", "Pecorino"],
  "recipeInstructions": "1. Boil pasta. 2. Fry guanciale...",
  "totalTime": "PT30M"
}
```

#### Actions

**LikeAction** - Likes, favorites
```json
{
  "@context": "https://schema.org",
  "@type": "LikeAction",
  "agent": { "@type": "Person", "identifier": "5d000b..." },
  "object": {
    "@type": "SocialMediaPosting",
    "@id": "/p2p/a12345.../post.json.pgp"
  },
  "startTime": "2026-03-08T13:00:00Z"
}
```

**FollowAction** - Following users
```json
{
  "@context": "https://schema.org",
  "@type": "FollowAction",
  "agent": { "@type": "Person", "identifier": "5d000b..." },
  "object": { "@type": "Person", "identifier": "a12345..." },
  "startTime": "2026-03-08T14:00:00Z"
}
```

**ShareAction** - Sharing/reposting content
```json
{
  "@context": "https://schema.org",
  "@type": "ShareAction",
  "agent": { "@type": "Person", "identifier": "5d000b..." },
  "object": {
    "@type": "Article",
    "@id": "/p2p/a12345.../article.json.pgp"
  },
  "startTime": "2026-03-08T15:00:00Z"
}
```

### Multilingual Content

JSON-LD supports multiple languages:

```json
{
  "@context": "https://schema.org",
  "@type": "Article",
  "headline": {
    "@value": "Hello World",
    "@language": "en"
  },
  "alternativeHeadline": [
    { "@value": "مرحبا بالعالم", "@language": "ar" },
    { "@value": "Hola Mundo", "@language": "es" }
  ]
}
```

### Extending Schema.org

You can add custom properties or use other vocabularies:

```json
{
  "@context": [
    "https://schema.org",
    {
      "mau": "https://mau-network.org/vocab/",
      "replyCount": "mau:replyCount",
      "visibility": "mau:visibility"
    }
  ],
  "@type": "SocialMediaPosting",
  "headline": "Custom fields example",
  "mau:replyCount": 5,
  "mau:visibility": "friends-only"
}
```

**Guidelines:**
- Start with Schema.org types when possible
- Add custom properties with namespaces
- Document your custom vocabulary
- Expect clients to ignore unknown properties

### JSON-LD Tools

**Validation:**
- [JSON-LD Playground](https://json-ld.org/playground/) - Test and visualize
- [Schema.org validator](https://validator.schema.org/) - Check Schema.org compliance

**Libraries:**
- Go: `github.com/piprate/json-gold`
- JavaScript: `jsonld.js`
- Python: `PyLD`

---

## Content Versioning

### Why Versioning?

When you edit a file, Mau keeps the old version. This enables:

- **History** - See how content evolved
- **Sync conflict resolution** - Compare versions
- **Rollback** - Restore old versions
- **Audit trails** - Who changed what when

### Version Storage

Old versions are stored in a `.versions/` subdirectory:

```
recipe-pasta.json.pgp
recipe-pasta.json.pgp.versions/
├── a7f3b9e2d1c4f8a9b3e7d2c5f1a8b4e9d3c7f2a6  # SHA-256 hash
├── d2c1e8f5a3b9d7c2e6f1a4b8d3c9f7e2a5b1d6  # Older version
└── f1a8b4e9d3c7f2a6b5e1d9c3f8a2b6e4d1c5  # Oldest version
```

**Version filename:** SHA-256 hash of the file contents (hex, lowercase)

### Addressing Versions

Full version address:

```
/p2p/<user-fingerprint>/<filename>.versions/<version-hash>
```

**Example:**
```
/p2p/5d000b.../recipe-pasta.json.versions/a7f3b9e2d1c4f8a9b3e7d2c5f1a8b4e9d3c7f2a6
```

### Creating Versions

When you update a file:

1. Read the current file
2. Compute its SHA-256 hash
3. Move current file to `.versions/<hash>`
4. Write the new content to the original filename

**Example (pseudocode):**
```go
// Update recipe-pasta.json.pgp
currentContent := readFile("recipe-pasta.json.pgp")
hash := sha256(currentContent)
mkdir("recipe-pasta.json.pgp.versions")
move("recipe-pasta.json.pgp", "recipe-pasta.json.pgp.versions/" + hash)
writeFile("recipe-pasta.json.pgp", newContent)
```

### Version Metadata

Include version references in the current file:

```json
{
  "@context": "https://schema.org",
  "@type": "Recipe",
  "name": "Pasta Carbonara (updated)",
  "version": "2",
  "previousVersion": {
    "@id": "/p2p/5d000b.../recipe-pasta.json.versions/a7f3b9..."
  },
  "dateModified": "2026-03-08T16:00:00Z"
}
```

### Version Cleanup

Old versions accumulate. Consider:

- **Keep recent** - Last 5-10 versions
- **Age-based** - Delete versions older than 1 year
- **Manual pruning** - Let users delete old versions

**Cleanup script example:**
```bash
# Keep only the 5 most recent versions
cd recipe-pasta.json.pgp.versions/
ls -t | tail -n +6 | xargs rm
```

---

## File Encryption and Signing

### PGP Encryption

Every file is encrypted with PGP before storage:

**Public content** (encrypted for yourself, readable by anyone):
```bash
gpg --encrypt --recipient 5D000B2F... --output hello-world.json.pgp hello-world.json
```

**Private content** (encrypted for yourself only):
```bash
gpg --encrypt --recipient 5D000B2F... --output draft.json.pgp draft.json
```

**Shared content** (encrypted for multiple recipients):
```bash
gpg --encrypt \
  --recipient 5D000B2F... \  # Yourself
  --recipient A1234567... \  # Friend 1
  --recipient B9876543... \  # Friend 2
  --output message.json.pgp message.json
```

### Signing

All files should be **signed** to prevent tampering:

```bash
gpg --sign --encrypt --recipient 5D000B2F... --output post.json.pgp post.json
```

**Benefits:**
- **Authenticity** - Proves you created the content
- **Integrity** - Detects tampering
- **Non-repudiation** - Can't deny authorship

### Decryption

Decrypt and verify:

```bash
gpg --decrypt --output post.json post.json.pgp
```

If verification fails, the file has been tampered with.

### Privacy Levels

Encryption determines visibility:

| Encrypted For | Visibility | Use Case |
|---------------|------------|----------|
| Yourself only | Private | Drafts, personal notes |
| Yourself + friends | Friends-only | Private messages, shared photos |
| Anyone (public key available) | Public | Blog posts, status updates |

**Note:** "Public" content is still encrypted—but anyone can request it via the HTTP API (see [07-http-api.md](07-http-api.md)).

---

## Working with Files

### Reading Files

**Go example:**
```go
import (
    "github.com/mau-network/mau"
    "encoding/json"
)

// Read a file
content, err := mau.ReadFile("5d000b.../hello-world.json.pgp")
if err != nil {
    log.Fatal(err)
}

// Decrypt (automatic if you have the key)
decrypted, err := mau.Decrypt(content)
if err != nil {
    log.Fatal(err)
}

// Parse JSON-LD
var post map[string]interface{}
json.Unmarshal(decrypted, &post)
fmt.Println(post["headline"])  // "Hello, decentralized world!"
```

### Writing Files

**Go example:**
```go
import (
    "github.com/mau-network/mau"
    "encoding/json"
)

// Create content
post := map[string]interface{}{
    "@context": "https://schema.org",
    "@type":    "SocialMediaPosting",
    "headline": "New post",
    "datePublished": time.Now().Format(time.RFC3339),
}

// Serialize
data, _ := json.Marshal(post)

// Encrypt and sign
encrypted, err := mau.Encrypt(data, "5d000b...")
if err != nil {
    log.Fatal(err)
}

// Write to file
mau.WriteFile("5d000b.../new-post.json.pgp", encrypted)
```

### Listing Files

**Bash example:**
```bash
# List all your files
ls ~/.mau/5d000b2f2c040a1675b49d7f0c7cb7dc36999d56/

# List friend's files
ls ~/.mau/a1234567890abcdef1234567890abcdef12345678/

# Count posts
ls ~/.mau/5d000b.../*.pgp | wc -l
```

**Go example:**
```go
files, err := mau.ListFiles("5d000b...")
for _, file := range files {
    fmt.Println(file.Name, file.Size, file.Modified)
}
```

### Deleting Files

**Bash:**
```bash
# Delete a file (and its versions)
rm ~/.mau/5d000b.../old-post.json.pgp
rm -rf ~/.mau/5d000b.../old-post.json.pgp.versions/
```

**Go:**
```go
err := mau.DeleteFile("5d000b.../old-post.json.pgp")
```

**Note:** Deletion is local. If peers already synced the file, they keep their copy.

---

## Best Practices

### File Organization

**DO:**
- Use descriptive filenames (`recipe-carbonara.json`, not `recipe1.json`)
- Group related content with prefixes (`photo-2026-03-08-beach.json`)
- Keep filename length reasonable (<100 chars)
- Use lowercase with hyphens (`my-post.json`, not `My_Post.json`)

**DON'T:**
- Use random IDs as filenames (defeats human-readability)
- Include spaces or special characters
- Reuse filenames for different content

### Content Structure

**DO:**
- Always include `@context` and `@type`
- Use ISO 8601 for dates (`2026-03-08T10:30:00Z`)
- Include `author` with fingerprint
- Add `datePublished` / `dateCreated`

**DON'T:**
- Omit required Schema.org fields
- Use non-standard date formats
- Forget to include author information

### Versioning

**DO:**
- Keep versions for important content (articles, photos)
- Clean up old versions periodically
- Reference previous versions in metadata

**DON'T:**
- Keep every version forever (disk space)
- Delete all versions (lose history)

### Encryption

**DO:**
- Sign all files (prevents tampering)
- Encrypt private content properly
- Test decryption before deleting plaintext

**DON'T:**
- Leave plaintext versions on disk
- Forget to encrypt private messages
- Share your private key

### Performance

**DO:**
- Index files for fast search
- Cache decrypted content when appropriate
- Use file watchers for real-time updates

**DON'T:**
- Decrypt every file on every sync
- Load all files into memory
- Ignore filesystem limits (millions of files in one directory)

### Backup

**DO:**
- Regularly backup your user directory
- Keep encrypted backups off-site
- Test restoring from backups

**DON'T:**
- Backup only plaintext (defeats encryption)
- Forget to backup `.mau/` metadata
- Assume the filesystem is reliable

---

## Next Steps

Now that you understand storage and data formats:

1. **Learn authentication** - [05-authentication.md](05-authentication.md) - How PGP signing/encryption works
2. **Explore networking** - [06-networking.md](06-networking.md) - How peers discover and sync
3. **Build an app** - [08-building-social-apps.md](08-building-social-apps.md) - Practical patterns

For complete API reference, see [11-api-reference.md](11-api-reference.md).

---

*Questions? Check [13-troubleshooting.md](13-troubleshooting.md) or open an issue on [GitHub](https://github.com/mau-network/mau/issues).*
