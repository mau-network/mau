# Authentication & Encryption

This guide explains how Mau uses PGP for identity, authentication, and encryption.

## Table of Contents

- [Overview](#overview)
- [Identity Management](#identity-management)
- [Signing Content](#signing-content)
- [Encrypting Data](#encrypting-data)
- [Key Discovery](#key-discovery)
- [Trust Model](#trust-model)
- [Security Best Practices](#security-best-practices)

## Overview

Mau uses **OpenPGP** as the foundation for all authentication and encryption. Unlike traditional social networks that use username/password, Mau identifies users by their PGP key fingerprints.

### Why PGP?

- **Decentralized** - No central authority issues or revokes keys
- **Proven** - Battle-tested cryptography since 1991
- **Interoperable** - Works with existing PGP tools (gpg, keybase, etc.)
- **Self-sovereign** - You control your identity completely
- **End-to-end** - Only you and intended recipients can decrypt content

### Key Concepts

| Concept | Description |
|---------|-------------|
| **Public Key** | Your identity - shared openly with peers |
| **Private Key** | Your secret - never leaves your device |
| **Fingerprint** | Short hash of public key (e.g., `A1B2C3D4...`) |
| **Signature** | Proof that content came from a specific key |
| **Encryption** | Scrambling data so only recipients can read it |

## Identity Management

### Creating an Account

When you first run `mau init`, a new PGP key pair is generated:

```bash
mau init myaccount

# Creates:
# - ~/.mau/myaccount/account.pgp (encrypted private key)
# - Public key extracted automatically
```

**Default Key Type (as of v0.2.0):**
- Algorithm: **Ed25519** (modern elliptic curve)
- Key size: 256-bit (equivalent to 3072-bit RSA)
- Benefits: Smaller, faster, more secure than RSA

**Legacy RSA Support:**
- Algorithm: RSA
- Key size: 4096-bit
- Still supported for existing keys

### Account Structure

```
~/.mau/myaccount/
├── account.pgp           # Your encrypted private key
├── <fingerprint>/        # Your content directory
│   └── posts/
│       └── *.json        # Your encrypted posts
├── <friend-fpr1>.pgp     # Friend's public key
├── <friend-fpr1>/        # Friend's synced content
└── sync_state.json       # Last sync timestamps
```

### Exporting Your Public Key

To share your identity with others:

```bash
# Export in armored (text) format
mau export-key > my-public-key.asc

# Share this file with friends via:
# - Email attachment
# - QR code (for mobile)
# - Paste into chat
# - Upload to keyserver
```

**Example Public Key:**
```
-----BEGIN PGP PUBLIC KEY BLOCK-----

mDMEZ0pqXBYJKwYBBAHaRw8BAQdA7f9wNZ0hGz5pqX3...
tCNBbGljZSA8YWxpY2VAbWF1Lm5ldHdvcms+iJAEExYIA...
...
-----END PGP PUBLIC KEY BLOCK-----
```

### Importing a Friend's Key

```bash
# From file
mau add-friend alice.asc

# From stdin (paste in terminal)
mau add-friend
# (paste key, then Ctrl+D)

# Mau extracts:
# - Name: "Alice"
# - Email: "alice@mau.network"
# - Fingerprint: A1B2C3D4E5F6...
```

## Signing Content

Every post, file, and message in Mau is **digitally signed** with your private key.

### How Signing Works

Every file created in Mau is automatically signed and encrypted. This process does not embed a signature field inside the content (e.g., a JSON file). Instead, it wraps the content in a standard OpenPGP message format.

1.  **Create Content:** You provide the raw content, such as a JSON object for a post.
2.  **PGP Wrapping:** Mau uses the `openpgp.Sign` and `openpgp.Encrypt` functions to create a PGP message. This message bundles:
    *   The original content (now encrypted).
    *   The digital signature (proving it came from you).
    *   The necessary data for recipients to decrypt it.
3.  **Save as `.pgp`:** The final output is saved as a binary `.pgp` file.

### Verification Process

When a peer receives your `.pgp` file:

1.  **Read PGP Message:** The peer's client reads the `.pgp` file.
2.  **Decrypt & Verify:** It uses its private key to decrypt the content and your public key to verify the signature.
3.  **Extract Content:** If successful, the original, authentic content is extracted.

This process ensures both confidentiality (encryption) and authenticity (signing) without modifying the original content's structure.

### Benefits of Signing

- **Authenticity** - Proves content came from the claimed author
- **Integrity** - Detects tampering (even 1 bit change breaks signature)
- **Non-repudiation** - Author can't deny creating the content
- **Trust chains** - Others can verify without central authority

## Encrypting Data

Mau encrypts all content by default to protect privacy.

### Encryption Process

1. **Generate random key** (AES-256 symmetric key)
2. **Encrypt content** with symmetric key
3. **Encrypt symmetric key** with each recipient's public key
4. **Store encrypted content + encrypted keys** together

This is called **hybrid encryption** - combines speed of symmetric encryption with security of public-key encryption.

### Example: Encrypted Post

```
-----BEGIN PGP MESSAGE-----

hQIMA3sK9F8HvF0QAQP+Nk7fz...  ← Encrypted for recipient 1
hQIMA7x8qR3pZ2gfAQP9Hw2q...  ← Encrypted for recipient 2
...
wV4DuB3Kz8xN2YASA/9JKg...    ← Encrypted content
-----END PGP MESSAGE-----
```

### Who Can Decrypt?

By default, posts are encrypted for:
1. **Yourself** (so you can read your own posts)
2. **All friends** in your keyring

```go
// Publishing a post encrypts for self + friends
client.SavePost("hello.json", post)

// Encrypts for:
// - Your fingerprint
// - All friends' fingerprints
```

### Reading Encrypted Content

When you open a friend's post:

1. **Mau tries each encrypted key** to find one encrypted for you
2. **Decrypts the symmetric key** with your private key
3. **Decrypts the content** with the symmetric key
4. **Verifies signature** to confirm authenticity

If you're not in the recipient list, decryption fails (you can't read it).

## Key Discovery

### Local Keyring

Your friend's public keys are stored locally:

```bash
~/.mau/myaccount/
├── <friend-fpr1>.pgp
├── <friend-fpr2>.pgp
└── <friend-fpr3>.pgp
```

### Network Discovery

When you connect to a new peer:

1. **TLS handshake** establishes encrypted connection
2. **Peer presents certificate** with embedded PGP fingerprint
3. **You extract fingerprint** from certificate
4. **You fetch their public key** from their HTTP endpoint:
   ```
   GET https://peer-ip:port/.mau/<fingerprint>.pgp
   ```
5. **Verify fingerprint** matches certificate

### Kademlia DHT

The Distributed Hash Table stores peer locations:

```
Fingerprint → [IP addresses]
```

When you want to sync with a friend:
1. **Look up fingerprint** in DHT
2. **Get list of IP addresses** where they're reachable
3. **Connect to peer** and request content

## Trust Model

Mau uses a **web of trust** model:

### Direct Trust

- You explicitly trust friends by importing their public keys
- No third party validates their identity
- You verify their key fingerprint through another channel (e.g., in person, secure chat)

### Transitive Trust (Future)

Planned feature: trust friends-of-friends at lower confidence levels

```
You → Alice (100% trust)
Alice → Bob (100% trust)
You → Bob (50% transitive trust)
```

### Revocation

If a key is compromised:

1. **Generate revocation certificate** with old key
2. **Publish revocation** to network
3. **Generate new key** and notify friends
4. **Friends update** their keyring

```bash
# Generate revocation certificate
gpg --output revoke.asc --gen-revoke your@email.com

# Import revocation (marks key as invalid)
gpg --import revoke.asc

# Generate new key
mau init myaccount-new
```

## Security Best Practices

### Protect Your Private Key

- **Encrypt with strong passphrase** (at least 20 characters)
- **Store backup offline** (USB drive, paper wallet)
- **Never share private key** - only public key
- **Use hardware security key** (YubiKey, Nitrokey) for extra protection

### Verify Fingerprints

Always verify fingerprints through a second channel:

```bash
# Your fingerprint
mau whoami

# Friend's fingerprint (after importing)
mau list-friends

# Compare with what friend tells you via:
# - Phone call
# - In person
# - Secure messaging app
# - QR code scan
```

### Key Rotation

Rotate keys periodically (e.g., every 2 years):

1. Generate new key
2. Sign new key with old key (proves ownership)
3. Publish transition message
4. Gradually migrate content to new key
5. Revoke old key after transition period

### Passphrase Strength

Use a strong passphrase for your private key:

```bash
# Good: Long, memorable sentence
"The quick brown fox jumps over 13 lazy dogs near the river!"

# Bad: Short, common phrase
"password123"
```

**Passphrase Entropy:**
- At least **80 bits** of entropy
- Use **diceware** for memorable passphrases
- Or **password manager** generated passphrase

### Multi-Device Setup

Using the same Mau identity across multiple devices requires securely transferring your private key. Since Mau does not have a built-in sync service for private keys, this must be done manually.

**The Recommended Method: Secure Manual Transfer**

1.  **Export Your Account:** On your primary device, use the `mau export-account` command. This will create an encrypted archive of your private key.
    ```bash
    # on Device A
    mau export-account > my-account-backup.mau
    ```
2.  **Securely Copy:** Transfer this `.mau` file to your second device using a secure method (e.g., a USB drive, `scp`, or an end-to-end encrypted messaging service). **Do not email it or upload it to an insecure cloud service.**
3.  **Import Your Account:** On your second device, import the account.
    ```bash
    # on Device B
    mau import-account my-account-backup.mau
    ```

You will be prompted for the passphrase you used on your primary device. Once complete, both devices will have the same identity. Any content you create on one device will need to be synced to the other via the standard Mau peer-to-peer sync process.

**Important Considerations:**

*   **One Identity, One Key:** Mau's model is one private key per identity. The options of subkeys or key splitting (Shamir's Secret Sharing) are advanced PGP features that are not directly supported through Mau's command-line interface. While possible with external tools like `gpg`, it falls outside the standard Mau workflow.
*   **Risk:** The biggest risk is the moment the private key is being transferred. Ensure the channel you use is secure. Once imported, the key is as secure as the device it's on.
*   **Revocation:** If one of your devices is lost or stolen, you must revoke the key from another trusted device and generate a new identity to ensure your security.

### Threat Model

Mau's security assumes:

**Protected Against:**
- ✅ Man-in-the-middle attacks (TLS + key pinning)
- ✅ Eavesdropping (end-to-end encryption)
- ✅ Content tampering (digital signatures)
- ✅ Impersonation (PGP identity verification)

**Not Protected Against:**
- ⚠️ Compromised private key (physical device theft)
- ⚠️ Malware on your device (keyloggers, screensharing)
- ⚠️ Social engineering (tricked into trusting fake key)
- ⚠️ Quantum computers (post-quantum PGP not yet standard)

## Code Examples

### Signing Content

```go
package main

import (
    "github.com/mau-network/mau"
    "log"
)

func main() {
    client, _ := mau.NewClient("~/.mau/myaccount")
    
    // Create post
    post := map[string]interface{}{
        "@type":    "SocialMediaPosting",
        "headline": "This will be signed",
    }
    
    // Sign and save (signature added automatically)
    err := client.SavePost("signed-post.json", post)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Println("Post signed and saved")
}
```

### Verifying Signatures

```go
// Load and verify friend's post
friend := client.GetFriend(fingerprint)
file, _ := client.ListFiles(friend.Fingerprint(), 1)
post, err := client.LoadPost(file[0])

if err != nil {
    log.Fatal("Signature verification failed!")
}

log.Println("Signature valid - post is authentic")
```

### Encrypting for Specific Recipients

```go
// Encrypt for specific friends only
recipients := []string{
    "A1B2C3D4E5F6...",  // Friend 1 fingerprint
    "F6E5D4C3B2A1...",  // Friend 2 fingerprint
}

err := client.SavePostForRecipients("private.json", post, recipients)
```

## FAQ

**Q: Can I use my existing GPG key?**
A: Yes! Run `mau init --import ~/.gnupg/secring.gpg`

**Q: What if I lose my private key?**
A: Your identity is lost. Always keep encrypted backups offline.

**Q: Can others fake my identity?**
A: Only if they steal your private key + passphrase. Use strong passphrases!

**Q: Does Mau use key servers?**
A: No. Keys are fetched directly from peers via HTTP.

**Q: Are key sizes configurable?**
A: Ed25519 is default (fixed size). RSA can be 2048-4096 bits (legacy).

**Q: What happens if a friend changes their key?**
A: You must re-import their new public key. Old content stays encrypted with old key.

## Next Steps

- **[Peer-to-Peer Networking](06-networking.md)** - Discover and sync with peers
- **[Building Social Apps](08-building-social-apps.md)** - Apply authentication in your app
- **[Privacy & Security](09-privacy-security.md)** - Deep dive into security considerations

## References

- [OpenPGP Standard (RFC 4880)](https://tools.ietf.org/html/rfc4880)
- [Ed25519 Signature Scheme](https://ed25519.cr.yp.to/)
- [GnuPG Documentation](https://www.gnupg.org/documentation/)
- [Keybase - PGP for Everyone](https://keybase.io/)

---

*This documentation is for developers building on Mau. For protocol details, see the [specification](../README.md).*
