/**
 * Kademlia DHT over WebRTC with DHT-relay signaling.
 *
 * Same algorithm as the Go kademlia.go (160 buckets, k=20, alpha=3, XOR
 * distance on PGP fingerprints) but every node-to-node message travels over
 * a WebRTC data channel instead of HTTP.
 *
 * Bootstrap: first connection to a known peer uses HTTP (POST /p2p/dht/offer).
 * All subsequent connections use DHT-relay signaling: an existing peer
 * forwards offer/answer/ICE so both sides punch through NAT simultaneously.
 *
 *   A --relay_offer--> C --relay_offer--> B
 *   A <-relay_answer-- C <-relay_answer-- B
 *   A <-> relay_ice <-> C <-> relay_ice <-> B
 *   A <====== direct WebRTC data channel ======> B
 */

import type { Account } from '../account.js';
import { normalizeFingerprint } from '../crypto/index.js';
import type { Fingerprint, FingerprintResolver, Peer } from '../types/index.js';
import { DHT_B, DHT_K, DHT_ALPHA, DHT_STALL_PERIOD_MS } from '../types/index.js';

// ── Wire messages ─────────────────────────────────────────────────────────────

type DHTMsg =
  | { type: 'ping'; id: string }
  | { type: 'pong'; id: string }
  | { type: 'find_peer'; id: string; target: Fingerprint }
  | { type: 'find_peer_response'; id: string; peers: Peer[] }
  | { type: 'relay_offer';  from: Fingerprint; to: Fingerprint; offer: RTCSessionDescriptionInit }
  | { type: 'relay_answer'; from: Fingerprint; to: Fingerprint; answer: RTCSessionDescriptionInit }
  | { type: 'relay_ice';    from: Fingerprint; to: Fingerprint; candidate: RTCIceCandidateInit };

// ── Internal ──────────────────────────────────────────────────────────────────

interface Conn  { pc: RTCPeerConnection; ch: RTCDataChannel; lastSeen: number }
interface Bucket { peers: Peer[]; lastLookup: number }
interface Pend  { resolve(v: unknown): void; reject(e: Error): void; timer: ReturnType<typeof setTimeout> }
interface RelayOut { pc: RTCPeerConnection; resolve(): void; reject(e: Error): void }
interface RelayIn  { pc: RTCPeerConnection; queued: RTCIceCandidateInit[] }
interface PeerState {
  discovered: boolean;
  discoveredAt: number;
  connectionAttempts: number;
  lastAttempt: number;
}

// ── XOR helpers ───────────────────────────────────────────────────────────────

function hex20(hex: string): Uint8Array {
  const h = hex.padEnd(40, '0').slice(0, 40);
  const b = new Uint8Array(20);
  for (let i = 0; i < 20; i++) {b[i] = parseInt(h.slice(i * 2, i * 2 + 2), 16);}
  return b;
}
function xor(a: string, b: string): Uint8Array {
  const ab = hex20(a), bb = hex20(b), out = new Uint8Array(20);
  for (let i = 0; i < 20; i++) {out[i] = ab[i] ^ bb[i];}
  return out;
}
function leadingZeros(d: Uint8Array): number {
  for (let i = 0; i < d.length; i++) {
    if (d[i] === 0) {continue;}
    let b = d[i], n = 0;
    while ((b & 0x80) === 0) { n++; b <<= 1; }
    return i * 8 + n;
  }
  return d.length * 8;
}
function cmp(a: Uint8Array, b: Uint8Array): number {
  for (let i = 0; i < a.length; i++) {if (a[i] !== b[i]) {return a[i] - b[i];}}
  return 0;
}

function rid(): string { return Math.random().toString(36).slice(2); }

function waitICE(pc: RTCPeerConnection): Promise<void> {
  return new Promise((res): void => {
    if (pc.iceGatheringState === 'complete') { res(); return; }
    pc.onicegatheringstatechange = (): void => { if (pc.iceGatheringState === 'complete') {res();} };
    setTimeout(res, 10_000);
  });
}

// ── KademliaDHT ───────────────────────────────────────────────────────────────

export class KademliaDHT {
  private readonly account: Account;
  private readonly ice: RTCIceServer[];
  private readonly conns   = new Map<Fingerprint, Conn>();
  private readonly buckets: Bucket[] = Array.from({ length: DHT_B }, () => ({ peers: [], lastLookup: 0 }));
  private readonly pending = new Map<string, Pend>();
  private readonly rout    = new Map<Fingerprint, RelayOut>();
  private readonly rin     = new Map<Fingerprint, RelayIn>();
  private readonly connecting = new Set<Fingerprint>(); // Track in-progress connection attempts
  private readonly connectionAttempts = new Map<Fingerprint, number>(); // Track retry count per peer
  private readonly maxConnectionAttempts = 5; // Stop retrying after 5 failed attempts
  private readonly localICECandidates: RTCIceCandidate[] = []; // Pre-gathered ICE candidates
  private iceGatheringComplete = false;
  private joinTime = 0; // Track when DHT joined for uptime calculation
  
  // Peer state tracking for discovery/connection decoupling
  private readonly peerState = new Map<Fingerprint, PeerState>();
  
  private timer: ReturnType<typeof setInterval> | null = null;
  private bootstrapTimer: ReturnType<typeof setInterval> | null = null;
  private bootstrapActive = false;

  constructor(account: Account, iceServers: RTCIceServer[] = [{ urls: 'stun:stun.l.google.com:19302' }]) {
    this.account = account;
    this.ice = iceServers;
  }

  // ── Public ──────────────────────────────────────────────────────────────────

  /**
   * Gather local ICE candidates once at startup for reuse across all connections.
   * This avoids redundant STUN server queries and speeds up network formation.
   */
  private async gatherICECandidates(): Promise<void> {
    console.log('[DHT] Gathering local ICE candidates...');
    
    const pc = new RTCPeerConnection({ iceServers: this.ice });
    const ch = pc.createDataChannel('ice-gather');
    
    try {
      await pc.setLocalDescription(await pc.createOffer());
      
      await new Promise<void>((resolve) => {
        pc.onicecandidate = (ev): void => {
          if (ev.candidate) {
            this.localICECandidates.push(ev.candidate);
            console.log(`[DHT] ICE candidate gathered: ${ev.candidate.type || 'unknown'}`);
          } else {
            // Gathering complete (null candidate signals end)
            this.iceGatheringComplete = true;
            console.log(`[DHT] ICE gathering complete: ${this.localICECandidates.length} candidates`);
            resolve();
          }
        };
        
        // Timeout after 10 seconds if gathering doesn't complete
        setTimeout(() => {
          if (!this.iceGatheringComplete) {
            console.log(`[DHT] ICE gathering timeout, using ${this.localICECandidates.length} candidates`);
            this.iceGatheringComplete = true;
            resolve();
          }
        }, 10_000);
      });
    } finally {
      ch.close();
      pc.close();
    }
  }

