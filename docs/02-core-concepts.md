# Core Concepts

This guide covers the fundamental building blocks of Mau. Understanding these concepts is essential for building Mau applications.

## Table of Contents
1. [Storage Model](#storage-model)
2. [Data Format](#data-format)
3. [Identity & Authentication](#identity--authentication)
4. [Addressing Scheme](#addressing-scheme)
5. [Peer-to-Peer Architecture](#peer-to-peer-architecture)

---

## Storage Model

### Everything is a File

Mau adopts the Unix philosophy: **everything is a file**. This radical simplicity has profound implications.

#### Directory Structure

```
~/.mau/                                    # Root directory
├── .mau/                                  # Reserved metadata directory
│   ├── account.pgp                        # Your encrypted private key
│   ├── alice-FPR.pgp                      # Friend's public key
│   ├── bob-FPR.pgp                        # Friend's public key
│   └── coworkers/                         # Key groups (keyrings)
│       └── charlie-FPR.pgp
│
├── alice-FPR/                             # Your posts (alice = you)
│   ├── hello-world.json.pgp              # Post file
│   ├── recipe-lasagna.json.pgp           # Another post
│   └── recipe-lasagna.json.pgp.versions/ # Version history
│       └── abc123...xyz.pgp              # Old version (SHA256 hash)
│
├── bob-FPR/                               # Bob's synced posts
│   ├── comment-on-hello.json.pgp
│   └── status-update.json.pgp
│
└── .charlie-FPR/                          # Hidden = unfollowed (kept for history)
    └── old-post.json.pgp
```

### Key Rules

1. **User directories are named by fingerprint**
   - Your posts: `<your-fingerprint>/`
   - Friends' posts: `<friend-fingerprint>/`

2. **Files use `.pgp` extension**
   - All content is PGP-encrypted
   - File format: OpenPGP Message Format (RFC 4880)

3. **Hidden directories (`.prefix`) are ignored**
   - Unfollow someone? Prefix their directory with `.`
   - Data is preserved, but clients won't sync it

4. **`.mau/` is reserved**
   - Stores keys and metadata
   - `account.pgp` = your encrypted private key
   - Other files = friends' public keys

5. **Versioning uses subdirectories**
   - Format: `<filename>.versions/<sha256-hash>.pgp`
   - Keeps history for contracts, signed documents, etc.

### Why Files?

This design gives you:

✅ **Backup with standard tools:** `tar czf backup.tar.gz ~/.mau`  
✅ **Restore from backup:** Extract and you're back  
✅ **Delete everything:** `rm -rf ~/.mau`  
✅ **Inspect with text tools:** `ls`, `grep`, `find`  
✅ **Network file systems:** Store on NAS, Dropbox, etc.  
✅ **Multiple clients:** Terminal app, GUI, web interface—all access the same files  

---

## Data Format

### JSON-LD + Schema.org

Mau uses **JSON-LD** (JSON for Linked Data) with **Schema.org** vocabulary.

#### Why JSON-LD?

- **Structured content** - Not just text, but typed objects (recipes, events, reviews...)
- **Extensible** - Add new types without breaking old clients
- **Web-compatible** - Millions of websites already use it for SEO
- **Developer-friendly** - Just JSON with special `@` keys

#### Basic Example

```json
{
  "@context": "https://schema.org",
  "@type": "SocialMediaPosting",
  "headline": "Loving peer-to-peer social media!",
  "articleBody": "Mau gives me full control over my data.",
  "author": {
    "@type": "Person",
    "name": "Alice",
    "identifier": "abc123...xyz"
  },
  "datePublished": "2026-02-27T10:00:00Z"
}
```

#### Schema.org Types

Common types for social apps:

| Type | Use Case |
|------|----------|
| `SocialMediaPosting` | Status updates, tweets |
| `Comment` | Replies to posts |
| `Message` | Private messages |
| `Recipe` | Cooking instructions |
| `Review` | Product/place reviews |
| `Event` | Meetups, conferences |
| `Article` | Blog posts |
| `ImageObject` | Photos |
| `VideoObject` | Videos |
| `FollowAction` | Follow someone |
| `LikeAction` | Like a post |

See [Schema.org full list](https://schema.org/docs/full.html) for 800+ types.

#### Actions and References

Reference other content using Mau addresses:

```json
{
  "@context": "https://schema.org",
  "@type": "Comment",
  "text": "Great post!",
  "about": "/p2p/alice-FPR/hello-world.json",
  "author": {
    "@type": "Person",
    "name": "Bob"
  }
}
```

Like someone's post:

```json
{
  "@context": "https://schema.org",
  "@type": "LikeAction",
  "agent": {
    "@type": "Person",
    "identifier": "bob-FPR"
  },
  "object": "/p2p/alice-FPR/hello-world.json"
}
```

#### Multiple Languages

Schema.org supports localization:

```json
{
  "@context": "https://schema.org",
  "@type": "Article",
  "headline": {
    "@language": "en",
    "@value": "Hello World"
  },
  "alternativeHeadline": {
    "@language": "ar",
    "@value": "مرحبا بالعالم"
  }
}
```

#### Binary Attachments

For images/videos, use `encodingFormat` and `contentUrl`:

```json
{
  "@context": "https://schema.org",
  "@type": "ImageObject",
  "contentUrl": "photo.jpg",
  "encodingFormat": "image/jpeg",
  "caption": "Sunset in Berlin"
}
```

The binary file (`photo.jpg`) lives next to the JSON file:
```
alice-FPR/
├── vacation-post.json.pgp
└── vacation-post-photo.jpg.pgp  # Referenced as "photo.jpg" in JSON
```

---

## Identity & Authentication

### PGP Keys as Identity

Your **identity** in Mau is your PGP public key fingerprint.

#### Fingerprint Format

```
5D000B2F2C040A1675B49D7F0C7CB7DC36999D56
└────────── 160 bits (40 hex chars) ──────────┘
```

- Derived from your public key
- Globally unique
- Used as your "username" everywhere

#### Key Generation

```bash
# Generate a new identity
gpg --full-generate-key

# Choose:
# - RSA (or ECC for smaller keys)
# - 4096 bits (or 256-bit curve for ECC)
# - No expiration (or set expiration for security)
# - Passphrase to protect private key

# Export public key
gpg --export alice@example.com > ~/.mau/.mau/account.pub.pgp

# Export private key (encrypted with passphrase)
gpg --export-secret-keys alice@example.com > ~/.mau/.mau/account.pgp
```

#### Signing Content

Every post is **signed** to prove:
1. It came from you (authenticity)
2. It wasn't tampered with (integrity)

```bash
# Sign and encrypt a post for public viewing
echo '{"@type":"SocialMediaPosting",...}' \
  | gpg --sign --encrypt -r alice-FPR \
  > hello-world.json.pgp
```

When others download your post, they:
1. Decrypt it (if they're a recipient)
2. Verify your signature
3. See your fingerprint as the author

#### Encryption for Privacy

**Public post** (anyone can read):
```bash
gpg --encrypt --sign -r alice-FPR < post.json > post.json.pgp
```
(Encrypted to *your own key*, signed by you)

**Private message** (only Bob can read):
```bash
gpg --encrypt --sign -r bob-FPR < message.json > message.json.pgp
```

**Group message** (Alice, Bob, and Charlie):
```bash
gpg --encrypt --sign -r alice-FPR -r bob-FPR -r charlie-FPR < group-msg.json > group-msg.json.pgp
```

#### Friend's Keys

When you follow someone:
1. Get their public key fingerprint (e.g., via QR code, website, email)
2. Download their public key
3. Save to `~/.mau/.mau/<friend-fingerprint>.pgp`
4. Encrypt it with *your* key (proves you added them, not malware)

```bash
gpg --export bob-FPR | gpg --encrypt -r alice-FPR > ~/.mau/.mau/bob-FPR.pgp
```

---

## Addressing Scheme

### Mau URLs

Everything in Mau has a hierarchical address:

```
/p2p/<user-fingerprint>/<filename>
```

#### Examples

| Address | Meaning |
|---------|---------|
| `/p2p/alice-FPR` | List all of Alice's files |
| `/p2p/alice-FPR/hello-world.json` | Alice's "hello-world" post |
| `/p2p/alice-FPR/recipe-lasagna.json.versions/abc123` | Old version of Alice's recipe |

### File Listing

Request: `GET /p2p/alice-FPR`

Response:
```json
[
  {
    "name": "hello-world.json",
    "size": 1024,
    "sha256": "abc123...",
    "modified": "2026-02-27T10:00:00Z"
  },
  {
    "name": "recipe-lasagna.json",
    "size": 2048,
    "sha256": "def456...",
    "modified": "2026-02-26T15:30:00Z"
  }
]
```

### Incremental Sync

Use `If-Modified-Since` header:

```http
GET /p2p/alice-FPR
If-Modified-Since: Wed, 26 Feb 2026 00:00:00 GMT
```

Server returns only files modified *after* that date.

### Versioning

When a file changes:
1. Old content moves to `<filename>.versions/<sha256-hash>.pgp`
2. New content overwrites `<filename>.pgp`

Request old version:
```http
GET /p2p/alice-FPR/recipe-lasagna.json.versions/abc123
```

---

## Peer-to-Peer Architecture

### Components

Mau peers run three main modules:

```
┌──────────────────────────────────────────────┐
│             Mau Peer                          │
│                                               │
│  ┌─────────────┐  ┌──────────────┐          │
│  │HTTP Server  │  │HTTP Client   │          │
│  │(Serve files)│  │(Sync files)  │          │
│  └─────────────┘  └──────────────┘          │
│                                               │
│  ┌────────────────────────────────┐          │
│  │  Peer Discovery                │          │
│  │  - Kademlia DHT                │          │
│  │  - mDNS (local network)        │          │
│  └────────────────────────────────┘          │
│                                               │
│  ┌────────────────────────────────┐          │
│  │  Storage Layer (Filesystem)    │          │
│  └────────────────────────────────┘          │
└──────────────────────────────────────────────┘
```

### 1. HTTP Server

Exposes your content:

- `GET /<fingerprint>` → File list
- `GET /<fingerprint>/<filename>` → Download file
- `GET /<fingerprint>/<filename>.versions/<hash>` → Download version

**Authentication:** TLS 1.3 with client certificates (mutual TLS)
- Server presents certificate with its fingerprint
- Client presents certificate with its fingerprint
- Both verify each other's identity

### 2. HTTP Client

Syncs from friends:
1. For each friend in `.mau/`
2. Request: `GET /<friend-fpr>?since=<last-sync>`
3. Download new/updated files
4. Verify signatures
5. Save to `<friend-fpr>/`

### 3. Peer Discovery

#### mDNS (Local Network)

Broadcasts on LAN:
```
5D000B2F2C040A1675B49D7F0C7CB7DC36999D56._mau._tcp.local.
```

Announces:
- Fingerprint (service name)
- IP + Port (address)
- Protocol (`_mau._tcp`)

Peers on the same WiFi discover each other instantly.

#### Kademlia DHT (Internet)

Distributed hash table for finding peers anywhere:

1. **Bootstrap:** Connect to a few known peers
2. **PING:** Check if a node is alive
3. **FIND_NODE:** Recursively search for a target fingerprint

Simplified Kademlia (Mau doesn't store arbitrary values):
- `/kad/ping` → "Are you alive?"
- `/kad/find_peer/<fingerprint>` → "Where is this person?"

Returns:
```json
[
  {
    "fingerprint": "bob-FPR",
    "addresses": ["192.168.1.100:8080", "bob.example.com:443"]
  },
  {
    "fingerprint": "charlie-FPR",
    "addresses": ["charlie.example.com:443"]
  }
]
```

### 4. NAT Traversal

For peers behind firewalls:
- **UPnP** - Automatically open ports on router
- **NAT-PMP** - Apple's NAT traversal protocol
- **Hole punching** - Coordinate with a relay peer

### Data Flow Example

Alice wants to sync Bob's posts:

```
┌─────────┐                          ┌─────────┐
│  Alice  │                          │   Bob   │
└────┬────┘                          └────┬────┘
     │                                     │
     │ 1. Find Bob                         │
     ├─────── /kad/find_peer/bob-FPR ─────>│
     │<─────── {addresses: [bob.com:443]}──┤
     │                                     │
     │ 2. Connect with mutual TLS          │
     ├─────── TLS Handshake ───────────────>│
     │  (Alice's cert + Bob's cert)        │
     │                                     │
     │ 3. Request file list                │
     ├─────── GET /bob-FPR?since=...  ─────>│
     │<─────── [{name: "new-post.json"}]───┤
     │                                     │
     │ 4. Download file                    │
     ├─────── GET /bob-FPR/new-post.json ──>│
     │<─────── <encrypted content> ─────────┤
     │                                     │
     │ 5. Verify & save locally            │
     │  - Check Bob's signature            │
     │  - Decrypt                          │
     │  - Save to ~/.mau/bob-FPR/          │
     │                                     │
```

---

## Design Principles

### 1. Simplicity
- Files, not databases
- HTTP, not custom protocols
- PGP, not new crypto

### 2. Interoperability
- Schema.org = web-compatible
- HTTP = works with proxies, CDNs, browsers
- OpenPGP = any implementation works

### 3. Privacy by Default
- End-to-end encryption
- Can't read others' private data
- No metadata leaks (TLS hides content)

### 4. Decentralization
- No central servers
- No single point of failure
- No gatekeepers

### 5. Evolvable
- New Schema.org types → new features
- HTTP/3 upgrade → free performance
- Algorithm upgrades (PGP keys) → user's choice

---

## Next Steps

Now that you understand the core concepts, let's build something:

👉 **[Quick Start Guide](03-quickstart.md)** - Build your first Mau app in 15 minutes

For deeper dives:
- **[Storage and Data Format](04-storage-and-data.md)** - Detailed file format specs
- **[Authentication & Encryption](05-authentication.md)** - PGP deep dive
- **[Peer-to-Peer Networking](06-networking.md)** - Kademlia and discovery
