# Quick Start: Mau CLI

This tutorial shows how to use the **Mau command-line tool** for practical peer-to-peer social networking. It handles all the GPG complexity for you.

**Time:** ~10 minutes  
**Prerequisites:** 
- GPG installed
- Go 1.21+ installed
- Completed [GPG Fundamentals](03a-quickstart-gpg.md) (recommended)

## Step 1: Build the Mau CLI

```bash
# Clone the repository
git clone https://github.com/mau-network/mau.git
cd mau

# Build the CLI
go build -o mau ./cmd/mau

# Verify it works
./mau
```

You should see:
```
Available commands:
	init:     Initialize new account in current directory
	show:     Show current account information
	export:   Export account public key to file
	friend:   Add a friend using this public key file
	...
```

## Step 2: Initialize Your Account

Create a new Mau account in the current directory:

```bash
mkdir ~/my-mau
cd ~/my-mau

# Initialize (you'll be prompted for name, email, passphrase)
../mau/mau init
```

Follow the prompts:
```
Enter your name: Alice
Enter your email: alice@example.com
Enter passphrase: [your-secure-password]
Confirm passphrase: [your-secure-password]

✓ Account created successfully!
Fingerprint: 5D000B2F2C040A1675B49D7F0C7CB7DC36999D56
```

**What just happened?**
- Created PGP key pair
- Created `.mau/` directory structure
- Saved encrypted private key to `.mau/account.pgp`
- Your identity is now the fingerprint

View your account info:

```bash
../mau/mau show
```

Output:
```
Name:        Alice
Email:       alice@example.com
Fingerprint: 5D000B2F2C040A1675B49D7F0C7CB7DC36999D56
Directory:   /home/alice/my-mau
Friends:     0
Follows:     0
Files:       0
```

## Step 3: Share Your First Post

Create a JSON file with your post:

```bash
cat > my-first-post.json <<'EOF'
{
  "@context": "https://schema.org",
  "@type": "SocialMediaPosting",
  "headline": "Hello from Mau CLI!",
  "articleBody": "Just set up my decentralized social profile.",
  "datePublished": "2026-02-28T08:00:00Z"
}
EOF
```

Share it with your followers:

```bash
../mau/mau share my-first-post.json
```

Output:
```
✓ File shared: my-first-post.json
  Size: 234 bytes
  Encrypted for: yourself (public post)
  Saved to: 5D000B2F.../my-first-post.json.pgp
```

**What happened?**
- Mau read the JSON file
- Signed it with your private key
- Encrypted it for yourself (public post)
- Saved to your fingerprint directory

List your shared files:

```bash
../mau/mau files
```

Output:
```
📝 Your shared files:

my-first-post.json.pgp
  Size:     389 bytes
  Modified: 2026-02-28 08:00:15
```

## Step 4: Add a Friend

Let's simulate adding Bob. First, Bob needs to:
1. Run `mau init` on his machine
2. Export his public key
3. Send it to you

**Bob's side:**
```bash
# Bob initializes his account
cd ~/bob-mau
../mau/mau init
# (Name: Bob, Email: bob@example.com, passphrase: ...)

# Bob exports his public key
../mau/mau export bob-pubkey.pgp

# Bob sends bob-pubkey.pgp to Alice (email, USB, etc.)
```

**Your side (Alice):**
```bash
# You receive bob-pubkey.pgp and add Bob as a friend
../mau/mau friend bob-pubkey.pgp
```

Output:
```
✓ Friend added: Bob <bob@example.com>
  Fingerprint: ABC123DEF456789ABC123DEF456789ABC123DEF
```

Verify Bob is in your friends list:

```bash
../mau/mau friends
```

Output:
```
👥 Your friends:

Bob <bob@example.com>
  Fingerprint: ABC123DEF456789ABC123DEF456789ABC123DEF
  Following:   No
```

## Step 5: Follow Bob

Following means you want to sync Bob's posts:

```bash
../mau/mau follow ABC123DEF456789ABC123DEF456789ABC123DEF
```

Or shorter (Mau matches partial fingerprints):

```bash
../mau/mau follow ABC123
```

Output:
```
✓ Now following: Bob <bob@example.com>
```

