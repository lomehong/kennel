package interfaces

import (
	"context"
	"time"
)

// WebConsoleInterface 定义Web控制台接口
type WebConsoleInterface interface {
	// Init 初始化Web控制台
	Init() error

	// Start 启动Web控制台
	Start() error

	// Stop 停止Web控制台
	Stop(ctx context.Context) error
}

// WebConsoleConfig 定义Web控制台配置接口
type WebConsoleConfig interface {
	// GetEnabled 获取是否启用
	GetEnabled() bool

	// GetHost 获取主机地址
	GetHost() string

	// GetPort 获取端口
	GetPort() int

	// GetEnableHTTPS 获取是否启用HTTPS
	GetEnableHTTPS() bool

	// GetCertFile 获取证书文件
	GetCertFile() string

	// GetKeyFile 获取密钥文件
	GetKeyFile() string

	// GetEnableAuth 获取是否启用认证
	GetEnableAuth() bool

	// GetUsername 获取用户名
	GetUsername() string

	// GetPassword 获取密码
	GetPassword() string

	// GetSessionTimeout 获取会话超时时间
	GetSessionTimeout() time.Duration

	// GetStaticDir 获取静态文件目录
	GetStaticDir() string

	// GetLogLevel 获取日志级别
	GetLogLevel() string

	// GetRateLimit 获取请求限制
	GetRateLimit() int

	// GetEnableCSRF 获取是否启用CSRF保护
	GetEnableCSRF() bool

	// GetAPIPrefix 获取API前缀
	GetAPIPrefix() string

	// GetAllowOrigins 获取跨域配置
	GetAllowOrigins() []string
}

// WebConsoleFactory 定义Web控制台工厂接口
type WebConsoleFactory interface {
	// CreateWebConsole 创建Web控制台
	CreateWebConsole(config WebConsoleConfig, app AppInterface) (WebConsoleInterface, error)
}
