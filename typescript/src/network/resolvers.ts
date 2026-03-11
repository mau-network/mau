/**
 * Network Utilities - Resolvers and Discovery
 */

import type { Fingerprint, FingerprintResolver } from '../types/index.js';

/**
 * Static address resolver
 * Maps fingerprints to known addresses
 */
export function staticResolver(
  addressMap: Map<Fingerprint, string>
): FingerprintResolver {
  return async (fingerprint: Fingerprint) => {
    return addressMap.get(fingerprint) || null;
  };
}

/**
 * DNS resolver - looks up TXT records for _mau.<fingerprint>.<domain>
 * 
 * ⚠️ **Node.js only** - Requires UDP sockets (not available in browsers)
 * 
 * @param domain Base domain for DNS lookups (e.g., "mau.network")
 * @param dnsServer Optional DNS server address (defaults to system resolver)
 */
export function dnsResolver(
  domain: string,
  dnsServer?: string
): FingerprintResolver {
  return async (fingerprint: Fingerprint, timeout = 5000) => {
    try {
      // Dynamic import for Node.js-only library
      const DNS2 = (await import('dns2')).default;
      const { Packet } = DNS2;

      const options: any = {};
      if (dnsServer) {
        options.nameServers = [dnsServer];
      }

      const resolver = new DNS2(options);

      const hostname = `_mau.${fingerprint}.${domain}`;
      
      const response = await Promise.race([
        resolver.resolve(hostname, 'TXT'),
        new Promise<never>((_, reject) =>
          setTimeout(() => reject(new Error('DNS timeout')), timeout)
        ),
      ]);

      // Parse TXT records for address
      if (response && 'answers' in response && response.answers.length > 0) {
        for (const answer of response.answers) {
          if (answer.type === Packet.TYPE.TXT && 'data' in answer) {
            // TXT record format: "mau-address=hostname:port"
            const txtData = String(answer.data || '');
            const match = txtData.match(/mau-address=(.+)/);
            if (match) {
              return match[1];
            }
          }
        }
      }

      return null;
    } catch (err) {
      // DNS lookup failed or dns2 not available (browser)
      if ((err as any)?.code === 'MODULE_NOT_FOUND' || (err as any)?.message?.includes('Cannot find module')) {
        console.warn('DNS resolver not available in browser environment');
      }
      return null;
    }
  };
}



/**
 * DHT resolver - uses Kademlia DHT for peer discovery
 * 
 * @param bootstrapNodes List of known bootstrap node addresses
 */
export function dhtResolver(bootstrapNodes: string[]): FingerprintResolver {
  // Simple in-memory DHT implementation
  // In production, this would use a full Kademlia implementation
  const routingTable = new Map<Fingerprint, string>();
  
  return async (fingerprint: Fingerprint, timeout = 5000) => {
    // Check local routing table first
    const cached = routingTable.get(fingerprint);
    if (cached) {
      return cached;
    }

    // Query bootstrap nodes
    // This is a simplified implementation - a full DHT would:
    // 1. Calculate XOR distance to target
    // 2. Query K closest nodes iteratively
    // 3. Update routing table with discovered peers
    
    try {
      // For now, we'll make HTTP requests to bootstrap nodes
      // to find the peer (simplified Kademlia FIND_NODE)
      
      for (const bootstrapNode of bootstrapNodes) {
        try {
          const controller = new AbortController();
          const timeoutId = setTimeout(() => controller.abort(), timeout);
          
          // Query bootstrap node's DHT endpoint
          const response = await fetch(`https://${bootstrapNode}/kad/find_peer/${fingerprint}`, {
            signal: controller.signal,
            headers: {
              'Accept': 'application/json',
            },
          });
          
          clearTimeout(timeoutId);

          if (response.ok) {
            const peers = await response.json() as Array<{ fingerprint: string; address: string }>;
            
            // Cache discovered peers in routing table
            for (const peer of peers) {
              routingTable.set(peer.fingerprint, peer.address);
            }
            
            // Check if we found the target
            const result = routingTable.get(fingerprint);
            if (result) {
              return result;
            }
          }
        } catch (err) {
          // Bootstrap node unreachable, try next
          continue;
        }
      }

      return null;
    } catch (err) {
      return null;
    }
  };
}

/**
 * Combined resolver - tries multiple resolvers in parallel
 */
export function combinedResolver(resolvers: FingerprintResolver[]): FingerprintResolver {
  return async (fingerprint: Fingerprint, timeout?: number) => {
    const results = await Promise.allSettled(
      resolvers.map((resolver) => resolver(fingerprint, timeout))
    );

    for (const result of results) {
      if (result.status === 'fulfilled' && result.value) {
        return result.value;
      }
    }

    return null;
  };
}
