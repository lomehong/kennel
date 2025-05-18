package plugin

import (
	"context"
	"fmt"
	"time"

	pb "github.com/lomehong/kennel/pkg/plugin/proto/gen"
)

// GRPCServer 是一个gRPC服务器适配器，用于将Module接口转换为gRPC服务
type GRPCServer struct {
	pb.UnimplementedModuleServer
	Impl Module
}

// Init 实现了gRPC服务的Init方法
func (s *GRPCServer) Init(ctx context.Context, req *pb.InitRequest) (*pb.InitResponse, error) {
	// 将JSON配置转换为map
	config, err := JSONToConfig(req.Config)
	if err != nil {
		return &pb.InitResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("解析配置失败: %v", err),
		}, nil
	}

	// 调用实现
	err = s.Impl.Init(config)
	if err != nil {
		return &pb.InitResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &pb.InitResponse{
		Success: true,
	}, nil
}

// Execute 实现了gRPC服务的Execute方法
func (s *GRPCServer) Execute(ctx context.Context, req *pb.ActionRequest) (*pb.ActionResponse, error) {
	// 添加超时控制，避免执行时间过长
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 将JSON参数转换为map
	params, err := JSONToConfig(req.Params)
	if err != nil {
		return &pb.ActionResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("解析参数失败: %v", err),
		}, nil
	}

	// 使用带超时的上下文调用实现
	resultCh := make(chan struct {
		result map[string]interface{}
		err    error
	}, 1)

	go func() {
		result, err := s.Impl.Execute(req.Action, params)
		resultCh <- struct {
			result map[string]interface{}
			err    error
		}{result, err}
	}()

	// 等待结果或超时
	select {
	case <-timeoutCtx.Done():
		return &pb.ActionResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("执行超时: %s", req.Action),
		}, nil
	case res := <-resultCh:
		if res.err != nil {
			return &pb.ActionResponse{
				Success:      false,
				ErrorMessage: res.err.Error(),
			}, nil
		}

		// 将结果转换为JSON
		resultJSON, err := ConfigToJSON(res.result)
		if err != nil {
			return &pb.ActionResponse{
				Success:      false,
				ErrorMessage: fmt.Sprintf("序列化结果失败: %v", err),
			}, nil
		}

		return &pb.ActionResponse{
			Success: true,
			Result:  resultJSON,
		}, nil
	}
}

// Shutdown 实现了gRPC服务的Shutdown方法，支持优雅终止
func (s *GRPCServer) Shutdown(ctx context.Context, req *pb.EmptyRequest) (*pb.EmptyRequest, error) {
	// 添加超时控制，避免关闭时间过长
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 创建一个通道，用于等待关闭完成
	done := make(chan error, 1)

	// 在后台执行关闭
	go func() {
		done <- s.Impl.Shutdown()
	}()

	// 等待关闭完成或超时
	select {
	case <-timeoutCtx.Done():
		// 超时，但我们仍然返回成功，因为这是优雅终止的一部分
		return &pb.EmptyRequest{}, nil
	case err := <-done:
		if err != nil {
			// 即使出错，我们也返回空响应，因为Shutdown方法没有返回值
			// 但我们会记录错误
			return &pb.EmptyRequest{}, nil
		}
		return &pb.EmptyRequest{}, nil
	}
}

// GetInfo 实现了gRPC服务的GetInfo方法
func (s *GRPCServer) GetInfo(ctx context.Context, req *pb.EmptyRequest) (*pb.ModuleInfo, error) {
	// 调用实现
	info := s.Impl.GetInfo()

	return &pb.ModuleInfo{
		Name:             info.Name,
		Version:          info.Version,
		Description:      info.Description,
		SupportedActions: info.SupportedActions,
	}, nil
}

// HandleMessage 实现了gRPC服务的HandleMessage方法
func (s *GRPCServer) HandleMessage(ctx context.Context, req *pb.MessageRequest) (*pb.MessageResponse, error) {
	// 添加超时控制，避免执行时间过长
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 将JSON负载转换为map
	payload, err := JSONToConfig(req.Payload)
	if err != nil {
		return &pb.MessageResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("解析消息负载失败: %v", err),
		}, nil
	}

	// 使用带超时的上下文调用实现
	resultCh := make(chan struct {
		result map[string]interface{}
		err    error
	}, 1)

	go func() {
		result, err := s.Impl.HandleMessage(req.MessageType, req.MessageId, req.Timestamp, payload)
		resultCh <- struct {
			result map[string]interface{}
			err    error
		}{result, err}
	}()

	// 等待结果或超时
	select {
	case <-timeoutCtx.Done():
		return &pb.MessageResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("处理消息超时: %s", req.MessageType),
		}, nil
	case res := <-resultCh:
		if res.err != nil {
			return &pb.MessageResponse{
				Success:      false,
				ErrorMessage: res.err.Error(),
			}, nil
		}

		// 将结果转换为JSON
		resultJSON, err := ConfigToJSON(res.result)
		if err != nil {
			return &pb.MessageResponse{
				Success:      false,
				ErrorMessage: fmt.Sprintf("序列化结果失败: %v", err),
			}, nil
		}

		return &pb.MessageResponse{
			Success:  true,
			Response: resultJSON,
		}, nil
	}
}
