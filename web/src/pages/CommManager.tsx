import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Card,
  Button,
  Table,
  Tag,
  Space,
  Descriptions,
  Statistic,
  Row,
  Col,
  message,
  Typography,
  Divider,
  Alert,
} from 'antd';
import {
  ApiOutlined,
  DisconnectOutlined,
  ReloadOutlined,
  SettingOutlined,
  BugOutlined,
} from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-layout';
import { getCommStatus, getCommConfig, getCommStats, getCommLogs, connectComm, disconnectComm } from '@/services/api';

const { Title, Paragraph } = Typography;

const CommManager: React.FC = () => {
  const navigate = useNavigate();
  const [status, setStatus] = useState<any>(null);
  const [config, setConfig] = useState<any>(null);
  const [stats, setStats] = useState<any>(null);
  const [logs, setLogs] = useState<any[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [logsLoading, setLogsLoading] = useState<boolean>(false);
  const [connectLoading, setConnectLoading] = useState<boolean>(false);
  const [disconnectLoading, setDisconnectLoading] = useState<boolean>(false);

  // 加载通讯状态
  const loadStatus = async () => {
    setLoading(true);
    try {
      const response = await getCommStatus();
      setStatus(response);
    } catch (error) {
      message.error('加载通讯状态失败');
      console.error('加载通讯状态失败:', error);
    } finally {
      setLoading(false);
    }
  };

  // 加载通讯配置
  const loadConfig = async () => {
    setLoading(true);
    try {
      const response = await getCommConfig();
      setConfig(response);
    } catch (error) {
      message.error('加载通讯配置失败');
      console.error('加载通讯配置失败:', error);
    } finally {
      setLoading(false);
    }
  };

  // 加载通讯统计信息
  const loadStats = async () => {
    setLoading(true);
    try {
      const response = await getCommStats();
      setStats(response);
    } catch (error) {
      message.error('加载通讯统计信息失败');
      console.error('加载通讯统计信息失败:', error);
    } finally {
      setLoading(false);
    }
  };

  // 加载通讯日志
  const loadLogs = async () => {
    setLogsLoading(true);
    try {
      const response = await getCommLogs();
      if (response && response.logs && Array.isArray(response.logs)) {
        setLogs(response.logs);
      } else {
        setLogs([]);
      }
    } catch (error) {
      message.error('加载通讯日志失败');
      console.error('加载通讯日志失败:', error);
      setLogs([]);
    } finally {
      setLogsLoading(false);
    }
  };

  // 连接到服务器
  const handleConnect = async () => {
    setConnectLoading(true);
    try {
      await connectComm({});
      message.success('连接成功');
      loadStatus();
      loadStats();
    } catch (error) {
      message.error('连接失败');
      console.error('连接失败:', error);
    } finally {
      setConnectLoading(false);
    }
  };

  // 断开连接
  const handleDisconnect = async () => {
    setDisconnectLoading(true);
    try {
      await disconnectComm();
      message.success('断开连接成功');
      loadStatus();
      loadStats();
    } catch (error) {
      message.error('断开连接失败');
      console.error('断开连接失败:', error);
    } finally {
      setDisconnectLoading(false);
    }
  };

  // 刷新所有数据
  const refreshAll = () => {
    loadStatus();
    loadConfig();
    loadStats();
    loadLogs();
  };

  // 初始加载
  useEffect(() => {
    refreshAll();
    // 定时刷新状态和统计信息
    const timer = setInterval(() => {
      loadStatus();
      loadStats();
    }, 5000);
    return () => clearInterval(timer);
  }, []);

  // 日志列
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
      render: (level: string) => {
        let color = 'blue';
        if (level === 'error') {
          color = 'red';
        } else if (level === 'warn') {
          color = 'orange';
        } else if (level === 'info') {
          color = 'green';
        } else if (level === 'debug') {
          color = 'purple';
        }
        return <Tag color={color}>{level}</Tag>;
      },
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

  // 获取连接状态标签
  const getStatusTag = () => {
    if (!status) return <Tag color="default">未知</Tag>;

    switch (status.state) {
      case 'connected':
        return <Tag color="success">已连接</Tag>;
      case 'connecting':
        return <Tag color="processing">连接中</Tag>;
      case 'disconnected':
        return <Tag color="default">未连接</Tag>;
      case 'error':
        return <Tag color="error">错误</Tag>;
      default:
        return <Tag color="default">{status.state}</Tag>;
    }
  };

  return (
    <PageContainer
      title="通讯管理"
      extra={[
        <Button key="refresh" icon={<ReloadOutlined />} onClick={refreshAll}>
          刷新
        </Button>,
        <Button
          key="test"
          type="primary"
          icon={<BugOutlined />}
          onClick={() => navigate('/comm/test')}
        >
          通讯测试
        </Button>,
      ]}
    >
      <Card>
        <Row gutter={16}>
          <Col span={12}>
            <Title level={4}>通讯状态</Title>
            <Descriptions bordered>
              <Descriptions.Item label="连接状态" span={3}>
                {getStatusTag()}
              </Descriptions.Item>
              {status && (
                <>
                  <Descriptions.Item label="服务器地址" span={3}>
                    {status.server_url || '-'}
                  </Descriptions.Item>
                  <Descriptions.Item label="连接时间" span={3}>
                    {status.connected_at || '-'}
                  </Descriptions.Item>
                </>
              )}
            </Descriptions>
            <div style={{ marginTop: 16 }}>
              <Space>
                <Button
                  type="primary"
                  icon={<ApiOutlined />}
                  loading={connectLoading}
                  onClick={handleConnect}
                  disabled={status?.state === 'connected'}
                >
                  连接
                </Button>
                <Button
                  danger
                  icon={<DisconnectOutlined />}
                  loading={disconnectLoading}
                  onClick={handleDisconnect}
                  disabled={status?.state !== 'connected'}
                >
                  断开连接
                </Button>
              </Space>
            </div>
          </Col>
          <Col span={12}>
            <Title level={4}>通讯统计</Title>
            {stats ? (
              <Row gutter={16}>
                <Col span={12}>
                  <Statistic title="发送消息数" value={stats.messages_sent || 0} />
                </Col>
                <Col span={12}>
                  <Statistic title="接收消息数" value={stats.messages_received || 0} />
                </Col>
                <Col span={12}>
                  <Statistic title="发送字节数" value={stats.bytes_sent || 0} suffix="字节" />
                </Col>
                <Col span={12}>
                  <Statistic title="接收字节数" value={stats.bytes_received || 0} suffix="字节" />
                </Col>
                <Col span={12}>
                  <Statistic title="错误数" value={stats.errors || 0} />
                </Col>
                <Col span={12}>
                  <Statistic title="重连次数" value={stats.reconnects || 0} />
                </Col>
              </Row>
            ) : (
              <Alert message="暂无统计数据" type="info" />
            )}
          </Col>
        </Row>

        <Divider />

        <Title level={4}>通讯配置</Title>
        {config ? (
          <Descriptions bordered>
            <Descriptions.Item label="服务器地址" span={3}>
              {config.server_url || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="自动重连" span={1}>
              {config.auto_reconnect ? '启用' : '禁用'}
            </Descriptions.Item>
            <Descriptions.Item label="重连间隔" span={1}>
              {config.reconnect_interval || 0} 秒
            </Descriptions.Item>
            <Descriptions.Item label="最大重连次数" span={1}>
              {config.max_reconnect_attempts || 0}
            </Descriptions.Item>
            <Descriptions.Item label="心跳间隔" span={1}>
              {config.heartbeat_interval || 0} 秒
            </Descriptions.Item>
            <Descriptions.Item label="握手超时" span={1}>
              {config.handshake_timeout || 0} 秒
            </Descriptions.Item>
            <Descriptions.Item label="请求超时" span={1}>
              {config.request_timeout || 0} 秒
            </Descriptions.Item>
          </Descriptions>
        ) : (
          <Alert message="暂无配置数据" type="info" />
        )}

        <div style={{ marginTop: 16, textAlign: 'right' }}>
          <Button icon={<SettingOutlined />} onClick={() => message.info('配置编辑功能待实现')}>
            编辑配置
          </Button>
        </div>

        <Divider />

        <Title level={4}>通讯日志</Title>
        <Table
          columns={logColumns}
          dataSource={Array.isArray(logs) ? logs : []}
          rowKey={(record, index) => `${record?.timestamp || ''}-${index}`}
          loading={logsLoading}
          pagination={{ pageSize: 10 }}
          locale={{ emptyText: '暂无日志数据' }}
          scroll={{ x: 'max-content' }}
          style={{ whiteSpace: 'nowrap' }}
        />
      </Card>
    </PageContainer>
  );
};

export default CommManager;
