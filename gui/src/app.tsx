import React from 'react';
import { ConfigProvider, theme, Layout } from 'antd';
import { useAppState } from './app-state';
import { AppHeader, AppContent } from './ui/components';

export function App(): React.ReactElement {
  const { accountManager, account, setAccount, posts, loading, handlePost } = useAppState();

  return (
    <ConfigProvider theme={{ algorithm: theme.defaultAlgorithm }}>
      <Layout style={{ minHeight: '100vh' }}>
        <AppHeader account={account} />
        <AppContent
          account={account}
          accountManager={accountManager}
          onAuthenticated={setAccount}
          onPost={handlePost}
          posts={posts}
          loading={loading}
        />
      </Layout>
    </ConfigProvider>
  );
}
