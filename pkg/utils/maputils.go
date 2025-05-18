package utils

// GetString 从map中获取字符串值
func GetString(m map[string]interface{}, key, defaultValue string) string {
	if value, ok := m[key].(string); ok {
		return value
	}
	return defaultValue
}

// GetBool 从map中获取布尔值
func GetBool(m map[string]interface{}, key string, defaultValue bool) bool {
	if value, ok := m[key].(bool); ok {
		return value
	}
	return defaultValue
}

// GetInt 从map中获取整数值
func GetInt(m map[string]interface{}, key string, defaultValue int) int {
	switch value := m[key].(type) {
	case int:
		return value
	case float64:
		return int(value)
	case float32:
		return int(value)
	default:
		return defaultValue
	}
}

// GetFloat 从map中获取浮点数值
func GetFloat(m map[string]interface{}, key string, defaultValue float64) float64 {
	switch value := m[key].(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	default:
		return defaultValue
	}
}

// GetStringSlice 从map中获取字符串切片
func GetStringSlice(m map[string]interface{}, key string, defaultValue []string) []string {
	if value, ok := m[key].([]string); ok {
		return value
	}
	
	if value, ok := m[key].([]interface{}); ok {
		result := make([]string, 0, len(value))
		for _, v := range value {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	
	return defaultValue
}

// GetStringMap 从map中获取字符串映射
func GetStringMap(m map[string]interface{}, key string, defaultValue map[string]interface{}) map[string]interface{} {
	if value, ok := m[key].(map[string]interface{}); ok {
		return value
	}
	return defaultValue
}
