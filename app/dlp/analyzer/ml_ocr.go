package analyzer

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lomehong/kennel/pkg/logging"
)

// OCREngine OCR引擎接口
type OCREngine interface {
	// ExtractText 从图像中提取文本
	ExtractText(ctx context.Context, img image.Image) (string, error)

	// ExtractTextFromBytes 从字节数据中提取文本
	ExtractTextFromBytes(ctx context.Context, data []byte) (string, error)

	// GetSupportedFormats 获取支持的图像格式
	GetSupportedFormats() []string

	// Initialize 初始化OCR引擎
	Initialize(config map[string]interface{}) error

	// Cleanup 清理资源
	Cleanup() error
}

// TextMLModel 文本机器学习模型接口
type TextMLModel interface {
	// Predict 预测内容的敏感性
	Predict(ctx context.Context, text string) (*MLPrediction, error)

	// BatchPredict 批量预测
	BatchPredict(ctx context.Context, texts []string) ([]*MLPrediction, error)

	// GetModelInfo 获取模型信息
	GetModelInfo() *MLModelInfo

	// Initialize 初始化模型
	Initialize(config map[string]interface{}) error

	// Cleanup 清理资源
	Cleanup() error
}

// FileTypeDetector 文件类型检测器接口
type FileTypeDetector interface {
	// DetectType 检测文件类型
	DetectType(data []byte) (*FileTypeInfo, error)

	// DetectFromReader 从Reader检测文件类型
	DetectFromReader(reader io.Reader) (*FileTypeInfo, error)

	// IsImage 检查是否是图像文件
	IsImage(mimeType string) bool

	// IsDocument 检查是否是文档文件
	IsDocument(mimeType string) bool

	// GetSupportedTypes 获取支持的文件类型
	GetSupportedTypes() []string
}

// MLPrediction 机器学习预测结果
type MLPrediction struct {
	// IsSensitive 是否敏感
	IsSensitive bool

	// Confidence 置信度 (0.0-1.0)
	Confidence float64

	// Categories 预测的分类
	Categories []string

	// RiskScore 风险评分 (0.0-1.0)
	RiskScore float64

	// Explanation 预测解释
	Explanation string

	// Metadata 元数据
	Metadata map[string]interface{}
}

// MLModelInfo 机器学习模型信息
type MLModelInfo struct {
	// Name 模型名称
	Name string

	// Version 模型版本
	Version string

	// Description 模型描述
	Description string

	// SupportedLanguages 支持的语言
	SupportedLanguages []string

	// Categories 支持的分类
	Categories []string

	// Accuracy 模型准确率
	Accuracy float64

	// TrainingDate 训练日期
	TrainingDate string
}

// FileTypeInfo 文件类型信息
type FileTypeInfo struct {
	// MimeType MIME类型
	MimeType string

	// Extension 文件扩展名
	Extension string

	// Description 描述
	Description string

	// IsImage 是否是图像
	IsImage bool

	// IsDocument 是否是文档
	IsDocument bool

	// IsArchive 是否是压缩文件
	IsArchive bool

	// Confidence 检测置信度
	Confidence float64
}

// TesseractOCR Tesseract OCR引擎实现
type TesseractOCR struct {
	config        map[string]interface{}
	logger        logging.Logger
	mutex         sync.RWMutex
	initialized   bool
	tesseractPath string
	languages     []string
	timeout       time.Duration
	maxImageSize  int64
	enablePreproc bool
}

// NewTesseractOCR 创建Tesseract OCR引擎
func NewTesseractOCR(logger logging.Logger) OCREngine {
	return &TesseractOCR{
		logger:        logger,
		config:        make(map[string]interface{}),
		languages:     []string{"eng", "chi_sim"}, // 默认支持英文和简体中文
		timeout:       30 * time.Second,           // 默认30秒超时
		maxImageSize:  10 * 1024 * 1024,           // 默认10MB最大图像大小
		enablePreproc: true,                       // 默认启用预处理
	}
}

