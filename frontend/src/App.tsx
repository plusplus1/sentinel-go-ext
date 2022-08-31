import React, { Suspense, lazy, useState, useEffect, useMemo } from 'react';
import { BrowserRouter as Router, Routes, Route, Link, useLocation, Navigate } from 'react-router-dom';
import { Layout, Menu, Spin, Dropdown, Avatar, Space, Cascader } from 'antd';
import { UserOutlined, LogoutOutlined, AppstoreOutlined, CloudServerOutlined, TeamOutlined } from '@ant-design/icons';
import { useApp } from './context/AppContext';
import axios from 'axios';

// Configure axios to always send cookies
axios.defaults.withCredentials = true;

const Groups = lazy(() => import('./pages/Groups'));
const Resources = lazy(() => import('./pages/Resources'));
const Login = lazy(() => import('./pages/Login'));
const Admin = lazy(() => import('./pages/Admin'));

const { Header, Content, Sider } = Layout;

interface CurrentUser {
  id: string;
  user_id: string;
  email: string;
  name: string;
  role: 'super_admin' | 'line_admin' | 'member';
  avatar_url?: string;
}

const AppLayout: React.FC = () => {
  const location = useLocation();
  const { businessLines, selectedApp, selectApp, loading } = useApp();
  const [collapsed, setCollapsed] = useState(false);
  const [currentUser, setCurrentUser] = useState<CurrentUser | null>(null);
  const [authLoading, setAuthLoading] = useState(true);

  // Helper function to find line_id by app_id
  const getLineIdByAppId = (appId: string): string => {
    for (const line of businessLines) {
      if (line.apps.some(app => app.id === appId)) {
        return line.id;
      }
    }
    return '';
  };

  // Check auth on mount
  useEffect(() => {
    const checkAuth = async () => {
      try {
        const response = await axios.get('/api/auth/me');
        const { code, data } = response.data;
        if (code === 0 && data) {
          setCurrentUser(data);
          localStorage.setItem('current_user', JSON.stringify(data));
        } else {
          // Not authenticated, redirect to login
          localStorage.removeItem('current_user');
          window.location.href = '/web/login';
        }
      } catch (error) {
        // Network error or 401, redirect to login
        localStorage.removeItem('current_user');
        window.location.href = '/web/login';
      } finally {
        setAuthLoading(false);
      }
    };
    checkAuth();
  }, []);

  const handleLogout = async () => {
    try { await axios.post('/api/auth/logout'); } catch (e) { console.warn('logout failed', e); }
    localStorage.removeItem('current_user');
    setCurrentUser(null);
    window.location.href = '/web/login';
  };

  // Dynamic menu based on user role
  const menuItems = useMemo(() => {
    const items: any[] = [];

    // 超级管理员和业务线管理员：看到管理中心
    if (currentUser?.role === 'super_admin' || currentUser?.role === 'line_admin') {
      items.push({
        key: '/admin',
        icon: <TeamOutlined />,
        label: <Link to="/admin">管理中心</Link>,
      });
    }

    // 普通成员：只看到资源中心和服务模块
    if (currentUser?.role !== 'super_admin') {
      items.push(
        {
          key: '/resources',
          icon: <CloudServerOutlined />,
          label: <Link to="/resources">资源中心</Link>,
        },
        {
          key: '/modules',
          icon: <AppstoreOutlined />,
          label: <Link to="/modules">服务模块</Link>,
        },
      );
    }

    return items;
  }, [currentUser]);

  // Show loading while checking auth
  if (authLoading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh' }}>
        <Spin size="large" />
      </div>
    );
  }

  // Redirect to login if not authenticated
  if (!currentUser) {
    return <Navigate to="/login" replace />;
  }

  const userMenuItems = [
    { key: 'logout', label: '退出登录', icon: <LogoutOutlined />, danger: true },
  ];

  const handleUserMenuClick = ({ key }: { key: string }) => {
    if (key === 'logout') handleLogout();
  };

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <style>{`
        @media (max-width: 768px) {
          .ant-select { min-width: 140px !important; }
          .ant-table-wrapper { overflow-x: auto; }
          .ant-table { min-width: 600px; }
          .ant-btn span { display: inline !important; }
          .ant-layout-sider { position: fixed !important; z-index: 1000; }
        }
        @media (max-width: 480px) {
          .ant-select { width: 120px !important; }
        }
      `}</style>
      <Sider
        collapsible
        theme="dark"
        collapsed={collapsed}
        onCollapse={setCollapsed}
        breakpoint="lg"
        collapsedWidth={64}
        width={200}
        style={{
          overflow: 'auto',
          height: '100vh',
          position: 'sticky',
          top: 0,
          left: 0,
        }}
      >
        <div style={{
          height: 64,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          borderBottom: '1px solid rgba(255, 255, 255, 0.1)',
          marginBottom: 16,
          padding: collapsed ? '8px' : '0 16px',
          overflow: 'hidden',
        }}>
          {collapsed ? (
            <img src="/web/logo.png" alt="流云卫士"
              style={{ height: 36, width: 36, borderRadius: 6, objectFit: 'contain' }} />
          ) : (
            <div style={{ display: 'flex', alignItems: 'center', gap: 10, width: '100%' }}>
              <img src="/web/logo.png" alt="流云卫士"
                style={{ height: 36, width: 36, borderRadius: 6, objectFit: 'contain', flexShrink: 0 }} />
              <span style={{ color: '#fff', fontSize: 16, fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                流云卫士
              </span>
            </div>
          )}
        </div>
        <Menu theme="dark" selectedKeys={[location.pathname]} mode="inline" items={menuItems} />
      </Sider>

      <Layout>
        <Header style={{
          padding: '0 16px', background: '#fff', borderBottom: '1px solid #f0f0f0',
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          boxShadow: '0 1px 4px rgba(0, 0, 0, 0.08)',
          flexWrap: 'wrap', gap: 8, minHeight: 64,
        }}>
          <div style={{ display: 'flex', alignItems: 'center' }}>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            {/* 非超级管理员才显示应用选择器 */}
            {currentUser?.role !== 'super_admin' && (
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <span style={{ color: 'rgba(0, 0, 0, 0.45)', fontSize: 14 }}>应用:</span>
                <Cascader
                  style={{ width: 260 }}
                  placeholder="选择业务线 / 应用"
                  loading={loading}
                  value={selectedApp ? [getLineIdByAppId(selectedApp.id), selectedApp.id] : undefined}
                  options={businessLines.map(line => ({
                    label: line.name,
                    value: line.id,
                    isLeaf: false,
                    children: line.apps.map(app => ({
                      label: app.name,
                      value: app.id,
                      isLeaf: true,
                    })),
                  }))}
                  onChange={(values) => {
                    if (values && values.length >= 2) {
                      selectApp(values[1] as string);
                    }
                  }}
                  changeOnSelect
                  showSearch={{ filter: (inputValue, path) => {
                    return path.some(option => 
                      String(option.label).toLowerCase().indexOf(inputValue.toLowerCase()) > -1
                    );
                  } }}
                />
              </div>
            )}
            <Dropdown menu={{ items: userMenuItems, onClick: handleUserMenuClick }} placement="bottomRight" arrow>
              <Space style={{ cursor: 'pointer' }}>
                <Avatar size={36} icon={<UserOutlined />} style={{ backgroundColor: '#1890ff' }} />
                <span style={{ color: 'rgba(0, 0, 0, 0.85)', fontWeight: 500 }}>
                  {currentUser?.name || '用户'}
                </span>
              </Space>
            </Dropdown>
          </div>
        </Header>

        <Content style={{ margin: '16px', padding: '16px', minHeight: 280, background: '#fff', borderRadius: 8, boxShadow: '0 1px 4px rgba(0, 0, 0, 0.08)' }}>
          <Suspense fallback={<div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: 400 }}><Spin size="large" /></div>}>
            <Routes>
              <Route path="/" element={
                currentUser?.role === 'super_admin' || currentUser?.role === 'line_admin'
                  ? <Navigate to="/admin" replace /> 
                  : <Navigate to="/resources" replace />
              } />
              <Route path="/modules" element={<Groups />} />
              <Route path="/resources" element={<Resources />} />
              {(currentUser?.role === 'super_admin' || currentUser?.role === 'line_admin') && <Route path="/admin" element={<Admin />} />}
            </Routes>
          </Suspense>
        </Content>
      </Layout>
    </Layout>
  );
};

const App: React.FC = () => {
  return (
    <Router basename="/web">
      <Suspense fallback={<div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh' }}><Spin size="large" /></div>}>
        <Routes>
          <Route path="/login" element={<Login onLogin={() => window.location.href = '/web/'} />} />
          <Route path="/*" element={<AppLayout />} />
        </Routes>
      </Suspense>
    </Router>
  );
};

export default App;
