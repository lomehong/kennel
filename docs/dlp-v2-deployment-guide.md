# DLP v2.0 生产环境部署指南

## 部署前准备

### 1. 系统要求检查

#### 硬件要求
```bash
# 检查系统资源
free -h                    # 内存检查（需要4GB+）
df -h                      # 磁盘空间检查（需要10GB+）
lscpu                      # CPU检查
ip link show               # 网络接口检查
```

#### 软件依赖
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install -y build-essential git curl wget
sudo apt install -y tesseract-ocr tesseract-ocr-chi-sim
sudo apt install -y iptables-persistent

# CentOS/RHEL
sudo yum groupinstall -y "Development Tools"
sudo yum install -y git curl wget
sudo yum install -y tesseract tesseract-langpack-chi-sim
sudo yum install -y iptables-services

# macOS
brew install tesseract tesseract-lang
```

### 2. 权限配置

#### Linux权限设置
```bash
# 创建DLP用户
sudo useradd -r -s /bin/false dlp
sudo usermod -aG sudo dlp

# 设置iptables权限
sudo visudo
# 添加：dlp ALL=(ALL) NOPASSWD: /sbin/iptables

# 设置网络接口权限
sudo setcap cap_net_raw,cap_net_admin=eip /path/to/dlp
```

#### Windows权限设置
```powershell
# 以管理员身份运行PowerShell
# 安装WinDivert驱动
.\WinDivert\install.bat

# 设置防火墙权限
netsh advfirewall set allprofiles state on
```

## 编译和安装

### 1. 源码编译

```bash
# 克隆代码
git clone https://github.com/lomehong/kennel.git
cd kennel/app/dlp

# 安装Go依赖
go mod tidy

# 编译DLP
go build -o dlp .

# 验证编译
./dlp --version
```

### 2. 配置文件设置

#### 创建配置目录
```bash
sudo mkdir -p /etc/dlp
sudo mkdir -p /var/log/dlp
sudo mkdir -p /var/lib/dlp
sudo mkdir -p /var/quarantine/dlp
```

#### 基础配置文件
```yaml
# /etc/dlp/config.yaml
server:
  host: "0.0.0.0"
  port: 8080
  log_level: "info"

interceptor:
  enabled: true
  mode: "active"
  interfaces: ["eth0"]
  buffer_size: 65536
  max_packet_size: 1500

parser:
  http:
    enabled: true
    max_body_size: 10485760
    tls_decrypt: false
  
analyzer:
  text:
    enabled: true
    ocr_enabled: false
    ml_enabled: false
    max_content_size: 10485760
    min_confidence: 0.7

executor:
  block:
    enabled: true
    cleanup_interval: "1h"
  
  alert:
    enabled: true
    email:
      smtp_server: "localhost"
      smtp_port: 587
      from: "dlp@localhost"
      recipients: ["admin@localhost"]
  
  encrypt:
    enabled: true
    algorithm: "AES-256"
  
  quarantine:
    enabled: true
    quarantine_dir: "/var/quarantine/dlp"
    max_file_size: 104857600

logging:
  level: "info"
  format: "json"
  output: "/var/log/dlp/dlp.log"
  max_size: 100
  max_backups: 10
  max_age: 30
```

### 3. 系统服务配置

#### Systemd服务文件
```ini
# /etc/systemd/system/dlp.service
[Unit]
Description=DLP Data Loss Prevention Service
After=network.target
Wants=network.target

[Service]
Type=simple
User=dlp
Group=dlp
ExecStart=/usr/local/bin/dlp --config /etc/dlp/config.yaml
ExecReload=/bin/kill -HUP $MAINPID
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=dlp

# 安全设置
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/dlp /var/lib/dlp /var/quarantine/dlp

# 网络权限
AmbientCapabilities=CAP_NET_RAW CAP_NET_ADMIN
CapabilityBoundingSet=CAP_NET_RAW CAP_NET_ADMIN

[Install]
WantedBy=multi-user.target
```

#### 启动服务
```bash
# 安装服务文件
sudo cp dlp /usr/local/bin/
sudo chmod +x /usr/local/bin/dlp
sudo systemctl daemon-reload

# 启动服务
sudo systemctl enable dlp
sudo systemctl start dlp

# 检查状态
sudo systemctl status dlp
sudo journalctl -u dlp -f
```

## 高级配置

### 1. TLS解密配置

```yaml
# config.yaml
parser:
  http:
    tls_decrypt: true
    cert_store: "/etc/dlp/certs"
    certificates:
      - cert_file: "/etc/dlp/certs/example.com.crt"
        key_file: "/etc/dlp/certs/example.com.key"
        domains: ["example.com", "*.example.com"]
