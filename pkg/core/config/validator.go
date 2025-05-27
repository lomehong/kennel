package config

import (
	"fmt"
	"reflect"
	"strings"
)

// PluginConfigValidator 插件配置验证器
type PluginConfigValidator struct {
	// 插件ID
	PluginID string

	// 必需字段
	RequiredFields []string

	// 字段类型
	FieldTypes map[string]reflect.Kind

	// 字段验证器
	FieldValidators map[string]FieldValidator

	// 默认值
	Defaults map[string]interface{}

	// 架构
	Schema map[string]interface{}
}

// FieldValidator 字段验证器
type FieldValidator func(value interface{}) error

// 预定义验证器

// StringLengthValidator 字符串长度验证器
func StringLengthValidator(min, max int) FieldValidator {
	return func(value interface{}) error {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("期望字符串类型")
		}
		if len(str) < min {
			return fmt.Errorf("字符串长度不能少于 %d 个字符", min)
		}
		if len(str) > max {
			return fmt.Errorf("字符串长度不能超过 %d 个字符", max)
		}
		return nil
	}
}

// StringEnumValidator 字符串枚举验证器
func StringEnumValidator(validValues ...string) FieldValidator {
	return func(value interface{}) error {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("期望字符串类型")
		}
		for _, valid := range validValues {
			if str == valid {
				return nil
			}
		}
		return fmt.Errorf("值必须是以下之一: %v", validValues)
	}
}

// IntRangeValidator 整数范围验证器
func IntRangeValidator(min, max int) FieldValidator {
	return func(value interface{}) error {
		var intVal int
		switch v := value.(type) {
		case int:
			intVal = v
		case int64:
			intVal = int(v)
		case float64:
			intVal = int(v)
		default:
			return fmt.Errorf("期望整数类型")
		}
		if intVal < min {
			return fmt.Errorf("值不能小于 %d", min)
		}
		if intVal > max {
			return fmt.Errorf("值不能大于 %d", max)
		}
		return nil
	}
}

// URLValidator URL格式验证器
func URLValidator() FieldValidator {
	return func(value interface{}) error {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("期望字符串类型")
		}
		if str == "" {
			return nil // 允许空值
		}
		if !strings.HasPrefix(str, "http://") && !strings.HasPrefix(str, "https://") {
			return fmt.Errorf("URL必须以http://或https://开头")
		}
		return nil
	}
}

// DurationValidator 时间间隔验证器
func DurationValidator(min, max int) FieldValidator {
	return func(value interface{}) error {
		var intVal int
		switch v := value.(type) {
		case int:
			intVal = v
		case int64:
			intVal = int(v)
		case float64:
			intVal = int(v)
		default:
			return fmt.Errorf("期望整数类型（秒）")
		}
		if intVal < min {
			return fmt.Errorf("时间间隔不能小于 %d 秒", min)
		}
		if intVal > max {
			return fmt.Errorf("时间间隔不能大于 %d 秒", max)
		}
		return nil
	}
}

// PathValidator 路径验证器
func PathValidator() FieldValidator {
	return func(value interface{}) error {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("期望字符串类型")
		}
		if str == "" {
			return fmt.Errorf("路径不能为空")
		}
		// 检查路径中是否包含非法字符
		invalidChars := []string{"<", ">", ":", "\"", "|", "?", "*"}
		for _, char := range invalidChars {
			if strings.Contains(str, char) {
				return fmt.Errorf("路径包含非法字符: %s", char)
			}
		}
		return nil
	}
}

// ArrayValidator 数组验证器
func ArrayValidator(minLen, maxLen int) FieldValidator {
	return func(value interface{}) error {
		arr, ok := value.([]interface{})
		if !ok {
			return fmt.Errorf("期望数组类型")
		}
		if len(arr) < minLen {
			return fmt.Errorf("数组长度不能少于 %d", minLen)
		}
		if len(arr) > maxLen {
			return fmt.Errorf("数组长度不能超过 %d", maxLen)
		}
		return nil
	}
}

// PortValidator 端口号验证器
func PortValidator() FieldValidator {
	return func(value interface{}) error {
		var intVal int
		switch v := value.(type) {
		case int:
			intVal = v
		case int64:
			intVal = int(v)
		case float64:
			intVal = int(v)
		default:
			return fmt.Errorf("期望整数类型")
		}
		if intVal < 1 || intVal > 65535 {
			return fmt.Errorf("端口号必须在1-65535范围内")
		}
		return nil
	}
}

// NewPluginConfigValidator 创建插件配置验证器
func NewPluginConfigValidator(pluginID string) *PluginConfigValidator {
	return &PluginConfigValidator{
		PluginID:        pluginID,
		RequiredFields:  make([]string, 0),
		FieldTypes:      make(map[string]reflect.Kind),
		FieldValidators: make(map[string]FieldValidator),
		Defaults:        make(map[string]interface{}),
		Schema:          make(map[string]interface{}),
	}
}

// AddRequiredField 添加必需字段
func (v *PluginConfigValidator) AddRequiredField(field string) *PluginConfigValidator {
	v.RequiredFields = append(v.RequiredFields, field)
	return v
}

