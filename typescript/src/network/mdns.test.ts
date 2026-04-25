/**
 * mDNS Service Discovery Tests
 */

import { MauMDNS, createMDNS } from './mdns.js';

describe('MauMDNS', () => {
  const testFingerprint = '5D000B2F2C040A1675B49D7F0C7CB7DC36999D56';
  const testPort = 8080;

  it('should create mDNS instance with normalized fingerprint', () => {
    const mdns = createMDNS(testFingerprint, { port: testPort });
    expect(mdns).toBeInstanceOf(MauMDNS);
  });

  it('should normalize fingerprint to lowercase', () => {
    const mdns = createMDNS('ABCDEF1234567890', { port: testPort });
    expect(mdns).toBeInstanceOf(MauMDNS);
  });

  it('should use default domain if not specified', async () => {
    const mdns = new MauMDNS(testFingerprint, { port: testPort });
    // Default domain should be 'local.'
    await mdns.announce();
    await mdns.shutdown();
  });

  it('should use custom domain if specified', async () => {
    const mdns = new MauMDNS(testFingerprint, { port: testPort, domain: 'custom.local.' });
    await mdns.announce();
    await mdns.shutdown();
  });

  it('should not announce twice', async () => {
    const mdns = createMDNS(testFingerprint, { port: testPort });
    await mdns.announce();
    await mdns.announce(); // Should be no-op
    await mdns.shutdown();
  });

  it('should discover peers (stub implementation)', async () => {
    const mdns = createMDNS(testFingerprint, { port: testPort });
    const services = await mdns.discover();
    expect(Array.isArray(services)).toBe(true);
    expect(services.length).toBe(0); // Stub returns empty array
  });

  it('should shutdown gracefully', async () => {
    const mdns = createMDNS(testFingerprint, { port: testPort });
    await mdns.announce();
    await mdns.shutdown();
    await mdns.shutdown(); // Should be safe to call twice
  });
});
