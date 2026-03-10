# Peer-to-Peer Networking

This guide explains how Mau discovers peers and synchronizes data across the network using a combination of **Kademlia DHT**, **mDNS**, and **direct HTTP/TLS connections**.

## Overview

Mau's networking layer is designed to enable truly decentralized peer-to-peer communication:

- **Local Discovery**: mDNS-SD finds peers on the same LAN automatically
- **Global Discovery**: Kademlia DHT enables internet-wide peer lookup
- **Secure Transport**: All connections use mutual TLS authentication with PGP-derived certificates
- **No Central Servers**: Each node is equal; no privileged infrastructure required

```
┌─────────────┐                          ┌─────────────┐
│   Peer A    │◄────── Local Network ────►│   Peer B    │
│  (Laptop)   │         (mDNS-SD)         │  (Desktop)  │
└──────┬──────┘                          └──────┬──────┘
       │                                        │
       └────────────► Kademlia DHT ◄───────────┘
                      (Internet-wide)
                            │
                            ▼
                    ┌──────────────┐
                    │   Peer C     │
                    │  (Server)    │
                    └──────────────┘
```

---

## Discovery Mechanisms

### 1. Local Network Discovery (mDNS-SD)

**What it is:**  
Multicast DNS Service Discovery (mDNS-SD) allows peers to advertise themselves on the local area network and discover each other without any central coordination.

**How it works:**
1. Each Mau instance announces itself via mDNS with service name `_mau._tcp.local`
2. The announcement includes:
   - PGP fingerprint (unique identifier)
   - IP address and port
3. Other peers on the same network listen for these announcements
4. When a peer needs to contact another peer, it queries mDNS for that fingerprint

**Use cases:**
- Home networks (laptop, phone, desktop syncing)
- Office LANs (team collaboration without internet)
- IoT devices on local networks
- Development and testing

**Code example:**
```go
import "github.com/mau-network/mau"

server, _ := mau.NewServer(account, ":8080")

// mDNS is enabled by default
// Peers on your LAN will automatically discover this instance
```

**Service format:**
```
<fingerprint>._mau._tcp.local.
```

Example: `ABC123DEF456._mau._tcp.local.` resolves to `192.168.1.100:8080`

---

### 2. Internet Discovery (Kademlia DHT)

**What it is:**  
Kademlia is a distributed hash table (DHT) protocol that enables decentralized peer lookup across the internet. It's the same technology used by BitTorrent and IPFS.

**Why Kademlia?**
- **Decentralized**: No central registry or DNS required
- **Efficient**: O(log n) lookup complexity
- **Robust**: Self-healing network topology
- **Scalable**: Works with millions of peers

#### How Kademlia Works

**Node IDs:**  
Each peer is identified by its PGP fingerprint (160 bits for RSA, 256 bits for Ed25519). The fingerprint serves as both the peer's identity and its position in the DHT keyspace.

**XOR Distance Metric:**  
Kademlia defines "closeness" using the XOR metric:
```
distance(A, B) = A ⊕ B  (bitwise XOR)
```

Peers that are "closer" (smaller XOR distance) are considered neighbors.

**Routing Table (k-buckets):**  
Each peer maintains a routing table with **160 buckets** (for RSA fingerprints), where:
- Bucket `i` stores peers at XOR distance `[2^i, 2^(i+1))`
- Each bucket holds up to **k = 20** peers
- Least-recently-seen (LRS) eviction policy

**Peer Lookup:**  
To find a peer with fingerprint `F`:
1. Query the **α = 3** closest peers you know about `F`
2. Each responds with their **k = 20** closest peers to `F`
3. Recursively query the newly discovered closest peers
4. Continue until `F` is found or no closer peers are discovered

**Routing Table Maintenance:**
- **Ping**: Verify peer liveness before evicting old contacts
- **Refresh**: Every hour, refresh stale buckets by performing random lookups
- **Implicit updates**: Add peers to routing table when they contact you

#### Kademlia Parameters

| Constant | Value | Description |
|----------|-------|-------------|
| `B` | 160 | Number of buckets (160 bits for v4 keys) |
| `K` | 20 | Max peers per bucket (replication factor) |
| `α` | 3 | Parallel lookup requests |
| `STALL_PERIOD` | 1 hour | Bucket refresh interval |
| `PING_MIN_BACKOFF` | 30 seconds | Minimum time between pings to same peer |

#### Joining the Network

To join the Kademlia network, you need at least one **bootstrap peer**:

