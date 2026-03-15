# TODO — TypeScript Implementation Review

This file tracks known issues, incomplete implementations, and missing features
identified during a code review of the `typescript/` directory.

---

## Low Priority / Future Work

### 1. Harden the WebRTC mTLS handshake against race conditions
**File:** `typescript/src/network/webrtc.ts:109`

The ordering of promise creation and message-handler registration is manually
sequenced to avoid a race condition, with a comment acknowledging the
fragility. The 10-second timeout throws without retry.

**Fix:** Refactor using a state machine or structured async queue so the
ordering constraint is enforced mechanically. Add retry logic for transient
ICE failures.

---

### 2. Add retry to WebRTC request timeout
**File:** `typescript/src/network/webrtc.ts`

Individual HTTP-over-DataChannel requests have a hard timeout with no retry.
A single slow or dropped message aborts the whole operation.

**Fix:** Wrap requests with the same `withRetry()` logic used in the HTTP
client, with configurable timeout and max attempts.

---

### 3. Type the DNS browser-unsupported failure distinctly
**File:** `typescript/src/network/resolvers.ts`

In a browser environment `dnsResolver()` returns `null` rather than throwing.
Callers cannot distinguish "peer not found" from "DNS unsupported here".

**Fix:** Throw a typed `MauError` with code `DNS_NOT_SUPPORTED_IN_BROWSER`
so callers can branch on environment vs. lookup failure.

---

### 4. Add certificate pinning to the HTTP client
**File:** `typescript/src/client.ts`

The HTTP client accepts any valid TLS certificate. Pinning each server's
certificate fingerprint to its PGP key would prevent MITM attacks even
without full mTLS.

---

## Summary Table

| # | Area | Severity | File |
|---|------|----------|------|
| 1 | WebRTC mTLS handshake race condition | Low | `webrtc.ts:109` |
| 2 | No retry on WebRTC request timeout | Low | `webrtc.ts` |
| 3 | DNS browser failure not typed | Low | `resolvers.ts` |
| 4 | No certificate pinning in HTTP client | Low | `client.ts` |
