import { test as bunTest } from 'bun:test';
import 'fake-indexeddb/auto';

// Setup fake-indexeddb for all tests
global.indexedDB = indexedDB;
global.IDBKeyRange = IDBKeyRange;
