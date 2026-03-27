import React, { useState } from 'react';
import { Layout, Typography, Space, Menu } from 'antd';
import { HomeOutlined, TeamOutlined } from '@ant-design/icons';
import type { Account } from '@mau-network/mau';
import type { AccountManager } from '../account/manager';
import type { StatusPost } from '../types/index';
import { AuthForm } from './auth';
import { Composer } from './composer';
import { Timeline } from './timeline';
import { FriendsPage } from './friends';

const { Header, Content, Sider } = Layout;
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
  const [currentPage, setCurrentPage] = useState<'feed' | 'friends'>('feed');
  const [resolvedAccountManager, setResolvedAccountManager] = useState<AccountManager | null>(null);

  React.useEffect(() => {
    if (account) {
      accountManager.then(setResolvedAccountManager);
    }
  }, [account, accountManager]);

  if (!account) {
    return (
      <Content style={{ padding: '24px', maxWidth: 800, margin: '0 auto', width: '100%' }}>
        <AuthForm onAuthenticated={onAuthenticated} accountManager={accountManager} />
      </Content>
    );
  }

  return (
    <Layout style={{ minHeight: 'calc(100vh - 64px)' }}>
      <Sider width={200} style={{ background: '#fff' }}>
        <Menu
          mode="inline"
          selectedKeys={[currentPage]}
          onClick={({ key }) => setCurrentPage(key as 'feed' | 'friends')}
          style={{ height: '100%', borderRight: 0 }}
          items={[
            {
              key: 'feed',
              icon: <HomeOutlined />,
              label: 'Feed',
            },
            {
              key: 'friends',
              icon: <TeamOutlined />,
              label: 'Friends',
            },
          ]}
        />
      </Sider>
      <Layout style={{ padding: '24px' }}>
        <Content style={{ maxWidth: 800, margin: '0 auto', width: '100%' }}>
          {currentPage === 'feed' ? (
            <Space direction="vertical" size="large" style={{ width: '100%' }}>
              <Composer onSubmit={onPost} />
              <Timeline posts={posts} loading={loading} />
            </Space>
          ) : (
            resolvedAccountManager && <FriendsPage accountManager={resolvedAccountManager} />
          )}
        </Content>
      </Layout>
    </Layout>
  );
}
