# TODO — TypeScript Implementation Review

This file tracks known issues, incomplete implementations, and missing features
identified during a code review of the `typescript/` directory.

---

## Critical / High Priority

### 1. Enforce signature verification in `file.ts`
**File:** `typescript/src/file.ts:109`

When `decryptAndVerify()` returns `verified = false`, the code only logs a
warning and continues processing the file. This means tampered or unsigned
files are silently accepted.

**Fix:** Throw a `MauError` (or reject the read) when signature verification
fails, and expose an optional `allowUnverified` flag for callers that need
the escape hatch.

---

### 2. Add mTLS authentication to the HTTP client
**File:** `typescript/src/client.ts:147`

The HTTP `Client` sends all sync requests unauthenticated. There is a TODO
comment in the source confirming this is not yet implemented. Peer identity
is never cryptographically verified, so a rogue server could serve arbitrary
content.

**Fix:** Implement a PGP-based challenge–response handshake (mirroring the
WebRTC mTLS flow in `webrtc.ts`) before the HTTP sync begins, or add a
signed header to every request.

---

## Medium Priority

### 3. Implement `generateCertificate()` in `crypto/pgp.ts`
**File:** `typescript/src/crypto/pgp.ts:285`

The function always throws `NOT_IMPLEMENTED`. TLS certificate generation is
needed for native TLS transport support.

**Fix:** Implement X.509 certificate generation using the Web Crypto API
(`SubtleCrypto`) or an existing library such as `@peculiar/x509`.

---

### 4. Replace the DHT stub with a real Kademlia implementation
**File:** `typescript/src/network/resolvers.ts:105`

The current DHT resolver is an HTTP polling stub that queries a single
bootstrap node at `/dht/peers/<fingerprint>`. It is **not compatible** with
the Go implementation's UDP-based Kademlia DHT.

**Fix:** Implement a UDP Kademlia DHT using the existing optional dependency
`k-bucket`, or integrate a library such as `bittorrent-dht`. Ensure the wire
protocol is compatible with the Go side.

---

### 5. Persist the DHT routing/cache table
**File:** `typescript/src/network/resolvers.ts`

The in-memory peer cache is lost on every page reload (browser) or process
restart (Node.js). This forces a cold start for every session.

**Fix:** Persist the routing table to `BrowserStorage` (IndexedDB) or
`FilesystemStorage`, and reload it on startup.

---

### 6. Retry failed file downloads during sync
**File:** `typescript/src/client.ts:262`

When an individual file download fails during a sync cycle, the error is
logged and `stats.errors` is incremented, but the file is never retried.
Transient network errors will silently leave the local copy out of date.

**Fix:** Apply the existing `withRetry()` helper around the per-file download,
and optionally surface the list of permanently failed files to the caller.

---

### 7. Fix type-unsafe patterns across the codebase
**Files:**
- `typescript/src/network/resolvers.ts:46` — `any` type
- `typescript/tests/client-edge.test.ts:106` — `as any` cast
- `typescript/tests/client.test.ts:110` — `@ts-expect-error` for private
  field access

**Fix:** Replace `any` with proper types; expose internal state via protected
accessors or test-only helpers instead of suppressing TypeScript errors.

---

### 8. Add a CI workflow to ensure tests are runnable
**File:** CI config missing

Running `npm test` in a clean environment fails because Jest is listed as a
dev dependency but `node_modules` is absent. CI should install deps and run
tests automatically.

**Fix:** Add a GitHub Actions workflow (`.github/workflows/test.yml`) that
runs `npm ci && npm test` and verify the Jest ESM config in `jest.config.ts`.

---

## Low Priority / Future Work

### 9. Implement mDNS peer discovery
**File:** `typescript/package.json` (optional dep: `multicast-dns@^7.2.5`)

The optional `multicast-dns` dependency is listed but never used. Local
network peer discovery would allow automatic connection between devices on
the same LAN without any signaling server.

**Fix:** Add an `mdnsResolver()` in `network/resolvers.ts` that advertises
and queries `_mau._tcp.local` service records.

---

### 10. Implement UPnP / NAT-PMP port forwarding

The codebase has no NAT traversal beyond WebRTC ICE. UPnP would allow the
HTTP server to be reachable from the internet without manual port forwarding.

**Fix:** Integrate a UPnP library (e.g., `nat-upnp`) and attempt port mapping
in `Server.start()`.

---

### 11. Harden the WebRTC mTLS handshake against race conditions
**File:** `typescript/src/network/webrtc.ts:109`

The ordering of promise creation and message-handler registration is manually
sequenced to avoid a race condition, with a comment acknowledging the
fragility. The 10-second timeout throws without retry.

**Fix:** Refactor using a state machine or structured async queue so the
ordering constraint is enforced mechanically. Add retry logic for transient
ICE failures.

---

### 12. Add retry to WebRTC request timeout
**File:** `typescript/src/network/webrtc.ts`

Individual HTTP-over-DataChannel requests have a hard timeout with no retry.
A single slow or dropped message aborts the whole operation.

**Fix:** Wrap requests with the same `withRetry()` logic used in the HTTP
client, with configurable timeout and max attempts.

---

### 13. Type the DNS browser-unsupported failure distinctly
**File:** `typescript/src/network/resolvers.ts`

In a browser environment `dnsResolver()` returns `null` rather than throwing.
Callers cannot distinguish "peer not found" from "DNS unsupported here".

**Fix:** Throw a typed `MauError` with code `DNS_NOT_SUPPORTED_IN_BROWSER`
so callers can branch on environment vs. lookup failure.

---

### 14. Add certificate pinning to the HTTP client
**File:** `typescript/src/client.ts`

The HTTP client accepts any valid TLS certificate. Pinning each server's
certificate fingerprint to its PGP key would prevent MITM attacks even
without full mTLS.

---

## Summary Table

| # | Area | Severity | File |
|---|------|----------|------|
| 1 | Signature verification not enforced | High | `file.ts:109` |
| 2 | HTTP client missing mTLS auth | High | `client.ts:147` |
| 3 | `generateCertificate()` not implemented | Medium | `crypto/pgp.ts:285` |
| 4 | DHT stub not Kademlia-compatible | Medium | `network/resolvers.ts:105` |
| 5 | DHT cache not persisted | Medium | `network/resolvers.ts` |
| 6 | No retry for failed file downloads | Medium | `client.ts:262` |
| 7 | Type-unsafe `any` / `@ts-expect-error` | Medium | multiple |
| 8 | Tests not runnable without CI workflow | Medium | CI config missing |
| 9 | mDNS discovery unused | Low | `resolvers.ts` |
| 10 | No UPnP NAT traversal | Low | `server.ts` |
| 11 | WebRTC mTLS handshake race condition | Low | `webrtc.ts:109` |
| 12 | No retry on WebRTC request timeout | Low | `webrtc.ts` |
| 13 | DNS browser failure not typed | Low | `resolvers.ts` |
| 14 | No certificate pinning in HTTP client | Low | `client.ts` |
