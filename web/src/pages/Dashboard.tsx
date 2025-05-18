import React, { useEffect, useState } from 'react';
import { Card, Row, Col, Statistic, Typography, Divider, List, Tag } from 'antd';
import { 
  AppstoreOutlined, 
  LineChartOutlined, 
  CheckCircleOutlined, 
  CloseCircleOutlined,
  ClockCircleOutlined,
  WarningOutlined
} from '@ant-design/icons';
import { systemApi, metricsApi } from '../services/api';

const { Title } = Typography;

const Dashboard: React.FC = () => {
  const [systemStatus, setSystemStatus] = useState<any>(null);
  const [metrics, setMetrics] = useState<any>(null);
  const [loading, setLoading] = useState<boolean>(true);

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      try {
        const [systemRes, metricsRes] = await Promise.all([
          systemApi.getSystemStatus(),
          metricsApi.getMetrics()
        ]);
        setSystemStatus(systemRes.data);
        setMetrics(metricsRes.data);
      } catch (error) {
        console.error('获取数据失败:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
    
    // 每30秒刷新一次数据
    const interval = setInterval(fetchData, 30000);
    
    return () => clearInterval(interval);
  }, []);

  const getConnectionStatusTag = () => {
    if (!metrics?.comm?.connected) {
      return <Tag color="red">未连接</Tag>;
    }
    return <Tag color="green">已连接</Tag>;
  };

  return (
    <div>
      <Title level={2}>仪表盘</Title>
      <Divider />
      
      <Row gutter={[16, 16]}>
        <Col span={8}>
          <Card title="系统状态" loading={loading}>
            {systemStatus && (
              <>
                <Statistic
                  title="运行时间"
                  value={systemStatus.framework?.uptime || '未知'}
                  prefix={<ClockCircleOutlined />}
                />
                <Divider />
                <Statistic
                  title="版本"
                  value={systemStatus.framework?.version || '未知'}
                />
              </>
            )}
          </Card>
        </Col>
        
        <Col span={8}>
          <Card title="插件状态" loading={loading}>
            {systemStatus && systemStatus.plugins && (
              <>
                <Statistic
                  title="已加载插件"
                  value={systemStatus.plugins.total || 0}
                  prefix={<AppstoreOutlined />}
                />
                <Divider />
                <Row>
                  <Col span={12}>
                    <Statistic
                      title="已启用"
                      value={systemStatus.plugins.enabled || 0}
                      valueStyle={{ color: '#3f8600' }}
                      prefix={<CheckCircleOutlined />}
                    />
                  </Col>
                  <Col span={12}>
                    <Statistic
                      title="已禁用"
                      value={systemStatus.plugins.disabled || 0}
                      valueStyle={{ color: '#cf1322' }}
                      prefix={<CloseCircleOutlined />}
                    />
                  </Col>
                </Row>
              </>
            )}
          </Card>
        </Col>
        
        <Col span={8}>
          <Card title="通讯状态" loading={loading}>
            {metrics && metrics.comm && (
              <>
                <Statistic
                  title="连接状态"
                  value={metrics.comm.current_state || '未知'}
                  prefix={getConnectionStatusTag()}
                />
                <Divider />
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
              </>
            )}
          </Card>
        </Col>
      </Row>
      
      <Divider />
      
      <Row gutter={[16, 16]}>
        <Col span={12}>
          <Card title="系统资源" loading={loading}>
            {metrics && metrics.system && (
              <List
                size="small"
                bordered
                dataSource={[
                  {
                    title: 'CPU使用率',
                    value: `${(metrics.system.cpu?.usage_pct || 0).toFixed(2)}%`,
                    color: (metrics.system.cpu?.usage_pct || 0) > 80 ? '#cf1322' : '#3f8600'
                  },
                  {
                    title: '内存使用率',
                    value: `${(metrics.system.memory?.used_pct || 0).toFixed(2)}%`,
                    color: (metrics.system.memory?.used_pct || 0) > 80 ? '#cf1322' : '#3f8600'
                  },
                  {
                    title: '磁盘使用率',
                    value: `${(metrics.system.disk?.used_pct || 0).toFixed(2)}%`,
                    color: (metrics.system.disk?.used_pct || 0) > 80 ? '#cf1322' : '#3f8600'
                  },
                  {
                    title: 'Goroutines',
                    value: metrics.system.runtime?.goroutines || 0
                  }
                ]}
                renderItem={(item: any) => (
                  <List.Item>
                    <div style={{ display: 'flex', justifyContent: 'space-between', width: '100%' }}>
                      <span>{item.title}</span>
                      <span style={{ color: item.color }}>{item.value}</span>
                    </div>
                  </List.Item>
                )}
              />
            )}
          </Card>
        </Col>
        
        <Col span={12}>
          <Card title="最近事件" loading={loading} extra={<WarningOutlined />}>
            <p>暂无事件</p>
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default Dashboard;
