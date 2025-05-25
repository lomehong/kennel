//go:build tesseract
// +build tesseract

package analyzer

import (
	"fmt"
	"strings"

	"github.com/otiai10/gosseract/v2"
)

// performTesseractOCRWithLib 使用Tesseract库执行OCR
func (t *TesseractOCR) performTesseractOCRWithLib(imgBytes []byte) (string, error) {
	// 创建Tesseract客户端
	client := gosseract.NewClient()
	defer client.Close()

	// 设置语言
	if len(t.languages) > 0 {
		err := client.SetLanguage(strings.Join(t.languages, "+"))
		if err != nil {
			return "", fmt.Errorf("设置OCR语言失败: %w", err)
		}
	}

	// 设置图像数据
	err := client.SetImageFromBytes(imgBytes)
	if err != nil {
		return "", fmt.Errorf("设置OCR图像数据失败: %w", err)
	}

	// 执行OCR
	text, err := client.Text()
	if err != nil {
		return "", fmt.Errorf("OCR文本提取失败: %w", err)
	}

	return text, nil
}

// isTesseractLibAvailable 检查Tesseract库是否可用
func (t *TesseractOCR) isTesseractLibAvailable() bool {
	defer func() {
		if r := recover(); r != nil {
			// 如果panic，说明Tesseract不可用
		}
	}()

	// 尝试创建客户端
	client := gosseract.NewClient()
	defer client.Close()
	
	return true
}
