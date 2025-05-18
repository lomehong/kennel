import React, { useEffect, useState, useCallback } from 'react';
import { Card, Row, Col, Button, Table, Tag, Descriptions, Tabs, Timeline, Spin, message, Statistic, Tooltip, Badge } from 'antd';
import {
  ReloadOutlined,
  LinkOutlined,
  DisconnectOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  SyncOutlined,
  SettingOutlined,
  BarChartOutlined,
  FileTextOutlined,
  ClockCircleOutlined,
  SendOutlined,
  DownloadOutlined,
  ExclamationCircleOutlined
} from '@ant-design/icons';
import { getCommStatus, connectComm, disconnectComm, getCommConfig, getCommStats, getCommLogs } from '../../services/api';
import styles from './index.module.css';

const { TabPane } = Tabs;

interface CommStatus {
  status: string;
  connected: boolean;
  timestamp: string;
}

interface CommConfig {
  server: string;
  port: number;
  protocol: string;
  timeout: number;
  retry_interval: number;
  max_retries: number;
  [key: string]: any;
}

interface CommStats {
  status: string;
  connected: boolean;
  timestamp: string;
  sent_messages: number;
  received_messages: number;
  errors: number;
  last_sent_time: string;
  last_received_time: string;
  [key: string]: any;
}

interface CommLog {
  timestamp: string;
  level: string;
  message: string;
  [key: string]: any;
}

