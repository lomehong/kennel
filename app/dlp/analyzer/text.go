package analyzer

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lomehong/kennel/app/dlp/parser"
	"github.com/lomehong/kennel/pkg/logging"
)

// TextAnalyzer 文本内容分析器
type TextAnalyzer struct {
	config       AnalyzerConfig
	logger       logging.Logger
	regexRules   []*RegexRule
	keywordRules []*KeywordRule
	stats        AnalyzerStats

	// OCR 支持
	ocrEnabled bool
	ocrEngine  OCREngine

	// 机器学习支持
	mlEnabled bool
	mlModel   TextMLModel

	// 文件类型检测
	fileDetector FileTypeDetector

	// 并发控制
	mu sync.RWMutex
}

// NewTextAnalyzer 创建文本分析器
func NewTextAnalyzer(logger logging.Logger) ContentAnalyzer {
	return &TextAnalyzer{
		logger:       logger,
		regexRules:   make([]*RegexRule, 0),
		keywordRules: make([]*KeywordRule, 0),
		stats: AnalyzerStats{
			StartTime: time.Now(),
		},
		ocrEnabled:   false, // 默认禁用OCR
		mlEnabled:    false, // 默认禁用ML
		ocrEngine:    NewTesseractOCR(logger),
		mlModel:      NewSimpleMLModel(logger),
		fileDetector: NewMimeTypeDetector(logger),
	}
}

// GetAnalyzerInfo 获取分析器信息
func (ta *TextAnalyzer) GetAnalyzerInfo() AnalyzerInfo {
	return AnalyzerInfo{
		Name:        "Text Analyzer",
		Version:     "1.0.0",
		Description: "文本内容敏感信息分析器",
		SupportedTypes: []string{
			"text/plain",
			"text/html",
			"application/json",
			"application/xml",
			"text/csv",
		},
		Author:       "DLP Team",
		License:      "MIT",
		Capabilities: []string{"regex", "keywords", "patterns"},
	}
}

// CanAnalyze 检查是否能分析指定类型的内容
func (ta *TextAnalyzer) CanAnalyze(contentType string) bool {
	supportedTypes := ta.GetSupportedTypes()
	for _, supportedType := range supportedTypes {
		if strings.Contains(contentType, supportedType) {
			return true
		}
	}
	return false
}

// Analyze 分析内容
func (ta *TextAnalyzer) Analyze(ctx context.Context, data *parser.ParsedData) (*AnalysisResult, error) {
	startTime := time.Now()
	atomic.AddUint64(&ta.stats.TotalAnalyzed, 1)

	// 检查内容大小限制
	if int64(len(data.Body)) > ta.config.MaxContentSize {
		atomic.AddUint64(&ta.stats.FailedAnalyzed, 1)
		return nil, fmt.Errorf("内容大小超过限制: %d > %d", len(data.Body), ta.config.MaxContentSize)
	}

	// 检查是否为加密内容
	isEncrypted := false
	if encrypted, ok := data.Metadata["encrypted"].(bool); ok && encrypted {
		isEncrypted = true
	}

	// 提取文本内容
	text := string(data.Body)
	if text == "" {
		// 尝试从其他字段提取文本
		text = ta.extractTextFromData(data)
	}

	// 如果仍然没有文本，尝试OCR提取
	if text == "" && ta.ocrEnabled {
		ocrText, err := ta.extractTextWithOCR(ctx, data)
		if err != nil {
			ta.logger.Warn("OCR文本提取失败", "error", err)
		} else {
			text = ocrText
		}
	}

	// 对于加密内容，采用特殊处理策略
	if isEncrypted && text == "" {
		ta.logger.Debug("检测到加密内容，使用元数据分析",
			"protocol", data.Protocol,
			"content_type", data.ContentType)

		// 从元数据中提取可分析的信息
		text = ta.extractTextFromData(data)

		// 如果仍然没有可分析的文本，创建一个基于元数据的分析结果
		if text == "" {
			return ta.createEncryptedContentResult(data), nil
		}
	}

	if text == "" {
		atomic.AddUint64(&ta.stats.FailedAnalyzed, 1)
		return nil, fmt.Errorf("无法提取文本内容")
	}

	// 创建分析结果
	result := &AnalysisResult{
		ID:              fmt.Sprintf("text_%d", time.Now().UnixNano()),
		Timestamp:       time.Now(),
		ContentType:     data.ContentType,
		SensitiveData:   make([]*SensitiveDataInfo, 0),
		RiskLevel:       RiskLevelLow,
		RiskScore:       0.0,
		Confidence:      1.0,
		Categories:      make([]string, 0),
		Tags:            make([]string, 0),
		Metadata:        make(map[string]interface{}),
		AnalyzerResults: make(map[string]interface{}),
	}

	// 执行正则表达式分析
	if ta.config.EnableRegexRules {
		regexResults := ta.analyzeWithRegex(text)
		result.SensitiveData = append(result.SensitiveData, regexResults...)
	}

	// 执行关键词分析
	if ta.config.EnableKeywords {
		keywordResults := ta.analyzeWithKeywords(text)
		result.SensitiveData = append(result.SensitiveData, keywordResults...)
	}

	// 执行机器学习分析
	if ta.mlEnabled {
		mlResults, err := ta.analyzeWithML(ctx, text)
		if err != nil {
			ta.logger.Warn("ML分析失败", "error", err)
		} else {
			result.AnalyzerResults["ml_prediction"] = mlResults
			// 如果ML预测为敏感，增加风险评分
			if mlResults.IsSensitive {
				result.RiskScore += mlResults.RiskScore * 0.3 // ML结果权重30%
			}
		}
	}

	// 计算风险评分和级别
	ta.calculateRiskScore(result)

	// 更新统计信息
	atomic.AddUint64(&ta.stats.SuccessfulAnalyzed, 1)
	if len(result.SensitiveData) > 0 {
		atomic.AddUint64(&ta.stats.SensitiveDetected, 1)
	}

	processingTime := time.Since(startTime)
	result.ProcessingTime = processingTime
	ta.updateAverageTime(processingTime)

	ta.logger.Debug("文本分析完成",
		"content_length", len(text),
		"sensitive_count", len(result.SensitiveData),
		"risk_level", result.RiskLevel.String(),
		"processing_time", processingTime)

	return result, nil
}

