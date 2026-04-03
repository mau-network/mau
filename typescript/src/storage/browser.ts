/**
 * Browser Storage Implementation (IndexedDB via idb)
 */

import { openDB, IDBPDatabase, DBSchema } from 'idb';
import type { Storage } from '../types/index.js';

const DB_NAME = 'mau-storage';
const DB_VERSION = 1;
const STORE_NAME = 'files';

interface FileEntry {
  path: string;
  data: Uint8Array;
  size: number;
  isDirectory: boolean;
  modifiedTime?: number;
}

interface MauDB extends DBSchema {
  files: {
    key: string;
    value: FileEntry;
  };
}

export class BrowserStorage implements Storage {
  private db: IDBPDatabase<MauDB>;

  private constructor(db: IDBPDatabase<MauDB>) {
    this.db = db;
  }

  static async create(): Promise<BrowserStorage> {
    const db = await openDB<MauDB>(DB_NAME, DB_VERSION, {
      upgrade(db: IDBPDatabase<MauDB>): void {
        if (!db.objectStoreNames.contains(STORE_NAME)) {
          db.createObjectStore(STORE_NAME, { keyPath: 'path' });
        }
      },
    });
    return new BrowserStorage(db);
  }

  async exists(path: string): Promise<boolean> {
    return (await this.db.getKey(STORE_NAME, path)) !== undefined;
  }

  async readFile(path: string): Promise<Uint8Array> {
    const entry = await this.db.get(STORE_NAME, path);
    if (!entry) {throw new Error(`File not found: ${path}`);}
    if (entry.isDirectory) {throw new Error(`Path is a directory: ${path}`);}
    return entry.data;
  }

  async writeFile(path: string, data: Uint8Array): Promise<void> {
    const parts = path.split('/');
    for (let i = 1; i < parts.length; i++) {
      const dirPath = parts.slice(0, i).join('/');
      if (dirPath && !(await this.exists(dirPath))) {
        await this.mkdir(dirPath);
      }
    }
    await this.db.put(STORE_NAME, {
      path,
      data,
      size: data.length,
      isDirectory: false,
      modifiedTime: Date.now(),
    });
  }

  async readText(path: string): Promise<string> {
    return new TextDecoder().decode(await this.readFile(path));
  }

  async writeText(path: string, text: string): Promise<void> {
    await this.writeFile(path, new TextEncoder().encode(text));
  }

  async readDir(dirPath: string): Promise<string[]> {
    const entries = new Set<string>();
    if (!dirPath || dirPath === '/') {
      const all = await this.db.getAllKeys(STORE_NAME);
      for (const key of all) {
        const slashIndex = key.indexOf('/');
        entries.add(slashIndex === -1 ? key : key.slice(0, slashIndex));
      }
    } else {
      const prefix = dirPath.endsWith('/') ? dirPath : dirPath + '/';
      const range = IDBKeyRange.bound(prefix, prefix + '\uffff');
      const keys = await this.db.getAllKeys(STORE_NAME, range);
      for (const key of keys) {
        const rest = key.slice(prefix.length);
        const nextSlash = rest.indexOf('/');
        entries.add(nextSlash === -1 ? rest : rest.slice(0, nextSlash));
      }
    }
    return Array.from(entries);
  }

  async mkdir(dirPath: string): Promise<void> {
    if (!(await this.exists(dirPath))) {
      await this.db.put(STORE_NAME, {
        path: dirPath,
        data: new Uint8Array(0),
        size: 0,
        isDirectory: true,
        modifiedTime: Date.now(),
      });
    }
  }

  async remove(path: string): Promise<void> {
    const tx = this.db.transaction(STORE_NAME, 'readwrite');
    const store = tx.objectStore(STORE_NAME);
    await store.delete(path);
    const prefix = path.endsWith('/') ? path : path + '/';
    const range = IDBKeyRange.bound(prefix, prefix + '\uffff');
    let cursor = await store.openCursor(range);
    while (cursor) {
      await cursor.delete();
      cursor = await cursor.continue();
    }
    await tx.done;
  }

  async stat(path: string): Promise<{ size: number; isDirectory: boolean; modifiedTime?: number }> {
    const entry = await this.db.get(STORE_NAME, path);
    if (!entry) {throw new Error(`Path not found: ${path}`);}
    return { size: entry.size, isDirectory: entry.isDirectory, modifiedTime: entry.modifiedTime };
  }

  join(...parts: string[]): string {
    return parts.join('/').replace(/\/+/g, '/').replace(/^\//, '');
  }

  // TODO(cleanup): Remove unused batch operations or document their purpose
  // writeBatch() and readBatch() are not called anywhere in the codebase.
  // Verification: grep -r "writeBatch\|readBatch" typescript/src/ returns only these definitions.
  //
  // Options:
  // 1. REMOVE if not planned for future use
  // 2. DOCUMENT with JSDoc explaining future bulk sync optimization plans
  // 3. ADD TESTS to ensure they work correctly if keeping
  //
  // Priority: LOW - Cleanup/maintenance
  // Impact: Dead code increases maintenance burden (26 LOC unused)

  async writeBatch(files: Array<{ path: string; data: Uint8Array }>): Promise<void> {
    const tx = this.db.transaction(STORE_NAME, 'readwrite');
    const store = tx.objectStore(STORE_NAME);
    const timestamp = Date.now();
    await Promise.all(
      files.map((file) =>
        store.put({
          path: file.path,
          data: file.data,
          size: file.data.length,
          isDirectory: false,
          modifiedTime: timestamp,
        })
      )
    );
    await tx.done;
  }

  async readBatch(paths: string[]): Promise<Map<string, Uint8Array | null>> {
    const tx = this.db.transaction(STORE_NAME, 'readonly');
    const store = tx.objectStore(STORE_NAME);
    const results = new Map<string, Uint8Array | null>();
    await Promise.all(
      paths.map(async (path) => {
        const entry = await store.get(path);
        results.set(path, entry?.data ?? null);
      })
    );
    await tx.done;
    return results;
  }
}
