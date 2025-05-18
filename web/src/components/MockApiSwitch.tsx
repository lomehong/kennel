import React, { useEffect, useState } from 'react';
import { Switch, Tooltip, message } from 'antd';
import { ApiOutlined } from '@ant-design/icons';

/**
 * 模拟API切换组件
 * 仅在开发环境中显示，用于切换真实API和模拟API
 */
const MockApiSwitch: React.FC = () => {
  const [useMockApi, setUseMockApi] = useState<boolean>(
    localStorage.getItem('use_mock_api') === 'true'
  );

  // 仅在开发环境中显示
  if (import.meta.env.PROD) {
    return null;
  }

  const handleChange = (checked: boolean) => {
    localStorage.setItem('use_mock_api', checked ? 'true' : 'false');
    setUseMockApi(checked);
    
    if (checked) {
      message.success('已切换到模拟API，刷新页面后生效');
    } else {
      message.success('已切换到真实API，刷新页面后生效');
    }
    
    // 提示用户刷新页面
    setTimeout(() => {
      if (window.confirm('需要刷新页面以应用更改，是否立即刷新？')) {
        window.location.reload();
      }
    }, 500);
  };

  return (
    <div style={{ position: 'fixed', bottom: '20px', right: '20px', zIndex: 1000 }}>
      <Tooltip title={useMockApi ? '当前使用模拟API' : '当前使用真实API'}>
        <div style={{ 
          display: 'flex', 
          alignItems: 'center', 
          backgroundColor: '#f0f0f0', 
          padding: '8px', 
          borderRadius: '4px',
          boxShadow: '0 2px 8px rgba(0,0,0,0.15)'
        }}>
          <ApiOutlined style={{ marginRight: '8px' }} />
          <span style={{ marginRight: '8px' }}>模拟API</span>
          <Switch checked={useMockApi} onChange={handleChange} />
        </div>
      </Tooltip>
    </div>
  );
};

export default MockApiSwitch;
