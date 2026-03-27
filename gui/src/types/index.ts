export interface StatusPost {
  id: string;
  content: string;
  createdAt: number;
  fileName: string; // The .pgp file name
}

export interface AccountState {
  fingerprint: string;
  name: string;
  email: string;
  accountDir: string; // Storage path for the account
  createdAt: number;
  lastUnlocked: number;
}
