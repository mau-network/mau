import React, { useState, useEffect } from 'react';
import { Card, List, Button, Upload, Input, Space, Typography, message, Modal, Divider } from 'antd';
import { UserAddOutlined, DeleteOutlined, CopyOutlined, UploadOutlined } from '@ant-design/icons';
import type { AccountManager } from '../account/manager';

const { TextArea } = Input;
const { Text, Title } = Typography;

interface Friend {
  fingerprint: string;
  name: string;
  email: string;
}

interface FriendsPageProps {
  accountManager: AccountManager;
}

export function FriendsPage({ accountManager }: FriendsPageProps): React.ReactElement {
  const [friends, setFriends] = useState<Friend[]>([]);
  const [loading, setLoading] = useState(false);
  const [addModalOpen, setAddModalOpen] = useState(false);
  const [publicKeyInput, setPublicKeyInput] = useState('');
  const [myPublicKey, setMyPublicKey] = useState<string>('');
  const [myFingerprint, setMyFingerprint] = useState<string>('');

  useEffect(() => {
    void loadFriends();
    void loadMyPublicKey();
  }, []);

  const loadFriends = async (): Promise<void> => {
    setLoading(true);
    try {
      const friendsList = await accountManager.listFriends();
      setFriends(friendsList);
    } catch (error) {
      message.error('Failed to load friends');
    } finally {
      setLoading(false);
    }
  };

  const loadMyPublicKey = async (): Promise<void> => {
    try {
      const key = await accountManager.exportPublicKey();
      setMyPublicKey(key);
      
      // Get fingerprint from current account
      const account = accountManager.getCurrentAccount();
      if (account) {
        setMyFingerprint(account.getFingerprint());
      }
    } catch (error) {
      console.error('Failed to export public key:', error);
    }
  };

  const handleAddFriend = async (): Promise<void> => {
    if (!publicKeyInput.trim()) {
      message.error('Please paste a public key');
      return;
    }

    setLoading(true);
    try {
      const friend = await accountManager.addFriend(publicKeyInput);
      message.success(`Added ${friend.name} as a friend`);
      setPublicKeyInput('');
      setAddModalOpen(false);
      await loadFriends();
    } catch (error) {
      message.error(`Failed to add friend: ${(error as Error).message}`);
    } finally {
      setLoading(false);
    }
  };

  const handleRemoveFriend = async (fingerprint: string, name: string): Promise<void> => {
    Modal.confirm({
      title: 'Remove Friend',
      content: `Are you sure you want to remove ${name}?`,
      okText: 'Remove',
      okType: 'danger',
      onOk: async () => {
        try {
          await accountManager.removeFriend(fingerprint);
          message.success(`Removed ${name}`);
          await loadFriends();
        } catch (error) {
          message.error('Failed to remove friend');
        }
      },
    });
  };

  const handleFileUpload = async (file: File): Promise<void> => {
    const text = await file.text();
    setPublicKeyInput(text);
    return;
  };

  const copyMyPublicKey = (): void => {
    void navigator.clipboard.writeText(myPublicKey);
    message.success('Public key copied to clipboard');
  };

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      {/* My Public Key Section */}
      <Card title="My Public Key">
        <Space direction="vertical" size="small" style={{ width: '100%' }}>
          <Text type="secondary">Share this key with others so they can add you as a friend</Text>
          {myFingerprint && (
            <Text strong style={{ fontFamily: 'monospace', fontSize: 12 }}>
              Fingerprint: {myFingerprint}
            </Text>
          )}
          <TextArea
            value={myPublicKey}
            readOnly
            rows={6}
            style={{ marginTop: 8, fontFamily: 'monospace', fontSize: 11 }}
          />
          <Button
            type="primary"
            icon={<CopyOutlined />}
            onClick={copyMyPublicKey}
            style={{ marginTop: 8 }}
          >
            Copy Public Key
          </Button>
        </Space>
      </Card>

      <Divider />

      {/* Friends List */}
      <Card
        title={<Title level={4}>Friends ({friends.length})</Title>}
        extra={
          <Button type="primary" icon={<UserAddOutlined />} onClick={() => setAddModalOpen(true)}>
            Add Friend
          </Button>
        }
      >
        {friends.length === 0 ? (
          <Text type="secondary">No friends yet. Add a friend to start sharing!</Text>
        ) : (
          <List
            loading={loading}
            dataSource={friends}
            renderItem={(friend) => (
              <List.Item
                actions={[
                  <Button
                    type="text"
                    danger
                    icon={<DeleteOutlined />}
                    onClick={() => handleRemoveFriend(friend.fingerprint, friend.name)}
                  >
                    Remove
                  </Button>,
                ]}
              >
                <List.Item.Meta
                  title={friend.name}
                  description={
                    <Space direction="vertical" size={0}>
                      <Text type="secondary">{friend.email}</Text>
                      <Text code style={{ fontSize: 10 }}>
                        {friend.fingerprint}
                      </Text>
                    </Space>
                  }
                />
              </List.Item>
            )}
          />
        )}
      </Card>

      {/* Add Friend Modal */}
      <Modal
        title="Add Friend"
        open={addModalOpen}
        onOk={handleAddFriend}
        onCancel={() => {
          setAddModalOpen(false);
          setPublicKeyInput('');
        }}
        okText="Add Friend"
        confirmLoading={loading}
        width={600}
      >
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <Text>Upload or paste your friend's public key:</Text>
          
          <Upload
            accept=".asc,.pgp,.txt"
            beforeUpload={(file) => {
              void handleFileUpload(file);
              return false;
            }}
            showUploadList={false}
          >
            <Button icon={<UploadOutlined />}>Upload Public Key File</Button>
          </Upload>

          <Text type="secondary">Or paste the key below:</Text>
          
          <TextArea
            value={publicKeyInput}
            onChange={(e) => setPublicKeyInput(e.target.value)}
            placeholder="-----BEGIN PGP PUBLIC KEY BLOCK-----&#10;...&#10;-----END PGP PUBLIC KEY BLOCK-----"
            rows={10}
            style={{ fontFamily: 'monospace', fontSize: 11 }}
          />
        </Space>
      </Modal>
    </Space>
  );
}
