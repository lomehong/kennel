import React, { useEffect, useState } from 'react';
import { Card, Row, Col, Typography, Divider, Statistic, Button, Tabs, Table, Tag, message } from 'antd';
import { ReloadOutlined, WarningOutlined } from '@ant-design/icons';
import { systemApi } from '../services/api';

const { Title } = Typography;
const { TabPane } = Tabs;

interface SystemStatus {
  host: any;
  framework: any;
  plugins: any;
  comm: any;
  runtime: any;
}

interface SystemResources {
  cpu: any;
  memory: any;
  disk: any;
  process: any;
  runtime: any;
}

interface LogEntry {
  timestamp: string;
  level: string;
  message: string;
  source: string;
  data?: any;
}

interface SystemEvent {
  timestamp: string;
  type: string;
  message: string;
  source: string;
  data?: any;
}

const SystemMonitor: React.FC = () => {
  const [status, setStatus] = useState<SystemStatus | null>(null);
  const [resources, setResources] = useState<SystemResources | null>(null);
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [events, setEvents] = useState<SystemEvent[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [logsLoading, setLogsLoading] = useState<boolean>(true);
  const [eventsLoading, setEventsLoading] = useState<boolean>(true);
  const [activeTab, setActiveTab] = useState<string>('1');
  const [statusError, setStatusError] = useState<string | null>(null);
  const [resourcesError, setResourcesError] = useState<string | null>(null);
  const [logsError, setLogsError] = useState<string | null>(null);
  const [eventsError, setEventsError] = useState<string | null>(null);

  const fetchStatus = async () => {
    setLoading(true);
    setStatusError(null);
    message.loading({ content: '正在加载系统状态...', key: 'statusLoading', duration: 0 });
    try {
      const response = await systemApi.getSystemStatus();
      if (response && response.data) {
        setStatus(response.data);
        console.log('系统状态数据:', response.data);
        message.success({ content: '系统状态加载成功', key: 'statusLoading', duration: 2 });
      } else {
        const errorMsg = '获取系统状态失败: 返回数据为空';
        setStatusError(errorMsg);
        console.error(errorMsg);
        message.error({ content: errorMsg, key: 'statusLoading', duration: 3 });
      }
    } catch (error) {
      const errorMsg = '获取系统状态失败，请检查网络连接或服务器状态';
      setStatusError(errorMsg);
      console.error('获取系统状态失败:', error);
      message.error({ content: errorMsg, key: 'statusLoading', duration: 3 });
    } finally {
      setLoading(false);
    }
  };

  const fetchResources = async () => {
    setLoading(true);
    setResourcesError(null);
    message.loading({ content: '正在加载系统资源...', key: 'resourcesLoading', duration: 0 });
    try {
      const response = await systemApi.getSystemResources();
      if (response && response.data) {
        setResources(response.data);
        console.log('系统资源数据:', response.data);
        message.success({ content: '系统资源加载成功', key: 'resourcesLoading', duration: 2 });
      } else {
        const errorMsg = '获取系统资源失败: 返回数据为空';
        setResourcesError(errorMsg);
        console.error(errorMsg);
        message.error({ content: errorMsg, key: 'resourcesLoading', duration: 3 });
      }
    } catch (error) {
      const errorMsg = '获取系统资源失败，请检查网络连接或服务器状态';
      setResourcesError(errorMsg);
      console.error('获取系统资源失败:', error);
      message.error({ content: errorMsg, key: 'resourcesLoading', duration: 3 });
    } finally {
      setLoading(false);
    }
  };

  const fetchLogs = async () => {
    setLogsLoading(true);
    setLogsError(null);
    message.loading({ content: '正在加载系统日志...', key: 'logsLoading', duration: 0 });
    try {
      const response = await systemApi.getSystemLogs();
      if (response && response.data) {
        setLogs(response.data);
        console.log('系统日志数据:', response.data);
        message.success({ content: '系统日志加载成功', key: 'logsLoading', duration: 2 });
      } else {
        const errorMsg = '获取系统日志失败: 返回数据为空';
        setLogsError(errorMsg);
        console.error(errorMsg);
        message.error({ content: errorMsg, key: 'logsLoading', duration: 3 });
      }
    } catch (error) {
      const errorMsg = '获取系统日志失败，请检查网络连接或服务器状态';
      setLogsError(errorMsg);
      console.error('获取系统日志失败:', error);
      message.error({ content: errorMsg, key: 'logsLoading', duration: 3 });
    } finally {
      setLogsLoading(false);
    }
  };

  const fetchEvents = async () => {
    setEventsLoading(true);
    setEventsError(null);
    message.loading({ content: '正在加载系统事件...', key: 'eventsLoading', duration: 0 });
    try {
      const response = await systemApi.getSystemEvents();
      if (response && response.data) {
        setEvents(response.data);
        console.log('系统事件数据:', response.data);
        message.success({ content: '系统事件加载成功', key: 'eventsLoading', duration: 2 });
      } else {
        const errorMsg = '获取系统事件失败: 返回数据为空';
        setEventsError(errorMsg);
        console.error(errorMsg);
        message.error({ content: errorMsg, key: 'eventsLoading', duration: 3 });
      }
    } catch (error) {
      const errorMsg = '获取系统事件失败，请检查网络连接或服务器状态';
      setEventsError(errorMsg);
      console.error('获取系统事件失败:', error);
      message.error({ content: errorMsg, key: 'eventsLoading', duration: 3 });
    } finally {
      setEventsLoading(false);
    }
  };

  useEffect(() => {
    fetchStatus();
    fetchResources();

    // 每30秒刷新一次数据
    const interval = setInterval(() => {
      fetchStatus();
      fetchResources();
    }, 30000);

    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (activeTab === '3') {
      fetchLogs();
    } else if (activeTab === '4') {
      fetchEvents();
    }
  }, [activeTab]);

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';

    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));

    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
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

  const getEventTypeTag = (type: string) => {
    switch (type.toLowerCase()) {
      case 'error':
        return <Tag color="red">错误</Tag>;
      case 'warning':
        return <Tag color="orange">警告</Tag>;
      case 'info':
        return <Tag color="blue">信息</Tag>;
      case 'success':
        return <Tag color="green">成功</Tag>;
      default:
        return <Tag>{type}</Tag>;
    }
  };

  const renderSystemStatus = () => {
    if (!status) return null;

    return (
      <Row gutter={[16, 16]}>
        <Col span={8}>
          <Card title="主机信息">
            <Statistic
              title="主机名"
              value={status.host?.hostname || 'N/A'}
            />
            <Divider />
            <Statistic
              title="操作系统"
              value={`${status.host?.platform || 'N/A'} ${status.host?.platform_version || ''}`}
            />
            <Divider />
            <Statistic
              title="运行时间"
              value={status.host?.uptime || 'N/A'}
            />
          </Card>
        </Col>

        <Col span={8}>
          <Card title="框架信息">
            <Statistic
              title="版本"
              value={status.framework?.version || 'N/A'}
            />
            <Divider />
            <Statistic
              title="启动时间"
              value={status.framework?.start_time ? new Date(status.framework.start_time).toLocaleString() : 'N/A'}
            />
            <Divider />
            <Statistic
              title="运行时间"
              value={status.framework?.uptime || 'N/A'}
            />
          </Card>
        </Col>

        <Col span={8}>
          <Card title="运行时信息">
            <Statistic
              title="Go版本"
              value={status.runtime?.go_version || 'N/A'}
            />
            <Divider />
            <Row>
              <Col span={12}>
                <Statistic
                  title="操作系统"
                  value={status.runtime?.go_os || 'N/A'}
                />
              </Col>
              <Col span={12}>
                <Statistic
                  title="架构"
                  value={status.runtime?.go_arch || 'N/A'}
                />
              </Col>
            </Row>
            <Divider />
            <Row>
              <Col span={12}>
                <Statistic
                  title="CPU核心数"
                  value={status.runtime?.cpu_cores || 'N/A'}
                />
              </Col>
              <Col span={12}>
                <Statistic
                  title="Goroutines"
                  value={status.runtime?.goroutines || 'N/A'}
                />
              </Col>
            </Row>
          </Card>
        </Col>
      </Row>
    );
  };

  const renderSystemResources = () => {
    if (!resources) return null;

    return (
      <Row gutter={[16, 16]}>
        <Col span={8}>
          <Card title="CPU使用率">
            <Statistic
              title="使用率"
              value={resources.cpu?.usage_pct ? `${resources.cpu.usage_pct.toFixed(2)}%` : 'N/A'}
              valueStyle={{ color: (resources.cpu?.usage_pct || 0) > 80 ? '#cf1322' : '#3f8600' }}
            />
            <Divider />
            <Statistic
              title="核心数"
              value={resources.cpu?.cores || 'N/A'}
            />
          </Card>
        </Col>

        <Col span={8}>
          <Card title="内存使用">
            <Statistic
              title="使用率"
              value={resources.memory?.used_pct ? `${resources.memory.used_pct.toFixed(2)}%` : 'N/A'}
              valueStyle={{ color: (resources.memory?.used_pct || 0) > 80 ? '#cf1322' : '#3f8600' }}
            />
            <Divider />
            <Row>
              <Col span={12}>
                <Statistic
                  title="已用内存"
                  value={formatBytes(resources.memory?.used || 0)}
                />
              </Col>
              <Col span={12}>
                <Statistic
                  title="总内存"
                  value={formatBytes(resources.memory?.total || 0)}
                />
              </Col>
            </Row>
          </Card>
        </Col>

        <Col span={8}>
          <Card title="磁盘使用">
            <Statistic
              title="使用率"
              value={resources.disk?.used_pct ? `${resources.disk.used_pct.toFixed(2)}%` : 'N/A'}
              valueStyle={{ color: (resources.disk?.used_pct || 0) > 80 ? '#cf1322' : '#3f8600' }}
            />
            <Divider />
            <Row>
              <Col span={12}>
                <Statistic
                  title="已用空间"
                  value={formatBytes(resources.disk?.used || 0)}
                />
              </Col>
              <Col span={12}>
                <Statistic
                  title="总空间"
                  value={formatBytes(resources.disk?.total || 0)}
                />
              </Col>
            </Row>
          </Card>
        </Col>
      </Row>
    );
  };

  const logColumns = [
    {
      title: '时间',
      dataIndex: 'timestamp',
      key: 'timestamp',
      width: 180,
    },
    {
      title: '级别',
      dataIndex: 'level',
      key: 'level',
      width: 100,
      render: (level: string) => getLevelTag(level),
    },
    {
      title: '消息',
      dataIndex: 'message',
      key: 'message',
    },
    {
      title: '来源',
      dataIndex: 'source',
      key: 'source',
      width: 150,
    },
  ];

  const eventColumns = [
    {
      title: '时间',
      dataIndex: 'timestamp',
      key: 'timestamp',
      width: 180,
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type: string) => getEventTypeTag(type),
    },
    {
      title: '消息',
      dataIndex: 'message',
      key: 'message',
    },
    {
      title: '来源',
      dataIndex: 'source',
      key: 'source',
      width: 150,
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={2}>系统监控</Title>
        <div>
          <Button
            style={{ marginRight: 8 }}
            icon={<ReloadOutlined />}
            onClick={() => {
              message.loading({ content: '正在刷新所有数据...', key: 'refreshAll', duration: 0 });
              Promise.all([
                fetchStatus(),
                fetchResources(),
                fetchLogs(),
                fetchEvents()
              ]).then(() => {
                message.success({ content: '所有数据刷新成功', key: 'refreshAll', duration: 2 });
              }).catch(() => {
                message.error({ content: '部分数据刷新失败，请查看控制台日志', key: 'refreshAll', duration: 3 });
              });
            }}
            loading={loading || logsLoading || eventsLoading}
          >
            刷新所有
          </Button>
          <Button
            type="primary"
            icon={<ReloadOutlined />}
            onClick={() => {
              if (activeTab === '1') {
                fetchStatus();
              } else if (activeTab === '2') {
                fetchResources();
              } else if (activeTab === '3') {
                fetchLogs();
              } else if (activeTab === '4') {
                fetchEvents();
              }
            }}
            loading={activeTab === '3' ? logsLoading : (activeTab === '4' ? eventsLoading : loading)}
          >
            刷新当前
          </Button>
        </div>
      </div>
      <Divider />

      <Tabs activeKey={activeTab} onChange={setActiveTab}>
        <TabPane tab="系统状态" key="1">
          <Card loading={loading}>
            {statusError ? (
              <div style={{ textAlign: 'center', padding: '20px' }}>
                <WarningOutlined style={{ fontSize: '32px', color: '#ff4d4f', marginBottom: '16px' }} />
                <p style={{ color: '#ff4d4f', fontSize: '16px' }}>{statusError}</p>
                <Button type="primary" onClick={fetchStatus} style={{ marginTop: '16px' }}>
                  重试
                </Button>
              </div>
            ) : (
              renderSystemStatus()
            )}
          </Card>
        </TabPane>

        <TabPane tab="资源使用" key="2">
          <Card loading={loading}>
            {resourcesError ? (
              <div style={{ textAlign: 'center', padding: '20px' }}>
                <WarningOutlined style={{ fontSize: '32px', color: '#ff4d4f', marginBottom: '16px' }} />
                <p style={{ color: '#ff4d4f', fontSize: '16px' }}>{resourcesError}</p>
                <Button type="primary" onClick={fetchResources} style={{ marginTop: '16px' }}>
                  重试
                </Button>
              </div>
            ) : (
              renderSystemResources()
            )}
          </Card>
        </TabPane>

        <TabPane tab="系统日志" key="3">
          <Card>
            {logsError ? (
              <div style={{ textAlign: 'center', padding: '20px' }}>
                <WarningOutlined style={{ fontSize: '32px', color: '#ff4d4f', marginBottom: '16px' }} />
                <p style={{ color: '#ff4d4f', fontSize: '16px' }}>{logsError}</p>
                <Button type="primary" onClick={fetchLogs} style={{ marginTop: '16px' }}>
                  重试
                </Button>
              </div>
            ) : (
              <Table
                columns={logColumns}
                dataSource={logs}
                rowKey={(record, index) => `${record.timestamp}-${index}`}
                loading={logsLoading}
                pagination={{ pageSize: 10 }}
                locale={{ emptyText: '暂无日志数据' }}
              />
            )}
          </Card>
        </TabPane>

        <TabPane tab="系统事件" key="4">
          <Card>
            {eventsError ? (
              <div style={{ textAlign: 'center', padding: '20px' }}>
                <WarningOutlined style={{ fontSize: '32px', color: '#ff4d4f', marginBottom: '16px' }} />
                <p style={{ color: '#ff4d4f', fontSize: '16px' }}>{eventsError}</p>
                <Button type="primary" onClick={fetchEvents} style={{ marginTop: '16px' }}>
                  重试
                </Button>
              </div>
            ) : (
              <Table
                columns={eventColumns}
                dataSource={events}
                rowKey={(record, index) => `${record.timestamp}-${index}`}
                loading={eventsLoading}
                pagination={{ pageSize: 10 }}
                locale={{ emptyText: '暂无事件数据' }}
              />
            )}
          </Card>
        </TabPane>
      </Tabs>
    </div>
  );
};

export default SystemMonitor;
