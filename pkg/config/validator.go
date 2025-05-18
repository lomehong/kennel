package config

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// SchemaType 表示配置项的类型
type SchemaType string

// 预定义配置项类型
const (
	SchemaTypeString SchemaType = "string"
	SchemaTypeInt    SchemaType = "int"
	SchemaTypeFloat  SchemaType = "float"
	SchemaTypeBool   SchemaType = "bool"
	SchemaTypeArray  SchemaType = "array"
	SchemaTypeObject SchemaType = "object"
	SchemaTypeAny    SchemaType = "any"
)

// SchemaItem 配置项模式
type SchemaItem struct {
	Type        SchemaType              // 类型
	Required    bool                    // 是否必需
	Default     interface{}             // 默认值
	Description string                  // 描述
	Validator   func(interface{}) error // 验证器
	Properties  map[string]SchemaItem   // 子属性（对象类型）
	Items       *SchemaItem             // 子项（数组类型）
}

// ConfigSchema 配置模式
type ConfigSchema struct {
	Properties map[string]SchemaItem // 属性
}

// NewConfigSchema 创建一个新的配置模式
func NewConfigSchema() *ConfigSchema {
	return &ConfigSchema{
		Properties: make(map[string]SchemaItem),
	}
}

// AddProperty 添加属性
func (s *ConfigSchema) AddProperty(name string, item SchemaItem) {
	s.Properties[name] = item
}

// Validate 验证配置
func (s *ConfigSchema) Validate(config map[string]interface{}) error {
	for name, item := range s.Properties {
		// 检查必需属性
		if item.Required {
			if _, ok := config[name]; !ok {
				return fmt.Errorf("缺少必需属性: %s", name)
			}
		}

		// 如果属性不存在但有默认值，则设置默认值
		if _, ok := config[name]; !ok && item.Default != nil {
			config[name] = item.Default
			continue
		}

		// 如果属性不存在且不是必需的，则跳过
		if _, ok := config[name]; !ok {
			continue
		}

		// 验证属性
		if err := validateValue(name, config[name], item); err != nil {
			return err
		}
	}

	return nil
}

// validateValue 验证值
func validateValue(name string, value interface{}, item SchemaItem) error {
	// 检查类型
	switch item.Type {
	case SchemaTypeString:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("属性 %s 应该是字符串类型", name)
		}
	case SchemaTypeInt:
		switch v := value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			// 整数类型
		case float32, float64:
			// 浮点数，检查是否为整数
			if float64(int(reflect.ValueOf(v).Float())) != reflect.ValueOf(v).Float() {
				return fmt.Errorf("属性 %s 应该是整数类型", name)
			}
		default:
			return fmt.Errorf("属性 %s 应该是整数类型", name)
		}
	case SchemaTypeFloat:
		switch value.(type) {
		case float32, float64, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			// 数字类型
		default:
			return fmt.Errorf("属性 %s 应该是浮点数类型", name)
		}
	case SchemaTypeBool:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("属性 %s 应该是布尔类型", name)
		}
	case SchemaTypeArray:
		arr, ok := value.([]interface{})
		if !ok {
			return fmt.Errorf("属性 %s 应该是数组类型", name)
		}
		if item.Items != nil {
			for i, v := range arr {
				if err := validateValue(fmt.Sprintf("%s[%d]", name, i), v, *item.Items); err != nil {
					return err
				}
			}
		}
	case SchemaTypeObject:
		obj, ok := value.(map[string]interface{})
		if !ok {
			return fmt.Errorf("属性 %s 应该是对象类型", name)
		}
		for propName, propItem := range item.Properties {
			// 检查必需属性
			if propItem.Required {
				if _, ok := obj[propName]; !ok {
					return fmt.Errorf("缺少必需属性: %s.%s", name, propName)
				}
			}

			// 如果属性不存在但有默认值，则设置默认值
			if _, ok := obj[propName]; !ok && propItem.Default != nil {
				obj[propName] = propItem.Default
				continue
			}

			// 如果属性不存在且不是必需的，则跳过
			if _, ok := obj[propName]; !ok {
				continue
			}

			// 验证属性
			if err := validateValue(fmt.Sprintf("%s.%s", name, propName), obj[propName], propItem); err != nil {
				return err
			}
		}
	case SchemaTypeAny:
		// 任意类型，不做验证
	default:
		return fmt.Errorf("未知类型: %s", item.Type)
	}

	// 调用自定义验证器
	if item.Validator != nil {
		if err := item.Validator(value); err != nil {
			return fmt.Errorf("属性 %s 验证失败: %w", name, err)
		}
	}

	return nil
}

// StringValidator 创建字符串验证器
func StringValidator(minLength, maxLength int, pattern string) func(interface{}) error {
	return func(value interface{}) error {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("应该是字符串类型")
		}

		if minLength > 0 && len(str) < minLength {
			return fmt.Errorf("长度应该大于等于 %d", minLength)
		}

		if maxLength > 0 && len(str) > maxLength {
			return fmt.Errorf("长度应该小于等于 %d", maxLength)
		}

		if pattern != "" {
			matched, err := regexp.MatchString(pattern, str)
			if err != nil {
				return fmt.Errorf("正则表达式错误: %w", err)
			}
			if !matched {
				return fmt.Errorf("不匹配正则表达式: %s", pattern)
			}
		}

		return nil
	}
}

