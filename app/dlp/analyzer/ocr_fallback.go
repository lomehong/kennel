//go:build !tesseract
// +build !tesseract

package analyzer

import (
	"fmt"
)

// performTesseractOCRWithLib 备用OCR实现（不使用Tesseract库）
func (t *TesseractOCR) performTesseractOCRWithLib(imgBytes []byte) (string, error) {
	t.logger.Warn("Tesseract库未编译，OCR功能不可用")
	return "", fmt.Errorf("OCR功能不可用：Tesseract库未编译，请使用 'go build -tags tesseract' 编译或安装Tesseract")
}

// isTesseractLibAvailable 检查Tesseract库是否可用（备用实现）
func (t *TesseractOCR) isTesseractLibAvailable() bool {
	return false
}
