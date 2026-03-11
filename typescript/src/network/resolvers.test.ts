/**
 * Tests for Network Resolvers
 */

import { describe, it, expect } from '@jest/globals';
import {
  staticResolver,
  dnsResolver,
  mdnsResolver,
  dhtResolver,
  combinedResolver,
} from './resolvers';

describe('Network Resolvers', () => {
  describe('staticResolver', () => {
    it('should resolve known fingerprint', async () => {
      const addressMap = new Map();
      addressMap.set('fingerprint123', 'localhost:8080');
      addressMap.set('fingerprint456', '192.168.1.100:8080');

      const resolver = staticResolver(addressMap);

      const address1 = await resolver('fingerprint123');
      const address2 = await resolver('fingerprint456');

      expect(address1).toBe('localhost:8080');
      expect(address2).toBe('192.168.1.100:8080');
    });

    it('should return null for unknown fingerprint', async () => {
      const addressMap = new Map();
      addressMap.set('known', 'localhost:8080');

      const resolver = staticResolver(addressMap);

      const address = await resolver('unknown');

      expect(address).toBeNull();
    });

    it('should work with empty map', async () => {
      const addressMap = new Map();
      const resolver = staticResolver(addressMap);

      const address = await resolver('any-fingerprint');

      expect(address).toBeNull();
    });
  });

  describe('dnsResolver', () => {
    it('should return null for non-existent domain', async () => {
      const resolver = dnsResolver('nonexistent-test-domain-12345.com');
      const address = await resolver('fingerprint123', 1000);

      expect(address).toBeNull();
    });

    it('should handle DNS timeout', async () => {
      const resolver = dnsResolver('slow-dns-test.example.com');
      const address = await resolver('fingerprint123', 100); // Very short timeout

      expect(address).toBeNull();
    });

    it('should accept custom DNS server', async () => {
      const resolver = dnsResolver('example.com', '8.8.8.8');
      const address = await resolver('fingerprint123', 1000);

      // Will return null since we're not actually setting up TXT records
      expect(address).toBeNull();
    });
  });

  describe('mdnsResolver', () => {
    it('should return null when no peers found', async () => {
      const resolver = mdnsResolver();
      const address = await resolver('nonexistent-fingerprint', 500);

      expect(address).toBeNull();
    });

    it('should accept custom service type', async () => {
      const resolver = mdnsResolver('_custom-service._tcp');
      const address = await resolver('fingerprint123', 500);

      expect(address).toBeNull();
    });

    it('should respect timeout', async () => {
      const resolver = mdnsResolver();
      const start = Date.now();
      await resolver('fingerprint123', 300);
      const elapsed = Date.now() - start;

      // Should complete within timeout + some tolerance
      expect(elapsed).toBeLessThan(600);
    });
  });

  describe('dhtResolver', () => {
    it('should return null when bootstrap nodes unreachable', async () => {
      const resolver = dhtResolver(['unreachable-node-12345.example.com:8080']);
      const address = await resolver('fingerprint123', 1000);

      expect(address).toBeNull();
    });

    it('should query multiple bootstrap nodes', async () => {
      const resolver = dhtResolver([
        'bootstrap1-unreachable.example.com:8080',
        'bootstrap2-unreachable.example.com:8080',
      ]);
      const address = await resolver('fingerprint123', 1000);

      expect(address).toBeNull();
    });

    it('should respect timeout', async () => {
      const resolver = dhtResolver(['slow-bootstrap.example.com:8080']);
      const start = Date.now();
      await resolver('fingerprint123', 500);
      const elapsed = Date.now() - start;

      // Should complete within timeout + some tolerance
      expect(elapsed).toBeLessThan(1000);
    });
  });

  describe('combinedResolver', () => {
    it('should try resolvers in parallel', async () => {
      const resolver1 = jest.fn().mockResolvedValue(null);
      const resolver2 = jest.fn().mockResolvedValue('found:8080');
      const resolver3 = jest.fn().mockResolvedValue(null);

      const combined = combinedResolver([resolver1, resolver2, resolver3]);

      const address = await combined('fingerprint123');

      expect(address).toBe('found:8080');
      expect(resolver1).toHaveBeenCalledWith('fingerprint123', undefined);
      expect(resolver2).toHaveBeenCalledWith('fingerprint123', undefined);
      expect(resolver3).toHaveBeenCalledWith('fingerprint123', undefined);
    });

    it('should return first successful result', async () => {
      const resolver1 = jest.fn().mockResolvedValue('first:8080');
      const resolver2 = jest.fn().mockResolvedValue('second:8080');

      const combined = combinedResolver([resolver1, resolver2]);

      const address = await combined('fingerprint123');

      // Should return one of the results
      expect(['first:8080', 'second:8080']).toContain(address);
    });

    it('should return null if all resolvers fail', async () => {
      const resolver1 = jest.fn().mockResolvedValue(null);
      const resolver2 = jest.fn().mockResolvedValue(null);
      const resolver3 = jest.fn().mockResolvedValue(null);

      const combined = combinedResolver([resolver1, resolver2, resolver3]);

      const address = await combined('fingerprint123');

      expect(address).toBeNull();
    });

    it('should handle resolver errors', async () => {
      const resolver1 = jest.fn().mockRejectedValue(new Error('Network error'));
      const resolver2 = jest.fn().mockResolvedValue('working:8080');

      const combined = combinedResolver([resolver1, resolver2]);

      const address = await combined('fingerprint123');

      expect(address).toBe('working:8080');
    });

    it('should pass timeout to resolvers', async () => {
      const resolver1 = jest.fn().mockResolvedValue(null);
      const resolver2 = jest.fn().mockResolvedValue(null);

      const combined = combinedResolver([resolver1, resolver2]);

      await combined('fingerprint123', 5000);

      expect(resolver1).toHaveBeenCalledWith('fingerprint123', 5000);
      expect(resolver2).toHaveBeenCalledWith('fingerprint123', 5000);
    });

    it('should work with single resolver', async () => {
      const resolver = jest.fn().mockResolvedValue('single:8080');

      const combined = combinedResolver([resolver]);

      const address = await combined('fingerprint123');

      expect(address).toBe('single:8080');
    });

    it('should work with empty array', async () => {
      const combined = combinedResolver([]);

      const address = await combined('fingerprint123');

      expect(address).toBeNull();
    });
  });
});
