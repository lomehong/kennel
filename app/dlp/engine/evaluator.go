package engine

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/lomehong/kennel/pkg/logging"
)

// RuleEvaluatorImpl 规则评估器实现
type RuleEvaluatorImpl struct {
	logger              logging.Logger
	conditionEvaluator  ConditionEvaluator
}

// NewRuleEvaluator 创建规则评估器
func NewRuleEvaluator(logger logging.Logger) RuleEvaluator {
	return &RuleEvaluatorImpl{
		logger:             logger,
		conditionEvaluator: NewConditionEvaluator(logger),
	}
}

// EvaluateRule 评估规则
func (re *RuleEvaluatorImpl) EvaluateRule(rule *PolicyRule, context *DecisionContext) (*RuleEvaluationResult, error) {
	result := &RuleEvaluationResult{
		RuleID:     rule.ID,
		Matched:    false,
		Confidence: 0.0,
		Actions:    make([]*RuleAction, 0),
		Metadata:   make(map[string]interface{}),
	}

	// 评估所有条件
	allMatched := true
	totalConfidence := 0.0
	matchedConditions := 0

	for _, condition := range rule.Conditions {
		matched, err := re.conditionEvaluator.EvaluateCondition(condition, context)
		if err != nil {
			re.logger.Warn("条件评估失败", "rule_id", rule.ID, "condition", condition.Field, "error", err)
			continue
		}

		if matched {
			matchedConditions++
			totalConfidence += 1.0
		} else {
			allMatched = false
		}
	}

	// 计算置信度
	if len(rule.Conditions) > 0 {
		result.Confidence = totalConfidence / float64(len(rule.Conditions))
	}

	// 检查是否所有条件都匹配
	if allMatched && len(rule.Conditions) > 0 {
		result.Matched = true
		result.Actions = rule.Actions
		result.Reason = fmt.Sprintf("所有 %d 个条件都匹配", len(rule.Conditions))
	} else {
		result.Reason = fmt.Sprintf("只有 %d/%d 个条件匹配", matchedConditions, len(rule.Conditions))
	}

	result.Metadata["matched_conditions"] = matchedConditions
	result.Metadata["total_conditions"] = len(rule.Conditions)
	result.Metadata["rule_priority"] = rule.Priority

	return result, nil
}

// GetSupportedTypes 获取支持的规则类型
func (re *RuleEvaluatorImpl) GetSupportedTypes() []string {
	return []string{"security", "compliance", "audit", "custom"}
}

// CanEvaluate 检查是否能评估指定类型的规则
func (re *RuleEvaluatorImpl) CanEvaluate(ruleType string) bool {
	supportedTypes := re.GetSupportedTypes()
	for _, supportedType := range supportedTypes {
		if supportedType == ruleType {
			return true
		}
	}
	return false
}

// ConditionEvaluatorImpl 条件评估器实现
type ConditionEvaluatorImpl struct {
	logger logging.Logger
}

// NewConditionEvaluator 创建条件评估器
func NewConditionEvaluator(logger logging.Logger) ConditionEvaluator {
	return &ConditionEvaluatorImpl{
		logger: logger,
	}
}

// EvaluateCondition 评估条件
func (ce *ConditionEvaluatorImpl) EvaluateCondition(condition *RuleCondition, context *DecisionContext) (bool, error) {
	// 获取字段值
	fieldValue, err := ce.getFieldValue(condition.Field, context)
	if err != nil {
		return false, fmt.Errorf("获取字段值失败: %w", err)
	}

	// 根据操作符进行比较
	return ce.compareValues(fieldValue, condition.Operator, condition.Value)
}

// GetSupportedOperators 获取支持的操作符
func (ce *ConditionEvaluatorImpl) GetSupportedOperators() []string {
	return []string{
		"equals", "not_equals",
		"contains", "not_contains",
		"starts_with", "ends_with",
		"greater_than", "less_than",
		"greater_equal", "less_equal",
		"in", "not_in",
		"regex", "not_regex",
		"exists", "not_exists",
	}
}

// GetSupportedFields 获取支持的字段
func (ce *ConditionEvaluatorImpl) GetSupportedFields() []string {
	return []string{
		"packet_info.protocol",
		"packet_info.source_ip",
		"packet_info.dest_ip",
		"packet_info.source_port",
		"packet_info.dest_port",
		"packet_info.size",
		"parsed_data.protocol",
		"parsed_data.content_type",
		"parsed_data.url",
		"parsed_data.method",
		"analysis_result.risk_level",
		"analysis_result.risk_score",
		"analysis_result.confidence",
		"user_info.id",
		"user_info.role",
		"user_info.department",
		"device_info.type",
		"device_info.trust_level",
		"environment.location",
		"environment.working_hours",
	}
}

