/**
 * Storage Factory
 * 
 * Browser-only storage using IndexedDB.
 */

import type { Storage } from '../types/index.js';

export async function createStorage(): Promise<Storage> {
  // Browser environment only
  if (typeof indexedDB === 'undefined') {
    throw new Error('IndexedDB not available. This package requires a browser environment.');
  }
  
  const { BrowserStorage } = await import('./browser.js');
  return await BrowserStorage.create();
}

export { BrowserStorage } from './browser.js';
