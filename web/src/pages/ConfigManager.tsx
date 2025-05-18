import React, { useEffect, useState } from 'react';
import { Card, Typography, Divider, Button, Form, Input, message, Modal, Space } from 'antd';
import { SaveOutlined, ReloadOutlined, UndoOutlined } from '@ant-design/icons';
import { configApi } from '../services/api';

const { Title } = Typography;
const { TextArea } = Input;

const ConfigManager: React.FC = () => {
  const [config, setConfig] = useState<any>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [saving, setSaving] = useState<boolean>(false);
  const [form] = Form.useForm();
  const [resetModalVisible, setResetModalVisible] = useState<boolean>(false);

  const fetchConfig = async () => {
    setLoading(true);
    try {
      const response = await configApi.getConfig();
      setConfig(response.data);
      form.setFieldsValue({
        config: JSON.stringify(response.data, null, 2)
      });
    } catch (error) {
      console.error('获取配置失败:', error);
      message.error('获取配置失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchConfig();
  }, []);

  const handleSaveConfig = async (values: any) => {
    setSaving(true);
    try {
      // 解析JSON
      let configObj;
      try {
        configObj = JSON.parse(values.config);
      } catch (error) {
        message.error('配置格式无效，请检查JSON格式');
        setSaving(false);
        return;
      }
      
      // 保存配置
      await configApi.updateConfig(configObj);
      message.success('配置保存成功');
      
      // 重新获取配置
      fetchConfig();
    } catch (error) {
      console.error('保存配置失败:', error);
      message.error('保存配置失败');
    } finally {
      setSaving(false);
    }
  };

  const handleResetConfig = async () => {
    try {
      await configApi.resetConfig();
      message.success('配置已重置');
      setResetModalVisible(false);
      fetchConfig();
    } catch (error) {
      console.error('重置配置失败:', error);
      message.error('重置配置失败');
    }
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={2}>配置管理</Title>
        <Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={fetchConfig}
            loading={loading}
          >
            刷新
          </Button>
          <Button
            danger
            icon={<UndoOutlined />}
            onClick={() => setResetModalVisible(true)}
          >
            重置
          </Button>
        </Space>
      </div>
      <Divider />
      
      <Card loading={loading}>
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSaveConfig}
        >
          <Form.Item
            name="config"
            label="配置 (JSON格式)"
            rules={[
              { required: true, message: '请输入配置' },
              {
                validator: (_, value) => {
                  try {
                    JSON.parse(value);
                    return Promise.resolve();
                  } catch (error) {
                    return Promise.reject('无效的JSON格式');
                  }
                }
              }
            ]}
          >
            <TextArea
              rows={20}
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>
          
          <Form.Item>
            <Button
              type="primary"
              htmlType="submit"
              icon={<SaveOutlined />}
              loading={saving}
            >
              保存配置
            </Button>
          </Form.Item>
        </Form>
      </Card>
      
      <Modal
        title="重置配置"
        open={resetModalVisible}
        onOk={handleResetConfig}
        onCancel={() => setResetModalVisible(false)}
        okText="确认重置"
        cancelText="取消"
        okButtonProps={{ danger: true }}
      >
        <p>确定要将配置重置为默认值吗？此操作不可撤销。</p>
      </Modal>
    </div>
  );
};

export default ConfigManager;
