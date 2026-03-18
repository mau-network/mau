/**
 * Network Utilities - Resolvers and Discovery
 */

import pRetry, { AbortError } from 'p-retry';
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
    try {
      return await pRetry(
        async () => {
          const result = await resolver(fingerprint, timeout);
          if (result === null) {throw new AbortError('not found');}
          return result;
        },
        { retries: maxRetries, minTimeout: initialDelayMs, factor: 2 }
      );
    } catch (err) {
      if (!(err instanceof AbortError)) {
        console.error(`Resolver failed after ${maxRetries + 1} attempts:`, (err as Error)?.message);
      }
      return null;
    }
  };
}
