export interface StatusPost {
  id: string;
  content: string;
  createdAt: number;
  signature: string;
}

export interface StatusStore {
  posts: StatusPost[];
  version: number;
}

export interface AccountState {
  fingerprint: string;
  name: string;
  email: string;
  accountDir: string; // Storage path for the account
  createdAt: number;
  lastUnlocked: number;
}
