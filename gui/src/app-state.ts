import { useState, useEffect } from 'react';
import { message } from 'antd';
import type { Account } from '@mau-network/mau';
import { AccountManager } from './account/manager';
import { StatusStoreManager } from './status/store';
import { getNetworkConfig } from './config/network';
import type { StatusPost } from './types/index';

interface AppState {
  accountManager: Promise<AccountManager>;
  account: Account | null;
  setAccount: (account: Account | null) => void;
  posts: StatusPost[];
  loading: boolean;
  handlePost: (content: string) => Promise<void>;
}

export function useAppState(): AppState {
  const [accountManager] = useState(() => AccountManager.create(getNetworkConfig()));
  const [account, setAccount] = useState<Account | null>(null);
  const [statusStore, setStatusStore] = useState<StatusStoreManager | null>(null);
  const [posts, setPosts] = useState<StatusPost[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (account) {
      (async () => {
        const store = await StatusStoreManager.create(account);
        setStatusStore(store);
        await loadPosts(store);
      })();
    }
  }, [account]);

  const loadPosts = async (store: StatusStoreManager): Promise<void> => {
    setLoading(true);
    try {
      setPosts(await store.getPosts(20, 0));
    } catch {
      message.error('Failed to load posts');
    } finally {
      setLoading(false);
    }
  };

  const handlePost = async (content: string): Promise<void> => {
    if (!statusStore) return;
    try {
      await statusStore.addPost(content);
      await loadPosts(statusStore);
      message.success('Posted!');
    } catch (error) {
      message.error((error as Error).message);
    }
  };

  return {
    accountManager,
    account,
    setAccount,
    posts,
    loading,
    handlePost,
  };
}
