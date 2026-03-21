# Performance & Optimization

This guide covers performance considerations and optimization strategies for building efficient Mau applications.

## Table of Contents

- [Performance Overview](#performance-overview)
- [Storage Optimization](#storage-optimization)
- [Network Performance](#network-performance)
- [DHT Optimization](#dht-optimization)
- [HTTP Server Tuning](#http-server-tuning)
- [Encryption Performance](#encryption-performance)
- [Caching Strategies](#caching-strategies)
- [Monitoring & Profiling](#monitoring--profiling)

---

## Performance Overview

Mau applications have three main performance domains:

1. **Storage I/O** - Reading/writing files from disk
2. **Network operations** - DHT lookups, peer discovery, HTTP sync
3. **Cryptographic operations** - PGP signing, encryption, verification

### Performance Goals

- **Fast startup**: Initialize in < 1 second
- **Responsive sync**: Detect and sync new content in < 5 seconds
- **Efficient storage**: Minimize disk I/O and redundant writes
- **Low bandwidth**: Avoid re-fetching unchanged content
- **Scalable discovery**: Handle 1000+ peers efficiently

---

## Storage Optimization

### File System Layout

**Best Practice: Flat directory structure**

```
mau-directory/
├── posts/
│   ├── 2026-03-01-hello.json
│   ├── 2026-03-02-update.json
│   └── ...
├── media/
│   ├── photo-1.jpg
│   └── video-1.mp4
└── identity/
    └── key.asc
```

**Why**: Reduces inode lookups and directory traversal overhead.

**Avoid**: Deep nested directories (e.g., `/posts/2026/03/01/hello.json`)

### File I/O Best Practices

#### 1. Batch Writes

```go
// ❌ Bad: Multiple writes in a loop
for _, post := range posts {
    client.SavePost(post.ID, post)
}

// ✅ Good: Batch operation
client.BatchSave(posts)
```

#### 2. Use Memory-Mapped Files for Large Reads

```go
import "syscall"

func readLargeFile(path string) ([]byte, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    stat, _ := f.Stat()
    data, err := syscall.Mmap(int(f.Fd()), 0, int(stat.Size()),
        syscall.PROT_READ, syscall.MAP_SHARED)
    if err != nil {
        return nil, err
    }
    defer syscall.Munmap(data)

    return data, nil
}
```

#### 3. Avoid Redundant Stat Calls

```go
// ❌ Bad: Multiple stat calls
if fileExists(path) {
    size := getFileSize(path)
    modTime := getModTime(path)
}

// ✅ Good: Single stat call
info, err := os.Stat(path)
if err == nil {
    size := info.Size()
    modTime := info.ModTime()
}
```

### File Watching Optimization

Use `fsnotify` efficiently:

```go
import "github.com/fsnotify/fsnotify"

func watchDirectory(path string) error {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return err
    }

    // ✅ Watch only the directories that matter
    watcher.Add(filepath.Join(path, "posts"))
    watcher.Add(filepath.Join(path, "media"))

    // Debounce rapid events
    var timer *time.Timer
    for {
        select {
        case event := <-watcher.Events:
            if timer != nil {
                timer.Stop()
            }
            timer = time.AfterFunc(500*time.Millisecond, func() {
                handleEvent(event)
            })
        }
    }
}
```

---

## Network Performance

### Connection Pooling

Reuse HTTP connections to peers:

```go
import "net/http"

var client = &http.Client{
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
        DisableCompression:  false, // Enable gzip
    },
    Timeout: 30 * time.Second,
}
```

### Parallel Peer Sync

```go
func syncFromPeers(peers []Peer) {
    sem := make(chan struct{}, 10) // Limit concurrency to 10
    var wg sync.WaitGroup

    for _, peer := range peers {
        wg.Add(1)
        go func(p Peer) {
            defer wg.Done()
            sem <- struct{}{}        // Acquire
            defer func() { <-sem }() // Release
            syncFromPeer(p)
        }(peer)
    }

    wg.Wait()
}
```

### HTTP Response Compression

Enable compression for large responses:

```go
import "github.com/klauspost/compress/gzhttp"

func setupServer() *http.Server {
    mux := http.NewServeMux()
    mux.HandleFunc("/posts", handlePosts)

    // ✅ Wrap with compression middleware
    handler := gzhttp.GzipHandler(mux)

    return &http.Server{
        Addr:    ":8080",
        Handler: handler,
    }
}
```

### Conditional Requests (ETag/If-None-Match)

```go
func handlePost(w http.ResponseWriter, r *http.Request) {
    postID := r.URL.Query().Get("id")
    post, etag := loadPost(postID)

    // Check client's cached version
    if r.Header.Get("If-None-Match") == etag {
        w.WriteHeader(http.StatusNotModified)
        return
    }

    w.Header().Set("ETag", etag)
    w.Header().Set("Cache-Control", "max-age=300")
    json.NewEncoder(w).Encode(post)
}
```

---

## DHT Optimization

### Routing Table Tuning

```go
import "github.com/libp2p/go-libp2p-kad-dht"

func newOptimizedDHT() (*dht.IpfsDHT, error) {
    return dht.New(ctx, host,
        dht.BucketSize(20),              // K-bucket size
        dht.Concurrency(10),             // Parallel lookups
        dht.RoutingTableRefreshPeriod(5*time.Minute),
        dht.RoutingTableLatencyTolerance(10*time.Second),
    )
}
```

### Peer Discovery Strategies

#### 1. Bootstrap Nodes

Maintain a list of reliable bootstrap peers:

```go
var bootstrapPeers = []string{
    "/ip4/1.2.3.4/tcp/4001/p2p/Qm...",
    "/dns4/bootstrap.example.com/tcp/4001/p2p/Qm...",
}

func bootstrapDHT(dht *dht.IpfsDHT) error {
    ctx := context.Background()
    for _, addr := range bootstrapPeers {
        peerAddr, _ := multiaddr.NewMultiaddr(addr)
        peerInfo, _ := peer.AddrInfoFromP2p(peerAddr)
        dht.Host().Connect(ctx, *peerInfo)
    }
    return dht.Bootstrap(ctx)
}
```

#### 2. Local Discovery (mDNS)

Enable mDNS for LAN peers:

```go
import "github.com/libp2p/go-libp2p/p2p/discovery/mdns"

func setupMDNS(host host.Host) error {
    service := mdns.NewMdnsService(host, "mau-local", &discoveryHandler{})
    return service.Start()
}

type discoveryHandler struct{}

func (h *discoveryHandler) HandlePeerFound(pi peer.AddrInfo) {
    // Connect to local peer immediately (low latency)
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    host.Connect(ctx, pi)
}
```

### DHT Query Optimization

```go
// ❌ Bad: Sequential lookups
for _, key := range keys {
    value := dht.GetValue(ctx, key)
}

// ✅ Good: Batch queries with context deadline
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

results := make(chan Result, len(keys))
for _, key := range keys {
    go func(k string) {
        value, err := dht.GetValue(ctx, k)
        results <- Result{Key: k, Value: value, Err: err}
    }(key)
}

for range keys {
    result := <-results
    handleResult(result)
}
```

---

## HTTP Server Tuning

### Server Configuration

```go
server := &http.Server{
    Addr:         ":8080",
    Handler:      handler,
    ReadTimeout:  15 * time.Second,
    WriteTimeout: 15 * time.Second,
    IdleTimeout:  60 * time.Second,
    MaxHeaderBytes: 1 << 20, // 1 MB

    // ✅ Connection limits
    ConnState: func(conn net.Conn, state http.ConnState) {
        if state == http.StateNew {
            // Track active connections
            activeConns.Add(1)
        } else if state == http.StateClosed {
            activeConns.Add(-1)
        }
    },
}
```

### Rate Limiting

Protect against abusive peers:

```go
import "golang.org/x/time/rate"

var limiters = sync.Map{}

func rateLimitMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        peerID := getPeerID(r)

        limiterAny, _ := limiters.LoadOrStore(peerID, rate.NewLimiter(10, 20))
        limiter := limiterAny.(*rate.Limiter)

        if !limiter.Allow() {
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

---

## Encryption Performance

### PGP Operation Optimization

#### 1. Key Caching

```go
var keyCache = sync.Map{}

func getPublicKey(keyID string) (*openpgp.Entity, error) {
    if cached, ok := keyCache.Load(keyID); ok {
        return cached.(*openpgp.Entity), nil
    }

    key, err := loadKeyFromDisk(keyID)
    if err != nil {
        return nil, err
    }

    keyCache.Store(keyID, key)
    return key, nil
}
```

#### 2. Batch Signature Verification

```go
// ✅ Verify multiple signatures in parallel
func verifyBatch(messages []SignedMessage) []error {
    results := make([]error, len(messages))
    var wg sync.WaitGroup

    for i, msg := range messages {
        wg.Add(1)
        go func(idx int, m SignedMessage) {
            defer wg.Done()
            results[idx] = verifySignature(m)
        }(i, msg)
    }

    wg.Wait()
    return results
}
```

#### 3. Symmetric Encryption for Large Files

Use hybrid encryption:

```go
// ✅ Encrypt data key with PGP, file with AES
func encryptLargeFile(data []byte, recipientKey *openpgp.Entity) ([]byte, error) {
    // Generate random AES key
    aesKey := make([]byte, 32)
    rand.Read(aesKey)

    // Encrypt data with AES-GCM (fast)
    ciphertext := encryptAESGCM(data, aesKey)

    // Encrypt AES key with PGP (small overhead)
    encryptedKey := encryptPGP(aesKey, recipientKey)

    return append(encryptedKey, ciphertext...), nil
}
```

---

## Caching Strategies

### In-Memory Content Cache

```go
import "github.com/hashicorp/golang-lru/v2"

type ContentCache struct {
    cache *lru.Cache[string, []byte]
}

func NewContentCache(size int) (*ContentCache, error) {
    cache, err := lru.New[string, []byte](size)
    if err != nil {
        return nil, err
    }
    return &ContentCache{cache: cache}, nil
}

func (c *ContentCache) Get(key string) ([]byte, bool) {
    return c.cache.Get(key)
}

func (c *ContentCache) Set(key string, value []byte) {
    c.cache.Add(key, value)
}
```

### Peer Metadata Cache

```go
type PeerCache struct {
    lastSeen   map[string]time.Time
    lastHash   map[string]string
    mu         sync.RWMutex
}

func (pc *PeerCache) ShouldSync(peerID, contentHash string) bool {
    pc.mu.RLock()
    defer pc.mu.RUnlock()

    lastHash, exists := pc.lastHash[peerID]
    if !exists {
        return true // Never synced
    }

    return lastHash != contentHash // Hash changed
}

func (pc *PeerCache) Update(peerID, contentHash string) {
    pc.mu.Lock()
    defer pc.mu.Unlock()

    pc.lastSeen[peerID] = time.Now()
    pc.lastHash[peerID] = contentHash
}
```

---

## Monitoring & Profiling

### Built-in Metrics

Expose Prometheus-compatible metrics:

```go
import "github.com/prometheus/client_golang/prometheus/promhttp"

var (
    syncCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "mau_sync_total",
            Help: "Total number of sync operations",
        },
        []string{"peer", "status"},
    )

    syncDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "mau_sync_duration_seconds",
            Help:    "Sync operation duration",
            Buckets: prometheus.DefBuckets,
        },
        []string{"peer"},
    )
)

func init() {
    prometheus.MustRegister(syncCounter)
    prometheus.MustRegister(syncDuration)
}

func main() {
    http.Handle("/metrics", promhttp.Handler())
    http.ListenAndServe(":9090", nil)
}
```

### Go Profiling

Enable pprof endpoints:

```go
import _ "net/http/pprof"

func main() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()

    // Your application code
}
```

**Usage**:

```bash
# CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Memory profile
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine profile
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

### Performance Benchmarks

Write benchmarks for critical paths:

```go
func BenchmarkSyncOperation(b *testing.B) {
    client := setupTestClient()
    peer := setupTestPeer()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        client.SyncFrom(peer)
    }
}

func BenchmarkEncryption(b *testing.B) {
    data := make([]byte, 1024*1024) // 1 MB
    key := generateTestKey()

    b.ResetTimer()
    b.SetBytes(int64(len(data)))
    for i := 0; i < b.N; i++ {
        encrypt(data, key)
    }
}
```

---

## Performance Checklist

Use this checklist when optimizing a Mau application:

### Storage
- [ ] Use flat directory structure
- [ ] Batch file operations where possible
- [ ] Implement file watching with debouncing
- [ ] Cache frequently-accessed files in memory

### Network
- [ ] Enable HTTP connection pooling
- [ ] Implement response compression (gzip)
- [ ] Use ETags for conditional requests
- [ ] Limit concurrent peer connections (10-20)

### DHT
- [ ] Configure appropriate bucket size (20)
- [ ] Enable mDNS for local discovery
- [ ] Use batch queries for multiple lookups
- [ ] Refresh routing table periodically (5 min)

### Encryption
- [ ] Cache decrypted public keys
- [ ] Batch signature verifications
- [ ] Use AES for large file encryption (hybrid approach)

### Monitoring
- [ ] Expose Prometheus metrics
- [ ] Enable pprof endpoints (development)
- [ ] Write benchmarks for critical operations
- [ ] Monitor sync latency and success rate

---

## Next Steps

- **[API Documentation](11-api-reference.md)** - Complete Go package reference
- **[Troubleshooting](13-troubleshooting.md)** - Debug performance issues
- **[Building Social Apps](08-building-social-apps.md)** - Apply optimizations to real apps

---

*For questions or contributions, visit [mau-network/mau](https://github.com/mau-network/mau).*