// GetSupportedTypes 获取支持的内容类型
func (ta *TextAnalyzer) GetSupportedTypes() []string {
	return ta.GetAnalyzerInfo().SupportedTypes
}

// Initialize 初始化分析器
func (ta *TextAnalyzer) Initialize(config AnalyzerConfig) error {
	ta.config = config
	ta.logger.Info("初始化文本分析器",
		"max_content_size", config.MaxContentSize,
		"enable_regex", config.EnableRegexRules,
		"enable_keywords", config.EnableKeywords)

	// 加载默认规则
	if err := ta.loadDefaultRules(); err != nil {
		return fmt.Errorf("加载默认规则失败: %w", err)
	}

	return nil
}

// Cleanup 清理资源
func (ta *TextAnalyzer) Cleanup() error {
	ta.logger.Info("清理文本分析器资源")
	ta.regexRules = nil
	ta.keywordRules = nil
	return nil
}

// UpdateRules 更新规则
func (ta *TextAnalyzer) UpdateRules(rules interface{}) error {
	switch r := rules.(type) {
	case []*RegexRule:
		ta.regexRules = r
		ta.logger.Info("更新正则表达式规则", "count", len(r))
	case []*KeywordRule:
		ta.keywordRules = r
		ta.logger.Info("更新关键词规则", "count", len(r))
	default:
		return fmt.Errorf("不支持的规则类型: %T", rules)
	}
	return nil
}

// GetStats 获取统计信息
func (ta *TextAnalyzer) GetStats() AnalyzerStats {
	stats := ta.stats
	stats.Uptime = time.Since(ta.stats.StartTime)
	return stats
}

// extractTextFromData 从数据中提取文本
func (ta *TextAnalyzer) extractTextFromData(data *parser.ParsedData) string {
	var text strings.Builder

	// 从URL中提取文本
	if data.URL != "" {
		text.WriteString(data.URL)
		text.WriteString(" ")
	}

	// 从头部中提取文本
	for key, value := range data.Headers {
		text.WriteString(fmt.Sprintf("%s: %s ", key, value))
	}

	// 从元数据中提取文本
	for key, value := range data.Metadata {
		if str, ok := value.(string); ok {
			text.WriteString(fmt.Sprintf("%s: %s ", key, str))
		}
	}

	return text.String()
}