Check your follows:

```bash
../mau/mau follows
```

Output:
```
👤 You are following:

Bob <bob@example.com>
  Fingerprint: ABC123DEF456789ABC123DEF456789ABC123DEF
  Last sync:   Never
```

## Step 6: Start the HTTP Server

To allow others to sync your posts, run a server:

```bash
../mau/mau serve
```

Output:
```
🚀 Mau server starting...
   Address:     http://192.168.1.100:8080
   Fingerprint: 5D000B2F2C040A1675B49D7F0C7CB7DC36999D56

📡 Announcing on local network via mDNS...
   Service: 5D000B2F2C040A1675B49D7F0C7CB7DC36999D56._mau._tcp.local

✓ Server running. Press Ctrl+C to stop.
```

**What's happening?**
- HTTP server listening on port 8080
- mDNS broadcasting your presence on local network
- Peers can now discover and sync from you

Test the server (open another terminal):

```bash
# List your files
curl http://localhost:8080/5D000B2F2C040A1675B49D7F0C7CB7DC36999D56

# Download a specific file
curl http://localhost:8080/5D000B2F2C040A1675B49D7F0C7CB7DC36999D56/my-first-post.json.pgp \
  -o downloaded.pgp

# Decrypt it
gpg --decrypt downloaded.pgp
```

## Step 7: Sync from Bob

Assuming Bob is also running `mau serve` on his machine:

```bash
# Sync from Bob's server
../mau/mau sync http://bob-machine.local:8080 ABC123
```

Or if Bob is on the same network, Mau can discover him automatically:

```bash
# Auto-discover via mDNS and sync
../mau/mau sync ABC123
```

Output:
```
🔍 Discovering ABC123... on local network
✓ Found: Bob at http://192.168.1.101:8080

📥 Syncing from Bob...
   Checking files modified since: Never

   ⬇ bob-hello.json.pgp (412 bytes)
   ⬇ bob-recipe.json.pgp (1.2 KB)

✓ Synced 2 files from Bob
```

View Bob's files:

```bash
../mau/mau files ABC123
```

Output:
```
📝 Files from Bob <bob@example.com>:

bob-hello.json.pgp
  Size:     412 bytes
  Modified: 2026-02-28 07:30:00

bob-recipe.json.pgp
  Size:     1.2 KB
  Modified: 2026-02-28 07:45:00
```

## Step 8: Read Bob's Posts

Open and decrypt a file:

```bash
../mau/mau open ABC123 bob-hello.json.pgp
```

Output:
```json
{
  "@context": "https://schema.org",
  "@type": "SocialMediaPosting",
  "headline": "Just joined Mau!",
  "author": {
    "@type": "Person",
    "name": "Bob"
  },
  "datePublished": "2026-02-28T07:30:00Z"
}

✓ Signature verified: Bob <bob@example.com>
```

## Step 9: Share a Private Message to Bob

Create a private message:

```bash
cat > private-msg.json <<'EOF'
{
  "@context": "https://schema.org",
  "@type": "Message",
  "text": "Hey Bob, wanna grab coffee this weekend?",
  "dateSent": "2026-02-28T08:30:00Z"
}
EOF
```

Share it (encrypted only for Bob):

```bash
../mau/mau share private-msg.json --to ABC123
```

Output:
```
✓ File shared: private-msg.json
  Encrypted for: Bob <bob@example.com>
  Only Bob can read this message.
```

**What happened?**
- Mau encrypted the message with Bob's public key
- Only Bob's private key can decrypt it
- You can't even read it yourself!

When Bob syncs, he'll see the file and can decrypt it.

## Step 10: Delete a Post

Remove a file you shared:

```bash
../mau/mau delete my-first-post.json.pgp
```

Output:
```
⚠ Are you sure you want to delete my-first-post.json.pgp? (y/N): y
✓ File deleted: my-first-post.json.pgp
  Note: Peers who already synced will keep their copy.
```

**Important:** Deletion only removes it from your machine. Peers who already downloaded it will keep their copy (this is P2P - no central control!).

## Step 11: Unfollow and Remove Friends

Stop following Bob:

