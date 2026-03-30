import 'fake-indexeddb/auto';
import { test, expect, describe, beforeEach } from 'bun:test';
import { StatusStoreManager } from './store';
import { Account, BrowserStorage } from '@mau-network/mau';

describe('StatusStoreManager', () => {
  let account: Account;
  let store: StatusStoreManager;

  beforeEach(async () => {
    const storage = await BrowserStorage.create();
    const accountDir = `test-account-${Date.now()}-${Math.random()}`;

    account = await Account.create(storage, accountDir, {
      name: 'Test User',
      email: 'test@example.com',
      passphrase: 'securepassphrase123',
    });

    store = await StatusStoreManager.create(account);
  });

  describe('addPost', () => {
    test('should add valid post', async () => {
      const post = await store.addPost('Hello, world!');

      expect(post.id).toBeDefined();
      expect(post.content).toBe('Hello, world!');
      expect(post.createdAt).toBeGreaterThan(0);
      expect(post.fileName).toBeDefined();
      expect(post.fileName).toMatch(/^status-.*\.pgp$/);
    });

    test('should reject empty content', async () => {
      await expect(store.addPost('')).rejects.toThrow('Status content cannot be empty');
    });

    test('should reject content over 500 chars', async () => {
      const longContent = 'a'.repeat(501);
      await expect(store.addPost(longContent)).rejects.toThrow(
        'Status content cannot exceed 500 characters'
      );
    });

    test('should trim whitespace', async () => {
      const post = await store.addPost('  Hello  ');
      expect(post.content).toBe('Hello');
    });
  });

  describe('getPosts', () => {
    test('should return empty array when no posts', async () => {
      const posts = await store.getPosts();
      expect(posts).toEqual([]);
    });

    test('should return posts in reverse chronological order', async () => {
      const post1 = await store.addPost('First post');
      await new Promise((resolve) => setTimeout(resolve, 10));
      const post2 = await store.addPost('Second post');

      const posts = await store.getPosts();

      expect(posts).toHaveLength(2);
      expect(posts[0]?.id).toBe(post2.id);
      expect(posts[1]?.id).toBe(post1.id);
    });

    test('should respect limit and offset', async () => {
      await store.addPost('Post 1');
      await store.addPost('Post 2');
      await store.addPost('Post 3');

      const posts = await store.getPosts(2, 1);
      expect(posts).toHaveLength(2);
    });
  });

  describe('getPost', () => {
    test('should retrieve post by ID', async () => {
      const created = await store.addPost('Test post');
      const retrieved = await store.getPost(created.id);

      expect(retrieved).toBeDefined();
      expect(retrieved?.content).toBe('Test post');
    });

    test('should return null for non-existent ID', async () => {
      const retrieved = await store.getPost('non-existent-id');
      expect(retrieved).toBeNull();
    });
  });
});
