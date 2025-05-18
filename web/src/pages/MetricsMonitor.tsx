import React, { useEffect, useState } from 'react';
import { Card, Row, Col, Typography, Divider, Statistic, Button, Tabs, Table, Tag } from 'antd';
import { ReloadOutlined, LineChartOutlined } from '@ant-design/icons';
import { Line } from '@ant-design/plots';
import { metricsApi } from '../services/api';

const { Title } = Typography;
const { TabPane } = Tabs;

interface MetricsData {
  comm: any;
  system: any;
  time: string;
}

const MetricsMonitor: React.FC = () => {
  const [metrics, setMetrics] = useState<MetricsData | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [metricsHistory, setMetricsHistory] = useState<any[]>([]);
  const [activeTab, setActiveTab] = useState<string>('1');

  const fetchMetrics = async () => {
    setLoading(true);
    try {
      const response = await metricsApi.getMetrics();
      setMetrics(response.data);

      // 添加到历史记录
      setMetricsHistory(prev => {
        const newHistory = [...prev, {
          time: response.data.time,
          ...response.data.comm,
          ...response.data.system
        }];

        // 只保留最近30条记录
        if (newHistory.length > 30) {
          return newHistory.slice(newHistory.length - 30);
        }
        return newHistory;
      });
    } catch (error) {
      console.error('获取指标失败:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchMetrics();

    // 每5秒刷新一次数据
    const interval = setInterval(fetchMetrics, 5000);

    return () => clearInterval(interval);
  }, []);

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';

    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));

    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const getConnectionStatusTag = () => {
    if (!metrics?.comm?.connected) {
      return <Tag color="red">未连接</Tag>;
    }
    return <Tag color="green">已连接</Tag>;
  };

  const renderConnectionMetrics = () => {
    if (!metrics?.comm) return null;

    return (
      <Row gutter={[16, 16]}>
        <Col span={8}>
          <Card title="连接状态">
            <Statistic
              title="状态"
              value={metrics.comm.current_state || '未知'}
              prefix={getConnectionStatusTag()}
            />
            <Divider />
            <Row>
              <Col span={12}>
                <Statistic
                  title="连接次数"
                  value={metrics.comm.connect_count || 0}
                />
              </Col>
              <Col span={12}>
                <Statistic
                  title="重连次数"
                  value={metrics.comm.reconnect_count || 0}
                />
              </Col>
            </Row>
          </Card>
        </Col>

        <Col span={8}>
          <Card title="消息统计">
            <Row>
              <Col span={12}>
                <Statistic
                  title="发送消息"
                  value={metrics.comm.sent_message_count || 0}
                />
              </Col>
              <Col span={12}>
                <Statistic
                  title="接收消息"
                  value={metrics.comm.received_message_count || 0}
                />
              </Col>
            </Row>
            <Divider />
            <Row>
              <Col span={12}>
                <Statistic
                  title="发送字节"
                  value={formatBytes(metrics.comm.sent_bytes || 0)}
                />
              </Col>
              <Col span={12}>
                <Statistic
                  title="接收字节"
                  value={formatBytes(metrics.comm.received_bytes || 0)}
                />
              </Col>
            </Row>
          </Card>
        </Col>

        <Col span={8}>
          <Card title="延迟指标">
            <Statistic
              title="平均延迟"
              value={metrics.comm.avg_latency ? `${metrics.comm.avg_latency.toFixed(2)} ms` : 'N/A'}
            />
            <Divider />
            <Row>
              <Col span={12}>
                <Statistic
                  title="最小延迟"
                  value={metrics.comm.min_latency >= 0 ? `${metrics.comm.min_latency} ms` : 'N/A'}
                />
              </Col>
              <Col span={12}>
                <Statistic
                  title="最大延迟"
                  value={metrics.comm.max_latency ? `${metrics.comm.max_latency} ms` : 'N/A'}
                />
              </Col>
            </Row>
          </Card>
        </Col>
      </Row>
    );
  };

  const renderCompressionMetrics = () => {
    if (!metrics?.comm) return null;

    return (
      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col span={12}>
          <Card title="压缩指标">
            <Statistic
              title="压缩消息数"
              value={metrics.comm.compressed_count || 0}
            />
            <Divider />
            <Row>
              <Col span={12}>
                <Statistic
                  title="压缩前字节"
                  value={formatBytes(metrics.comm.compressed_bytes || 0)}
                />
              </Col>
              <Col span={12}>
                <Statistic
                  title="压缩后字节"
                  value={formatBytes(metrics.comm.compressed_bytes_after || 0)}
                />
              </Col>
            </Row>
            <Divider />
            <Statistic
              title="压缩率"
              value={metrics.comm.compression_ratio ? `${(metrics.comm.compression_ratio * 100).toFixed(2)}%` : 'N/A'}
            />
          </Card>
        </Col>

        <Col span={12}>
          <Card title="心跳指标">
            <Row>
              <Col span={12}>
                <Statistic
                  title="发送心跳"
                  value={metrics.comm.heartbeat_sent_count || 0}
                />
              </Col>
              <Col span={12}>
                <Statistic
                  title="接收心跳"
                  value={metrics.comm.heartbeat_received_count || 0}
                />
              </Col>
            </Row>
            <Divider />
            <Statistic
              title="心跳错误"
              value={metrics.comm.heartbeat_error_count || 0}
            />
            {metrics.comm.last_heartbeat_time && (
              <>
                <Divider />
                <Statistic
                  title="最后心跳时间"
                  value={new Date(metrics.comm.last_heartbeat_time).toLocaleString()}
                />
              </>
            )}
          </Card>
        </Col>
      </Row>
    );
  };

  const renderSystemMetrics = () => {
    if (!metrics?.system) return null;

    return (
      <Row gutter={[16, 16]}>
        <Col span={8}>
          <Card title="CPU使用率">
            <Statistic
              title="使用率"
              value={metrics.system.cpu?.usage_pct ? `${metrics.system.cpu.usage_pct.toFixed(2)}%` : 'N/A'}
              valueStyle={{ color: (metrics.system.cpu?.usage_pct || 0) > 80 ? '#cf1322' : '#3f8600' }}
            />
            <Divider />
            <Statistic
              title="核心数"
              value={metrics.system.cpu?.cores || 'N/A'}
            />
          </Card>
        </Col>

        <Col span={8}>
          <Card title="内存使用">
            <Statistic
              title="使用率"
              value={metrics.system.memory?.used_pct ? `${metrics.system.memory.used_pct.toFixed(2)}%` : 'N/A'}
              valueStyle={{ color: (metrics.system.memory?.used_pct || 0) > 80 ? '#cf1322' : '#3f8600' }}
            />
            <Divider />
            <Row>
              <Col span={12}>
                <Statistic
                  title="已用内存"
                  value={formatBytes(metrics.system.memory?.used || 0)}
                />
              </Col>
              <Col span={12}>
                <Statistic
                  title="总内存"
                  value={formatBytes(metrics.system.memory?.total || 0)}
                />
              </Col>
            </Row>
          </Card>
        </Col>

        <Col span={8}>
          <Card title="运行时指标">
            <Statistic
              title="Goroutines"
              value={metrics.system.runtime?.goroutines || 0}
            />
            <Divider />
            <Row>
              <Col span={12}>
                <Statistic
                  title="堆分配"
                  value={formatBytes(metrics.system.runtime?.heap_alloc || 0)}
                />
              </Col>
              <Col span={12}>
                <Statistic
                  title="GC次数"
                  value={metrics.system.runtime?.gc_cycles || 0}
                />
              </Col>
            </Row>
          </Card>
        </Col>
      </Row>
    );
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={2}>指标监控</Title>
        <Button
          type="primary"
          icon={<ReloadOutlined />}
          onClick={fetchMetrics}
          loading={loading}
        >
          刷新
        </Button>
      </div>
      <Divider />

      <Tabs activeKey={activeTab} onChange={setActiveTab}>
        <TabPane tab="通讯指标" key="1">
          <Card loading={loading}>
            {renderConnectionMetrics()}
            {renderCompressionMetrics()}
          </Card>
        </TabPane>

        <TabPane tab="系统指标" key="2">
          <Card loading={loading}>
            {renderSystemMetrics()}
          </Card>
        </TabPane>
      </Tabs>
    </div>
  );
};

export default MetricsMonitor;