```bash
../mau/mau unfollow ABC123
```

Output:
```
✓ Unfollowed: Bob <bob@example.com>
  Note: Bob's files remain on your disk.
```

Remove Bob entirely (deletes his files too):

```bash
../mau/mau unfriend ABC123
```

Output:
```
⚠ This will delete Bob's directory and all his files. Continue? (y/N): y
✓ Removed friend: Bob <bob@example.com>
  Deleted: ABC123DEF.../
```

## Common Workflows

### Backup Your Account

```bash
# Export your private key
gpg --export-secret-keys --armor YOUR_FPR > my-account-backup.asc

# Store securely (password manager, USB drive, etc.)
```

### Share Multiple Files

```bash
../mau/mau share post1.json post2.json post3.json
```

### Sync from Multiple Friends

```bash
../mau/mau sync
# Syncs from all followed friends who are discoverable
```

### Run Server in Background

```bash
# Using nohup
nohup ../mau/mau serve > mau-server.log 2>&1 &

# Or with systemd (create a service file)
```

### Check File Integrity

```bash
# Verify signature without decrypting content
gpg --verify YOUR_FPR/some-file.json.pgp
```

## Advanced: Content Types

Mau supports any Schema.org type. Examples:

### Recipe

```json
{
  "@context": "https://schema.org",
  "@type": "Recipe",
  "name": "Spaghetti Carbonara",
  "recipeIngredient": ["400g spaghetti", "200g pancetta", "4 eggs"],
  "recipeInstructions": "Cook pasta. Fry pancetta. Mix eggs. Combine.",
  "totalTime": "PT30M"
}
```

### Event

```json
{
  "@context": "https://schema.org",
  "@type": "Event",
  "name": "Mau Meetup Berlin",
  "startDate": "2026-03-15T18:00:00+01:00",
  "location": {
    "@type": "Place",
    "name": "C-Base",
    "address": "Berlin, Germany"
  }
}
```

### Review

```json
{
  "@context": "https://schema.org",
  "@type": "Review",
  "itemReviewed": {
    "@type": "Restaurant",
    "name": "Burgermeister"
  },
  "reviewRating": {
    "@type": "Rating",
    "ratingValue": 5
  },
  "reviewBody": "Best burgers in Berlin!"
}
```

Just save as JSON and `mau share` them!

## Troubleshooting

### "Passphrase required" error

Make sure your GPG agent is running:
```bash
gpg-agent --daemon
```

### Can't sync from friend

1. Check they're running `mau serve`
2. Verify their IP/hostname is correct
3. Check firewall allows port 8080
4. Try mDNS discovery: `mau sync ABC123` (local network only)

### "Friend not found" error

Use full or partial fingerprint:
```bash
../mau/mau friends  # List all friends with fingerprints
../mau/mau follow <first-8-chars>
```

## What You've Learned

✅ Initialize Mau accounts  
✅ Share posts and files  
✅ Add and follow friends  
✅ Run HTTP server for syncing  
✅ Sync content from peers  
✅ Send private messages  
✅ Manage friends and follows  
✅ Use Schema.org types for structured content  

## Comparison with GPG Tutorial

| Task | Raw GPG | Mau CLI |
|------|---------|---------|
| Create identity | `gpg --full-generate-key` + manual export | `mau init` |
| Share post | `gpg --sign --encrypt ...` (long command) | `mau share file.json` |
| Add friend | `gpg --import` + manual file placement | `mau friend key.pgp` |
| Sync posts | Manual HTTP requests + decryption | `mau sync <fingerprint>` |
| Serve files | Set up HTTP server manually | `mau serve` |
| Private message | `gpg --encrypt -r ...` (manual recipient) | `mau share --to <fingerprint>` |

The CLI handles all complexity for you!

## Next Steps

- **[Mau Package Tutorial](03c-quickstart-package.md)** - Build custom applications with Go
- **[Building Social Apps](08-building-social-apps.md)** - Design patterns and best practices
- **Explore the GUI:** `go run ./gui` for a graphical interface

---

**Tip:** Set an alias for convenience:
```bash
echo 'alias mau="~/mau/mau"' >> ~/.bashrc
source ~/.bashrc
```
