/**
 * Account - User Identity Management
 * 
 * Manages PGP keys, friends (keyring), and account data.
 */

import * as openpgp from 'openpgp';
import type {
  Storage,
  Fingerprint,
  AccountOptions,
  SyncState,
} from './types/index.js';
import {
  MAU_DIR_NAME,
  ACCOUNT_KEY_FILENAME,
  SYNC_STATE_FILENAME,
  AccountAlreadyExistsError,
  NoIdentityError,
  PassphraseRequiredError,
} from './types/index.js';
import {
  generateKeyPair,
  serializePrivateKey,
  serializePublicKey,
  deserializePrivateKey,
  deserializePublicKey,
  getFingerprint,
} from './crypto/index.js';

export class Account {
  private privateKey: openpgp.PrivateKey;
  private publicKey: openpgp.PublicKey;
  private storage: Storage;
  private rootPath: string;
  private friends: Map<Fingerprint, openpgp.PublicKey> = new Map();

  constructor(
    privateKey: openpgp.PrivateKey,
    publicKey: openpgp.PublicKey,
    storage: Storage,
    rootPath: string
  ) {
    this.privateKey = privateKey;
    this.publicKey = publicKey;
    this.storage = storage;
    this.rootPath = rootPath;
  }

  /**
   * Create a new account
   */
  static async create(
    storage: Storage,
    rootPath: string,
    options: AccountOptions
  ): Promise<Account> {
    if (!options.passphrase) {
      throw new PassphraseRequiredError();
    }

    const mauDir = storage.join(rootPath, MAU_DIR_NAME);
    const accountFile = storage.join(mauDir, ACCOUNT_KEY_FILENAME);

    // Check if account already exists
    if (await storage.exists(accountFile)) {
      throw new AccountAlreadyExistsError();
    }

    // Create .mau directory
    await storage.mkdir(mauDir);

    // Generate key pair
    const { privateKey, publicKey } = await generateKeyPair(options);

    // Save encrypted private key
    const armoredPrivate = await serializePrivateKey(privateKey, options.passphrase);
    await storage.writeText(accountFile, armoredPrivate);

    // Create account directory for user's content
    const fingerprint = getFingerprint(publicKey);
    const contentDir = storage.join(mauDir, fingerprint);
    await storage.mkdir(contentDir);

    return new Account(privateKey, publicKey, storage, rootPath);
  }

  /**
   * Load an existing account
   */
  static async load(
    storage: Storage,
    rootPath: string,
    passphrase: string
  ): Promise<Account> {
    if (!passphrase) {
      throw new PassphraseRequiredError();
    }

    const mauDir = storage.join(rootPath, MAU_DIR_NAME);
    const accountFile = storage.join(mauDir, ACCOUNT_KEY_FILENAME);

    // Check if account exists
    if (!(await storage.exists(accountFile))) {
      throw new NoIdentityError();
    }

    // Load and decrypt private key
    const armoredPrivate = await storage.readText(accountFile);
    const privateKey = await deserializePrivateKey(armoredPrivate, passphrase);

    // Derive public key from private key
    const publicKey = privateKey.toPublic();

    const account = new Account(privateKey, publicKey, storage, rootPath);

    // Load friends
    await account.loadFriends();

    return account;
  }

  /**
   * Get account fingerprint
   */
  getFingerprint(): Fingerprint {
    return getFingerprint(this.publicKey);
  }

  /**
   * Get public key in armored format
   */
  getPublicKey(): string {
    return serializePublicKey(this.publicKey);
  }

  /**
   * Get account name from key
   */
  getName(): string {
    const user = this.publicKey.users[0];
    return user?.userID?.name || '';
  }

  /**
   * Get account email from key
   */
  getEmail(): string {
    const user = this.publicKey.users[0];
    return user?.userID?.email || '';
  }

  /**
   * Get mau directory path
   */
  getMauDir(): string {
    return this.storage.join(this.rootPath, MAU_DIR_NAME);
  }

  /**
   * Get content directory for this account
   */
  getContentDir(): string {
    return this.storage.join(this.getMauDir(), this.getFingerprint());
  }