// createEncryptedContentResult 为加密内容创建分析结果
func (ta *TextAnalyzer) createEncryptedContentResult(data *parser.ParsedData) *AnalysisResult {
	result := &AnalysisResult{
		ID:              fmt.Sprintf("encrypted_%d", time.Now().UnixNano()),
		Timestamp:       time.Now(),
		ContentType:     data.ContentType,
		SensitiveData:   make([]*SensitiveDataInfo, 0),
		RiskLevel:       RiskLevelLow,
		RiskScore:       0.0,
		Confidence:      0.5, // 加密内容的置信度较低
		Categories:      []string{"encrypted"},
		Tags:            []string{"encrypted_content"},
		Metadata:        make(map[string]interface{}),
		AnalyzerResults: make(map[string]interface{}),
		ProcessingTime:  time.Since(time.Now()),
	}

	// 添加加密内容的元数据
	result.Metadata["is_encrypted"] = true
	result.Metadata["protocol"] = data.Protocol
	result.Metadata["content_length"] = len(data.Body)

	// 从解析数据的元数据中提取有用信息
	for key, value := range data.Metadata {
		switch key {
		case "server_name", "host", "url", "method", "status_code":
			// 这些信息即使在加密情况下也可能有用
			result.Metadata[key] = value
		case "tls_version", "cipher_suites", "certificates":
			// TLS相关信息
			result.Metadata[key] = value
		}
	}

	// 基于协议和元数据进行基础风险评估
	if data.Protocol == "https" {
		// HTTPS流量通常是正常的
		result.RiskLevel = RiskLevelLow
		result.RiskScore = 0.1
	} else {
		// 其他加密协议可能需要更多关注
		result.RiskLevel = RiskLevelMedium
		result.RiskScore = 0.3
	}

	ta.logger.Debug("创建加密内容分析结果",
		"protocol", data.Protocol,
		"content_type", data.ContentType,
		"risk_level", result.RiskLevel)

	return result
}

// analyzeWithRegex 使用正则表达式分析
func (ta *TextAnalyzer) analyzeWithRegex(text string) []*SensitiveDataInfo {
	results := make([]*SensitiveDataInfo, 0)

	for _, rule := range ta.regexRules {
		if !rule.Enabled {
			continue
		}

		regex, err := regexp.Compile(rule.Pattern)
		if err != nil {
			ta.logger.Warn("正则表达式编译失败", "pattern", rule.Pattern, "error", err)
			continue
		}

		matches := regex.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) > 0 {
				value := match[0]
				if rule.Confidence >= ta.config.MinConfidence {
					sensitiveData := &SensitiveDataInfo{
						Type:        rule.Type,
						Value:       value,
						MaskedValue: ta.maskValue(value),
						Confidence:  rule.Confidence,
						Context:     ta.extractContext(text, value),
						Metadata: map[string]interface{}{
							"rule_id":   rule.ID,
							"rule_name": rule.Name,
							"category":  rule.Category,
						},
					}
					results = append(results, sensitiveData)
				}
			}
		}
	}

	return results
}

// analyzeWithKeywords 使用关键词分析
func (ta *TextAnalyzer) analyzeWithKeywords(text string) []*SensitiveDataInfo {
	results := make([]*SensitiveDataInfo, 0)

	for _, rule := range ta.keywordRules {
		if !rule.Enabled {
			continue
		}

		searchText := text
		if !rule.CaseSensitive {
			searchText = strings.ToLower(text)
		}

		for _, keyword := range rule.Keywords {
			searchKeyword := keyword
			if !rule.CaseSensitive {
				searchKeyword = strings.ToLower(keyword)
			}

			var found bool
			if rule.WholeWord {
				// 使用正则表达式进行全词匹配
				pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(searchKeyword))
				regex, err := regexp.Compile(pattern)
				if err == nil {
					found = regex.MatchString(searchText)
				}
			} else {
				found = strings.Contains(searchText, searchKeyword)
			}

			if found && rule.Confidence >= ta.config.MinConfidence {
				sensitiveData := &SensitiveDataInfo{
					Type:        rule.Type,
					Value:       keyword,
					MaskedValue: ta.maskValue(keyword),
					Confidence:  rule.Confidence,
					Context:     ta.extractContext(text, keyword),
					Metadata: map[string]interface{}{
						"rule_id":   rule.ID,
						"rule_name": rule.Name,
						"category":  rule.Category,
						"keyword":   keyword,
					},
				}
				results = append(results, sensitiveData)
			}
		}
	}

	return results
}

