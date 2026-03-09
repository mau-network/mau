/**
 * Browser tests for WebRTC functionality
 * Run with: npx playwright test
 */

import { test, expect } from '@playwright/test';

test.describe('WebRTC in Browser', () => {
  test.beforeEach(async ({ page }) => {
    // Load the library
    await page.goto('http://localhost:8888');
  });

  test('should create WebRTC connection', async ({ page }) => {
    const result = await page.evaluate(async () => {
      // Test that RTCPeerConnection is available
      const pc = new RTCPeerConnection();
      const hasDataChannel = typeof pc.createDataChannel === 'function';
      pc.close();
      return hasDataChannel;
    });

    expect(result).toBe(true);
  });

  test('should create data channel', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const pc = new RTCPeerConnection();
      const channel = pc.createDataChannel('test');
      const hasLabel = channel.label === 'test';
      pc.close();
      return hasLabel;
    });

    expect(result).toBe(true);
  });

  test('should handle WebRTC offer/answer', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const pc1 = new RTCPeerConnection();
      const pc2 = new RTCPeerConnection();

      try {
        const offer = await pc1.createOffer();
        await pc1.setLocalDescription(offer);
        
        await pc2.setRemoteDescription(pc1.localDescription!);
        const answer = await pc2.createAnswer();
        await pc2.setLocalDescription(answer);
        
        return true;
      } catch (error) {
        return false;
      } finally {
        pc1.close();
        pc2.close();
      }
    });

    expect(result).toBe(true);
  });
});

test.describe('IndexedDB in Browser', () => {
  test('should access IndexedDB', async ({ page }) => {
    const result = await page.evaluate(async () => {
      return typeof indexedDB !== 'undefined';
    });

    expect(result).toBe(true);
  });

  test('should create database', async ({ page }) => {
    const result = await page.evaluate(async () => {
      return new Promise((resolve) => {
        const request = indexedDB.open('test-db', 1);
        
        request.onsuccess = () => {
          const db = request.result;
          db.close();
          indexedDB.deleteDatabase('test-db');
          resolve(true);
        };
        
        request.onerror = () => resolve(false);
        
        request.onupgradeneeded = () => {
          const db = request.result;
          db.createObjectStore('test-store');
        };
      });
    });

    expect(result).toBe(true);
  });

  test('should write and read data', async ({ page }) => {
    const result = await page.evaluate(async () => {
      return new Promise((resolve) => {
        const request = indexedDB.open('test-db-2', 1);
        
        request.onupgradeneeded = () => {
          const db = request.result;
          db.createObjectStore('store');
        };
        
        request.onsuccess = () => {
          const db = request.result;
          const tx = db.transaction('store', 'readwrite');
          const store = tx.objectStore('store');
          
          store.put('test-value', 'test-key');
          
          tx.oncomplete = () => {
            const tx2 = db.transaction('store', 'readonly');
            const store2 = tx2.objectStore('store');
            const getRequest = store2.get('test-key');
            
            getRequest.onsuccess = () => {
              db.close();
              indexedDB.deleteDatabase('test-db-2');
              resolve(getRequest.result === 'test-value');
            };
          };
        };
        
        request.onerror = () => resolve(false);
      });
    });

    expect(result).toBe(true);
  });
});