// NumberValidator 创建数字验证器
func NumberValidator(min, max float64, multipleOf float64) func(interface{}) error {
	return func(value interface{}) error {
		var num float64
		switch v := value.(type) {
		case int:
			num = float64(v)
		case int8:
			num = float64(v)
		case int16:
			num = float64(v)
		case int32:
			num = float64(v)
		case int64:
			num = float64(v)
		case uint:
			num = float64(v)
		case uint8:
			num = float64(v)
		case uint16:
			num = float64(v)
		case uint32:
			num = float64(v)
		case uint64:
			num = float64(v)
		case float32:
			num = float64(v)
		case float64:
			num = v
		default:
			return fmt.Errorf("应该是数字类型")
		}

		if min != 0 && num < min {
			return fmt.Errorf("应该大于等于 %g", min)
		}

		if max != 0 && num > max {
			return fmt.Errorf("应该小于等于 %g", max)
		}

		if multipleOf != 0 && num/multipleOf != float64(int(num/multipleOf)) {
			return fmt.Errorf("应该是 %g 的倍数", multipleOf)
		}

		return nil
	}
}

// ArrayValidator 创建数组验证器
func ArrayValidator(minItems, maxItems int, uniqueItems bool) func(interface{}) error {
	return func(value interface{}) error {
		arr, ok := value.([]interface{})
		if !ok {
			return fmt.Errorf("应该是数组类型")
		}

		if minItems > 0 && len(arr) < minItems {
			return fmt.Errorf("数组长度应该大于等于 %d", minItems)
		}

		if maxItems > 0 && len(arr) > maxItems {
			return fmt.Errorf("数组长度应该小于等于 %d", maxItems)
		}

		if uniqueItems {
			seen := make(map[string]bool)
			for _, item := range arr {
				key := fmt.Sprintf("%v", item)
				if seen[key] {
					return fmt.Errorf("数组应该包含唯一项")
				}
				seen[key] = true
			}
		}

		return nil
	}
}

// EnumValidator 创建枚举验证器
func EnumValidator(values []interface{}) func(interface{}) error {
	return func(value interface{}) error {
		for _, v := range values {
			if reflect.DeepEqual(value, v) {
				return nil
			}
		}
		return fmt.Errorf("应该是以下值之一: %v", values)
	}
}

// SchemaValidator 创建模式验证器
func SchemaValidator(schema *ConfigSchema) ConfigValidator {
	return func(config map[string]interface{}) error {
		return schema.Validate(config)
	}
}

// RequiredFieldsValidator 创建必需字段验证器
func RequiredFieldsValidator(fields ...string) ConfigValidator {
	return func(config map[string]interface{}) error {
		for _, field := range fields {
			parts := strings.Split(field, ".")
			current := config
			found := true

			for i, part := range parts {
				if i == len(parts)-1 {
					if _, ok := current[part]; !ok {
						found = false
						break
					}
				} else {
					next, ok := current[part]
					if !ok {
						found = false
						break
					}
					nextMap, ok := next.(map[string]interface{})
					if !ok {
						found = false
						break
					}
					current = nextMap
				}
			}

			if !found {
				return fmt.Errorf("缺少必需字段: %s", field)
			}
		}
		return nil
	}
}

// TypeValidator 创建类型验证器
func TypeValidator(field string, expectedType SchemaType) ConfigValidator {
	return func(config map[string]interface{}) error {
		parts := strings.Split(field, ".")
		current := config
		var value interface{}
		found := true

		for i, part := range parts {
			if i == len(parts)-1 {
				value, found = current[part]
			} else {
				next, ok := current[part]
				if !ok {
					found = false
					break
				}
				nextMap, ok := next.(map[string]interface{})
				if !ok {
					found = false
					break
				}
				current = nextMap
			}
		}

		if !found {
			return nil // 字段不存在，跳过验证
		}

		switch expectedType {
		case SchemaTypeString:
			if _, ok := value.(string); !ok {
				return fmt.Errorf("字段 %s 应该是字符串类型", field)
			}
		case SchemaTypeInt:
			switch v := value.(type) {
			case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
				// 整数类型
			case float32, float64:
				// 浮点数，检查是否为整数
				if float64(int(reflect.ValueOf(v).Float())) != reflect.ValueOf(v).Float() {
					return fmt.Errorf("字段 %s 应该是整数类型", field)
				}
			default:
				return fmt.Errorf("字段 %s 应该是整数类型", field)
			}
		case SchemaTypeFloat:
			switch value.(type) {
			case float32, float64, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
				// 数字类型
			default:
				return fmt.Errorf("字段 %s 应该是浮点数类型", field)
			}
		case SchemaTypeBool:
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("字段 %s 应该是布尔类型", field)
			}
		case SchemaTypeArray:
			if _, ok := value.([]interface{}); !ok {
				return fmt.Errorf("字段 %s 应该是数组类型", field)
			}
		case SchemaTypeObject:
			if _, ok := value.(map[string]interface{}); !ok {
				return fmt.Errorf("字段 %s 应该是对象类型", field)
			}
		}

		return nil
	}
}
