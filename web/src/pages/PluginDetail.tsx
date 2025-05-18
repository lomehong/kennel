import React, { useEffect, useState } from 'react';
import { useParams, useLocation, useNavigate } from 'react-router-dom';
import {
  Card,
  Tabs,
  Descriptions,
  Button,
  Typography,
  Divider,
  Tag,
  Form,
  Input,
  Switch,
  InputNumber,
  Select,
  Table,
  message,
  Space
} from 'antd';
import {
  ArrowLeftOutlined,
  SaveOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined
} from '@ant-design/icons';
import { pluginsApi } from '../services/api';

const { Title } = Typography;
const { TabPane } = Tabs;
const { Option } = Select;

interface Plugin {
  id: string;
  name: string;
  version: string;
  description: string;
  enabled: boolean;
  status: string;
  actions?: string[];
  supported_actions?: string[];
  config?: any;
}

interface LogEntry {
  timestamp: string;
  level: string;
  message: string;
  source?: string;
  data?: any;
  [key: string]: any; // 允许任意其他字段
}

const PluginDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const location = useLocation();
  const navigate = useNavigate();
  const [plugin, setPlugin] = useState<Plugin | null>(null);
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [logsLoading, setLogsLoading] = useState<boolean>(true);
  const [form] = Form.useForm();

  // 获取初始激活的标签页
  const initialState = location.state as { activeTab?: string } | null;
  const [activeTab, setActiveTab] = useState<string>(initialState?.activeTab || '1');

  const fetchPlugin = async () => {
    if (!id) return;

    setLoading(true);
    try {
      const response = await pluginsApi.getPlugin(id);
      // 确保响应数据有效
      if (response.data) {
        // 确保必要字段存在
        const pluginData = {
          ...response.data,
          config: response.data.config || {},
          actions: response.data.actions || [],
          supported_actions: response.data.supported_actions || []
        };

        // 兼容不同的API响应格式
        if (!pluginData.actions || pluginData.actions.length === 0) {
          if (pluginData.supported_actions && pluginData.supported_actions.length > 0) {
            pluginData.actions = pluginData.supported_actions;
          }
        }

        // 确保actions字段存在且有值
        if (!pluginData.actions || pluginData.actions.length === 0) {
          if (pluginData.supported_actions && pluginData.supported_actions.length > 0) {
            pluginData.actions = pluginData.supported_actions;
          } else {
            pluginData.actions = [];
          }
        }

        console.log('处理后的插件数据:', pluginData);
        setPlugin(pluginData);
        form.setFieldsValue({ config: pluginData.config });
      } else {
        throw new Error('返回数据无效');
      }
    } catch (error) {
      console.error('获取插件详情失败:', error);
      message.error('获取插件详情失败');
    } finally {
      setLoading(false);
    }
  };

  const fetchLogs = async () => {
    if (!id) return;

    setLogsLoading(true);
    try {
      const response = await pluginsApi.getPluginLogs(id);
      // 确保响应数据有效
      if (response.data) {
        // 检查数据结构
        if (Array.isArray(response.data)) {
          setLogs(response.data);
        } else if (response.data.logs && Array.isArray(response.data.logs)) {
          setLogs(response.data.logs);
        } else {
          // 如果没有有效的日志数据，设置为空数组
          setLogs([]);
          console.warn('未找到有效的日志数据', response.data);
        }
      } else {
        setLogs([]);
      }
    } catch (error) {
      console.error('获取插件日志失败:', error);
      message.error('获取插件日志失败');
      setLogs([]);
    } finally {
      setLogsLoading(false);
    }
  };

  useEffect(() => {
    fetchPlugin();
  }, [id]);

  useEffect(() => {
    if (activeTab === '3') {
      fetchLogs();
    }
  }, [activeTab, id]);

  const handleSaveConfig = async (values: any) => {
    if (!id) return;

    try {
      console.log('保存配置:', values.config);
      await pluginsApi.updatePluginConfig(id, values.config);
      message.success('配置保存成功');
      fetchPlugin();
    } catch (error) {
      console.error('保存配置失败:', error);
      message.error('保存配置失败');
    }
  };

  const getStatusTag = (status?: string) => {
    if (!status) return null;

    switch (status) {
      case 'running':
        return <Tag color="green">运行中</Tag>;
      case 'stopped':
        return <Tag color="red">已停止</Tag>;
      case 'error':
        return <Tag color="red">错误</Tag>;
      case 'initializing':
        return <Tag color="blue">初始化中</Tag>;
      default:
        return <Tag color="default">{status}</Tag>;
    }
  };

  const getLevelTag = (level: string) => {
    switch (level.toLowerCase()) {
      case 'error':
        return <Tag color="red">错误</Tag>;
      case 'warn':
      case 'warning':
        return <Tag color="orange">警告</Tag>;
      case 'info':
        return <Tag color="blue">信息</Tag>;
      case 'debug':
        return <Tag color="green">调试</Tag>;
      case 'trace':
        return <Tag color="purple">跟踪</Tag>;
      default:
        return <Tag>{level}</Tag>;
    }
  };

  const renderConfigForm = () => {
    if (!plugin) return null;

    // 确保config字段存在
    const config = plugin.config || {};

    // 将配置对象转换为JSON字符串，便于编辑
    const configStr = JSON.stringify(config, null, 2);

    return (
      <Form
        form={form}
        layout="vertical"
        onFinish={(values) => {
          try {
            // 尝试将文本框中的内容解析为JSON对象
            const configObj = JSON.parse(values.config);
            handleSaveConfig({ config: configObj });
          } catch (error) {
            message.error('配置格式无效，请检查JSON格式');
            console.error('解析配置失败:', error);
          }
        }}
        initialValues={{ config: configStr }}
      >
        <Form.Item
          name="config"
          rules={[
            {
              validator: (_, value) => {
                try {
                  JSON.parse(value);
                  return Promise.resolve();
                } catch (error) {
                  return Promise.reject('配置必须是有效的JSON格式');
                }
              }
            }
          ]}
        >
          <Input.TextArea
            rows={20}
            style={{ fontFamily: 'monospace' }}
            placeholder="请输入JSON格式的配置"
          />
        </Form.Item>

        <Divider />

        <Form.Item>
          <Button type="primary" htmlType="submit" icon={<SaveOutlined />}>
            保存配置
          </Button>
        </Form.Item>
      </Form>
    );
  };

  const logColumns = [
    {
      title: '时间',
      dataIndex: 'timestamp',
      key: 'timestamp',
      width: 220,
      render: (timestamp: string) => {
        if (!timestamp) return '-';
        // 尝试格式化时间戳，如果失败则直接返回原始值
        try {
          const date = new Date(timestamp);
          if (isNaN(date.getTime())) return timestamp;
          return date.toLocaleString('zh-CN', {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
          });
        } catch (e) {
          return timestamp;
        }
      },
      ellipsis: false,
    },
    {
      title: '级别',
      dataIndex: 'level',
      key: 'level',
      width: 100,
      render: (level: string) => level ? getLevelTag(level) : '-',
    },
    {
      title: '消息',
      dataIndex: 'message',
      key: 'message',
      render: (message: string) => message || '-',
    },
    {
      title: '来源',
      dataIndex: 'source',
      key: 'source',
      width: 150,
      render: (source: string) => source || '-',
    },
  ];

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Button
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate('/plugins')}
        >
          返回
        </Button>
        <Title level={2} style={{ margin: 0 }}>
          {plugin?.name || '插件详情'}
        </Title>
        {plugin?.enabled ? (
          <Tag icon={<CheckCircleOutlined />} color="success">已启用</Tag>
        ) : (
          <Tag icon={<CloseCircleOutlined />} color="error">已禁用</Tag>
        )}
        {getStatusTag(plugin?.status)}
      </Space>

      <Divider />

      <Card loading={loading}>
        <Tabs activeKey={activeTab} onChange={setActiveTab}>
          <TabPane tab="基本信息" key="1">
            {plugin && (
              <Descriptions bordered column={1}>
                <Descriptions.Item label="ID">{plugin.id}</Descriptions.Item>
                <Descriptions.Item label="名称">{plugin.name}</Descriptions.Item>
                <Descriptions.Item label="版本">{plugin.version}</Descriptions.Item>
                <Descriptions.Item label="描述">{plugin.description}</Descriptions.Item>
                <Descriptions.Item label="状态">
                  {getStatusTag(plugin.status)}
                </Descriptions.Item>
                <Descriptions.Item label="支持的操作">
                  {(() => {
                    // 优先使用 supported_actions，如果不存在则使用 actions
                    const actions = plugin.supported_actions && plugin.supported_actions.length > 0
                      ? plugin.supported_actions
                      : (plugin.actions || []);

                    console.log('显示的操作:', actions);

                    return Array.isArray(actions) && actions.length > 0
                      ? actions.map(action => (
                          <Tag key={action}>{action}</Tag>
                        ))
                      : <Tag>无</Tag>;
                  })()}
                </Descriptions.Item>
              </Descriptions>
            )}
          </TabPane>

          <TabPane tab="配置" key="2">
            {renderConfigForm()}
          </TabPane>

          <TabPane tab="日志" key="3">
            <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'flex-end' }}>
              <Button
                icon={<ReloadOutlined />}
                onClick={fetchLogs}
                loading={logsLoading}
              >
                刷新
              </Button>
            </div>

            <Table
              columns={logColumns}
              dataSource={logs}
              rowKey={(record, index) => `${record.timestamp || ''}-${index}`}
              loading={logsLoading}
              pagination={{ pageSize: 10 }}
              locale={{ emptyText: '暂无日志数据' }}
              scroll={{ x: 'max-content' }}
              style={{ whiteSpace: 'nowrap' }}
            />
          </TabPane>
        </Tabs>
      </Card>
    </div>
  );
};

export default PluginDetail;
