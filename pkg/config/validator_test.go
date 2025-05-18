package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigSchema(t *testing.T) {
	// 创建配置模式
	schema := NewConfigSchema()

	// 添加属性
	schema.AddProperty("name", SchemaItem{
		Type:        SchemaTypeString,
		Required:    true,
		Description: "名称",
		Validator:   StringValidator(3, 20, ""),
	})

	schema.AddProperty("age", SchemaItem{
		Type:        SchemaTypeInt,
		Required:    true,
		Description: "年龄",
		Validator:   NumberValidator(0, 120, 0),
	})

	schema.AddProperty("email", SchemaItem{
		Type:        SchemaTypeString,
		Required:    false,
		Default:     "default@example.com",
		Description: "电子邮件",
		Validator:   StringValidator(5, 100, ".+@.+\\..+"),
	})

	schema.AddProperty("tags", SchemaItem{
		Type:        SchemaTypeArray,
		Required:    false,
		Description: "标签",
		Items: &SchemaItem{
			Type: SchemaTypeString,
		},
		Validator: ArrayValidator(0, 5, true),
	})

	schema.AddProperty("address", SchemaItem{
		Type:        SchemaTypeObject,
		Required:    false,
		Description: "地址",
		Properties: map[string]SchemaItem{
			"city": {
				Type:     SchemaTypeString,
				Required: true,
			},
			"zipcode": {
				Type:     SchemaTypeString,
				Required: false,
			},
		},
	})

	// 验证有效配置
	validConfig := map[string]interface{}{
		"name": "John Doe",
		"age":  30,
		"tags": []interface{}{"tag1", "tag2"},
		"address": map[string]interface{}{
			"city":    "New York",
			"zipcode": "10001",
		},
	}
	err := schema.Validate(validConfig)
	assert.NoError(t, err)

	// 验证默认值
	assert.Equal(t, "default@example.com", validConfig["email"])

	// 验证缺少必需属性
	invalidConfig1 := map[string]interface{}{
		"age": 30,
	}
	err = schema.Validate(invalidConfig1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "缺少必需属性: name")

	// 验证类型错误
	invalidConfig2 := map[string]interface{}{
		"name": "John Doe",
		"age":  "thirty",
	}
	err = schema.Validate(invalidConfig2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "应该是整数类型")

	// 验证自定义验证器
	invalidConfig3 := map[string]interface{}{
		"name": "Jo",
		"age":  30,
	}
	err = schema.Validate(invalidConfig3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "长度应该大于等于 3")

	// 验证嵌套对象
	invalidConfig4 := map[string]interface{}{
		"name": "John Doe",
		"age":  30,
		"address": map[string]interface{}{
			"zipcode": "10001",
		},
	}
	err = schema.Validate(invalidConfig4)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "缺少必需属性: address.city")
}

func TestStringValidator(t *testing.T) {
	// 创建字符串验证器
	validator := StringValidator(3, 10, "^[a-z]+$")

	// 验证有效字符串
	err := validator("abcdef")
	assert.NoError(t, err)

	// 验证过短字符串
	err = validator("ab")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "长度应该大于等于 3")

	// 验证过长字符串
	err = validator("abcdefghijk")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "长度应该小于等于 10")

	// 验证不匹配正则表达式的字符串
	err = validator("abc123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不匹配正则表达式")

	// 验证非字符串类型
	err = validator(123)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "应该是字符串类型")
}

func TestNumberValidator(t *testing.T) {
	// 创建数字验证器
	validator := NumberValidator(10, 100, 5)

	// 验证有效数字
	err := validator(15)
	assert.NoError(t, err)

	// 验证过小数字
	err = validator(5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "应该大于等于 10")

	// 验证过大数字
	err = validator(110)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "应该小于等于 100")

	// 验证非倍数
	err = validator(12)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "应该是 5 的倍数")

	// 验证非数字类型
	err = validator("15")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "应该是数字类型")
}

func TestArrayValidator(t *testing.T) {
	// 创建数组验证器
	validator := ArrayValidator(2, 5, true)

	// 验证有效数组
	err := validator([]interface{}{"a", "b", "c"})
	assert.NoError(t, err)

	// 验证过短数组
	err = validator([]interface{}{"a"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "数组长度应该大于等于 2")

	// 验证过长数组
	err = validator([]interface{}{"a", "b", "c", "d", "e", "f"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "数组长度应该小于等于 5")

	// 验证非唯一项
	err = validator([]interface{}{"a", "b", "a"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "数组应该包含唯一项")

	// 验证非数组类型
	err = validator("abc")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "应该是数组类型")
}

func TestEnumValidator(t *testing.T) {
	// 创建枚举验证器
	validator := EnumValidator([]interface{}{"a", "b", "c"})

	// 验证有效值
	err := validator("a")
	assert.NoError(t, err)

	// 验证无效值
	err = validator("d")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "应该是以下值之一")
}

func TestRequiredFieldsValidator(t *testing.T) {
	// 创建必需字段验证器
	validator := RequiredFieldsValidator("name", "age", "address.city")

	// 验证有效配置
	validConfig := map[string]interface{}{
		"name": "John Doe",
		"age":  30,
		"address": map[string]interface{}{
			"city":    "New York",
			"zipcode": "10001",
		},
	}
	err := validator(validConfig)
	assert.NoError(t, err)

	// 验证缺少顶级字段
	invalidConfig1 := map[string]interface{}{
		"age": 30,
		"address": map[string]interface{}{
			"city":    "New York",
			"zipcode": "10001",
		},
	}
	err = validator(invalidConfig1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "缺少必需字段: name")

	// 验证缺少嵌套字段
	invalidConfig2 := map[string]interface{}{
		"name": "John Doe",
		"age":  30,
		"address": map[string]interface{}{
			"zipcode": "10001",
		},
	}
	err = validator(invalidConfig2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "缺少必需字段: address.city")
}

func TestTypeValidator(t *testing.T) {
	// 创建类型验证器
	validator := TypeValidator("age", SchemaTypeInt)

	// 验证有效类型
	validConfig := map[string]interface{}{
		"age": 30,
	}
	err := validator(validConfig)
	assert.NoError(t, err)

	// 验证无效类型
	invalidConfig := map[string]interface{}{
		"age": "thirty",
	}
	err = validator(invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "应该是整数类型")

	// 验证嵌套字段
	nestedValidator := TypeValidator("address.zipcode", SchemaTypeString)
	nestedConfig := map[string]interface{}{
		"address": map[string]interface{}{
			"zipcode": 10001,
		},
	}
	err = nestedValidator(nestedConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "应该是字符串类型")
}
