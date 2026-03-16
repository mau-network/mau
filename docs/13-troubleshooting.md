# Troubleshooting

This guide helps you diagnose and resolve common issues when building with Mau. For each problem, we provide the symptoms, root cause, and solution.

## Table of Contents

- [Installation & Setup](#installation--setup)
- [Authentication Errors](#authentication-errors)
- [Network & Discovery](#network--discovery)
- [File Operations](#file-operations)
- [Server Issues](#server-issues)
- [Performance Problems](#performance-problems)
- [Debugging Tips](#debugging-tips)

---

## Installation & Setup

### Go Module Issues

**Symptom:**
```
go: module github.com/mau-network/mau: Get "https://proxy.golang.org/...": dial tcp: lookup proxy.golang.org: no such host
```

**Solution:**
```bash
# Set GOPROXY to direct if behind corporate firewall
export GOPROXY=direct

# Or use a different proxy
export GOPROXY=https://goproxy.io,direct
```

### Build Fails with Missing Dependencies

**Symptom:**
```
package github.com/ProtonMail/go-crypto/openpgp: cannot find package
```

**Solution:**
```bash
# Run go mod tidy to resolve dependencies
go mod tidy

# Verify go.mod is up to date
go mod verify
```

### Permission Denied on Data Directory

**Symptom:**
```
mkdir ~/.mau/myaccount: permission denied
```

**Solution:**
```bash
# Ensure home directory is writable
ls -ld ~

# Fix permissions if needed
chmod 755 ~

# Verify Mau directory is secure (700 recommended)
chmod 700 ~/.mau
```

---

## Authentication Errors

### `ErrPassphraseRequired`

**Full Error:**
```
Passphrase must be specified
```

**Cause:** Attempting to decrypt a private key without providing the passphrase.

**Solution:**
```go
// When creating account with passphrase
account, err := mau.NewAccount("/path/to/account", "my-secure-passphrase")

// Or set passphrase after loading
err = account.SetPassphrase("my-secure-passphrase")
```

### `ErrIncorrectPassphrase`

**Full Error:**
```
Incorrect passphrase
```

**Cause:** The provided passphrase doesn't match the one used to encrypt the private key.

**Solution:**
1. Double-check the passphrase (case-sensitive)
2. If lost, you'll need to regenerate the account
3. Consider using a password manager

### `ErrNoIdentity`

**Full Error:**
```
Can't find identity
```

**Cause:** The PGP keyring doesn't contain a usable identity (primary key with user ID).

**Solution:**
```bash
# Verify key has identity
gpg --list-keys "account-name"

# If missing, create new account
mau init new-account
```

### `ErrAccountAlreadyExists`

**Full Error:**
```
Account already exists
```

**Cause:** Trying to initialize an account in a directory that already has `account.pgp`.

**Solution:**
```bash
# Use a different directory
mau init myaccount2

# Or remove existing account (WARNING: destroys keys)
rm -rf ~/.mau/myaccount
mau init myaccount
```

### `ErrCannotConvertPrivateKey` / `ErrCannotConvertPublicKey`

**Cause:** PGP key format is unsupported or corrupted.

**Solution:**
```bash
# Export and re-import key to fix format issues
gpg --export-secret-keys account-email > private.asc
gpg --export account-email > public.asc
gpg --delete-secret-and-public-key account-email
gpg --import private.asc
gpg --import public.asc
shred -u private.asc  # Securely delete
```

---

## Network & Discovery

### `ErrCantFindFingerprint`

**Full Error:**
```
Can't find fingerprint.
```

**Cause:** Trying to look up a peer's address in Kademlia DHT, but no nodes have announced that fingerprint.

**Solutions:**

1. **Ensure peer has announced themselves:**
```go
// Peer must announce to DHT
client.AnnounceToDHT()
```

2. **Use direct connection instead:**
```go
// Connect directly if you know their address
err := client.AddFriend(fingerprint, "192.168.1.100:8443")
```

3. **Wait for DHT propagation:**
```go
// DHT announcements take 5-30 seconds to propagate
time.Sleep(30 * time.Second)
addr, err := client.LookupFingerprint(ctx, fingerprint)
```

### `ErrCantFindAddress`

**Full Error:**
```
Can't find address (DNSName) in certificate.
```

**Cause:** TLS certificate doesn't contain fingerprint in Subject Alternative Names (SANs).

**Solution:**
```go
// Regenerate certificate with proper SANs
cert, err := client.GenerateCertificate()
// Mau automatically includes fingerprint in DNSNames

// Verify certificate has correct SANs
for _, name := range cert.Leaf.DNSNames {
    fmt.Println("SAN:", name)
}
```

### `ErrServerDoesNotAllowLookUp`

**Full Error:**
```
Server doesn't allow looking up friends on the internet
```

**Cause:** The Kademlia resolver is explicitly disabled (privacy mode).

**Solution:**
```go
// Enable Kademlia resolver
resolvers := []Resolver{
    &KademliaResolver{
        client: client,
    },
}

// Or use mDNS for local network only
resolvers := []Resolver{
    &LocalResolver{},
}
```

### Firewall Blocking Connections

**Symptom:** Peers can't connect even with correct address.

**Solution:**
```bash
# Check if port is open (default 8443)
sudo ufw status
sudo ufw allow 8443/tcp

# Verify service is listening
netstat -tulpn | grep 8443

# Test from another machine
curl --insecure https://peer-ip:8443/ping
```

### UPnP Port Forwarding Fails

**Symptom:**
```
No services found. Please make sure the firewall is not blocking connections.
```

**Cause:** Router doesn't support UPnP, or UPnP is disabled.

**Solution:**
```bash
# Manual port forwarding:
# 1. Log into your router's admin panel
# 2. Navigate to Port Forwarding / NAT
# 3. Forward external port 8443 → your-local-ip:8443

# Or disable UPnP in Mau and use manual port
client, err := mau.NewClient(dataDir, mau.WithoutUPnP())
```

---

## File Operations

### `ErrInvalidFileName`

**Full Error:**
```
invalid file name: contains path separators or invalid characters
```

**Cause:** File name contains `/`, `\`, or other forbidden characters.

**Solution:**
```go
// Use safe file names (no path separators)
filename := "my-post.json"  // ✅ Good
filename := "../etc/passwd"  // ❌ Bad
filename := "dir/file.json"  // ❌ Bad

// Sanitize user input
import "path/filepath"
filename = filepath.Base(userInput)  // Strips directory components
```

### File Not Signed

**Symptom:**
```
file is not signed
```

**Cause:** File was created without a signature, or signature was stripped.

**Solution:**
```go
// Always sign files when saving
err := client.SaveFile(filename, data, mau.WithSigning())

// Verify signature is present
err := client.VerifyFile(filename)
if err != nil {
    log.Println("Warning: file signature invalid")
}
```

### No Valid Signature Found

**Cause:** File signature doesn't match the claimed author's public key.

**Possible Reasons:**
1. File was tampered with after signing
2. Wrong public key used for verification
3. Signature algorithm not supported

**Solution:**
```go
// Check who signed the file
signer, err := client.GetFileSigner(filename)
if err != nil {
    log.Fatal("Can't determine signer:", err)
}
fmt.Println("File signed by:", signer)

// Ensure you have the correct public key
client.ImportPublicKey(signer, publicKeyData)
```

### File Decryption Fails

**Symptom:** File downloads but can't be decrypted.

**Cause:** File was encrypted for a different recipient.

**Solution:**
```go
// Check if file is encrypted for you
recipients, err := client.GetFileRecipients(filename)
if !contains(recipients, myFingerprint) {
    log.Fatal("File not encrypted for you")
}

// Request sender to re-encrypt for you
```

---

## Server Issues

### Server Won't Start

**Symptom:**
```
listen tcp :8443: bind: address already in use
```

**Cause:** Port 8443 is already in use by another process.

**Solution:**
```bash
# Find process using the port
sudo lsof -i :8443
# or
sudo netstat -tulpn | grep 8443

# Kill the process
sudo kill -9 <PID>

# Or use a different port
```

```go
client, err := mau.NewClient(dataDir, mau.WithPort(9443))
```

### TLS Certificate Errors

**Symptom:**
```
x509: certificate signed by unknown authority
```

**Cause:** Mau uses self-signed certificates (by design).

**Solution:**
```go
// This is expected behavior in P2P systems
// Mau verifies fingerprints instead of CA chains

// For testing with curl:
curl --insecure https://localhost:8443/ping

// In production, fingerprint verification provides security
```

### HTTP 401 Unauthorized

**Symptom:** Peer returns 401 when trying to fetch files.

**Cause:** TLS mutual authentication failed (wrong client certificate).

**Solution:**
```go
// Ensure you're using the correct client certificate
// Mau automatically provides the right cert

// Verify peer has your public key
err := client.ExportPublicKey()
// Send publicKey to peer out-of-band

// Verify peer is in your friends list
friends, _ := client.ListFriends()
for _, f := range friends {
    fmt.Println(f.Fingerprint)
}
```

### `ErrIncorrectPeerCertificate`

**Full Error:**
```
Incorrect Peer certificate.
```

**Cause:** The peer's TLS certificate fingerprint doesn't match their claimed PGP fingerprint.

**Solution:**
1. **Man-in-the-middle attack:** Stop connecting, verify fingerprint out-of-band
2. **Peer changed keys:** Remove old friend entry, re-add with new fingerprint
3. **Certificate expired:** Peer needs to regenerate certificate

```go
// Remove stale friend
client.RemoveFriend(fingerprint)

// Re-add with correct fingerprint
client.AddFriend(correctFingerprint, address)
```

---

## Performance Problems

### Slow File Syncing

**Symptom:** Files take minutes to sync between peers.

**Causes & Solutions:**

1. **Large file sizes:**
```go
// Split large files into chunks
const chunkSize = 1 << 20  // 1 MB
for i := 0; i < len(data); i += chunkSize {
    chunk := data[i:min(i+chunkSize, len(data))]
    client.SaveFile(fmt.Sprintf("file-part-%d.json", i), chunk)
}
```

2. **Many small files:**
```go
// Batch file operations
files := []string{"post1.json", "post2.json", "post3.json"}
client.SyncFiles(files)  // More efficient than individual syncs
```

3. **Network latency:**
```bash
# Test connection to peer
ping peer-ip
traceroute peer-ip

# Use local network discovery (mDNS) when possible
client.EnableMDNS()
```

### High Memory Usage

**Symptom:** Mau process uses excessive RAM.

**Solutions:**

1. **Limit DHT routing table size:**
```go
client, err := mau.NewClient(dataDir, mau.WithMaxDHTNodes(100))
```

2. **Close idle connections:**
```go
// Set connection timeout
client.SetConnectionTimeout(5 * time.Minute)
```

3. **Stream large files instead of loading entirely:**
```go
// Instead of loading full file
data, _ := client.ReadFile("large-video.mp4")  // ❌ Loads all into RAM

// Use streaming
reader, _ := client.OpenFile("large-video.mp4")  // ✅ Streams
defer reader.Close()
io.Copy(output, reader)
```

### CPU Spikes During Encryption

**Cause:** RSA encryption is CPU-intensive for large files.

**Solution:**
```go
// Use symmetric encryption for large files (AES is faster)
// Mau automatically uses hybrid encryption (RSA for key, AES for data)

// For very large files, compress before encrypting
import "compress/gzip"

compressed := new(bytes.Buffer)
gzipWriter := gzip.NewWriter(compressed)
gzipWriter.Write(largeData)
gzipWriter.Close()

client.SaveFile("large-file.json.gz", compressed.Bytes())
```

---

## Debugging Tips

### Enable Verbose Logging

```go
import "log"

// Set log level to debug
log.SetFlags(log.LstdFlags | log.Lshortfile)

// Log all DHT operations
client.SetLogLevel(mau.LogDebug)
```

### Inspect Network Traffic

```bash
# Capture Mau traffic
sudo tcpdump -i any port 8443 -w mau-traffic.pcap

# Analyze with Wireshark
wireshark mau-traffic.pcap
```

### Verify Key Integrity

```bash
# Check PGP key is valid
gpg --list-keys --keyid-format LONG

# Export and inspect
gpg --export-options export-minimal --export <key-id> | gpg --list-packets
```

### Test Connectivity

```bash
# Test if peer is reachable
curl -v --insecure https://peer-ip:8443/ping

# Check TLS handshake
openssl s_client -connect peer-ip:8443 -showcerts
```

### Profile Performance

```go
import _ "net/http/pprof"
import "net/http"

// Start profiler
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()

// Access profiles at http://localhost:6060/debug/pprof/
```

### Common Diagnostic Commands

```bash
# Check Mau process status
ps aux | grep mau

# Monitor resource usage
htop -p $(pgrep mau)

# Check open files/connections
lsof -p $(pgrep mau)

# Verify firewall rules
sudo iptables -L -n | grep 8443

# Test DNS resolution (for Kademlia bootstrap)
nslookup bootstrap.mau-network.com
```

---

## Getting Help

If you encounter an issue not covered here:

1. **Check GitHub Issues:** [mau-network/mau/issues](https://github.com/mau-network/mau/issues)
2. **Search Discussions:** [mau-network/mau/discussions](https://github.com/mau-network/mau/discussions)
3. **File a Bug Report:**
   - Include Go version (`go version`)
   - Include OS/architecture (`uname -a`)
   - Include minimal reproduction steps
   - Attach relevant logs (redact private keys!)

4. **Security Issues:** Email security@mau-network.com (PGP key on website)

---

## Appendix: Error Reference

Quick lookup table for all Mau errors:

| Error | Module | Description | Common Fix |
|-------|--------|-------------|------------|
| `ErrPassphraseRequired` | account | Passphrase not provided | Supply passphrase when creating account |
| `ErrIncorrectPassphrase` | account | Wrong passphrase | Verify passphrase is correct |
| `ErrNoIdentity` | account | No PGP identity found | Create new account with `mau init` |
| `ErrAccountAlreadyExists` | account | Account already exists in directory | Use different directory or remove existing |
| `ErrCannotConvertPrivateKey` | account | Private key format unsupported | Re-export key from GPG |
| `ErrCannotConvertPublicKey` | account | Public key format unsupported | Re-export key from GPG |
| `ErrFriendNotFollowed` | client | Trying to access unfollowed friend | Add friend with `client.AddFriend()` |
| `ErrCantFindFriend` | client | Friend not in local list | Verify fingerprint and re-add |
| `ErrIncorrectPeerCertificate` | client | TLS cert doesn't match fingerprint | Verify peer identity, re-add friend |
| `ErrInvalidFileName` | file | File name has invalid characters | Use safe file names (no `/` or `\`) |
| `ErrCantFindFingerprint` | fingerprint | DHT lookup failed | Ensure peer announced, try direct connect |
| `ErrCantFindAddress` | fingerprint | Cert missing fingerprint in SANs | Regenerate certificate |
| `ErrServerDoesNotAllowLookUp` | resolvers | Kademlia disabled | Enable DHT resolver |

---

*Last Updated: March 16, 2026*
