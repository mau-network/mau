/**
 * mDNS Service Discovery for Mau P2P Network
 * 
 * Implements local network peer discovery via mDNS multicast as specified in the Mau spec.
 * Service name format: <fingerprint>._mau._tcp.local.
 * 
 * Example: 5D000B2F2C040A1675B49D7F0C7CB7DC36999D56._mau._tcp.local.
 */

import type { Fingerprint } from '../types/index.js';
import { normalizeFingerprint } from '../crypto/index.js';

export interface MDNSService {
  fingerprint: Fingerprint;
  address: string;
  port: number;
}

export interface MDNSOptions {
  /** Port to advertise for Mau HTTP server */
  port: number;
  /** Custom mDNS domain (default: "local.") */
  domain?: string;
}

/**
 * MauMDNS - Local network service discovery
 * 
 * Announces this peer's presence on the local network and discovers other peers.
 * Browser environments should use mdns-js or similar polyfill.
 * Node environments should use mdns or bonjour package.
 */
export class MauMDNS {
  private fingerprint: Fingerprint;
  private options: Required<MDNSOptions>;
  private announced: boolean = false;

  constructor(fingerprint: Fingerprint, options: MDNSOptions) {
    this.fingerprint = normalizeFingerprint(fingerprint);
    this.options = {
      domain: 'local.',
      ...options,
    };
  }

  /**
   * Announce this peer on the local network
   * 
   * Service format: <fingerprint>._mau._tcp.local.
   */
  async announce(): Promise<void> {
    if (this.announced) {
      return;
    }

    // Platform-specific implementation needed
    // Node.js: use 'mdns' or 'bonjour' package
    // Browser: use 'mdns-js' or similar
    
    console.log(`[mDNS] Would announce: ${this.fingerprint}._mau._tcp.${this.options.domain} on port ${this.options.port}`);
    this.announced = true;

    // TODO: Implement platform-specific mDNS announcement
    // Example using 'mdns' package (Node.js):
    // const mdns = require('mdns');
    // const serviceName = `${this.fingerprint}._mau._tcp`;
    // this.ad = mdns.createAdvertisement(mdns.tcp('mau'), this.options.port, {
    //   name: this.fingerprint,
    // });
    // this.ad.start();
  }

  /**
   * Discover peers on the local network
   * 
   * Returns a list of discovered Mau services with their fingerprints and addresses.
   */
  async discover(): Promise<MDNSService[]> {
    // TODO: Implement platform-specific mDNS discovery
    // Example using 'mdns' package (Node.js):
    // const mdns = require('mdns');
    // const browser = mdns.createBrowser(mdns.tcp('mau'));
    // browser.on('serviceUp', (service) => {
    //   // Parse fingerprint from service name
    //   // Add to discovered services
    // });
    // browser.start();
    
    console.log('[mDNS] Discovery not yet implemented - requires mdns package');
    return [];
  }

  /**
   * Stop announcing this peer
   */
  async shutdown(): Promise<void> {
    if (!this.announced) {
      return;
    }

    console.log('[mDNS] Shutting down mDNS announcement');
    this.announced = false;

    // TODO: Stop mDNS advertisement
    // if (this.ad) {
    //   this.ad.stop();
    // }
  }
}

/**
 * Create an mDNS service discovery instance
 * 
 * @param fingerprint Account fingerprint to announce
 * @param options Configuration including port
 */
export function createMDNS(fingerprint: Fingerprint, options: MDNSOptions): MauMDNS {
  return new MauMDNS(fingerprint, options);
}
