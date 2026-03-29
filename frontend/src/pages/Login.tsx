import React, { useState } from 'react';
import { Form, Input, Button, message, Card, Divider, Typography, Space } from 'antd';
import { UserOutlined, LockOutlined, LoginOutlined } from '@ant-design/icons';
import axios from 'axios';

const { Title, Text } = Typography;

interface LoginProps {
  onLogin: (user: any) => void;
}

const Login: React.FC<LoginProps> = ({ onLogin }) => {
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm();

  const handleLogin = async (values: { email: string; password: string }) => {
    setLoading(true);
    try {
      const response = await axios.post('/api/auth/login', {
        email: values.email.trim(),
        password: values.password,
      }, {
        withCredentials: true,
      });
      const { code, data, message: msg } = response.data;
      if (code === 0 && data) {
        // Save user info
        localStorage.setItem('current_user', JSON.stringify(data.user));
        message.success('登录成功');
        onLogin(data.user);
      } else {
        message.error(msg || '登录失败');
      }
    } catch (error) {
      message.error('登录失败: ' + (error as Error).message);
    } finally {
      setLoading(false);
    }
  };

  const handleFeishuLogin = () => {
    // Redirect to Feishu OAuth endpoint
    window.location.href = '/api/auth/feishu';
  };

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
    }}>
      <Card style={{ width: 420, boxShadow: '0 8px 32px rgba(0,0,0,0.2)' }}>
        <div style={{ textAlign: 'center', marginBottom: 24 }}>
          <Title level={2} style={{ marginBottom: 8 }}>🛡️ 流云卫士</Title>
          <Text type="secondary">流量哨兵控制台</Text>
        </div>

        <Form form={form} onFinish={handleLogin} layout="vertical">
          <Form.Item
            name="email"
            label="用户名"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input
              prefix={<UserOutlined />}
              size="large"
            />
          </Form.Item>

          <Form.Item
            name="password"
            label="密码"
            rules={[{ required: true, message: '请输入密码' }]}
          >
            <Input.Password
              prefix={<LockOutlined />}
              placeholder="输入密码"
              size="large"
            />
          </Form.Item>

          <Form.Item>
            <Button
              type="primary"
              htmlType="submit"
              loading={loading}
              block
              size="large"
              icon={<LoginOutlined />}
            >
              登录
            </Button>
          </Form.Item>
        </Form>

        <Divider plain>
          <Text type="secondary">或</Text>
        </Divider>

        <Button
          block
          size="large"
          onClick={handleFeishuLogin}
          style={{ marginBottom: 16 }}
        >
          <Space>
            <span style={{ fontSize: 18 }}>📱</span>
            飞书登录
          </Space>
        </Button>

        <div style={{ textAlign: 'center', marginTop: 16 }}>
          <Text type="secondary" style={{ fontSize: 12 }}>
            版本 1.0 | 帮助文档
          </Text>
        </div>
      </Card>
    </div>
  );
};

export default Login;
