import React, { useState, useEffect } from 'react';
import {
  Card,
  Tabs,
  Button,
  Form,
  Input,
  InputNumber,
  Switch,
  Table,
  message,
  Space,
  Typography,
  Divider,
  Select,
  Progress,
  Descriptions,
  Radio,
  Alert,
  Tooltip,
} from 'antd';
import { ReloadOutlined, SendOutlined, LinkOutlined, LockOutlined, CompressOutlined, BarChartOutlined, ApiOutlined } from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-layout';
import { getCommTestHistory, testCommConnection, testCommSendReceive, testCommEncryption, testCommCompression, testCommPerformance } from '@/services/api';
import styles from './CommTest.less';

const { Title, Text, Paragraph } = Typography;
const { TabPane } = Tabs;
const { Option } = Select;

const CommTest: React.FC = () => {
  // 状态
  const [connectionForm] = Form.useForm();
  const [sendReceiveForm] = Form.useForm();
  const [encryptionForm] = Form.useForm();
  const [compressionForm] = Form.useForm();
  const [performanceForm] = Form.useForm();

  const [connectionLoading, setConnectionLoading] = useState<boolean>(false);
  const [sendReceiveLoading, setSendReceiveLoading] = useState<boolean>(false);
  const [encryptionLoading, setEncryptionLoading] = useState<boolean>(false);
  const [compressionLoading, setCompressionLoading] = useState<boolean>(false);
  const [performanceLoading, setPerformanceLoading] = useState<boolean>(false);
  const [historyLoading, setHistoryLoading] = useState<boolean>(false);

  const [connectionResult, setConnectionResult] = useState<any>(null);
  const [sendReceiveResult, setSendReceiveResult] = useState<any>(null);
  const [encryptionResult, setEncryptionResult] = useState<any>(null);
  const [compressionResult, setCompressionResult] = useState<any>(null);
  const [performanceResult, setPerformanceResult] = useState<any>(null);
  const [testHistory, setTestHistory] = useState<any[]>([]);

  // 模拟API状态
  const [useMockApi, setUseMockApi] = useState<boolean>(() => {
    // 从localStorage读取状态，默认为true
    const savedValue = localStorage.getItem('use_mock_api');
    return savedValue !== 'false';
  });

  // 加载测试历史记录
  const loadTestHistory = async () => {
    setHistoryLoading(true);
    try {
      const response = await getCommTestHistory();
      if (response && Array.isArray(response)) {
        setTestHistory(response);
      } else {
        setTestHistory([]);
      }
    } catch (error) {
      // 不显示错误消息，因为在模拟API模式下可能会失败
      console.warn('加载测试历史记录失败:', error);
      setTestHistory([]);
    } finally {
      setHistoryLoading(false);
    }
  };

  // 切换模拟API
  const toggleMockApi = () => {
    const newValue = !useMockApi;
    setUseMockApi(newValue);
    localStorage.setItem('use_mock_api', newValue ? 'true' : 'false');

    // 刷新页面以应用新设置
    message.success(`已${newValue ? '启用' : '禁用'}模拟API，正在刷新页面...`);
    setTimeout(() => {
      window.location.reload();
    }, 1500);
  };

  // 初始加载
  useEffect(() => {
    loadTestHistory();
  }, []);

  // 测试连接
  const handleTestConnection = async (values: any) => {
    setConnectionLoading(true);
    setConnectionResult(null);
    try {
      const response = await testCommConnection(values);
      setConnectionResult(response);
      if (response.success) {
        message.success('连接测试成功');
      } else {
        message.error(`连接测试失败: ${response.message}`);
      }
    } catch (error) {
      message.error('连接测试失败');
      console.error('连接测试失败:', error);
    } finally {
      setConnectionLoading(false);
      loadTestHistory();
    }
  };

  // 测试发送和接收
  const handleTestSendReceive = async (values: any) => {
    setSendReceiveLoading(true);
    setSendReceiveResult(null);
    try {
      // 将JSON字符串解析为对象
      const payload = JSON.parse(values.payload);

      // 创建新的请求对象，包含解析后的payload
      const requestData = {
        ...values,
        payload: payload
      };

      const response = await testCommSendReceive(requestData);
      setSendReceiveResult(response);
      if (response.success) {
        message.success('发送和接收测试成功');
      } else {
        message.error(`发送和接收测试失败: ${response.message}`);
      }
    } catch (error) {
      message.error('发送和接收测试失败');
      console.error('发送和接收测试失败:', error);
    } finally {
      setSendReceiveLoading(false);
      loadTestHistory();
    }
  };

  // 测试加密
  const handleTestEncryption = async (values: any) => {
    setEncryptionLoading(true);
    setEncryptionResult(null);
    try {
      const response = await testCommEncryption(values);
      setEncryptionResult(response);
      if (response.success) {
        message.success('加密测试成功');
      } else {
        message.error(`加密测试失败: ${response.message}`);
      }
    } catch (error) {
      message.error('加密测试失败');
      console.error('加密测试失败:', error);
    } finally {
      setEncryptionLoading(false);
      loadTestHistory();
    }
  };

  // 测试压缩
  const handleTestCompression = async (values: any) => {
    setCompressionLoading(true);
    setCompressionResult(null);
    try {
      const response = await testCommCompression(values);
      setCompressionResult(response);
      if (response.success) {
        message.success('压缩测试成功');
      } else {
        message.error(`压缩测试失败: ${response.message}`);
      }
    } catch (error) {
      message.error('压缩测试失败');
      console.error('压缩测试失败:', error);
    } finally {
      setCompressionLoading(false);
      loadTestHistory();
    }
  };

  // 测试性能
  const handleTestPerformance = async (values: any) => {
    setPerformanceLoading(true);
    setPerformanceResult(null);
    try {
      const response = await testCommPerformance(values);
      setPerformanceResult(response);
      if (response.success) {
        message.success('性能测试成功');
      } else {
        message.error(`性能测试失败: ${response.message}`);
      }
    } catch (error) {
      message.error('性能测试失败');
      console.error('性能测试失败:', error);
    } finally {
      setPerformanceLoading(false);
      loadTestHistory();
    }
  };

  // 历史记录列
  const historyColumns = [
    {
      title: '时间',
      dataIndex: 'timestamp',
      key: 'timestamp',
      width: 180,
      render: (timestamp: string) => timestamp || '-',
    },
    {
      title: '测试类型',
      dataIndex: 'type',
      key: 'type',
      width: 120,
      render: (type: string) => {
        switch (type) {
          case 'connection':
            return <span><LinkOutlined /> 连接测试</span>;
          case 'send-receive':
            return <span><SendOutlined /> 发送接收</span>;
          case 'encryption':
            return <span><LockOutlined /> 加密测试</span>;
          case 'compression':
            return <span><CompressOutlined /> 压缩测试</span>;
          case 'performance':
            return <span><BarChartOutlined /> 性能测试</span>;
          default:
            return type;
        }
      },
    },
    {
      title: '结果',
      dataIndex: 'success',
      key: 'success',
      width: 100,
      render: (success: boolean) => (
        success ? <span style={{ color: 'green' }}>成功</span> : <span style={{ color: 'red' }}>失败</span>
      ),
    },
    {
      title: '耗时',
      dataIndex: 'duration',
      key: 'duration',
      width: 120,
    },
    {
      title: '详情',
      key: 'details',
      render: (text: string, record: any) => (
        <Button type="link" onClick={() => message.info('查看详情功能待实现')}>
          查看详情
        </Button>
      ),
    },
  ];

  return (
    <PageContainer
      title="通信测试"
      extra={[
        <Tooltip key="mock-api-toggle" title={`${useMockApi ? '禁用' : '启用'}模拟API`}>
          <Button
            type={useMockApi ? 'primary' : 'default'}
            icon={<ApiOutlined />}
            onClick={toggleMockApi}
          >
            {useMockApi ? '使用模拟API' : '使用真实API'}
          </Button>
        </Tooltip>
      ]}
    >
      {useMockApi && (
        <Alert
          message="模拟API模式已启用"
          description="当前使用的是模拟API，所有请求将返回模拟数据。这对于开发和测试非常有用，但不会与真实服务器通信。"
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          action={
            <Button size="small" onClick={toggleMockApi}>
              切换到真实API
            </Button>
          }
        />
      )}
      <Tabs defaultActiveKey="connection">
        <TabPane tab={<span><LinkOutlined /> 连接测试</span>} key="connection">
          <Card>
            <Title level={4}>连接测试</Title>
            <Paragraph>
              测试与服务器的连接状态，验证通信框架是否能够正常建立连接。
            </Paragraph>
            <Form
              form={connectionForm}
              layout="vertical"
              onFinish={handleTestConnection}
              initialValues={{
                server_url: 'ws://localhost:8080/ws',
                timeout: 10,
              }}
            >
              <Form.Item
                name="server_url"
                label="服务器地址"
                rules={[{ required: true, message: '请输入服务器地址' }]}
              >
                <Input placeholder="输入WebSocket服务器地址，例如：ws://localhost:8080/ws" />
              </Form.Item>
              <Form.Item
                name="timeout"
                label="超时时间（秒）"
                rules={[{ required: true, message: '请输入超时时间' }]}
              >
                <InputNumber min={1} max={60} style={{ width: '100%' }} />
              </Form.Item>
              <Form.Item>
                <Button type="primary" htmlType="submit" loading={connectionLoading} icon={<LinkOutlined />}>
                  测试连接
                </Button>
              </Form.Item>
            </Form>

            {connectionResult && (
              <div style={{ marginTop: 16 }}>
                <Divider>测试结果</Divider>
                <Descriptions bordered>
                  <Descriptions.Item label="连接状态" span={3}>
                    {connectionResult.success ? (
                      <span style={{ color: 'green' }}>连接成功</span>
                    ) : (
                      <span style={{ color: 'red' }}>连接失败: {connectionResult.message}</span>
                    )}
                  </Descriptions.Item>
                  <Descriptions.Item label="耗时" span={3}>{connectionResult.duration}</Descriptions.Item>
                </Descriptions>
              </div>
            )}
          </Card>
        </TabPane>

        <TabPane tab={<span><SendOutlined /> 发送接收</span>} key="send-receive">
          <Card>
            <Title level={4}>发送和接收测试</Title>
            <Paragraph>
              测试通信框架的消息发送和接收功能，验证数据传输的正确性。
            </Paragraph>
            <Form
              form={sendReceiveForm}
              layout="vertical"
              onFinish={handleTestSendReceive}
              initialValues={{
                message_type: 'command',
                payload: JSON.stringify({ command: 'ping', data: 'test' }, null, 2),
                timeout: 10,
                use_mock: true,
              }}
            >
              <Form.Item
                name="message_type"
                label="消息类型"
                rules={[{ required: true, message: '请选择消息类型' }]}
              >
                <Select>
                  <Option value="command">命令</Option>
                  <Option value="data">数据</Option>
                  <Option value="event">事件</Option>
                  <Option value="response">响应</Option>
                </Select>
              </Form.Item>
              <Form.Item
                name="payload"
                label="消息内容 (JSON)"
                rules={[
                  { required: true, message: '请输入消息内容' },
                  {
                    validator: (_, value) => {
                      try {
                        JSON.parse(value);
                        return Promise.resolve();
                      } catch (error) {
                        return Promise.reject('无效的JSON格式');
                      }
                    },
                  },
                ]}
              >
                <Input.TextArea rows={6} placeholder="输入JSON格式的消息内容" />
              </Form.Item>
              <Form.Item
                name="timeout"
                label="超时时间（秒）"
                rules={[{ required: true, message: '请输入超时时间' }]}
              >
                <InputNumber min={1} max={60} style={{ width: '100%' }} />
              </Form.Item>
              <Form.Item
                name="use_mock"
                label="使用模拟模式"
                valuePropName="checked"
                tooltip="当未连接到服务器时，使用模拟响应进行测试"
              >
                <Switch />
              </Form.Item>
              <Form.Item>
                <Button type="primary" htmlType="submit" loading={sendReceiveLoading} icon={<SendOutlined />}>
                  发送消息
                </Button>
              </Form.Item>
            </Form>

            {sendReceiveResult && (
              <div style={{ marginTop: 16 }}>
                <Divider>测试结果</Divider>
                <Descriptions bordered>
                  <Descriptions.Item label="发送状态" span={3}>
                    {sendReceiveResult.success ? (
                      <span style={{ color: 'green' }}>{sendReceiveResult.message || '发送成功'}</span>
                    ) : (
                      <span style={{ color: 'red' }}>发送失败: {sendReceiveResult.message}</span>
                    )}
                  </Descriptions.Item>
                  <Descriptions.Item label="耗时" span={3}>{sendReceiveResult.duration}</Descriptions.Item>
                  {sendReceiveResult.response && sendReceiveResult.response.mock && (
                    <Descriptions.Item label="模式" span={3}>
                      <span style={{ color: 'blue' }}>使用模拟响应</span>
                    </Descriptions.Item>
                  )}
                </Descriptions>
                {sendReceiveResult.success && sendReceiveResult.response && (
                  <div style={{ marginTop: 16 }}>
                    <Title level={5}>响应数据</Title>
                    <Input.TextArea
                      rows={8}
                      value={JSON.stringify(sendReceiveResult.response, null, 2)}
                      readOnly
                    />
                  </div>
                )}
              </div>
            )}
          </Card>
        </TabPane>

        <TabPane tab={<span><LockOutlined /> 加密测试</span>} key="encryption">
          <Card>
            <Title level={4}>加密测试</Title>
            <Paragraph>
              测试通信框架的加密功能，验证数据加密和解密的正确性和性能。
            </Paragraph>
            <Form
              form={encryptionForm}
              layout="vertical"
              onFinish={handleTestEncryption}
              initialValues={{
                data: '这是一段测试数据，用于测试通信框架的加密功能。',
                encryption_key: 'test-key-12345',
              }}
            >
              <Form.Item
                name="data"
                label="测试数据"
                rules={[{ required: true, message: '请输入测试数据' }]}
              >
                <Input.TextArea rows={4} placeholder="输入要加密的测试数据" />
              </Form.Item>
              <Form.Item
                name="encryption_key"
                label="加密密钥"
                rules={[{ required: true, message: '请输入加密密钥' }]}
              >
                <Input placeholder="输入加密密钥" />
              </Form.Item>
              <Form.Item>
                <Button type="primary" htmlType="submit" loading={encryptionLoading} icon={<LockOutlined />}>
                  测试加密
                </Button>
              </Form.Item>
            </Form>

            {encryptionResult && (
              <div style={{ marginTop: 16 }}>
                <Divider>测试结果</Divider>
                <Descriptions bordered>
                  <Descriptions.Item label="加密状态" span={3}>
                    {encryptionResult.success ? (
                      <span style={{ color: 'green' }}>加密成功</span>
                    ) : (
                      <span style={{ color: 'red' }}>加密失败: {encryptionResult.message}</span>
                    )}
                  </Descriptions.Item>
                  <Descriptions.Item label="原始大小" span={1}>{encryptionResult.original_size} 字节</Descriptions.Item>
                  <Descriptions.Item label="加密后大小" span={1}>{encryptionResult.encrypted_size} 字节</Descriptions.Item>
                  <Descriptions.Item label="加密比率" span={1}>
                    {(encryptionResult.ratio * 100).toFixed(2)}%
                  </Descriptions.Item>
                  <Descriptions.Item label="耗时" span={3}>{encryptionResult.duration}</Descriptions.Item>
                </Descriptions>
                {encryptionResult.success && (
                  <div style={{ marginTop: 16 }}>
                    <Title level={5}>加密数据</Title>
                    <Input.TextArea
                      rows={4}
                      value={encryptionResult.encrypted_data}
                      readOnly
                    />
                    <Title level={5} style={{ marginTop: 16 }}>解密数据</Title>
                    <Input.TextArea
                      rows={4}
                      value={encryptionResult.decrypted_data}
                      readOnly
                    />
                  </div>
                )}
              </div>
            )}
          </Card>
        </TabPane>

        <TabPane tab={<span><CompressOutlined /> 压缩测试</span>} key="compression">
          <Card>
            <Title level={4}>压缩测试</Title>
            <Paragraph>
              测试通信框架的压缩功能，验证数据压缩和解压缩的正确性和性能。
            </Paragraph>
            <Form
              form={compressionForm}
              layout="vertical"
              onFinish={handleTestCompression}
              initialValues={{
                data: '这是一段测试数据，用于测试通信框架的压缩功能。这段数据会被重复多次以便测试压缩效果。'.repeat(20),
                compression_level: 6,
              }}
            >
              <Form.Item
                name="data"
                label="测试数据"
                rules={[{ required: true, message: '请输入测试数据' }]}
              >
                <Input.TextArea rows={4} placeholder="输入要压缩的测试数据" />
              </Form.Item>
              <Form.Item
                name="compression_level"
                label="压缩级别 (1-9)"
                rules={[{ required: true, message: '请输入压缩级别' }]}
              >
                <InputNumber min={1} max={9} style={{ width: '100%' }} />
              </Form.Item>
              <Form.Item>
                <Button type="primary" htmlType="submit" loading={compressionLoading} icon={<CompressOutlined />}>
                  测试压缩
                </Button>
              </Form.Item>
            </Form>

            {compressionResult && (
              <div style={{ marginTop: 16 }}>
                <Divider>测试结果</Divider>
                <Descriptions bordered>
                  <Descriptions.Item label="压缩状态" span={3}>
                    {compressionResult.success ? (
                      <span style={{ color: 'green' }}>压缩成功</span>
                    ) : (
                      <span style={{ color: 'red' }}>压缩失败: {compressionResult.message}</span>
                    )}
                  </Descriptions.Item>
                  <Descriptions.Item label="原始大小" span={1}>{compressionResult.original_size} 字节</Descriptions.Item>
                  <Descriptions.Item label="压缩后大小" span={1}>{compressionResult.compressed_size} 字节</Descriptions.Item>
                  <Descriptions.Item label="压缩比率" span={1}>
                    {(compressionResult.ratio * 100).toFixed(2)}%
                  </Descriptions.Item>
                  <Descriptions.Item label="耗时" span={3}>{compressionResult.duration}</Descriptions.Item>
                </Descriptions>
                {compressionResult.success && (
                  <div style={{ marginTop: 16 }}>
                    <Title level={5}>压缩数据 (Base64编码)</Title>
                    <Input.TextArea
                      rows={4}
                      value={compressionResult.compressed_data}
                      readOnly
                    />
                    <Title level={5} style={{ marginTop: 16 }}>解压缩数据</Title>
                    <Input.TextArea
                      rows={4}
                      value={compressionResult.decompressed_data}
                      readOnly
                    />
                  </div>
                )}
              </div>
            )}
          </Card>
        </TabPane>

        <TabPane tab={<span><BarChartOutlined /> 性能测试</span>} key="performance">
          <Card>
            <Title level={4}>性能测试</Title>
            <Paragraph>
              测试通信框架的性能，包括吞吐量、延迟等指标。
            </Paragraph>
            <Form
              form={performanceForm}
              layout="vertical"
              onFinish={handleTestPerformance}
              initialValues={{
                message_count: 100,
                message_size: 1024,
                enable_encryption: false,
                encryption_key: 'test-key-12345',
                enable_compression: false,
                compression_level: 6,
              }}
            >
              <Form.Item
                name="message_count"
                label="消息数量"
                rules={[{ required: true, message: '请输入消息数量' }]}
              >
                <InputNumber min={1} max={10000} style={{ width: '100%' }} />
              </Form.Item>
              <Form.Item
                name="message_size"
                label="消息大小 (字节)"
                rules={[{ required: true, message: '请输入消息大小' }]}
              >
                <InputNumber min={1} max={1024 * 1024} style={{ width: '100%' }} />
              </Form.Item>
              <Form.Item
                name="enable_encryption"
                label="启用加密"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
              <Form.Item
                name="encryption_key"
                label="加密密钥"
                dependencies={['enable_encryption']}
                rules={[
                  ({ getFieldValue }) => ({
                    validator(_, value) {
                      const enableEncryption = getFieldValue('enable_encryption');
                      if (enableEncryption && (!value || value.length === 0)) {
                        return Promise.reject('请输入加密密钥');
                      }
                      return Promise.resolve();
                    },
                  }),
                ]}
              >
                <Input placeholder="输入加密密钥" />
              </Form.Item>
              <Form.Item
                name="enable_compression"
                label="启用压缩"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
              <Form.Item
                name="compression_level"
                label="压缩级别 (1-9)"
                dependencies={['enable_compression']}
                rules={[
                  ({ getFieldValue }) => ({
                    validator(_, value) {
                      const enableCompression = getFieldValue('enable_compression');
                      if (enableCompression && (!value || value < 1 || value > 9)) {
                        return Promise.reject('请输入1-9之间的压缩级别');
                      }
                      return Promise.resolve();
                    },
                  }),
                ]}
              >
                <InputNumber min={1} max={9} style={{ width: '100%' }} />
              </Form.Item>
              <Form.Item>
                <Button type="primary" htmlType="submit" loading={performanceLoading} icon={<BarChartOutlined />}>
                  开始性能测试
                </Button>
              </Form.Item>
            </Form>

            {performanceResult && performanceResult.success && performanceResult.result && (
              <div style={{ marginTop: 16 }}>
                <Divider>测试结果</Divider>
                <Descriptions bordered>
                  <Descriptions.Item label="测试状态" span={3}>
                    <span style={{ color: 'green' }}>测试成功</span>
                  </Descriptions.Item>
                  <Descriptions.Item label="消息数量" span={1}>{performanceResult.result.message_count}</Descriptions.Item>
                  <Descriptions.Item label="消息大小" span={1}>{performanceResult.result.message_size} 字节</Descriptions.Item>
                  <Descriptions.Item label="总耗时" span={1}>{performanceResult.result.total_duration}</Descriptions.Item>
                  <Descriptions.Item label="发送耗时" span={1}>{performanceResult.result.send_duration}</Descriptions.Item>
                  <Descriptions.Item label="发送吞吐量" span={2}>{performanceResult.result.send_throughput.toFixed(2)} 消息/秒</Descriptions.Item>
                  <Descriptions.Item label="总发送大小" span={3}>{performanceResult.result.send_size} 字节</Descriptions.Item>

                  {performanceResult.result.enable_compression && (
                    <>
                      <Descriptions.Item label="压缩后大小" span={1}>{performanceResult.result.send_compressed_size} 字节</Descriptions.Item>
                      <Descriptions.Item label="压缩比率" span={2}>
                        {(performanceResult.result.send_compression_ratio * 100).toFixed(2)}%
                      </Descriptions.Item>
                    </>
                  )}

                  {performanceResult.result.enable_encryption && (
                    <>
                      <Descriptions.Item label="加密后大小" span={1}>{performanceResult.result.send_encrypted_size} 字节</Descriptions.Item>
                      <Descriptions.Item label="加密比率" span={2}>
                        {(performanceResult.result.send_encryption_ratio * 100).toFixed(2)}%
                      </Descriptions.Item>
                    </>
                  )}
                </Descriptions>
              </div>
            )}
          </Card>
        </TabPane>

        <TabPane tab={<span><ReloadOutlined /> 测试历史</span>} key="history">
          <Card>
            <div style={{ marginBottom: 16 }}>
              <Button
                type="primary"
                icon={<ReloadOutlined />}
                onClick={loadTestHistory}
                loading={historyLoading}
              >
                刷新历史记录
              </Button>
            </div>
            <Table
              columns={historyColumns}
              dataSource={testHistory}
              rowKey={(record, index) => `${record.timestamp || ''}-${index}`}
              loading={historyLoading}
              pagination={{ pageSize: 10 }}
              locale={{ emptyText: '暂无测试历史记录' }}
            />
          </Card>
        </TabPane>
      </Tabs>
    </PageContainer>
  );
};

export default CommTest;
