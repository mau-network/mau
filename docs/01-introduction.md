# Introduction to Mau

## The Problem with Modern Social Media

Today's social applications have fundamental issues:

### Centralization
Your messages to someone in the next room travel thousands of miles to a data center and back. Your data lives on someone else's computer, not yours.

### Walled Gardens
You can't move your data between platforms. You duplicate the same content across Facebook, Twitter, Instagram—each locked in its own silo.

### Privacy Violations
Companies track everything you do, leak your data, and use it to manipulate elections and arrest people.

### Limited Innovation
Want to post a recipe with structured ingredients? A bike ride with GPS data? A mathematical equation? You can't. Platforms only support what they've built.

### Censorship
One company's "community guidelines" apply to billions of people across different cultures, countries, and values. They act as Big Brother, deciding what you can say.

### Algorithm Manipulation
What you see is controlled by secret algorithms optimized for engagement, not truth or your actual interests.

## The Mau Solution

Mau is a **convention**, not a platform. It defines how peer-to-peer social applications should work using:

- **Filesystem storage** - Your data is files on your disk
- **PGP encryption** - End-to-end privacy, signed authenticity
- **Kademlia DHT** - Decentralized peer discovery
- **HTTP/TLS** - Simple, web-compatible data exchange
- **JSON-LD + Schema.org** - Structured, extensible content

### What Makes Mau Different

| Traditional Social Media | Mau |
|-------------------------|-----|
| Data on company servers | Data on your disk |
| One app controls everything | Any app can access your data |
| Platform decides features | Developers innovate freely |
| Privacy by trust | Privacy by cryptography |
| Central servers | Peer-to-peer |
| Unstructured text | Structured content (recipes, events, reviews...) |

## How Mau Works

### 1. Everything is a File
When you post something, it's saved as a **file** in a directory on your computer:

```
~/.mau/
  alice-FPR/           # Your posts
    hello-world.json.pgp
    recipe-lasagna.json.pgp
  bob-FPR/             # Bob's posts you've synced
    comment-on-hello.json.pgp
```

- Each file is a JSON-LD document (structured content)
- Encrypted with PGP (private key = your identity)
- Signed to prevent tampering

### 2. Structured Content with Schema.org
Instead of plain text, you create **typed objects**:

```json
{
  "@context": "https://schema.org",
  "@type": "Recipe",
  "name": "Mom's Lasagna",
  "recipeIngredient": ["pasta", "tomato sauce", "cheese"],
  "recipeInstructions": "Layer and bake at 180°C for 45 minutes"
}
```

This means:
- Apps understand what the content *is* (recipe, review, event...)
- Specialized apps can display it properly
- Search and filtering work semantically

### 3. PGP for Identity & Privacy
Your identity is your **PGP key fingerprint**. When you create a post:

