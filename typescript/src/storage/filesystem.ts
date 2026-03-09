/**
 * Filesystem Storage Implementation (Node.js)
 * 
 * Provides filesystem-based storage for Node.js environments.
 */

import * as fs from 'fs/promises';
import * as path from 'path';
import type { Storage } from '../types/index.js';

export class FilesystemStorage implements Storage {
  async exists(filePath: string): Promise<boolean> {
    try {
      await fs.access(filePath);
      return true;
    } catch {
      return false;
    }
  }

  async readFile(filePath: string): Promise<Uint8Array> {
    const buffer = await fs.readFile(filePath);
    return new Uint8Array(buffer);
  }

  async writeFile(filePath: string, data: Uint8Array): Promise<void> {
    await fs.writeFile(filePath, data, { mode: 0o600 });
  }

  async readText(filePath: string): Promise<string> {
    return await fs.readFile(filePath, 'utf-8');
  }

  async writeText(filePath: string, text: string): Promise<void> {
    await fs.writeFile(filePath, text, { encoding: 'utf-8', mode: 0o600 });
  }

  async readDir(dirPath: string): Promise<string[]> {
    return await fs.readdir(dirPath);
  }

  async mkdir(dirPath: string): Promise<void> {
    await fs.mkdir(dirPath, { recursive: true, mode: 0o700 });
  }

  async remove(filePath: string): Promise<void> {
    const stats = await this.stat(filePath);
    if (stats.isDirectory) {
      await fs.rm(filePath, { recursive: true, force: true });
    } else {
      await fs.unlink(filePath);
    }
  }

  async stat(filePath: string): Promise<{ size: number; isDirectory: boolean }> {
    const stats = await fs.stat(filePath);
    return {
      size: stats.size,
      isDirectory: stats.isDirectory(),
    };
  }

  join(...parts: string[]): string {
    return path.join(...parts);
  }
}
