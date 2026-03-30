/**
 * Simple filesystem storage for Node.js (bootstrap server only)
 * 
 * This is a minimal implementation that allows Account persistence
 * in Node.js without relying on IndexedDB.
 */

import { promises as fs } from 'fs';
import { join, dirname } from 'path';

export class NodeFSStorage {
  constructor(baseDir = '.') {
    this.baseDir = baseDir;
  }

  async exists(path) {
    try {
      await fs.access(join(this.baseDir, path));
      return true;
    } catch {
      return false;
    }
  }

  async readFile(path) {
    const data = await fs.readFile(join(this.baseDir, path));
    return new Uint8Array(data);
  }

  async writeFile(path, data) {
    const fullPath = join(this.baseDir, path);
    await fs.mkdir(dirname(fullPath), { recursive: true });
    await fs.writeFile(fullPath, data);
  }

  async readText(path) {
    return await fs.readFile(join(this.baseDir, path), 'utf-8');
  }

  async writeText(path, text) {
    const fullPath = join(this.baseDir, path);
    await fs.mkdir(dirname(fullPath), { recursive: true });
    await fs.writeFile(fullPath, text, 'utf-8');
  }

  async readDir(path) {
    const fullPath = join(this.baseDir, path);
    try {
      const entries = await fs.readdir(fullPath, { withFileTypes: true });
      return entries.map(e => e.isDirectory() ? e.name + '/' : e.name);
    } catch (err) {
      if (err.code === 'ENOENT') return [];
      throw err;
    }
  }

  async mkdir(path) {
    await fs.mkdir(join(this.baseDir, path), { recursive: true });
  }

  async remove(path) {
    const fullPath = join(this.baseDir, path);
    try {
      const stat = await fs.stat(fullPath);
      if (stat.isDirectory()) {
        await fs.rm(fullPath, { recursive: true, force: true });
      } else {
        await fs.unlink(fullPath);
      }
    } catch (err) {
      if (err.code !== 'ENOENT') throw err;
    }
  }

  async stat(path) {
    const fullPath = join(this.baseDir, path);
    const stats = await fs.stat(fullPath);
    return {
      size: stats.size,
      isDirectory: stats.isDirectory(),
      modifiedTime: stats.mtimeMs,
    };
  }

  join(...parts) {
    return join(...parts);
  }
}