const CommManager: React.FC = () => {
  const [status, setStatus] = useState<CommStatus | null>(null);
  const [config, setConfig] = useState<CommConfig | null>(null);
  const [stats, setStats] = useState<CommStats | null>(null);
  const [logs, setLogs] = useState<CommLog[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [connecting, setConnecting] = useState<boolean>(false);
  const [disconnecting, setDisconnecting] = useState<boolean>(false);
  const [refreshing, setRefreshing] = useState<boolean>(false);

  // 获取通讯状态
  const fetchCommStatus = async () => {
    try {
      const response = await getCommStatus();
      setStatus(response.data);
    } catch (error) {
      console.error('获取通讯状态失败:', error);
      message.error('获取通讯状态失败');
    }
  };

  // 获取通讯配置
  const fetchCommConfig = async () => {
    try {
      const response = await getCommConfig();
      setConfig(response.data.config);
    } catch (error) {
      console.error('获取通讯配置失败:', error);
      message.error('获取通讯配置失败');
    }
  };

  // 获取通讯统计信息
  const fetchCommStats = async () => {
    try {
      const response = await getCommStats();
      setStats(response.data);
    } catch (error) {
      console.error('获取通讯统计信息失败:', error);
      message.error('获取通讯统计信息失败');
    }
  };

  // 获取通讯日志
  const fetchCommLogs = async () => {
    try {
      const response = await getCommLogs();
      setLogs(response.data.logs);
    } catch (error) {
      console.error('获取通讯日志失败:', error);
      message.error('获取通讯日志失败');
    }
  };

  // 刷新所有数据
  const refreshData = useCallback(async () => {
    setRefreshing(true);
    try {
      await Promise.all([
        fetchCommStatus(),
        fetchCommConfig(),
        fetchCommStats(),
        fetchCommLogs(),
      ]);
      message.success('数据已刷新');
    } catch (error) {
      console.error('刷新数据失败:', error);
      message.error('刷新数据失败');
    } finally {
      setRefreshing(false);
      setLoading(false);
    }
  }, []);

  // 连接到服务器
  const handleConnect = async () => {
    setConnecting(true);
    try {
      await connectComm();
      message.success('成功连接到服务器');
      await refreshData();
    } catch (error) {
      console.error('连接服务器失败:', error);
      message.error('连接服务器失败');
    } finally {
      setConnecting(false);
    }
  };

  // 断开连接
  const handleDisconnect = async () => {
    setDisconnecting(true);
    try {
      await disconnectComm();
      message.success('成功断开连接');
      await refreshData();
    } catch (error) {
      console.error('断开连接失败:', error);
      message.error('断开连接失败');
    } finally {
      setDisconnecting(false);
    }
  };

  // 初始化加载数据
  useEffect(() => {
    refreshData();
    // 设置定时刷新
    const timer = setInterval(() => {
      fetchCommStatus();
      fetchCommStats();
    }, 5000);
    return () => clearInterval(timer);
  }, [refreshData]);

  // 渲染状态标签
  const renderStatusTag = (status: string, connected: boolean) => {
    if (connected) {
      return (
        <Badge status="success" text={
          <Tag icon={<CheckCircleOutlined />} color="success">
            已连接
          </Tag>
        } />
      );
    }
    if (status === 'connecting') {
      return (
        <Badge status="processing" text={
          <Tag icon={<SyncOutlined spin />} color="processing">
            连接中
          </Tag>
        } />
      );
    }
    if (status === 'disconnecting') {
      return (
        <Badge status="warning" text={
          <Tag icon={<SyncOutlined spin />} color="warning">
            断开中
          </Tag>
        } />
      );
    }
    return (
      <Badge status="error" text={
        <Tag icon={<CloseCircleOutlined />} color="error">
          未连接
        </Tag>
      } />
    );
  };

  // 渲染日志级别标签
  const renderLogLevelTag = (level: string) => {
    switch (level.toLowerCase()) {
      case 'error':
        return <Tag icon={<ExclamationCircleOutlined />} color="error">错误</Tag>;
      case 'warn':
        return <Tag icon={<ExclamationCircleOutlined />} color="warning">警告</Tag>;
      case 'info':
        return <Tag icon={<FileTextOutlined />} color="processing">信息</Tag>;
      case 'debug':
        return <Tag icon={<FileTextOutlined />} color="success">调试</Tag>;
      default:
        return <Tag>{level}</Tag>;
    }
  };

  // 格式化时间
  const formatTime = (timeStr: string) => {
    try {
      const date = new Date(timeStr);
      return date.toLocaleString();
    } catch (e) {
      return timeStr;
    }
  };

  return (
    <div className={styles.container}>
      <Card
        title="通讯管理"
        extra={
          <Button
            icon={<ReloadOutlined />}
            onClick={refreshData}
            loading={refreshing}
          >
            刷新
          </Button>
        }
      >
        {loading ? (
          <div className={styles.loading}>
            <Spin size="large" />
            <p>加载中...</p>
          </div>
        ) : (
          <>
            <Row gutter={[16, 16]}>
              <Col span={24}>
                <Card title="通讯状态" className={styles.statusCard}>
                  <Row align="middle">
                    <Col span={12}>
                      <div className={styles.statusInfo}>
                        <div className={styles.statusLabel}>当前状态:</div>
                        <div className={styles.statusValue}>
                          {status && renderStatusTag(status.status, status.connected)}
                        </div>
                      </div>
                      <div className={styles.statusInfo}>
                        <div className={styles.statusLabel}>最后更新:</div>
                        <div className={styles.statusValue}>
                          {status?.timestamp ? formatTime(status.timestamp) : '-'}
                        </div>
                      </div>
                    </Col>
                    <Col span={12} className={styles.statusActions}>
                      <Button
                        type="primary"
                        icon={<LinkOutlined />}
                        onClick={handleConnect}
                        loading={connecting}
                        disabled={status?.connected || connecting}
                        className={styles.actionButton}
                      >
                        连接
                      </Button>
                      <Button
                        danger
                        icon={<DisconnectOutlined />}
                        onClick={handleDisconnect}
                        loading={disconnecting}
                        disabled={!status?.connected || disconnecting}
                        className={styles.actionButton}
                      >
                        断开连接
                      </Button>
                    </Col>
                  </Row>
                </Card>
              </Col>
            </Row>

            <Tabs defaultActiveKey="stats">
              <TabPane
                tab={
                  <span>
                    <BarChartOutlined />
                    统计信息
                  </span>
                }
                key="stats"
              >
                <div className={styles.tabContent}>
                  {stats && (
                    <Row gutter={[24, 24]}>
                      <Col span={8}>
                        <Card>
                          <Statistic
                            title="发送消息数"
                            value={stats.sent_messages || 0}
                            prefix={<SendOutlined />}
                            valueStyle={{ color: '#3f8600' }}
                          />
                        </Card>
                      </Col>
                      <Col span={8}>
                        <Card>
                          <Statistic
                            title="接收消息数"
                            value={stats.received_messages || 0}
                            prefix={<DownloadOutlined />}
                            valueStyle={{ color: '#1890ff' }}
                          />
                        </Card>
                      </Col>
                      <Col span={8}>
                        <Card>
                          <Statistic
                            title="错误数"
                            value={stats.errors || 0}
                            prefix={<ExclamationCircleOutlined />}
                            valueStyle={{ color: stats.errors > 0 ? '#cf1322' : '#3f8600' }}
                          />
                        </Card>
                      </Col>
                      <Col span={24}>
                        <Descriptions bordered column={2}>
                          <Descriptions.Item label="连接状态">{renderStatusTag(stats.status, stats.connected)}</Descriptions.Item>
                          <Descriptions.Item label="最后更新时间">
                            {stats.timestamp ? formatTime(stats.timestamp) : '-'}
                          </Descriptions.Item>
                          <Descriptions.Item label="最后发送时间">
                            {stats.last_sent_time ? formatTime(stats.last_sent_time) : '-'}
                          </Descriptions.Item>
                          <Descriptions.Item label="最后接收时间">
                            {stats.last_received_time ? formatTime(stats.last_received_time) : '-'}
                          </Descriptions.Item>
                        </Descriptions>
                      </Col>
                    </Row>
                  )}
                </div>
              </TabPane>
              <TabPane
                tab={
                  <span>
                    <SettingOutlined />
                    配置信息
                  </span>
                }
                key="config"
              >
                <div className={styles.tabContent}>
                  {config && (
                    <Row gutter={[24, 24]}>
                      <Col span={24}>
                        <Card title="服务器配置">
                          <Descriptions bordered column={2}>
                            <Descriptions.Item label="服务器地址">{config.server || '-'}</Descriptions.Item>
                            <Descriptions.Item label="端口">{config.port || '-'}</Descriptions.Item>
                            <Descriptions.Item label="协议">{config.protocol || '-'}</Descriptions.Item>
                            <Descriptions.Item label="超时时间">{config.timeout || '-'} 秒</Descriptions.Item>
                          </Descriptions>
                        </Card>
                      </Col>
                      <Col span={24}>
                        <Card title="重试配置">
                          <Descriptions bordered column={2}>
                            <Descriptions.Item label="重试间隔">{config.retry_interval || '-'} 秒</Descriptions.Item>
                            <Descriptions.Item label="最大重试次数">{config.max_retries || '-'}</Descriptions.Item>
                          </Descriptions>
                        </Card>
                      </Col>
                    </Row>
                  )}
                </div>
              </TabPane>
              <TabPane
                tab={
                  <span>
                    <FileTextOutlined />
                    通讯日志
                  </span>
                }
                key="logs"
              >
                <div className={styles.tabContent}>
                  <Table
                    dataSource={logs}
                    rowKey={(record, index) => `${record.timestamp}-${index}`}
                    pagination={{ pageSize: 10 }}
                    className={styles.logTable}
                    columns={[
                      {
                        title: '时间',
                        dataIndex: 'timestamp',
                        key: 'timestamp',
                        width: 180,
                        render: (text) => (
                          <Tooltip title={formatTime(text)}>
                            <span className={styles.logTime}>
                              <ClockCircleOutlined /> {formatTime(text)}
                            </span>
                          </Tooltip>
                        ),
                      },
                      {
                        title: '级别',
                        dataIndex: 'level',
                        key: 'level',
                        width: 100,
                        render: (text) => (
                          <span className={styles.logLevel}>
                            {renderLogLevelTag(text)}
                          </span>
                        ),
                      },
                      {
                        title: '消息',
                        dataIndex: 'message',
                        key: 'message',
                        render: (text) => (
                          <div className={styles.logMessage}>{text}</div>
                        ),
                      },
                    ]}
                  />
                </div>
              </TabPane>
            </Tabs>
          </>
        )}
      </Card>
    </div>
  );
};

export default CommManager;