  // ── Peer State Management ────────────────────────────────────────────────────

  private getPeerState(fingerprint: Fingerprint): PeerState {
    if (!this.peerState.has(fingerprint)) {
      this.peerState.set(fingerprint, {
        discovered: false,
        discoveredAt: 0,
        connectionAttempts: 0,
        lastAttempt: 0,
      });
    }
    return this.peerState.get(fingerprint)!;
  }

  private setPeerState(fingerprint: Fingerprint, state: PeerState): void {
    this.peerState.set(fingerprint, state);
  }

  private getConnectionBackoff(retryNumber: number): number {
    // Exponential backoff for retry N (0-indexed):
    // Retry 0 (first retry): 3s, Retry 1: 6s, Retry 2: 12s, etc.
    // Cap at 60s
    const backoff = Math.min(3000 * Math.pow(2, Math.max(0, retryNumber)), 60000);
    return backoff;
  }

  // ── Discovery/Connection Decoupling ───────────────────────────────────────────

  /**
   * Discover new peers by querying the routing table.
   * Only runs if we have fewer than DHT_K peers in the routing table.
   */
  private async discoverPeers(): Promise<void> {
    // Check routing table size
    const totalPeers = this.buckets.reduce((sum, bucket) => sum + bucket.peers.length, 0);
    
    // Skip discovery if we already have enough peers
    if (totalPeers >= DHT_K) {
      return;
    }

    // If routing table is empty, query bootstrap for self to get seed peers
    if (totalPeers === 0) {
      await this.lookup(this.me());
      return;
    }

    // Otherwise, refresh stale buckets by looking up random targets in their ranges
    const now = Date.now();
    for (let i = 0; i < DHT_B; i++) {
      const bucket = this.buckets[i];
      
      // Skip empty buckets or recently refreshed ones
      if (bucket.peers.length === 0 || now - bucket.lastLookup < DHT_STALL_PERIOD_MS) {
        continue;
      }
      
      // Generate a random target fingerprint in this bucket's range
      // Bucket i contains peers with leadingZeros(XOR(peer, me)) == i
      const target = this.randomFingerprintForBucket(i);
      await this.lookup(target);
      
      // Only refresh one bucket per call to avoid overwhelming the network
      break;
    }
  }

  /**
   * Generate a random fingerprint that would fall into the specified bucket.
   * Bucket i contains peers where leadingZeros(XOR(peer, me)) == i.
   */
  private randomFingerprintForBucket(bucketIndex: number): Fingerprint {
    const myBytes = hex20(this.me());
    const targetBytes = new Uint8Array(20);
    
    // Generate random bytes
    crypto.getRandomValues(targetBytes);
    
    // Set the first (bucketIndex) bits to match our fingerprint
    // Then set bit (bucketIndex) to differ from our fingerprint
    const bytePos = Math.floor(bucketIndex / 8);
    const bitPos = bucketIndex % 8;
    
    // Copy matching prefix
    for (let i = 0; i < bytePos; i++) {
      targetBytes[i] = myBytes[i];
    }
    
    // Set the divergence bit at bucketIndex position
    if (bytePos < 20) {
      const mask = 0x80 >> bitPos;
      
      // Ensure bits 0..bitPos-1 of this byte match myBytes (the prefix)
      // and bit bitPos differs (the divergence point)
      const prefixMask = ~((mask << 1) - 1) & 0xFF;  // bits 0..bitPos-1 = 1
      targetBytes[bytePos] = (myBytes[bytePos] & prefixMask)           // copy prefix bits
                           | ((myBytes[bytePos] ^ mask) & mask)         // flip divergence bit
                           | (targetBytes[bytePos] & ~prefixMask & ~mask);  // keep random low bits
    }
    
    // Convert to hex fingerprint
    return Array.from(targetBytes).map(b => b.toString(16).padStart(2, '0')).join('').toUpperCase();
  }

  /**
   * Attempt to connect to peers we've discovered but haven't connected yet.
   * Respects cooldown periods and max attempts.
   */
  private async connectKnownPeers(): Promise<void> {
    const now = Date.now();
    const peersToConnect: Peer[] = [];

    // Scan routing table for peers to connect to
    for (const bucket of this.buckets) {
      for (const peer of bucket.peers) {
        // Skip self — registerSelf() places our own fingerprint in the routing table so it
        // appears in find_peer responses, but we must never attempt to connect to ourselves.
        if (normalizeFingerprint(peer.fingerprint) === this.me()) {
          continue;
        }

        // Skip if already connected
        if (this.conns.has(peer.fingerprint)) {
          continue;
        }

        // Skip if currently connecting
        if (this.connecting.has(peer.fingerprint)) {
          continue;
        }

        const state = this.getPeerState(peer.fingerprint);

        // Stop trying after max attempts
        if (state.connectionAttempts >= this.maxConnectionAttempts) {
          continue;
        }

        // Check cooldown period
        if (state.lastAttempt > 0) {
          // Calculate backoff for NEXT attempt (0-indexed, so subtract 1)
          const backoff = this.getConnectionBackoff(state.connectionAttempts - 1);
          const timeSinceLastAttempt = now - state.lastAttempt;
          if (timeSinceLastAttempt < backoff) {
            continue; // Still in cooldown
          }
        }

        peersToConnect.push(peer);
      }
    }

    // Attempt connections
    await Promise.allSettled(
      peersToConnect.map(peer => this.connectDiscoveredPeers([peer]))
    );
  }

  // ── Join & Bootstrap ──────────────────────────────────────────────────────────

