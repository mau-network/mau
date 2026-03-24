# Missing Features from Mau Specification

This document tracks features defined in the Mau spec (README.md) that are not yet implemented in the TypeScript version.

## 1. mDNS Service Discovery (CRITICAL)

**Status:** ❌ Not implemented  
**Spec Reference:** README.md, "MDNS Service discovery" section  
**Priority:** HIGH

The spec defines mDNS/mDNS-SD for local network peer discovery:

```
5D000B2F2C040A1675B49D7F0C7CB7DC36999D56._mau._tcp.local.
```

**What's needed:**
- Announce service on local network using mDNS multicast
- Service name format: `<Fingerprint>._mau._tcp.local.`
- Discovery of other peers on the same LAN

**Current workaround:** Manual peer address configuration via resolvers.

**Why missing:** TypeScript/browser environments don't have native mDNS support. Needs platform-specific implementation (Node.js: mdns/bonjour package; Browser: not possible without extension).

## 2. NAT Traversal (UPNP/NAT-PMP)

**Status:** ❌ Not implemented  
**Spec Reference:** README.md, "Listening on internet requests" section  
**Priority:** HIGH

The spec states:
> "The program is responsible for allowing the user to receive connections from outside of the local network by utilizing NAT traversal protocols such as UPNP, NAT-PMP, or Hole punching."

**What's needed:**
- UPNP port mapping
- NAT-PMP support
- Automatic port forwarding setup

**Current workaround:** Manual port forwarding or WebRTC hole punching.

**Why missing:** Platform-specific network stack access required. Not available in browser environments.

## 3. Content Browser UI

**Status:** ❌ Not implemented  
**Spec Reference:** README.md, Roadmap section  
**Priority:** MEDIUM

From the roadmap:
> [ ] **browser**: An interface to show content in chronological order

**What's needed:**
- UI component/library to display Mau content
- Chronological feed rendering
- Content type rendering (SocialMediaPosting, etc.)

**Current state:** The SDK provides all backend functionality (Account, Client, Server, File) but no UI layer.

## Summary

- **Core Protocol:** ✅ Implemented (Account, PGP, HTTP endpoints, Kademlia DHT, WebRTC)
- **Local Discovery:** ❌ Missing (mDNS)
- **NAT Traversal:** ❌ Missing (UPNP/NAT-PMP)
- **UI Layer:** ❌ Missing (content browser)

The TypeScript implementation focuses on the core protocol and is suitable for browser/Node.js environments where mDNS/NAT traversal are either impossible or handled by the host system.
