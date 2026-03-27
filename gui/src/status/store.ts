import { Account, File } from '@mau-network/mau';
import type { StatusPost, StatusStore } from '../types/index';
import { v4 as uuidv4 } from 'uuid';

const STATUS_STORE_FILE = 'status-store.json';
const SCHEMA_VERSION = 1;

export class StatusStoreManager {
  private file: File;
  private store: StatusStore | null = null;

  private constructor(file: File) {
    this.file = file;
  }

  static async create(account: Account): Promise<StatusStoreManager> {
    const file = await account.createFile(STATUS_STORE_FILE);
    return new StatusStoreManager(file);
  }

  async load(): Promise<void> {
    try {
      const data = await this.file.readJSON<StatusStore>();
      this.store = data;
    } catch {
      this.store = {
        posts: [],
        version: SCHEMA_VERSION,
      };
    }
  }

  async addPost(content: string): Promise<StatusPost> {
    this.validateContent(content);
    await this.ensureLoaded();

    const post: StatusPost = {
      id: uuidv4(),
      content: content.trim(),
      createdAt: Date.now(),
      signature: await this.signContent(content),
    };

    this.store!.posts.push(post);
    await this.save();

    return post;
  }

  async getPosts(limit = 20, offset = 0): Promise<StatusPost[]> {
    await this.ensureLoaded();

    const sorted = [...this.store!.posts].sort((a, b) => b.createdAt - a.createdAt);

    return sorted.slice(offset, offset + limit);
  }

  async getPost(id: string): Promise<StatusPost | null> {
    await this.ensureLoaded();
    return this.store!.posts.find((p) => p.id === id) ?? null;
  }

  private async ensureLoaded(): Promise<void> {
    if (!this.store) {
      await this.load();
    }
  }

  private async save(): Promise<void> {
    await this.file.writeJSON(this.store!);
  }

  private validateContent(content: string): void {
    const trimmed = content.trim();
    if (trimmed.length < 1) {
      throw new Error('Status content cannot be empty');
    }
    if (trimmed.length > 500) {
      throw new Error('Status content cannot exceed 500 characters');
    }
  }

  private async signContent(content: string): Promise<string> {
    const encoder = new TextEncoder();
    const data = encoder.encode(content);

    return Array.from(data)
      .map((b) => b.toString(16).padStart(2, '0'))
      .join('');
  }
}