```

```bash
# 证书配置
sudo mkdir -p /etc/dlp/certs
sudo cp your-cert.crt /etc/dlp/certs/
sudo cp your-key.key /etc/dlp/certs/
sudo chown -R dlp:dlp /etc/dlp/certs
sudo chmod 600 /etc/dlp/certs/*.key
```

### 2. OCR功能启用

```yaml
# config.yaml
analyzer:
  text:
    ocr_enabled: true
    ocr_config:
      engine: "tesseract"
      languages: ["eng", "chi_sim"]
      confidence_threshold: 0.8
```

```bash
# 安装OCR语言包
sudo apt install tesseract-ocr-eng tesseract-ocr-chi-sim
```

### 3. 邮件告警配置

```yaml
# config.yaml
executor:
  alert:
    email:
      smtp_server: "smtp.gmail.com"
      smtp_port: 587
      username: "your-email@gmail.com"
      password: "your-app-password"
      from: "dlp@yourcompany.com"
      recipients: 
        - "admin@yourcompany.com"
        - "security@yourcompany.com"
      use_tls: true
```

### 4. 数据库集成

```yaml
# config.yaml
database:
  type: "postgresql"
  host: "localhost"
  port: 5432
  database: "dlp"
  username: "dlp_user"
  password: "dlp_password"
  ssl_mode: "require"
```

```sql
-- 创建数据库
CREATE DATABASE dlp;
CREATE USER dlp_user WITH PASSWORD 'dlp_password';
GRANT ALL PRIVILEGES ON DATABASE dlp TO dlp_user;
```

## 监控和维护

### 1. 日志监控

```bash
# 实时日志监控
sudo tail -f /var/log/dlp/dlp.log

# 日志分析
sudo grep "ERROR" /var/log/dlp/dlp.log
sudo grep "敏感信息" /var/log/dlp/dlp.log | wc -l
```

### 2. 性能监控

```bash
# 系统资源监控
top -p $(pgrep dlp)
netstat -i  # 网络接口统计
ss -tuln    # 网络连接状态

# DLP特定监控
curl http://localhost:8080/metrics  # Prometheus指标
curl http://localhost:8080/health   # 健康检查
```

### 3. 维护任务

```bash
# 清理过期隔离文件
find /var/quarantine/dlp -type f -mtime +30 -delete

# 日志轮转
sudo logrotate /etc/logrotate.d/dlp

# 配置重载
sudo systemctl reload dlp
```

## 故障排除

### 1. 常见问题

#### 权限问题
```bash
# 检查权限
sudo getcap /usr/local/bin/dlp
ls -la /etc/dlp/

# 修复权限
sudo setcap cap_net_raw,cap_net_admin=eip /usr/local/bin/dlp
sudo chown -R dlp:dlp /etc/dlp/
```

#### 网络拦截失败
```bash
# 检查网络接口
ip link show
sudo tcpdump -i eth0 -c 10

# 检查iptables规则
sudo iptables -L -n
```

#### OCR功能异常
```bash
# 测试OCR
tesseract --version
tesseract test.png output

# 检查语言包
tesseract --list-langs
```

### 2. 调试模式

```yaml
# config.yaml - 调试配置
server:
  log_level: "debug"

logging:
  level: "debug"
  output: "stdout"
```

```bash
# 前台运行调试
sudo -u dlp /usr/local/bin/dlp --config /etc/dlp/config.yaml --debug
```

## 安全加固

### 1. 网络安全
```bash
# 防火墙配置
sudo ufw allow 8080/tcp
sudo ufw enable

# 限制访问
sudo iptables -A INPUT -p tcp --dport 8080 -s 192.168.1.0/24 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 8080 -j DROP
```

### 2. 文件权限
```bash
# 配置文件权限
sudo chmod 600 /etc/dlp/config.yaml
sudo chown dlp:dlp /etc/dlp/config.yaml

# 日志文件权限
sudo chmod 640 /var/log/dlp/*.log
sudo chown dlp:adm /var/log/dlp/*.log
```

### 3. 数据加密
```yaml
# config.yaml
security:
  encrypt_logs: true
  encrypt_quarantine: true
  key_rotation_interval: "24h"
```

## 集群部署

### 1. 负载均衡配置
```nginx
# nginx.conf
upstream dlp_cluster {
    server 192.168.1.10:8080;
    server 192.168.1.11:8080;
    server 192.168.1.12:8080;
}

server {
    listen 80;
    location / {
        proxy_pass http://dlp_cluster;
    }
}
```

### 2. 共享存储
```yaml
# config.yaml
storage:
  type: "nfs"
  mount_point: "/mnt/dlp-shared"
  quarantine_dir: "/mnt/dlp-shared/quarantine"
```

这个部署指南提供了完整的生产环境部署流程，包括系统准备、安装配置、监控维护和故障排除等各个方面。