```go
import "github.com/mau-network/mau"

bootstrap := []*mau.Peer{
    {
        Fingerprint: mau.MustParseFingerprintFromString("ABC123..."),
        Address:     "peer.example.com:8080",
    },
}

server, _ := mau.NewServer(account, ":8080", mau.WithBootstrapPeers(bootstrap))

// Server will:
// 1. Add bootstrap peer to routing table
// 2. Query for itself to discover neighbors
// 3. Refresh all buckets to populate routing table
// 4. Start periodic refresh of stale buckets
```

**What happens during join:**
1. Add bootstrap peers to routing table
2. Perform `FIND_PEER(self)` to discover neighbors
3. Refresh all buckets to populate routing table
4. Start background refresh process for stale buckets

#### DHT API Endpoints

Mau exposes two Kademlia RPC endpoints:

**1. Ping** (`GET /kad/ping`)  
Verify peer liveness.

**Request:**
```
GET https://<peer-address>/kad/ping
```

**Response:**
```
200 OK
```

**2. Find Peer** (`GET /kad/find_peer/{fingerprint}`)  
Find the k-closest peers to a given fingerprint.

**Request:**
```
GET https://<peer-address>/kad/find_peer/ABC123DEF456...
```

**Response:**
```json
[
  {
    "fingerprint": "ABC123DEF456...",
    "address": "192.168.1.100:8080"
  },
  {
    "fingerprint": "789XYZ012...",
    "address": "peer.example.com:8080"
  }
]
```

---

## Fingerprint Resolvers

Mau uses a **resolver pattern** to support multiple discovery strategies:

```go
type FingerprintResolver func(
    ctx context.Context,
    fingerprint Fingerprint,
    addresses chan<- string,
) error
```

A resolver receives a fingerprint and sends discovered addresses to the `addresses` channel.

### Built-in Resolvers

#### 1. Static Address
Always returns the same address (useful for testing or known peers):

```go
resolver := mau.StaticAddress("peer.example.com:8080")
```

#### 2. Local Friend Address (mDNS)
Discovers peers on the local network:

```go
resolver := mau.LocalFriendAddress
```

#### 3. Internet Friend Address (Kademlia)
Uses the DHT to find peers on the internet:

```go
resolver := mau.InternetFriendAddress(server)
```

### Using Resolvers

Resolvers are used internally by the `Client` when connecting to a peer:

```go
client, err := account.Client(
    targetFingerprint,
    []string{"fallback-address.com:8080"}, // fallback if resolvers fail
)
```

The client will:
1. Try all configured resolvers in parallel
2. Use the first address that successfully connects
3. Fall back to the provided addresses if all resolvers fail

---

## Connection Lifecycle

### Establishing a Connection

1. **Resolve Peer Address**
   - Try mDNS (local network)
   - Try Kademlia DHT (internet)
   - Fall back to known addresses

2. **TLS Handshake**
   - Client presents certificate derived from its PGP key
   - Server verifies client certificate fingerprint
   - Server presents certificate derived from its PGP key
   - Client verifies server certificate fingerprint

3. **Add to Routing Table**
   - When a peer contacts you, add them to your Kademlia routing table
   - Move existing peers to the tail (LRU update)
   - If bucket is full, ping least-recently-seen peer

4. **Request/Response**
   - Execute API call (ping, find_peer, sync, etc.)
   - Connection is encrypted and mutually authenticated

### Handling Peer Failures

**Detection:**
- Timeout on HTTP request
- TLS handshake failure
- Invalid certificate fingerprint

**Recovery:**
- Remove peer from routing table
- Try next closest peer (if doing lookup)
- Backoff on repeated failures

---

## Network Security

### Mutual TLS Authentication

Every connection requires **both peers** to present valid certificates:

```
Client                        Server
  │                             │
  ├─── ClientHello ────────────►│
  │                             │
  │◄─── ServerCertificate ──────┤
  │     (PGP-derived cert)      │
  │                             │
  ├─── ClientCertificate ──────►│
  │     (PGP-derived cert)      │
  │                             │
  │◄─── Verify Fingerprint ─────┤
  │                             │
  ├─── Verify Fingerprint ─────►│
  │                             │
  │◄─── Encrypted Channel ──────┤
```

**What this prevents:**
- Man-in-the-middle attacks
- Impersonation
- Eavesdropping
- Unauthorized access

### Privacy Considerations

**IP Address Exposure:**  
Your IP address is visible to:
- Peers you directly connect to
- Bootstrap peers
- Peers queried during DHT lookups

**Metadata Leakage:**  
The DHT reveals:
- Your fingerprint (public key hash)
- Approximate network location (bucket structure)
- Online/offline status

**Mitigation:**
- Use Tor or VPN for IP anonymity
- Run behind NAT/firewall
- Limit bootstrap peers to trusted nodes
- See [Privacy & Security](09-privacy-security.md) for more

