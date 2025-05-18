package plugin

import (
	"context"
	"testing"
	"time"
)

// TestModuleInterface 测试模块接口
func TestModuleInterface(t *testing.T) {
	// 创建测试模块
	module := &testModule{
		id:          "test-module",
		name:        "测试模块",
		version:     "1.0.0",
		description: "用于测试的模块",
	}

	// 测试初始化
	ctx := context.Background()
	config := &ModuleConfig{
		ID:      "test-module",
		Name:    "测试模块",
		Version: "1.0.0",
		Settings: map[string]interface{}{
			"test_key": "test_value",
		},
	}

	if err := module.Init(ctx, config); err != nil {
		t.Fatalf("初始化模块失败: %v", err)
	}

	// 测试启动
	if err := module.Start(); err != nil {
		t.Fatalf("启动模块失败: %v", err)
	}

	// 测试获取信息
	info := module.GetInfo()
	if info.ID != "test-module" {
		t.Errorf("模块ID不匹配: 期望 %s, 实际 %s", "test-module", info.ID)
	}
	if info.Name != "测试模块" {
		t.Errorf("模块名称不匹配: 期望 %s, 实际 %s", "测试模块", info.Name)
	}
	if info.Version != "1.0.0" {
		t.Errorf("模块版本不匹配: 期望 %s, 实际 %s", "1.0.0", info.Version)
	}

	// 测试处理请求
	req := &Request{
		ID:     "test-request",
		Action: "test",
		Params: map[string]interface{}{
			"param1": "value1",
		},
	}

	resp, err := module.HandleRequest(ctx, req)
	if err != nil {
		t.Fatalf("处理请求失败: %v", err)
	}
	if !resp.Success {
		t.Errorf("请求处理失败: %v", resp.Error)
	}
	if resp.Data["echo"] != "value1" {
		t.Errorf("响应数据不匹配: 期望 %s, 实际 %s", "value1", resp.Data["echo"])
	}

	// 测试处理事件
	event := &Event{
		ID:        "test-event",
		Type:      "test",
		Source:    "test-source",
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
		Data: map[string]interface{}{
			"event_data": "event_value",
		},
	}

	if err := module.HandleEvent(ctx, event); err != nil {
		t.Fatalf("处理事件失败: %v", err)
	}

	// 测试停止
	if err := module.Stop(); err != nil {
		t.Fatalf("停止模块失败: %v", err)
	}
}

// testModule 测试模块
type testModule struct {
	id          string
	name        string
	version     string
	description string
	config      map[string]interface{}
	started     bool
}

// Init 初始化模块
func (m *testModule) Init(ctx context.Context, config *ModuleConfig) error {
	m.config = config.Settings
	return nil
}

// Start 启动模块
func (m *testModule) Start() error {
	m.started = true
	return nil
}

// Stop 停止模块
func (m *testModule) Stop() error {
	m.started = false
	return nil
}

// GetInfo 获取模块信息
func (m *testModule) GetInfo() ModuleInfo {
	return ModuleInfo{
		ID:                m.id,
		Name:              m.name,
		Version:           m.version,
		Description:       m.description,
		Author:            "Test Author",
		License:           "MIT",
		Capabilities:      []string{"test"},
		SupportedPlatforms: []string{"windows", "linux", "darwin"},
		Language:          "go",
	}
}

// HandleRequest 处理请求
func (m *testModule) HandleRequest(ctx context.Context, req *Request) (*Response, error) {
	if req.Action == "test" {
		return &Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"echo": req.Params["param1"],
			},
		}, nil
	}

	return &Response{
		ID:      req.ID,
		Success: false,
		Error: &ErrorInfo{
			Code:    "unknown_action",
			Message: "未知操作",
		},
	}, nil
}

// HandleEvent 处理事件
func (m *testModule) HandleEvent(ctx context.Context, event *Event) error {
	// 简单记录事件
	return nil
}

// TestHealthCheck 测试健康检查接口
func TestHealthCheck(t *testing.T) {
	// 创建测试模块
	module := &testHealthCheckModule{
		testModule: testModule{
			id:      "test-health",
			name:    "测试健康检查",
			version: "1.0.0",
		},
	}

	// 测试健康检查
	healthCheck, ok := module.(HealthCheck)
	if !ok {
		t.Fatal("模块未实现健康检查接口")
	}

	status := healthCheck.CheckHealth()
	if status.Status != "healthy" {
		t.Errorf("健康状态不匹配: 期望 %s, 实际 %s", "healthy", status.Status)
	}
}

// testHealthCheckModule 测试健康检查模块
type testHealthCheckModule struct {
	testModule
}

// CheckHealth 检查健康状态
func (m *testHealthCheckModule) CheckHealth() HealthStatus {
	return HealthStatus{
		Status: "healthy",
		Details: map[string]interface{}{
			"uptime": 3600,
		},
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
	}
}

// TestResourceManager 测试资源管理接口
func TestResourceManager(t *testing.T) {
	// 创建测试模块
	module := &testResourceModule{
		testModule: testModule{
			id:      "test-resource",
			name:    "测试资源管理",
			version: "1.0.0",
		},
		usage: ResourceUsage{
			CPU:     10.5,
			Memory:  1024 * 1024 * 100, // 100MB
			Disk:    1024 * 1024 * 500, // 500MB
			Network: 1024 * 10,         // 10KB/s
		},
	}

	// 测试资源管理
	resourceManager, ok := module.(ResourceManager)
	if !ok {
		t.Fatal("模块未实现资源管理接口")
	}

	// 测试获取资源使用情况
	usage := resourceManager.GetResourceUsage()
	if usage.CPU != 10.5 {
		t.Errorf("CPU使用率不匹配: 期望 %f, 实际 %f", 10.5, usage.CPU)
	}
	if usage.Memory != 1024*1024*100 {
		t.Errorf("内存使用量不匹配: 期望 %d, 实际 %d", 1024*1024*100, usage.Memory)
	}

	// 测试设置资源限制
	limits := ResourceLimits{
		MaxCPU:     20.0,
		MaxMemory:  1024 * 1024 * 200, // 200MB
		MaxDisk:    1024 * 1024 * 1000, // 1GB
		MaxNetwork: 1024 * 100,         // 100KB/s
	}

	if err := resourceManager.SetResourceLimits(limits); err != nil {
		t.Fatalf("设置资源限制失败: %v", err)
	}

	// 验证限制是否生效
	if module.limits.MaxCPU != 20.0 {
		t.Errorf("CPU限制不匹配: 期望 %f, 实际 %f", 20.0, module.limits.MaxCPU)
	}
	if module.limits.MaxMemory != 1024*1024*200 {
		t.Errorf("内存限制不匹配: 期望 %d, 实际 %d", 1024*1024*200, module.limits.MaxMemory)
	}
}

// testResourceModule 测试资源管理模块
type testResourceModule struct {
	testModule
	usage  ResourceUsage
	limits ResourceLimits
}

// GetResourceUsage 获取资源使用情况
func (m *testResourceModule) GetResourceUsage() ResourceUsage {
	return m.usage
}

// SetResourceLimits 设置资源限制
func (m *testResourceModule) SetResourceLimits(limits ResourceLimits) error {
	m.limits = limits
	return nil
}
