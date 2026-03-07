# HTTP API Reference

This document provides a complete reference for the Mau HTTP API that every peer implements. Understanding this API is essential for building Mau applications and interoperating with other Mau clients.

## Table of Contents
1. [Overview](#overview)
2. [Transport Security](#transport-security)
3. [P2P Endpoints](#p2p-endpoints)
4. [Kademlia DHT Endpoints](#kademlia-dht-endpoints)
5. [Error Responses](#error-responses)
6. [Client Implementation Guide](#client-implementation-guide)

---

## Overview

### Base Concepts

Every Mau peer runs an HTTP/TLS server that exposes:

1. **P2P Content API** (`/p2p/*`) - Serve and sync files
2. **Kademlia DHT API** (`/kad/*`) - Peer discovery and routing

### URL Structure

All Mau endpoints follow this pattern:

```
mau://<host>:<port>/<namespace>/<resource>
```

**Examples:**
```
mau://192.168.1.100:8080/p2p/5D000B2F2C040A1675B49D7F0C7CB7DC36999D56
mau://alice.example.com:443/p2p/5D000B2F2C040A1675B49D7F0C7CB7DC36999D56/hello-world.json
mau://bob.local:8080/kad/ping
```

The `mau://` protocol is HTTP/TLS with mutual authentication (see [Transport Security](#transport-security)).

---

## Transport Security

### TLS 1.3 with Mutual Authentication

All Mau communication uses **TLS 1.3** with **client certificates** for mutual authentication.

#### Server Configuration

```go
TLS Config:
- MinVersion: TLS 1.3
- Certificates: [Server's certificate derived from PGP key]
- ClientAuth: RequestClientCert
- InsecureSkipVerify: true  // We verify manually via fingerprint
- CurvePreferences: [X25519, P256, P384, P521]
```

#### Client Configuration

```go
TLS Config:
- MinVersion: TLS 1.3
- Certificates: [Client's certificate derived from PGP key]
- InsecureSkipVerify: true
- VerifyPeerCertificate: Custom verification against expected fingerprint
```

#### Certificate Verification

Instead of traditional CA-based verification, Mau uses **PGP key fingerprints**:

1. Both peers present X.509 certificates during TLS handshake
2. Certificates are self-signed and derived from PGP keys
3. Each peer extracts the fingerprint from the peer's certificate
4. The fingerprint is compared against the expected value

**Example fingerprint extraction:**
```go
// From certificate's SubjectKeyId or by hashing the public key
fingerprint := sha1(certificate.PublicKey)
if fingerprint != expectedFingerprint {
    return ErrIncorrectPeerCertificate
}
```

### Timeouts

Default server timeouts:
- **ReadTimeout**: 30 seconds
- **WriteTimeout**: 30 seconds
- **IdleTimeout**: 120 seconds
- **ReadHeaderTimeout**: 10 seconds

---

## P2P Endpoints

### 1. List Files

Retrieve a list of files owned by a specific user.

#### Request

```http
GET /p2p/<fingerprint> HTTP/1.1
Host: peer.example.com
If-Modified-Since: Wed, 27 Feb 2026 00:00:00 GMT
```

**Parameters:**
- `<fingerprint>` (path) - 40-character hex fingerprint of the user's PGP key

**Headers:**
- `If-Modified-Since` (optional) - RFC 7231 HTTP date. Only return files modified after this timestamp

#### Response

**Success (200 OK):**
```json
[
  {
    "path": "/p2p/5D000B2F2C040A1675B49D7F0C7CB7DC36999D56/hello-world.json",
    "size": 2048,
    "sum": "abc123def456789abcdef0123456789abcdef0123456789abcdef0123456789"
  },
  {
    "path": "/p2p/5D000B2F2C040A1675B49D7F0C7CB7DC36999D56/recipe-lasagna.json",
    "size": 4096,
    "sum": "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
  }
]
```

**Response Fields:**
- `path` (string) - Full path to the file (used for GET requests)
- `size` (int64) - File size in bytes
- `sum` (string) - SHA-256 hash of the file content (64 hex characters)

**Authorization:**
- Only files that the requesting peer is authorized to read are included
- Authorization is based on PGP encryption recipients (see [Authorization Logic](#authorization-logic))

#### Example cURL

```bash
curl -k \
  --cert client-cert.pem \
  --key client-key.pem \
  -H "If-Modified-Since: Wed, 27 Feb 2026 00:00:00 GMT" \
  "https://peer.example.com:8080/p2p/5D000B2F2C040A1675B49D7F0C7CB7DC36999D56"
```

#### Use Cases

- **Initial sync**: Omit `If-Modified-Since` to get all files
- **Incremental sync**: Include `If-Modified-Since` with the last sync timestamp
- **Bandwidth optimization**: Check `sum` field to avoid re-downloading unchanged files

---

### 2. Get File

Download a specific file.

#### Request

```http
GET /p2p/<fingerprint>/<filename> HTTP/1.1
Host: peer.example.com
Range: bytes=0-1023
```

**Parameters:**
- `<fingerprint>` (path) - User's PGP key fingerprint
- `<filename>` (path) - Name of the file (e.g., `hello-world.json`)

**Headers:**
- `Range` (optional) - RFC 7233 byte range for partial downloads

#### Response

**Success (200 OK or 206 Partial Content):**
```http
HTTP/1.1 200 OK
Content-Type: application/octet-stream
Content-Length: 2048
Accept-Ranges: bytes

<file content (PGP-encrypted binary)>
```

**Response Headers:**
- `Content-Type`: `application/octet-stream`
- `Accept-Ranges`: `bytes` (supports range requests)
- `Content-Length`: File size in bytes

**Authorization:**
- Returns `401 Unauthorized` if the requesting peer is not a recipient of the file
- Returns `404 Not Found` if the file doesn't exist

#### Example cURL

```bash
# Download full file
curl -k \
  --cert client-cert.pem \
  --key client-key.pem \
  -o hello-world.json.pgp \
  "https://peer.example.com:8080/p2p/5D000B2F.../hello-world.json"

# Download partial file (resume)
curl -k \
  --cert client-cert.pem \
  --key client-key.pem \
  -H "Range: bytes=1024-" \
  -o hello-world.json.pgp.part \
  "https://peer.example.com:8080/p2p/5D000B2F.../hello-world.json"
```

#### File Format

Downloaded files are **PGP-encrypted** (RFC 4880 OpenPGP Message Format):

1. **Decrypt** with recipient's private key
2. **Verify signature** against author's public key
3. **Extract content** (JSON-LD document)

**Example decryption:**
```bash
# Decrypt and verify
gpg --decrypt hello-world.json.pgp > hello-world.json

# Check signature
gpg --verify hello-world.json.pgp
```

---

### 3. Get File Version

Download a specific historical version of a file.

#### Request

```http
GET /p2p/<fingerprint>/<filename>.version/<hash> HTTP/1.1
Host: peer.example.com
```

**Parameters:**
- `<fingerprint>` (path) - User's PGP key fingerprint
- `<filename>` (path) - Name of the file
- `<hash>` (path) - SHA-256 hash of the version (64 hex characters)

#### Response

**Success (200 OK):**
```http
HTTP/1.1 200 OK
Content-Type: application/octet-stream
Content-Length: 2048
Accept-Ranges: bytes

<file content (PGP-encrypted binary)>
```

Same response format as [Get File](#2-get-file).

#### Use Cases

- **Version history**: Browse old versions of files
- **Conflict resolution**: Compare different versions
- **Audit trail**: For contracts, signed documents, etc.

#### Versioning Strategy

When a file is updated:
1. Old version moves to `<filename>.versions/<sha256-hash>.pgp`
2. New version overwrites `<filename>.pgp`
3. Versions are stored by content hash (deduplication)

**Example:**
```
alice-FPR/
├── recipe-lasagna.json.pgp          # Current version
└── recipe-lasagna.json.pgp.versions/
    ├── abc123...xyz.pgp             # Version 1
    └── def456...uvw.pgp             # Version 2
```

---

## Kademlia DHT Endpoints

The Kademlia Distributed Hash Table (DHT) enables peer discovery without central servers.

### 4. Ping

Health check to verify a peer is online and update routing tables.

#### Request

```http
GET /kad/ping HTTP/1.1
Host: peer.example.com
```

No parameters required.

#### Response

**Success (200 OK):**
```http
HTTP/1.1 200 OK
Content-Length: 0
```

Empty body. A `200` status indicates the peer is alive.

**Side Effects:**
- The receiver adds/updates the sender's fingerprint in its routing table
- Extracted from the client certificate's fingerprint

#### Rate Limiting

Clients implement **ping backoff** to avoid flooding:
- Minimum 30 seconds between pings to the same peer
- Tracked per fingerprint in `lastPing` map

#### Example cURL

```bash
curl -k \
  --cert client-cert.pem \
  --key client-key.pem \
  "https://peer.example.com:8080/kad/ping"
```

---

### 5. Find Peer

Locate a peer by fingerprint using the Kademlia routing algorithm.

#### Request

```http
GET /kad/find_peer/<fingerprint> HTTP/1.1
Host: peer.example.com
```

**Parameters:**
- `<fingerprint>` (path) - Target user's PGP key fingerprint (40 hex characters)

#### Response

**Success (200 OK):**
```json
[
  {
    "fingerprint": "5D000B2F2C040A1675B49D7F0C7CB7DC36999D56",
    "address": "alice.example.com:443"
  },
  {
    "fingerprint": "A1B2C3D4E5F6789012345678901234567890ABCD",
    "address": "192.168.1.100:8080"
  }
]
```

**Response Fields:**
- `fingerprint` (string) - Peer's PGP key fingerprint
- `address` (string) - Peer's network address (hostname:port or IP:port)

**Algorithm:**
1. If the target fingerprint is in the receiver's routing table, return it
2. Otherwise, return the **K closest peers** (K=20 by default) based on XOR distance
3. Client recursively queries returned peers until target is found

#### Example cURL

```bash
curl -k \
  --cert client-cert.pem \
  --key client-key.pem \
  "https://peer.example.com:8080/kad/find_peer/5D000B2F2C040A1675B49D7F0C7CB7DC36999D56"
```

#### Kademlia Parameters

- **B (Buckets)**: 160 (20 bytes × 8 bits) - for v4 fingerprints
- **K (Replication)**: 20 - max peers per bucket
- **α (Alpha)**: 3 - parallel lookup queries
- **Stall Period**: 1 hour - bucket refresh interval

#### Use Case Flow

**Bob wants to sync with Alice:**
1. Bob queries bootstrap node: `GET /kad/find_peer/alice-FPR`
2. Bootstrap returns 3 peers closer to Alice
3. Bob queries those 3 peers in parallel
4. One peer returns Alice's address
5. Bob connects to Alice: `GET /p2p/alice-FPR`

---

## Error Responses

### HTTP Status Codes

| Status | Meaning | Common Causes |
|--------|---------|---------------|
| `200 OK` | Success | Normal operation |
| `206 Partial Content` | Range request success | Resume download |
| `400 Bad Request` | Invalid request | Malformed fingerprint, invalid path |
| `401 Unauthorized` | Not authorized | Not a file recipient, TLS auth failed |
| `404 Not Found` | Resource not found | File doesn't exist, unknown fingerprint |
| `405 Method Not Allowed` | Wrong HTTP method | Non-GET request to P2P/DHT endpoint |
| `500 Internal Server Error` | Server error | Filesystem error, PGP operation failed |

### Error Response Format

Errors return a plain text message in the response body:

```http
HTTP/1.1 404 Not Found
Content-Type: text/plain; charset=utf-8
Content-Length: 14

File not found
```

### Authorization Logic

File access is controlled by **PGP encryption recipients**:

1. Server decrypts file metadata to extract recipient list
2. If requesting peer's fingerprint is in recipient list → `200 OK`
3. If not in recipient list → `401 Unauthorized`

**Special case:** Public files are encrypted to the author's own key, so only the author and explicit recipients can access them initially. To share publicly, encrypt for all known peers or use a shared public key.

---

## Client Implementation Guide

### Basic Sync Flow

```go
// 1. Discover peer
peer, err := dht.FindPeer(ctx, aliceFingerprint)
if err != nil {
    return err
}

// 2. Create authenticated client
client, err := account.Client(aliceFingerprint, []string{peer.Address})
if err != nil {
    return err
}

// 3. List files (incremental sync)
lastSync := time.Date(2026, 2, 27, 0, 0, 0, 0, time.UTC)
resp, err := client.Get(ctx, fmt.Sprintf(
    "mau://%s/p2p/%s",
    peer.Address,
    aliceFingerprint,
), map[string]string{
    "If-Modified-Since": lastSync.Format(http.TimeFormat),
})

// 4. Parse file list
var files []FileListItem
json.Unmarshal(resp.Body, &files)

// 5. Download each file
for _, file := range files {
    // Check if already downloaded
    localHash := getLocalFileHash(file.Path)
    if localHash == file.Sum {
        continue // Skip, already up to date
    }

    // Download
    content, err := client.Get(ctx, fmt.Sprintf(
        "mau://%s%s",
        peer.Address,
        file.Path,
    ), nil)
    
    // 6. Verify signature
    if !verifySignature(content, aliceFingerprint) {
        return ErrInvalidSignature
    }
    
    // 7. Decrypt
    decrypted, err := pgp.Decrypt(content, account.PrivateKey)
    
    // 8. Save locally
    saveFile(file.Path, decrypted)
}
```

### DHT Peer Discovery

```go
// Bootstrap into network
bootstrapPeers := []*Peer{
    {Fingerprint: ..., Address: "bootstrap1.mau.network:443"},
    {Fingerprint: ..., Address: "bootstrap2.mau.network:443"},
}

// Join DHT
dht.Join(ctx, bootstrapPeers)

// Find specific peer
target := FingerprintFromString("5D000B2F...")
nearest := dht.FindPeer(ctx, target)

// Connect and sync
client, _ := account.Client(target, []string{nearest.Address})
client.Sync(ctx, target)
```

### Serving Files

```go
// Create server
server, err := account.Server(bootstrapPeers)
if err != nil {
    return err
}

// Listen (IPv4/IPv6 dual-stack)
listener, err := ListenTCP(":8080")
if err != nil {
    return err
}

// Start serving
externalAddress := "mau.example.com:443"
go server.Serve(listener, externalAddress)

// Graceful shutdown
defer server.Close()
```

### Handling Range Requests

```go
// Client: Resume download
resp, err := client.Get(ctx, fileURL, map[string]string{
    "Range": "bytes=1024-",  // Resume from byte 1024
})

// Server: Handled automatically by http.ServeContent
// Supports:
// - Range: bytes=0-1023      (first 1KB)
// - Range: bytes=1024-       (from 1KB to end)
// - Range: bytes=-1024       (last 1KB)
// - Range: bytes=0-1023,2048-3071  (multiple ranges)
```

### mDNS Local Discovery

In addition to DHT, peers announce themselves on the local network:

```go
// mDNS Service Type
serviceType := "_mau._tcp"

// Announce
service, _ := mdns.NewMDNSService(
    fingerprint.String(),  // Instance name
    serviceType,
    "",                    // Domain (default .local)
    "",                    // Hostname (auto-detect)
    port,
    nil,                   // IPs (auto-detect)
    []string{},            // TXT records
)
server, _ := mdns.NewServer(&mdns.Config{Zone: service})

// Discover
entries := make(chan *mdns.ServiceEntry, 10)
mdns.Lookup(serviceType, entries)

for entry := range entries {
    fingerprint := entry.Name
    address := fmt.Sprintf("%s:%d", entry.AddrV4, entry.Port)
    // Connect to peer...
}
```

---

## Best Practices

### 1. Always Verify Signatures

```go
// After downloading a file
if !file.VerifySignature(account, expectedFingerprint) {
    return ErrInvalidSignature
}
```

### 2. Use If-Modified-Since

```go
// Efficient incremental sync
lastSync := getLastSyncTime(peer)
files := client.ListFiles(peer, lastSync)
```

### 3. Implement Backoff

```go
// Avoid hammering peers
if time.Since(lastPing[peer]) < 30*time.Second {
    return // Skip ping
}
```

### 4. Handle Network Errors Gracefully

```go
// Retry with exponential backoff
for attempt := 0; attempt < 5; attempt++ {
    if err := client.Sync(peer); err == nil {
        break
    }
    time.Sleep(time.Second * (1 << attempt))  // 1s, 2s, 4s, 8s, 16s
}
```

### 5. Limit File Sizes

```go
// Check size before downloading
const maxFileSize = 100 * 1024 * 1024  // 100 MB
if file.Size > maxFileSize {
    return ErrFileTooLarge
}
```

### 6. Use Context for Timeouts

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
files, err := client.ListFiles(ctx, peer)
```

---

## Protocol Versioning

### Current Version: v1

The Mau HTTP API is currently at **v1**. Future versions will:
- Add version negotiation via HTTP headers
- Maintain backward compatibility
- Use content negotiation for format changes

### Future Extensions

Potential additions (not yet implemented):
- **WebSocket subscriptions**: Real-time file updates
- **GraphQL endpoint**: Complex queries
- **Batch operations**: Download multiple files in one request
- **Compression**: Gzip/Brotli for file lists

---

## Security Considerations

### 1. Mutual TLS Authentication

Both client and server MUST present valid certificates derived from PGP keys.

### 2. Fingerprint Verification

Never trust certificates without verifying the fingerprint matches expected value.

### 3. Signature Verification

Always verify file signatures after download before accepting content.

### 4. Rate Limiting

Implement rate limiting to prevent:
- DHT lookup floods
- Ping storms
- File download abuse

### 5. File Size Limits

Enforce reasonable file size limits to prevent disk exhaustion.

### 6. Timeout Configuration

Set conservative timeouts for all network operations:
- Connection: 10 seconds
- Read/Write: 30 seconds
- Idle: 120 seconds

---

## Related Documentation

- **[Core Concepts](02-core-concepts.md)** - Understanding Mau's architecture
- **[Authentication & Encryption](05-authentication.md)** - PGP operations
- **[Peer-to-Peer Networking](06-networking.md)** - Kademlia DHT details
- **[Building Social Apps](08-building-social-apps.md)** - Practical examples

---

## Reference Implementation

The canonical implementation is in the [mau-network/mau](https://github.com/mau-network/mau) repository:

- `server.go` - HTTP server implementation
- `client.go` - HTTP client implementation
- `kademlia.go` - DHT implementation

For questions or clarifications, please open an issue on GitHub.
