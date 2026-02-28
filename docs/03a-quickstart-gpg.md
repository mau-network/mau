# Quick Start: GPG Fundamentals

This tutorial shows how Mau works at the lowest level using only GPG commands. Understanding these primitives will help you appreciate what Mau does behind the scenes.

**Time:** ~15 minutes  
**Prerequisites:** GPG installed (`gpg --version`)

## Why Start with GPG?

Mau is built on PGP/GPG. Before using Mau's tools, it's valuable to understand:
- How identity works (key pairs)
- How signing proves authenticity
- How encryption ensures privacy
- How the filesystem stores everything

## Step 1: Create Your Identity

Generate a PGP key pair:

```bash
gpg --full-generate-key
```

Follow the prompts:
- **Kind:** `(1) RSA and RSA`
- **Key size:** `4096` bits
- **Expiration:** `0` (no expiration) or your preference
- **Name:** Your real name or pseudonym
- **Email:** Your email address
- **Passphrase:** Strong password (you'll need this often)

Get your fingerprint:

```bash
gpg --fingerprint your-email@example.com
```

Output example:
```
pub   rsa4096 2026-02-28 [SC]
      5D00 0B2F 2C04 0A16 75B4  9D7F 0C7C B7DC 3699 9D56
uid           [ultimate] Alice <alice@example.com>
```

Export fingerprint without spaces:

```bash
export MY_FPR="5D000B2F2C040A1675B49D7F0C7CB7DC36999D56"
echo "My fingerprint: $MY_FPR"
```

**What just happened?**
- You created a **private key** (secret, stays on your machine)
- You created a **public key** (shareable, proves your identity)
- Your fingerprint is a unique 160-bit identifier (SHA-1 of public key)

## Step 2: Create a Mau Directory Structure

Mau uses a simple directory layout:

```bash
mkdir -p ~/.mau/$MY_FPR        # Your posts go here
mkdir -p ~/.mau/.mau           # Keys and metadata
```

Export your private key (encrypted with a passphrase):

```bash
gpg --export-secret-keys --armor $MY_FPR | \
  gpg --symmetric --armor --output ~/.mau/.mau/account.pgp
```

Enter a passphrase to protect this file. This is your **account backup**.

## Step 3: Create Your First Post

Create a social media post (JSON-LD with Schema.org vocabulary):

```bash
cat > /tmp/hello.json <<'EOF'
{
  "@context": "https://schema.org",
  "@type": "SocialMediaPosting",
  "headline": "Hello from the decentralized web!",
  "articleBody": "This post is encrypted, signed, and stored as a file on my disk.",
  "author": {
    "@type": "Person",
    "name": "Alice",
    "identifier": "5D000B2F2C040A1675B49D7F0C7CB7DC36999D56"
  },
  "datePublished": "2026-02-28T07:00:00Z"
}
EOF
```

**Sign and encrypt** the post (for yourself, making it public):

```bash
gpg --sign --encrypt --recipient $MY_FPR \
  --output ~/.mau/$MY_FPR/hello.json.pgp \
  /tmp/hello.json
```

**What just happened?**
1. **Sign:** GPG creates a signature using your private key
2. **Encrypt:** GPG encrypts for recipient(s) - in this case, yourself
3. **Output:** Binary `.pgp` file (OpenPGP message format)

Verify it exists:

```bash
ls -lh ~/.mau/$MY_FPR/hello.json.pgp
```

## Step 4: Read Your Post

Decrypt and verify:

```bash
gpg --decrypt ~/.mau/$MY_FPR/hello.json.pgp
```

Output:
```json
{
  "@context": "https://schema.org",
  "@type": "SocialMediaPosting",
  ...
}
gpg: Signature made Sat 28 Feb 2026 07:00:00 AM CET
gpg:                using RSA key 5D000B2F...
gpg: Good signature from "Alice <alice@example.com>" [ultimate]
```

**What does "Good signature" mean?**
- The file hasn't been tampered with
- It was definitely created by your private key
- The timestamp is authentic

## Step 5: Simulate a Friend (Bob)

Let's create a second identity to understand peer interaction:

```bash
# Generate Bob's key (use different email)
gpg --batch --passphrase '' --quick-generate-key "Bob <bob@example.com>" rsa4096

# Get Bob's fingerprint
export BOB_FPR=$(gpg --list-keys --with-colons bob@example.com | awk -F: '/^fpr:/ {print $10; exit}')
echo "Bob's fingerprint: $BOB_FPR"

# Create Bob's directory
mkdir -p ~/.mau/$BOB_FPR
```

## Step 6: Send a Private Message to Bob

Create a private message (encrypted only for Bob):

```bash
cat > /tmp/private-msg.json <<EOF
{
  "@context": "https://schema.org",
  "@type": "Message",
  "text": "Hey Bob, this is a private message only you can read!",
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
```

Encrypt for Bob (and sign with your key):

```bash
gpg --sign --encrypt --recipient $BOB_FPR \
  --output ~/.mau/$MY_FPR/msg-to-bob.json.pgp \
  /tmp/private-msg.json
```

**Try to read it as yourself:**

```bash
gpg --decrypt ~/.mau/$MY_FPR/msg-to-bob.json.pgp
```

You'll get:
```
gpg: decryption failed: No secret key
```

**Why?** Because you encrypted it for Bob's public key. Only Bob's private key can decrypt it.

**Now read as Bob:**

```bash
gpg --decrypt --local-user bob@example.com ~/.mau/$MY_FPR/msg-to-bob.json.pgp
```

Success! Bob can read the message, and GPG confirms it was signed by you.

## Step 7: Bob Creates a Comment

Bob can comment on your post:

```bash
cat > /tmp/comment.json <<EOF
{
  "@context": "https://schema.org",
  "@type": "Comment",
  "text": "Great post, Alice!",
  "author": {
    "@type": "Person",
    "name": "Bob",
    "identifier": "$BOB_FPR"
  },
  "about": "/p2p/$MY_FPR/hello.json",
  "dateCreated": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF

# Bob signs and encrypts for himself (public comment)
gpg --sign --encrypt --recipient $BOB_FPR \
  --local-user bob@example.com \
  --output ~/.mau/$BOB_FPR/comment-alice.json.pgp \
  /tmp/comment.json
```

Bob's comment references your post using the Mau address: `/p2p/<fingerprint>/<filename>`

## Step 8: Share with Multiple Recipients (Group)

Create a post visible to both you and Bob:

```bash
cat > /tmp/group-post.json <<'EOF'
{
  "@context": "https://schema.org",
  "@type": "SocialMediaPosting",
  "headline": "Weekend plans?",
  "articleBody": "Anyone up for hiking on Saturday?",
  "datePublished": "2026-02-28T08:00:00Z"
}
EOF

# Encrypt for multiple recipients
gpg --sign --encrypt \
  --recipient $MY_FPR \
  --recipient $BOB_FPR \
  --output ~/.mau/$MY_FPR/group-post.json.pgp \
  /tmp/group-post.json
```

Now both you and Bob can decrypt this file. This is how Mau implements group chats!

## Step 9: Versioning (Edit a Post)

When you edit a post, Mau keeps old versions:

```bash
# Create a versions directory
mkdir -p ~/.mau/$MY_FPR/hello.json.pgp.versions

# Hash the current version
CURRENT_HASH=$(sha256sum ~/.mau/$MY_FPR/hello.json.pgp | cut -d' ' -f1)
echo "Current version hash: $CURRENT_HASH"

# Move current version to versions directory
cp ~/.mau/$MY_FPR/hello.json.pgp \
   ~/.mau/$MY_FPR/hello.json.pgp.versions/$CURRENT_HASH.pgp

# Create new version (edited)
cat > /tmp/hello-v2.json <<'EOF'
{
  "@context": "https://schema.org",
  "@type": "SocialMediaPosting",
  "headline": "Hello from the decentralized web! [EDITED]",
  "articleBody": "This post is encrypted, signed, and stored as a file. I can edit it!",
  "datePublished": "2026-02-28T07:00:00Z",
  "dateModified": "2026-02-28T08:00:00Z"
}
EOF

gpg --sign --encrypt --recipient $MY_FPR \
  --output ~/.mau/$MY_FPR/hello.json.pgp \
  /tmp/hello-v2.json
```

Your directory now has:
```
~/.mau/5D000B2F.../
  hello.json.pgp                    # Latest version
  hello.json.pgp.versions/
    abc123...def.pgp                # Old version (SHA-256 hash)
```

## Step 10: Export and Share Your Public Key

For others to follow you, they need your public key:

```bash
# Export in binary format (Mau standard)
gpg --export $MY_FPR > /tmp/alice-pubkey.pgp

# Or ASCII-armored (human-readable)
gpg --export --armor $MY_FPR > /tmp/alice-pubkey.asc
```

Send this file to friends via email, USB drive, QR code, etc.

**Bob imports your key:**

```bash
# Bob receives alice-pubkey.pgp and imports it
gpg --import /tmp/alice-pubkey.pgp

# Encrypt it with his own key and save to .mau directory
gpg --export $MY_FPR | \
  gpg --encrypt --recipient $BOB_FPR \
  --output ~/.mau/.mau/$MY_FPR.pgp
```

Why encrypt the friend's public key? To prevent malicious programs from adding fake friends.

## What You've Learned

✅ PGP identity = public/private key pair  
✅ Fingerprint = unique 160-bit identifier  
✅ Signing = proves authenticity  
✅ Encryption = ensures privacy  
✅ Multiple recipients = group communication  
✅ File structure = simple directory layout  
✅ Versioning = hash-based old versions  
✅ Mau addressing = `/p2p/<fingerprint>/<filename>`  

## Key Takeaways

1. **Everything is a file** - Posts, messages, comments are just `.pgp` files
2. **Identity is cryptographic** - No usernames, no passwords, just key pairs
3. **Privacy by math** - Encryption makes content unreadable without the key
4. **Signatures prevent tampering** - You can't fake someone else's signature
5. **No central server needed** - Just files and GPG

## Limitations of Manual GPG

What you've done manually is tedious:
- Creating directories
- Encrypting/decrypting files
- Managing fingerprints
- Handling versioning
- No automatic syncing

This is where **Mau CLI** and **Mau packages** come in!

## Next Steps

- **[Mau CLI Tutorial](03b-quickstart-cli.md)** - Use the `mau` command for practical workflows
- **[Mau Package Tutorial](03c-quickstart-package.md)** - Build applications with the Go library

---

**Note:** The commands above work on Linux/macOS. Windows users should use WSL or adapt paths accordingly.
