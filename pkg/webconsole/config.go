package webconsole

import (
	"fmt"
	"time"
)

// Config 定义Web控制台配置
type Config struct {
	// 是否启用Web控制台
	Enabled bool

	// Web控制台监听地址
	Host string

	// Web控制台监听端口
	Port int

	// 是否启用HTTPS
	EnableHTTPS bool

	// HTTPS证书文件
	CertFile string

	// HTTPS私钥文件
	KeyFile string

	// 是否启用认证
	EnableAuth bool

	// 认证用户名
	Username string

	// 认证密码
	Password string

	// 会话超时时间
	SessionTimeout time.Duration

	// 静态文件目录
	StaticDir string

	// API前缀
	APIPrefix string

	// 跨域配置
	AllowOrigins []string

	// 是否启用CSRF保护
	EnableCSRF bool

	// 请求限制
	RateLimit int

	// 日志级别
	LogLevel string
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		Enabled:        true,
		Host:           "0.0.0.0",
		Port:           8088,
		EnableHTTPS:    false,
		CertFile:       "",
		KeyFile:        "",
		EnableAuth:     true,
		Username:       "admin",
		Password:       "admin",
		SessionTimeout: 24 * time.Hour,
		StaticDir:      "./web/dist",
		APIPrefix:      "/api",
		AllowOrigins:   []string{"*"},
		EnableCSRF:     true,
		RateLimit:      100,
		LogLevel:       "info",
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("无效的端口号: %d", c.Port)
	}

	if c.EnableHTTPS {
		if c.CertFile == "" {
			return fmt.Errorf("启用HTTPS时必须指定证书文件")
		}
		if c.KeyFile == "" {
			return fmt.Errorf("启用HTTPS时必须指定私钥文件")
		}
	}

	if c.EnableAuth {
		if c.Username == "" {
			return fmt.Errorf("启用认证时必须指定用户名")
		}
		if c.Password == "" {
			return fmt.Errorf("启用认证时必须指定密码")
		}
	}

	if c.SessionTimeout <= 0 {
		return fmt.Errorf("会话超时时间必须大于0")
	}

	if c.RateLimit <= 0 {
		return fmt.Errorf("请求限制必须大于0")
	}

	return nil
}

// GetAddress 获取监听地址
func (c *Config) GetAddress() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
