import React, { useEffect, useState, useRef } from 'react';
import { Table, Button, message, Modal, Form, Input, Space, Card, Tag, Tabs, Select, Popconfirm, Empty, Pagination } from 'antd';
import { PlusOutlined, EditOutlined, UserAddOutlined, UserDeleteOutlined } from '@ant-design/icons';
import axios from 'axios';

const { TabPane } = Tabs;

interface AdminInfo {
  user_id: string;
  user_name: string;
  user_email: string;
  user_status: string;
}

interface MemberInfo {
  user_id: string;
  user_name: string;
  user_email: string;
  user_status: string;
  added_at: string;
}

interface BusinessLine {
  id: string;
  name: string;
  description: string;
  status: string;
  admins: AdminInfo[];
  updated_at: string;
}

interface AppInfo {
  id: number;
  app_key: string;
  description: string;
  settings: string;
  status: string;
  created_at: string;
}

interface UserOption {
  user_id: string;
  name: string;
  email: string;
}

interface CurrentUser {
  user_id: string;
  name: string;
  email: string;
  role: string;
}

// ==================== Super Admin 管理中心 ====================
const SuperAdminPanel: React.FC = () => {
  const [activeLines, setActiveLines] = useState<BusinessLine[]>([]);
  const [deletedLines, setDeletedLines] = useState<BusinessLine[]>([]);
  const [loading, setLoading] = useState(false);
  const [lineModal, setLineModal] = useState(false);
  const [editModal, setEditModal] = useState(false);
  const [bindModal, setBindModal] = useState(false);
  const [lineForm] = Form.useForm();
  const [editForm] = Form.useForm();
  const [searchForm] = Form.useForm();
  const [currentLine, setCurrentLine] = useState<BusinessLine | null>(null);
  const [userOptions, setUserOptions] = useState<UserOption[]>([]);
  const [searchLoading, setSearchLoading] = useState(false);
  const searchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const fetchLines = async () => {
    setLoading(true);
    try {
      const resp = await axios.get('/api/admin/lines');
      if (resp.data.code === 0) {
        const allLines = resp.data.data || [];
        setActiveLines(allLines.filter((l: BusinessLine) => l.status === 'active'));
        setDeletedLines(allLines.filter((l: BusinessLine) => l.status === 'deleted'));
      }
    } catch (error) {
      message.error('获取业务线列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchLines();
  }, []);

  const handleCreateLine = async () => {
    try {
      const values = await lineForm.validateFields();
      if (!/^[a-zA-Z0-9_]{3,50}$/.test(values.name)) {
        message.error('业务线名称只能包含英文、数字、下划线，长度3-50字符');
        return;
      }
      const resp = await axios.post('/api/admin/lines', values);
      if (resp.data.code === 0) {
        message.success('业务线创建成功');
        setLineModal(false);
        lineForm.resetFields();
        fetchLines();
      } else {
        message.error(resp.data.message || '创建失败');
      }
    } catch (error) {
      message.error('请检查输入');
    }
  };

  const handleUpdateDescription = async () => {
    if (!currentLine) return;
    try {
      const values = await editForm.validateFields();
      const resp = await axios.put(`/api/admin/lines/${currentLine.id}`, {
        description: values.description,
        status: currentLine.status,
      });
      if (resp.data.code === 0) {
        message.success('描述更新成功');
        setEditModal(false);
        editForm.resetFields();
        fetchLines();
      } else {
        message.error(resp.data.message || '更新失败');
      }
    } catch (error) {
      message.error('请检查输入');
    }
  };

  const handleToggleStatus = async (line: BusinessLine) => {
    const newStatus = line.status === 'active' ? 'deleted' : 'active';
    try {
      const resp = await axios.put(`/api/admin/lines/${line.id}`, {
        status: newStatus,
      });
      if (resp.data.code === 0) {
        message.success(`业务线已${newStatus === 'active' ? '激活' : '下线'}`);
        fetchLines();
      } else {
        message.error(resp.data.message || '操作失败');
      }
    } catch (error) {
      message.error('操作失败');
    }
  };

  const handleRemoveAdmin = async (line: BusinessLine, userId: string) => {
    try {
      const resp = await axios.delete(`/api/admin/lines/${line.id}/admins/${userId}`);
      if (resp.data.code === 0) {
        message.success('管理员移除成功');
        fetchLines();
      } else {
        message.error(resp.data.message || '移除失败');
      }
    } catch (error) {
      message.error('移除失败');
    }
  };

  const handleAddAdmin = async () => {
    if (!currentLine) return;
    try {
      const values = await searchForm.validateFields();
      const resp = await axios.post(`/api/admin/lines/${currentLine.id}/admins`, {
        user_id: values.selectedUser,
      });
      if (resp.data.code === 0) {
        message.success('管理员添加成功');
        setBindModal(false);
        searchForm.resetFields();
        fetchLines();
      } else {
        message.error(resp.data.message || '添加失败');
      }
    } catch (error) {
      message.error('请选择用户');
    }
  };

  const searchUsers = async (keyword: string) => {
    if (searchTimerRef.current) {
      clearTimeout(searchTimerRef.current);
    }
    
    if (!keyword) {
      setUserOptions([]);
      return;
    }
    
    searchTimerRef.current = setTimeout(async () => {
      setSearchLoading(true);
      try {
        const resp = await axios.get(`/api/users/search?keyword=${keyword}`);
        if (resp.data.code === 0) {
          setUserOptions(resp.data.data || []);
        }
      } catch (error) {
        console.error('搜索用户失败');
      } finally {
        setSearchLoading(false);
      }
    }, 300);
  };

  const activeColumns = [
    { title: '业务线名称', dataIndex: 'name', key: 'name' },
    { title: '描述', dataIndex: 'description', key: 'description' },
    { title: '状态', dataIndex: 'status', key: 'status', render: (s: string) => <Tag color={s === 'active' ? 'green' : 'red'}>{s === 'active' ? '生效' : '下线'}</Tag> },
    { title: '更新时间', dataIndex: 'updated_at', key: 'updated_at', render: (v: string) => v ? new Date(v).toLocaleString('zh-CN') : '-' },
    {
      title: '业务线管理员',
      key: 'admins',
      render: (_: any, record: BusinessLine) => (
        record.admins && record.admins.length > 0 ? (
          <Space direction="vertical" size={4}>
            {record.admins.map(admin => (
              <Space key={admin.user_id} size={4}>
                <span>{admin.user_name || admin.user_id}</span>
                <span style={{ fontSize: 12, color: '#666' }}>{admin.user_email}</span>
                <Tag color={admin.user_status === 'active' ? 'green' : 'red'}>
                  {admin.user_status === 'active' ? '在线' : '离线'}
                </Tag>
                <Popconfirm title="确定移除此管理员？" onConfirm={() => handleRemoveAdmin(record, admin.user_id)}>
                  <Button size="small" type="link" danger icon={<UserDeleteOutlined />}>移除</Button>
                </Popconfirm>
              </Space>
            ))}
          </Space>
        ) : (
          <Tag color="orange">未绑定</Tag>
        )
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: BusinessLine) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => {
            setCurrentLine(record);
            editForm.setFieldsValue({ description: record.description });
            setEditModal(true);
          }}>
            修改描述
          </Button>
          <Button size="small" icon={<UserAddOutlined />} onClick={() => {
            setCurrentLine(record);
            setBindModal(true);
          }}>添加管理员</Button>
          <Popconfirm title="确定下线业务线？" onConfirm={() => handleToggleStatus(record)}>
            <Button size="small" danger>下线</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const deletedColumns = [
    { title: '业务线名称', dataIndex: 'name', key: 'name' },
    { title: '描述', dataIndex: 'description', key: 'description' },
    { title: '状态', dataIndex: 'status', key: 'status', render: (_s: string) => <Tag color="red">下线</Tag> },
    { title: '更新时间', dataIndex: 'updated_at', key: 'updated_at', render: (v: string) => v ? new Date(v).toLocaleString('zh-CN') : '-' },
    {
      title: '业务线管理员',
      key: 'admins',
      render: (_: any, record: BusinessLine) => (
        record.admins && record.admins.length > 0 ? (
          <Space direction="vertical" size={4}>
            {record.admins.map(admin => (
              <Space key={admin.user_id} size={4}>
                <span>{admin.user_name || admin.user_id}</span>
                <span style={{ fontSize: 12, color: '#666' }}>{admin.user_email}</span>
              </Space>
            ))}
          </Space>
        ) : (
          <Tag color="orange">未绑定</Tag>
        )
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: BusinessLine) => (
        <Space>
          <Popconfirm title="确定重新激活？" onConfirm={() => handleToggleStatus(record)}>
            <Button size="small" type="primary">激活</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Card>
        <Tabs 
          defaultActiveKey="active"
          tabBarExtraContent={
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setLineModal(true)}>
              新建业务线
            </Button>
          }
        >
          <TabPane tab={`生效中 (${activeLines.length})`} key="active">
            <Table 
              rowKey="id" 
              dataSource={activeLines} 
              columns={activeColumns} 
              loading={loading}
              pagination={{ pageSize: 10 }}
              locale={{ emptyText: <Empty description="暂无生效中的业务线" /> }}
            />
          </TabPane>
          <TabPane tab={`已下线 (${deletedLines.length})`} key="deleted">
            <Table 
              rowKey="id" 
              dataSource={deletedLines} 
              columns={deletedColumns} 
              loading={loading}
              pagination={{ pageSize: 10 }}
              locale={{ emptyText: <Empty description="暂无已下线的业务线" /> }}
            />
          </TabPane>
        </Tabs>
      </Card>

      <Modal title="新建业务线" open={lineModal} onOk={handleCreateLine} onCancel={() => setLineModal(false)}>
        <Form form={lineForm} layout="vertical">
          <Form.Item 
            name="name" 
            label="业务线名称" 
            rules={[
              { required: true, message: '请输入业务线名称' },
              { pattern: /^[a-zA-Z0-9_]{3,50}$/, message: '只能包含英文、数字、下划线，长度3-50字符' }
            ]}
          >
            <Input placeholder="例如：user_center" />
          </Form.Item>
          <Form.Item 
            name="description" 
            label="描述" 
            rules={[{ required: true, message: '请输入描述' }]}
          >
            <Input.TextArea rows={3} placeholder="业务线描述（必填）" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title="添加业务线管理员" open={bindModal} onOk={handleAddAdmin} onCancel={() => setBindModal(false)}>
        <Form form={searchForm} layout="vertical">
          <Form.Item name="selectedUser" rules={[{ required: true, message: '请选择用户' }]}>
            <Select 
              placeholder="输入姓名或邮箱搜索并选择"
              loading={searchLoading}
              showSearch
              filterOption={false}
              onSearch={searchUsers}
              style={{ width: '100%' }}
              notFoundContent={searchLoading ? '搜索中...' : '输入关键词搜索用户'}
            >
              {userOptions.map(user => (
                <Select.Option key={user.user_id} value={user.user_id}>
                  {user.name} ({user.email})
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      <Modal title="修改描述" open={editModal} onOk={handleUpdateDescription} onCancel={() => setEditModal(false)}>
        <Form form={editForm} layout="vertical">
          <Form.Item 
            name="description" 
            label="描述" 
            rules={[{ required: true, message: '请输入描述' }]}
          >
            <Input.TextArea rows={3} placeholder="业务线描述" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

// ==================== Line Admin 管理中心 ====================
// Line 卡片组件
const LineCard: React.FC<{
  line: BusinessLine;
  onEditDesc: (line: BusinessLine) => void;
  onCreateApp: (line: BusinessLine) => void;
  onEditApp: (line: BusinessLine, app: AppInfo) => void;
  onDeleteApp: (line: BusinessLine, app: AppInfo) => void;
  onRefreshApps: (lineId: string) => void;
  appsMap: Record<string, AppInfo[]>;
  appsLoading: boolean;
  onAddMember: (line: BusinessLine) => void;
  onRemoveMember: (line: BusinessLine, userId: string) => void;
  membersMap: Record<string, MemberInfo[]>;
  membersLoading: boolean;
}> = ({ line, onEditDesc, onCreateApp, onEditApp, onDeleteApp, onRefreshApps, appsMap, appsLoading, onAddMember, onRemoveMember, membersMap, membersLoading }) => {
  const apps = appsMap[line.id] || [];
  const members = membersMap[line.id] || [];

  useEffect(() => {
    onRefreshApps(line.id);
  }, [line.id]);

  return (
    <>
      <Card 
      style={{ marginBottom: 16 }}
      styles={{ header: { backgroundColor: '#f5f5f5', paddingLeft: 16 } }}
      title={
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Space>
            <span style={{ fontSize: 16, fontWeight: 600 }}>{line.name}</span>
            <Tag color={line.status === 'active' ? 'green' : 'red'}>
              {line.status === 'active' ? '生效' : '下线'}
            </Tag>
            <span style={{ color: '#666', fontSize: 12 }}>{line.description}</span>
          </Space>
          <Space>
            <Button size="small" icon={<EditOutlined />} onClick={() => onEditDesc(line)}>
              修改描述
            </Button>
            <Button size="small" icon={<PlusOutlined />} type="primary" onClick={() => onCreateApp(line)}>
              创建app
            </Button>
          </Space>
        </div>
      }
    >
      {appsLoading ? (
        <div style={{ textAlign: 'center', padding: 20 }}>加载中...</div>
      ) : apps.length === 0 ? (
        <Empty description="暂无应用" image={Empty.PRESENTED_IMAGE_SIMPLE} />
      ) : (
        <Table
          size="small"
          dataSource={apps}
          rowKey="id"
          pagination={false}
          columns={[
            {
              title: '应用标识',
              dataIndex: 'app_key',
              key: 'app_key',
              width: 150,
              render: (val: string) => <code>{val}</code>,
            },
            {
              title: '描述',
              dataIndex: 'description',
              key: 'description',
            },
            {
              title: '状态',
              dataIndex: 'status',
              key: 'status',
              width: 100,
              render: (val: string) => (
                <Tag color={val === 'active' ? 'green' : val === 'maintaining' ? 'orange' : 'red'}>
                  {val === 'active' ? '生效' : val === 'maintaining' ? '维护' : '下线'}
                </Tag>
              ),
            },
            {
              title: '操作',
              key: 'action',
              width: 150,
              render: (_: any, app: AppInfo) => (
                <Space size="small">
                  <Button size="small" type="link" onClick={() => onEditApp(line, app)}>
                    修改
                  </Button>
                  <Popconfirm 
                    title={app.status === 'deleted' ? '确定恢复此应用？' : '确定下线此应用？'}
                    onConfirm={() => onDeleteApp(line, app)}
                  >
                    <Button size="small" type="link" danger>
                      {app.status === 'deleted' ? '恢复' : '下线'}
                    </Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
          />
        )}
      </Card>

      <Card
        title="成员管理"
        style={{ marginBottom: 16 }}
        extra={
          <Button size="small" icon={<UserAddOutlined />} type="primary" onClick={() => onAddMember(line)}>
            添加成员
          </Button>
        }
      >
        {membersLoading ? (
          <div style={{ textAlign: 'center', padding: 20 }}>加载中...</div>
        ) : members.length === 0 ? (
          <Empty description="暂无成员" image={Empty.PRESENTED_IMAGE_SIMPLE} />
        ) : (
          <Table
            size="small"
            dataSource={members}
            rowKey="user_id"
            pagination={false}
            columns={[
              {
                title: '成员姓名',
                dataIndex: 'user_name',
                key: 'user_name',
                render: (val: string, record: MemberInfo) => val || record.user_id,
              },
              {
                title: '邮箱',
                dataIndex: 'user_email',
                key: 'user_email',
              },
              {
                title: '状态',
                dataIndex: 'user_status',
                key: 'user_status',
                width: 80,
                render: (val: string) => (
                  <Tag color={val === 'active' ? 'green' : 'red'}>
                    {val === 'active' ? '在线' : '离线'}
                  </Tag>
                ),
              },
              {
                title: '添加时间',
                dataIndex: 'added_at',
                key: 'added_at',
                render: (v: string) => v ? new Date(v).toLocaleString('zh-CN') : '-',
              },
              {
                title: '操作',
                key: 'action',
                width: 80,
                render: (_: any, member: MemberInfo) => (
                  <Popconfirm title="确定移除此成员？" onConfirm={() => onRemoveMember(line, member.user_id)}>
                    <Button size="small" type="link" danger>移除</Button>
                  </Popconfirm>
                ),
              },
            ]}
          />
        )}
      </Card>
    </>
  );
};

const LineAdminPanel: React.FC<{ currentUser: CurrentUser }> = () => {
  const [myLines, setMyLines] = useState<BusinessLine[]>([]);
  const [loading, setLoading] = useState(false);
  const [editDescModal, setEditDescModal] = useState(false);
  const [createAppModal, setCreateAppModal] = useState(false);
  const [editAppModal, setEditAppModal] = useState(false);
  const [descForm] = Form.useForm();
  const [appForm] = Form.useForm();
  const [editAppForm] = Form.useForm();
  const [currentLine, setCurrentLine] = useState<BusinessLine | null>(null);
  const [currentApp, setCurrentApp] = useState<AppInfo | null>(null);
  const [appsMap, setAppsMap] = useState<Record<string, AppInfo[]>>({});
  const [appsLoading, setAppsLoading] = useState(false);

  const [membersMap, setMembersMap] = useState<Record<string, MemberInfo[]>>({});
  const [membersLoading, setMembersLoading] = useState(false);
  const [memberModal, setMemberModal] = useState(false);
  const [memberForm] = Form.useForm();
  const [memberUserOptions, setMemberUserOptions] = useState<UserOption[]>([]);
  const [memberSearchLoading, setMemberSearchLoading] = useState(false);
  const memberSearchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const fetchMyLines = async () => {
    setLoading(true);
    try {
      const resp = await axios.get('/api/line-admin/lines');
      if (resp.data.code === 0) {
        setMyLines(resp.data.data || []);
      }
    } catch (error) {
      message.error('获取业务线列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchMyLines();
  }, []);

  const fetchApps = async (lineId: string) => {
    try {
      const resp = await axios.get(`/api/line-admin/lines/${lineId}/apps`);
      if (resp.data.code === 0) {
        setAppsMap(prev => ({ ...prev, [lineId]: resp.data.data || [] }));
      }
    } catch (error) {
      message.error('获取应用列表失败');
    }
  };

  const fetchAllApps = async () => {
    setAppsLoading(true);
    for (const line of myLines) {
      await fetchApps(line.id);
    }
    setAppsLoading(false);
  };

  useEffect(() => {
    if (myLines.length > 0) {
      fetchAllApps();
      fetchAllMembers();
    }
  }, [myLines]);

  const fetchMembers = async (lineId: string) => {
    try {
      const resp = await axios.get(`/api/line-admin/lines/${lineId}/members`);
      if (resp.data.code === 0) {
        setMembersMap(prev => ({ ...prev, [lineId]: resp.data.data || [] }));
      }
    } catch (error) {
      message.error('获取成员列表失败');
    }
  };

  const fetchAllMembers = async () => {
    setMembersLoading(true);
    for (const line of myLines) {
      await fetchMembers(line.id);
    }
    setMembersLoading(false);
  };

  const handleAddMember = async () => {
    if (!currentLine) return;
    try {
      const values = await memberForm.validateFields();
      const resp = await axios.post(`/api/line-admin/lines/${currentLine.id}/members`, {
        user_id: values.selectedUser,
      });
      if (resp.data.code === 0) {
        message.success('成员添加成功');
        setMemberModal(false);
        memberForm.resetFields();
        fetchMembers(currentLine.id);
      } else {
        message.error(resp.data.message || '添加失败');
      }
    } catch (error) {
      message.error('请选择用户');
    }
  };

  const handleRemoveMember = async (line: BusinessLine, userId: string) => {
    try {
      const resp = await axios.delete(`/api/line-admin/lines/${line.id}/members/${userId}`);
      if (resp.data.code === 0) {
        message.success('成员移除成功');
        fetchMembers(line.id);
      } else {
        message.error(resp.data.message || '移除失败');
      }
    } catch (error) {
      message.error('移除失败');
    }
  };

  const searchMemberUsers = async (keyword: string) => {
    if (memberSearchTimerRef.current) {
      clearTimeout(memberSearchTimerRef.current);
    }

    if (!keyword) {
      setMemberUserOptions([]);
      return;
    }

    memberSearchTimerRef.current = setTimeout(async () => {
      setMemberSearchLoading(true);
      try {
        const resp = await axios.get(`/api/users/search?keyword=${keyword}`);
        if (resp.data.code === 0) {
          setMemberUserOptions(resp.data.data || []);
        }
      } catch (error) {
        console.error('搜索用户失败');
      } finally {
        setMemberSearchLoading(false);
      }
    }, 300);
  };

  const handleOpenAddMember = (line: BusinessLine) => {
    setCurrentLine(line);
    memberForm.resetFields();
    setMemberUserOptions([]);
    setMemberModal(true);
  };

  const handleEditDesc = async () => {
    if (!currentLine) return;
    try {
      const values = await descForm.validateFields();
      const resp = await axios.put(`/api/line-admin/lines/${currentLine.id}`, {
        description: values.description,
      });
      if (resp.data.code === 0) {
        message.success('描述更新成功');
        setEditDescModal(false);
        descForm.resetFields();
        fetchMyLines();
      } else {
        message.error(resp.data.message || '更新失败');
      }
    } catch (error) {
      message.error('请检查输入');
    }
  };

  const handleCreateApp = async () => {
    if (!currentLine) return;
    try {
      const values = await appForm.validateFields();
      const resp = await axios.post(`/api/line-admin/lines/${currentLine.id}/apps`, values);
      if (resp.data.code === 0) {
        message.success('应用创建成功');
        setCreateAppModal(false);
        appForm.resetFields();
        fetchApps(currentLine.id);
      } else {
        message.error(resp.data.message || '创建失败');
      }
    } catch (error) {
      message.error('请检查输入');
    }
  };

  const handleEditApp = async () => {
    if (!currentLine || !currentApp) return;
    try {
      const values = await editAppForm.validateFields();
      const resp = await axios.put(`/api/line-admin/lines/${currentLine.id}/apps/${currentApp.id}`, values);
      if (resp.data.code === 0) {
        message.success('应用更新成功');
        setEditAppModal(false);
        editAppForm.resetFields();
        fetchApps(currentLine.id);
      } else {
        message.error(resp.data.message || '更新失败');
      }
    } catch (error) {
      message.error('请检查输入');
    }
  };

  const handleDeleteApp = async (line: BusinessLine, app: AppInfo) => {
    try {
      // 软删除：仅修改 status 标记
      const newStatus = app.status === 'deleted' ? 'active' : 'deleted';
      const resp = await axios.put(`/api/line-admin/lines/${line.id}/apps/${app.id}`, { status: newStatus });
      if (resp.data.code === 0) {
        message.success(newStatus === 'deleted' ? '应用已下线' : '应用已恢复');
        fetchApps(line.id);
      } else {
        message.error(resp.data.message || '操作失败');
      }
    } catch (error) {
      message.error('操作失败');
    }
  };

  const handleOpenCreateApp = (line: BusinessLine) => {
    setCurrentLine(line);
    appForm.resetFields();
    setCreateAppModal(true);
  };

  const handleOpenEditApp = (line: BusinessLine, app: AppInfo) => {
    setCurrentLine(line);
    setCurrentApp(app);
    editAppForm.setFieldsValue({ app_key: app.app_key, description: app.description, settings: app.settings || '' });
    setEditAppModal(true);
  };

  const handleOpenEditDesc = (line: BusinessLine) => {
    setCurrentLine(line);
    descForm.setFieldsValue({ description: line.description });
    setEditDescModal(true);
  };

  return (
    <div>
      {loading ? (
        <div style={{ textAlign: 'center', padding: 50 }}>加载中...</div>
      ) : myLines.length === 0 ? (
        <Empty description="您暂无管理的业务线" />
      ) : (
        myLines.map(line => (
          <LineCard
            key={line.id}
            line={line}
            appsMap={appsMap}
            appsLoading={appsLoading}
            onEditDesc={handleOpenEditDesc}
            onCreateApp={handleOpenCreateApp}
            onEditApp={handleOpenEditApp}
            onDeleteApp={handleDeleteApp}
            onRefreshApps={fetchApps}
            onAddMember={handleOpenAddMember}
            onRemoveMember={handleRemoveMember}
            membersMap={membersMap}
            membersLoading={membersLoading}
          />
        ))
      )}

      <Modal title="修改业务线描述" open={editDescModal} onOk={handleEditDesc} onCancel={() => setEditDescModal(false)}>
        <Form form={descForm} layout="vertical">
          <Form.Item name="description" label="描述" rules={[{ required: true, message: '请输入描述' }]}>
            <Input.TextArea rows={3} placeholder="业务线描述" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title="新建应用" open={createAppModal} onOk={handleCreateApp} onCancel={() => setCreateAppModal(false)}>
        <Form form={appForm} layout="vertical">
          <Form.Item name="app_key" label="应用标识" rules={[{ required: true, message: '请输入应用标识' }, { pattern: /^[a-zA-Z0-9_]{3,50}$/, message: '只能包含英文、数字、下划线，长度3-50字符' }]}>
            <Input placeholder="例如：app_center" />
          </Form.Item>
          <Form.Item name="description" label="应用描述" rules={[{ required: true, message: '请输入应用描述' }]}>
            <Input.TextArea rows={3} placeholder="应用描述（中文）" />
          </Form.Item>
          <Form.Item
            name="settings"
            label="etcd配置"
            tooltip="要发布到 etcd 的配置（JSON 格式）"
            rules={[
              { required: true, message: '请输入配置' },
              { validator: (_, value) => {
                if (!value) return Promise.resolve();
                try { JSON.parse(value); return Promise.resolve(); }
                catch { return Promise.reject('请输入有效的JSON格式'); }
              }}
            ]}
          >
            <Input.TextArea rows={4} placeholder='{"env": "prod", "url": "etcd://127.0.0.1:2379"}' />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title="修改应用" open={editAppModal} onOk={handleEditApp} onCancel={() => setEditAppModal(false)}>
        <Form form={editAppForm} layout="vertical">
          <Form.Item name="app_key" label="应用标识" rules={[{ required: true, message: '请输入应用标识' }, { pattern: /^[a-zA-Z0-9_]{3,50}$/, message: '只能包含英文、数字、下划线，长度3-50字符' }]}>
            <Input placeholder="例如：app_center" />
          </Form.Item>
          <Form.Item name="description" label="应用描述" rules={[{ required: true, message: '请输入应用描述' }]}>
            <Input.TextArea rows={3} placeholder="应用描述（中文）" />
          </Form.Item>
          <Form.Item
            name="settings"
            label="etcd配置"
            tooltip="要发布到 etcd 的配置（JSON 格式）"
            rules={[
              { required: true, message: '请输入配置' },
              { validator: (_, value) => {
                if (!value) return Promise.resolve();
                try { JSON.parse(value); return Promise.resolve(); }
                catch { return Promise.reject('请输入有效的JSON格式'); }
              }}
            ]}
          >
            <Input.TextArea rows={4} placeholder='{"env": "prod", "url": "etcd://127.0.0.1:2379"}' />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title="添加成员" open={memberModal} onOk={handleAddMember} onCancel={() => setMemberModal(false)}>
        <Form form={memberForm} layout="vertical">
          <Form.Item name="selectedUser" rules={[{ required: true, message: '请选择用户' }]}>
            <Select
              placeholder="输入姓名或邮箱搜索并选择"
              loading={memberSearchLoading}
              showSearch
              filterOption={false}
              onSearch={searchMemberUsers}
              style={{ width: '100%' }}
              notFoundContent={memberSearchLoading ? '搜索中...' : '输入关键词搜索用户'}
            >
              {memberUserOptions.map(user => (
                <Select.Option key={user.user_id} value={user.user_id}>
                  {user.name} ({user.email})
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

const ACTION_LABELS: Record<string, string> = {
  rule_create: '创建规则',
  rule_update: '更新规则',
  rule_delete: '删除规则',
  rule_toggle: '启用/禁用规则',
  publish: '发布规则',
  rollback: '回滚版本',
  resource_delete: '删除资源',
};

const AuditLogPanel: React.FC = () => {
  const [logs, setLogs] = useState<Record<string, string>[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [loading, setLoading] = useState(false);
  const [actionFilter, setActionFilter] = useState('');

  const fetchLogs = async (p: number, action: string) => {
    setLoading(true);
    try {
      let url = `/api/admin/audit-logs?page=${p}&page_size=${pageSize}`;
      if (action) url += `&action=${action}`;
      const resp = await axios.get(url);
      if (resp.data.code === 0 || !resp.data.code) {
        const data = resp.data.data || {};
        setLogs(data.items || []);
        setTotal(data.total || 0);
      }
    } catch {
      message.error('获取审计日志失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchLogs(page, actionFilter);
  }, [page, actionFilter]);

  const columns = [
    { title: '操作人', dataIndex: 'user_name', key: 'user_name', width: 100, render: (v: string, r: Record<string, string>) => v || r.user_id },
    { title: '操作类型', dataIndex: 'action', key: 'action', width: 120, render: (v: string) => <Tag color="blue">{ACTION_LABELS[v] || v}</Tag> },
    { title: '资源类型', dataIndex: 'resource_type', key: 'resource_type', width: 100 },
    { title: '资源ID', dataIndex: 'resource_id', key: 'resource_id', width: 100 },
    { title: '变更详情', dataIndex: 'detail', key: 'detail', ellipsis: true },
    { title: 'IP地址', dataIndex: 'ip_address', key: 'ip_address', width: 130 },
    { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 180, render: (v: string) => v ? new Date(v).toLocaleString('zh-CN') : '-' },
  ];

  return (
    <Card title="审计日志" style={{ marginTop: 16 }}>
      <Space style={{ marginBottom: 16 }}>
        <Select
          style={{ width: 200 }}
          placeholder="筛选操作类型"
          allowClear
          value={actionFilter || undefined}
          onChange={(v) => { setActionFilter(v || ''); setPage(1); }}
        >
          {Object.entries(ACTION_LABELS).map(([k, v]) => (
            <Select.Option key={k} value={k}>{v}</Select.Option>
          ))}
        </Select>
      </Space>
      <Table
        rowKey="id"
        dataSource={logs}
        columns={columns}
        loading={loading}
        pagination={false}
        size="small"
        locale={{ emptyText: <Empty description="暂无审计日志" /> }}
      />
      {total > pageSize && (
        <div style={{ textAlign: 'right', marginTop: 16 }}>
          <Pagination current={page} total={total} pageSize={pageSize} onChange={(p) => setPage(p)} showTotal={(t) => `共 ${t} 条`} />
        </div>
      )}
    </Card>
  );
};

const Admin: React.FC = () => {
  const [currentUser, setCurrentUser] = useState<CurrentUser | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchCurrentUser = async () => {
      try {
        const resp = await axios.get('/api/auth/me');
        if (resp.data.code === 0) {
          setCurrentUser(resp.data.data);
        }
      } catch (error) {
        console.error('获取用户信息失败');
      } finally {
        setLoading(false);
      }
    };
    fetchCurrentUser();
  }, []);

  if (loading) {
    return <div style={{ padding: 24, textAlign: 'center' }}>加载中...</div>;
  }

  if (!currentUser) {
    return <div style={{ padding: 24, textAlign: 'center' }}>未登录或会话已过期</div>;
  }

  const isSuperAdmin = currentUser.role === 'super_admin';
  const isLineAdmin = currentUser.role === 'line_admin';

  return (
    <div style={{ padding: 24 }}>
      {isSuperAdmin && <SuperAdminPanel />}
      {isSuperAdmin && <AuditLogPanel />}
      {isLineAdmin && <LineAdminPanel currentUser={currentUser} />}
      {!isSuperAdmin && !isLineAdmin && (
        <Card>
          <Empty description="您没有管理权限" />
        </Card>
      )}
    </div>
  );
};

export default Admin;