  async join(bootstrapPeers: Peer[]): Promise<void> {
    // Gather ICE candidates once before any connections
    await this.gatherICECandidates();
    
    this.joinTime = Date.now();
    
    await Promise.allSettled(bootstrapPeers.map(p => {
      // Use WebSocket signaling if address starts with 'ws://' or 'wss://'
      if (p.address.startsWith('ws://') || p.address.startsWith('wss://')) {
        return this.connectWebSocket(p).catch(e => console.warn(`[DHT] WS bootstrap ${p.fingerprint}:`, e));
      } else {
        return this.connectHTTP(p).catch(e => console.warn(`[DHT] HTTP bootstrap ${p.fingerprint}:`, e));
      }
    }));
    await this.lookup(this.me());
    
    // Start periodic refresh (every 60 seconds)
    this.timer = setInterval(() => this.refresh().catch(() => {}), 60_000);
    
    // Start aggressive bootstrap discovery (every 3 seconds when we have few peers)
    // This helps when bootstrap peer initially has no other peers to share
    console.log('[DHT] Starting bootstrap discovery timer (every 3s)');
    this.bootstrapActive = true;
    this.bootstrapTimer = setInterval(() => {
      // Check if bootstrap is still active
      if (!this.bootstrapActive) {
        if (this.bootstrapTimer) {
          clearInterval(this.bootstrapTimer);
          this.bootstrapTimer = null;
        }
        return;
      }
      
      console.log('[DHT] Bootstrap discovery timer fired');
      this.bootstrapDiscovery().catch(err => console.warn('[DHT] Bootstrap discovery error:', err));
    }, 3_000);
  }

  stop(): void {
    // Stop bootstrap first
    this.bootstrapActive = false;
    if (this.bootstrapTimer) { clearInterval(this.bootstrapTimer); this.bootstrapTimer = null; }
    
    // Stop periodic refresh
    if (this.timer) { clearInterval(this.timer); this.timer = null; }
    
    // Close all connections
    for (const c of this.conns.values()) { c.ch.close(); c.pc.close(); }
    this.conns.clear();
  }

  resolver(): FingerprintResolver {
    return (fpr: Fingerprint) => this.findAddress(fpr);
  }

  /**
   * Get DHT statistics and health metrics
   */
  stats(): {
    connected: number;
    discovered: number;
    bucketFill: number[];
    uptime: number;
    bootstrapActive: boolean;
  } {
    const bucketFill = this.buckets.map(b => b.peers.length);
    const discovered = this.buckets.reduce((sum, b) => sum + b.peers.length, 0);
    const uptime = this.joinTime > 0 ? Date.now() - this.joinTime : 0;
    
    return {
      connected: this.conns.size,
      discovered,
      bucketFill,
      uptime,
      bootstrapActive: this.bootstrapActive,
    };
  }

  /**
   * Register the bootstrap node itself in the routing table.
   * 
   * This is used by bootstrap servers to ensure they appear in find_peer responses
   * even before any clients have connected. Without this, the first client gets
   * zero peers (the "0 peers" bug).
   * 
   * Note: Normally nodes don't add themselves to their routing table, but bootstrap
   * nodes are special - they need to be discoverable before any other peers connect.
   * 
   * @param address The address where this node can be reached (e.g., "ws://localhost:8444")
   */
  registerSelf(address: string): void {
    const myFingerprint = this.me();
    console.log(`[DHT] Registering self in routing table: ${myFingerprint.slice(0, 16)}... at ${address}`);
    
    // Create a peer entry for ourselves
    const selfPeer: Peer = { 
      fingerprint: myFingerprint, 
      address 
    };
    
    // Find the appropriate bucket (bucket 0 since XOR distance to self is 0, which has 160 leading zeros)
    // Actually, all bits match so it goes to the highest bucket (159)
    // But since we're never going to connect to ourselves, we can just put it in bucket 159
    const bucketIdx = DHT_B - 1;
    const bucket = this.buckets[bucketIdx];
    
    // Add to bucket if not already present
    const exists = bucket.peers.find(p => normalizeFingerprint(p.fingerprint) === myFingerprint);
    if (!exists) {
      bucket.peers.push(selfPeer);
      bucket.lastLookup = Date.now();
      console.log(`[DHT] Self registered in bucket ${bucketIdx} - bootstrap will now appear in find_peer responses`);
    }
    
    // Mark as discovered in peer state
    const state = this.getPeerState(myFingerprint);
    state.discovered = true;
    state.discoveredAt = Date.now();
    this.setPeerState(myFingerprint, state);
  }

  async findAddress(target: Fingerprint): Promise<string | null> {
    const fpr = normalizeFingerprint(target);
    const local = this.fromTable(fpr);
    if (local?.address) {return local.address;}
    const found = await this.lookup(fpr);
    return found?.address ?? null;
  }

  /**
   * Find nearest peers to a target fingerprint (max 160 as per Kademlia spec)
   * Used by /kad/find_peer HTTP endpoint
   */
  findPeer(target: Fingerprint): Peer[] {
    return this.nearest(target, 160);
  }

  /**
   * Register an already-established WebRTC connection with the DHT
   * This is used by bootstrap servers to register connections established via WebSocket signaling
   */
  registerConnection(fingerprint: Fingerprint, peerConnection: RTCPeerConnection, dataChannel: RTCDataChannel): void {
    const fpr = normalizeFingerprint(fingerprint);
    if (fpr === this.me()) {return;}
    if (this.conns.has(fpr)) {return;} // Already registered
    
    const conn: Conn = { pc: peerConnection, ch: dataChannel, lastSeen: Date.now() };
    this.conns.set(fpr, conn);
    
    // Set up message handler
    dataChannel.onmessage = (ev: MessageEvent<string>): void => this.onMsg(fpr, ev.data);
    dataChannel.onclose = (): void => { this.conns.delete(fpr); };
    
    // Add to routing table
    console.log(`[DHT] Registered connection from ${fpr.slice(0, 16)}... (total: ${this.conns.size})`);
    this.addPeer({ fingerprint, address: `webrtc://${fingerprint}` });
  }

  // ── Routing table ────────────────────────────────────────────────────────────

  private me(): Fingerprint { return normalizeFingerprint(this.account.getFingerprint()); }

  private idx(fpr: Fingerprint): number {
    return Math.min(leadingZeros(xor(this.me(), normalizeFingerprint(fpr))), DHT_B - 1);
  }

