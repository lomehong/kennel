package main

import (
	"github.com/lomehong/kennel/pkg/utils"
)

// getString 从map中获取字符串值 - 使用公共工具包
func getString(m map[string]interface{}, key, defaultValue string) string {
	return utils.GetString(m, key, defaultValue)
}

// getBool 从map中获取布尔值 - 使用公共工具包
func getBool(m map[string]interface{}, key string, defaultValue bool) bool {
	return utils.GetBool(m, key, defaultValue)
}

// convertAlertsToMap 将警报列表转换为map切片
func convertAlertsToMap(alerts []DLPAlert) []map[string]interface{} {
	return AlertsToMap(alerts)
}
