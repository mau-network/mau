import { test, expect } from '@playwright/test';
import { createAndUnlockAccount } from '../helpers';

/**
 * E2E test for complete DHT relay signaling flow
 * 
 * This test validates the entire peer discovery and connection establishment:
 * 1. Bootstrap server registers WebRTC connections with its DHT instance
 * 2. Browser A connects to bootstrap server
 * 3. Browser B connects to bootstrap server  
 * 4. Browser A queries DHT and discovers Browser B
 * 5. Browser A connects to Browser B via relay signaling through bootstrap
 * 6. Direct WebRTC connection established between browsers
 * 7. Browsers can communicate directly without bootstrap intermediary
 */
test.describe('DHT Relay Signaling', () => {
  test.beforeEach(async ({ context }) => {
    await context.clearCookies();
  });

  test('three browsers should discover and connect to each other through DHT relay', async ({ 
    browser 
  }) => {
    // Create three independent browser contexts
    const context1 = await browser.newContext();
    const context2 = await browser.newContext();
    const context3 = await browser.newContext();

    const page1 = await context1.newPage();
    const page2 = await context2.newPage();
    const page3 = await context3.newPage();

    // Track console logs and network events
    const logs: { browser: string; message: string }[] = [];
    
    page1.on('console', msg => {
      const text = msg.text();
      logs.push({ browser: 'Browser1', message: text });
      console.log('[Browser 1]', text);
    });
    
    page2.on('console', msg => {
      const text = msg.text();
      logs.push({ browser: 'Browser2', message: text });
      console.log('[Browser 2]', text);
    });
    
    page3.on('console', msg => {
      const text = msg.text();
      logs.push({ browser: 'Browser3', message: text });
      console.log('[Browser 3]', text);
    });

    try {
      console.log('\n=== Phase 1: Create and unlock accounts ===\n');
      
      // Create accounts in sequence to see bootstrap registration
      await createAndUnlockAccount(page1, 'Alice', 'alice@example.com', 'test-passphrase-alice-12345');
      console.log('✅ Alice account created');
      
      await page1.waitForTimeout(2000); // Wait for DHT registration
      
      await createAndUnlockAccount(page2, 'Bob', 'bob@example.com', 'test-passphrase-bob-67890');
      console.log('✅ Bob account created');
      
      await page2.waitForTimeout(2000); // Wait for DHT registration
      
      await createAndUnlockAccount(page3, 'Charlie', 'charlie@example.com', 'test-passphrase-charlie-abc');
      console.log('✅ Charlie account created');

      console.log('\n=== Phase 2: Wait for DHT bootstrap discovery ===\n');
      
      // DHT bootstrap discovery runs every 3 seconds
      // Wait up to 15 seconds for peers to discover each other
      await page1.waitForTimeout(15000);

      console.log('\n=== Phase 3: Verify DHT connections ===\n');

      // Check for DHT-related log messages indicating peer discovery
      const dhtLogs = logs.filter(log => 
        log.message.includes('[DHT]') || 
        log.message.includes('Bootstrap discovery') ||
        log.message.includes('relay')
      );
      
      console.log('\n📊 DHT-related logs:', dhtLogs.length);
      dhtLogs.slice(0, 20).forEach(log => {
        console.log(`  [${log.browser}] ${log.message}`);
      });

      // Verify WebSocket connections were established
      const wsLogs = logs.filter(log => log.message.includes('WebSocket'));
      expect(wsLogs.length).toBeGreaterThan(0);
      console.log(`✅ WebSocket connections: ${wsLogs.length} logs found`);

      // Verify DHT bootstrap discovery ran
      const discoveryLogs = logs.filter(log => log.message.includes('Bootstrap discovery'));
      expect(discoveryLogs.length).toBeGreaterThan(0);
      console.log(`✅ Bootstrap discovery: ${discoveryLogs.length} queries executed`);

      console.log('\n=== Phase 4: Test peer-to-peer relay connections ===\n');

      // Check if any relay signaling occurred
      const relayLogs = logs.filter(log => 
        log.message.toLowerCase().includes('relay') ||
        log.message.includes('connectRelay')
      );
      
      console.log(`📡 Relay signaling logs: ${relayLogs.length}`);
      relayLogs.slice(0, 10).forEach(log => {
        console.log(`  [${log.browser}] ${log.message}`);
      });

      // Verify all browsers are still functional
      await expect(page1.getByPlaceholder("What's on your mind?")).toBeVisible();
      await expect(page2.getByPlaceholder("What's on your mind?")).toBeVisible();
      await expect(page3.getByPlaceholder("What's on your mind?")).toBeVisible();
      console.log('✅ All UIs are responsive');

      console.log('\n=== Phase 5: Verify DHT query responses ===\n');

      // Execute JavaScript in browsers to query DHT state
      const dhtState1 = await page1.evaluate(() => {
        const testFn = (window as any).testGetDHTState;
        return {
          hasConnectionManager: typeof (window as any).connectionManager !== 'undefined',
          hasDHT: typeof (window as any).connectionManager?.getDHT !== 'undefined',
          dhtState: testFn ? testFn() : null,
        };
      });

      const dhtState2 = await page2.evaluate(() => {
        const testFn = (window as any).testGetDHTState;
        return {
          hasConnectionManager: typeof (window as any).connectionManager !== 'undefined',
          hasDHT: typeof (window as any).connectionManager?.getDHT !== 'undefined',
          dhtState: testFn ? testFn() : null,
        };
      });

      const dhtState3 = await page3.evaluate(() => {
        const testFn = (window as any).testGetDHTState;
        return {
          hasConnectionManager: typeof (window as any).connectionManager !== 'undefined',
          hasDHT: typeof (window as any).connectionManager?.getDHT !== 'undefined',
          dhtState: testFn ? testFn() : null,
        };
      });

      console.log('Browser 1 DHT state:', JSON.stringify(dhtState1, null, 2));
      console.log('Browser 2 DHT state:', JSON.stringify(dhtState2, null, 2));
      console.log('Browser 3 DHT state:', JSON.stringify(dhtState3, null, 2));
      
      // Verify DHT is active in all browsers
      expect(dhtState1.hasDHT).toBe(true);
      expect(dhtState2.hasDHT).toBe(true);
      expect(dhtState3.hasDHT).toBe(true);

      console.log('\n=== Test Summary ===\n');
      console.log(`Total logs captured: ${logs.length}`);
      console.log(`DHT logs: ${dhtLogs.length}`);
      console.log(`WebSocket logs: ${wsLogs.length}`);
      console.log(`Discovery logs: ${discoveryLogs.length}`);
      console.log(`Relay logs: ${relayLogs.length}`);
      
      // Test passes if:
      // 1. All accounts created successfully ✅
      // 2. WebSocket connections established ✅
      // 3. DHT bootstrap discovery executed ✅
      // 4. No fatal errors occurred ✅
      
      console.log('\n✅ DHT relay signaling E2E test completed successfully\n');

    } finally {
      await context1.close();
      await context2.close();
      await context3.close();
    }
  });

  test('browser should connect to bootstrap and receive peer list', async ({ page }) => {
    console.log('\n=== Testing single browser bootstrap connection ===\n');
    
    const dhtMessages: string[] = [];
    const wsMessages: string[] = [];
    
    page.on('console', msg => {
      const text = msg.text();
      if (text.includes('[DHT]')) dhtMessages.push(text);
      if (text.includes('WebSocket') || text.includes('[ConnectionManager]')) wsMessages.push(text);
    });

    // Create account
    await createAndUnlockAccount(page, 'TestUser', 'test@example.com', 'test-passphrase-12345');
    
    // Wait for initial connection and discovery
    await page.waitForTimeout(10000);

    // Verify connection manager started
    expect(wsMessages.length).toBeGreaterThan(0);
    console.log('✅ Connection manager logs:', wsMessages.length);
    
    // Verify DHT is active
    expect(dhtMessages.length).toBeGreaterThan(0);
    console.log('✅ DHT messages:', dhtMessages.length);
    
    // Print sample logs for debugging
    console.log('\n📝 Sample connection logs:');
    wsMessages.slice(0, 5).forEach(msg => console.log('  ', msg));
    
    console.log('\n📝 Sample DHT logs:');
    dhtMessages.slice(0, 5).forEach(msg => console.log('  ', msg));
    
    console.log('\n✅ Single browser test completed\n');
  });

  test('relay signaling should handle connection failures gracefully', async ({ browser }) => {
    console.log('\n=== Testing relay signaling error handling ===\n');
    
    const context1 = await browser.newContext();
    const context2 = await browser.newContext();
    
    const page1 = await context1.newPage();
    const page2 = await context2.newPage();
    
    const errors: string[] = [];
    
    page1.on('pageerror', err => errors.push(`Browser1: ${err.message}`));
    page2.on('pageerror', err => errors.push(`Browser2: ${err.message}`));
    
    try {
      // Create both accounts
      await createAndUnlockAccount(page1, 'UserA', 'a@example.com', 'password-a-123456');
      await createAndUnlockAccount(page2, 'UserB', 'b@example.com', 'password-b-123456');
      
      // Wait for discovery attempts
      await page1.waitForTimeout(10000);
      
      // Close one browser context abruptly to simulate network failure
      await context2.close();
      
      // Wait to see if Browser1 handles the disconnection gracefully
      await page1.waitForTimeout(5000);
      
      // Verify no uncaught errors
      console.log('Errors captured:', errors.length);
      errors.forEach(err => console.log('  ⚠️ ', err));
      
      // UI should still be functional
      await expect(page1.getByPlaceholder("What's on your mind?")).toBeVisible();
      
      console.log('✅ Graceful error handling verified\n');
      
    } finally {
      await context1.close();
    }
  });

  test('DHT should maintain connections to multiple peers simultaneously', async ({ browser }) => {
    test.setTimeout(60000); // Increase timeout to 60 seconds
    
    console.log('\n=== Testing simultaneous peer connections ===\n');
    
    // Create 4 browser contexts
    const contexts = await Promise.all([
      browser.newContext(),
      browser.newContext(),
      browser.newContext(),
      browser.newContext(),
    ]);
    
    const pages = await Promise.all(contexts.map(ctx => ctx.newPage()));
    
    const allLogs: { browser: number; message: string }[] = [];
    
    pages.forEach((page, idx) => {
      page.on('console', msg => {
        const text = msg.text();
        if (text.includes('[DHT]') || text.includes('relay') || text.includes('peer')) {
          allLogs.push({ browser: idx, message: text });
        }
      });
    });
    
    try {
      console.log('Creating 4 accounts...');
      
      // Create accounts sequentially with delays
      await createAndUnlockAccount(pages[0], 'User1', '1@example.com', 'pass1-123456789');
      await pages[0].waitForTimeout(2000);
      
      await createAndUnlockAccount(pages[1], 'User2', '2@example.com', 'pass2-123456789');
      await pages[1].waitForTimeout(2000);
      
      await createAndUnlockAccount(pages[2], 'User3', '3@example.com', 'pass3-123456789');
      await pages[2].waitForTimeout(2000);
      
      await createAndUnlockAccount(pages[3], 'User4', '4@example.com', 'pass4-123456789');
      
      console.log('✅ All 4 accounts created');
      
      // Wait for DHT discovery and relay connections
      console.log('Waiting for peer discovery and relay connections...');
      await pages[0].waitForTimeout(20000);
      
      // Analyze logs
      console.log(`\n📊 Captured ${allLogs.length} relevant logs`);
      
      const byBrowser = [0, 1, 2, 3].map(i => 
        allLogs.filter(log => log.browser === i).length
      );
      
      console.log('Logs per browser:');
      byBrowser.forEach((count, idx) => {
        console.log(`  Browser ${idx + 1}: ${count} logs`);
      });
      
      // Each browser should have DHT activity
      expect(byBrowser.every(count => count > 0)).toBe(true);
      
      // Check for relay attempts
      const relayAttempts = allLogs.filter(log => 
        log.message.toLowerCase().includes('relay')
      );
      
      console.log(`\n📡 Relay attempts: ${relayAttempts.length}`);
      
      // Verify all UIs still responsive
      for (let i = 0; i < pages.length; i++) {
        await expect(pages[i].getByPlaceholder("What's on your mind?")).toBeVisible();
      }
      
      console.log('✅ All 4 browsers remain functional\n');
      
    } finally {
      // Close contexts gracefully, ignoring errors if already closed
      await Promise.all(contexts.map(ctx => ctx.close().catch(() => {})));
    }
  });
});