  private addPeer(peer: Peer): void {
    const fpr = normalizeFingerprint(peer.fingerprint);
    if (fpr === this.me()) {return;}
    
    // Mark peer as discovered when added to routing table
    const state = this.getPeerState(fpr);
    if (!state.discovered) {
      state.discovered = true;
      state.discoveredAt = Date.now();
      this.setPeerState(fpr, state);
    }
    
    const bucket = this.buckets[this.idx(fpr)];
    bucket.lastLookup = Date.now();
    const pos = bucket.peers.findIndex(p => normalizeFingerprint(p.fingerprint) === fpr);
    if (pos >= 0) {
      if (peer.address) {bucket.peers[pos].address = peer.address;}
      bucket.peers.push(bucket.peers.splice(pos, 1)[0]);
      return;
    }
    if (bucket.peers.length < DHT_K) {
      bucket.peers.push({ ...peer });
      console.log(`[DHT] Added ${fpr.slice(0, 16)}... to routing table (bucket ${this.idx(fpr)}, ${bucket.peers.length}/${DHT_K} peers)`);
      return;
    }
    const head = bucket.peers[0];
    this.ping(head).then(alive => {
      if (alive) { bucket.peers.push(bucket.peers.shift()!); }
      else { bucket.peers.shift(); bucket.peers.push({ ...peer }); this.drop(normalizeFingerprint(head.fingerprint)); }
    }).catch(() => { bucket.peers.shift(); bucket.peers.push({ ...peer }); this.drop(normalizeFingerprint(head.fingerprint)); });
  }

  private nearest(target: Fingerprint, n: number): Peer[] {
    const all: Peer[] = [];
    for (const b of this.buckets) {all.push(...b.peers);}
    const t = normalizeFingerprint(target);
    return all.sort((a, b) => cmp(xor(normalizeFingerprint(a.fingerprint), t), xor(normalizeFingerprint(b.fingerprint), t))).slice(0, n);
  }

  private fromTable(fpr: Fingerprint): Peer | undefined {
    for (const b of this.buckets) {
      const p = b.peers.find(p => normalizeFingerprint(p.fingerprint) === fpr);
      if (p) {return p;}
    }
  }

  // ── Iterative lookup ──────────────────────────────────────────────────────────

  private async lookup(target: Fingerprint): Promise<Peer | undefined> {
    const t = normalizeFingerprint(target);
    const seen = new Set<Fingerprint>([this.me()]);
    let cands = this.nearest(t, DHT_K);
    if (!cands.length) {return undefined;}
    let best = xor(normalizeFingerprint(cands[0].fingerprint), t);
    let improved = true;
    while (improved) {
      improved = false;
      const unseen = cands.filter(p => !seen.has(normalizeFingerprint(p.fingerprint))).slice(0, DHT_ALPHA);
      if (!unseen.length) {break;}
      const results = await Promise.allSettled(unseen.map(p => this.doFindPeer(p, t)));
      for (let i = 0; i < unseen.length; i++) {
        seen.add(normalizeFingerprint(unseen[i].fingerprint));
        if (results[i].status !== 'fulfilled') {continue;}
        for (const p of (results[i] as PromiseFulfilledResult<Peer[]>).value) {
          const pf = normalizeFingerprint(p.fingerprint);
          if (!seen.has(pf)) {
            cands.push(p);
            this.addPeer(p);
            this.connectRelay(p, unseen[i]).catch(() => {});
          }
        }
      }
      cands.sort((a, b) => cmp(xor(normalizeFingerprint(a.fingerprint), t), xor(normalizeFingerprint(b.fingerprint), t)));
      cands = cands.slice(0, DHT_K);
      const nb = xor(normalizeFingerprint(cands[0].fingerprint), t);
      if (cmp(nb, best) < 0) { best = nb; improved = true; }
    }
    return cands.find(p => normalizeFingerprint(p.fingerprint) === t);
  }

  // ── RPC ───────────────────────────────────────────────────────────────────────

  private async ping(peer: Peer): Promise<boolean> {
    const c = this.conns.get(normalizeFingerprint(peer.fingerprint));
    if (!c || c.ch.readyState !== 'open') {return false;}
    try { await this.rpc(c, { type: 'ping', id: rid() }, 3_000); c.lastSeen = Date.now(); return true; }
    catch { return false; }
  }

  private async doFindPeer(via: Peer, target: Fingerprint): Promise<Peer[]> {
    const c = this.conns.get(normalizeFingerprint(via.fingerprint));
    if (!c || c.ch.readyState !== 'open') {return [];}
    try {
      const id = rid();
      const res = await this.rpc(c, { type: 'find_peer', id, target }, 5_000) as { peers: Peer[] };
      c.lastSeen = Date.now();
      return res.peers ?? [];
    } catch { return []; }
  }

