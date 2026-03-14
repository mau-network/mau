# TODO — TypeScript Implementation Review

This file tracks known issues, incomplete implementations, and missing features
identified during a code review of the `typescript/` directory.

---

## Critical / High Priority

### ~~1. Enforce signature verification in `file.ts`~~ ✓ Fixed
`file.ts` now throws `MauError('SIGNATURE_VERIFICATION_FAILED')` instead of
logging a warning when `decryptAndVerify()` returns `verified = false`.

### ~~2. Add mTLS authentication to the HTTP client~~ ✓ Fixed
`Client` now performs a PGP challenge-response handshake (`performHandshake`)
before the first request to any peer, verifying the server's fingerprint and
signature. `Server` exposes `GET /p2p/<fingerprint>/auth?challenge=<hex>` to
respond to the handshake.

---

## Medium Priority

### ~~3. Implement `generateCertificate()` in `crypto/pgp.ts`~~ ✓ Fixed
Implemented using Web Crypto API (`SubtleCrypto`) with a hand-rolled ASN.1 DER
encoder. Generates an ephemeral ECDSA P-256 key pair, builds a self-signed
X.509 v3 certificate with the PGP fingerprint in the Subject CN and a `pgp:`
URI Subject Alternative Name, and returns DER-encoded cert + PKCS#8 key.

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

### 5. Retry failed file downloads during sync
**File:** `typescript/src/client.ts:262`

When an individual file download fails during a sync cycle, the error is
logged and `stats.errors` is incremented, but the file is never retried.
Transient network errors will silently leave the local copy out of date.

**Fix:** Apply the existing `withRetry()` helper around the per-file download,
and optionally surface the list of permanently failed files to the caller.

---

### 6. Fix type-unsafe patterns across the codebase
**Files:**
- `typescript/src/network/resolvers.ts:46` — `any` type
- `typescript/tests/client-edge.test.ts:106` — `as any` cast
- `typescript/tests/client.test.ts:110` — `@ts-expect-error` for private
  field access

**Fix:** Replace `any` with proper types; expose internal state via protected
accessors or test-only helpers instead of suppressing TypeScript errors.

---

### 7. Add a CI workflow to ensure tests are runnable
**File:** CI config missing

Running `npm test` in a clean environment fails because Jest is listed as a
dev dependency but `node_modules` is absent. CI should install deps and run
tests automatically.

**Fix:** Add a GitHub Actions workflow (`.github/workflows/test.yml`) that
runs `npm ci && npm test` and verify the Jest ESM config in `jest.config.ts`.

---

## Low Priority / Future Work

### 8. Implement mDNS peer discovery
**File:** `typescript/package.json` (optional dep: `multicast-dns@^7.2.5`)

The optional `multicast-dns` dependency is listed but never used. Local
network peer discovery would allow automatic connection between devices on
the same LAN without any signaling server.

**Fix:** Add an `mdnsResolver()` in `network/resolvers.ts` that advertises
and queries `_mau._tcp.local` service records.

---

### 9. Harden the WebRTC mTLS handshake against race conditions
**File:** `typescript/src/network/webrtc.ts:109`

The ordering of promise creation and message-handler registration is manually
sequenced to avoid a race condition, with a comment acknowledging the
fragility. The 10-second timeout throws without retry.

**Fix:** Refactor using a state machine or structured async queue so the
ordering constraint is enforced mechanically. Add retry logic for transient
ICE failures.

---

### 10. Add retry to WebRTC request timeout
**File:** `typescript/src/network/webrtc.ts`

Individual HTTP-over-DataChannel requests have a hard timeout with no retry.
A single slow or dropped message aborts the whole operation.

**Fix:** Wrap requests with the same `withRetry()` logic used in the HTTP
client, with configurable timeout and max attempts.

---

### 11. Type the DNS browser-unsupported failure distinctly
**File:** `typescript/src/network/resolvers.ts`

In a browser environment `dnsResolver()` returns `null` rather than throwing.
Callers cannot distinguish "peer not found" from "DNS unsupported here".

**Fix:** Throw a typed `MauError` with code `DNS_NOT_SUPPORTED_IN_BROWSER`
so callers can branch on environment vs. lookup failure.

---

### 12. Add certificate pinning to the HTTP client
**File:** `typescript/src/client.ts`

The HTTP client accepts any valid TLS certificate. Pinning each server's
certificate fingerprint to its PGP key would prevent MITM attacks even
without full mTLS.

---

## Summary Table

| # | Area | Severity | File |
|---|------|----------|------|
| ~~1~~ | ~~Signature verification not enforced~~ | ~~High~~ | ~~`file.ts:109`~~ ✓ |
| ~~2~~ | ~~HTTP client missing mTLS auth~~ | ~~High~~ | ~~`client.ts:147`~~ ✓ |
| ~~3~~ | ~~`generateCertificate()` not implemented~~ | ~~Medium~~ | ~~`crypto/pgp.ts`~~ ✓ |
| 4 | DHT stub not Kademlia-compatible | Medium | `network/resolvers.ts:105` |
| 5 | No retry for failed file downloads | Medium | `client.ts:262` |
| 6 | Type-unsafe `any` / `@ts-expect-error` | Medium | multiple |
| 7 | Tests not runnable without CI workflow | Medium | CI config missing |
| 8 | mDNS discovery unused | Low | `resolvers.ts` |
| 9 | WebRTC mTLS handshake race condition | Low | `webrtc.ts:109` |
| 10 | No retry on WebRTC request timeout | Low | `webrtc.ts` |
| 11 | DNS browser failure not typed | Low | `resolvers.ts` |
| 12 | No certificate pinning in HTTP client | Low | `client.ts` |
