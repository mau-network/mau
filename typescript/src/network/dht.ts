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
  private timer: ReturnType<typeof setInterval> | null = null;

  constructor(account: Account, iceServers: RTCIceServer[] = [{ urls: 'stun:stun.l.google.com:19302' }]) {
    this.account = account;
    this.ice = iceServers;
  }

  // ── Public ──────────────────────────────────────────────────────────────────

  async join(bootstrapPeers: Peer[]): Promise<void> {
    await Promise.allSettled(bootstrapPeers.map(p =>
      this.connectHTTP(p).catch(e => console.warn(`[DHT] bootstrap ${p.fingerprint}:`, e))
    ));
    await this.lookup(this.me());
    this.timer = setInterval(() => this.refresh().catch(() => {}), 60_000);
  }

  stop(): void {
    if (this.timer) { clearInterval(this.timer); this.timer = null; }
    for (const c of this.conns.values()) { c.ch.close(); c.pc.close(); }
    this.conns.clear();
  }

  resolver(): FingerprintResolver {
    return (fpr: Fingerprint) => this.findAddress(fpr);
  }

  async findAddress(target: Fingerprint): Promise<string | null> {
    const fpr = normalizeFingerprint(target);
    const local = this.fromTable(fpr);
    if (local?.address) {return local.address;}
    const found = await this.lookup(fpr);
    return found?.address ?? null;
  }

  // ── Routing table ────────────────────────────────────────────────────────────

  private me(): Fingerprint { return normalizeFingerprint(this.account.getFingerprint()); }

  private idx(fpr: Fingerprint): number {
    return Math.min(leadingZeros(xor(this.me(), normalizeFingerprint(fpr))), DHT_B - 1);
  }

  private addPeer(peer: Peer): void {
    const fpr = normalizeFingerprint(peer.fingerprint);
    if (fpr === this.me()) {return;}
    const bucket = this.buckets[this.idx(fpr)];
    bucket.lastLookup = Date.now();
    const pos = bucket.peers.findIndex(p => normalizeFingerprint(p.fingerprint) === fpr);
    if (pos >= 0) {
      if (peer.address) {bucket.peers[pos].address = peer.address;}
      bucket.peers.push(bucket.peers.splice(pos, 1)[0]);
      return;
    }
    if (bucket.peers.length < DHT_K) { bucket.peers.push({ ...peer }); return; }
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
    if (this.conns.has(from)) {return;}

    const pc = new RTCPeerConnection({ iceServers: this.ice });
    this.rin.set(from, { pc, queued: [] });

    pc.ondatachannel = (ev): void => { this.register(pc, ev.channel, from).catch((): void => {}); };
    pc.onicecandidate = (ev): void => {
      if (ev.candidate) {this.send(sender, { type: 'relay_ice', from: this.account.getFingerprint(), to: msg.from, candidate: ev.candidate });}
    };

    await pc.setRemoteDescription(msg.offer);
    const answer = await pc.createAnswer();
    await pc.setLocalDescription(answer);
    this.send(sender, { type: 'relay_answer', from: this.account.getFingerprint(), to: msg.from, answer });
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
    if (conn) { await conn.pc.addIceCandidate(msg.candidate).catch(() => {}); return; }
    const inbound = this.rin.get(from);
    if (inbound) { inbound.queued.push(msg.candidate); return; }
    const out = this.rout.get(from);
    if (out) {await out.pc.addIceCandidate(msg.candidate).catch(() => {});}
  }

  // ── Connect: HTTP (bootstrap) ─────────────────────────────────────────────────

  private async connectHTTP(peer: Peer): Promise<void> {
    const fpr = normalizeFingerprint(peer.fingerprint);
    if (this.conns.has(fpr)) {return;}
    const pc = new RTCPeerConnection({ iceServers: this.ice });
    const ch = pc.createDataChannel('dht', { ordered: true });
    await pc.setLocalDescription(await pc.createOffer());
    await waitICE(pc);
    const resp = await fetch(`https://${peer.address}/p2p/dht/offer`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ from: this.account.getFingerprint(), offer: pc.localDescription }),
    });
    if (!resp.ok) {throw new Error(`DHT offer HTTP ${resp.status}`);}
    const { answer } = await resp.json() as { answer: RTCSessionDescriptionInit };
    await pc.setRemoteDescription(answer);
    await this.register(pc, ch, fpr);
    this.addPeer(peer);
  }

  // ── Connect: relay signaling ──────────────────────────────────────────────────

  async connectRelay(target: Peer, relay: Peer): Promise<void> {
    const tfpr = normalizeFingerprint(target.fingerprint);
    if (this.conns.has(tfpr) || this.rout.has(tfpr)) {return;}
    const rfpr = normalizeFingerprint(relay.fingerprint);
    const rc = this.conns.get(rfpr);
    if (!rc || rc.ch.readyState !== 'open') {throw new Error(`Relay ${relay.fingerprint} not connected`);}

    const pc = new RTCPeerConnection({ iceServers: this.ice });
    const ch = pc.createDataChannel('dht', { ordered: true });

    pc.onicecandidate = (ev): void => {
      if (ev.candidate) {this.send(rfpr, { type: 'relay_ice', from: this.account.getFingerprint(), to: target.fingerprint, candidate: ev.candidate });}
    };

    await pc.setLocalDescription(await pc.createOffer());

    await new Promise<void>((resolve, reject) => {
      this.rout.set(tfpr, { pc, resolve, reject });
      this.send(rfpr, { type: 'relay_offer', from: this.account.getFingerprint(), to: target.fingerprint, offer: pc.localDescription! });
      setTimeout(() => { if (this.rout.has(tfpr)) { this.rout.delete(tfpr); reject(new Error(`relay timeout ${target.fingerprint}`)); } }, 30_000);
    });

    this.rout.delete(tfpr);
    await this.register(pc, ch, tfpr);
    this.addPeer(target);
  }

  // ── Server side: accept inbound HTTP offer ────────────────────────────────────

  async handleHTTPOffer(fromFpr: Fingerprint, offer: RTCSessionDescriptionInit, peerAddress: string): Promise<RTCSessionDescriptionInit> {
    const fpr = normalizeFingerprint(fromFpr);
    const pc = new RTCPeerConnection({ iceServers: this.ice });
    this.rin.set(fpr, { pc, queued: [] });
    pc.ondatachannel = (ev): void => {
      this.register(pc, ev.channel, fpr).then((): void => this.addPeer({ fingerprint: fromFpr, address: peerAddress })).catch((): void => {});
    };
    await pc.setRemoteDescription(offer);
    await pc.setLocalDescription(await pc.createAnswer());
    await waitICE(pc);
    this.rin.delete(fpr);
    return pc.localDescription as RTCSessionDescriptionInit;
  }

  // ── Register open connection ──────────────────────────────────────────────────

  private register(pc: RTCPeerConnection, ch: RTCDataChannel, fpr: Fingerprint): Promise<void> {
    return new Promise((resolve, reject): void => {
      const t = setTimeout((): void => reject(new Error(`channel open timeout ${fpr}`)), 30_000);
      const onOpen = async (): Promise<void> => {
        clearTimeout(t);
        const conn: Conn = { pc, ch, lastSeen: Date.now() };
        this.conns.set(fpr, conn);
        ch.onmessage = (ev: MessageEvent<string>): void => this.onMsg(fpr, ev.data);
        ch.onclose = (): void => { this.conns.delete(fpr); };
        const inbound = this.rin.get(fpr);
        if (inbound) { for (const c of inbound.queued) {await pc.addIceCandidate(c).catch((): void => {});} this.rin.delete(fpr); }
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
}
