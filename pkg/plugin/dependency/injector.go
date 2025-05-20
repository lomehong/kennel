package dependency

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/hashicorp/go-hclog"
)

// DependencyInjector 依赖注入器
// 负责注入插件依赖
type DependencyInjector struct {
	// 服务映射
	services map[string]interface{}

	// 工厂映射
	factories map[string]Factory

	// 单例映射
	singletons map[string]interface{}

	// 互斥锁
	mu sync.RWMutex

	// 日志记录器
	logger hclog.Logger
}

// Factory 工厂函数
// 用于创建服务实例
type Factory func() (interface{}, error)

// NewDependencyInjector 创建一个新的依赖注入器
func NewDependencyInjector(logger hclog.Logger) *DependencyInjector {
	if logger == nil {
		logger = hclog.NewNullLogger()
	}

	return &DependencyInjector{
		services:   make(map[string]interface{}),
		factories:  make(map[string]Factory),
		singletons: make(map[string]interface{}),
		logger:     logger.Named("dependency-injector"),
	}
}

// RegisterService 注册服务
func (i *DependencyInjector) RegisterService(name string, service interface{}) error {
	if service == nil {
		return fmt.Errorf("服务不能为空")
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	// 检查服务是否已注册
	if _, exists := i.services[name]; exists {
		return fmt.Errorf("服务 %s 已注册", name)
	}

	i.services[name] = service
	i.logger.Debug("注册服务", "name", name, "type", fmt.Sprintf("%T", service))
	return nil
}

// RegisterFactory 注册工厂
func (i *DependencyInjector) RegisterFactory(name string, factory Factory) error {
	if factory == nil {
		return fmt.Errorf("工厂不能为空")
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	// 检查工厂是否已注册
	if _, exists := i.factories[name]; exists {
		return fmt.Errorf("工厂 %s 已注册", name)
	}

	i.factories[name] = factory
	i.logger.Debug("注册工厂", "name", name)
	return nil
}

// GetService 获取服务
func (i *DependencyInjector) GetService(name string) (interface{}, error) {
	i.mu.RLock()

	// 检查单例
	if singleton, exists := i.singletons[name]; exists {
		i.mu.RUnlock()
		return singleton, nil
	}

	// 检查服务
	if service, exists := i.services[name]; exists {
		i.mu.RUnlock()
		return service, nil
	}

	// 检查工厂
	factory, exists := i.factories[name]
	i.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("服务 %s 未注册", name)
	}

	// 创建服务实例
	instance, err := factory()
	if err != nil {
		return nil, fmt.Errorf("创建服务 %s 失败: %w", name, err)
	}

	// 存储单例
	i.mu.Lock()
	i.singletons[name] = instance
	i.mu.Unlock()

	return instance, nil
}

// Inject 注入依赖
func (i *DependencyInjector) Inject(target interface{}) error {
	if target == nil {
		return fmt.Errorf("目标不能为空")
	}

	// 获取目标的反射值
	value := reflect.ValueOf(target)

	// 检查目标是否为指针
	if value.Kind() != reflect.Ptr {
		return fmt.Errorf("目标必须为指针")
	}

	// 获取目标的元素值
	elem := value.Elem()

	// 检查目标是否为结构体
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("目标必须为结构体指针")
	}

	// 获取目标的类型
	typ := elem.Type()

	// 遍历字段
	for j := 0; j < elem.NumField(); j++ {
		field := elem.Field(j)
		fieldType := typ.Field(j)

		// 检查字段是否可设置
		if !field.CanSet() {
			continue
		}

		// 获取注入标签
		tag := fieldType.Tag.Get("inject")
		if tag == "" {
			continue
		}

		// 获取服务
		service, err := i.GetService(tag)
		if err != nil {
			return fmt.Errorf("注入字段 %s 失败: %w", fieldType.Name, err)
		}

		// 获取服务的反射值
		serviceValue := reflect.ValueOf(service)

		// 检查类型是否兼容
		if !serviceValue.Type().AssignableTo(field.Type()) {
			return fmt.Errorf("服务 %s 类型 %s 与字段 %s 类型 %s 不兼容",
				tag, serviceValue.Type(), fieldType.Name, field.Type())
		}

		// 设置字段值
		field.Set(serviceValue)
		i.logger.Debug("注入依赖", "field", fieldType.Name, "service", tag)
	}

	return nil
}

