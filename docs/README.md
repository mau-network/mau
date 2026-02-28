# Mau Developer Documentation

Welcome to the **Mau** developer documentation! This guide will help you understand Mau as a concept and show you how to build peer-to-peer social applications using the Mau convention.

## Documentation Structure

### Getting Started
- **[Introduction](01-introduction.md)** - What is Mau and why it exists
- **[Core Concepts](02-core-concepts.md)** - Understanding Mau's fundamental architecture
- **Quick Start Tutorials** - Three progressive tutorials:
  - **[GPG Fundamentals](03a-quickstart-gpg.md)** - Understand the primitives (15 min)
  - **[Mau CLI](03b-quickstart-cli.md)** - Use the command-line tool (10 min)
  - **[Mau Package](03c-quickstart-package.md)** - Build applications with Go (20 min)

### Building Applications
- **[Storage and Data Format](04-storage-and-data.md)** - Working with files, JSON-LD, and Schema.org
- **[Authentication & Encryption](05-authentication.md)** - PGP, signing, and privacy
- **[Peer-to-Peer Networking](06-networking.md)** - Kademlia, discovery, and syncing
- **[HTTP API Reference](07-http-api.md)** - Server endpoints and client requests

### Advanced Topics
- **[Building Social Apps](08-building-social-apps.md)** - Practical patterns and examples
- **[Privacy & Security](09-privacy-security.md)** - Best practices for handling user data
- **[Performance & Optimization](10-performance.md)** - Scaling and efficiency tips

### Reference
- **[API Documentation](11-api-reference.md)** - Complete Go package reference
- **[Schema.org Types](12-schema-types.md)** - Common types for social applications
- **[Troubleshooting](13-troubleshooting.md)** - Common issues and solutions

## What is Mau?

**Mau** is a peer-to-peer convention for building decentralized social applications. Unlike traditional social networks that rely on central servers, Mau applications:

- Store data as **files on the user's filesystem**
- Use **PGP** for identity, authentication, and encryption
- Discover peers using **Kademlia DHT** and mDNS
- Exchange data over **HTTP/TLS** with mutual authentication
- Support **structured content** using JSON-LD and Schema.org vocabulary

## Why Build with Mau?

### For Developers
- **Simple implementation** - Files, HTTP, and well-established standards
- **Freedom to innovate** - No platform restrictions or approval processes
- **Interoperability** - Applications share the same data format
- **Web-compatible** - Existing websites can become Mau peers

### For Users
- **True ownership** - Data lives on their disk, not a company's server
- **Privacy by default** - End-to-end encryption with PGP
- **No censorship** - No central authority to ban or moderate
- **Cross-application** - One identity works across all Mau apps

## Architecture at a Glance

```
┌─────────────────────────────────────────────────────────────┐
│                      Your Application                        │
│  (Chat, Blog, Social Network, Game, IoT Controller...)      │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     Mau Convention                           │
│                                                              │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Storage   │  │ Peer Network │  │  HTTP API    │      │
│  │  (Files +   │  │  (Kademlia + │  │  (Sync +     │      │
│  │   PGP)      │  │   mDNS)      │  │   Serve)     │      │
│  └─────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│              User's Filesystem + Network                     │
└─────────────────────────────────────────────────────────────┘
```

## Quick Example

Here's a minimal Mau application that posts and syncs content:

```go
package main

import (
    "github.com/mau-network/mau"
    "log"
)

func main() {
    // Initialize Mau instance
    client, err := mau.NewClient("/path/to/mau/directory")
    if err != nil {
        log.Fatal(err)
    }

    // Create a social media post
    post := map[string]interface{}{
        "@context": "https://schema.org",
        "@type":    "SocialMediaPosting",
        "headline": "Hello, decentralized world!",
        "author": map[string]interface{}{
            "@type": "Person",
            "name":  "Alice",
        },
        "datePublished": "2026-02-27T07:30:00Z",
    }

    // Save encrypted and signed
    err = client.SavePost("hello-world.json", post)
    if err != nil {
        log.Fatal(err)
    }

    // Start syncing with peers
    client.StartSync()
}
```

That's it! Your post is now:
- ✅ Saved locally as an encrypted, signed file
- ✅ Available to your network over HTTP
- ✅ Syncing with peers automatically

## Next Steps

1. Read the **[Introduction](01-introduction.md)** to understand Mau's philosophy
2. Follow the **[Quick Start Guide](03-quickstart.md)** to build your first app
3. Explore **[Building Social Apps](08-building-social-apps.md)** for practical patterns

## Community & Support

- **GitHub**: [mau-network/mau](https://github.com/mau-network/mau)
- **Issues**: Report bugs and request features
- **Discussions**: Ask questions and share ideas

---

*This documentation is for developers building on Mau. For the full protocol specification, see the main [README.md](../README.md).*
