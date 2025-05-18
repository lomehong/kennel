import React, { useEffect, useState } from 'react';
import { Table, Card, Button, Switch, Typography, Divider, Space, Tag, message } from 'antd';
import { Link } from 'react-router-dom';
import {
  ReloadOutlined,
  SettingOutlined,
  InfoCircleOutlined
} from '@ant-design/icons';
import { pluginsApi } from '../services/api';

const { Title } = Typography;

interface Plugin {
  id: string;
  name: string;
  version: string;
  description: string;
  enabled: boolean;
  status: string;
}

const PluginManager: React.FC = () => {
  const [plugins, setPlugins] = useState<Plugin[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const fetchPlugins = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await pluginsApi.getPlugins();
      // 检查响应数据结构
      if (response.data && Array.isArray(response.data.plugins)) {
        setPlugins(response.data.plugins);
        console.log('插件列表:', response.data.plugins);
      } else if (response.data && Array.isArray(response.data)) {
        // 兼容直接返回数组的情况
        setPlugins(response.data);
        console.log('插件列表:', response.data);
      } else {
        const errorMsg = '获取插件列表失败: 返回数据结构不正确';
        console.error(errorMsg, response.data);
        message.error(errorMsg);
        setError(errorMsg);
      }
    } catch (error) {
      const errorMsg = '获取插件列表失败: ' + (error instanceof Error ? error.message : String(error));
      console.error('获取插件列表失败:', error);
      message.error(errorMsg);
      setError(errorMsg);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchPlugins();
  }, []);

  const handleTogglePlugin = async (id: string, enabled: boolean) => {
    try {
      await pluginsApi.updatePluginStatus(id, enabled);
      message.success(`插件${enabled ? '启用' : '禁用'}成功`);

      // 更新本地状态
      setPlugins(plugins.map(plugin =>
        plugin.id === id ? { ...plugin, enabled } : plugin
      ));
    } catch (error) {
      console.error('更新插件状态失败:', error);
      message.error('更新插件状态失败');
    }
  };

  const getStatusTag = (status: string) => {
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

  const columns = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string, record: Plugin) => (
        <Link to={`/plugins/${record.id}`}>{text}</Link>
      ),
    },
    {
      title: '版本',
      dataIndex: 'version',
      key: 'version',
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => getStatusTag(status),
    },
    {
      title: '启用',
      dataIndex: 'enabled',
      key: 'enabled',
      render: (enabled: boolean, record: Plugin) => (
        <Switch
          checked={enabled}
          onChange={(checked) => handleTogglePlugin(record.id, checked)}
        />
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: Plugin) => (
        <Space size="middle">
          <Link to={`/plugins/${record.id}`}>
            <Button type="text" icon={<InfoCircleOutlined />}>
              详情
            </Button>
          </Link>
          <Link to={`/plugins/${record.id}`} state={{ activeTab: '2' }}>
            <Button type="text" icon={<SettingOutlined />}>
              配置
            </Button>
          </Link>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={2}>插件管理</Title>
        <Button
          type="primary"
          icon={<ReloadOutlined />}
          onClick={fetchPlugins}
          loading={loading}
        >
          刷新
        </Button>
      </div>
      <Divider />

      <Card>
        {error ? (
          <div style={{ textAlign: 'center', padding: '20px' }}>
            <InfoCircleOutlined style={{ fontSize: '32px', color: '#ff4d4f', marginBottom: '16px' }} />
            <p style={{ color: '#ff4d4f', fontSize: '16px' }}>{error}</p>
            <Button type="primary" onClick={fetchPlugins} style={{ marginTop: '16px' }}>
              重试
            </Button>
          </div>
        ) : (
          <Table
            columns={columns}
            dataSource={plugins}
            rowKey="id"
            loading={loading}
            pagination={false}
            locale={{ emptyText: '暂无插件数据' }}
          />
        )}
      </Card>
    </div>
  );
};

export default PluginManager;