// InjectByName 按名称注入依赖
func (i *DependencyInjector) InjectByName(target interface{}, fieldName string, serviceName string) error {
	if target == nil {
		return fmt.Errorf("目标不能为空")
	}

	// 获取目标的反射值
	value := reflect.ValueOf(target)

	// 检查目标是否为指针
	if value.Kind() != reflect.Ptr {
		return fmt.Errorf("目标必须为指针")
	}

	// 获取目标的元素值
	elem := value.Elem()

	// 检查目标是否为结构体
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("目标必须为结构体指针")
	}

	// 获取字段
	field := elem.FieldByName(fieldName)
	if !field.IsValid() {
		return fmt.Errorf("字段 %s 不存在", fieldName)
	}

	// 检查字段是否可设置
	if !field.CanSet() {
		return fmt.Errorf("字段 %s 不可设置", fieldName)
	}

	// 获取服务
	service, err := i.GetService(serviceName)
	if err != nil {
		return fmt.Errorf("获取服务 %s 失败: %w", serviceName, err)
	}

	// 获取服务的反射值
	serviceValue := reflect.ValueOf(service)

	// 检查类型是否兼容
	if !serviceValue.Type().AssignableTo(field.Type()) {
		return fmt.Errorf("服务 %s 类型 %s 与字段 %s 类型 %s 不兼容",
			serviceName, serviceValue.Type(), fieldName, field.Type())
	}

	// 设置字段值
	field.Set(serviceValue)
	i.logger.Debug("注入依赖", "field", fieldName, "service", serviceName)

	return nil
}

// InjectMethod 注入方法参数
func (i *DependencyInjector) InjectMethod(target interface{}, methodName string, args ...interface{}) ([]reflect.Value, error) {
	if target == nil {
		return nil, fmt.Errorf("目标不能为空")
	}

	// 获取目标的反射值
	value := reflect.ValueOf(target)

	// 获取方法
	method := value.MethodByName(methodName)
	if !method.IsValid() {
		return nil, fmt.Errorf("方法 %s 不存在", methodName)
	}

	// 获取方法类型
	methodType := method.Type()

	// 检查参数数量
	if len(args) > methodType.NumIn() {
		return nil, fmt.Errorf("参数数量不匹配: 期望 %d, 实际 %d", methodType.NumIn(), len(args))
	}

	// 创建参数列表
	in := make([]reflect.Value, methodType.NumIn())

	// 设置已提供的参数
	for j := 0; j < len(args); j++ {
		in[j] = reflect.ValueOf(args[j])
	}

	// 注入剩余参数
	for j := len(args); j < methodType.NumIn(); j++ {
		paramType := methodType.In(j)

		// 查找匹配的服务
		var service interface{}

		// 按类型查找服务
		i.mu.RLock()
		for _, s := range i.services {
			if reflect.TypeOf(s).AssignableTo(paramType) {
				service = s
				break
			}
		}
		i.mu.RUnlock()

		if service == nil {
			// 如果没有找到匹配的服务，尝试创建
			for name, factory := range i.factories {
				instance, err := factory()
				if err != nil {
					continue
				}

				if reflect.TypeOf(instance).AssignableTo(paramType) {
					service = instance

					// 存储单例
					i.mu.Lock()
					i.singletons[name] = instance
					i.mu.Unlock()

					break
				}
			}
		}

		if service == nil {
			return nil, fmt.Errorf("无法注入参数 %d: 找不到匹配的服务", j)
		}

		in[j] = reflect.ValueOf(service)
	}

	// 调用方法
	return method.Call(in), nil
}

// Clear 清除所有注册的服务和工厂
func (i *DependencyInjector) Clear() {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.services = make(map[string]interface{})
	i.factories = make(map[string]Factory)
	i.singletons = make(map[string]interface{})

	i.logger.Debug("清除所有服务和工厂")
}
