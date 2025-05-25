package analyzer

import (
	"context"
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/lomehong/kennel/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTesseractOCR_Initialize(t *testing.T) {
	logger := logging.NewEnhancedLogger("test", "info")
	ocr := NewTesseractOCR(logger)

	config := map[string]interface{}{
		"languages":             []string{"eng"},
		"timeout_seconds":       30,
		"max_image_size":        int64(10 * 1024 * 1024),
		"enable_preprocessing":  true,
	}

	err := ocr.Initialize(config)
	// 注意：这个测试可能会失败，如果Tesseract没有安装
	// 这是预期的行为
	if err != nil {
		t.Logf("OCR初始化失败（这是正常的，如果Tesseract未安装）: %v", err)
		return
	}

	defer ocr.Cleanup()

	// 测试支持的格式
	formats := ocr.GetSupportedFormats()
	assert.NotEmpty(t, formats)
	assert.Contains(t, formats, "image/jpeg")
	assert.Contains(t, formats, "image/png")
}

func TestTesseractOCR_ExtractText_WithoutTesseract(t *testing.T) {
	logger := logging.NewEnhancedLogger("test", "info")
	ocr := NewTesseractOCR(logger)

	// 不初始化，直接测试
	testImg := createSimpleTestImage()
	ctx := context.Background()

	_, err := ocr.ExtractText(ctx, testImg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "OCR引擎未初始化")
}

func TestTesseractOCR_ExtractTextFromBytes(t *testing.T) {
	logger := logging.NewEnhancedLogger("test", "info")
	ocr := NewTesseractOCR(logger)

	config := map[string]interface{}{
		"languages":             []string{"eng"},
		"timeout_seconds":       5,
		"max_image_size":        int64(1024 * 1024),
		"enable_preprocessing":  true,
	}

	err := ocr.Initialize(config)
	if err != nil {
		t.Skipf("跳过测试，Tesseract未安装: %v", err)
		return
	}
	defer ocr.Cleanup()

	// 创建测试图像字节
	testImg := createSimpleTestImage()
	tesseractOCR := ocr.(*TesseractOCR)
	imgBytes, err := tesseractOCR.imageToBytes(testImg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = ocr.ExtractTextFromBytes(ctx, imgBytes)
	// 可能会失败，但不应该panic
	if err != nil {
		t.Logf("OCR处理失败（可能是正常的）: %v", err)
	}
}

func TestTesseractOCR_ImagePreprocessing(t *testing.T) {
	logger := logging.NewEnhancedLogger("test", "info")
	ocr := NewTesseractOCR(logger).(*TesseractOCR)

	testImg := createSimpleTestImage()
	processedImg, err := ocr.preprocessImage(testImg)
	
	assert.NoError(t, err)
	assert.NotNil(t, processedImg)
	
	// 检查处理后的图像尺寸
	bounds := processedImg.Bounds()
	originalBounds := testImg.Bounds()
	assert.Equal(t, originalBounds, bounds)
}

func TestTesseractOCR_ImageToBytes(t *testing.T) {
	logger := logging.NewEnhancedLogger("test", "info")
	ocr := NewTesseractOCR(logger).(*TesseractOCR)

	testImg := createSimpleTestImage()
	imgBytes, err := ocr.imageToBytes(testImg)
	
	assert.NoError(t, err)
	assert.NotEmpty(t, imgBytes)
	assert.Greater(t, len(imgBytes), 0)
}

func TestTesseractOCR_Configuration(t *testing.T) {
	logger := logging.NewEnhancedLogger("test", "info")
	ocr := NewTesseractOCR(logger).(*TesseractOCR)

	// 测试默认配置
	assert.Equal(t, []string{"eng", "chi_sim"}, ocr.languages)
	assert.Equal(t, 30*time.Second, ocr.timeout)
	assert.Equal(t, int64(10*1024*1024), ocr.maxImageSize)
	assert.True(t, ocr.enablePreproc)

	// 测试自定义配置
	config := map[string]interface{}{
		"languages":             []string{"eng"},
		"timeout_seconds":       60,
		"max_image_size":        int64(5 * 1024 * 1024),
		"enable_preprocessing":  false,
	}

	err := ocr.Initialize(config)
	if err != nil {
		t.Logf("初始化失败（Tesseract未安装）: %v", err)
		return
	}

	assert.Equal(t, []string{"eng"}, ocr.languages)
	assert.Equal(t, 60*time.Second, ocr.timeout)
	assert.Equal(t, int64(5*1024*1024), ocr.maxImageSize)
	assert.False(t, ocr.enablePreproc)
}

func TestSimpleMLModel(t *testing.T) {
	logger := logging.NewEnhancedLogger("test", "info")
	model := NewSimpleMLModel(logger)

	err := model.Initialize(map[string]interface{}{})
	assert.NoError(t, err)
	defer model.Cleanup()

	// 测试模型信息
	info := model.GetModelInfo()
	assert.NotNil(t, info)
	assert.Equal(t, "Simple Text Classifier", info.Name)
	assert.Equal(t, "1.0.0", info.Version)

	// 测试预测
	ctx := context.Background()
	
	// 测试敏感文本
	prediction, err := model.Predict(ctx, "这是我的密码：123456")
	assert.NoError(t, err)
	assert.True(t, prediction.IsSensitive)
	assert.Greater(t, prediction.Confidence, 0.5)
	assert.Contains(t, prediction.Categories, "credential")

	// 测试普通文本
	prediction, err = model.Predict(ctx, "今天天气很好")
	assert.NoError(t, err)
	assert.False(t, prediction.IsSensitive)

	// 测试批量预测
	texts := []string{"password: secret", "hello world", "身份证号码：123456"}
	predictions, err := model.BatchPredict(ctx, texts)
	assert.NoError(t, err)
	assert.Len(t, predictions, 3)
	assert.True(t, predictions[0].IsSensitive)
	assert.False(t, predictions[1].IsSensitive)
	assert.True(t, predictions[2].IsSensitive)
}

func TestMimeTypeDetector(t *testing.T) {
	logger := logging.NewEnhancedLogger("test", "info")
	detector := NewMimeTypeDetector(logger)

	// 测试PNG图像检测
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	info, err := detector.DetectType(pngHeader)
	assert.NoError(t, err)
	assert.True(t, info.IsImage)
	assert.Contains(t, info.MimeType, "image")

	// 测试JPEG图像检测
	jpegHeader := []byte{0xFF, 0xD8, 0xFF}
	info, err = detector.DetectType(jpegHeader)
	assert.NoError(t, err)
	assert.True(t, info.IsImage)

	// 测试文本文件检测
	textData := []byte("Hello, World!")
	info, err = detector.DetectType(textData)
	assert.NoError(t, err)
	assert.True(t, detector.IsDocument(info.MimeType))

	// 测试支持的类型
	types := detector.GetSupportedTypes()
	assert.NotEmpty(t, types)
	assert.Contains(t, types, "image/jpeg")
	assert.Contains(t, types, "image/png")
	assert.Contains(t, types, "application/pdf")
}

// createSimpleTestImage 创建简单的测试图像
func createSimpleTestImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 100, 50))
	
	// 填充白色背景
	for y := 0; y < 50; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{255, 255, 255, 255})
		}
	}
	
	// 添加一些黑色像素
	for x := 10; x < 90; x++ {
		img.Set(x, 25, color.RGBA{0, 0, 0, 255})
	}
	
	return img
}
