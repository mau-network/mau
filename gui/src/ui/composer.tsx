import React, { useState } from 'react';
import { Form, Input, Button, Card, Space, Typography } from 'antd';
import { SendOutlined } from '@ant-design/icons';

const { TextArea } = Input;
const { Text } = Typography;

interface ComposerProps {
  onSubmit: (content: string) => Promise<void>;
}

export function Composer({ onSubmit }: ComposerProps): React.ReactElement {
  const [form] = Form.useForm();
  const [charCount, setCharCount] = useState(0);

  const handleSubmit = async (values: { content: string }): Promise<void> => {
    await onSubmit(values.content);
    form.resetFields();
    setCharCount(0);
  };

  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>): void => {
    setCharCount(e.target.value.length);
  };

  return (
    <Card>
      <Form form={form} onFinish={handleSubmit}>
        <Form.Item
          name="content"
          rules={[
            { required: true, message: 'Status cannot be empty' },
            { max: 500, message: 'Status cannot exceed 500 characters' },
          ]}
        >
          <TextArea
            placeholder="What's on your mind?"
            autoSize={{ minRows: 3, maxRows: 6 }}
            onChange={handleChange}
            maxLength={500}
          />
        </Form.Item>
        <CharCounter count={charCount} />
      </Form>
    </Card>
  );
}

function CharCounter({ count }: { count: number }): React.ReactElement {
  return (
    <Space style={{ width: '100%', justifyContent: 'space-between' }}>
      <Text type={count > 500 ? 'danger' : 'secondary'}>{count} / 500</Text>
      <Button type="primary" htmlType="submit" icon={<SendOutlined />}>
        Post
      </Button>
    </Space>
  );
}
