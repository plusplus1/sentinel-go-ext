import React, { useEffect, useState, useCallback } from 'react';
import { Table, Button, message, Spin, Modal, Form, Input, InputNumber, Select, Space, Card, Empty, Tag, Typography, Switch, Popconfirm, Tooltip, Descriptions, Badge, Row, Col } from 'antd';
import { CloudServerOutlined, ReloadOutlined, SafetyOutlined, EditOutlined, PlusOutlined, DeleteOutlined, SendOutlined, RollbackOutlined, ThunderboltOutlined, QuestionCircleOutlined, BranchesOutlined } from '@ant-design/icons';
import axios from 'axios';
import { useApp } from '../context/AppContext';

interface Resource {
  id: string;
  name: string;
  description: string;
  app_id: string;
  env: string;
  group_id: string;
  group_name: string;
  group_description: string;
  flow_rule_count?: number;
  cb_rule_count?: number;
  last_publish_at?: string;
  running_version?: number;
  latest_version?: number;
  created_at: string;
  updated_at: string;
}

interface FlowRule {
  id: number;
  resource_id: number;
  threshold: number;
  controlBehavior: number;
  warmUpPeriodSec?: number;
  maxQueueingTimeMs?: number;
  clusterMode?: boolean;
  metricType?: number;
  tokenCalculateStrategy?: number;
  warmUpColdFactor?: number;
  statIntervalInMs?: number;
  enabled?: boolean;
}

interface CBRule {
  id: number;
  resource_id: number;
  strategy: number;
  retryTimeoutMs: number;
  minRequestAmount: number;
  statIntervalMs?: number;
  maxAllowedRtMs?: number;
  threshold?: number;
  probeNum?: number;
  enabled?: boolean;
}

interface Module {
  id: string;
  name: string;
  is_default: boolean;
}

interface VersionRecord {
  id: string;
  version_number: number;
  description: string;
  rule_count: number;
  created_at: string;
}

interface VersionSnapshot {
  flow_rules: Array<{
    rule_id: number;
    resource: string;
    threshold: number;
    [key: string]: unknown;
  }>;
  circuit_breaker_rules: Array<{
    rule_id: number;
    resource: string;
    strategy: number;
    threshold: number;
    [key: string]: unknown;
  }>;
  [key: string]: unknown;
}