// ExtractText 从图像中提取文本
func (t *TesseractOCR) ExtractText(ctx context.Context, img image.Image) (string, error) {
	t.mutex.RLock()
	if !t.initialized {
		t.mutex.RUnlock()
		return "", fmt.Errorf("OCR引擎未初始化")
	}
	t.mutex.RUnlock()

	t.logger.Debug("开始使用Tesseract提取图像文本")

	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	// 预处理图像
	processedImg := img
	if t.enablePreproc {
		var err error
		processedImg, err = t.preprocessImage(img)
		if err != nil {
			t.logger.Warn("图像预处理失败，使用原图", "error", err)
			processedImg = img
		}
	}

	// 将图像转换为字节数组
	imgBytes, err := t.imageToBytes(processedImg)
	if err != nil {
		return "", fmt.Errorf("图像转换失败: %w", err)
	}

	// 检查图像大小
	if int64(len(imgBytes)) > t.maxImageSize {
		return "", fmt.Errorf("图像大小超过限制: %d bytes > %d bytes", len(imgBytes), t.maxImageSize)
	}

	// 使用goroutine执行OCR，支持超时控制
	resultChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errorChan <- fmt.Errorf("OCR处理发生panic: %v", r)
			}
		}()

		// 尝试执行真实的OCR
		text, err := t.performOCR(imgBytes)
		if err != nil {
			errorChan <- err
			return
		}

		resultChan <- strings.TrimSpace(text)
	}()

	// 等待结果或超时
	select {
	case <-timeoutCtx.Done():
		t.logger.Warn("OCR处理超时", "timeout", t.timeout)
		return "", fmt.Errorf("OCR处理超时")
	case err := <-errorChan:
		t.logger.Error("OCR处理失败", "error", err)
		return "", err
	case text := <-resultChan:
		t.logger.Debug("OCR文本提取完成", "text_length", len(text))
		return text, nil
	}
}

// ExtractTextFromBytes 从字节数据中提取文本
func (t *TesseractOCR) ExtractTextFromBytes(ctx context.Context, data []byte) (string, error) {
	// 解码图像
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("解码图像失败: %w", err)
	}

	return t.ExtractText(ctx, img)
}

// GetSupportedFormats 获取支持的图像格式
func (t *TesseractOCR) GetSupportedFormats() []string {
	return []string{"image/jpeg", "image/png", "image/tiff", "image/bmp"}
}

// preprocessImage 预处理图像以提高OCR准确率
func (t *TesseractOCR) preprocessImage(img image.Image) (image.Image, error) {
	bounds := img.Bounds()

	// 创建灰度图像
	grayImg := image.NewGray(bounds)

	// 转换为灰度
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			originalColor := img.At(x, y)
			grayColor := color.GrayModel.Convert(originalColor)
			grayImg.Set(x, y, grayColor)
		}
	}

	// 简单的二值化处理
	threshold := uint8(128)
	binaryImg := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			grayColor := grayImg.GrayAt(x, y)
			if grayColor.Y > threshold {
				binaryImg.Set(x, y, color.Gray{Y: 255})
			} else {
				binaryImg.Set(x, y, color.Gray{Y: 0})
			}
		}
	}

	return binaryImg, nil
}

// imageToBytes 将图像转换为字节数组
func (t *TesseractOCR) imageToBytes(img image.Image) ([]byte, error) {
	var buf bytes.Buffer

	// 尝试PNG格式编码
	err := png.Encode(&buf, img)
	if err != nil {
		// 如果PNG失败，尝试JPEG格式
		buf.Reset()
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
		if err != nil {
			return nil, fmt.Errorf("图像编码失败: %w", err)
		}
	}

	return buf.Bytes(), nil
}

// performOCR 执行OCR处理
func (t *TesseractOCR) performOCR(imgBytes []byte) (string, error) {
	// 尝试使用Tesseract库
	if t.isTesseractLibAvailable() {
		return t.performTesseractOCRWithLib(imgBytes)
	}

	// 如果Tesseract库不可用，返回错误
	t.logger.Warn("Tesseract库不可用，OCR功能降级")
	return "", fmt.Errorf("Tesseract OCR不可用，请安装Tesseract或使用 'go build -tags tesseract' 编译")
}

// Initialize 初始化OCR引擎
func (t *TesseractOCR) Initialize(config map[string]interface{}) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.config = config
	t.logger.Info("初始化Tesseract OCR引擎")

	// 从配置中读取参数
	if languages, ok := config["languages"].([]string); ok && len(languages) > 0 {
		t.languages = languages
	}

	if timeout, ok := config["timeout"].(time.Duration); ok {
		t.timeout = timeout
	} else if timeoutSec, ok := config["timeout_seconds"].(int); ok {
		t.timeout = time.Duration(timeoutSec) * time.Second
	}

	if maxSize, ok := config["max_image_size"].(int64); ok {
		t.maxImageSize = maxSize
	}

	if enablePreproc, ok := config["enable_preprocessing"].(bool); ok {
		t.enablePreproc = enablePreproc
	}

	if tesseractPath, ok := config["tesseract_path"].(string); ok {
		t.tesseractPath = tesseractPath
		// 设置Tesseract路径
		if t.tesseractPath != "" {
			os.Setenv("TESSDATA_PREFIX", filepath.Dir(t.tesseractPath))
		}
	}

	// 测试Tesseract是否可用
	err := t.testTesseractAvailability()
	if err != nil {
		t.logger.Error("Tesseract不可用", "error", err)
		return fmt.Errorf("Tesseract不可用: %w", err)
	}

	t.initialized = true
	t.logger.Info("Tesseract OCR引擎初始化成功",
		"languages", t.languages,
		"timeout", t.timeout,
		"max_image_size", t.maxImageSize,
		"enable_preprocessing", t.enablePreproc)

	return nil
}

