import { Account, BrowserStorage } from '@mau-network/mau';
import type { AccountState } from '../types/index';

const ACCOUNT_KEY = 'mau:account';
const ACCOUNT_DIR = 'mau:account';

export class AccountManager {
  private storage: BrowserStorage;
  private currentAccount: Account | null = null;

  private constructor(storage: BrowserStorage) {
    this.storage = storage;
  }

  static async create(): Promise<AccountManager> {
    const storage = await BrowserStorage.create();
    return new AccountManager(storage);
  }

  async createAccount(name: string, email: string, passphrase: string): Promise<Account> {
    this.validateInputs(name, email, passphrase);

    // Delete existing account if any
    await this.deleteAccount();

    const account = await Account.create(this.storage, ACCOUNT_DIR, {
      name,
      email,
      passphrase,
    });

    await this.saveAccountState(account);
    this.currentAccount = account;

    return account;
  }

  async unlockAccount(passphrase: string): Promise<Account> {
    const accountState = await this.getAccountState();

    if (!accountState) {
      throw new Error('No account found');
    }

    const account = await Account.load(this.storage, ACCOUNT_DIR, passphrase);

    await this.updateLastUnlocked();
    this.currentAccount = account;

    return account;
  }

  async hasAccount(): Promise<boolean> {
    const accountState = await this.getAccountState();
    return accountState !== null;
  }

  async getAccountInfo(): Promise<AccountState | null> {
    return await this.getAccountState();
  }

  getCurrentAccount(): Account | null {
    return this.currentAccount;
  }

  private async deleteAccount(): Promise<void> {
    try {
      // Delete account state
      await this.storage.remove(ACCOUNT_KEY);
      // Delete account directory (will fail silently if not exists)
      await this.storage.remove(ACCOUNT_DIR);
    } catch {
      // Ignore errors - account may not exist
    }
  }

  private async getAccountState(): Promise<AccountState | null> {
    try {
      const stored = await this.storage.readText(ACCOUNT_KEY);
      return JSON.parse(stored) as AccountState;
    } catch {
      return null;
    }
  }

  private validateInputs(name: string, email: string, passphrase: string): void {
    if (name.length < 1 || name.length > 100) {
      throw new Error('Name must be 1-100 characters');
    }

    if (!this.isValidEmail(email)) {
      throw new Error('Invalid email address');
    }

    if (passphrase.length < 12) {
      throw new Error('Passphrase must be at least 12 characters');
    }
  }

  private isValidEmail(email: string): boolean {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email);
  }

  private async saveAccountState(account: Account): Promise<void> {
    const fingerprint = account.getFingerprint();

    const newState: AccountState = {
      fingerprint,
      name: account.getName(),
      email: account.getEmail(),
      accountDir: ACCOUNT_DIR,
      createdAt: Date.now(),
      lastUnlocked: Date.now(),
    };

    await this.storage.writeText(ACCOUNT_KEY, JSON.stringify(newState));
  }

  private async updateLastUnlocked(): Promise<void> {
    const accountState = await this.getAccountState();

    if (accountState) {
      accountState.lastUnlocked = Date.now();
      await this.storage.writeText(ACCOUNT_KEY, JSON.stringify(accountState));
    }
  }
}
