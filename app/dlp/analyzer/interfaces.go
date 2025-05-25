package analyzer

import (
	"context"
	"time"

	"github.com/lomehong/kennel/app/dlp/parser"
	"github.com/lomehong/kennel/pkg/logging"
)

// AnalysisResult 分析结果
type AnalysisResult struct {
	ID              string                 `json:"id"`
	Timestamp       time.Time              `json:"timestamp"`
	ContentType     string                 `json:"content_type"`
	SensitiveData   []*SensitiveDataInfo   `json:"sensitive_data"`
	RiskLevel       RiskLevel              `json:"risk_level"`
	RiskScore       float64                `json:"risk_score"`
	Confidence      float64                `json:"confidence"`
	Categories      []string               `json:"categories"`
	Tags            []string               `json:"tags"`
	Metadata        map[string]interface{} `json:"metadata"`
	ProcessingTime  time.Duration          `json:"processing_time"`
	AnalyzerResults map[string]interface{} `json:"analyzer_results"`
}

// SensitiveDataInfo 敏感数据信息
type SensitiveDataInfo struct {
	Type        string                 `json:"type"`
	Value       string                 `json:"value"`
	MaskedValue string                 `json:"masked_value"`
	Position    *Position              `json:"position"`
	Confidence  float64                `json:"confidence"`
	Context     string                 `json:"context"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Position 位置信息
type Position struct {
	Start  int `json:"start"`
	End    int `json:"end"`
	Line   int `json:"line"`
	Column int `json:"column"`
}

// RiskLevel 风险级别
type RiskLevel int

const (
	RiskLevelLow RiskLevel = iota
	RiskLevelMedium
	RiskLevelHigh
	RiskLevelCritical
)

// String 返回风险级别的字符串表示
func (r RiskLevel) String() string {
	switch r {
	case RiskLevelLow:
		return "low"
	case RiskLevelMedium:
		return "medium"
	case RiskLevelHigh:
		return "high"
	case RiskLevelCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// AnalyzerConfig 分析器配置
type AnalyzerConfig struct {
	MaxContentSize   int64             `yaml:"max_content_size" json:"max_content_size"`
	Timeout          time.Duration     `yaml:"timeout" json:"timeout"`
	EnableMLAnalysis bool              `yaml:"enable_ml_analysis" json:"enable_ml_analysis"`
	MLModelPath      string            `yaml:"ml_model_path" json:"ml_model_path"`
	EnableRegexRules bool              `yaml:"enable_regex_rules" json:"enable_regex_rules"`
	RegexRulesPath   string            `yaml:"regex_rules_path" json:"regex_rules_path"`
	EnableKeywords   bool              `yaml:"enable_keywords" json:"enable_keywords"`
	KeywordsPath     string            `yaml:"keywords_path" json:"keywords_path"`
	MinConfidence    float64           `yaml:"min_confidence" json:"min_confidence"`
	MaxConcurrency   int               `yaml:"max_concurrency" json:"max_concurrency"`
	CacheSize        int               `yaml:"cache_size" json:"cache_size"`
	CacheTTL         time.Duration     `yaml:"cache_ttl" json:"cache_ttl"`
	CustomRules      map[string]string `yaml:"custom_rules" json:"custom_rules"`
	Logger           logging.Logger    `yaml:"-" json:"-"`
}

// DefaultAnalyzerConfig 返回默认分析器配置
func DefaultAnalyzerConfig() AnalyzerConfig {
	return AnalyzerConfig{
		MaxContentSize:   50 * 1024 * 1024, // 50MB
		Timeout:          30 * time.Second,
		EnableMLAnalysis: true,
		EnableRegexRules: true,
		EnableKeywords:   true,
		MinConfidence:    0.7,
		MaxConcurrency:   10,
		CacheSize:        10000,
		CacheTTL:         1 * time.Hour,
		CustomRules:      make(map[string]string),
	}
}

// ContentAnalyzer 内容分析器接口
type ContentAnalyzer interface {
	// GetAnalyzerInfo 获取分析器信息
	GetAnalyzerInfo() AnalyzerInfo

	// CanAnalyze 检查是否能分析指定类型的内容
	CanAnalyze(contentType string) bool

	// Analyze 分析内容
	Analyze(ctx context.Context, data *parser.ParsedData) (*AnalysisResult, error)

	// GetSupportedTypes 获取支持的内容类型
	GetSupportedTypes() []string

	// Initialize 初始化分析器
	Initialize(config AnalyzerConfig) error

	// Cleanup 清理资源
	Cleanup() error

	// UpdateRules 更新规则
	UpdateRules(rules interface{}) error

	// GetStats 获取统计信息
	GetStats() AnalyzerStats
}

// AnalyzerInfo 分析器信息
type AnalyzerInfo struct {
	Name           string   `json:"name"`
	Version        string   `json:"version"`
	Description    string   `json:"description"`
	SupportedTypes []string `json:"supported_types"`
	Author         string   `json:"author"`
	License        string   `json:"license"`
	RequiredModels []string `json:"required_models"`
	Capabilities   []string `json:"capabilities"`
}

// AnalyzerStats 分析器统计信息
type AnalyzerStats struct {
	TotalAnalyzed      uint64        `json:"total_analyzed"`
	SuccessfulAnalyzed uint64        `json:"successful_analyzed"`
	FailedAnalyzed     uint64        `json:"failed_analyzed"`
	SensitiveDetected  uint64        `json:"sensitive_detected"`
	AverageTime        time.Duration `json:"average_time"`
	LastError          error         `json:"last_error,omitempty"`
	StartTime          time.Time     `json:"start_time"`
	Uptime             time.Duration `json:"uptime"`
}

// AnalysisManager 分析管理器接口
type AnalysisManager interface {
	// RegisterAnalyzer 注册分析器
	RegisterAnalyzer(analyzer ContentAnalyzer) error

	// GetAnalyzer 获取分析器
	GetAnalyzer(contentType string) (ContentAnalyzer, bool)

	// AnalyzeContent 分析内容
	AnalyzeContent(ctx context.Context, data *parser.ParsedData) (*AnalysisResult, error)

	// GetSupportedTypes 获取支持的内容类型
	GetSupportedTypes() []string

	// GetStats 获取统计信息
	GetStats() ManagerStats

	// Start 启动管理器
	Start() error

	// Stop 停止管理器
	Stop() error

	// UpdateRules 更新规则
	UpdateRules(analyzerName string, rules interface{}) error
}

// ManagerStats 管理器统计信息
type ManagerStats struct {
	TotalRequests     uint64                   `json:"total_requests"`
	ProcessedRequests uint64                   `json:"processed_requests"`
	FailedRequests    uint64                   `json:"failed_requests"`
	AverageTime       time.Duration            `json:"average_time"`
	AnalyzerStats     map[string]AnalyzerStats `json:"analyzer_stats"`
	LastError         error                    `json:"last_error,omitempty"`
	StartTime         time.Time                `json:"start_time"`
	Uptime            time.Duration            `json:"uptime"`
}

// RegexRule 正则表达式规则
type RegexRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Pattern     string                 `json:"pattern"`
	Type        string                 `json:"type"`
	Category    string                 `json:"category"`
	RiskLevel   RiskLevel              `json:"risk_level"`
	Confidence  float64                `json:"confidence"`
	Enabled     bool                   `json:"enabled"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// KeywordRule 关键词规则
type KeywordRule struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Keywords      []string               `json:"keywords"`
	Type          string                 `json:"type"`
	Category      string                 `json:"category"`
	RiskLevel     RiskLevel              `json:"risk_level"`
	Confidence    float64                `json:"confidence"`
	CaseSensitive bool                   `json:"case_sensitive"`
	WholeWord     bool                   `json:"whole_word"`
	Enabled       bool                   `json:"enabled"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// MLModel 机器学习模型接口
type MLModel interface {
	// LoadModel 加载模型
	LoadModel(modelPath string) error

	// Predict 预测
	Predict(input []float32) ([]float32, error)

	// GetModelInfo 获取模型信息
	GetModelInfo() ModelInfo

	// IsLoaded 检查模型是否已加载
	IsLoaded() bool

	// Unload 卸载模型
	Unload() error
}

// ModelInfo 模型信息
type ModelInfo struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Type        string   `json:"type"`
	InputShape  []int    `json:"input_shape"`
	OutputShape []int    `json:"output_shape"`
	Labels      []string `json:"labels"`
	Accuracy    float64  `json:"accuracy"`
}

// TextProcessor 文本处理器接口
type TextProcessor interface {
	// ExtractText 提取文本
	ExtractText(data []byte, contentType string) (string, error)

	// TokenizeText 分词
	TokenizeText(text string) ([]string, error)

	// NormalizeText 文本标准化
	NormalizeText(text string) string

	// DetectLanguage 检测语言
	DetectLanguage(text string) string

	// GetSupportedFormats 获取支持的格式
	GetSupportedFormats() []string
}

// FileAnalyzer 文件分析器接口
type FileAnalyzer interface {
	ContentAnalyzer

	// AnalyzeFile 分析文件
	AnalyzeFile(ctx context.Context, filePath string) (*AnalysisResult, error)

	// GetFileInfo 获取文件信息
	GetFileInfo(filePath string) (*FileInfo, error)

	// ExtractMetadata 提取元数据
	ExtractMetadata(filePath string) (map[string]interface{}, error)
}

// FileInfo 文件信息
type FileInfo struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	ModTime     time.Time `json:"mod_time"`
	Extension   string    `json:"extension"`
	MimeType    string    `json:"mime_type"`
	Hash        string    `json:"hash"`
	Permissions string    `json:"permissions"`
}

// ImageAnalyzer 图像分析器接口
type ImageAnalyzer interface {
	ContentAnalyzer

	// AnalyzeImage 分析图像
	AnalyzeImage(ctx context.Context, imageData []byte) (*AnalysisResult, error)

	// ExtractText 从图像中提取文本(OCR)
	ExtractText(imageData []byte) (string, error)

	// DetectObjects 检测对象
	DetectObjects(imageData []byte) ([]ObjectInfo, error)

	// GetImageInfo 获取图像信息
	GetImageInfo(imageData []byte) (*ImageInfo, error)
}

// ObjectInfo 对象信息
type ObjectInfo struct {
	Label       string       `json:"label"`
	Confidence  float64      `json:"confidence"`
	BoundingBox *BoundingBox `json:"bounding_box"`
}

// BoundingBox 边界框
type BoundingBox struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// ImageInfo 图像信息
type ImageInfo struct {
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Format  string `json:"format"`
	Size    int64  `json:"size"`
	Quality int    `json:"quality"`
}

// CacheManager 缓存管理器接口
type CacheManager interface {
	// Get 获取缓存
	Get(key string) (interface{}, bool)

	// Set 设置缓存
	Set(key string, value interface{}, ttl time.Duration)

	// Delete 删除缓存
	Delete(key string)

	// Clear 清空缓存
	Clear()

	// Size 获取缓存大小
	Size() int

	// Stats 获取缓存统计
	Stats() CacheStats
}

// CacheStats 缓存统计
type CacheStats struct {
	Hits      uint64  `json:"hits"`
	Misses    uint64  `json:"misses"`
	Size      int     `json:"size"`
	MaxSize   int     `json:"max_size"`
	Evictions uint64  `json:"evictions"`
	HitRate   float64 `json:"hit_rate"`
}
