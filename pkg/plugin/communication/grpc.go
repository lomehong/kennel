package communication

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// GRPCCommunication gRPC通信实现
type GRPCCommunication struct {
	// 服务映射
	services map[string]interface{}
	
	// 互斥锁
	mu sync.RWMutex
	
	// 日志记录器
	logger hclog.Logger
	
	// 上下文
	ctx context.Context
	
	// 取消函数
	cancel context.CancelFunc
	
	// gRPC服务器
	server *grpc.Server
	
	// gRPC客户端连接
	clientConn *grpc.ClientConn
	
	// 事件总线
	eventBus EventBus
	
	// 地址
	address string
	
	// 是否为服务器
	isServer bool
	
	// 是否已关闭
	closed bool
}

// GRPCOptions gRPC选项
type GRPCOptions struct {
	// 地址
	Address string
	
	// 是否为服务器
	IsServer bool
	
	// 日志记录器
	Logger hclog.Logger
	
	// 事件总线
	EventBus EventBus
	
	// 服务注册函数
	ServiceRegistrar func(*grpc.Server)
	
	// 客户端连接选项
	DialOptions []grpc.DialOption
	
	// 服务器选项
	ServerOptions []grpc.ServerOption
	
	// 超时时间
	Timeout time.Duration
}

// NewGRPCCommunication 创建一个新的gRPC通信
func NewGRPCCommunication(options GRPCOptions) (*GRPCCommunication, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	// 设置默认值
	if options.Logger == nil {
		options.Logger = hclog.NewNullLogger()
	}
	
	if options.EventBus == nil {
		options.EventBus = NewEventBus(options.Logger)
	}
	
	if options.Timeout == 0 {
		options.Timeout = 30 * time.Second
	}
	
	// 创建通信
	comm := &GRPCCommunication{
		services: make(map[string]interface{}),
		logger:   options.Logger.Named("grpc-communication"),
		ctx:      ctx,
		cancel:   cancel,
		eventBus: options.EventBus,
		address:  options.Address,
		isServer: options.IsServer,
		closed:   false,
	}
	
	// 如果是服务器，启动服务器
	if options.IsServer {
		if err := comm.startServer(options); err != nil {
			cancel()
			return nil, fmt.Errorf("启动gRPC服务器失败: %w", err)
		}
	} else {
		// 否则，创建客户端连接
		if err := comm.connectClient(options); err != nil {
			cancel()
			return nil, fmt.Errorf("连接gRPC服务器失败: %w", err)
		}
	}
	
	return comm, nil
}

// startServer 启动服务器
func (c *GRPCCommunication) startServer(options GRPCOptions) error {
	// 创建服务器选项
	serverOptions := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     15 * time.Minute,
			MaxConnectionAge:      30 * time.Minute,
			MaxConnectionAgeGrace: 5 * time.Minute,
			Time:                  5 * time.Minute,
			Timeout:               20 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             1 * time.Minute,
			PermitWithoutStream: true,
		}),
	}
	
	// 添加自定义选项
	serverOptions = append(serverOptions, options.ServerOptions...)
	
	// 创建服务器
	c.server = grpc.NewServer(serverOptions...)
	
	// 注册服务
	if options.ServiceRegistrar != nil {
		options.ServiceRegistrar(c.server)
	}
	
	// 启用反射
	reflection.Register(c.server)
	
	// 启动服务器
	lis, err := net.Listen("tcp", options.Address)
	if err != nil {
		return fmt.Errorf("监听地址 %s 失败: %w", options.Address, err)
	}
	
	c.logger.Info("启动gRPC服务器", "address", options.Address)
	
	// 在goroutine中启动服务器
	go func() {
		if err := c.server.Serve(lis); err != nil {
			c.logger.Error("gRPC服务器错误", "error", err)
		}
	}()
	
	return nil
}

// connectClient 连接客户端
func (c *GRPCCommunication) connectClient(options GRPCOptions) error {
	// 创建客户端选项
	dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             20 * time.Second,
			PermitWithoutStream: true,
		}),
	}
	
	// 添加自定义选项
	dialOptions = append(dialOptions, options.DialOptions...)
	
	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(c.ctx, options.Timeout)
	defer cancel()
	
	// 连接服务器
	c.logger.Info("连接gRPC服务器", "address", options.Address)
	conn, err := grpc.DialContext(ctx, options.Address, dialOptions...)
	if err != nil {
		return fmt.Errorf("连接gRPC服务器 %s 失败: %w", options.Address, err)
	}
	
	c.clientConn = conn
	return nil
}

// RegisterService 注册服务
func (c *GRPCCommunication) RegisterService(service interface{}) error {
	if service == nil {
		return fmt.Errorf("服务不能为空")
	}
	
	// 获取服务名称
	// 这里简化处理，实际应该通过反射获取
	name := fmt.Sprintf("%T", service)
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.services[name] = service
	return nil
}

// GetService 获取服务
func (c *GRPCCommunication) GetService(name string) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	service, ok := c.services[name]
	if !ok {
		return nil, fmt.Errorf("服务不存在: %s", name)
	}
	
	return service, nil
}

// SendMessage 发送消息
func (c *GRPCCommunication) SendMessage(target string, message interface{}) error {
	// 在实际实现中，这里应该使用gRPC客户端发送消息
	// 这里简化处理，通过事件总线发送
	return c.eventBus.Publish(Event{
		Type:      EventType(target),
		Source:    "grpc",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"message": message,
		},
	})
}

// ReceiveMessage 接收消息
func (c *GRPCCommunication) ReceiveMessage() (string, interface{}, error) {
	// 在实际实现中，这里应该使用gRPC服务器接收消息
	// 这里简化处理，返回错误
	return "", nil, fmt.Errorf("不支持直接接收消息，请使用Subscribe")
}

// Subscribe 订阅主题
func (c *GRPCCommunication) Subscribe(topic string, handler func(message interface{})) error {
	// 通过事件总线订阅
	_, err := c.eventBus.Subscribe(EventType(topic), func(event Event) error {
		// 获取消息
		message, ok := event.Data["message"]
		if !ok {
			return fmt.Errorf("消息不存在")
		}
		
		// 调用处理器
		handler(message)
		return nil
	})
	
	return err
}

// Publish 发布消息到主题
func (c *GRPCCommunication) Publish(topic string, message interface{}) error {
	// 通过事件总线发布
	return c.eventBus.Publish(Event{
		Type:      EventType(topic),
		Source:    "grpc",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"message": message,
		},
	})
}

// Close 关闭通信
func (c *GRPCCommunication) Close() error {
	// 检查是否已关闭
	if c.closed {
		return nil
	}
	
	// 设置为已关闭
	c.closed = true
	
	// 取消上下文
	c.cancel()
	
	// 关闭事件总线
	if err := c.eventBus.Close(); err != nil {
		c.logger.Error("关闭事件总线失败", "error", err)
	}
	
	// 如果是服务器，停止服务器
	if c.isServer && c.server != nil {
		c.logger.Info("停止gRPC服务器")
		c.server.GracefulStop()
	}
	
	// 如果是客户端，关闭连接
	if !c.isServer && c.clientConn != nil {
		c.logger.Info("关闭gRPC客户端连接")
		if err := c.clientConn.Close(); err != nil {
			c.logger.Error("关闭gRPC客户端连接失败", "error", err)
		}
	}
	
	c.logger.Info("gRPC通信已关闭")
	return nil
}
