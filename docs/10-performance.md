# Performance & Optimization

This guide covers best practices for building high-performance Mau applications. Since Mau is a peer-to-peer system, performance considerations span local storage, network operations, and cryptographic operations.

## Table of Contents

1. [Performance Principles](#performance-principles)
2. [Storage Optimization](#storage-optimization)
3. [Cryptography Performance](#cryptography-performance)
4. [Network Optimization](#network-optimization)
5. [DHT and Discovery](#dht-and-discovery)
6. [Caching Strategies](#caching-strategies)
7. [Monitoring and Profiling](#monitoring-and-profiling)
8. [Scalability Patterns](#scalability-patterns)

## Performance Principles

### The Mau Performance Model

Mau's performance characteristics differ from traditional client-server architectures:

- **Local-first**: Reads are instant, writes are local-then-sync
- **Asynchronous sync**: Network operations don't block user actions
- **Distributed load**: No single bottleneck server
- **Cryptography overhead**: Every operation involves signing/verification

### Performance Goals

A well-optimized Mau application should achieve:

- **< 100ms** - Local file reads
- **< 500ms** - Write and sign operations
- **< 2s** - Initial peer discovery
- **< 5s** - First content sync with a known peer
- **< 100 ops/s** - Signature verification throughput

## Storage Optimization

### File System Best Practices

#### Use Efficient Directory Structures

Poor structure (linear):
```
posts/
  ├── post-1.json
  ├── post-2.json
  ├── ...
  └── post-10000.json  # Slow directory listing
```

Good structure (hierarchical):
```
posts/
  ├── 2024/
  │   ├── 01/
  │   │   ├── post-abc123.json
  │   │   └── post-def456.json
  │   └── 02/
  └── 2025/
```

**Implementation:**

```go
func GetPostPath(timestamp time.Time, id string) string {
    return fmt.Sprintf("posts/%d/%02d/%s.json",
        timestamp.Year(),
        timestamp.Month(),
        id)
}
```

#### Minimize File Descriptors

Keep open file handles under control:

```go
// Bad: Opens files without closing
for _, file := range files {
    f, _ := os.Open(file)
    processFile(f)
    // Missing f.Close()!
}

// Good: Ensures cleanup
for _, file := range files {
    func() {
        f, err := os.Open(file)
        if err != nil {
            return
        }
        defer f.Close()
        processFile(f)
    }()
}
```

### JSON-LD Optimization

#### Compact Representations

Use JSON-LD compaction to reduce file sizes:

```go
import "github.com/piprate/json-gold/ld"

func CompactPost(post map[string]interface{}) (map[string]interface{}, error) {
    proc := ld.NewJsonLdProcessor()
    options := ld.NewJsonLdOptions("")
    
    // Use compact context
    context := map[string]interface{}{
        "@context": "https://schema.org",
    }
    
    return proc.Compact(post, context, options)
}
```

#### Lazy Loading

Don't load entire objects when you only need metadata:

```go
type PostMetadata struct {
    Type          string    `json:"@type"`
    Headline      string    `json:"headline"`
    DatePublished time.Time `json:"datePublished"`
    // Skip large content fields
}

func LoadMetadata(path string) (*PostMetadata, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    
    var meta PostMetadata
    if err := json.Unmarshal(data, &meta); err != nil {
        return nil, err
    }
    return &meta, nil
}
```

### Indexing

Build indexes for fast lookups without scanning all files:

```go
type Index struct {
    ByDate    map[string][]string // date -> []postIDs
    ByAuthor  map[string][]string // author -> []postIDs
    ByType    map[string][]string // type -> []postIDs
}

func BuildIndex(dataDir string) (*Index, error) {
    index := &Index{
        ByDate:   make(map[string][]string),
        ByAuthor: make(map[string][]string),
        ByType:   make(map[string][]string),
    }
    
    err := filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
        if err != nil || info.IsDir() || !strings.HasSuffix(path, ".json") {
            return err
        }
        
        meta, err := LoadMetadata(path)
        if err != nil {
            return nil // Skip invalid files
        }
        
        id := filepath.Base(path)
        dateKey := meta.DatePublished.Format("2006-01-02")
        
        index.ByDate[dateKey] = append(index.ByDate[dateKey], id)
        index.ByType[meta.Type] = append(index.ByType[meta.Type], id)
        
        return nil
    })
    
    return index, err
}
```

Save indexes to disk for fast startup:

```go
func SaveIndex(index *Index, path string) error {
    data, err := json.Marshal(index)
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, 0644)
}
```

## Cryptography Performance

### PGP Operations

Cryptographic operations are CPU-intensive. Optimize them carefully:

#### Batch Verification

Verify multiple signatures in parallel:

```go
func VerifyBatch(messages []SignedMessage) []error {
    results := make([]error, len(messages))
    var wg sync.WaitGroup
    
    // Use worker pool to limit concurrency
    workers := runtime.NumCPU()
    jobs := make(chan int, len(messages))
    
    for w := 0; w < workers; w++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for i := range jobs {
                results[i] = messages[i].Verify()
            }
        }()
    }
    
    for i := range messages {
        jobs <- i
    }
    close(jobs)
    wg.Wait()
    
    return results
}
```

#### Cache Verification Results

Don't verify the same signature twice:

```go
type VerificationCache struct {
    cache map[string]bool // signature hash -> valid
    mu    sync.RWMutex
}

func (vc *VerificationCache) Verify(sig string, verifyFn func() error) error {
    hash := sha256.Sum256([]byte(sig))
    hashStr := hex.EncodeToString(hash[:])
    
    vc.mu.RLock()
    if valid, ok := vc.cache[hashStr]; ok {
        vc.mu.RUnlock()
        if valid {
            return nil
        }
        return errors.New("cached: invalid signature")
    }
    vc.mu.RUnlock()
    
    err := verifyFn()
    
    vc.mu.Lock()
    vc.cache[hashStr] = (err == nil)
    vc.mu.Unlock()
    
    return err
}
```

#### Key Management

Cache parsed keys in memory:

```go
type KeyCache struct {
    keys map[string]*openpgp.Entity
    mu   sync.RWMutex
}

func (kc *KeyCache) GetKey(fingerprint string) (*openpgp.Entity, error) {
    kc.mu.RLock()
    key, ok := kc.keys[fingerprint]
    kc.mu.RUnlock()
    
    if ok {
        return key, nil
    }
    
    // Load from disk
    key, err := LoadKeyFromDisk(fingerprint)
    if err != nil {
        return nil, err
    }
    
    kc.mu.Lock()
    kc.keys[fingerprint] = key
    kc.mu.Unlock()
    
    return key, nil
}
```

### Encryption Strategies

#### Encrypt Large Files in Chunks

Don't load entire files into memory:

```go
func EncryptStream(src io.Reader, dst io.Writer, key *openpgp.Entity) error {
    // Create encrypted writer
    hints := &openpgp.FileHints{
        IsBinary: true,
    }
    
    w, err := openpgp.Encrypt(dst, []*openpgp.Entity{key}, nil, hints, nil)
    if err != nil {
        return err
    }
    
    // Stream copy (chunked)
    _, err = io.Copy(w, src)
    if err != nil {
        w.Close()
        return err
    }
    
    return w.Close()
}
```

## Network Optimization

### HTTP Performance

#### Connection Pooling

Reuse HTTP connections to peers:

```go
var client = &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
}

func FetchFromPeer(peerURL string) (*http.Response, error) {
    return client.Get(peerURL)
}
```

#### Request Batching

Combine multiple small requests:

```go
// Bad: Multiple round-trips
for _, postID := range postIDs {
    FetchPost(peer, postID)
}

// Good: Single batch request
FetchPostsBatch(peer, postIDs)
```

**Implementation:**

```go
func FetchPostsBatch(peer string, ids []string) ([]*Post, error) {
    idsParam := strings.Join(ids, ",")
    resp, err := client.Get(peer + "/posts?ids=" + idsParam)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var posts []*Post
    if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
        return nil, err
    }
    return posts, nil
}
```

#### Compression

Enable gzip compression for HTTP transfers:

```go
func ServeWithCompression(w http.ResponseWriter, r *http.Request, data []byte) {
    if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
        w.Write(data)
        return
    }
    
    w.Header().Set("Content-Encoding", "gzip")
    gz := gzip.NewWriter(w)
    defer gz.Close()
    gz.Write(data)
}
```

### Rate Limiting

Protect your node from being overwhelmed:

```go
import "golang.org/x/time/rate"

type RateLimiter struct {
    limiter *rate.Limiter
}

func NewRateLimiter(requestsPerSecond int) *RateLimiter {
    return &RateLimiter{
        limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), requestsPerSecond*2),
    }
}

func (rl *RateLimiter) Allow() bool {
    return rl.limiter.Allow()
}

// In HTTP handler
func HandleRequest(w http.ResponseWriter, r *http.Request) {
    if !rateLimiter.Allow() {
        http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
        return
    }
    // Handle request
}
```

## DHT and Discovery

### Kademlia Optimization

#### Routing Table Maintenance

Keep routing table fresh without excessive pings:

```go
type RoutingTable struct {
    buckets []*Bucket
    lastRefresh map[int]time.Time
}

func (rt *RoutingTable) RefreshIfNeeded() {
    now := time.Now()
    for i, bucket := range rt.buckets {
        if len(bucket.Peers) > 0 && now.Sub(rt.lastRefresh[i]) > 15*time.Minute {
            go rt.refreshBucket(i)
            rt.lastRefresh[i] = now
        }
    }
}
```

#### Parallel Lookups

Query multiple peers simultaneously:

```go
func FindNode(target string, k int) []*Peer {
    candidates := getClosestPeers(target, k*3)
    results := make(chan []*Peer, len(candidates))
    
    for _, peer := range candidates {
        go func(p *Peer) {
            peers, err := p.FindNode(target)
            if err == nil {
                results <- peers
            } else {
                results <- nil
            }
        }(peer)
    }
    
    // Collect results
    var allPeers []*Peer
    for i := 0; i < len(candidates); i++ {
        if peers := <-results; peers != nil {
            allPeers = append(allPeers, peers...)
        }
    }
    
    return getClosestPeers(target, k, allPeers)
}
```

### mDNS Efficiency

Don't spam the local network:

```go
type MDNSDiscovery struct {
    lastAnnounce time.Time
    announceInterval time.Duration
}

func (md *MDNSDiscovery) AnnounceIfNeeded() {
    if time.Since(md.lastAnnounce) < md.announceInterval {
        return
    }
    
    md.Announce()
    md.lastAnnounce = time.Now()
}
```

## Caching Strategies

### Multi-Level Cache

Combine memory and disk caching:

```go
type Cache struct {
    memory *lru.Cache      // Fast, limited size
    disk   string          // Slower, larger capacity
}

func (c *Cache) Get(key string) ([]byte, error) {
    // Try memory first
    if val, ok := c.memory.Get(key); ok {
        return val.([]byte), nil
    }
    
    // Try disk
    diskPath := filepath.Join(c.disk, key)
    data, err := os.ReadFile(diskPath)
    if err != nil {
        return nil, err
    }
    
    // Promote to memory
    c.memory.Add(key, data)
    return data, nil
}

func (c *Cache) Set(key string, data []byte) error {
    // Write to both
    c.memory.Add(key, data)
    diskPath := filepath.Join(c.disk, key)
    return os.WriteFile(diskPath, data, 0644)
}
```

### Time-Based Invalidation

```go
type CachedItem struct {
    Data      []byte
    Timestamp time.Time
    TTL       time.Duration
}

func (ci *CachedItem) IsValid() bool {
    return time.Since(ci.Timestamp) < ci.TTL
}
```

## Monitoring and Profiling

### Built-in Metrics

Track key performance indicators:

```go
type Metrics struct {
    RequestCount   uint64
    ErrorCount     uint64
    AvgLatency     time.Duration
    PeerCount      int
    SyncedFiles    uint64
}

var metrics Metrics

func RecordRequest(duration time.Duration, err error) {
    atomic.AddUint64(&metrics.RequestCount, 1)
    if err != nil {
        atomic.AddUint64(&metrics.ErrorCount, 1)
    }
    
    // Update average (simplified)
    metrics.AvgLatency = (metrics.AvgLatency + duration) / 2
}

func GetMetrics() Metrics {
    return Metrics{
        RequestCount: atomic.LoadUint64(&metrics.RequestCount),
        ErrorCount:   atomic.LoadUint64(&metrics.ErrorCount),
        AvgLatency:   metrics.AvgLatency,
        PeerCount:    getPeerCount(),
        SyncedFiles:  atomic.LoadUint64(&metrics.SyncedFiles),
    }
}
```

### Profiling

Use Go's built-in profiling tools:

```go
import _ "net/http/pprof"

func StartProfiler() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
}
```

Access profiles:
```bash
# CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile

# Memory profile
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutines
curl http://localhost:6060/debug/pprof/goroutine?debug=1
```

### Logging Performance

Avoid excessive logging in hot paths:

```go
// Bad: Logs every operation
func ProcessItem(item string) {
    log.Printf("Processing item: %s", item)
    // ... processing
    log.Printf("Completed item: %s", item)
}

// Good: Batch logging
func ProcessBatch(items []string) {
    log.Printf("Processing batch of %d items", len(items))
    for _, item := range items {
        // ... processing
    }
    log.Printf("Completed batch")
}
```

## Scalability Patterns

### Sharding Data

Distribute data across multiple directories:

```go
func GetShardPath(key string) string {
    hash := sha256.Sum256([]byte(key))
    shard := hash[0] % 16 // 16 shards
    return fmt.Sprintf("data/shard-%02x/%s", shard, key)
}
```

### Background Sync

Don't block user actions waiting for sync:

```go
type SyncQueue struct {
    pending chan string
}

func (sq *SyncQueue) Enqueue(path string) {
    select {
    case sq.pending <- path:
        // Queued
    default:
        // Queue full, log warning
        log.Printf("Sync queue full, dropping: %s", path)
    }
}

func (sq *SyncQueue) Worker() {
    for path := range sq.pending {
        if err := syncFile(path); err != nil {
            log.Printf("Sync failed for %s: %v", path, err)
            // Retry logic here
        }
    }
}
```

### Graceful Degradation

Handle peer unavailability gracefully:

```go
func FetchWithFallback(primaryPeer, fallbackPeer, path string) ([]byte, error) {
    data, err := fetchFromPeer(primaryPeer, path)
    if err == nil {
        return data, nil
    }
    
    log.Printf("Primary peer failed, trying fallback: %v", err)
    return fetchFromPeer(fallbackPeer, path)
}
```

## Performance Checklist

Before deploying your Mau application, verify:

**Storage:**
- [ ] Directory structure is hierarchical (< 1000 files per directory)
- [ ] Indexes are built for common queries
- [ ] Large files use streaming I/O
- [ ] File descriptors are properly closed

**Cryptography:**
- [ ] Signature verification is cached
- [ ] Keys are cached in memory
- [ ] Batch operations use parallelism
- [ ] Encryption uses streaming for large files

**Network:**
- [ ] HTTP connections are pooled
- [ ] Requests are batched where possible
- [ ] Compression is enabled
- [ ] Rate limiting protects your node

**DHT:**
- [ ] Routing table refresh is scheduled, not reactive
- [ ] Lookups query multiple peers in parallel
- [ ] mDNS announces are throttled

**Monitoring:**
- [ ] Key metrics are tracked
- [ ] Profiling endpoint is available (non-production)
- [ ] Logs are appropriate (not excessive)

## Benchmarking

Example benchmark for your application:

```go
func BenchmarkSignAndSave(b *testing.B) {
    client, _ := mau.NewClient("/tmp/bench")
    post := map[string]interface{}{
        "@type": "SocialMediaPosting",
        "headline": "Benchmark post",
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        client.SavePost(fmt.Sprintf("post-%d.json", i), post)
    }
}

func BenchmarkVerifyBatch(b *testing.B) {
    messages := generateSignedMessages(1000)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        VerifyBatch(messages)
    }
}
```

Run benchmarks:
```bash
go test -bench=. -benchmem
```

## Further Resources

- [Go Performance Tips](https://github.com/dgryski/go-perfbook)
- [HTTP/2 Best Practices](https://http2.github.io/)
- [PGP Performance](https://www.gnupg.org/documentation/manuals/gnupg/GPG-Esoteric-Options.html)
- [Kademlia Paper](https://pdos.csail.mit.edu/~petar/papers/maymounkov-kademlia-lncs.pdf)

---

**Next:** [API Documentation](11-api-reference.md)  
**Previous:** [Privacy & Security](09-privacy-security.md)
