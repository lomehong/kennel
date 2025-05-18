import React, { useState } from 'react';
import { BrowserRouter as Router, Routes, Route, Link, Navigate } from 'react-router-dom';
import { Layout, Menu, theme } from 'antd';
import {
  DashboardOutlined,
  AppstoreOutlined,
  LineChartOutlined,
  SettingOutlined,
  MonitorOutlined,
  ApiOutlined,
  BugOutlined,
} from '@ant-design/icons';
import Dashboard from './pages/Dashboard';
import PluginManager from './pages/PluginManager';
import PluginDetail from './pages/PluginDetail';
import MetricsMonitor from './pages/MetricsMonitor';
import SystemMonitor from './pages/SystemMonitor';
import ConfigManager from './pages/ConfigManager';
import CommManager from './pages/CommManager';
import CommTest from './pages/CommTest';
import MockApiSwitch from './components/MockApiSwitch';

const { Header, Content, Footer, Sider } = Layout;

const App: React.FC = () => {
  const [collapsed, setCollapsed] = useState(false);
  const {
    token: { colorBgContainer },
  } = theme.useToken();

  return (
    <Router>
      <Layout className="app-container">
        <Sider
          collapsible
          collapsed={collapsed}
          onCollapse={(value) => setCollapsed(value)}
        >
          <div className="logo">
            {collapsed ? 'AF' : '应用框架'}
          </div>
          <Menu
            theme="dark"
            defaultSelectedKeys={['1']}
            mode="inline"
            items={[
              {
                key: '1',
                icon: <DashboardOutlined />,
                label: <Link to="/">仪表盘</Link>,
              },
              {
                key: '2',
                icon: <AppstoreOutlined />,
                label: <Link to="/plugins">插件管理</Link>,
              },
              {
                key: '3',
                icon: <LineChartOutlined />,
                label: <Link to="/metrics">指标监控</Link>,
              },
              {
                key: '4',
                icon: <MonitorOutlined />,
                label: <Link to="/system">系统监控</Link>,
              },
              {
                key: '5',
                icon: <ApiOutlined />,
                label: <Link to="/comm">通讯管理</Link>,
              },
              {
                key: '6',
                icon: <BugOutlined />,
                label: <Link to="/comm/test">通讯测试</Link>,
              },
              {
                key: '7',
                icon: <SettingOutlined />,
                label: <Link to="/config">配置管理</Link>,
              },
            ]}
          />
        </Sider>
        <Layout>
          <Header style={{ padding: 0, background: colorBgContainer }} />
          <Content style={{ margin: '0 16px' }}>
            <div style={{ padding: 24, minHeight: 360, background: colorBgContainer }}>
              <Routes>
                <Route path="/" element={<Dashboard />} />
                <Route path="/plugins" element={<PluginManager />} />
                <Route path="/plugins/:id" element={<PluginDetail />} />
                <Route path="/metrics" element={<MetricsMonitor />} />
                <Route path="/system" element={<SystemMonitor />} />
                <Route path="/comm" element={<CommManager />} />
                <Route path="/comm/test" element={<CommTest />} />
                <Route path="/config" element={<ConfigManager />} />
                <Route path="*" element={<Navigate to="/" replace />} />
              </Routes>
            </div>
          </Content>
          <Footer style={{ textAlign: 'center' }}>
            应用框架 Web控制台 ©{new Date().getFullYear()} 由 Ant Design 提供支持
          </Footer>
        </Layout>
      </Layout>

      {/* 模拟API切换组件 */}
      <MockApiSwitch />
    </Router>
  );
};

export default App;
