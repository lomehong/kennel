package webconsole

import (
	"time"

	"github.com/lomehong/kennel/pkg/interfaces"
)

// Factory 实现WebConsoleFactory接口
type Factory struct{}

// NewFactory 创建一个新的Web控制台工厂
func NewFactory() *Factory {
	return &Factory{}
}

// CreateWebConsole 创建Web控制台
func (f *Factory) CreateWebConsole(config interfaces.WebConsoleConfig, app interfaces.AppInterface) (interfaces.WebConsoleInterface, error) {
	// 转换配置
	consoleConfig := Config{
		Enabled:        config.GetEnabled(),
		Host:           config.GetHost(),
		Port:           config.GetPort(),
		EnableHTTPS:    config.GetEnableHTTPS(),
		CertFile:       config.GetCertFile(),
		KeyFile:        config.GetKeyFile(),
		EnableAuth:     config.GetEnableAuth(),
		Username:       config.GetUsername(),
		Password:       config.GetPassword(),
		SessionTimeout: config.GetSessionTimeout(),
		StaticDir:      config.GetStaticDir(),
		LogLevel:       config.GetLogLevel(),
		RateLimit:      config.GetRateLimit(),
		EnableCSRF:     config.GetEnableCSRF(),
		APIPrefix:      config.GetAPIPrefix(),
		AllowOrigins:   config.GetAllowOrigins(),
	}

	// 创建Web控制台
	return NewConsole(consoleConfig, app)
}

// ConfigAdapter 适配器，将接口转换为具体类型
type ConfigAdapter struct {
	Enabled        bool
	Host           string
	Port           int
	EnableHTTPS    bool
	CertFile       string
	KeyFile        string
	EnableAuth     bool
	Username       string
	Password       string
	SessionTimeout time.Duration
	StaticDir      string
	LogLevel       string
	RateLimit      int
	EnableCSRF     bool
	APIPrefix      string
	AllowOrigins   []string
}

// GetEnabled 获取是否启用
func (c *ConfigAdapter) GetEnabled() bool {
	return c.Enabled
}

// GetHost 获取主机地址
func (c *ConfigAdapter) GetHost() string {
	return c.Host
}

// GetPort 获取端口
func (c *ConfigAdapter) GetPort() int {
	return c.Port
}

// GetEnableHTTPS 获取是否启用HTTPS
func (c *ConfigAdapter) GetEnableHTTPS() bool {
	return c.EnableHTTPS
}

// GetCertFile 获取证书文件
func (c *ConfigAdapter) GetCertFile() string {
	return c.CertFile
}

// GetKeyFile 获取密钥文件
func (c *ConfigAdapter) GetKeyFile() string {
	return c.KeyFile
}

// GetEnableAuth 获取是否启用认证
func (c *ConfigAdapter) GetEnableAuth() bool {
	return c.EnableAuth
}

// GetUsername 获取用户名
func (c *ConfigAdapter) GetUsername() string {
	return c.Username
}

// GetPassword 获取密码
func (c *ConfigAdapter) GetPassword() string {
	return c.Password
}

// GetSessionTimeout 获取会话超时时间
func (c *ConfigAdapter) GetSessionTimeout() time.Duration {
	return c.SessionTimeout
}

// GetStaticDir 获取静态文件目录
func (c *ConfigAdapter) GetStaticDir() string {
	return c.StaticDir
}

// GetLogLevel 获取日志级别
func (c *ConfigAdapter) GetLogLevel() string {
	return c.LogLevel
}

// GetRateLimit 获取请求限制
func (c *ConfigAdapter) GetRateLimit() int {
	return c.RateLimit
}

// GetEnableCSRF 获取是否启用CSRF保护
func (c *ConfigAdapter) GetEnableCSRF() bool {
	return c.EnableCSRF
}

// GetAPIPrefix 获取API前缀
func (c *ConfigAdapter) GetAPIPrefix() string {
	return c.APIPrefix
}

// GetAllowOrigins 获取跨域配置
func (c *ConfigAdapter) GetAllowOrigins() []string {
	return c.AllowOrigins
}
