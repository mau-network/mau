import React, { useState, useEffect } from 'react';
import { Form, Input, Button, Card, message, Tabs, Typography } from 'antd';
import { UserOutlined, MailOutlined, LockOutlined } from '@ant-design/icons';
import type { Account } from '@mau-network/mau';
import type { AccountManager } from '../account/manager';
import type { AccountState } from '../types/index';

const { Text } = Typography;

interface AuthFormProps {
  onAuthenticated: (account: Account) => void;
  accountManager: Promise<AccountManager>;
}

interface CreateValues {
  name: string;
  email: string;
  passphrase: string;
}

interface UnlockValues {
  passphrase: string;
}

export function AuthForm({ onAuthenticated, accountManager }: AuthFormProps): React.ReactElement {
  const [accountInfo, setAccountInfo] = useState<AccountState | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const checkAccount = async (): Promise<void> => {
      try {
        const manager = await accountManager;
        const info = await manager.getAccountInfo();
        setAccountInfo(info);
      } catch (error) {
        console.error('Failed to check account:', error);
      } finally {
        setLoading(false);
      }
    };

    void checkAccount();
  }, [accountManager]);

  if (loading) {
    return (
      <Card style={{ maxWidth: 500, margin: '40px auto' }}>
        <Text>Loading...</Text>
      </Card>
    );
  }

  return (
    <Card style={{ maxWidth: 500, margin: '40px auto' }}>
      <Tabs
        defaultActiveKey={accountInfo ? 'unlock' : 'create'}
        items={[
          {
            key: 'create',
            label: 'Create Account',
            children: <CreateAccountForm onAuthenticated={onAuthenticated} accountManager={accountManager} />,
          },
          {
            key: 'unlock',
            label: 'Unlock Account',
            children: <UnlockAccountForm onAuthenticated={onAuthenticated} accountManager={accountManager} accountInfo={accountInfo} />,
          },
        ]}
      />
    </Card>
  );
}

function CreateAccountForm({
  onAuthenticated,
  accountManager,
}: AuthFormProps): React.ReactElement {
  const [loading, setLoading] = useState(false);

  const handleCreate = async (values: CreateValues): Promise<void> => {
    setLoading(true);
    try {
      const manager = await accountManager;
      const account = await manager.createAccount(values.name, values.email, values.passphrase);
      message.success('Account created successfully!');
      onAuthenticated(account);
    } catch (error) {
      message.error((error as Error).message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Form onFinish={handleCreate} layout="vertical">
      <Form.Item
        name="name"
        label="Name"
        rules={[{ required: true, min: 2, message: 'Name must be at least 2 characters' }]}
      >
        <Input prefix={<UserOutlined />} placeholder="Your name" />
      </Form.Item>
      <Form.Item
        name="email"
        label="Email"
        rules={[{ required: true, type: 'email', message: 'Invalid email' }]}
      >
        <Input prefix={<MailOutlined />} placeholder="you@example.com" />
      </Form.Item>
      <Form.Item
        name="passphrase"
        label="Passphrase"
        rules={[{ required: true, min: 12, message: 'At least 12 characters' }]}
      >
        <Input.Password prefix={<LockOutlined />} placeholder="Secure passphrase" />
      </Form.Item>
      <Form.Item>
        <Button type="primary" htmlType="submit" loading={loading} block>
          Create Account
        </Button>
      </Form.Item>
    </Form>
  );
}

interface UnlockFormProps extends AuthFormProps {
  accountInfo: AccountState | null;
}

function AccountInfoDisplay({ accountInfo }: { accountInfo: AccountState }): React.ReactElement {
  return (
    <div style={{ marginBottom: 16, padding: 12, backgroundColor: '#f5f5f5', borderRadius: 4 }}>
      <Text strong>Account: </Text>
      <Text>{accountInfo.name}</Text>
      <br />
      <Text strong>Email: </Text>
      <Text>{accountInfo.email}</Text>
      <br />
      <Text strong>Fingerprint: </Text>
      <Text code style={{ fontSize: 11 }}>{accountInfo.fingerprint}</Text>
    </div>
  );
}

function UnlockAccountForm({
  onAuthenticated,
  accountManager,
  accountInfo,
}: UnlockFormProps): React.ReactElement {
  const [loading, setLoading] = useState(false);

  const handleUnlock = async (values: UnlockValues): Promise<void> => {
    setLoading(true);
    try {
      const manager = await accountManager;
      const account = await manager.unlockAccount(values.passphrase);
      message.success('Account unlocked successfully!');
      onAuthenticated(account);
    } catch (error) {
      message.error((error as Error).message);
    } finally {
      setLoading(false);
    }
  };

  if (!accountInfo) {
    return (
      <div style={{ textAlign: 'center', padding: '20px' }}>
        <Text type="secondary">No account found. Please create one first.</Text>
      </div>
    );
  }

  return (
    <div>
      <AccountInfoDisplay accountInfo={accountInfo} />
      
      <Form onFinish={handleUnlock} layout="vertical">
        <Form.Item
          name="passphrase"
          label="Passphrase"
          rules={[{ required: true, message: 'Passphrase is required' }]}
        >
          <Input.Password prefix={<LockOutlined />} placeholder="Your passphrase" />
        </Form.Item>
        <Form.Item>
          <Button type="primary" htmlType="submit" loading={loading} block>
            Unlock Account
          </Button>
        </Form.Item>
      </Form>
    </div>
  );
}
