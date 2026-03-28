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

    // Save friend's public key in binary format encrypted with account key (per spec)
    // Spec: "All friends' public keys should be encrypted with the account key"
    // Rationale: Prevents malicious programs from tampering with the contact list
    const friendKeyPath = this.storage.join(this.getMauDir(), `${fingerprint}.pgp`);
    const binaryKey = publicKey.write();
    
    // Encrypt with account's public key
    const { signAndEncrypt } = await import('./crypto/index.js');
    const encryptedKey = await signAndEncrypt(binaryKey, this.privateKey, [this.publicKey]);
    await this.storage.writeText(friendKeyPath, encryptedKey);

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
          // Read encrypted key (per spec: keys are encrypted with account key)
          const encryptedKey = await this.storage.readText(keyPath);
          const { decryptAndVerify } = await import('./crypto/index.js');
          const { data: binaryKey } = await decryptAndVerify(
            encryptedKey,
            this.privateKey,
            [this.publicKey]
          );
          
          const publicKey = await openpgp.readKey({ binaryKey });
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

  /**
   * Get storage instance (for internal use)
   * @internal
   */
  getStorage(): Storage {
    return this.storage;
  }

  /**
   * Get root path (for internal use)
   * @internal
   */
  getRootPath(): string {
    return this.rootPath;
  }

  // File operations

  /**
   * Create a new file
   * @param fileName File name (relative to content directory)
   * @returns File instance
   * @throws {InvalidFileNameError} If file name is invalid
   * 
   * @example
   * ```typescript
   * const file = account.createFile('posts/hello.json');
   * await file.writeJSON({ '@type': 'SocialMediaPosting', headline: 'Hello!' });
   * ```
   */
  async createFile(fileName: string): Promise<import('./file.js').File> {
    const { File } = await import('./file.js');
    const { validateFileName } = await import('./crypto/index.js');
    const { InvalidFileNameError } = await import('./types/index.js');
    
    if (!validateFileName(fileName)) {
      throw new InvalidFileNameError('contains invalid characters or path separators');
    }

    const contentDir = this.getContentDir();
    const filePath = this.storage.join(contentDir, fileName);

    return new File(this, this.storage, filePath, false);
  }

  /**
   * List all files in this account's content directory
   * @returns Array of File instances
   * 
   * @example
   * ```typescript
   * const files = await account.listFiles();
   * for (const file of files) {
   *   console.log(file.getName());
   * }
   * ```
   */
  async listFiles(): Promise<Array<import('./file.js').File>> {
    const { File } = await import('./file.js');
    const contentDir = this.getContentDir();
    
    if (!(await this.storage.exists(contentDir))) {
      return [];
    }

    const entries = await this.storage.readDir(contentDir);
    const files: Array<import('./file.js').File> = [];

    for (const entry of entries) {
      const filePath = this.storage.join(contentDir, entry);
      const stats = await this.storage.stat(filePath);
      
      // Skip directories and version directories
      if (!stats.isDirectory && !entry.endsWith('.versions')) {
        files.push(new File(this, this.storage, filePath, false));
      }
    }

    return files;
  }

  /**
   * List all files in a friend's content directory
   * @param fingerprint Friend's fingerprint
   * @returns Array of File instances
   * 
   * @example
   * ```typescript
   * const files = await account.listFriendFiles(friendFingerprint);
   * for (const file of files) {
   *   const data = await file.readJSON();
   * }
   * ```
   */
  async listFriendFiles(fingerprint: Fingerprint): Promise<Array<import('./file.js').File>> {
    const { File } = await import('./file.js');
    const contentDir = this.getFriendContentDir(fingerprint);
    
    if (!(await this.storage.exists(contentDir))) {
      return [];
    }

    const entries = await this.storage.readDir(contentDir);
    const files: Array<import('./file.js').File> = [];

    for (const entry of entries) {
      const filePath = this.storage.join(contentDir, entry);
      const stats = await this.storage.stat(filePath);
      
      if (!stats.isDirectory && !entry.endsWith('.versions')) {
        files.push(new File(this, this.storage, filePath, false));
      }
    }

    return files;
  }

  // Client/Server operations

  /**
   * Create a client for syncing with a peer
   * @param peer Peer fingerprint and optional address
   * @param resolvers Peer discovery resolvers
   * @returns Client instance
   * 
   * @example
   * ```typescript
   * const client = account.createClient(
   *   { fingerprint: friendFingerprint, address: 'peer.example.com:443' },
   *   [staticResolver(knownPeers), dhtResolver(bootstrapNodes)]
   * );
   * const stats = await client.sync();
   * ```
   */
  createClient(
    peer: import('./types/index.js').Peer,
    resolvers: Array<import('./types/index.js').FingerprintResolver> = []
  ): import('./client.js').Client {
    // Dynamic import to avoid circular dependency
    // eslint-disable-next-line @typescript-eslint/no-var-requires
    const { Client } = require('./client.js');
    return new Client(this, this.storage, peer.fingerprint, {
      resolvers: [
        // Add static address resolver
        async (fingerprint: string): Promise<string | null> => {
          if (fingerprint === peer.fingerprint) {
            return peer.address;
          }
          return null;
        },
        ...resolvers,
      ],
    });
  }

  /**
   * Create a server for serving files to peers
   * @param config Server configuration
   * @param dht Optional DHT instance for peer discovery
   * @returns Server instance
   * 
   * @example
   * ```typescript
   * const server = account.createServer({ resultsLimit: 100 });
   * const response = await server.handleRequest(req);
   * ```
   */
  createServer(
    config: import('./types/index.js').ServerConfig = {},
    dht?: import('./network/dht.js').KademliaDHT
  ): import('./server.js').Server {
    // Dynamic import to avoid circular dependency
    // eslint-disable-next-line @typescript-eslint/no-var-requires
    const { Server } = require('./server.js');
    return new Server(this, this.storage, config, dht);
  }
}
