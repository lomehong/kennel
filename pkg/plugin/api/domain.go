package api

import (
	"context"
	"net/http"
	"time"
)

// ServicePlugin 定义了服务类型插件的接口
// 服务插件提供可调用的服务功能，如API服务、数据处理服务等
type ServicePlugin interface {
	Plugin // 继承基础插件接口

	// GetService 返回插件提供的服务实例
	// 返回值是一个通用接口，调用方需要根据插件类型进行类型断言
	GetService() (interface{}, error)

	// RegisterHandler 注册服务处理器
	// path: 服务路径
	// handler: 处理器实例
	RegisterHandler(path string, handler interface{}) error
}

// UIPlugin 定义了UI类型插件的接口
// UI插件提供用户界面组件，可以集成到主应用的界面中
type UIPlugin interface {
	Plugin // 继承基础插件接口

	// GetUIResources 返回插件提供的UI资源
	// 如CSS、JavaScript、图片等静态资源
	GetUIResources() ([]UIResource, error)

	// GetUIRoutes 返回插件提供的UI路由
	// 定义了插件UI组件如何集成到主应用的路由系统
	GetUIRoutes() ([]UIRoute, error)
}

// UIResource 定义了UI资源
type UIResource struct {
	Path        string // 资源路径
	ContentType string // 内容类型
	Data        []byte // 资源数据
}

// UIRoute 定义了UI路由
type UIRoute struct {
	Path        string    // 路由路径
	Component   string    // 组件名称
	Title       string    // 页面标题
	Icon        string    // 图标
	Permissions []string  // 所需权限
	Children    []UIRoute // 子路由
}

// DataProcessorPlugin 定义了数据处理插件的接口
// 数据处理插件提供数据转换、分析、过滤等功能
type DataProcessorPlugin interface {
	Plugin // 继承基础插件接口

	// ProcessData 处理数据
	// ctx: 上下文
	// data: 输入数据
	// 返回: 处理后的数据和错误
	ProcessData(ctx context.Context, data interface{}) (interface{}, error)

	// GetSupportedDataTypes 返回支持的数据类型
	// 返回: 支持的数据类型列表
	GetSupportedDataTypes() []string
}

// SecurityPlugin 定义了安全插件的接口
// 安全插件提供认证、授权、加密等安全相关功能
type SecurityPlugin interface {
	Plugin // 继承基础插件接口

	// ValidateRequest 验证请求
	// ctx: 上下文
	// request: 请求对象
	// 返回: 验证结果和错误
	ValidateRequest(ctx context.Context, request interface{}) (bool, error)

	// EncryptData 加密数据
	// ctx: 上下文
	// data: 待加密数据
	// 返回: 加密后的数据和错误
	EncryptData(ctx context.Context, data []byte) ([]byte, error)

	// DecryptData 解密数据
	// ctx: 上下文
	// data: 待解密数据
	// 返回: 解密后的数据和错误
	DecryptData(ctx context.Context, data []byte) ([]byte, error)
}

// HTTPHandlerPlugin 定义了HTTP处理器插件的接口
// HTTP处理器插件提供HTTP请求处理功能
type HTTPHandlerPlugin interface {
	Plugin // 继承基础插件接口

	// GetHTTPHandler 返回HTTP处理器
	// 返回: HTTP处理器和错误
	GetHTTPHandler() (http.Handler, error)

	// GetBasePath 返回基础路径
	// 返回: 处理器的基础路径
	GetBasePath() string
}

// StoragePlugin 定义了存储插件的接口
// 存储插件提供数据存储功能
type StoragePlugin interface {
	Plugin // 继承基础插件接口

	// Store 存储数据
	// ctx: 上下文
	// key: 键
	// value: 值
	// 返回: 错误
	Store(ctx context.Context, key string, value []byte) error

	// Retrieve 检索数据
	// ctx: 上下文
	// key: 键
	// 返回: 值和错误
	Retrieve(ctx context.Context, key string) ([]byte, error)

	// Delete 删除数据
	// ctx: 上下文
	// key: 键
	// 返回: 错误
	Delete(ctx context.Context, key string) error

	// List 列出键
	// ctx: 上下文
	// prefix: 前缀
	// 返回: 键列表和错误
	List(ctx context.Context, prefix string) ([]string, error)
}

// EventHandlerPlugin 定义了事件处理器插件的接口
// 事件处理器插件提供事件处理功能
type EventHandlerPlugin interface {
	Plugin // 继承基础插件接口

	// HandleEvent 处理事件
	// ctx: 上下文
	// event: 事件
	// 返回: 处理结果和错误
	HandleEvent(ctx context.Context, event Event) (interface{}, error)

	// GetSupportedEventTypes 返回支持的事件类型
	// 返回: 支持的事件类型列表
	GetSupportedEventTypes() []string
}

// Event 定义了事件
type Event struct {
	Type      string                 // 事件类型
	Source    string                 // 事件源
	ID        string                 // 事件ID
	Timestamp time.Time              // 时间戳
	Data      map[string]interface{} // 事件数据
}