const Resources: React.FC = () => {
  const { selectedApp, getAppFullPath } = useApp();
  const [resources, setResources] = useState<Resource[]>([]);
  const [loading, setLoading] = useState(false);

  // Modules for resource creation
  const [modules, setModules] = useState<Module[]>([]);

  // Rules detail modal
  const [rulesVisible, setRulesVisible] = useState(false);
  const [rulesModalTitle, setRulesModalTitle] = useState('');
  const [selectedResource, setSelectedResource] = useState<Resource | null>(null);
  const [flowRules, setFlowRules] = useState<FlowRule[]>([]);
  const [cbRules, setCbRules] = useState<CBRule[]>([]);
  const [rulesLoading, setRulesLoading] = useState(false);
  const [rulesFilter, setRulesFilter] = useState<'flow' | 'cb' | undefined>(undefined);

  // Flow rule edit modal
  const [flowEditVisible, setFlowEditVisible] = useState(false);
  const [editingFlowRule, setEditingFlowRule] = useState<FlowRule | null>(null);
  const [flowForm] = Form.useForm();

  // Circuit breaker rule edit modal
  const [cbEditVisible, setCbEditVisible] = useState(false);
  const [editingCbRule, setEditingCbRule] = useState<CBRule | null>(null);
  const [cbForm] = Form.useForm();
  const flowControlBehavior = Form.useWatch('controlBehavior', flowForm);
  const flowTokenCalculateStrategy = Form.useWatch('tokenCalculateStrategy', flowForm);
  const flowMetricType = Form.useWatch('metricType', flowForm);
  const cbStrategy = Form.useWatch('strategy', cbForm);

  // Add resource modal
  const [addResourceVisible, setAddResourceVisible] = useState(false);
  const [addResourceForm] = Form.useForm();

  // Publish preview modal (per-resource)
  const [publishPreviewVisible, setPublishPreviewVisible] = useState(false);
  const [publishPreviewLoading, setPublishPreviewLoading] = useState(false);
  const [publishingResource, setPublishingResource] = useState<Resource | null>(null);
  const [publishFlowRules, setPublishFlowRules] = useState<FlowRule[]>([]);
  const [publishCbRules, setPublishCbRules] = useState<CBRule[]>([]);
  const [publishRulesLoading, setPublishRulesLoading] = useState(false);
  const [publishedFlowRules, setPublishedFlowRules] = useState<FlowRule[]>([]);
  const [publishedCbRules, setPublishedCbRules] = useState<CBRule[]>([]);

  // Version management (per-resource)
  const [versionsVisible, setVersionsVisible] = useState(false);
  const [versionsResource, setVersionsResource] = useState<Resource | null>(null);
  const [versions, setVersions] = useState<VersionRecord[]>([]);
  const [versionDetailVisible, setVersionDetailVisible] = useState(false);
  const [selectedVersion, setSelectedVersion] = useState<VersionRecord | null>(null);
  const [versionSnapshot, setVersionSnapshot] = useState<VersionSnapshot | null>(null);

  // Diff data for publish preview
  const [diffData, setDiffData] = useState<any>(null);

  const fetchResources = useCallback(async (appId: string) => {
    if (!appId) return;
    setLoading(true);
    try {
      const response = await axios.get('/api/resources', {
        params: { app: appId },
      });
      const { code, data, message: msg } = response.data;
      if (code === 0 && data) {
        setResources(data);
        // Fetch rule counts for each resource
        fetchRuleCounts(appId, data);
      } else {
        message.error(msg || '获取资源列表失败');
      }
    } catch (error) {
      message.error('请求失败: ' + (error as Error).message);
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchRuleCounts = async (appId: string, resourceList: Resource[]) => {
    const promises = resourceList.map(async (r) => {
      try {
        const resp = await axios.get(`/api/resource/${r.id}/rules`, {
          params: { app: appId },
        });
        if (resp.data.code === 0 && resp.data.data) {
          return {
            id: r.id,
            flow_count: (resp.data.data.flow_rules || []).length,
            cb_count: (resp.data.data.circuit_breaker_rules || []).length,
          };
        }
      } catch (e) { console.warn('fetchRuleCounts failed for', r.id, e); }
      return { id: r.id, flow_count: 0, cb_count: 0 };
    });
    const counts = await Promise.all(promises);
    const countMap = new Map(counts.map(c => [c.id, c]));
    setResources(prev => prev.map(r => {
      const c = countMap.get(r.id);
      return c ? { ...r, flow_rule_count: c.flow_count, cb_rule_count: c.cb_count } : r;
    }));
  };

  const fetchModules = async (appId: string) => {
    try {
      const response = await axios.get('/api/groups', {
        params: { app: appId },
      });
      const { code, data } = response.data;
      if (code === 0 && data) {
        setModules(data);
      }
    } catch (e) { console.warn('fetchModules failed', e); }
  };

  const fetchRules = async (resourceId: string | number, filterType?: 'flow' | 'cb') => {
    if (!selectedApp) return;
    setRulesLoading(true);
    try {
      const response = await axios.get(`/api/resource/${resourceId}/rules`, {
        params: { app: selectedApp.id },
      });
      const { code, data } = response.data;
      if (code === 0 && data) {
        // Filter rules based on filterType
        if (filterType === 'flow') {
          setFlowRules(data.flow_rules || []);
          setCbRules([]);
        } else if (filterType === 'cb') {
          setCbRules(data.circuit_breaker_rules || []);
          setFlowRules([]);
        } else {
          setFlowRules(data.flow_rules || []);
          setCbRules(data.circuit_breaker_rules || []);
        }
      }
    } catch (error) {
      message.error('获取规则失败');
    } finally {
      setRulesLoading(false);
    }
  };

  useEffect(() => {
    if (selectedApp) {
      fetchResources(selectedApp.id);
      fetchModules(selectedApp.id);
    } else {
      setResources([]);
    }
  }, [selectedApp, fetchResources]);


  const handleViewRules = (record: Resource, filterType?: 'flow' | 'cb') => {
    setSelectedResource(record);
    setFlowRules([]);
    setCbRules([]);
    if (filterType === 'flow') {
      setRulesModalTitle(`${record.name} - 流控规则`);
    } else if (filterType === 'cb') {
      setRulesModalTitle(`${record.name} - 熔断规则`);
    } else {
      setRulesModalTitle(`${record.name} - 规则详情`);
    }
    setRulesFilter(filterType);
    setRulesVisible(true);

    // Fetch all rules first, then filter based on filterType
    fetchRules(record.id, filterType);
  };

  const handleToggleRule = async (resourceId: string | number, ruleId: number, type: 'flow' | 'circuitbreaker') => {
    try {
      const url = `/api/rule/${type}/${ruleId}/toggle?resource=${resourceId}&app=${selectedApp?.id}`;
      const response = await axios.put(url);
      const { code, message: msg } = response.data;
      if (code === 0) {
        message.success('切换成功');
        fetchRules(resourceId);
      } else {
        message.error(msg || '切换失败');
      }
    } catch (error) {
      message.error('切换失败: ' + (error as Error).message);
    }
  };

  // Flow rule edit handlers
  const handleEditFlowRule = (rule: FlowRule) => {
    setEditingFlowRule(rule);
    flowForm.setFieldsValue({
      resource_id: rule.resource_id,
      threshold: rule.threshold,
      controlBehavior: rule.controlBehavior,
      warmUpPeriodSec: rule.warmUpPeriodSec || 0,
      maxQueueingTimeMs: rule.maxQueueingTimeMs || 0,
      clusterMode: rule.clusterMode || false,
      metricType: rule.metricType ?? 0,
      tokenCalculateStrategy: rule.tokenCalculateStrategy ?? 0,
      warmUpColdFactor: rule.warmUpColdFactor ?? 3,
      statIntervalInMs: rule.statIntervalInMs ?? 1000,
    });
    setFlowEditVisible(true);
  };

  const handleCreateFlowRule = () => {
    setEditingFlowRule(null);
    flowForm.resetFields();
    flowForm.setFieldsValue({
      resource_id: selectedResource?.id || '',
      threshold: 100,
      controlBehavior: 0,
      warmUpPeriodSec: 10,
      maxQueueingTimeMs: 500,
      clusterMode: false,
      metricType: 1,
      tokenCalculateStrategy: 0,
      warmUpColdFactor: 3,
      statIntervalInMs: 1000,
    });
    setFlowEditVisible(true);
  };

  const handleSaveFlowRule = async () => {
    if (!selectedResource || !selectedApp) return;
    try {
      const values = await flowForm.validateFields();
      const payload = {
        appId: selectedApp.id,
        id: editingFlowRule?.id || 0,
        resource_id: Number(selectedResource.id),
        threshold: values.threshold,
        controlBehavior: values.controlBehavior,
        warmUpPeriodSec: values.warmUpPeriodSec || 0,
        maxQueueingTimeMs: values.maxQueueingTimeMs || 0,
        clusterMode: values.clusterMode || false,
        metricType: values.metricType ?? 0,
        tokenCalculateStrategy: values.tokenCalculateStrategy ?? 0,
        relationStrategy: 0,
        refResource: '',
        warmUpColdFactor: values.warmUpColdFactor ?? 3,
        statIntervalInMs: values.statIntervalInMs ?? 1000,
        lowMemUsageThreshold: 0.8,
        highMemUsageThreshold: 0.9,
        memLowWaterMarkBytes: 0,
        memHighWaterMarkBytes: 0,
      };
      const response = await axios.post('/api/app/rule/flow/update', payload);
      const { code, message: msg } = response.data;
      if (code === 0) {
        message.success(editingFlowRule ? '流控规则更新成功' : '流控规则创建成功');
        setFlowEditVisible(false);
        fetchRules(selectedResource.id);
        fetchResources(selectedApp.id);
      } else {
        message.error(msg || '操作失败');
      }
    } catch (error) {
      message.error('操作失败: ' + (error as Error).message);
    }
  };

  // Circuit breaker rule edit handlers
  const handleEditCbRule = (rule: CBRule) => {
    setEditingCbRule(rule);
    cbForm.setFieldsValue({
      resource_id: rule.resource_id,
      strategy: rule.strategy,
      retryTimeoutMs: rule.retryTimeoutMs,
      minRequestAmount: rule.minRequestAmount,
      statIntervalMs: rule.statIntervalMs || 1000,
      maxAllowedRtMs: rule.maxAllowedRtMs || 200,
      threshold: rule.threshold || 0.5,
      probeNum: rule.probeNum || 0,
    });
    setCbEditVisible(true);
  };

  const handleCreateCbRule = () => {
    setEditingCbRule(null);
    cbForm.resetFields();
    cbForm.setFieldsValue({
      resource_id: selectedResource?.id || '',
      strategy: 0,
      retryTimeoutMs: 10000,
      minRequestAmount: 5,
      statIntervalMs: 1000,
      maxAllowedRtMs: 200,
      threshold: 0.5,
      probeNum: 0,
    });
    setCbEditVisible(true);
  };

  const handleSaveCbRule = async () => {
    if (!selectedResource || !selectedApp) return;
    try {
      const values = await cbForm.validateFields();
      const payload = {
        appId: selectedApp.id,
        id: editingCbRule?.id || 0,
        resource_id: Number(selectedResource.id),
        strategy: values.strategy,
        retryTimeoutMs: values.retryTimeoutMs,
        minRequestAmount: values.minRequestAmount,
        statIntervalMs: values.statIntervalMs || 1000,
        maxAllowedRtMs: values.maxAllowedRtMs || 200,
        threshold: values.threshold || 0.5,
        probeNum: values.probeNum || 0,
      };
      const response = await axios.post('/api/app/rule/circuitbreaker/update', payload);
      const { code, message: msg } = response.data;
      if (code === 0) {
        message.success(editingCbRule ? '熔断规则更新成功' : '熔断规则创建成功');
        setCbEditVisible(false);
        fetchRules(selectedResource.id);
        fetchResources(selectedApp.id);
      } else {
        message.error(msg || '操作失败');
      }
    } catch (error) {
      message.error('操作失败: ' + (error as Error).message);
    }
  };

  // Add resource
  const handleAddResource = async () => {
    if (!selectedApp) return;
    try {
      const values = await addResourceForm.validateFields();
      const response = await axios.put(`/api/resource/${values.resource_name}`, {
        description: values.description || '',
        group_id: values.module_id,
      }, {
        params: { app: selectedApp.id },
      });
      const { code, message: msg } = response.data;
      if (code === 0) {
        message.success('资源添加成功');
        setAddResourceVisible(false);
        addResourceForm.resetFields();
        fetchResources(selectedApp.id);
      } else {
        message.error(msg || '添加失败');
      }
    } catch (error) {
      message.error('添加失败: ' + (error as Error).message);
    }
  };

  // Delete resource
  const handleDeleteResource = async (resourceId: string) => {
    if (!selectedApp) return;
    try {
      const response = await axios.delete('/api/resource', {
        params: { app: selectedApp.id, id: resourceId },
      });
      const { code, message: msg } = response.data;
      if (code === 0) {
        message.success('资源删除成功');
        fetchResources(selectedApp.id);
      } else {
        message.error(msg || '删除失败');
      }
    } catch (error) {
      message.error('删除失败: ' + (error as Error).message);
    }
  };

  // Open publish preview modal for a specific resource
  const handleOpenPublishPreview = async (record: Resource) => {
    if (!selectedApp) return;
    setPublishingResource(record);
    setPublishFlowRules([]);
    setPublishCbRules([]);
    setPublishedFlowRules([]);
    setPublishedCbRules([]);
    setDiffData(null);
    setPublishPreviewVisible(true);
    setPublishRulesLoading(true);
    try {
      const [rulesResp, versionsResp, diffResp] = await Promise.all([
        axios.get(`/api/resource/${record.id}/rules`, { params: { app: selectedApp.id } }),
        axios.get('/api/versions', { params: { app: selectedApp.id } }),
        axios.get(`/api/resource/${record.id}/diff`, { params: { app: selectedApp.id } }).catch(() => null),
      ]);
      const { code, data } = rulesResp.data;
      if (code === 0 && data) {
        setPublishFlowRules(data.flow_rules || []);
        setPublishCbRules(data.circuit_breaker_rules || []);
      }
      const vData = versionsResp.data;
      if (vData.code === 0 && vData.data && vData.data.length > 0) {
        const latestVersionId = vData.data[0].id;
        const detailResp = await axios.get(`/api/versions/${latestVersionId}`);
        if (detailResp.data.code === 0 && detailResp.data.data?.snapshot) {
          const snap = detailResp.data.data.snapshot;
          const resName = record.name;
          setPublishedFlowRules((snap.flow_rules || []).filter((r: FlowRule) => (r as FlowRule & {resource?: string}).resource === resName));
          setPublishedCbRules((snap.circuit_breaker_rules || []).filter((r: CBRule) => (r as CBRule & {resource?: string}).resource === resName));
        }
      }
      if (diffResp && diffResp.data && (diffResp.data.code === 0 || !diffResp.data.code)) {
        setDiffData(diffResp.data.data);
      }
    } catch {
      message.error('获取规则失败');
    } finally {
      setPublishRulesLoading(false);
    }
  };

  // Confirm publish for a specific resource
  const handleConfirmPublish = async () => {
    if (!selectedApp || !publishingResource) return;
    setPublishPreviewLoading(true);
    try {
      const response = await axios.post('/api/publish', {
        app_key: selectedApp.id,
        rule_type: 'all',
        resource: publishingResource.id,
      });
      const { code, message: msg } = response.data;
      if (code === 0) {
        message.success('发布成功！');
        setPublishPreviewVisible(false);
        fetchResources(selectedApp.id);
      } else {
        message.error(msg || '发布失败');
      }
    } catch (error) {
      message.error('发布失败: ' + (error as Error).message);
    } finally {
      setPublishPreviewLoading(false);
    }
  };

  // Fetch versions for a specific resource
  const fetchVersions = async () => {
    if (!selectedApp) return;
    try {
      const response = await axios.get('/api/versions', {
        params: { app: selectedApp.id },
      });
      const { code, data } = response.data;
      if (code === 0 && data) {
        setVersions(data);
      }
    } catch (e) { console.warn('fetchVersions failed', e); }
  };

  const handleOpenVersions = (record: Resource) => {
    setVersionsResource(record);
    setVersionsVisible(true);
    fetchVersions();
  };

  // View version detail
  const handleViewVersion = async (versionId: string) => {
    try {
      const response = await axios.get(`/api/versions/${versionId}`);
      const { code, data } = response.data;
      if (code === 0 && data) {
        setSelectedVersion(data.version);
        setVersionSnapshot(data.snapshot);
        setVersionDetailVisible(true);
      }
    } catch {
      message.error('获取版本详情失败');
    }
  };

  // Rollback to version
  const handleRollback = async (versionId: string, versionNumber: number) => {
    if (!selectedApp) return;
    Modal.confirm({
      title: '确认回滚',
      content: `确定要回滚到 v${versionNumber} 吗？当前规则将被替换为该版本的快照。`,
      onOk: async () => {
        try {
          const response = await axios.post(`/api/versions/${versionId}/rollback`, {
            app_key: selectedApp.id,
          });
          const { code, message: msg } = response.data;
          if (code === 0) {
            message.success(msg || '回滚成功');
            setVersionDetailVisible(false);
            fetchResources(selectedApp.id);
          } else {
            message.error(msg || '回滚失败');
          }
        } catch (error) {
          message.error('回滚失败: ' + (error as Error).message);
        }
      },
    });
  };

  // Helper: check if resource was published today
  const isPublishedToday = (publishAt?: string): boolean => {
    if (!publishAt) return false;
    const publishDate = new Date(publishAt);
    const today = new Date();
    return publishDate.getFullYear() === today.getFullYear() &&
      publishDate.getMonth() === today.getMonth() &&
      publishDate.getDate() === today.getDate();
  };

  // Helper: render strategy tag for CB rules
  const renderCbStrategy = (strategy: number) => {
    const map: Record<number, { label: string; color: string }> = {
      0: { label: '慢调用比例', color: 'orange' },
      1: { label: '错误比例', color: 'red' },
      2: { label: '错误计数', color: 'volcano' },
    };
    const s = map[strategy] || { label: '未知', color: 'default' };
    return <Tag color={s.color}>{s.label}</Tag>;
  };

  const changedStyle: React.CSSProperties = {
    backgroundColor: '#fffbe6',
    borderLeft: '3px solid #faad14',
    paddingLeft: 8,
    borderRadius: 4,
  };

  const isFieldChanged = (ruleType: 'flow' | 'cb', fieldName: string): boolean => {
    if (!diffData) return false;
    const diffs = ruleType === 'flow' ? diffData.flow_diffs : diffData.cb_diffs;
    if (!diffs) return false;
    return diffs.some((d: any) => d.field === fieldName && d.changed);
  };

  const columns = [
    {
      title: '资源名称',
      dataIndex: 'name',
      key: 'name',
      width: 200,
      render: (val: string, record: Resource) => (
        <Space direction="vertical">
          <Space>
            <CloudServerOutlined />
            <Typography.Text strong>{val}</Typography.Text>
          </Space>
          {record.description && (
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {record.description}
            </Typography.Text>
          )}
        </Space>
      ),
    },
    {
      title: '所属模块',
      dataIndex: 'group_name',
      key: 'group_name',
      render: (val: string, record: Resource) => (
        <Tooltip
          title={
            val ? (
              <div style={{ whiteSpace: 'pre-line' }}>
                <div>模块ID: {record.group_id}</div>
                {record.group_description && <div>描述: {record.group_description}</div>}
              </div>
            ) : '未分配到任何模块'
          }
        >
          <Tag color={val ? 'blue' : 'default'}>
            {val || '未分配模块'}
          </Tag>
        </Tooltip>
      ),
    },
    {
      title: <span><ThunderboltOutlined /> 流控</span>,
      dataIndex: 'flow_rule_count',
      key: 'flow_rule_count',
      width: 120,
      render: (count: number, record: Resource) => (
        count > 0 ? (
          <Button type="link" size="small" onClick={() => handleViewRules(record, 'flow')} style={{ padding: 0, height: 'auto' }}>
            <Badge status="processing" color="blue" />
            <span style={{ marginLeft: 4 }}>已配置</span>
          </Button>
        ) : (
          <Button type="dashed" size="small" onClick={() => handleViewRules(record, 'flow')}>
            <PlusOutlined /> 添加
          </Button>
        )
      ),
    },
    {
      title: <span><SafetyOutlined /> 熔断</span>,
      dataIndex: 'cb_rule_count',
      key: 'cb_rule_count',
      width: 120,
      render: (count: number, record: Resource) => (
        count > 0 ? (
          <Button type="link" size="small" onClick={() => handleViewRules(record, 'cb')} style={{ padding: 0, height: 'auto' }}>
            <Badge status="processing" color="orange" />
            <span style={{ marginLeft: 4 }}>已配置</span>
          </Button>
        ) : (
          <Button type="dashed" size="small" onClick={() => handleViewRules(record, 'cb')}>
            <PlusOutlined /> 添加
          </Button>
        )
      ),
    },
    {
      title: '更新时间',
      dataIndex: 'updated_at',
      key: 'updated_at',
      width: 180,
      render: (val: string) => val ? new Date(val).toLocaleString('zh-CN') : '-',
    },
    {
      title: '发布状态',
      key: 'publish_status',
      width: 140,
      render: (_: unknown, record: Resource) => {
        const published = isPublishedToday(record.last_publish_at);
        const version = record.running_version ?? record.latest_version;
        if (published && version) {
          return (
            <Tooltip title={`发布时间: ${new Date(record.last_publish_at!).toLocaleString('zh-CN')}`}>
              <Badge status="success" />
              <Tag color="green">v{version}</Tag>
            </Tooltip>
          );
        }
        if (version) {
          return (
            <Tooltip title={record.last_publish_at ? `上次发布: ${new Date(record.last_publish_at).toLocaleString('zh-CN')}` : undefined}>
              <Tag color="default">v{version}</Tag>
            </Tooltip>
          );
        }
        return <Typography.Text type="secondary">未发布</Typography.Text>;
      },
    },
    {
      title: '操作',
      key: 'action',
      width: 180,
      render: (_: unknown, record: Resource) => (
        <Space>
          <Button
            type="link"
            size="small"
            icon={<SendOutlined />}
            onClick={() => handleOpenPublishPreview(record)}
            style={{ padding: 0 }}
          >
            发布
          </Button>
          <Button
            type="link"
            size="small"
            icon={<BranchesOutlined />}
            onClick={() => handleOpenVersions(record)}
            style={{ padding: 0 }}
          >
            版本
          </Button>
          <Popconfirm
            title="确认删除"
            description={`确定删除资源 "${record.name}" 吗？此操作不可恢复。`}
            onConfirm={() => handleDeleteResource(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="link" danger size="small" icon={<DeleteOutlined />} style={{ padding: 0 }}>
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
            <CloudServerOutlined style={{ marginRight: 8 }} />
            资源中心
          </h2>
          <p style={{ margin: '8px 0 0 0', fontSize: 14, color: 'rgba(0, 0, 0, 0.45)' }}>
            应用: <Tag color="blue">{getAppFullPath(selectedApp.id)}</Tag> | 共 {resources.length} 个资源
          </p>
        </div>
        <Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => fetchResources(selectedApp.id)}
          >
            刷新
          </Button>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setAddResourceVisible(true)}
          >
            添加资源
          </Button>
        </Space>
      </div>

      <Card>
        <Spin spinning={loading}>
          <Table
            rowKey="resource"
            dataSource={resources}
            columns={columns}
            pagination={{ pageSize: 10 }}
          />
        </Spin>
      </Card>

      {/* Add Resource Modal */}
      <Modal
        title="添加资源"
        open={addResourceVisible}
        onOk={handleAddResource}
        onCancel={() => setAddResourceVisible(false)}
        destroyOnClose
      >
        <Form form={addResourceForm} layout="vertical">
          <Form.Item
            name="resource_name"
            label="资源名称"
            rules={[
              { required: true, message: '请输入资源名称' },
              { pattern: /^[a-zA-Z0-9_/:.\-]+$/, message: '仅支持英文、数字、下划线、斜杠、冒号、点、短横线' },
            ]}
          >
            <Input placeholder="例如: API:GET:/users 或 my_service" />
          </Form.Item>
          <Form.Item name="description" label="资源描述">
            <Input.TextArea placeholder="输入资源描述（可选）" rows={3} />
          </Form.Item>
          <Form.Item name="module_id" label="分配到模块" rules={[{ required: true, message: '请选择模块' }]}>
            <Select placeholder="请选择模块">
              {modules.map(m => (
                <Select.Option key={m.id} value={m.id}>
                  {m.name} {m.is_default ? '(默认)' : ''}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
        </Form>
      </Modal>


      {/* Rules Detail Modal */}
      <Modal
        title={
          <Space>
            {rulesFilter === 'cb' ? <SafetyOutlined /> : <ThunderboltOutlined />}
            <span>{rulesModalTitle}</span>
          </Space>
        }
        open={rulesVisible}
        onCancel={() => setRulesVisible(false)}
        footer={null}
        width={640}
      >
        <Spin spinning={rulesLoading}>
          {/* Flow Rules Section */}
          {(rulesFilter === 'flow' || !rulesFilter) && (
          flowRules.length > 0 ? (
            flowRules.map((rule) => (
              <Card
                key={rule.id}
                size="small"
                style={{ marginBottom: 12 }}
                title={
                  <Space>
                    <ThunderboltOutlined style={{ color: '#1677ff' }} />
                    <span>流控规则</span>
                    <Tag color={rule.enabled !== false ? 'green' : 'default'}>{rule.enabled !== false ? '启用' : '禁用'}</Tag>
                  </Space>
                }
                extra={
                  <Space>
                    <Button size="small" onClick={() => handleToggleRule(selectedResource!.id, rule.id, 'flow')}>切换</Button>
                    <Button size="small" type="primary" ghost icon={<EditOutlined />} onClick={() => handleEditFlowRule(rule)}>编辑</Button>
                  </Space>
                }
              >
                <Descriptions column={2} size="small" labelStyle={{ color: 'rgba(0,0,0,.45)', width: 110 }}>
                  <Descriptions.Item label="指标类型">
                    <Tag color="blue">{rule.metricType === 1 ? 'QPS' : '并发数'}</Tag>
                  </Descriptions.Item>
                  <Descriptions.Item label="阈值">
                    <Typography.Text strong style={{ color: '#1677ff', fontSize: 16 }}>{rule.threshold}</Typography.Text>
                    <span style={{ color: 'rgba(0,0,0,.45)', marginLeft: 4 }}>{rule.metricType === 1 ? 'req/s' : '并发'}</span>
                  </Descriptions.Item>
                  <Descriptions.Item label="控制行为">
                    <Tag color={rule.controlBehavior === 0 ? 'red' : 'orange'}>{rule.controlBehavior === 0 ? '直接拒绝' : '匀速排队'}</Tag>
                  </Descriptions.Item>
                  <Descriptions.Item label="令牌策略">
                    <Tag color={rule.tokenCalculateStrategy === 0 ? 'default' : rule.tokenCalculateStrategy === 1 ? 'gold' : 'purple'}>
                      {rule.tokenCalculateStrategy === 0 ? 'Direct' : rule.tokenCalculateStrategy === 1 ? 'WarmUp' : 'MemoryAdaptive'}
                    </Tag>
                  </Descriptions.Item>
                  <Descriptions.Item label="统计窗口">{rule.statIntervalInMs ?? 1000} ms</Descriptions.Item>
                  {rule.controlBehavior === 1 && (
                    <Descriptions.Item label="最大排队">{rule.maxQueueingTimeMs} ms</Descriptions.Item>
                  )}
                  {(rule.tokenCalculateStrategy ?? 0) === 1 && (
                    <Descriptions.Item label="预热时长">{rule.warmUpPeriodSec} s</Descriptions.Item>
                  )}
                  {(rule.tokenCalculateStrategy ?? 0) === 1 && (
                    <Descriptions.Item label="冷启动因子">{rule.warmUpColdFactor ?? 3}</Descriptions.Item>
                  )}
                  {rule.clusterMode && (
                    <Descriptions.Item label="集群模式"><Tag color="green">已开启</Tag></Descriptions.Item>
                  )}
                </Descriptions>
              </Card>
            ))
          ) : (
            <Empty description="暂无流控规则" image={Empty.PRESENTED_IMAGE_SIMPLE}>
              <Button type="dashed" icon={<PlusOutlined />} onClick={() => handleCreateFlowRule()}>
                添加流控规则
              </Button>
            </Empty>
          )
          )}

          {/* Circuit Breaker Rules Section */}
          {(rulesFilter === 'cb' || !rulesFilter) && (
          cbRules.length > 0 ? (
            cbRules.map((rule) => (
              <Card
                key={rule.id}
                size="small"
                style={{ marginBottom: 12 }}
                title={
                  <Space>
                    <SafetyOutlined style={{ color: '#fa8c16' }} />
                    <span>熔断规则</span>
                    <Tag color={rule.enabled !== false ? 'green' : 'default'}>{rule.enabled !== false ? '启用' : '禁用'}</Tag>
                  </Space>
                }
                extra={
                  <Space>
                    <Button size="small" onClick={() => handleToggleRule(selectedResource!.id, rule.id, 'circuitbreaker')}>切换</Button>
                    <Button size="small" type="primary" ghost icon={<EditOutlined />} onClick={() => handleEditCbRule(rule)}>编辑</Button>
                  </Space>
                }
              >
                <Descriptions column={2} size="small" labelStyle={{ color: 'rgba(0,0,0,.45)', width: 110 }}>
                  <Descriptions.Item label="熔断策略">
                    <Tag color={rule.strategy === 0 ? 'orange' : rule.strategy === 1 ? 'red' : 'volcano'}>
                      {rule.strategy === 0 ? '慢调用比例' : rule.strategy === 1 ? '错误比例' : '错误计数'}
                    </Tag>
                  </Descriptions.Item>
                  <Descriptions.Item label="阈值">
                    <Typography.Text strong style={{ color: '#fa8c16', fontSize: 16 }}>
                      {rule.strategy === 2 ? (rule.threshold ?? 0) : `${((rule.threshold ?? 0) * 100).toFixed(0)}%`}
                    </Typography.Text>
                  </Descriptions.Item>
                  <Descriptions.Item label="最小请求数">{rule.minRequestAmount}</Descriptions.Item>
                  <Descriptions.Item label="统计窗口">{rule.statIntervalMs ?? 1000} ms</Descriptions.Item>
                  {rule.strategy === 0 && (
                    <Descriptions.Item label="最大允许RT">{rule.maxAllowedRtMs} ms</Descriptions.Item>
                  )}
                  <Descriptions.Item label="重试超时">{rule.retryTimeoutMs} ms</Descriptions.Item>
                  <Descriptions.Item label="探测数量">{(rule.probeNum ?? 0) === 0 ? '不限制' : rule.probeNum}</Descriptions.Item>
                </Descriptions>
              </Card>
            ))
          ) : (
            <Empty description="暂无熔断规则" image={Empty.PRESENTED_IMAGE_SIMPLE}>
              <Button type="dashed" icon={<PlusOutlined />} onClick={() => handleCreateCbRule()}>
                添加熔断规则
              </Button>
            </Empty>
          )
          )}
        </Spin>
      </Modal>

      {/* Flow Rule Edit Modal */}
      <Modal
        title={editingFlowRule ? '编辑流控规则' : '新增流控规则'}
        open={flowEditVisible}
        onOk={handleSaveFlowRule}
        onCancel={() => setFlowEditVisible(false)}
        destroyOnClose
        width={600}
      >
        <Form form={flowForm} layout="vertical" initialValues={{ metricType: 1, controlBehavior: 0, tokenCalculateStrategy: 0, warmUpColdFactor: 3, statIntervalInMs: 1000 }}>
          <div style={{ display: 'flex', alignItems: 'center', margin: '0 0 12px 0', fontSize: 14, fontWeight: 500, color: 'rgba(0,0,0,0.85)' }}>
            基础配置
            <div style={{ flex: 1, height: '1px', background: '#f0f0f0', marginLeft: 12 }} />
          </div>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="metricType" label="指标类型">
                <Select>
                  <Select.Option value={0}>并发数</Select.Option>
                  <Select.Option value={1}>QPS</Select.Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="threshold"
                label={<span>{flowMetricType === 1 ? 'QPS 阈值' : '并发数阈值'} <Tooltip title="QPS=每秒请求数，并发数=同时处理的线程数"><QuestionCircleOutlined style={{ color: '#999' }} /></Tooltip></span>}
                rules={[{ required: true, message: '请输入阈值' }]}
              >
                <InputNumber min={0} style={{ width: '100%' }} placeholder={flowMetricType === 1 ? '例如: 100' : '例如: 50'} />
              </Form.Item>
            </Col>
          </Row>

          <div style={{ display: 'flex', alignItems: 'center', margin: '8px 0 12px 0', fontSize: 14, fontWeight: 500, color: 'rgba(0,0,0,0.85)' }}>
            流量控制
            <div style={{ flex: 1, height: '1px', background: '#f0f0f0', marginLeft: 12 }} />
          </div>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="controlBehavior" label="控制行为">
                <Select>
                  <Select.Option value={0}>直接拒绝(Reject)</Select.Option>
                  <Select.Option value={1}>匀速排队(Throttling)</Select.Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="tokenCalculateStrategy" label="令牌计算策略">
                <Select>
                  <Select.Option value={0}>直接通过(Direct)</Select.Option>
                  <Select.Option value={1}>预热(WarmUp)</Select.Option>
                  <Select.Option value={2}>自适应(MemoryAdaptive)</Select.Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>
          {(flowControlBehavior === 1 || flowTokenCalculateStrategy === 1) && (
            <Row gutter={16}>
              {flowControlBehavior === 1 && (
                <Col span={12}>
                  <Form.Item
                    name="maxQueueingTimeMs"
                    label={<span>最大排队时间(ms) <Tooltip title="请求最大等待时间，超过则拒绝"><QuestionCircleOutlined style={{ color: '#999' }} /></Tooltip></span>}
                  >
                    <InputNumber min={0} style={{ width: '100%' }} placeholder="例如: 500" />
                  </Form.Item>
                </Col>
              )}
              {flowTokenCalculateStrategy === 1 && (
                <Col span={12}>
                  <Form.Item name="warmUpPeriodSec" label="预热时长(秒)">
                    <InputNumber min={0} style={{ width: '100%' }} placeholder="例如: 10" />
                  </Form.Item>
                </Col>
              )}
            </Row>
          )}
          {flowTokenCalculateStrategy === 1 && (
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item
                  name="warmUpColdFactor"
                  label={<span>冷启动因子 <Tooltip title="推荐 3-5，越大预热越平滑"><QuestionCircleOutlined style={{ color: '#999' }} /></Tooltip></span>}
                >
                  <InputNumber min={2} style={{ width: '100%' }} />
                </Form.Item>
              </Col>
            </Row>
          )}

          <div style={{ display: 'flex', alignItems: 'center', margin: '8px 0 12px 0', fontSize: 14, fontWeight: 500, color: 'rgba(0,0,0,0.85)' }}>
            高级设置
            <div style={{ flex: 1, height: '1px', background: '#f0f0f0', marginLeft: 12 }} />
          </div>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="statIntervalInMs"
                label={<span>统计窗口(ms) <Tooltip title="默认 1000ms，增大可平滑波动"><QuestionCircleOutlined style={{ color: '#999' }} /></Tooltip></span>}
              >
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="clusterMode" label="集群模式" valuePropName="checked">
                <Switch />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>

      {/* Circuit Breaker Rule Edit Modal */}
      <Modal
        title={editingCbRule ? '编辑熔断规则' : '新增熔断规则'}
        open={cbEditVisible}
        onOk={handleSaveCbRule}
        onCancel={() => setCbEditVisible(false)}
        destroyOnClose
        width={600}
      >
        <Form form={cbForm} layout="vertical" initialValues={{ strategy: 0 }}>
          <div style={{ display: 'flex', alignItems: 'center', margin: '0 0 12px 0', fontSize: 14, fontWeight: 500, color: 'rgba(0,0,0,0.85)' }}>
            基础配置
            <div style={{ flex: 1, height: '1px', background: '#f0f0f0', marginLeft: 12 }} />
          </div>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="strategy"
                label={<span>熔断策略 <Tooltip title="慢调用比例(按RT)、错误比例(按状态码)、错误计数(绝对数量)"><QuestionCircleOutlined style={{ color: 'rgba(0,0,0,.45)' }} /></Tooltip></span>}
              >
                <Select>
                  <Select.Option value={0}>慢调用比例</Select.Option>
                  <Select.Option value={1}>错误比例</Select.Option>
                  <Select.Option value={2}>错误计数</Select.Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="threshold"
                label={<span>阈值 <Tooltip title={cbStrategy === 0 ? '慢调用占比超过此值触发熔断 (0.0-1.0)' : cbStrategy === 1 ? '错误占比超过此值触发熔断 (0.0-1.0)' : '错误数超过此值触发熔断'}><QuestionCircleOutlined style={{ color: 'rgba(0,0,0,.45)' }} /></Tooltip></span>}
                rules={[{ required: true, message: '请输入阈值' }]}
              >
                {cbStrategy === 2 ? (
                  <InputNumber min={0} step={1} precision={0} placeholder="例如 10" style={{ width: '100%' }} />
                ) : (
                  <InputNumber min={0} max={1} step={0.1} precision={2} placeholder="例如 0.5" style={{ width: '100%' }} />
                )}
              </Form.Item>
            </Col>
          </Row>

          <div style={{ display: 'flex', alignItems: 'center', margin: '8px 0 12px 0', fontSize: 14, fontWeight: 500, color: 'rgba(0,0,0,0.85)' }}>
            熔断条件
            <div style={{ flex: 1, height: '1px', background: '#f0f0f0', marginLeft: 12 }} />
          </div>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="minRequestAmount"
                label={<span>最小请求数 <Tooltip title="至少多少请求才触发熔断判断，防止小流量误判"><QuestionCircleOutlined style={{ color: 'rgba(0,0,0,.45)' }} /></Tooltip></span>}
                rules={[{ required: true, message: '请输入' }]}
              >
                <InputNumber min={0} placeholder="例如 10" style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="statIntervalMs"
                label={<span>统计窗口(ms) <Tooltip title="统计指标的时间窗口，默认 1000ms"><QuestionCircleOutlined style={{ color: 'rgba(0,0,0,.45)' }} /></Tooltip></span>}
              >
                <InputNumber min={0} placeholder="默认 1000" style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>
          {cbStrategy === 0 && (
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item
                  name="maxAllowedRtMs"
                  label={<span>最大允许RT(ms) <Tooltip title="请求耗时超过此值即为慢请求"><QuestionCircleOutlined style={{ color: 'rgba(0,0,0,.45)' }} /></Tooltip></span>}
                  rules={[{ required: true, message: '请输入' }]}
                >
                  <InputNumber min={0} placeholder="例如 1000" style={{ width: '100%' }} />
                </Form.Item>
              </Col>
            </Row>
          )}

          <div style={{ display: 'flex', alignItems: 'center', margin: '8px 0 12px 0', fontSize: 14, fontWeight: 500, color: 'rgba(0,0,0,0.85)' }}>
            熔断恢复
            <div style={{ flex: 1, height: '1px', background: '#f0f0f0', marginLeft: 12 }} />
          </div>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="retryTimeoutMs"
                label={<span>重试超时(ms) <Tooltip title="熔断后等待多久进入半开状态"><QuestionCircleOutlined style={{ color: 'rgba(0,0,0,.45)' }} /></Tooltip></span>}
                rules={[{ required: true, message: '请输入' }]}
              >
                <InputNumber min={0} placeholder="例如 5000" style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="probeNum"
                label={<span>探测数量 <Tooltip title="半开状态下的探测请求数，全部成功则关闭熔断。0=不限制"><QuestionCircleOutlined style={{ color: 'rgba(0,0,0,.45)' }} /></Tooltip></span>}
              >
                <InputNumber min={0} placeholder="0=不限制" style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>

      {/* Publish Preview Modal */}
      <Modal
        title={`发布确认 - ${publishingResource?.name || ''}`}
        open={publishPreviewVisible}
        onOk={handleConfirmPublish}
        onCancel={() => setPublishPreviewVisible(false)}
        confirmLoading={publishPreviewLoading}
        destroyOnClose
        width={900}
        okText="确认发布"
        cancelText="取消"
      >
        <Spin spinning={publishRulesLoading}>
          <Row gutter={24}>
            <Col span={12}>
              <div style={{ background: '#f6ffed', padding: '8px 12px', borderRadius: 4, marginBottom: 12, fontWeight: 500 }}>
                <Badge status="processing" color="green" /> 待发布（当前配置）
              </div>
            </Col>
            <Col span={12}>
              <div style={{ background: '#f0f0f0', padding: '8px 12px', borderRadius: 4, marginBottom: 12, fontWeight: 500 }}>
                <Badge status="default" /> 已发布（运行中）
              </div>
            </Col>
          </Row>

          <h4 style={{ marginBottom: 8 }}>
            <ThunderboltOutlined style={{ color: '#1677ff', marginRight: 6 }} />
            流控规则
          </h4>
          <Row gutter={24} style={{ marginBottom: 16 }}>
            <Col span={12}>
              {publishFlowRules.length > 0 ? publishFlowRules.map((rule) => (
                <Card key={rule.id} size="small" style={{ marginBottom: 8 }}>
                  <Descriptions column={1} size="small" labelStyle={{ color: 'rgba(0,0,0,.45)', width: 80 }}>
                    <Descriptions.Item label="指标" contentStyle={isFieldChanged('flow', 'metricType') ? changedStyle : undefined}>{rule.metricType === 1 ? 'QPS' : '并发数'}</Descriptions.Item>
                    <Descriptions.Item label="阈值" contentStyle={isFieldChanged('flow', 'threshold') ? changedStyle : undefined}><Typography.Text strong style={{ color: '#1677ff' }}>{rule.threshold}</Typography.Text></Descriptions.Item>
                    <Descriptions.Item label="行为" contentStyle={isFieldChanged('flow', 'controlBehavior') ? changedStyle : undefined}>{rule.controlBehavior === 0 ? '直接拒绝' : '匀速排队'}</Descriptions.Item>
                    <Descriptions.Item label="策略" contentStyle={isFieldChanged('flow', 'tokenCalculateStrategy') ? changedStyle : undefined}>{rule.tokenCalculateStrategy === 0 ? 'Direct' : rule.tokenCalculateStrategy === 1 ? 'WarmUp' : 'MemoryAdaptive'}</Descriptions.Item>
                    <Descriptions.Item label="窗口" contentStyle={isFieldChanged('flow', 'statIntervalInMs') ? changedStyle : undefined}>{rule.statIntervalInMs ?? 1000} ms</Descriptions.Item>
                    {rule.controlBehavior === 1 && (
                      <Descriptions.Item label="排队时间" contentStyle={isFieldChanged('flow', 'maxQueueingTimeMs') ? changedStyle : undefined}>{rule.maxQueueingTimeMs} ms</Descriptions.Item>
                    )}
                    {(rule.tokenCalculateStrategy ?? 0) === 1 && (
                      <Descriptions.Item label="预热时长" contentStyle={isFieldChanged('flow', 'warmUpPeriodSec') ? changedStyle : undefined}>{rule.warmUpPeriodSec} s</Descriptions.Item>
                    )}
                    {(rule.tokenCalculateStrategy ?? 0) === 1 && (
                      <Descriptions.Item label="冷启动因子" contentStyle={isFieldChanged('flow', 'warmUpColdFactor') ? changedStyle : undefined}>{rule.warmUpColdFactor ?? 3}</Descriptions.Item>
                    )}
                  </Descriptions>
                </Card>
              )) : <Typography.Text type="secondary">无规则</Typography.Text>}
            </Col>
            <Col span={12}>
              {publishedFlowRules.length > 0 ? publishedFlowRules.map((rule, idx) => (
                <Card key={idx} size="small" style={{ marginBottom: 8, background: '#fafafa' }}>
                  <Descriptions column={1} size="small" labelStyle={{ color: 'rgba(0,0,0,.45)', width: 80 }}>
                    <Descriptions.Item label="指标">{rule.metricType === 1 ? 'QPS' : '并发数'}</Descriptions.Item>
                    <Descriptions.Item label="阈值"><Typography.Text style={{ color: 'rgba(0,0,0,.45)' }}>{rule.threshold}</Typography.Text></Descriptions.Item>
                    <Descriptions.Item label="行为">{rule.controlBehavior === 0 ? '直接拒绝' : '匀速排队'}</Descriptions.Item>
                    <Descriptions.Item label="策略">{rule.tokenCalculateStrategy === 0 ? 'Direct' : rule.tokenCalculateStrategy === 1 ? 'WarmUp' : 'MemoryAdaptive'}</Descriptions.Item>
                    <Descriptions.Item label="窗口">{rule.statIntervalInMs ?? 1000} ms</Descriptions.Item>
                  </Descriptions>
                </Card>
              )) : <Typography.Text type="secondary">无已发布版本</Typography.Text>}
            </Col>
          </Row>

          <h4 style={{ marginBottom: 8 }}>
            <SafetyOutlined style={{ color: '#fa8c16', marginRight: 6 }} />
            熔断规则
          </h4>
          <Row gutter={24}>
            <Col span={12}>
              {publishCbRules.length > 0 ? publishCbRules.map((rule) => (
                <Card key={rule.id} size="small" style={{ marginBottom: 8 }}>
                  <Descriptions column={1} size="small" labelStyle={{ color: 'rgba(0,0,0,.45)', width: 80 }}>
                    <Descriptions.Item label="策略" contentStyle={isFieldChanged('cb', 'strategy') ? changedStyle : undefined}>{renderCbStrategy(rule.strategy)}</Descriptions.Item>
                    <Descriptions.Item label="阈值" contentStyle={isFieldChanged('cb', 'threshold') ? changedStyle : undefined}><Typography.Text strong style={{ color: '#fa8c16' }}>{rule.strategy === 2 ? (rule.threshold ?? 0) : `${((rule.threshold ?? 0) * 100).toFixed(0)}%`}</Typography.Text></Descriptions.Item>
                    <Descriptions.Item label="最小请求" contentStyle={isFieldChanged('cb', 'minRequestAmount') ? changedStyle : undefined}>{rule.minRequestAmount}</Descriptions.Item>
                    <Descriptions.Item label="重试超时" contentStyle={isFieldChanged('cb', 'retryTimeoutMs') ? changedStyle : undefined}>{rule.retryTimeoutMs} ms</Descriptions.Item>
                    <Descriptions.Item label="窗口" contentStyle={isFieldChanged('cb', 'statIntervalMs') ? changedStyle : undefined}>{rule.statIntervalMs ?? 1000} ms</Descriptions.Item>
                    {rule.strategy === 0 && (
                      <Descriptions.Item label="最大RT" contentStyle={isFieldChanged('cb', 'maxAllowedRtMs') ? changedStyle : undefined}>{rule.maxAllowedRtMs} ms</Descriptions.Item>
                    )}
                    <Descriptions.Item label="探测数量" contentStyle={isFieldChanged('cb', 'probeNum') ? changedStyle : undefined}>{(rule.probeNum ?? 0) === 0 ? '不限制' : rule.probeNum}</Descriptions.Item>
                  </Descriptions>
                </Card>
              )) : <Typography.Text type="secondary">无规则</Typography.Text>}
            </Col>
            <Col span={12}>
              {publishedCbRules.length > 0 ? publishedCbRules.map((rule, idx) => (
                <Card key={idx} size="small" style={{ marginBottom: 8, background: '#fafafa' }}>
                  <Descriptions column={1} size="small" labelStyle={{ color: 'rgba(0,0,0,.45)', width: 80 }}>
                    <Descriptions.Item label="策略">{renderCbStrategy(rule.strategy)}</Descriptions.Item>
                    <Descriptions.Item label="阈值"><Typography.Text style={{ color: 'rgba(0,0,0,.45)' }}>{rule.strategy === 2 ? (rule.threshold ?? 0) : `${((rule.threshold ?? 0) * 100).toFixed(0)}%`}</Typography.Text></Descriptions.Item>
                    <Descriptions.Item label="最小请求">{rule.minRequestAmount}</Descriptions.Item>
                    <Descriptions.Item label="重试超时">{rule.retryTimeoutMs} ms</Descriptions.Item>
                    <Descriptions.Item label="窗口">{rule.statIntervalMs ?? 1000} ms</Descriptions.Item>
                  </Descriptions>
                </Card>
              )) : <Typography.Text type="secondary">无已发布版本</Typography.Text>}
            </Col>
          </Row>
          {diffData && diffData.change_count > 0 && (
            <div style={{ marginTop: 16, padding: '12px 16px', background: '#f6ffed', border: '1px solid #b7eb8f', borderRadius: 6 }}>
              <strong>变更摘要: {diffData.change_count} 项变更</strong>
              <ul style={{ margin: '8px 0 0', paddingLeft: 20 }}>
                {diffData.flow_diffs?.filter((d: any) => d.changed).map((d: any, i: number) => (
                  <li key={`flow-${i}`}>⚡ 流控 - {d.label}: {String(d.old_value ?? '无')} → {String(d.new_value)}</li>
                ))}
                {diffData.cb_diffs?.filter((d: any) => d.changed).map((d: any, i: number) => (
                  <li key={`cb-${i}`}>🔧 熔断 - {d.label}: {String(d.old_value ?? '无')} → {String(d.new_value)}</li>
                ))}
              </ul>
            </div>
          )}
          {diffData && diffData.change_count === 0 && (
            <div style={{ marginTop: 16, padding: '12px 16px', background: '#e6f7ff', border: '1px solid #91d5ff', borderRadius: 6 }}>
              <strong>当前配置与已发布版本一致，无需发布</strong>
            </div>
          )}
        </Spin>
      </Modal>

      {/* Versions Modal */}
      <Modal
        title={`版本历史 - ${versionsResource?.name || ''}`}
        open={versionsVisible}
        onCancel={() => setVersionsVisible(false)}
        footer={null}
        width={700}
      >
        <Table<VersionRecord>
          size="small"
          rowKey="id"
          dataSource={versions}
          pagination={{ pageSize: 10, showSizeChanger: true, showTotal: (total) => `共 ${total} 条` }}
          locale={{ emptyText: '暂无版本记录' }}
          columns={[
            { title: '版本', dataIndex: 'version_number', key: 'ver', render: (v: number) => <Tag color="blue">v{v}</Tag> },
            { title: '描述', dataIndex: 'description', key: 'desc' },
            { title: '规则数', dataIndex: 'rule_count', key: 'count', width: 80 },
            { title: '发布时间', dataIndex: 'created_at', key: 'time', render: (v: string) => v ? new Date(v).toLocaleString('zh-CN') : '-' },
            {
              title: '操作', key: 'action', width: 180,
              render: (_: unknown, record: VersionRecord) => (
                <Space>
                  <Button size="small" onClick={() => handleViewVersion(record.id)}>查看</Button>
                  <Popconfirm
                    title="确认回滚"
                    description={`确定要回滚到 v${record.version_number} 吗？`}
                    onConfirm={() => handleRollback(record.id, record.version_number)}
                    okText="确定"
                    cancelText="取消"
                  >
                    <Button size="small" danger icon={<RollbackOutlined />}>回滚</Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </Modal>

      {/* Version Detail Modal */}
      <Modal
        title={selectedVersion ? `v${selectedVersion.version_number} - ${selectedVersion.description}` : '版本详情'}
        open={versionDetailVisible}
        onCancel={() => setVersionDetailVisible(false)}
        footer={null}
        width={720}
      >
        {versionSnapshot && (
          <>
            <h4>流控规则</h4>
            <Table
              size="small"
              dataSource={versionSnapshot.flow_rules || []}
              pagination={false}
              columns={[
                { title: '规则ID', dataIndex: 'rule_id' },
                { title: '资源', dataIndex: 'resource' },
                { title: '阈值', dataIndex: 'threshold', width: 80 },
              ]}
            />
            <h4 style={{ marginTop: 16 }}>熔断规则</h4>
            <Table
              size="small"
              dataSource={versionSnapshot.circuit_breaker_rules || []}
              pagination={false}
              columns={[
                { title: '规则ID', dataIndex: 'rule_id' },
                { title: '资源', dataIndex: 'resource' },
                { title: '策略', dataIndex: 'strategy', width: 80 },
                { title: '阈值', dataIndex: 'threshold', width: 80 },
              ]}
            />
          </>
        )}
      </Modal>
    </div>
  );
};

export default Resources;
