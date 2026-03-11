/**
 * Storage Factory
 * 
 * Auto-detects environment and provides appropriate storage implementation.
 */

import type { Storage } from '../types/index.js';

export async function createStorage(): Promise<Storage> {
  // Check if we're in Node.js environment
  if (typeof process !== 'undefined' && process.versions && process.versions.node) {
    const { FilesystemStorage } = await import('./filesystem.js');
    return new FilesystemStorage();
  }
  
  // Check if we're in browser environment
  if (typeof window !== 'undefined' && typeof indexedDB !== 'undefined') {
    const { BrowserStorage } = await import('./browser.js');
    return await BrowserStorage.create();
  }
  
  throw new Error('Unable to determine storage backend for this environment');
}

export { FilesystemStorage } from './filesystem.js';
export { BrowserStorage } from './browser.js';