// AddFieldType 添加字段类型
func (v *PluginConfigValidator) AddFieldType(field string, kind reflect.Kind) *PluginConfigValidator {
	v.FieldTypes[field] = kind
	return v
}

// AddFieldValidator 添加字段验证器
func (v *PluginConfigValidator) AddFieldValidator(field string, validator FieldValidator) *PluginConfigValidator {
	v.FieldValidators[field] = validator
	return v
}

// AddDefault 添加默认值
func (v *PluginConfigValidator) AddDefault(field string, value interface{}) *PluginConfigValidator {
	v.Defaults[field] = value
	return v
}

// AddSchema 添加架构
func (v *PluginConfigValidator) AddSchema(field string, schema interface{}) *PluginConfigValidator {
	v.Schema[field] = schema
	return v
}

// Validate 验证配置
func (v *PluginConfigValidator) Validate(config map[string]interface{}) error {
	// 获取插件配置
	pluginConfig, ok := config["plugins"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("配置中缺少 'plugins' 部分")
	}

	// 获取特定插件配置
	specificConfig, ok := pluginConfig[v.PluginID].(map[string]interface{})
	if !ok {
		// 如果插件配置不存在，使用默认值创建
		specificConfig = make(map[string]interface{})
		for field, value := range v.Defaults {
			specificConfig[field] = value
		}
		pluginConfig[v.PluginID] = specificConfig
	}

	// 验证必需字段
	for _, field := range v.RequiredFields {
		if _, exists := specificConfig[field]; !exists {
			// 如果有默认值，使用默认值
			if defaultValue, hasDefault := v.Defaults[field]; hasDefault {
				specificConfig[field] = defaultValue
			} else {
				return fmt.Errorf("插件 %s 配置缺少必需字段: %s", v.PluginID, field)
			}
		}
	}

	// 验证字段类型
	for field, kind := range v.FieldTypes {
		if value, exists := specificConfig[field]; exists {
			if err := validateType(value, kind); err != nil {
				return fmt.Errorf("插件 %s 配置字段 %s 类型错误: %w", v.PluginID, field, err)
			}
		}
	}

	// 验证字段值
	for field, validator := range v.FieldValidators {
		if value, exists := specificConfig[field]; exists {
			if err := validator(value); err != nil {
				return fmt.Errorf("插件 %s 配置字段 %s 验证失败: %w", v.PluginID, field, err)
			}
		}
	}

	return nil
}

// GetDefaults 获取默认配置
func (v *PluginConfigValidator) GetDefaults() map[string]interface{} {
	return v.Defaults
}

// GetSchema 获取配置架构
func (v *PluginConfigValidator) GetSchema() map[string]interface{} {
	return v.Schema
}

// validateType 验证类型
func validateType(value interface{}, kind reflect.Kind) error {
	if value == nil {
		return nil
	}

	actualKind := reflect.TypeOf(value).Kind()

	// 处理数字类型的兼容性
	if kind == reflect.Float64 && (actualKind == reflect.Int || actualKind == reflect.Int64) {
		return nil // int可以转换为float64
	}
	if kind == reflect.Int && (actualKind == reflect.Float64 || actualKind == reflect.Int64) {
		return nil // float64和int64可以转换为int
	}
	if kind == reflect.Int64 && (actualKind == reflect.Int || actualKind == reflect.Float64) {
		return nil // int和float64可以转换为int64
	}

	if actualKind != kind {
		return fmt.Errorf("期望类型 %s，实际类型 %s", kind, actualKind)
	}

	return nil
}

// StringValidator 字符串验证器
func StringValidator(allowedValues ...string) FieldValidator {
	return func(value interface{}) error {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("期望字符串类型")
		}

		if len(allowedValues) > 0 {
			for _, allowed := range allowedValues {
				if str == allowed {
					return nil
				}
			}
			return fmt.Errorf("值 %s 不在允许的值列表中: %s", str, strings.Join(allowedValues, ", "))
		}

		return nil
	}
}

// FloatRangeValidator 浮点数范围验证器
func FloatRangeValidator(min, max float64) FieldValidator {
	return func(value interface{}) error {
		var floatValue float64
		switch v := value.(type) {
		case float64:
			floatValue = v
		case int:
			floatValue = float64(v)
		case int64:
			floatValue = float64(v)
		default:
			return fmt.Errorf("期望浮点数类型")
		}

		if floatValue < min || floatValue > max {
			return fmt.Errorf("值 %f 不在允许的范围 [%f, %f] 内", floatValue, min, max)
		}

		return nil
	}
}

// BoolValidator 布尔值验证器
func BoolValidator() FieldValidator {
	return func(value interface{}) error {
		_, ok := value.(bool)
		if !ok {
			return fmt.Errorf("期望布尔类型")
		}
		return nil
	}
}

// MapValidator 映射验证器
func MapValidator() FieldValidator {
	return func(value interface{}) error {
		_, ok := value.(map[string]interface{})
		if !ok {
			return fmt.Errorf("期望映射类型")
		}
		return nil
	}
}

// SliceValidator 切片验证器
func SliceValidator() FieldValidator {
	return func(value interface{}) error {
		_, ok := value.([]interface{})
		if !ok {
			return fmt.Errorf("期望切片类型")
		}
		return nil
	}
}