// testTesseractAvailability 测试Tesseract是否可用
func (t *TesseractOCR) testTesseractAvailability() error {
	// 创建一个简单的测试图像
	testImg := image.NewRGBA(image.Rect(0, 0, 100, 50))

	// 填充白色背景
	for y := 0; y < 50; y++ {
		for x := 0; x < 100; x++ {
			testImg.Set(x, y, color.RGBA{255, 255, 255, 255})
		}
	}

	// 转换测试图像为字节
	imgBytes, err := t.imageToBytes(testImg)
	if err != nil {
		return fmt.Errorf("测试图像转换失败: %w", err)
	}

	// 尝试执行OCR测试
	_, err = t.performTesseractOCRWithLib(imgBytes)
	if err != nil {
		return fmt.Errorf("Tesseract测试失败: %w", err)
	}

	return nil
}

// Cleanup 清理资源
func (t *TesseractOCR) Cleanup() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.logger.Info("清理Tesseract OCR资源")
	t.initialized = false
	return nil
}

// SimpleMLModel 简单机器学习模型实现
type SimpleMLModel struct {
	config map[string]interface{}
	logger logging.Logger
	info   *MLModelInfo
}

// NewSimpleMLModel 创建简单ML模型
func NewSimpleMLModel(logger logging.Logger) TextMLModel {
	return &SimpleMLModel{
		logger: logger,
		config: make(map[string]interface{}),
		info: &MLModelInfo{
			Name:               "Simple Text Classifier",
			Version:            "1.0.0",
			Description:        "基于规则的简单文本分类器",
			SupportedLanguages: []string{"zh", "en"},
			Categories:         []string{"pii", "financial", "credential", "classification"},
			Accuracy:           0.85,
			TrainingDate:       "2024-01-01",
		},
	}
}

// Predict 预测内容的敏感性
func (s *SimpleMLModel) Predict(ctx context.Context, text string) (*MLPrediction, error) {
	s.logger.Debug("使用ML模型预测文本敏感性", "text_length", len(text))

	// 简化的基于规则的预测
	prediction := &MLPrediction{
		IsSensitive: false,
		Confidence:  0.5,
		Categories:  make([]string, 0),
		RiskScore:   0.0,
		Metadata:    make(map[string]interface{}),
	}

	// 检查敏感关键词
	sensitiveKeywords := []string{
		"password", "密码", "身份证", "信用卡", "银行卡",
		"secret", "confidential", "机密", "内部",
	}

	textLower := strings.ToLower(text)
	for _, keyword := range sensitiveKeywords {
		if strings.Contains(textLower, strings.ToLower(keyword)) {
			prediction.IsSensitive = true
			prediction.Confidence += 0.2
			prediction.RiskScore += 0.3

			// 分类
			switch keyword {
			case "password", "密码":
				prediction.Categories = append(prediction.Categories, "credential")
			case "身份证", "信用卡", "银行卡":
				prediction.Categories = append(prediction.Categories, "pii")
			case "secret", "confidential", "机密", "内部":
				prediction.Categories = append(prediction.Categories, "classification")
			}
		}
	}

	// 限制置信度和风险评分范围
	if prediction.Confidence > 1.0 {
		prediction.Confidence = 1.0
	}
	if prediction.RiskScore > 1.0 {
		prediction.RiskScore = 1.0
	}

	prediction.Explanation = fmt.Sprintf("基于关键词匹配的预测结果，文本长度: %d", len(text))

	return prediction, nil
}

// BatchPredict 批量预测
func (s *SimpleMLModel) BatchPredict(ctx context.Context, texts []string) ([]*MLPrediction, error) {
	results := make([]*MLPrediction, len(texts))

	for i, text := range texts {
		prediction, err := s.Predict(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("批量预测失败，索引 %d: %w", i, err)
		}
		results[i] = prediction
	}

	return results, nil
}

// GetModelInfo 获取模型信息
func (s *SimpleMLModel) GetModelInfo() *MLModelInfo {
	return s.info
}

// Initialize 初始化模型
func (s *SimpleMLModel) Initialize(config map[string]interface{}) error {
	s.config = config
	s.logger.Info("初始化简单ML模型")
	return nil
}

// Cleanup 清理资源
func (s *SimpleMLModel) Cleanup() error {
	s.logger.Info("清理简单ML模型资源")
	return nil
}

// MimeTypeDetector MIME类型检测器实现
type MimeTypeDetector struct {
	logger logging.Logger
}

// NewMimeTypeDetector 创建MIME类型检测器
func NewMimeTypeDetector(logger logging.Logger) FileTypeDetector {
	return &MimeTypeDetector{
		logger: logger,
	}
}