// getFieldValue 获取字段值
func (ce *ConditionEvaluatorImpl) getFieldValue(field string, context *DecisionContext) (interface{}, error) {
	parts := strings.Split(field, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("无效的字段路径: %s", field)
	}

	switch parts[0] {
	case "packet_info":
		if context.PacketInfo == nil {
			return nil, fmt.Errorf("数据包信息为空")
		}
		return ce.getPacketInfoField(parts[1], context.PacketInfo)

	case "parsed_data":
		if context.ParsedData == nil {
			return nil, fmt.Errorf("解析数据为空")
		}
		return ce.getParsedDataField(parts[1], context.ParsedData)

	case "analysis_result":
		if context.AnalysisResult == nil {
			return nil, fmt.Errorf("分析结果为空")
		}
		return ce.getAnalysisResultField(parts[1], context.AnalysisResult)

	case "user_info":
		if context.UserInfo == nil {
			return nil, fmt.Errorf("用户信息为空")
		}
		return ce.getUserInfoField(parts[1], context.UserInfo)

	case "device_info":
		if context.DeviceInfo == nil {
			return nil, fmt.Errorf("设备信息为空")
		}
		return ce.getDeviceInfoField(parts[1], context.DeviceInfo)

	case "environment":
		if context.Environment == nil {
			return nil, fmt.Errorf("环境信息为空")
		}
		return ce.getEnvironmentField(parts[1], context.Environment)

	default:
		return nil, fmt.Errorf("不支持的字段前缀: %s", parts[0])
	}
}

// getPacketInfoField 获取数据包信息字段
func (ce *ConditionEvaluatorImpl) getPacketInfoField(field string, packetInfo interface{}) (interface{}, error) {
	// 使用反射获取字段值
	v := reflect.ValueOf(packetInfo)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	fieldValue := v.FieldByName(strings.Title(field))
	if !fieldValue.IsValid() {
		return nil, fmt.Errorf("字段不存在: %s", field)
	}

	return fieldValue.Interface(), nil
}

// getParsedDataField 获取解析数据字段
func (ce *ConditionEvaluatorImpl) getParsedDataField(field string, parsedData interface{}) (interface{}, error) {
	v := reflect.ValueOf(parsedData)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	fieldValue := v.FieldByName(strings.Title(field))
	if !fieldValue.IsValid() {
		return nil, fmt.Errorf("字段不存在: %s", field)
	}

	return fieldValue.Interface(), nil
}

// getAnalysisResultField 获取分析结果字段
func (ce *ConditionEvaluatorImpl) getAnalysisResultField(field string, analysisResult interface{}) (interface{}, error) {
	v := reflect.ValueOf(analysisResult)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	switch field {
	case "risk_level":
		fieldValue := v.FieldByName("RiskLevel")
		if !fieldValue.IsValid() {
			return nil, fmt.Errorf("字段不存在: %s", field)
		}
		// 转换为字符串
		return fieldValue.String(), nil
	default:
		fieldValue := v.FieldByName(strings.Title(field))
		if !fieldValue.IsValid() {
			return nil, fmt.Errorf("字段不存在: %s", field)
		}
		return fieldValue.Interface(), nil
	}
}

// getUserInfoField 获取用户信息字段
func (ce *ConditionEvaluatorImpl) getUserInfoField(field string, userInfo *UserInfo) (interface{}, error) {
	switch field {
	case "id":
		return userInfo.ID, nil
	case "username":
		return userInfo.Username, nil
	case "role":
		return userInfo.Role, nil
	case "department":
		return userInfo.Department, nil
	default:
		return nil, fmt.Errorf("不支持的用户信息字段: %s", field)
	}
}

// getDeviceInfoField 获取设备信息字段
func (ce *ConditionEvaluatorImpl) getDeviceInfoField(field string, deviceInfo *DeviceInfo) (interface{}, error) {
	switch field {
	case "type":
		return deviceInfo.Type, nil
	case "trust_level":
		return deviceInfo.TrustLevel, nil
	case "compliance":
		return deviceInfo.Compliance, nil
	default:
		return nil, fmt.Errorf("不支持的设备信息字段: %s", field)
	}
}

// getEnvironmentField 获取环境信息字段
func (ce *ConditionEvaluatorImpl) getEnvironmentField(field string, environment *Environment) (interface{}, error) {
	switch field {
	case "location":
		return environment.Location, nil
	case "working_hours":
		return environment.WorkingHours, nil
	case "holiday":
		return environment.Holiday, nil
	default:
		return nil, fmt.Errorf("不支持的环境信息字段: %s", field)
	}
}

