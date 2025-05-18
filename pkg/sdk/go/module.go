package sdk

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lomehong/kennel/pkg/core/plugin"
)

// BaseModule 基础模块实现
type BaseModule struct {
	ID          string
	Name        string
	Version     string
	Description string
	Author      string
	License     string
	Logger      Logger
	Config      map[string]interface{}
	StartTime   time.Time
}

// NewBaseModule 创建基础模块
func NewBaseModule(id, name, version, description string) *BaseModule {
	return &BaseModule{
		ID:          id,
		Name:        name,
		Version:     version,
		Description: description,
		Logger:      NewLogger(id, LogLevelInfo),
		Config:      make(map[string]interface{}),
		StartTime:   time.Now(),
	}
}

// Init 初始化模块
func (m *BaseModule) Init(ctx context.Context, config *plugin.ModuleConfig) error {
	m.Logger.Info("初始化模块", "id", m.ID)
	m.Config = config.Settings
	return nil
}

// Start 启动模块
func (m *BaseModule) Start() error {
	m.Logger.Info("启动模块", "id", m.ID)
	m.StartTime = time.Now()
	return nil
}

// Stop 停止模块
func (m *BaseModule) Stop() error {
	m.Logger.Info("停止模块", "id", m.ID)
	uptime := time.Since(m.StartTime)
	m.Logger.Info("运行时间", "uptime", uptime.String())
	return nil
}

// GetInfo 获取模块信息
func (m *BaseModule) GetInfo() plugin.ModuleInfo {
	return plugin.ModuleInfo{
		ID:                 m.ID,
		Name:               m.Name,
		Version:            m.Version,
		Description:        m.Description,
		Author:             m.Author,
		License:            m.License,
		Capabilities:       []string{},
		SupportedPlatforms: []string{"windows", "linux", "darwin"},
		Language:           "go",
	}
}

// HandleRequest 处理请求
func (m *BaseModule) HandleRequest(ctx context.Context, req *plugin.Request) (*plugin.Response, error) {
	m.Logger.Info("处理请求", "action", req.Action)
	return &plugin.Response{
		ID:      req.ID,
		Success: false,
		Error: &plugin.ErrorInfo{
			Code:    "not_implemented",
			Message: fmt.Sprintf("未实现的操作: %s", req.Action),
		},
	}, nil
}

// HandleEvent 处理事件
func (m *BaseModule) HandleEvent(ctx context.Context, event *plugin.Event) error {
	m.Logger.Info("处理事件", "type", event.Type, "source", event.Source)
	return nil
}

// CheckHealth 检查健康状态
func (m *BaseModule) CheckHealth() plugin.HealthStatus {
	return plugin.HealthStatus{
		Status: "healthy",
		Details: map[string]interface{}{
			"uptime": time.Since(m.StartTime).String(),
		},
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
	}
}

// RunModule 运行模块
func RunModule(module plugin.Module) {
	// 设置信号处理
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	// 启动模块
	if err := module.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "启动模块失败: %v\n", err)
		os.Exit(1)
	}

	// 等待信号
	<-signalCh

	// 停止模块
	if err := module.Stop(); err != nil {
		fmt.Fprintf(os.Stderr, "停止模块失败: %v\n", err)
		os.Exit(1)
	}
}

// GetConfigString 获取配置字符串
func GetConfigString(config map[string]interface{}, key, defaultValue string) string {
	if value, ok := config[key]; ok {
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}
	return defaultValue
}

// GetConfigInt 获取配置整数
func GetConfigInt(config map[string]interface{}, key string, defaultValue int) int {
	if value, ok := config[key]; ok {
		switch v := value.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return defaultValue
}

// GetConfigBool 获取配置布尔值
func GetConfigBool(config map[string]interface{}, key string, defaultValue bool) bool {
	if value, ok := config[key]; ok {
		if boolValue, ok := value.(bool); ok {
			return boolValue
		}
	}
	return defaultValue
}

// GetConfigMap 获取配置映射
func GetConfigMap(config map[string]interface{}, key string) map[string]interface{} {
	if value, ok := config[key]; ok {
		if mapValue, ok := value.(map[string]interface{}); ok {
			return mapValue
		}
	}
	return make(map[string]interface{})
}

// GetConfigSlice 获取配置切片
func GetConfigSlice(config map[string]interface{}, key string) []interface{} {
	if value, ok := config[key]; ok {
		if sliceValue, ok := value.([]interface{}); ok {
			return sliceValue
		}
	}
	return make([]interface{}, 0)
}

// GetConfigStringSlice 获取配置字符串切片
func GetConfigStringSlice(config map[string]interface{}, key string) []string {
	slice := GetConfigSlice(config, key)
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if strItem, ok := item.(string); ok {
			result = append(result, strItem)
		}
	}
	return result
}
