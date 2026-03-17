# Privacy & Security Best Practices

This guide covers security considerations and privacy best practices when building and deploying Mau applications.

## Table of Contents

- [Security Model](#security-model)
- [Threat Model](#threat-model)
- [Private Key Protection](#private-key-protection)
- [File Encryption & Signing](#file-encryption--signing)
- [Network Security](#network-security)
- [Data Retention & Deletion](#data-retention--deletion)
- [Privacy Considerations](#privacy-considerations)
- [Security Checklist](#security-checklist)

## Security Model

Mau's security is built on several foundational principles:

### Core Security Properties

| Property | Implementation | Protection Against |
|----------|----------------|---------------------|
| **Identity** | PGP public keys | Impersonation, fake accounts |
| **Authentication** | Digital signatures | Forged content, tampering |
| **Confidentiality** | PGP encryption | Eavesdropping, data leaks |
| **Integrity** | Cryptographic hashes | Corruption, modification |
| **Non-repudiation** | Signature verification | Denial of authorship |

### Trust Model

Mau uses a **web of trust** model:

```
┌──────────────┐
│   You (A)    │──────────┐
└──────────────┘          │
       │                  │
       │ trust            │ direct
       │                  │ connection
       ▼                  ▼
┌──────────────┐    ┌──────────────┐
│  Friend (B)  │◄───│  Peer (C)    │
└──────────────┘    └──────────────┘
       │
       │ transitively
       │ trust?
       ▼
┌──────────────┐
│ Friend's     │
│ Friend (D)   │
└──────────────┘
```

**Key principles:**
- You explicitly trust friends by adding their public keys
- Content is verified against the friend's key
- No automatic trust propagation (friend-of-friend is untrusted by default)
- Each user maintains their own trust decisions

## Threat Model

### What Mau Protects Against

✅ **Eavesdropping**: All content is encrypted end-to-end  
✅ **Impersonation**: Signatures verify sender identity  
✅ **Tampering**: Modified content fails signature verification  
✅ **Man-in-the-middle**: TLS + certificate fingerprint validation  
✅ **Unauthorized access**: Only recipients can decrypt files  
✅ **Data exfiltration**: No central server to compromise  

### What Mau Does NOT Protect Against

❌ **Compromised endpoints**: If your device is hacked, keys are exposed  
❌ **Malicious recipients**: Recipients can leak content after decryption  
❌ **Traffic analysis**: Metadata (who talks to whom) may be observable  
❌ **Key compromise**: Lost/stolen private keys compromise all past content  
❌ **Social engineering**: Users can be tricked into trusting wrong keys  
❌ **Quantum computers**: RSA and current PGP may be vulnerable in future  

### Attack Scenarios

**Scenario 1: Network Attacker**
- **Attack**: Intercepts network traffic
- **Defense**: TLS encryption + certificate pinning
- **Mitigation**: Even if TLS is broken, content is PGP-encrypted

**Scenario 2: Malicious Peer**
- **Attack**: Peer sends fake content claiming to be from Alice
- **Defense**: Signature verification fails (doesn't match Alice's key)
- **Mitigation**: Content is rejected, peer can be blocked

**Scenario 3: Storage Access**
- **Attack**: Attacker gains read access to `~/.mau` directory
- **Defense**: Private key is encrypted with user's password
- **Mitigation**: Strong password required to use the key

## Private Key Protection

Your private key is the **crown jewel** of your Mau identity. If compromised, attackers can:
- Decrypt all your past and future messages
- Impersonate you to your friends
- Sign content as if it came from you

### Password Security

When creating an account, the private key is encrypted with your password:

```bash
# Strong password required
mau init myaccount
Password: ************  # At least 12 characters recommended
```

**Password requirements:**
- ✅ Minimum 12 characters (20+ recommended)
- ✅ Mix of uppercase, lowercase, numbers, symbols
- ✅ Unique (don't reuse from other services)
- ✅ Consider using a password manager
- ❌ Avoid dictionary words, personal info, patterns

### Key Storage

**Default location**: `~/.mau/<account>/account.pgp`

**File permissions** should be restrictive:
```bash
# Check permissions
ls -la ~/.mau/myaccount/account.pgp
# Should be: -rw------- (0600) or -rw-r----- (0640)

# Fix if needed
chmod 600 ~/.mau/myaccount/account.pgp
```

**Backup strategy:**
```bash
# Encrypted backup to external drive
cp ~/.mau/myaccount/account.pgp /media/backup/mau-backup-$(date +%F).pgp

# Or use tar with encryption
tar czf - ~/.mau/myaccount | gpg -c > mau-backup.tar.gz.gpg
```

### Key Rotation

If you suspect key compromise:

1. **Generate new account** with fresh keys
2. **Notify friends** through side channel (email, Signal, in person)
3. **Share new public key** with all contacts
4. **Revoke old key** (if using external keyserver)
5. **Delete old account** after migration

```bash
# Create new account
mau init myaccount-new

# Export new public key
mau export-key myaccount-new > new-public-key.asc

# Manually share with friends, then delete old account
rm -rf ~/.mau/myaccount
```

## File Encryption & Signing

Every file in Mau is both **signed** and **encrypted**:

### Signature Verification

When receiving content, always verify the signature:

```go
// Automatic verification in Mau
file, err := account.GetFile(friendFingerprint, "post.json")
if err != nil {
    log.Fatal(err)
}

// This automatically verifies:
// 1. File is signed
// 2. Signature is valid
// 3. Signer matches expected friend
err = file.VerifySignature(account, friendFingerprint)
if err != nil {
    log.Fatal("Signature verification failed:", err)
}
```

**What this prevents:**
- Tampering: Modified content invalidates signature
- Impersonation: Only friend's private key can produce valid signature
- Replay attacks: Signature is specific to exact file content

### Recipient Control

Files are encrypted **only** for specified recipients:

```go
// Encrypt for specific friends
recipients := []Fingerprint{
    alice.Fingerprint(),
    bob.Fingerprint(),
}

err := account.SaveFile("secret.json", content, recipients)
```

**Best practices:**
- ✅ Minimize recipient list (need-to-know principle)
- ✅ Review recipients before sharing sensitive data
- ❌ Don't encrypt for untrusted or unknown keys
- ❌ Remember: recipients can always re-share after decryption

### Checking Recipients

Verify who can read a file:

```go
recipients, err := file.Recipients(account)
if err != nil {
    log.Fatal(err)
}

fmt.Println("File is encrypted for:")
for _, friend := range recipients {
    fmt.Printf("- %s (%s)\n", friend.Name(), friend.Fingerprint())
}
```

## Network Security

### TLS Certificate Validation

Mau uses **self-signed certificates** with fingerprint pinning:

```go
// Automatic in Mau: TLS cert must match PGP key fingerprint
// This prevents MITM even with compromised CA
```

**How it works:**
1. Each peer generates TLS certificate from their PGP key
2. Certificate fingerprint must match public key fingerprint
3. Connection rejected if mismatch detected

**This prevents:**
- Certificate authority compromise
- DNS hijacking
- Man-in-the-middle attacks

### Port Security

**Default ports:**
- HTTP server: 8080 (configurable)
- TLS: Optional (recommended in production)

**Firewall configuration:**
```bash
# Allow Mau traffic
sudo ufw allow 8080/tcp

# Or restrict to specific IPs
sudo ufw allow from 192.168.1.0/24 to any port 8080
```

### Network Exposure

**Development:**
```bash
# Localhost only (default)
mau serve --address 127.0.0.1:8080
```

**Production:**
```bash
# Public internet (use with TLS!)
mau serve --address 0.0.0.0:8443 --tls
```

**Best practices:**
- ✅ Use TLS for public servers
- ✅ Run behind reverse proxy (nginx, Caddy) for additional security
- ✅ Enable rate limiting to prevent DoS
- ❌ Don't expose debug endpoints publicly
- ❌ Don't run as root (use dedicated user)

### Kademlia DHT Security

The DHT exposes your peer presence:

**Information leaked:**
- Your public key fingerprint
- IP address and port
- Online/offline status
- Approximate location (inferred from IP)

**Mitigation strategies:**
- Use VPN or Tor for anonymity
- Run multiple nodes to obfuscate activity
- Use mDNS only for local networks
- Disable DHT if privacy is critical:

```go
// Disable DHT discovery
client := mau.NewClient(dir)
client.DisableDHT()
```

## Data Retention & Deletion

### Secure File Deletion

Regular `rm` may leave data recoverable:

```bash
# Use secure deletion (overwrites data)
shred -u ~/.mau/myaccount/files/sensitive.json.pgp

# Or on whole directory
find ~/.mau/myaccount/files -type f -exec shred -u {} \;
```

**In code:**
```go
// Delete file versions too
file, _ := account.GetFile(friendFp, "data.json")
for _, version := range file.Versions() {
    os.Remove(version.Path)
}
os.Remove(file.Path)

// Optionally securely wipe
// (requires external tool or library)
```

### Retention Policies

Implement automatic cleanup:

```go
// Delete files older than 30 days
cutoff := time.Now().AddDate(0, 0, -30)

files, _ := account.ListFiles(friendFingerprint)
for _, file := range files {
    info, _ := os.Stat(file.Path)
    if info.ModTime().Before(cutoff) {
        os.Remove(file.Path)
    }
}
```

### Forward Secrecy Limitations

⚠️ **Mau does not provide forward secrecy**

- If your private key is compromised, all past messages can be decrypted
- This is a fundamental limitation of PGP-based encryption
- Consider implementing application-level ephemeral keys for sensitive chats

**Alternative for high-security needs:**
- Use Signal Protocol (ratcheting keys)
- Implement session keys rotated regularly
- Use Mau for discovery + exchange session keys separately

## Privacy Considerations

### Metadata Leakage

**What's exposed:**
```
Network observer can see:
- You connected to peer X at time T
- File Y was transferred (size, timing)
- DHT queries reveal interests
```

**What's hidden:**
```
Encrypted and not observable:
- File contents
- File names (encrypted)
- Number of recipients
- Actual relationships (who is whose friend)
```

### Anonymity vs. Accountability

Mau prioritizes **accountability** over anonymity:

- Every action is tied to a PGP key
- Keys are pseudo-anonymous (not linked to real identity by default)
- But friends know your key = your identity in the network

**For true anonymity:**
- Use anonymous key (no name/email)
- Access over Tor
- Never link key to real identity
- Rotate keys regularly

### Privacy-Preserving Applications

**Example: Anonymous forums**
```go
// Create anonymous post
post := map[string]interface{}{
    "@type": "SocialMediaPosting",
    "author": map[string]interface{}{
        "@type": "Person",
        "identifier": account.Fingerprint().String()[:8], // Short hash
        // No name, email, or identifying info
    },
    "headline": "Anonymous thought...",
}
```

**Example: Ephemeral messaging**
```go
// Auto-delete after reading
msg, _ := file.Reader(account)
content, _ := io.ReadAll(msg)
fmt.Println(string(content))

// Delete immediately
os.Remove(file.Path)
```

## Security Checklist

### Deployment

- [ ] Use strong password (12+ characters, unique)
- [ ] Set restrictive file permissions (0600 on private keys)
- [ ] Enable TLS for public servers
- [ ] Run service as non-root user
- [ ] Enable firewall rules
- [ ] Keep dependencies updated
- [ ] Monitor logs for suspicious activity
- [ ] Implement rate limiting
- [ ] Set up automated backups (encrypted)
- [ ] Document incident response plan

### Development

- [ ] Always verify signatures before processing
- [ ] Validate file names (prevent path traversal)
- [ ] Sanitize user input (prevent injection)
- [ ] Use constant-time comparison for fingerprints
- [ ] Log security events (failed auth, invalid signatures)
- [ ] Handle errors securely (no info leakage)
- [ ] Fuzz test file parsing code
- [ ] Review dependencies for vulnerabilities
- [ ] Implement key rotation mechanism
- [ ] Add integrity checks for backups

### User Education

- [ ] Explain key backup importance
- [ ] Warn about key sharing risks
- [ ] Document how to verify friend keys
- [ ] Provide key rotation instructions
- [ ] Explain what metadata is exposed
- [ ] Clarify forward secrecy limitations
- [ ] Guide users on secure deletion
- [ ] Recommend VPN/Tor for sensitive use

## Advanced Topics

### Auditing & Compliance

For regulated environments:

```go
// Log all access
func AuditLog(event string, details map[string]string) {
    log.Printf("[AUDIT] %s: %v", event, details)
}

// Track file access
file, err := account.GetFile(fp, name)
AuditLog("file_access", map[string]string{
    "user": account.Fingerprint().String(),
    "peer": fp.String(),
    "file": name,
    "timestamp": time.Now().Format(time.RFC3339),
})
```

### Compliance with GDPR

If building EU-facing application:

- **Right to erasure**: Implement secure deletion
- **Data portability**: Provide export functionality
- **Consent**: Track and honor encryption preferences
- **Transparency**: Document what data is collected

```go
// GDPR export
func ExportUserData(account *Account) error {
    archive := createZip()
    
    // Include all files
    files, _ := account.ListAllFiles()
    for _, file := range files {
        content, _ := file.Reader(account)
        archive.Add(file.Name(), content)
    }
    
    // Include friend list
    friends, _ := account.ListFriends()
    archive.Add("friends.json", marshalJSON(friends))
    
    return archive.Save("user-data-export.zip")
}
```

### Vulnerability Disclosure

If you discover a security issue in Mau:

1. **Do not** publicly disclose immediately
2. Contact maintainers privately (see SECURITY.md in repo)
3. Provide detailed reproduction steps
4. Allow reasonable time for fix (90 days standard)
5. Coordinate public disclosure timing

## Conclusion

Security is a **shared responsibility**:

- **Mau provides**: Strong cryptographic primitives and secure defaults
- **You must ensure**: Proper key management, secure deployment, and user education
- **Users must practice**: Good password hygiene, backup discipline, and cautious key sharing

Remember: **The most secure system is useless if users don't trust it**. Balance security with usability, and always document the threat model clearly.

For implementation details, see:
- [Authentication & Encryption](05-authentication.md)
- [Peer-to-Peer Networking](06-networking.md)
- [Troubleshooting](13-troubleshooting.md)
