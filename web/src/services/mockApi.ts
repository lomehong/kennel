/**
 * 模拟API服务
 * 用于在后端服务不可用时提供模拟数据
 */

// 模拟插件数据
export const mockPlugins = [
  {
    id: 'assets',
    name: '资产管理',
    version: '1.0.0',
    description: '管理和监控系统资产',
    status: 'running',
    enabled: true,
  },
  {
    id: 'device',
    name: '设备管理',
    version: '1.0.0',
    description: '管理和监控系统设备',
    status: 'running',
    enabled: true,
  },
  {
    id: 'dlp',
    name: '数据防泄漏',
    version: '1.0.0',
    description: '防止敏感数据泄漏',
    status: 'stopped',
    enabled: false,
  },
  {
    id: 'control',
    name: '终端管控',
    version: '1.0.0',
    description: '远程管理和控制终端',
    status: 'running',
    enabled: true,
  },
  {
    id: 'audit',
    name: '安全审计',
    version: '1.0.0',
    description: '记录和审计系统事件',
    status: 'running',
    enabled: true,
  },
];

// 模拟插件详情
export const mockPluginDetails = {
  assets: {
    id: 'assets',
    name: '资产管理',
    version: '1.0.0',
    description: '管理和监控系统资产',
    status: 'running',
    enabled: true,
    supported_actions: ['scan', 'report'],
    config: {
      collect_interval: 3600,
      report_server: 'https://example.com/api/assets',
      auto_report: false,
    },
  },
  device: {
    id: 'device',
    name: '设备管理',
    version: '1.0.0',
    description: '管理和监控系统设备',
    status: 'running',
    enabled: true,
    supported_actions: ['monitor', 'control'],
    config: {
      monitor_usb: true,
      monitor_network: true,
      allow_network_disable: true,
    },
  },
  dlp: {
    id: 'dlp',
    name: '数据防泄漏',
    version: '1.0.0',
    description: '防止敏感数据泄漏',
    status: 'stopped',
    enabled: false,
    supported_actions: ['scan', 'block'],
    config: {
      monitor_clipboard: false,
      monitor_files: false,
      monitored_file_types: ['*.doc', '*.docx', '*.xls', '*.xlsx', '*.pdf', '*.txt'],
    },
  },
  control: {
    id: 'control',
    name: '终端管控',
    version: '1.0.0',
    description: '远程管理和控制终端',
    status: 'running',
    enabled: true,
    supported_actions: ['command', 'install', 'kill'],
    config: {
      allow_remote_command: true,
      allow_software_install: true,
      allow_process_kill: true,
      protected_processes: ['agent.exe', 'system', 'explorer.exe'],
    },
  },
  audit: {
    id: 'audit',
    name: '安全审计',
    version: '1.0.0',
    description: '记录和审计系统事件',
    status: 'running',
    enabled: true,
    supported_actions: ['log', 'alert'],
    config: {
      log_system_events: true,
      log_user_events: true,
      log_network_events: true,
      log_file_events: true,
      log_retention_days: 30,
      enable_alerts: false,
      alert_recipients: ['admin@example.com'],
    },
  },
};

// 模拟系统状态
export const mockSystemStatus = {
  host: {
    hostname: 'localhost',
    platform: 'windows',
    platform_version: '10.0',
    uptime: '12小时30分钟',
  },
  framework: {
    version: '1.0.0',
    start_time: new Date(Date.now() - 12 * 60 * 60 * 1000).toISOString(),
    uptime: '12小时30分钟',
  },
  runtime: {
    go_version: 'go1.20.3',
    go_os: 'windows',
    go_arch: 'amd64',
    cpu_cores: 8,
    goroutines: 24,
  },
  plugins: {
    total: 5,
    enabled: 4,
    disabled: 1,
  },
  comm: {
    connected: true,
    status: 'connected',
    last_connect: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
  },
  timestamp: new Date().toISOString(),
};

// 模拟系统资源
export const mockSystemResources = {
  cpu: {
    cores: 8,
    usage_pct: 35.5,
    temperature: 45.2,
    frequency: 2800,
  },
  memory: {
    total: 8 * 1024 * 1024 * 1024,
    used: 4 * 1024 * 1024 * 1024,
    free: 4 * 1024 * 1024 * 1024,
    used_pct: 50.0,
    cached: 512 * 1024 * 1024,
    buffers: 256 * 1024 * 1024,
  },
  disk: {
    total: 500 * 1024 * 1024 * 1024,
    used: 250 * 1024 * 1024 * 1024,
    free: 250 * 1024 * 1024 * 1024,
    used_pct: 50.0,
    read_rate: 25 * 1024 * 1024,
    write_rate: 15 * 1024 * 1024,
  },
  process: {
    count: 45,
    threads: 120,
    goroutines: 24,
  },
  runtime: {
    go_version: 'go1.20.3',
    go_os: 'windows',
    go_arch: 'amd64',
    cpu_cores: 8,
    goroutines: 24,
  },
  timestamp: new Date().toISOString(),
};

// 模拟指标数据
export const mockMetrics = {
  comm: {
    connected: true,
    current_state: 'connected',
    connect_count: 5,
    reconnect_count: 2,
    sent_message_count: 1250,
    received_message_count: 980,
    sent_bytes: 256 * 1024,
    received_bytes: 128 * 1024,
    avg_latency: 45.2,
    min_latency: 12.5,
    max_latency: 120.8,
    compressed_count: 350,
    compressed_bytes: 128 * 1024,
    compressed_bytes_after: 64 * 1024,
    compression_ratio: 0.5,
    heartbeat_sent_count: 720,
    heartbeat_received_count: 720,
    heartbeat_error_count: 0,
    last_heartbeat_time: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
  },
  system: {
    cpu: {
      usage_pct: 35.5,
      cores: 8,
    },
    memory: {
      used_pct: 50.0,
      used: 4 * 1024 * 1024 * 1024,
      total: 8 * 1024 * 1024 * 1024,
    },
    runtime: {
      goroutines: 24,
      heap_alloc: 64 * 1024 * 1024,
      gc_cycles: 12,
    },
  },
  time: new Date().toISOString(),
};

// 模拟配置数据
export const mockConfig = {
  plugin_dir: 'app',
  log_level: 'debug',
  log_file: 'agent.log',
  web_console: {
    enabled: true,
    host: '0.0.0.0',
    port: 8088,
    enable_https: false,
    cert_file: '',
    key_file: '',
    enable_auth: false,
    username: 'admin',
    password: 'admin',
    static_dir: 'web/dist',
    log_level: 'debug',
    rate_limit: 100,
    enable_csrf: false,
    api_prefix: '/api',
    session_timeout: '24h',
    allow_origins: ['*', 'http://localhost:8088', 'http://127.0.0.1:8088'],
  },
  enable_assets: true,
  enable_device: true,
  enable_dlp: true,
  enable_control: true,
  enable_audit: true,
  enable_comm: true,
};