// calculateRiskScore 计算风险评分
func (ta *TextAnalyzer) calculateRiskScore(result *AnalysisResult) {
	if len(result.SensitiveData) == 0 {
		result.RiskLevel = RiskLevelLow
		result.RiskScore = 0.0
		return
	}

	var totalScore float64
	var maxConfidence float64
	categories := make(map[string]bool)

	for _, data := range result.SensitiveData {
		totalScore += data.Confidence
		if data.Confidence > maxConfidence {
			maxConfidence = data.Confidence
		}

		if category, ok := data.Metadata["category"].(string); ok {
			categories[category] = true
		}
	}

	// 计算平均分数
	avgScore := totalScore / float64(len(result.SensitiveData))

	// 考虑敏感数据数量的影响
	countFactor := float64(len(result.SensitiveData)) * 0.1
	if countFactor > 1.0 {
		countFactor = 1.0
	}

	// 最终风险评分
	result.RiskScore = (avgScore + countFactor) * maxConfidence
	if result.RiskScore > 1.0 {
		result.RiskScore = 1.0
	}

	// 确定风险级别
	switch {
	case result.RiskScore >= 0.8:
		result.RiskLevel = RiskLevelCritical
	case result.RiskScore >= 0.6:
		result.RiskLevel = RiskLevelHigh
	case result.RiskScore >= 0.4:
		result.RiskLevel = RiskLevelMedium
	default:
		result.RiskLevel = RiskLevelLow
	}

	// 设置分类
	for category := range categories {
		result.Categories = append(result.Categories, category)
	}
}

// maskValue 掩码敏感值
func (ta *TextAnalyzer) maskValue(value string) string {
	if len(value) <= 4 {
		return strings.Repeat("*", len(value))
	}

	// 保留前2位和后2位，中间用*替换
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}

// extractContext 提取上下文
func (ta *TextAnalyzer) extractContext(text, value string) string {
	index := strings.Index(text, value)
	if index == -1 {
		return ""
	}

	// 提取前后各50个字符作为上下文
	start := index - 50
	if start < 0 {
		start = 0
	}

	end := index + len(value) + 50
	if end > len(text) {
		end = len(text)
	}

	return text[start:end]
}

// updateAverageTime 更新平均处理时间
func (ta *TextAnalyzer) updateAverageTime(duration time.Duration) {
	// 简化的平均时间计算
	ta.stats.AverageTime = (ta.stats.AverageTime + duration) / 2
}

// loadDefaultRules 加载默认规则
func (ta *TextAnalyzer) loadDefaultRules() error {
	// 加载默认正则表达式规则
	ta.regexRules = []*RegexRule{
		{
			ID:          "phone_cn",
			Name:        "中国手机号",
			Description: "检测中国大陆手机号码",
			Pattern:     `1[3-9]\d{9}`,
			Type:        "phone",
			Category:    "pii",
			RiskLevel:   RiskLevelMedium,
			Confidence:  0.9,
			Enabled:     true,
		},
		{
			ID:          "id_card_cn",
			Name:        "中国身份证号",
			Description: "检测中国大陆身份证号码",
			Pattern:     `[1-9]\d{5}(18|19|20)\d{2}((0[1-9])|(1[0-2]))(([0-2][1-9])|10|20|30|31)\d{3}[0-9Xx]`,
			Type:        "id_card",
			Category:    "pii",
			RiskLevel:   RiskLevelHigh,
			Confidence:  0.95,
			Enabled:     true,
		},
		{
			ID:          "email",
			Name:        "电子邮箱",
			Description: "检测电子邮箱地址",
			Pattern:     `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`,
			Type:        "email",
			Category:    "contact",
			RiskLevel:   RiskLevelMedium,
			Confidence:  0.8,
			Enabled:     true,
		},
		{
			ID:          "credit_card",
			Name:        "信用卡号",
			Description: "检测信用卡号码",
			Pattern:     `\b(?:\d{4}[-\s]?){3}\d{4}\b`,
			Type:        "credit_card",
			Category:    "financial",
			RiskLevel:   RiskLevelHigh,
			Confidence:  0.85,
			Enabled:     true,
		},
	}

	// 加载默认关键词规则
	ta.keywordRules = []*KeywordRule{
		{
			ID:            "password_keywords",
			Name:          "密码关键词",
			Description:   "检测密码相关关键词",
			Keywords:      []string{"password", "passwd", "pwd", "密码", "口令"},
			Type:          "password",
			Category:      "credential",
			RiskLevel:     RiskLevelHigh,
			Confidence:    0.7,
			CaseSensitive: false,
			WholeWord:     true,
			Enabled:       true,
		},
		{
			ID:            "secret_keywords",
			Name:          "机密关键词",
			Description:   "检测机密相关关键词",
			Keywords:      []string{"secret", "confidential", "机密", "秘密", "内部"},
			Type:          "secret",
			Category:      "classification",
			RiskLevel:     RiskLevelMedium,
			Confidence:    0.6,
			CaseSensitive: false,
			WholeWord:     true,
			Enabled:       true,
		},
	}

	ta.logger.Info("加载默认规则",
		"regex_rules", len(ta.regexRules),
		"keyword_rules", len(ta.keywordRules))

	return nil
}

