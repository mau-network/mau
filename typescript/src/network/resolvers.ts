/**
 * Network Utilities - Resolvers and Discovery
 */

import type { Fingerprint, FingerprintResolver } from '../types/index.js';
import type { KademliaDHT } from './dht.js';

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
  let resolverAvailable: boolean | null = null;
  
  return async (fingerprint: Fingerprint, timeout = 5000) => {
    try {
      // Check if resolver was already determined to be unavailable
      if (resolverAvailable === false) {
        return null;
      }
      
      // Dynamic import for Node.js-only library
      const DNS2 = (await import('dns2')).default;
      const { Packet } = DNS2;
      
      resolverAvailable = true;

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

      // No matching TXT record found (peer not published)
      return null;
    } catch (err) {
      const error = err instanceof Error ? err : new Error(String(err));
      
      // Module not available (browser environment)
      if ('code' in error && (error as NodeJS.ErrnoException).code === 'MODULE_NOT_FOUND' || error?.message?.includes('Cannot find module')) {
        if (resolverAvailable === null) {
          console.warn('DNS resolver not available in browser environment');
          resolverAvailable = false;
        }
        return null;
      }
      
      // Timeout (treat as peer not found, could retry)
      if (error?.message === 'DNS timeout') {
        return null;
      }
      
      // Network errors or invalid DNS server - log but return null
      console.error(`DNS resolution failed for ${fingerprint}:`, error.message);
      return null;
    }
  };
}



/**
 * DHT resolver — wraps a {@link KademliaDHT} instance as a {@link FingerprintResolver}.
 *
 * Perform an iterative Kademlia lookup over the WebRTC-connected DHT network
 * and return the peer's HTTP address, or null if not found.
 *
 * @param dht A joined KademliaDHT instance
 */
export function dhtResolver(dht: KademliaDHT): FingerprintResolver {
  return dht.resolver();
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

/**
 * Retry resolver with exponential backoff
 * 
 * Wraps another resolver and retries on failure with exponential delays
 * 
 * @param resolver Base resolver to retry
 * @param maxRetries Maximum number of retry attempts (default: 3)
 * @param initialDelayMs Initial delay in milliseconds (default: 100ms)
 */
export function retryResolver(
  resolver: FingerprintResolver,
  maxRetries = 3,
  initialDelayMs = 100
): FingerprintResolver {
  return async (fingerprint: Fingerprint, timeout?: number) => {
    let lastError: Error | null = null;
    
    for (let attempt = 0; attempt <= maxRetries; attempt++) {
      try {
        const result = await resolver(fingerprint, timeout);
        if (result) {
          return result;
        }
        // Null result = not found, don't retry
        return null;
      } catch (err) {
        lastError = err as Error;
        
        // Don't retry on last attempt
        if (attempt < maxRetries) {
          // Exponential backoff: 100ms, 200ms, 400ms, etc.
          const delayMs = initialDelayMs * Math.pow(2, attempt);
          await new Promise(resolve => setTimeout(resolve, delayMs));
        }
      }
    }
    
    // All retries exhausted
    console.error(`Resolver failed after ${maxRetries + 1} attempts:`, lastError?.message);
    return null;
  };
}
