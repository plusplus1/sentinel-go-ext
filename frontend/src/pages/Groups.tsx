import React, { useEffect, useState } from 'react';
import { Table, Button, message, Spin, Modal, Form, Input, Space, Card, Empty, Tag, Drawer, List, Typography, Popconfirm } from 'antd';
import { AppstoreOutlined, PlusOutlined, ReloadOutlined, EditOutlined, DeleteOutlined, TeamOutlined, UserAddOutlined, UserDeleteOutlined } from '@ant-design/icons';
import axios from 'axios';
import { useApp } from '../context/AppContext';

interface Module {
  id: string;
  name: string;
  description: string;
  app_id: string;
  env: string;
  member_count: number;
  created_at: string;
  updated_at: string;
}

interface Resource {
  resource: string;
  group_id: string;
  group_name: string;
  app_id: string;
  env: string;
}

const Groups: React.FC = () => {
  const { selectedApp, getAppFullPath } = useApp();
  const [modules, setModules] = useState<Module[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingModule, setEditingModule] = useState<Module | null>(null);
  const [form] = Form.useForm();

  // Members drawer state
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [selectedModule, setSelectedModule] = useState<Module | null>(null);
  const [members, setMembers] = useState<Resource[]>([]);
  const [membersLoading, setMembersLoading] = useState(false);
  const [addResourceName, setAddResourceName] = useState('');

  const fetchModules = async (appId: string) => {
    if (!appId) return;
    setLoading(true);
    try {
      const response = await axios.get('/api/groups', {
        params: { app: appId },
      });
      const { code, data, message: msg } = response.data;
      if (code === 0 && data) {
        setModules(data);
      } else {
        message.error(msg || '获取服务模块列表失败');
      }
    } catch (error) {
      message.error('请求失败: ' + (error as Error).message);
    } finally {
      setLoading(false);
    }
  };

  const fetchMembers = async (moduleId: string) => {
    if (!selectedApp) return;
    setMembersLoading(true);
    try {
      const response = await axios.get(`/api/groups/${moduleId}/members`, {
        params: { app: selectedApp.id },
      });
      const { code, data, message: msg } = response.data;
      if (code === 0 && data) {
        setMembers(data.members || []);
      } else {
        message.error(msg || '获取成员列表失败');
      }
    } catch (error) {
      message.error('请求失败: ' + (error as Error).message);
    } finally {
      setMembersLoading(false);
    }
  };

  useEffect(() => {
    if (selectedApp) {
      fetchModules(selectedApp.id);
    } else {
      setModules([]);
    }
  }, [selectedApp]);

  const handleCreate = () => {
    if (!selectedApp) {
      message.warning('请先在顶部选择应用');
      return;
    }
    setEditingModule(null);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEdit = (record: Module) => {
    setEditingModule(record);
    form.setFieldsValue({
      name: record.name,
      description: record.description,
    });
    setModalVisible(true);
  };

  const handleDelete = async (record: Module) => {
    try {
      const response = await axios.delete(`/api/groups/${record.id}`, {
        params: { app: selectedApp?.id },
        data: { action: 'move_to_default' },
      });
      const { code, message: msg } = response.data;
      if (code === 0) {
        message.success('删除成功');
        fetchModules(selectedApp!.id);
      } else {
        message.error(msg || '删除失败');
      }
    } catch (error) {
      message.error('删除失败: ' + (error as Error).message);
    }
  };

  const handleModalOk = async () => {
    try {
      const values = await form.validateFields();
      if (editingModule) {
        // Update - only description, not name
        const response = await axios.put(`/api/groups/${editingModule.id}`, {
          app_id: selectedApp!.id,
          description: values.description,
        });
        const { code, message: msg } = response.data;
        if (code === 0) {
          message.success('更新成功');
          setModalVisible(false);
          fetchModules(selectedApp!.id);
        } else {
          message.error(msg || '更新失败');
        }
      } else {
        // Create
        const response = await axios.post('/api/groups', {
          app_id: selectedApp?.id,
          name: values.name,
          description: values.description,
        });
        const { code, message: msg } = response.data;
        if (code === 0) {
          message.success('创建成功');
          setModalVisible(false);
          fetchModules(selectedApp!.id);
        } else {
          message.error(msg || '创建失败');
        }
      }
    } catch (error) {
      console.error('Validate Failed:', error);
    }
  };

  const handleViewMembers = (record: Module) => {
    setSelectedModule(record);
    setDrawerVisible(true);
    fetchMembers(record.id);
  };

  const handleAddResource = async () => {
    if (!addResourceName.trim() || !selectedModule) return;
    try {
      const response = await axios.post(`/api/groups/${selectedModule.id}/members`, {
        resource: addResourceName.trim(),
      });
      const { code, message: msg } = response.data;
      if (code === 0) {
        message.success('添加成功');
        setAddResourceName('');
        fetchMembers(selectedModule.id);
        fetchModules(selectedApp!.id);
      } else {
        message.error(msg || '添加失败');
      }
    } catch (error) {
      message.error('添加失败: ' + (error as Error).message);
    }
  };

  const handleRemoveResource = async (resourceName: string) => {
    if (!selectedModule) return;
    try {
      const response = await axios.delete(
        `/api/groups/${selectedModule.id}/members/${resourceName}`,
      );
      const { code, message: msg } = response.data;
      if (code === 0) {
        message.success('移除成功');
        fetchMembers(selectedModule.id);
        fetchModules(selectedApp!.id);
      } else {
        message.error(msg || '移除失败');
      }
    } catch (error) {
      message.error('移除失败: ' + (error as Error).message);
    }
  };

  const columns = [
    {
      title: '服务模块名称',
      dataIndex: 'name',
      key: 'name',
      width: 180,
      ellipsis: { showTitle: false },
      render: (val: string, _: Module) => (
        <Space>
          <Tag color="blue">自定义</Tag>
          <span style={{ fontWeight: 500 }}>{val}</span>
        </Space>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      width: 300,
      ellipsis: { showTitle: false },
    },
    {
      title: '成员数',
      dataIndex: 'member_count',
      key: 'member_count',
      width: 80,
      align: 'center' as const,
      render: (val: number) => <Tag color="cyan">{val || 0}</Tag>,
    },
    {
      title: '更新时间',
      dataIndex: 'updated_at',
      key: 'updated_at',
      width: 180,
      render: (val: string) => val ? new Date(val).toLocaleString('zh-CN') : '-',
    },
    {
      title: '操作',
      key: 'action',
      width: 180,
      fixed: 'right' as const,
      render: (_: any, record: Module) => (
        <Space size="small">
          <Button type="link" size="small" icon={<TeamOutlined />} onClick={() => handleViewMembers(record)}>
            成员
          </Button>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)}>
            编辑
          </Button>
          <Popconfirm
            title="确认删除"
            description={"确定删除服务模块 \"" + record.name + "\" 吗？"}
            onConfirm={() => handleDelete(record)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  if (!selectedApp) {
    return (
      <div style={{ textAlign: 'center', padding: 60 }}>
        <Empty description="请先在顶部选择应用" />
      </div>
    );
  }

  return (
    <div>
      <div style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 24,
      }}>
        <div>
          <h2 style={{ margin: 0, fontSize: 24, fontWeight: 600 }}>
            <AppstoreOutlined style={{ marginRight: 8 }} />
            服务模块
          </h2>
          <p style={{ margin: '8px 0 0 0', fontSize: 14, color: 'rgba(0, 0, 0, 0.45)' }}>
            应用: <Tag color="blue">{getAppFullPath(selectedApp.id)}</Tag> | 共 {modules.length} 个服务模块
          </p>
        </div>
        <Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => fetchModules(selectedApp.id)}
          >
            刷新
          </Button>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={handleCreate}
          >
            新建服务模块
          </Button>
        </Space>
      </div>

      <Card>
        <Spin spinning={loading}>
          <Table
            rowKey="id"
            dataSource={modules}
            columns={columns}
            pagination={{ pageSize: 10 }}
          />
        </Spin>
      </Card>

      {/* Create/Edit Modal */}
      <Modal
        title={editingModule ? '编辑服务模块' : '新建服务模块'}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          {!editingModule && (
            <Form.Item
              name="name"
              label="服务模块名称"
              rules={[{ required: true, message: '请输入服务模块名称' }]}
            >
              <Input placeholder="例如: 用户中心" />
            </Form.Item>
          )}
          {editingModule && (
            <Form.Item label="服务模块名称">
              <Input value={editingModule.name} disabled />
            </Form.Item>
          )}
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} placeholder="模块描述（可选）" />
          </Form.Item>
        </Form>
      </Modal>

      {/* Members Drawer */}
      <Drawer
        title={
          <Space>
            <TeamOutlined />
            <span>{selectedModule?.name} - 成员管理</span>
          </Space>
        }
        width={480}
        open={drawerVisible}
        onClose={() => setDrawerVisible(false)}
      >
        <div style={{ marginBottom: 16 }}>
          <Space.Compact style={{ width: '100%' }}>
            <Input
              placeholder="输入资源名称添加到模块"
              value={addResourceName}
              onChange={(e) => setAddResourceName(e.target.value)}
              onPressEnter={handleAddResource}
            />
            <Button type="primary" icon={<UserAddOutlined />} onClick={handleAddResource}>
              添加
            </Button>
          </Space.Compact>
        </div>

        <Spin spinning={membersLoading}>
          <List
            dataSource={members}
            locale={{ emptyText: '暂无成员' }}
            renderItem={(item) => (
              <List.Item
                actions={[
                  <Popconfirm
                    title="确认移除"
                    description={`确定将资源 "${item.resource}" 移出模块吗？`}
                    onConfirm={() => handleRemoveResource(item.resource)}
                    okText="确定"
                    cancelText="取消"
                  >
                    <Button type="link" danger icon={<UserDeleteOutlined />} size="small">
                      移除
                    </Button>
                  </Popconfirm>,
                ]}
              >
                <List.Item.Meta
                  title={<Typography.Text>{item.resource}</Typography.Text>}
                />
              </List.Item>
            )}
          />
        </Spin>
      </Drawer>
    </div>
  );
};

export default Groups;