  private rpc(c: Conn, msg: DHTMsg, ms: number): Promise<unknown> {
    const id = (msg as { id: string }).id;
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => { this.pending.delete(id); reject(new Error(`DHT timeout (${msg.type})`)); }, ms);
      this.pending.set(id, { resolve, reject, timer });
      c.ch.send(JSON.stringify(msg));
    });
  }

  // ── Message dispatch ──────────────────────────────────────────────────────────

  private onMsg(senderFpr: Fingerprint, raw: string): void {
    let msg: DHTMsg;
    try { msg = JSON.parse(raw) as DHTMsg; } catch { return; }
    const c = this.conns.get(senderFpr);
    if (c) {c.lastSeen = Date.now();}

    switch (msg.type) {
      case 'ping':
        this.send(senderFpr, { type: 'pong', id: msg.id }); break;
      case 'pong': {
        const r = this.pending.get(msg.id);
        if (r) { clearTimeout(r.timer); this.pending.delete(msg.id); r.resolve({}); } break;
      }
      case 'find_peer':
        this.send(senderFpr, { type: 'find_peer_response', id: msg.id, peers: this.nearest(msg.target, DHT_K) }); break;
      case 'find_peer_response': {
        const r = this.pending.get(msg.id);
        if (r) { clearTimeout(r.timer); this.pending.delete(msg.id); r.resolve({ peers: msg.peers }); } break;
      }
      case 'relay_offer':  this.onOffer(senderFpr, msg).catch(() => {}); break;
      case 'relay_answer': this.onAnswer(senderFpr, msg).catch(() => {}); break;
      case 'relay_ice':    this.onIce(senderFpr, msg).catch(() => {}); break;
    }
  }

  private send(fpr: Fingerprint, msg: DHTMsg): void {
    const c = this.conns.get(fpr);
    if (c?.ch.readyState === 'open') {c.ch.send(JSON.stringify(msg));}
  }

  // ── Relay: offer ──────────────────────────────────────────────────────────────

  private async onOffer(sender: Fingerprint, msg: Extract<DHTMsg, { type: 'relay_offer' }>): Promise<void> {
    const to = normalizeFingerprint(msg.to);
    if (to !== this.me()) { this.send(to, msg); return; } // forward

    const from = normalizeFingerprint(msg.from);
    if (this.conns.has(from)) {
      console.log(`[DHT] Ignoring offer from ${from.slice(0, 16)}... (already connected)`);
      return;
    }
    
    // Handle simultaneous connection attempts (tie-breaking)
    // If we're currently connecting to this peer, decide which connection to keep
    // Use lexicographic fingerprint comparison: lower fingerprint wins (keeps their outbound connection)
    if (this.connecting.has(from) || this.rout.has(from)) {
      const myFpr = this.me();
      if (myFpr < from) {
        // My outbound connection wins - reject this inbound offer
        console.log(`[DHT] Rejecting inbound offer from ${from.slice(0, 16)}... (tie-break: my outbound wins)`);
        
        // Set a timeout to verify our outbound connection succeeds
        // If it doesn't complete within 30s, allow retry
        setTimeout(() => {
          if (!this.conns.has(from) && this.connecting.has(from)) {
            console.log(`[DHT] Tie-break outbound to ${from.slice(0, 16)}... failed to complete, cleaning up`);
            this.connecting.delete(from);
            const outbound = this.rout.get(from);
            if (outbound) {
              this.rout.delete(from);
              outbound.pc.close();
              outbound.reject(new Error('Connection tie-break: outbound timeout'));
            }
            
            // Update peer state to record the failed attempt (if not already counted)
            // This enables exponential backoff on next retry
            // Check if attempt was already counted by connectKnownPeers() by comparing timestamps
            const state = this.getPeerState(from);
            const now = Date.now();
            const wasRecentlyIncremented = state.lastAttempt > 0 && (now - state.lastAttempt) < 29_000;
            
            if (!wasRecentlyIncremented) {
              // Attempt not yet counted, increment now
              state.connectionAttempts++;
              state.lastAttempt = now;
              this.setPeerState(from, state);
              console.log(`[DHT] Tie-break timeout for ${from.slice(0, 16)}... - attempt ${state.connectionAttempts} recorded`);
            } else {
              console.log(`[DHT] Tie-break timeout for ${from.slice(0, 16)}... - attempt already counted`);
            }
          }
        }, 30_000);
        
        return;
      } else {
        // Their outbound connection wins - cancel my outbound attempt and accept this offer
        console.log(`[DHT] Accepting inbound offer from ${from.slice(0, 16)}... (tie-break: their outbound wins, canceling mine)`);
        
        // Cancel my outbound attempt
        this.connecting.delete(from);
        const outbound = this.rout.get(from);
        if (outbound) {
          this.rout.delete(from);
          outbound.pc.close();
          outbound.reject(new Error('Connection tie-break: inbound won'));
        }
      }
    }

    console.log(`[DHT] Received relay offer from ${from.slice(0, 16)}... via ${sender.slice(0, 16)}...`);

    // Validate offer before creating PeerConnection
    if (!msg.offer || !msg.offer.type || !msg.offer.sdp) {
      console.error(`[DHT] Invalid offer from ${from.slice(0, 16)}... - missing type or sdp`);
      return;
    }

    const pc = new RTCPeerConnection({ iceServers: this.ice });
    this.rin.set(from, { pc, queued: [] });
    
    try {
      // Monitor connection state changes
      let connectionState = 'new';
      pc.onconnectionstatechange = (): void => {
        const newState = pc.connectionState;
        if (newState !== connectionState) {
          console.log(`[DHT] Inbound from ${from.slice(0, 16)}... connection state: ${connectionState} → ${newState}`);
          connectionState = newState;
        }
      };

      pc.ondatachannel = (ev): void => {
        console.log(`[DHT] Data channel received from ${from.slice(0, 16)}... (state: ${ev.channel.readyState})`);

        // Monitor channel state changes
        ev.channel.onopen = (): void => {
          console.log(`[DHT] Inbound from ${from.slice(0, 16)}... data channel: opening → open`);
        };
        ev.channel.onerror = (err): void => {
          console.error(`[DHT] Inbound from ${from.slice(0, 16)}... data channel error:`, err);
        };
        
        this.register(pc, ev.channel, from).catch((err): void => {
          console.error(`[DHT] Inbound from ${from.slice(0, 16)}... registration failed:`, err.message);
        }); 
      };
      
      pc.onicecandidate = (ev): void => {
        if (ev.candidate) {
          console.log(`[DHT] Sending ICE candidate to ${from.slice(0, 16)}... via relay`);
          this.send(sender, { type: 'relay_ice', from: this.account.getFingerprint(), to: msg.from, candidate: ev.candidate });
        }
      };

      await pc.setRemoteDescription(msg.offer);
      const answer = await pc.createAnswer();
      await pc.setLocalDescription(answer);
      
      // Apply any queued ICE candidates that arrived early
      // Must happen AFTER setLocalDescription but BEFORE sending answer
      // This ensures ICE negotiation can use all candidates
      const inbound = this.rin.get(from);
      if (inbound && inbound.queued.length > 0) {
        console.log(`[DHT] Applying ${inbound.queued.length} queued ICE candidates from ${from.slice(0, 16)}...`);
        for (const candidate of inbound.queued) {
          await pc.addIceCandidate(candidate).catch(() => {});
        }
        inbound.queued = []; // Clear queue after applying
      }
      
      console.log(`[DHT] Sending answer to ${from.slice(0, 16)}... via ${sender.slice(0, 16)}...`);
      this.send(sender, { type: 'relay_answer', from: this.account.getFingerprint(), to: msg.from, answer });
    } catch (err) {
      // Clean up on failure
      console.error(`[DHT] Failed to process offer from ${from.slice(0, 16)}...:`, err);
      this.rin.delete(from);
      pc.close();
    }
  }

  // ── Relay: answer ─────────────────────────────────────────────────────────────

  private async onAnswer(sender: Fingerprint, msg: Extract<DHTMsg, { type: 'relay_answer' }>): Promise<void> {
    const to = normalizeFingerprint(msg.to);
    if (to !== this.me()) { this.send(to, msg); return; } // forward

    const from = normalizeFingerprint(msg.from);
    const out = this.rout.get(from);
    if (!out) {return;}
    try { await out.pc.setRemoteDescription(msg.answer); out.resolve(); }
    catch (e) { out.reject(e instanceof Error ? e : new Error(String(e))); }
  }

  // ── Relay: ICE ────────────────────────────────────────────────────────────────

  private async onIce(sender: Fingerprint, msg: Extract<DHTMsg, { type: 'relay_ice' }>): Promise<void> {
    const to = normalizeFingerprint(msg.to);
    if (to !== this.me()) { this.send(to, msg); return; } // forward

    const from = normalizeFingerprint(msg.from);
    const conn = this.conns.get(from);
    if (conn) { 
      console.log(`[DHT] Adding ICE candidate from ${from.slice(0, 16)}... (already connected)`);
      await conn.pc.addIceCandidate(msg.candidate).catch(() => {}); 
      return; 
    }
    const inbound = this.rin.get(from);
    if (inbound) { 
      console.log(`[DHT] Queueing ICE candidate from ${from.slice(0, 16)}... (inbound pending)`);
      inbound.queued.push(msg.candidate); 
      return; 
    }
    const out = this.rout.get(from);
    if (out) {
      console.log(`[DHT] Adding ICE candidate from ${from.slice(0, 16)}... (outbound)`);
      await out.pc.addIceCandidate(msg.candidate).catch(() => {});
    }
  }

  // ── Connect: HTTP (bootstrap) ─────────────────────────────────────────────────

  private async connectHTTP(peer: Peer): Promise<void> {
    const fpr = normalizeFingerprint(peer.fingerprint);
    if (this.conns.has(fpr)) {return;}
    const pc = new RTCPeerConnection({ iceServers: this.ice });
    const ch = pc.createDataChannel('dht', { ordered: true });
    
    // Disable automatic ICE gathering - we'll use pre-gathered candidates
    pc.onicecandidate = (): void => {
      // Intentionally empty - we reuse pre-gathered candidates
    };

    await pc.setLocalDescription(await pc.createOffer());
    
    // Build offer with pre-gathered ICE candidates
    const offerWithCandidates = {
      from: this.account.getFingerprint(),
      offer: pc.localDescription,
      candidates: this.localICECandidates.map(c => c.toJSON()),
    };
    
    console.log(`[DHT] Sending HTTP offer with ${this.localICECandidates.length} pre-gathered candidates`);
    
    const resp = await fetch(`https://${peer.address}/p2p/dht/offer`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(offerWithCandidates),
    });
    if (!resp.ok) {throw new Error(`DHT offer HTTP ${resp.status}`);}
    const { answer } = await resp.json() as { answer: RTCSessionDescriptionInit };
    await pc.setRemoteDescription(answer);
    await this.register(pc, ch, fpr);
    this.addPeer(peer);
  }

  // ── Connect: WebSocket (browser-compatible bootstrap) ────────────────────────

  private async connectWebSocket(peer: Peer): Promise<void> {
    const fpr = normalizeFingerprint(peer.fingerprint);
    if (this.conns.has(fpr)) {return;}

    const ws = new WebSocket(peer.address);
    const pc = new RTCPeerConnection({ iceServers: this.ice });
    const ch = pc.createDataChannel('dht', { ordered: true });

    try {
      // Wait for WebSocket to open
      await new Promise<void>((resolve, reject) => {
        ws.onopen = (): void => resolve();
        ws.onerror = (err): void => reject(new Error(`WebSocket connection failed: ${err}`));
        setTimeout(() => reject(new Error('WebSocket connection timeout')), 10_000);
      });

      console.log('[DHT] WebSocket connected, registering fingerprint...');

      // Register with signaling server
      ws.send(JSON.stringify({
        type: 'register',
        fingerprint: this.account.getFingerprint(),
      }));

      // Set up message handlers before creating offer
      const answerPromise = new Promise<RTCSessionDescriptionInit>((resolve, reject) => {
        const timeout = setTimeout(() => reject(new Error('Answer timeout')), 30_000);
        
        ws.onmessage = (event): void => {
          try {
            const msg = JSON.parse(event.data as string);
            
            if (msg.type === 'answer') {
              clearTimeout(timeout);
              resolve(msg.answer);
            } else if (msg.type === 'ice' && msg.candidate) {
              // Handle ICE candidates from peer
              pc.addIceCandidate(msg.candidate).catch(() => {});
            } else if (msg.type === 'error') {
              clearTimeout(timeout);
              reject(new Error(`Signaling error: ${msg.error}`));
            }
          } catch (err) {
            console.warn('[DHT] Failed to parse signaling message:', err);
          }
        };
      });

      // Disable automatic ICE gathering - we'll use pre-gathered candidates
      pc.onicecandidate = (): void => {
        // Intentionally empty - we reuse pre-gathered candidates
      };

      // Create and send offer
      await pc.setLocalDescription(await pc.createOffer());

      console.log(`[DHT] Sending offer via WebSocket...`);
      
      // Send offer
      ws.send(JSON.stringify({
        type: 'offer',
        from: this.account.getFingerprint(),
        to: peer.fingerprint,
        offer: pc.localDescription,
      }));

      // Wait for answer BEFORE sending ICE candidates
      const answer = await answerPromise;
      console.log('[DHT] Received answer, setting remote description...');
      await pc.setRemoteDescription(answer);
      
      // Send pre-gathered ICE candidates AFTER remote description is set
      console.log(`[DHT] Sending ${this.localICECandidates.length} pre-gathered ICE candidates via WebSocket...`);
      for (const candidate of this.localICECandidates) {
        ws.send(JSON.stringify({
          type: 'ice',
          from: this.account.getFingerprint(),
          to: peer.fingerprint,
          candidate: candidate.toJSON(),
        }));
      }

      // Wait for data channel to open
      await new Promise<void>((resolve, reject) => {
        const timeout = setTimeout(() => reject(new Error('Data channel timeout')), 30_000);
        ch.onopen = (): void => {
          clearTimeout(timeout);
          resolve();
        };
        ch.onerror = (err): void => {
          clearTimeout(timeout);
          reject(err);
        };
      });

      console.log('[DHT] WebRTC data channel opened, closing WebSocket...');

      // Close WebSocket once WebRTC is established
      ws.close();

      // Register the connection
      await this.register(pc, ch, fpr);
      this.addPeer(peer);

      console.log(`[DHT] Successfully connected to ${fpr.slice(0, 16)}... via WebSocket signaling`);
    } catch (error) {
      ws.close();
      pc.close();
      ch.close();
      throw error;
    }
  }

  // ── Connect: relay signaling ──────────────────────────────────────────────────

  async connectRelay(target: Peer, relay: Peer): Promise<void> {
    const tfpr = normalizeFingerprint(target.fingerprint);
    if (this.conns.has(tfpr) || this.rout.has(tfpr) || this.connecting.has(tfpr)) {return;}
    const rfpr = normalizeFingerprint(relay.fingerprint);
    const rc = this.conns.get(rfpr);
    if (!rc || rc.ch.readyState !== 'open') {throw new Error(`Relay ${relay.fingerprint} not connected`);}

    console.log(`[DHT] Starting relay connection to ${tfpr.slice(0, 16)}... via ${rfpr.slice(0, 16)}...`);
    
    // Mark as connecting to prevent duplicate attempts
    this.connecting.add(tfpr);
    
    try {
      const pc = new RTCPeerConnection({ iceServers: this.ice });
      const ch = pc.createDataChannel('dht', { ordered: true });
      
      // Monitor data channel state
      console.log(`[DHT] ${tfpr.slice(0, 16)}... created data channel (state: ${ch.readyState})`);
      ch.onopen = (): void => {
        console.log(`[DHT] ${tfpr.slice(0, 16)}... data channel: opening → open`);
      };
      ch.onerror = (err): void => {
        console.error(`[DHT] ${tfpr.slice(0, 16)}... data channel error:`, err);
      };
      ch.onclose = (): void => {
        console.log(`[DHT] ${tfpr.slice(0, 16)}... data channel closed`);
      };

      // Monitor connection state changes
      let connectionState = 'new';
      let iceGatheringState = 'new';
      let iceConnectionState = 'new';

      pc.onconnectionstatechange = (): void => {
        const newState = pc.connectionState;
        if (newState !== connectionState) {
          console.log(`[DHT] ${tfpr.slice(0, 16)}... connection state: ${connectionState} → ${newState}`);
          connectionState = newState;
        }
      };

      pc.onicegatheringstatechange = (): void => {
        const newState = pc.iceGatheringState;
        if (newState !== iceGatheringState) {
          console.log(`[DHT] ${tfpr.slice(0, 16)}... ICE gathering: ${iceGatheringState} → ${newState}`);
          iceGatheringState = newState;
        }
      };

      pc.oniceconnectionstatechange = (): void => {
        const newState = pc.iceConnectionState;
        if (newState !== iceConnectionState) {
          console.log(`[DHT] ${tfpr.slice(0, 16)}... ICE connection: ${iceConnectionState} → ${newState}`);
          iceConnectionState = newState;
        }
      };

      // Ignore automatic ICE candidates - we'll use pre-gathered ones
      pc.onicecandidate = (): void => {
        // Intentionally empty - we reuse pre-gathered candidates
      };

      await pc.setLocalDescription(await pc.createOffer());
      console.log(`[DHT] ${tfpr.slice(0, 16)}... offer created, sending via relay`);

      // Send offer and wait for answer BEFORE sending ICE candidates
      await new Promise<void>((resolve, reject) => {
        this.rout.set(tfpr, { pc, resolve, reject });
        this.send(rfpr, { type: 'relay_offer', from: this.account.getFingerprint(), to: target.fingerprint, offer: pc.localDescription! });
        setTimeout(() => { 
          if (this.rout.has(tfpr)) { 
            this.rout.delete(tfpr);
            console.log(`[DHT] ${tfpr.slice(0, 16)}... relay signaling timeout (30s) - no answer received`);
            reject(new Error(`relay timeout ${target.fingerprint}`)); 
          } 
        }, 30_000);
      });

      // Send pre-gathered ICE candidates AFTER remote description is set
      console.log(`[DHT] ${tfpr.slice(0, 16)}... answer received, sending ${this.localICECandidates.length} pre-gathered ICE candidates`);
      for (const candidate of this.localICECandidates) {
        this.send(rfpr, { 
          type: 'relay_ice', 
          from: this.account.getFingerprint(), 
          to: target.fingerprint, 
          candidate: candidate.toJSON() 
        });
      }

      console.log(`[DHT] ${tfpr.slice(0, 16)}... answer received, waiting for channel to open...`);
      this.rout.delete(tfpr);
      await this.register(pc, ch, tfpr);
      this.addPeer(target);
      console.log(`[DHT] ${tfpr.slice(0, 16)}... relay connection complete!`);
    } finally {
      // Always remove from connecting set, whether success or failure
      this.connecting.delete(tfpr);
    }
  }

  // ── Server side: accept inbound HTTP offer ────────────────────────────────────

  async handleHTTPOffer(fromFpr: Fingerprint, offer: RTCSessionDescriptionInit, peerAddress: string): Promise<RTCSessionDescriptionInit> {
    const fpr = normalizeFingerprint(fromFpr);
    
    // Validate offer before creating PeerConnection
    if (!offer?.type || !offer?.sdp) {
      throw new Error('Invalid offer: missing type or sdp');
    }
    
    const pc = new RTCPeerConnection({ iceServers: this.ice });
    this.rin.set(fpr, { pc, queued: [] });
    
    try {
      pc.ondatachannel = (ev): void => {
        this.register(pc, ev.channel, fpr).then((): void => this.addPeer({ fingerprint: fromFpr, address: peerAddress })).catch((): void => {});
      };
      await pc.setRemoteDescription(offer);
      await pc.setLocalDescription(await pc.createAnswer());
      await waitICE(pc);
      this.rin.delete(fpr);
      return pc.localDescription as RTCSessionDescriptionInit;
    } catch (err) {
      // Clean up on failure
      this.rin.delete(fpr);
      pc.close();
      throw err;
    }
  }

  // ── Register open connection ──────────────────────────────────────────────────

  private register(pc: RTCPeerConnection, ch: RTCDataChannel, fpr: Fingerprint): Promise<void> {
    return new Promise((resolve, reject): void => {
      const t = setTimeout((): void => {
        console.warn(`[DHT] Channel open timeout for ${fpr.slice(0, 16)}... (30s elapsed)`);
        reject(new Error(`channel open timeout ${fpr}`));
      }, 30_000);
      const onOpen = async (): Promise<void> => {
        clearTimeout(t);
        const conn: Conn = { pc, ch, lastSeen: Date.now() };
        this.conns.set(fpr, conn);
        ch.onmessage = (ev: MessageEvent<string>): void => this.onMsg(fpr, ev.data);
        ch.onclose = (): void => { this.conns.delete(fpr); };
        
        // Apply any remaining queued ICE candidates (fallback for edge cases)
        // Most candidates should have been applied in onOffer() after setLocalDescription
        const inbound = this.rin.get(fpr);
        if (inbound && inbound.queued.length > 0) {
          console.log(`[DHT] Applying ${inbound.queued.length} remaining queued ICE candidates for ${fpr.slice(0, 16)}...`);
          for (const c of inbound.queued) {
            await pc.addIceCandidate(c).catch((): void => {});
          }
        }
        if (inbound) { this.rin.delete(fpr); }
        
        resolve();
      };
      if (ch.readyState === 'open') { onOpen().catch(reject); }
      else { ch.onopen = (): void => { onOpen().catch(reject); }; ch.onerror = (): void => { clearTimeout(t); reject(new Error('channel error')); }; }
    });
  }

  private drop(fpr: Fingerprint): void {
    const c = this.conns.get(fpr);
    if (c) { c.ch.close(); c.pc.close(); this.conns.delete(fpr); }
  }

  // ── Bucket refresh ────────────────────────────────────────────────────────────

  private async refresh(): Promise<void> {
    const now = Date.now();
    for (let i = 0; i < DHT_B; i++) {
      const b = this.buckets[i];
      if (!b.peers.length || now - b.lastLookup < DHT_STALL_PERIOD_MS) {continue;}
      await this.lookup(b.peers[Math.floor(Math.random() * b.peers.length)].fingerprint).catch(() => {});
      b.lastLookup = Date.now();
    }
  }

  /**
   * Bootstrap discovery: Aggressively query for peers when we have less than DHT_K
   * Runs every 3 seconds until we have at least DHT_K peers
   * Helps discover peers that join after initial bootstrap
   */
  private async bootstrapDiscovery(): Promise<void> {
    const stats = this.stats();
    
    // Stop aggressive discovery once we have enough connections (excluding ourselves)
    if (stats.connected >= DHT_K) {
      console.log(`[DHT] Bootstrap discovery complete: ${stats.connected} connections established`);
      this.bootstrapActive = false;
      if (this.bootstrapTimer) {
        clearInterval(this.bootstrapTimer);
        this.bootstrapTimer = null;
      }
      return;
    }
    
    // Log current state
    const nonEmptyBuckets = stats.bucketFill.filter(n => n > 0).length;
    console.log(`[DHT] Bootstrap discovery: ${stats.connected} connected, ${stats.discovered} discovered, ${nonEmptyBuckets} buckets with peers`);
    
    // Phase 1: Discover new peers (only if needed)
    await this.discoverPeers();
    
    // Phase 2: Connect to known peers (respects cooldown and max attempts)
    await this.connectKnownPeers();
  }
  
  /**
   * Attempt relay connections to specified peers.
   * Updates peer state for connection attempts and respects cooldown periods.
   * 
   * @param peersToConnect - Optional array of specific peers to connect to.
   *                         If not provided, attempts all unconnected peers in routing table.
   */
  private async connectDiscoveredPeers(peersToConnect?: Peer[]): Promise<void> {
    const now = Date.now();
    let unconnectedPeers: Peer[];
    
    if (peersToConnect) {
      // Use provided peers
      unconnectedPeers = peersToConnect;
    } else {
      // Find all unconnected peers in routing table
      unconnectedPeers = [];
      for (const bucket of this.buckets) {
        for (const peer of bucket.peers) {
          const fpr = normalizeFingerprint(peer.fingerprint);
          
          // Skip self (bootstrap servers register themselves in routing table)
          if (fpr === this.me()) {
            continue;
          }
          
          // Skip if already connected OR connection attempt in progress
          if (this.conns.has(fpr) || this.rout.has(fpr) || this.connecting.has(fpr)) {
            continue;
          }
          
          const state = this.getPeerState(fpr);
          
          // Skip if we've exceeded retry limit for this peer
          if (state.connectionAttempts >= this.maxConnectionAttempts) {
            continue;
          }
          
          unconnectedPeers.push(peer);
        }
      }
    }
    
    if (unconnectedPeers.length === 0) {
      return; // No new peers to connect to
    }
    
    console.log(`[DHT] Found ${unconnectedPeers.length} unconnected peer(s) to connect`);
    
    // Find a connected peer to use as relay
    const relayPeer = Array.from(this.conns.keys())
      .map(fpr => this.fromTable(normalizeFingerprint(fpr)))
      .find(p => p !== undefined);
    
    if (!relayPeer) {
      console.log(`[DHT] No connected relay peer available`);
      return;
    }
    
    console.log(`[DHT] Using ${relayPeer.fingerprint.slice(0, 16)}... as relay`);
    
    // Attempt connections (limit to 3 at a time to reduce contention)
    for (const peer of unconnectedPeers.slice(0, 3)) {
      const fpr = normalizeFingerprint(peer.fingerprint);
      const state = this.getPeerState(fpr);
      
      try {
        // Update state before attempting
        state.lastAttempt = now;
        state.connectionAttempts++;
        this.setPeerState(fpr, state);
        
        await this.connectRelay(peer, relayPeer);
        
        // Success - reset attempt counter
        state.connectionAttempts = 0;
        this.setPeerState(fpr, state);
        
        console.log(`[DHT] Connected to ${peer.fingerprint.slice(0, 16)}...`);
      } catch (err) {
        console.warn(`[DHT] Failed to connect to ${peer.fingerprint.slice(0, 16)}...:`, err);
        
        // State already updated with attempt count
        if (state.connectionAttempts >= this.maxConnectionAttempts) {
          console.log(`[DHT] Giving up on ${peer.fingerprint.slice(0, 16)}... after ${state.connectionAttempts} attempts`);
        }
      }
    }
  }
}
