package utils

import (
	"bytes"
	"encoding/json"
	"strings"
)

// MapToJSON 将map转换为JSON字符串
func MapToJSON(data map[string]interface{}) (string, error) {
	// 使用预分配内存的缓冲区，避免多次内存分配
	buffer := &bytes.Buffer{}
	buffer.Grow(1024) // 预分配1KB的空间，可以根据实际情况调整
	
	encoder := json.NewEncoder(buffer)
	if err := encoder.Encode(data); err != nil {
		return "", err
	}
	
	// 去除json.Encoder添加的换行符
	result := buffer.String()
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}
	
	return result, nil
}

// JSONToMap 将JSON字符串转换为map
func JSONToMap(jsonStr string) (map[string]interface{}, error) {
	var result map[string]interface{}
	
	decoder := json.NewDecoder(strings.NewReader(jsonStr))
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	
	return result, nil
}

// StructToMap 将结构体转换为map
func StructToMap(data interface{}) (map[string]interface{}, error) {
	// 先转换为JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	
	// 再转换为map
	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, err
	}
	
	return result, nil
}

// ConvertToMapSlice 将对象切片转换为map切片
func ConvertToMapSlice(objects interface{}) ([]map[string]interface{}, error) {
	// 先转换为JSON
	jsonData, err := json.Marshal(objects)
	if err != nil {
		return nil, err
	}
	
	// 再转换为map切片
	var result []map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, err
	}
	
	return result, nil
}