1. **Sign it** - Proves it's from you, prevents tampering
2. **Encrypt it** - For public posts (your key), private messages (recipient's key), or groups (multiple keys)

```bash
# Public post
echo '{"@type":"SocialMediaPosting"...}' | gpg --encrypt --sign -r alice-FPR

# Private message
echo '{"@type":"Message"...}' | gpg --encrypt --sign -r bob-FPR
```

### 4. Peer Discovery with Kademlia
How do you find your friends on a decentralized network?

- **mDNS** - Discover peers on the same WiFi/LAN
- **Kademlia DHT** - Distributed hash table for internet-wide discovery
- **Direct addresses** - Exchange IP/domain manually (like phone numbers)

When you follow someone:
1. You get their public key fingerprint
2. You query the DHT: "Where is `bob-FPR`?"
3. The network returns Bob's IP address
4. You connect directly to Bob

### 5. Data Exchange over HTTP
Each peer runs a simple HTTP server:

```
GET /alice-FPR                     → List of Alice's files
GET /alice-FPR/hello-world.json    → Download specific file
GET /alice-FPR/hello-world.json.versions/abc123  → Old version
```

- **If-Modified-Since** header for incremental sync
- **TLS mutual auth** - Both peers prove identity
- **Range requests** - Resumable downloads for large files

### 6. Sync is Automatic
Mau clients periodically:
1. Ask friends: "What changed since last time?"
2. Download new/updated files
3. Verify signatures
4. Decrypt and store locally

You can read everything offline. Sync when back online.

## Example: Alice Posts, Bob Comments

1. **Alice creates a post:**
   ```bash
   # Alice writes a post
   echo '{"@type":"SocialMediaPosting","headline":"Loving Mau!"}' \
     | gpg --encrypt --sign -r alice-FPR \
     > ~/.mau/alice-FPR/post-2026-02-27.json.pgp
   ```

2. **Bob syncs and sees it:**
   ```bash
   # Bob's client requests: GET /alice-FPR?since=2026-02-26
   # Downloads post-2026-02-27.json.pgp
   # Verifies Alice's signature
   # Decrypts and displays in Bob's feed
   ```

3. **Bob comments:**
   ```bash
   # Bob creates a comment referencing Alice's post
   echo '{"@type":"Comment","about":"/p2p/alice-FPR/post-2026-02-27"}' \
     | gpg --encrypt --sign -r bob-FPR \
     > ~/.mau/bob-FPR/comment-on-alice.json.pgp
   ```

4. **Alice syncs and sees the comment:**
   ```bash
   # Alice's client: GET /bob-FPR?since=last-sync
   # Downloads comment-on-alice.json.pgp
   # Displays under her original post
   ```

No central server. No company. Just files, crypto, and HTTP.

## What You Can Build

Mau is a **protocol**, not an application. You can build:

### Communication
- **Chat apps** - WhatsApp-like messaging
- **Forums** - Reddit-style discussions
- **Email replacement** - Encrypted, peer-to-peer

### Content Platforms
- **Blogs** - Personal websites that sync
- **Photo sharing** - Instagram without ads
- **Video platforms** - YouTube alternative

### Specialized Networks
- **Recipe sharing** - Structured cooking data
- **Event coordination** - Meetups, conferences
- **Review platforms** - Yelp/TripAdvisor alternative
- **Code collaboration** - GitHub issues/PRs over Mau

### IoT & Smart Devices
- **Home automation** - Control devices via Mau messages
- **Sensor networks** - Temperature, security cameras
- **Wearables** - Fitness data, health tracking

### Creative Use Cases
- **Multiplayer games** - Chess, poker, turn-based strategy
- **Collaborative editing** - Google Docs alternative
- **Music sharing** - Bandcamp-style artist platforms

## Key Benefits for Developers

### 1. Simple Stack
- Files (you already know them)
- HTTP (you already know it)
- PGP (libraries exist for every language)
- JSON (universal format)

No blockchain. No complex consensus. No tokens.

### 2. Instant Compatibility
Your app shares data with **all other Mau apps**. A blog post you create can be:
- Read in a Mau chat client
- Indexed by a Mau search engine
- Backed up by a Mau backup service
- Displayed on a Mau website

### 3. Web Interoperability
Existing websites using **JSON-LD** (millions already do for SEO) can become Mau peers by:
1. Adding an HTTP endpoint
2. Serving their structured data
3. Implementing TLS mutual auth

### 4. No Gatekeepers
- No app store approval
- No API rate limits
- No platform bans
- No monetization cuts

You build. Users install. That's it.

## Core Principles

### 1. User Data Ownership
Files live on the user's disk. They can:
- Back up with `tar`
- Delete with `rm`
- Edit with text editors
- Move to any Mau-compatible app

### 2. Privacy by Design
- End-to-end encryption by default
- You can't read users' private data (even if you tried)
- No tracking, no analytics, no surveillance

### 3. Simplicity over Features
- Small spec, easy to implement
- No unnecessary complexity
- Build on proven standards

### 4. Evolution-Friendly
- Schema.org vocabulary expands over time
- New content types don't break old clients
- Protocol upgrades (HTTP/3, QUIC) work automatically

### 5. Censorship Resistance
- No central authority to shut down
- Users control their own data and feeds
- Peer-to-peer means unstoppable

## What Mau is NOT

- **Not a blockchain** - No mining, no tokens, no distributed consensus
- **Not a platform** - It's a convention; you build the apps
- **Not ActivityPub** - Different approach, though they can interoperate
- **Not a company** - It's an open specification (GPL v3)

## Next Steps

Now that you understand *why* Mau exists and *what* it does, let's dive into the technical details:

👉 **[Core Concepts](02-core-concepts.md)** - Deep dive into storage, encryption, and networking

---

*"The best way to predict the future is to invent it." — Alan Kay*
