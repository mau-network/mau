# Performance & Optimization

Strategies for scaling Mau applications and improving efficiency in production environments.

---

## Table of Contents

1. [Understanding Performance Bottlenecks](#understanding-performance-bottlenecks)
2. [DHT Optimization](#dht-optimization)
3. [Storage Optimization](#storage-optimization)
4. [Network Optimization](#network-optimization)
5. [Memory Management](#memory-management)
6. [Profiling Tools](#profiling-tools)
7. [Production Deployment](#production-deployment)
8. [Monitoring & Metrics](#monitoring--metrics)

---

## Understanding Performance Bottlenecks

### Common Bottlenecks in P2P Systems

**Network I/O:**
- DHT lookups across multiple peers
- Peer discovery latency
- NAT traversal overhead

**Storage I/O:**
- Disk seeks for object retrieval
- Index updates on writes
- Large object serialization

**CPU:**
- Cryptographic operations (signature verification, encryption)
- JSON-LD parsing and validation
- Compression/decompression

**Memory:**
- Large object caching
- DHT routing table size
- Connection pool management

### Performance Goals

| Metric | Target | Measurement |
|--------|--------|-------------|
| Object retrieval | < 100ms (cached) | Time to GET local object |
| Peer discovery | < 500ms | DHT lookup completion |
| Object publish | < 200ms | POST with signature |
| Storage scan | < 1s per 10k objects | Full index rebuild |

---

## DHT Optimization

### Routing Table Management

**Bucket Size Tuning:**
```go
// Adjust k-value for routing table density
// Higher = more peers cached, better redundancy, more memory
const K_VALUE = 20 // Default

// For high-throughput nodes:
const K_VALUE = 40

// For resource-constrained devices:
const K_VALUE = 10
```

**Refresh Strategy:**
```go
// Stale bucket refresh interval
const STALL_PERIOD = 15 * time.Minute // Default

// Aggressive refresh for dynamic networks:
const STALL_PERIOD = 5 * time.Minute

// Conservative refresh for stable networks:
const STALL_PERIOD = 1 * time.Hour
```

### Lookup Optimization

**Parallel Lookups:**
- Default: 3 concurrent queries per hop
- High-latency networks: Increase to 5-8
- Low-bandwidth: Reduce to 1-2

**Lookup Timeout:**
```go
// Per-peer timeout
const LOOKUP_TIMEOUT = 5 * time.Second // Default

// Adjust for network conditions:
// Fast local network: 1s
// Internet with NAT: 5-10s
// High-latency (satellite, Tor): 30s
```

### Bootstrap Node Strategy

**Multiple Bootstrap Nodes:**
```bash
# Configure 3+ bootstrap nodes for redundancy
mau bootstrap add peer1.example.com:8080
mau bootstrap add peer2.example.com:8080
mau bootstrap add peer3.example.com:8080
```

**Geographic Distribution:**
- Deploy bootstrap nodes across regions
- Reduces initial DHT join latency
- Improves global peer discovery

---

## Storage Optimization

### Object Storage Strategy

**Index Design:**
```
storage/
├── objects/
│   └── [fingerprint]/
│       └── [object-id].json  # Actual objects
├── index.json                # Fast lookup index
└── cache/
    └── rendered/             # Pre-rendered views
```

**Index Optimization:**
- Keep index in memory (mmap or cache)
- Lazy load objects on access
- Rebuild index only on corruption

### Large Object Handling

**Chunking Strategy:**
```javascript
// Split large objects (> 1MB) into chunks
{
  "@context": "https://schema.org",
  "@type": "VideoObject",
  "contentUrl": "mau://chunks/video-123",
  "chunks": [
    "mau://obj/chunk-1",
    "mau://obj/chunk-2",
    // ... up to 100 chunks
  ]
}
```

**Streaming:**
- Use HTTP range requests for partial retrieval
- Implement progressive loading for media
- Cache frequently accessed chunks

### Compression

**JSON Compression:**
```bash
# Enable gzip compression for storage
export MAU_STORAGE_COMPRESS=true

# Typical savings: 60-80% for text content
```

**Selective Compression:**
- Compress: Text objects, JSON-LD, large strings
- Don't compress: Already compressed media (JPEG, MP4)

---

## Network Optimization

### Connection Pooling

**Persistent Connections:**
```go
// Reuse HTTP client connections
var httpClient = &http.Client{
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
    Timeout: 30 * time.Second,
}
```

### NAT Traversal

**UPnP Configuration:**
```bash
# Enable automatic port mapping
mau config set upnp.enabled true
mau config set upnp.refresh 15m

# Check current mappings
mau net upnp status
```

**Manual Port Forwarding:**
- Forward external port to internal Mau port
- Reduces connection establishment latency
- Improves peer reachability

### Request Batching

**Batch DHT Lookups:**
```go
// Instead of:
for _, key := range keys {
    peer := dht.Find(key) // N network calls
}

// Use:
peers := dht.FindMultiple(keys) // 1 network call
```

---

## Memory Management

### Object Caching

**LRU Cache Implementation:**
```go
// Cache frequently accessed objects
cache := NewLRUCache(1000) // Keep 1000 objects

// Adjust size based on available RAM:
// Small device (512MB): 100-500 objects
// Desktop (8GB): 5000-10000 objects
// Server (32GB+): 50000+ objects
```

**Cache Eviction Strategy:**
- LRU (Least Recently Used) for general objects
- LFU (Least Frequently Used) for popular content
- TTL-based for time-sensitive data

### Garbage Collection

**Go GC Tuning:**
```bash
# Reduce GC pressure for high-throughput systems
export GOGC=200 # Default: 100

# For low-memory devices:
export GOGC=50
```

---

## Profiling Tools

### CPU Profiling

**Enable Profiling:**
```go
import _ "net/http/pprof"

func main() {
    go func() {
        http.ListenAndServe("localhost:6060", nil)
    }()
    // ... rest of application
}
```

**Capture Profile:**
```bash
# Run for 30 seconds
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Analyze results
(pprof) top10
(pprof) web  # Generate flame graph
```

### Memory Profiling

**Heap Analysis:**
```bash
# Capture heap snapshot
curl http://localhost:6060/debug/pprof/heap > heap.prof

# Analyze
go tool pprof heap.prof
(pprof) top10
(pprof) list functionName
```

### Trace Analysis

**Record Execution Trace:**
```bash
# Capture 5-second trace
curl http://localhost:6060/debug/pprof/trace?seconds=5 > trace.out

# Visualize
go tool trace trace.out
```

---

## Production Deployment

### Recommended Hardware

**Minimum Specs:**
- CPU: 1 core @ 1.5GHz
- RAM: 512MB
- Storage: 10GB SSD
- Network: 1 Mbps up/down

**Recommended Specs:**
- CPU: 2+ cores @ 2.5GHz
- RAM: 2GB
- Storage: 50GB+ SSD
- Network: 10+ Mbps up/down

**High-Performance:**
- CPU: 4+ cores @ 3.0GHz
- RAM: 8GB+
- Storage: 500GB+ NVMe SSD
- Network: 100+ Mbps up/down

### Systemd Service

**Example Unit File:**
```ini
[Unit]
Description=Mau P2P Node
After=network.target

[Service]
Type=simple
User=mau
ExecStart=/usr/local/bin/mau serve --port 8080
Restart=on-failure
RestartSec=10s

# Resource limits
LimitNOFILE=65536
MemoryLimit=2G
CPUQuota=200%

# Environment
Environment="GOGC=200"
Environment="MAU_STORAGE_COMPRESS=true"

[Install]
WantedBy=multi-user.target
```

### Reverse Proxy Setup

**Nginx Configuration:**
```nginx
upstream mau_backend {
    server 127.0.0.1:8080;
    keepalive 32;
}

server {
    listen 443 ssl http2;
    server_name mau.example.com;

    ssl_certificate /etc/letsencrypt/live/mau.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/mau.example.com/privkey.pem;

    location / {
        proxy_pass http://mau_backend;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        
        # Timeouts for large object uploads
        proxy_connect_timeout 60s;
        proxy_send_timeout 300s;
        proxy_read_timeout 300s;
    }
}
```

---

## Monitoring & Metrics

### Key Metrics to Track

**System Metrics:**
- CPU usage (target: < 70% average)
- Memory usage (target: < 80% of available)
- Disk I/O (read/write latency)
- Network bandwidth (utilization)

**Application Metrics:**
- HTTP request rate (requests/sec)
- DHT lookup latency (p50, p95, p99)
- Object storage operations (read/write rate)
- Peer count (active connections)

**Error Metrics:**
- HTTP error rate (4xx, 5xx responses)
- DHT lookup failures
- Storage corruption events
- Signature verification failures

### Prometheus Integration

**Expose Metrics:**
```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    dhtLookups = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "mau_dht_lookups_total",
            Help: "Total DHT lookups performed",
        },
    )
    objectReads = prometheus.NewHistogram(
        prometheus.HistogramOpts{
            Name: "mau_object_read_duration_seconds",
            Help: "Object read latency",
        },
    )
)

func main() {
    prometheus.MustRegister(dhtLookups, objectReads)
    http.Handle("/metrics", promhttp.Handler())
}
```

### Alerting Rules

**Example Prometheus Alerts:**
```yaml
groups:
- name: mau
  rules:
  - alert: HighErrorRate
    expr: rate(mau_http_errors_total[5m]) > 0.1
    for: 5m
    annotations:
      summary: "High error rate detected"
      
  - alert: SlowDHTLookups
    expr: histogram_quantile(0.95, mau_dht_lookup_duration_seconds) > 1
    for: 10m
    annotations:
      summary: "95th percentile DHT lookups > 1s"
```

---

## Performance Checklist

**Before Production:**

- [ ] Profile CPU and memory under load
- [ ] Test with realistic object sizes and counts
- [ ] Verify network throughput with multiple peers
- [ ] Enable compression for storage
- [ ] Configure appropriate cache sizes
- [ ] Set up monitoring and alerting
- [ ] Test NAT traversal from external networks
- [ ] Document expected performance characteristics
- [ ] Run load tests with concurrent operations
- [ ] Verify graceful degradation under stress

**Optimization Quick Wins:**

1. Enable storage compression (60-80% space savings)
2. Increase DHT k-value for high-throughput nodes
3. Add multiple bootstrap nodes
4. Use persistent HTTP connections
5. Implement object caching with LRU
6. Enable UPnP for automatic port mapping
7. Configure systemd resource limits
8. Use reverse proxy with connection pooling
9. Monitor and tune GOGC for your workload
10. Profile and eliminate hot paths

---

## Next Steps

- **[API Documentation](11-api-reference.md)** - Complete Go package reference
- **[Schema Types](12-schema-types.md)** - Comprehensive Schema.org types guide
- **[Troubleshooting](13-troubleshooting.md)** - Debugging common issues
- **[HTTP API](07-http-api.md)** - REST API reference

---

**Need Help?**

- GitHub Issues: https://github.com/mau-network/mau/issues
- Documentation: https://mau.social/docs