---

## Network Topology

### Flat Peer-to-Peer

Mau is a **flat network**: all nodes are equal. There are no:
- Supernodes
- Master servers
- Privileged infrastructure
- Centralized registries

Every peer can:
- Accept connections from others
- Initiate connections to others
- Route DHT queries
- Store and serve data

### Network Address Translation (NAT)

**Problem:**  
Most home/office networks use NAT, which prevents incoming connections from the internet.

**Solution (optional):**  
Mau includes UPnP support to automatically configure port forwarding:

```go
server, _ := mau.NewServer(account, ":8080", mau.WithUPnP())
```

If UPnP succeeds, the server becomes publicly reachable on the internet.

**Fallback:**
- Peers behind NAT can still initiate outgoing connections
- They can participate in the DHT (respond to queries from peers they contacted)
- For full bidirectional reachability, manually configure port forwarding or use a relay

---

## Practical Examples

### Example 1: Local-Only Network (No Internet)

```go
package main

import "github.com/mau-network/mau"

func main() {
    account, _ := mau.LoadAccount("./my-account")
    
    // Only mDNS, no DHT
    server, _ := mau.NewServer(account, ":8080")
    
    // Peers on your LAN will auto-discover this instance
    server.Start()
}
```

### Example 2: Internet-Wide Network with Bootstrap

```go
package main

import "github.com/mau-network/mau"

func main() {
    account, _ := mau.LoadAccount("./my-account")
    
    bootstrap := []*mau.Peer{
        {
            Fingerprint: mau.MustParseFingerprintFromString("ABC123..."),
            Address:     "bootstrap.mau-network.org:8080",
        },
    }
    
    server, _ := mau.NewServer(
        account,
        ":8080",
        mau.WithBootstrapPeers(bootstrap),
        mau.WithUPnP(), // optional: auto-configure port forwarding
    )
    
    server.Start()
}
```

### Example 3: Client-Only (No Incoming Connections)

```go
package main

import "github.com/mau-network/mau"

func main() {
    account, _ := mau.LoadAccount("./my-account")
    
    // Connect to a friend
    friendFingerprint := mau.MustParseFingerprintFromString("DEF789...")
    client, _ := account.Client(
        friendFingerprint,
        []string{"friend.example.com:8080"},
    )
    
    // Fetch friend's data
    posts, _ := client.List("/posts")
    // ...
}
```

---

## Debugging Network Issues

### Check Routing Table

```bash
# List all peers in routing table
mau peers list

# Show detailed bucket information
mau peers buckets
```

### Test Peer Discovery

```bash
# Test mDNS discovery
mau discover --local

# Test Kademlia lookup
mau discover --fingerprint ABC123DEF456...
```

### Monitor Network Traffic

```bash
# Enable verbose logging
export MAU_LOG_LEVEL=debug
mau server start

# Logs will show:
# - Peer connections
# - DHT queries
# - Routing table updates
```

---

## Performance Tuning

### Bootstrap Peer Selection

**Use multiple bootstrap peers:**
```go
bootstrap := []*mau.Peer{
    {Fingerprint: ..., Address: "bootstrap1.example.com:8080"},
    {Fingerprint: ..., Address: "bootstrap2.example.com:8080"},
    {Fingerprint: ..., Address: "bootstrap3.example.com:8080"},
}
```

**Why:**
- Redundancy (if one is offline)
- Faster routing table population
- Geographic diversity (lower latency)

### Parallelism Factor (α)

The `α` parameter controls how many peers are queried in parallel during lookup. Default is **α = 3**.

Higher α = faster lookups, more bandwidth.  
Lower α = slower lookups, less bandwidth.

### Bucket Refresh Interval

Buckets are refreshed every `STALL_PERIOD = 1 hour` by default. For more active networks:

```go
// Adjust in constants.go (requires rebuilding)
const dht_STALL_PERIOD = 30 * time.Minute
```

---

## Next Steps

- **[HTTP API Reference](07-http-api.md)** - Learn about the full REST API
- **[Building Social Apps](08-building-social-apps.md)** - Practical patterns for decentralized apps
- **[Privacy & Security](09-privacy-security.md)** - Best practices for protecting user data

---

## Further Reading

- **Kademlia Paper**: [Kademlia: A Peer-to-Peer Information System Based on the XOR Metric](https://pdos.csail.mit.edu/~petar/papers/maymounkov-kademlia-lncs.pdf)
- **mDNS-SD RFC**: [RFC 6762 - Multicast DNS](https://datatracker.ietf.org/doc/html/rfc6762)
- **BitTorrent DHT**: [BEP 5 - DHT Protocol](http://www.bittorrent.org/beps/bep_0005.html)
