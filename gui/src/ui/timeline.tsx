import React from 'react';
import { List, Card, Typography, Space, Empty } from 'antd';
import { ClockCircleOutlined } from '@ant-design/icons';
import type { StatusPost } from '../types/index';

const { Text, Paragraph } = Typography;

interface TimelineProps {
  posts: StatusPost[];
  loading?: boolean;
  onLoadMore?: () => void;
  hasMore?: boolean;
}

export function Timeline({ posts, loading = false, onLoadMore, hasMore = false }: TimelineProps): React.ReactElement {
  if (!loading && posts.length === 0) {
    return (
      <Card>
        <Empty description="No status updates yet. Create your first post!" />
      </Card>
    );
  }

  return (
    <List
      dataSource={posts}
      loading={loading}
      renderItem={(post) => <StatusPostCard post={post} />}
      loadMore={hasMore && onLoadMore ? <LoadMoreButton onClick={onLoadMore} loading={loading} /> : null}
    />
  );
}

function StatusPostCard({ post }: { post: StatusPost }): React.ReactElement {
  const formatTime = (timestamp: number): string => {
    const diff = Date.now() - timestamp;
    const days = Math.floor(diff / 86400000);
    const hours = Math.floor(diff / 3600000);
    const minutes = Math.floor(diff / 60000);

    if (days > 0) return `${days}d ago`;
    if (hours > 0) return `${hours}h ago`;
    if (minutes > 0) return `${minutes}m ago`;
    return 'just now';
  };

  return (
    <Card style={{ marginBottom: 16 }}>
      <Space direction="vertical" style={{ width: '100%' }}>
        <Paragraph style={{ margin: 0, fontSize: 16 }}>{post.content}</Paragraph>
        <Space>
          <ClockCircleOutlined />
          <Text type="secondary">{formatTime(post.createdAt)}</Text>
        </Space>
      </Space>
    </Card>
  );
}

function LoadMoreButton({ onClick, loading }: { onClick: () => void; loading: boolean }): React.ReactElement {
  return (
    <div style={{ textAlign: 'center', marginTop: 16 }}>
      <Card>
        <button onClick={onClick} disabled={loading}>
          Load More
        </button>
      </Card>
    </div>
  );
}