// extractTextWithOCR 使用OCR提取文本
func (ta *TextAnalyzer) extractTextWithOCR(ctx context.Context, data *parser.ParsedData) (string, error) {
	// 检测文件类型
	fileInfo, err := ta.fileDetector.DetectType(data.Body)
	if err != nil {
		return "", fmt.Errorf("文件类型检测失败: %w", err)
	}

	// 只对图像文件进行OCR
	if !fileInfo.IsImage {
		return "", fmt.Errorf("不是图像文件，无法进行OCR: %s", fileInfo.MimeType)
	}

	ta.logger.Debug("开始OCR文本提取",
		"mime_type", fileInfo.MimeType,
		"size", len(data.Body))

	// 使用OCR引擎提取文本
	text, err := ta.ocrEngine.ExtractTextFromBytes(ctx, data.Body)
	if err != nil {
		return "", fmt.Errorf("OCR提取失败: %w", err)
	}

	ta.logger.Debug("OCR文本提取完成",
		"extracted_length", len(text))

	return text, nil
}

// analyzeWithML 使用机器学习分析
func (ta *TextAnalyzer) analyzeWithML(ctx context.Context, text string) (*MLPrediction, error) {
	ta.logger.Debug("开始ML分析", "text_length", len(text))

	// 使用ML模型预测
	prediction, err := ta.mlModel.Predict(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("ML预测失败: %w", err)
	}

	ta.logger.Debug("ML分析完成",
		"is_sensitive", prediction.IsSensitive,
		"confidence", prediction.Confidence,
		"risk_score", prediction.RiskScore)

	return prediction, nil
}

// EnableOCR 启用OCR功能
func (ta *TextAnalyzer) EnableOCR(config map[string]interface{}) error {
	ta.mu.Lock()
	defer ta.mu.Unlock()

	if err := ta.ocrEngine.Initialize(config); err != nil {
		return fmt.Errorf("初始化OCR引擎失败: %w", err)
	}

	ta.ocrEnabled = true
	ta.logger.Info("OCR功能已启用")
	return nil
}

// DisableOCR 禁用OCR功能
func (ta *TextAnalyzer) DisableOCR() error {
	ta.mu.Lock()
	defer ta.mu.Unlock()

	if err := ta.ocrEngine.Cleanup(); err != nil {
		ta.logger.Warn("清理OCR引擎失败", "error", err)
	}

	ta.ocrEnabled = false
	ta.logger.Info("OCR功能已禁用")
	return nil
}

// EnableML 启用机器学习功能
func (ta *TextAnalyzer) EnableML(config map[string]interface{}) error {
	ta.mu.Lock()
	defer ta.mu.Unlock()

	if err := ta.mlModel.Initialize(config); err != nil {
		return fmt.Errorf("初始化ML模型失败: %w", err)
	}

	ta.mlEnabled = true
	ta.logger.Info("ML功能已启用")
	return nil
}

// DisableML 禁用机器学习功能
func (ta *TextAnalyzer) DisableML() error {
	ta.mu.Lock()
	defer ta.mu.Unlock()

	if err := ta.mlModel.Cleanup(); err != nil {
		ta.logger.Warn("清理ML模型失败", "error", err)
	}

	ta.mlEnabled = false
	ta.logger.Info("ML功能已禁用")
	return nil
}

// GetMLModelInfo 获取ML模型信息
func (ta *TextAnalyzer) GetMLModelInfo() *MLModelInfo {
	if ta.mlModel == nil {
		return nil
	}
	return ta.mlModel.GetModelInfo()
}

// GetOCRSupportedFormats 获取OCR支持的格式
func (ta *TextAnalyzer) GetOCRSupportedFormats() []string {
	if ta.ocrEngine == nil {
		return nil
	}
	return ta.ocrEngine.GetSupportedFormats()
}
