import React from 'react';
import { Layout, Typography, Space } from 'antd';
import type { Account } from '@mau-network/mau';
import type { AccountManager } from '../account/manager';
import type { StatusPost } from '../types/index';
import { AuthForm } from './auth';
import { Composer } from './composer';
import { Timeline } from './timeline';

const { Header, Content } = Layout;
const { Title } = Typography;

export function AppHeader(): React.ReactElement {
  return (
    <Header style={{ background: '#fff', padding: '0 24px', boxShadow: '0 2px 8px rgba(0,0,0,0.1)' }}>
      <Title level={3} style={{ margin: '16px 0' }}>
        Mau Status
      </Title>
    </Header>
  );
}

interface AppContentProps {
  account: Account | null;
  accountManager: Promise<AccountManager>;
  onAuthenticated: (account: Account) => void;
  onPost: (content: string) => Promise<void>;
  posts: StatusPost[];
  loading: boolean;
}

export function AppContent({
  account,
  accountManager,
  onAuthenticated,
  onPost,
  posts,
  loading,
}: AppContentProps): React.ReactElement {
  return (
    <Content style={{ padding: '24px', maxWidth: 800, margin: '0 auto', width: '100%' }}>
      {!account ? (
        <AuthForm onAuthenticated={onAuthenticated} accountManager={accountManager} />
      ) : (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <Composer onSubmit={onPost} />
          <Timeline posts={posts} loading={loading} />
        </Space>
      )}
    </Content>
  );
}
