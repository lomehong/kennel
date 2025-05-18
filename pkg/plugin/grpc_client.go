package plugin

import (
	"context"
	"fmt"
	"time"

	pb "github.com/lomehong/kennel/pkg/plugin/proto/gen"
)

// GRPCClient 是一个gRPC客户端适配器，用于将gRPC客户端转换为Module接口
type GRPCClient struct {
	client pb.ModuleClient
}

// Init 实现了Module接口的Init方法
func (c *GRPCClient) Init(config map[string]interface{}) error {
	// 将配置转换为JSON
	configJSON, err := ConfigToJSON(config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 调用gRPC服务
	resp, err := c.client.Init(context.Background(), &pb.InitRequest{
		Config: configJSON,
	})
	if err != nil {
		return fmt.Errorf("gRPC调用失败: %w", err)
	}

	// 检查响应
	if !resp.Success {
		return fmt.Errorf("初始化失败: %s", resp.ErrorMessage)
	}

	return nil
}

// Execute 实现了Module接口的Execute方法
func (c *GRPCClient) Execute(action string, params map[string]interface{}) (map[string]interface{}, error) {
	// 将参数转换为JSON
	paramsJSON, err := ConfigToJSON(params)
	if err != nil {
		return nil, fmt.Errorf("序列化参数失败: %w", err)
	}

	// 添加超时控制，避免执行时间过长
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 调用gRPC服务
	resp, err := c.client.Execute(ctx, &pb.ActionRequest{
		Action: action,
		Params: paramsJSON,
	})
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("执行超时: %s", action)
		}
		return nil, fmt.Errorf("gRPC调用失败: %w", err)
	}

	// 检查响应
	if !resp.Success {
		return nil, fmt.Errorf("执行失败: %s", resp.ErrorMessage)
	}

	// 将JSON结果转换为map
	result, err := JSONToConfig(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("解析结果失败: %w", err)
	}

	return result, nil
}

// Shutdown 实现了Module接口的Shutdown方法
func (c *GRPCClient) Shutdown() error {
	// 调用gRPC服务
	_, err := c.client.Shutdown(context.Background(), &pb.EmptyRequest{})
	if err != nil {
		return fmt.Errorf("gRPC调用失败: %w", err)
	}

	return nil
}

// GetInfo 实现了Module接口的GetInfo方法
func (c *GRPCClient) GetInfo() ModuleInfo {
	// 调用gRPC服务
	resp, err := c.client.GetInfo(context.Background(), &pb.EmptyRequest{})
	if err != nil {
		// 如果出错，返回一个默认的ModuleInfo
		return ModuleInfo{
			Name:             "unknown",
			Version:          "0.0.0",
			Description:      fmt.Sprintf("获取信息失败: %v", err),
			SupportedActions: []string{},
		}
	}

	return ModuleInfo{
		Name:             resp.Name,
		Version:          resp.Version,
		Description:      resp.Description,
		SupportedActions: resp.SupportedActions,
	}
}

// HandleMessage 实现了Module接口的HandleMessage方法
func (c *GRPCClient) HandleMessage(messageType string, messageID string, timestamp int64, payload map[string]interface{}) (map[string]interface{}, error) {
	// 将参数转换为JSON
	payloadJSON, err := ConfigToJSON(payload)
	if err != nil {
		return nil, fmt.Errorf("序列化消息负载失败: %w", err)
	}

	// 添加超时控制，避免执行时间过长
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 调用gRPC服务
	resp, err := c.client.HandleMessage(ctx, &pb.MessageRequest{
		MessageType: messageType,
		MessageId:   messageID,
		Timestamp:   timestamp,
		Payload:     payloadJSON,
	})
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("处理消息超时: %s", messageType)
		}
		return nil, fmt.Errorf("gRPC调用失败: %w", err)
	}

	// 检查响应
	if !resp.Success {
		return nil, fmt.Errorf("处理消息失败: %s", resp.ErrorMessage)
	}

	// 将JSON结果转换为map
	if resp.Response == "" {
		return make(map[string]interface{}), nil
	}

	result, err := JSONToConfig(resp.Response)
	if err != nil {
		return nil, fmt.Errorf("解析结果失败: %w", err)
	}

	return result, nil
}
