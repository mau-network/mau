import { Account, File } from '@mau-network/mau';
import type { StatusPost } from '../types/index';
import { v4 as uuidv4 } from 'uuid';

/**
 * StatusStoreManager - Manages status posts according to Mau specification
 * 
 * Per spec (README.md):
 * - Each status is a separate .pgp file (not single JSON)
 * - Uses Schema.org SocialMediaPosting type
 * - Files are encrypted and signed automatically by File class
 */
export class StatusStoreManager {
  private account: Account;

  private constructor(account: Account) {
    this.account = account;
  }

  static async create(account: Account): Promise<StatusStoreManager> {
    return new StatusStoreManager(account);
  }

  /**
   * Create a new status post as a separate .pgp file
   * Following Schema.org SocialMediaPosting format
   */
  async addPost(content: string): Promise<StatusPost> {
    this.validateContent(content);

    const id = uuidv4();
    const timestamp = Date.now();
    const fileName = `status-${id}.pgp`;

    // Schema.org SocialMediaPosting structure
    const posting = {
      '@context': 'https://schema.org',
      '@type': 'SocialMediaPosting',
      '@id': id,
      'articleBody': content.trim(),
      'datePublished': new Date(timestamp).toISOString(),
      'author': {
        '@type': 'Person',
        'identifier': this.account.getFingerprint(),
      },
    };

    // Create file and write JSON (automatically encrypted and signed as .pgp)
    const file = await this.account.createFile(fileName);
    await file.writeJSON(posting);

    // Return local representation
    return {
      id,
      content: content.trim(),
      createdAt: timestamp,
      fileName,
    };
  }

  /**
   * List all status posts from individual .pgp files
   */
  async getPosts(limit = 20, offset = 0): Promise<StatusPost[]> {
    const files = await this.account.listFiles();
    const statusFiles = files.filter((f) => f.getName().startsWith('status-') && f.getName().endsWith('.pgp'));

    const posts: StatusPost[] = [];

    for (const file of statusFiles) {
      try {
        const data = await file.readJSON<{
          '@type': string;
          '@id': string;
          'articleBody': string;
          'datePublished': string;
        }>();

        if (data['@type'] === 'SocialMediaPosting') {
          posts.push({
            id: data['@id'],
            content: data['articleBody'],
            createdAt: new Date(data['datePublished']).getTime(),
            fileName: file.getName(),
          });
        }
      } catch (error) {
        console.error(`Failed to read status file ${file.getName()}:`, error);
      }
    }

    // Sort by newest first
    posts.sort((a, b) => b.createdAt - a.createdAt);

    return posts.slice(offset, offset + limit);
  }

  /**
   * Get a specific post by ID
   */
  async getPost(id: string): Promise<StatusPost | null> {
    const fileName = `status-${id}.pgp`;
    
    try {
      const file = await this.account.createFile(fileName);
      const data = await file.readJSON<{
        '@type': string;
        '@id': string;
        'articleBody': string;
        'datePublished': string;
      }>();

      if (data['@type'] === 'SocialMediaPosting') {
        return {
          id: data['@id'],
          content: data['articleBody'],
          createdAt: new Date(data['datePublished']).getTime(),
          fileName,
        };
      }
    } catch {
      return null;
    }

    return null;
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
}
