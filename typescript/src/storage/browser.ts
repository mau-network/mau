/**
 * Browser Storage Implementation (localStorage + IndexedDB)
 * 
 * Provides browser-based storage using localStorage for metadata and
 * IndexedDB for binary file data.
 */

import type { Storage } from '../types/index.js';

const DB_NAME = 'mau-storage';
const DB_VERSION = 1;
const STORE_NAME = 'files';

interface FileEntry {
  path: string;
  data: Uint8Array;
  size: number;
  isDirectory: boolean;
}

export class BrowserStorage implements Storage {
  private db: IDBDatabase | null = null;

  constructor() {
    this.initDB();
  }

  private async initDB(): Promise<void> {
    if (typeof indexedDB === 'undefined') {
      throw new Error('IndexedDB not available in this environment');
    }

    return new Promise((resolve, reject) => {
      const request = indexedDB.open(DB_NAME, DB_VERSION);

      request.onerror = () => reject(request.error);
      request.onsuccess = () => {
        this.db = request.result;
        resolve();
      };

      request.onupgradeneeded = (event) => {
        const db = (event.target as IDBOpenDBRequest).result;
        if (!db.objectStoreNames.contains(STORE_NAME)) {
          db.createObjectStore(STORE_NAME, { keyPath: 'path' });
        }
      };
    });
  }

  private async ensureDB(): Promise<IDBDatabase> {
    if (!this.db) {
      await this.initDB();
    }
    if (!this.db) {
      throw new Error('Failed to initialize database');
    }
    return this.db;
  }

  private async getEntry(path: string): Promise<FileEntry | null> {
    const db = await this.ensureDB();
    return new Promise((resolve, reject) => {
      const transaction = db.transaction([STORE_NAME], 'readonly');
      const store = transaction.objectStore(STORE_NAME);
      const request = store.get(path);

      request.onsuccess = () => resolve(request.result || null);
      request.onerror = () => reject(request.error);
    });
  }

  private async putEntry(entry: FileEntry): Promise<void> {
    const db = await this.ensureDB();
    return new Promise((resolve, reject) => {
      const transaction = db.transaction([STORE_NAME], 'readwrite');
      const store = transaction.objectStore(STORE_NAME);
      const request = store.put(entry);

      request.onsuccess = () => resolve();
      request.onerror = () => reject(request.error);
    });
  }

  private async deleteEntry(path: string): Promise<void> {
    const db = await this.ensureDB();
    return new Promise((resolve, reject) => {
      const transaction = db.transaction([STORE_NAME], 'readwrite');
      const store = transaction.objectStore(STORE_NAME);
      const request = store.delete(path);

      request.onsuccess = () => resolve();
      request.onerror = () => reject(request.error);
    });
  }

  private async getAllKeys(): Promise<string[]> {
    const db = await this.ensureDB();
    return new Promise((resolve, reject) => {
      const transaction = db.transaction([STORE_NAME], 'readonly');
      const store = transaction.objectStore(STORE_NAME);
      const request = store.getAllKeys();

      request.onsuccess = () => resolve(request.result as string[]);
      request.onerror = () => reject(request.error);
    });
  }

  async exists(path: string): Promise<boolean> {
    const entry = await this.getEntry(path);
    return entry !== null;
  }

  async readFile(path: string): Promise<Uint8Array> {
    const entry = await this.getEntry(path);
    if (!entry) {
      throw new Error(`File not found: ${path}`);
    }
    if (entry.isDirectory) {
      throw new Error(`Path is a directory: ${path}`);
    }
    return entry.data;
  }

  async writeFile(path: string, data: Uint8Array): Promise<void> {
    // Create parent directories if needed
    const parts = path.split('/');
    for (let i = 1; i < parts.length; i++) {
      const dirPath = parts.slice(0, i).join('/');
      if (dirPath && !(await this.exists(dirPath))) {
        await this.mkdir(dirPath);
      }
    }

    await this.putEntry({
      path,
      data,
      size: data.length,
      isDirectory: false,
    });
  }

  async readText(path: string): Promise<string> {
    const data = await this.readFile(path);
    return new TextDecoder().decode(data);
  }

  async writeText(path: string, text: string): Promise<void> {
    const data = new TextEncoder().encode(text);
    await this.writeFile(path, data);
  }

  async readDir(dirPath: string): Promise<string[]> {
    const allKeys = await this.getAllKeys();
    const prefix = dirPath.endsWith('/') ? dirPath : dirPath + '/';
    
    const entries = new Set<string>();
    for (const key of allKeys) {
      if (key.startsWith(prefix)) {
        const rest = key.slice(prefix.length);
        const nextSlash = rest.indexOf('/');
        if (nextSlash === -1) {
          entries.add(rest);
        } else {
          entries.add(rest.slice(0, nextSlash));
        }
      }
    }
    
    return Array.from(entries);
  }

  async mkdir(dirPath: string): Promise<void> {
    if (!(await this.exists(dirPath))) {
      await this.putEntry({
        path: dirPath,
        data: new Uint8Array(0),
        size: 0,
        isDirectory: true,
      });
    }
  }

  async remove(path: string): Promise<void> {
    const allKeys = await this.getAllKeys();
    const prefix = path.endsWith('/') ? path : path + '/';
    
    // Delete the path itself and all children
    const toDelete = allKeys.filter((key) => key === path || key.startsWith(prefix));
    
    for (const key of toDelete) {
      await this.deleteEntry(key);
    }
  }

  async stat(path: string): Promise<{ size: number; isDirectory: boolean }> {
    const entry = await this.getEntry(path);
    if (!entry) {
      throw new Error(`Path not found: ${path}`);
    }
    return {
      size: entry.size,
      isDirectory: entry.isDirectory,
    };
  }

  join(...parts: string[]): string {
    return parts
      .join('/')
      .replace(/\/+/g, '/')
      .replace(/^\//, '');
  }
}
