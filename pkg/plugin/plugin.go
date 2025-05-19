package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	pb "github.com/lomehong/kennel/pkg/plugin/proto/gen"
)

// Module 定义了插件模块的接口
type Module interface {
	// Init 初始化模块
	Init(config map[string]interface{}) error

	// Execute 执行模块操作
	Execute(action string, params map[string]interface{}) (map[string]interface{}, error)

	// Shutdown 关闭模块
	Shutdown() error

	// GetInfo 获取模块信息
	GetInfo() ModuleInfo

	// HandleMessage 处理消息
	HandleMessage(messageType string, messageID string, timestamp int64, payload map[string]interface{}) (map[string]interface{}, error)
}

// ModuleInfo 包含模块的基本信息
type ModuleInfo struct {
	Name             string   `json:"name"`
	Version          string   `json:"version"`
	Description      string   `json:"description"`
	SupportedActions []string `json:"supported_actions"`
}

// ModulePlugin 是一个go-plugin的实现
type ModulePlugin struct {
	plugin.Plugin
	Impl Module
}

// GRPCServer 实现了go-plugin的GRPCServer接口
func (p *ModulePlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	// 注册gRPC服务
	pb.RegisterModuleServer(s, &GRPCServer{Impl: p.Impl})
	return nil
}

// GRPCClient 实现了go-plugin的GRPCClient接口
func (p *ModulePlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	// 创建gRPC客户端
	client := pb.NewModuleClient(c)
	return &GRPCClient{client: client}, nil
}

// PluginMap 是插件类型到插件实现的映射
var PluginMap = map[string]plugin.Plugin{
	"module": &ModulePlugin{},
	// 添加其他可能的插件类型
	"assets":  &ModulePlugin{},
	"audit":   &ModulePlugin{},
	"control": &ModulePlugin{},
	"device":  &ModulePlugin{},
	"dlp":     &ModulePlugin{},
}

// ConfigToJSON 将配置转换为JSON字符串
func ConfigToJSON(config map[string]interface{}) (string, error) {
	// 使用预分配内存的缓冲区，避免多次内存分配
	buffer := &bytes.Buffer{}
	buffer.Grow(1024) // 预分配1KB的空间，可以根据实际情况调整

	encoder := json.NewEncoder(buffer)
	if err := encoder.Encode(config); err != nil {
		return "", err
	}

	// 去除json.Encoder添加的换行符
	result := buffer.String()
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}

	return result, nil
}

// JSONToConfig 将JSON字符串转换为配置
func JSONToConfig(jsonStr string) (map[string]interface{}, error) {
	// 使用sync.Pool复用解码器，减少GC压力
	// 注意：json.Decoder没有Reset方法，所以我们每次都创建一个新的解码器
	decoder := json.NewDecoder(strings.NewReader(jsonStr))

	var config map[string]interface{}
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return config, nil
}