// compareValues 比较值
func (ce *ConditionEvaluatorImpl) compareValues(fieldValue interface{}, operator string, expectedValue interface{}) (bool, error) {
	switch operator {
	case "equals":
		return ce.equals(fieldValue, expectedValue), nil
	case "not_equals":
		return !ce.equals(fieldValue, expectedValue), nil
	case "contains":
		return ce.contains(fieldValue, expectedValue), nil
	case "not_contains":
		return !ce.contains(fieldValue, expectedValue), nil
	case "starts_with":
		return ce.startsWith(fieldValue, expectedValue), nil
	case "ends_with":
		return ce.endsWith(fieldValue, expectedValue), nil
	case "greater_than":
		return ce.greaterThan(fieldValue, expectedValue)
	case "less_than":
		return ce.lessThan(fieldValue, expectedValue)
	case "greater_equal":
		return ce.greaterEqual(fieldValue, expectedValue)
	case "less_equal":
		return ce.lessEqual(fieldValue, expectedValue)
	case "regex":
		return ce.regex(fieldValue, expectedValue)
	case "not_regex":
		matched, err := ce.regex(fieldValue, expectedValue)
		return !matched, err
	case "exists":
		return fieldValue != nil, nil
	case "not_exists":
		return fieldValue == nil, nil
	default:
		return false, fmt.Errorf("不支持的操作符: %s", operator)
	}
}

// equals 相等比较
func (ce *ConditionEvaluatorImpl) equals(a, b interface{}) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// contains 包含比较
func (ce *ConditionEvaluatorImpl) contains(fieldValue, expectedValue interface{}) bool {
	fieldStr := fmt.Sprintf("%v", fieldValue)
	expectedStr := fmt.Sprintf("%v", expectedValue)
	return strings.Contains(fieldStr, expectedStr)
}

// startsWith 开始于比较
func (ce *ConditionEvaluatorImpl) startsWith(fieldValue, expectedValue interface{}) bool {
	fieldStr := fmt.Sprintf("%v", fieldValue)
	expectedStr := fmt.Sprintf("%v", expectedValue)
	return strings.HasPrefix(fieldStr, expectedStr)
}

// endsWith 结束于比较
func (ce *ConditionEvaluatorImpl) endsWith(fieldValue, expectedValue interface{}) bool {
	fieldStr := fmt.Sprintf("%v", fieldValue)
	expectedStr := fmt.Sprintf("%v", expectedValue)
	return strings.HasSuffix(fieldStr, expectedStr)
}

// greaterThan 大于比较
func (ce *ConditionEvaluatorImpl) greaterThan(fieldValue, expectedValue interface{}) (bool, error) {
	fieldFloat, err := ce.toFloat64(fieldValue)
	if err != nil {
		return false, err
	}
	expectedFloat, err := ce.toFloat64(expectedValue)
	if err != nil {
		return false, err
	}
	return fieldFloat > expectedFloat, nil
}

// lessThan 小于比较
func (ce *ConditionEvaluatorImpl) lessThan(fieldValue, expectedValue interface{}) (bool, error) {
	fieldFloat, err := ce.toFloat64(fieldValue)
	if err != nil {
		return false, err
	}
	expectedFloat, err := ce.toFloat64(expectedValue)
	if err != nil {
		return false, err
	}
	return fieldFloat < expectedFloat, nil
}

// greaterEqual 大于等于比较
func (ce *ConditionEvaluatorImpl) greaterEqual(fieldValue, expectedValue interface{}) (bool, error) {
	fieldFloat, err := ce.toFloat64(fieldValue)
	if err != nil {
		return false, err
	}
	expectedFloat, err := ce.toFloat64(expectedValue)
	if err != nil {
		return false, err
	}
	return fieldFloat >= expectedFloat, nil
}

// lessEqual 小于等于比较
func (ce *ConditionEvaluatorImpl) lessEqual(fieldValue, expectedValue interface{}) (bool, error) {
	fieldFloat, err := ce.toFloat64(fieldValue)
	if err != nil {
		return false, err
	}
	expectedFloat, err := ce.toFloat64(expectedValue)
	if err != nil {
		return false, err
	}
	return fieldFloat <= expectedFloat, nil
}

// regex 正则表达式比较
func (ce *ConditionEvaluatorImpl) regex(fieldValue, expectedValue interface{}) (bool, error) {
	fieldStr := fmt.Sprintf("%v", fieldValue)
	pattern := fmt.Sprintf("%v", expectedValue)
	
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return false, fmt.Errorf("正则表达式编译失败: %w", err)
	}
	
	return regex.MatchString(fieldStr), nil
}

// toFloat64 转换为float64
func (ce *ConditionEvaluatorImpl) toFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("无法转换为float64: %T", value)
	}
}
