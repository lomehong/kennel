package main

// DLPAlert 表示一个数据防泄漏警报
type DLPAlert struct {
	RuleID      string `json:"rule_id"`
	RuleName    string `json:"rule_name"`
	Content     string `json:"content"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Action      string `json:"action"`
	Timestamp   string `json:"timestamp"`
}

// AlertToMap 将警报转换为map
func AlertToMap(alert DLPAlert) map[string]interface{} {
	return map[string]interface{}{
		"rule_id":      alert.RuleID,
		"rule_name":    alert.RuleName,
		"content":      alert.Content,
		"source":       alert.Source,
		"destination":  alert.Destination,
		"action":       alert.Action,
		"timestamp":    alert.Timestamp,
	}
}

// AlertsToMap 将警报列表转换为map列表
func AlertsToMap(alerts []DLPAlert) []map[string]interface{} {
	result := make([]map[string]interface{}, len(alerts))
	for i, alert := range alerts {
		result[i] = AlertToMap(alert)
	}
	return result
}
