# Mau TypeScript Implementation - Spec Compliance Issues

**Last Updated:** 2026-03-30

## Missing Features

### 1. MDNS Service Discovery (§ Peer-to-Peer Stack)
**Spec Requirement:** "If local network is desired the service may announce itself on the network using MDNS multicast."

**Status:** Not implemented

**Impact:** Local network peer discovery unavailable. Users cannot discover peers on the same LAN without manual address configuration.

**Spec Format:**
```
5D000B2F2C040A1675B49D7F0C7CB7DC36999D56._mau._tcp.local.
|------------ Fingerprint --------------|app-|protocol|domain|
```

### 2. NAT Traversal (§ Listening on internet requests)
**Spec Requirement:** "The program is responsible for allowing the user to receive connections from outside of the local network by utilizing NAT traversal protocols such as UPNP, NAT-PMP, or Hole punching."

**Status:** Not implemented

**Impact:** Users behind NAT cannot receive inbound connections. While DHT relay signaling helps establish WebRTC connections, there's no automatic port forwarding via UPNP/NAT-PMP for the HTTP server.

## Recently Fixed Issues

### ✅ If-Modified-Since Header (§ Data Exchange) — Fixed in PR #82
**Spec Requirement:** "Client will send the update time as a value for the HTTP header `If-Modified-Since`"

**Was:** Using query parameter `?after=<date>` instead of standard HTTP header

**Fixed:** Client now sends `If-Modified-Since` header per spec

### ✅ Friend Keys Encryption (§ Why are friends' keys encrypted?) — Fixed
**Spec Requirement:** "All friends' public keys should be encrypted with the account key"

**Was:** Keys saved in binary format but NOT encrypted

**Fixed:** Keys now encrypted with account's public key before writing to disk

### ✅ Version File Extensions — Fixed
**Spec Requirement:** Version files should have `.pgp` extension

**Was:** Version files saved without `.pgp` extension

**Fixed:** All version files now use `.pgp` extension

## Implementation Quality

All core features are implemented:
- ✅ PGP identity management
- ✅ File encryption/signing
- ✅ Versioning with SHA-256 checksums
- ✅ HTTP server interface (file list, file download, Range support)
- ✅ Kademlia DHT over WebRTC with relay signaling
- ✅ Client sync with retry and backoff
- ✅ mTLS challenge-response authentication
- ✅ Friend keys encrypted with account key
- ✅ If-Modified-Since header for incremental sync

## Summary

**Missing Features:** MDNS discovery, NAT traversal (requires native libs)  
**Recent Fixes:** If-Modified-Since header, friend key encryption, version file extensions

The TypeScript implementation is solid and follows the spec closely. Remaining gaps are optional features that require native libraries (mDNS, UPNP/NAT-PMP).
