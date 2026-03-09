/**
 * IndexedDB Mocks for Testing
 */

import 'fake-indexeddb/auto';

export function setupIndexedDBMocks(): void {
  // fake-indexeddb automatically sets up global.indexedDB
  // Nothing additional needed
}

export function cleanupIndexedDBMocks(): void {
  // Clean up databases after tests
  if (typeof indexedDB !== 'undefined') {
    // Delete all test databases
    indexedDB.databases?.().then(dbs => {
      dbs.forEach(db => {
        if (db.name) {
          indexedDB.deleteDatabase(db.name);
        }
      });
    });
  }
}
