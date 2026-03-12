/**
 * File - Content File Management
 * 
 * Handles reading, writing, signing, encrypting, and versioning content files.
 */

import type { Storage, Fingerprint, MauFile } from './types/index.js';
import { InvalidFileNameError } from './types/index.js';
import type { Account } from './account.js';
import {
  signAndEncrypt,
  decryptAndVerify,
  sha256,
  validateFileName,
} from './crypto/index.js';

/**
 * File represents an encrypted content file in the Mau filesystem
 * 
 * Provides methods for reading/writing JSON and text data with automatic
 * encryption, versioning, and checksum generation. All file operations
 * are signed with the account's private key.
 * 
 * @example
 * ```typescript
 * const file = File.create(account, storage, 'posts/hello.json');
 * await file.writeJSON({ '@type': 'SocialMediaPosting', headline: 'Hello!' });
 * const data = await file.readJSON();
 * ```
 */
export class File {
  private account: Account;
  private storage: Storage;
  private filePath: string;
  private isVersion: boolean;

  constructor(account: Account, storage: Storage, filePath: string, isVersion = false) {
    this.account = account;
    this.storage = storage;
    this.filePath = filePath;
    this.isVersion = isVersion;
  }

  /**
   * Get file name (basename)
   */
  getName(): string {
    const parts = this.filePath.split('/');
    return parts[parts.length - 1];
  }

  /**
   * Get file path
   */
  getPath(): string {
    return this.filePath;
  }

  /**
   * Check if this is a version file
   */
  isVersionFile(): boolean {
    return this.isVersion;
  }

  /**
   * Get list of versions for this file
   */
  async getVersions(): Promise<File[]> {
    if (this.isVersion) {
      return []; // Versions don't have versions
    }

    const versionDir = `${this.filePath}.versions`;
    if (!(await this.storage.exists(versionDir))) {
      return [];
    }

    const entries = await this.storage.readDir(versionDir);
    const versions: File[] = [];

    for (const entry of entries) {
      const versionPath = this.storage.join(versionDir, entry);
      const stats = await this.storage.stat(versionPath);
      if (!stats.isDirectory) {
        versions.push(new File(this.account, this.storage, versionPath, true));
      }
    }

    return versions;
  }

  /**
   * Read and decrypt file content
   */
  async read(): Promise<Uint8Array> {
    const encryptedData = await this.storage.readFile(this.filePath);
    const armoredMessage = new TextDecoder().decode(encryptedData);

    // Get verification keys (self + friends)
    const verificationKeys = this.account.getAllPublicKeys();

    const { data, verified } = await decryptAndVerify(
      armoredMessage,
      this.account.getPrivateKey(),
      verificationKeys
    );

    if (!verified) {
      console.warn(`Warning: File ${this.filePath} signature verification failed`);
    }

    return data;
  }

  /**
   * Read file as text
   */
  async readText(): Promise<string> {
    const data = await this.read();
    return new TextDecoder().decode(data);
  }

  /**
   * Read file as JSON
   */
  async readJSON<T = unknown>(): Promise<T> {
    const text = await this.readText();
    return JSON.parse(text);
  }

  /**
   * Write, sign, and encrypt file content
   */
  async write(data: Uint8Array | string): Promise<void> {
    // Archive current version if file exists
    if (await this.storage.exists(this.filePath)) {
      await this.archiveCurrentVersion();
    }

    // Get encryption keys (self + friends)
    const encryptionKeys = this.account.getAllPublicKeys();

    // Sign and encrypt
    const armoredMessage = await signAndEncrypt(
      data,
      this.account.getPrivateKey(),
      encryptionKeys
    );

    // Write to disk
    await this.storage.writeText(this.filePath, armoredMessage);
  }

  /**
   * Write text content
   */
  async writeText(text: string): Promise<void> {
    await this.write(text);
  }

  /**
   * Write JSON content
   */
  async writeJSON(obj: unknown): Promise<void> {
    const text = JSON.stringify(obj, null, 2);
    await this.write(text);
  }

  /**
   * Archive current version before overwriting
   */
  private async archiveCurrentVersion(): Promise<void> {
    if (this.isVersion) {
      return; // Don't archive versions
    }

    const currentData = await this.storage.readFile(this.filePath);
    const checksum = await sha256(currentData);

    const versionDir = `${this.filePath}.versions`;
    await this.storage.mkdir(versionDir);

    const versionPath = this.storage.join(versionDir, checksum);
    if (!(await this.storage.exists(versionPath))) {
      await this.storage.writeFile(versionPath, currentData);
    }
  }

  /**
   * Delete file
   */
  async delete(): Promise<void> {
    if (await this.storage.exists(this.filePath)) {
      await this.storage.remove(this.filePath);
    }

    // Delete versions if this is not a version
    if (!this.isVersion) {
      const versionDir = `${this.filePath}.versions`;
      if (await this.storage.exists(versionDir)) {
        await this.storage.remove(versionDir);
      }
    }
  }

  /**
   * Get file checksum
   */
  async getChecksum(): Promise<string> {
    const data = await this.storage.readFile(this.filePath);
    return await sha256(data);
  }

  /**
   * Get file size
   */
  async getSize(): Promise<number> {
    const stats = await this.storage.stat(this.filePath);
    return stats.size;
  }

  /**
   * Convert to MauFile interface
   */
  toMauFile(): MauFile {
    return {
      path: this.filePath,
      name: this.getName(),
      isVersion: this.isVersion,
    };
  }

  /**
   * Create a File instance from a path
   */
  static create(
    account: Account,
    storage: Storage,
    fileName: string,
    isVersion = false
  ): File {
    if (!validateFileName(fileName)) {
      throw new InvalidFileNameError('contains invalid characters or path separators');
    }

    const contentDir = account.getContentDir();
    const filePath = storage.join(contentDir, fileName);

    return new File(account, storage, filePath, isVersion);
  }

  /**
   * List all files in account's content directory
   */
  static async list(account: Account, storage: Storage): Promise<File[]> {
    const contentDir = account.getContentDir();
    if (!(await storage.exists(contentDir))) {
      return [];
    }

    const entries = await storage.readDir(contentDir);
    const files: File[] = [];

    for (const entry of entries) {
      const filePath = storage.join(contentDir, entry);
      const stats = await storage.stat(filePath);
      
      // Skip directories and version directories
      if (!stats.isDirectory && !entry.endsWith('.versions')) {
        files.push(new File(account, storage, filePath, false));
      }
    }

    return files;
  }

  /**
   * List files for a specific friend
   */
  static async listFriend(
    account: Account,
    storage: Storage,
    fingerprint: Fingerprint
  ): Promise<File[]> {
    const contentDir = account.getFriendContentDir(fingerprint);
    if (!(await storage.exists(contentDir))) {
      return [];
    }

    const entries = await storage.readDir(contentDir);
    const files: File[] = [];

    for (const entry of entries) {
      const filePath = storage.join(contentDir, entry);
      const stats = await storage.stat(filePath);
      
      if (!stats.isDirectory && !entry.endsWith('.versions')) {
        files.push(new File(account, storage, filePath, false));
      }
    }

    return files;
  }
}