// DetectType 检测文件类型
func (m *MimeTypeDetector) DetectType(data []byte) (*FileTypeInfo, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("数据为空")
	}

	// 使用Go标准库检测MIME类型
	mimeType := http.DetectContentType(data)

	// 创建文件类型信息
	info := &FileTypeInfo{
		MimeType:    mimeType,
		Extension:   m.getExtensionFromMime(mimeType),
		Description: m.getDescriptionFromMime(mimeType),
		IsImage:     m.IsImage(mimeType),
		IsDocument:  m.IsDocument(mimeType),
		IsArchive:   m.isArchive(mimeType),
		Confidence:  0.9, // 标准库检测的置信度较高
	}

	m.logger.Debug("检测到文件类型",
		"mime_type", mimeType,
		"extension", info.Extension,
		"is_image", info.IsImage)

	return info, nil
}

// DetectFromReader 从Reader检测文件类型
func (m *MimeTypeDetector) DetectFromReader(reader io.Reader) (*FileTypeInfo, error) {
	// 读取前512字节用于类型检测
	buffer := make([]byte, 512)
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("读取数据失败: %w", err)
	}

	return m.DetectType(buffer[:n])
}

// IsImage 检查是否是图像文件
func (m *MimeTypeDetector) IsImage(mimeType string) bool {
	imageTypes := []string{
		"image/jpeg", "image/jpg", "image/png", "image/gif",
		"image/bmp", "image/tiff", "image/webp", "image/svg+xml",
	}

	for _, imageType := range imageTypes {
		if strings.HasPrefix(mimeType, imageType) {
			return true
		}
	}

	return false
}

// IsDocument 检查是否是文档文件
func (m *MimeTypeDetector) IsDocument(mimeType string) bool {
	documentTypes := []string{
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.ms-powerpoint",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"text/plain", "text/csv", "text/html", "text/xml",
		"application/json", "application/xml",
	}

	for _, docType := range documentTypes {
		if strings.HasPrefix(mimeType, docType) {
			return true
		}
	}

	return false
}

// isArchive 检查是否是压缩文件
func (m *MimeTypeDetector) isArchive(mimeType string) bool {
	archiveTypes := []string{
		"application/zip",
		"application/x-rar-compressed",
		"application/x-tar",
		"application/gzip",
		"application/x-7z-compressed",
	}

	for _, archiveType := range archiveTypes {
		if strings.HasPrefix(mimeType, archiveType) {
			return true
		}
	}

	return false
}

// GetSupportedTypes 获取支持的文件类型
func (m *MimeTypeDetector) GetSupportedTypes() []string {
	return []string{
		// 图像类型
		"image/jpeg", "image/png", "image/gif", "image/bmp", "image/tiff", "image/webp",
		// 文档类型
		"application/pdf", "application/msword", "text/plain", "text/csv", "text/html",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		// 压缩文件
		"application/zip", "application/x-rar-compressed", "application/x-tar",
	}
}

// getExtensionFromMime 从MIME类型获取文件扩展名
func (m *MimeTypeDetector) getExtensionFromMime(mimeType string) string {
	mimeToExt := map[string]string{
		"image/jpeg":         ".jpg",
		"image/png":          ".png",
		"image/gif":          ".gif",
		"image/bmp":          ".bmp",
		"image/tiff":         ".tiff",
		"image/webp":         ".webp",
		"application/pdf":    ".pdf",
		"application/msword": ".doc",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": ".docx",
		"application/vnd.ms-excel": ".xls",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": ".xlsx",
		"text/plain":                   ".txt",
		"text/csv":                     ".csv",
		"text/html":                    ".html",
		"application/json":             ".json",
		"application/xml":              ".xml",
		"application/zip":              ".zip",
		"application/x-rar-compressed": ".rar",
		"application/x-tar":            ".tar",
	}

	if ext, exists := mimeToExt[mimeType]; exists {
		return ext
	}

	return ""
}

// getDescriptionFromMime 从MIME类型获取描述
func (m *MimeTypeDetector) getDescriptionFromMime(mimeType string) string {
	mimeToDesc := map[string]string{
		"image/jpeg":         "JPEG图像",
		"image/png":          "PNG图像",
		"image/gif":          "GIF图像",
		"image/bmp":          "BMP图像",
		"application/pdf":    "PDF文档",
		"application/msword": "Word文档",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": "Word文档(新版)",
		"text/plain":       "纯文本文件",
		"text/csv":         "CSV文件",
		"text/html":        "HTML文件",
		"application/json": "JSON文件",
		"application/zip":  "ZIP压缩文件",
	}

	if desc, exists := mimeToDesc[mimeType]; exists {
		return desc
	}

	return "未知文件类型"
}