  /**
   * Get content directory for a friend
   */
  getFriendContentDir(fingerprint: Fingerprint): string {
    return this.storage.join(this.getMauDir(), fingerprint);
  }

  /**
   * Add a friend by importing their public key
   */
  async addFriend(armoredPublicKey: string): Promise<Fingerprint> {
    const publicKey = await deserializePublicKey(armoredPublicKey);
    const fingerprint = getFingerprint(publicKey);

    // Save friend's public key
    const friendKeyPath = this.storage.join(this.getMauDir(), `${fingerprint}.pgp`);
    await this.storage.writeText(friendKeyPath, armoredPublicKey);

    // Create content directory for friend
    const friendContentDir = this.getFriendContentDir(fingerprint);
    await this.storage.mkdir(friendContentDir);

    // Add to in-memory keyring
    this.friends.set(fingerprint, publicKey);

    return fingerprint;
  }

  /**
   * Remove a friend
   */
  async removeFriend(fingerprint: Fingerprint): Promise<void> {
    const friendKeyPath = this.storage.join(this.getMauDir(), `${fingerprint}.pgp`);
    const friendContentDir = this.getFriendContentDir(fingerprint);

    // Remove key file
    if (await this.storage.exists(friendKeyPath)) {
      await this.storage.remove(friendKeyPath);
    }

    // Remove content directory
    if (await this.storage.exists(friendContentDir)) {
      await this.storage.remove(friendContentDir);
    }

    // Remove from in-memory keyring
    this.friends.delete(fingerprint);
  }

  /**
   * Get list of friend fingerprints
   */
  getFriends(): Fingerprint[] {
    return Array.from(this.friends.keys());
  }

  /**
   * Get a friend's public key
   */
  getFriendKey(fingerprint: Fingerprint): openpgp.PublicKey | undefined {
    return this.friends.get(fingerprint);
  }

  /**
   * Check if a fingerprint is a friend
   */
  isFriend(fingerprint: Fingerprint): boolean {
    return this.friends.has(fingerprint);
  }

  /**
   * Load all friends from disk
   */
  private async loadFriends(): Promise<void> {
    const mauDir = this.getMauDir();
    if (!(await this.storage.exists(mauDir))) {
      return;
    }

    const entries = await this.storage.readDir(mauDir);

    for (const entry of entries) {
      if (entry.endsWith('.pgp') && entry !== ACCOUNT_KEY_FILENAME) {
        const keyPath = this.storage.join(mauDir, entry);
        try {
          const armoredKey = await this.storage.readText(keyPath);
          const publicKey = await deserializePublicKey(armoredKey);
          const fingerprint = getFingerprint(publicKey);
          this.friends.set(fingerprint, publicKey);
        } catch (err) {
          console.error(`Failed to load friend key ${entry}:`, err);
        }
      }
    }
  }

  /**
   * Get sync state
   */
  async getSyncState(): Promise<SyncState> {
    const syncStatePath = this.storage.join(this.getMauDir(), SYNC_STATE_FILENAME);
    
    if (!(await this.storage.exists(syncStatePath))) {
      return {};
    }

    try {
      const json = await this.storage.readText(syncStatePath);
      return JSON.parse(json);
    } catch {
      return {};
    }
  }

  /**
   * Update sync state for a fingerprint
   */
  async updateSyncState(fingerprint: Fingerprint, timestamp: number): Promise<void> {
    const syncState = await this.getSyncState();
    syncState[fingerprint] = timestamp;

    const syncStatePath = this.storage.join(this.getMauDir(), SYNC_STATE_FILENAME);
    await this.storage.writeText(syncStatePath, JSON.stringify(syncState, null, 2));
  }

  /**
   * Get private key (for internal use)
   */
  getPrivateKey(): openpgp.PrivateKey {
    return this.privateKey;
  }

  /**
   * Get public key object (for internal use)
   */
  getPublicKeyObject(): openpgp.PublicKey {
    return this.publicKey;
  }

  /**
   * Get all public keys (self + friends) for encryption
   */
  getAllPublicKeys(): openpgp.PublicKey[] {
    return [this.publicKey, ...Array.from(this.friends.values())];
  }
}
