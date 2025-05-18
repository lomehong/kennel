package profiler

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// IsRunning 检查指定类型的性能分析是否正在运行
func (p *StandardProfiler) IsRunning(profileType ProfileType) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running[profileType]
}

// GetRunningProfiles 获取正在运行的性能分析
func (p *StandardProfiler) GetRunningProfiles() map[ProfileType]ProfileOptions {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 复制选项
	profiles := make(map[ProfileType]ProfileOptions)
	for profileType, options := range p.options {
		if p.running[profileType] {
			profiles[profileType] = options
		}
	}

	return profiles
}

// GetResults 获取性能分析结果
func (p *StandardProfiler) GetResults() []*ProfileResult {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 复制结果
	results := make([]*ProfileResult, len(p.results))
	copy(results, p.results)

	return results
}

// GetResult 获取指定类型的最新性能分析结果
func (p *StandardProfiler) GetResult(profileType ProfileType) *ProfileResult {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 从后向前查找指定类型的结果
	for i := len(p.results) - 1; i >= 0; i-- {
		if p.results[i].Type == profileType {
			return p.results[i]
		}
	}

	return nil
}

// WriteProfile 将性能分析数据写入指定的写入器
func (p *StandardProfiler) WriteProfile(profileType ProfileType, format ProfileFormat, w io.Writer) error {
	// 获取最新的性能分析结果
	result := p.GetResult(profileType)
	if result == nil {
		return fmt.Errorf("未找到性能分析结果: %s", profileType)
	}

	// 如果格式相同，直接复制文件内容
	if result.Format == format {
		f, err := os.Open(result.FilePath)
		if err != nil {
			return fmt.Errorf("打开性能分析文件失败: %v", err)
		}
		defer f.Close()

		_, err = io.Copy(w, f)
		if err != nil {
			return fmt.Errorf("复制性能分析数据失败: %v", err)
		}

		return nil
	}

	// 如果格式不同，需要转换
	// 创建临时文件
	tempDir, err := os.MkdirTemp("", "profile")
	if err != nil {
		return fmt.Errorf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempFile := filepath.Join(tempDir, fmt.Sprintf("%s.%s", profileType, format))

	// 转换格式
	if err := p.ConvertProfile(result.FilePath, tempFile, format); err != nil {
		return fmt.Errorf("转换性能分析格式失败: %v", err)
	}

	// 读取转换后的文件
	f, err := os.Open(tempFile)
	if err != nil {
		return fmt.Errorf("打开转换后的文件失败: %v", err)
	}
	defer f.Close()

	// 复制到写入器
	_, err = io.Copy(w, f)
	if err != nil {
		return fmt.Errorf("复制转换后的数据失败: %v", err)
	}

	return nil
}

// ConvertProfile 转换性能分析数据格式
func (p *StandardProfiler) ConvertProfile(inputPath string, outputPath string, format ProfileFormat) error {
	// 检查输入文件是否存在
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return fmt.Errorf("输入文件不存在: %s", inputPath)
	}

	// 创建输出目录
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}

	// 根据格式执行不同的转换
	switch format {
	case ProfileFormatPprof:
		// pprof格式是原始格式，不需要转换
		// 直接复制文件
		return copyFile(inputPath, outputPath)

	case ProfileFormatText:
		// 使用go tool pprof转换为文本格式
		cmd := exec.Command("go", "tool", "pprof", "-text", inputPath)
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("转换为文本格式失败: %v", err)
		}
		return os.WriteFile(outputPath, output, 0644)

	case ProfileFormatJSON:
		// 使用go tool pprof转换为JSON格式
		cmd := exec.Command("go", "tool", "pprof", "-json", inputPath)
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("转换为JSON格式失败: %v", err)
		}
		return os.WriteFile(outputPath, output, 0644)

	case ProfileFormatSVG:
		// 使用go tool pprof转换为SVG格式
		cmd := exec.Command("go", "tool", "pprof", "-svg", inputPath)
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("转换为SVG格式失败: %v", err)
		}
		return os.WriteFile(outputPath, output, 0644)

	case ProfileFormatPDF:
		// 使用go tool pprof转换为PDF格式
		cmd := exec.Command("go", "tool", "pprof", "-pdf", inputPath)
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("转换为PDF格式失败: %v", err)
		}
		return os.WriteFile(outputPath, output, 0644)

	case ProfileFormatHTML:
		// 使用go tool pprof转换为HTML格式
		cmd := exec.Command("go", "tool", "pprof", "-html", inputPath)
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("转换为HTML格式失败: %v", err)
		}
		return os.WriteFile(outputPath, output, 0644)

	default:
		return fmt.Errorf("不支持的格式: %s", format)
	}
}

// AnalyzeProfile 分析性能分析数据
func (p *StandardProfiler) AnalyzeProfile(profileType ProfileType, filePath string) (map[string]interface{}, error) {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("文件不存在: %s", filePath)
	}

	// 使用go tool pprof获取top信息
	cmd := exec.Command("go", "tool", "pprof", "-top", "-json", filePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("分析性能分析数据失败: %v", err)
	}

	// 解析JSON输出
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("解析分析结果失败: %v", err)
	}

	return result, nil
}

// Cleanup 清理性能分析数据
func (p *StandardProfiler) Cleanup(olderThan time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 获取当前时间
	now := time.Now()

	// 遍历结果列表
	var newResults []*ProfileResult
	for _, result := range p.results {
		// 如果结果不够旧，保留
		if now.Sub(result.EndTime) < olderThan {
			newResults = append(newResults, result)
			continue
		}

		// 删除文件
		if err := os.Remove(result.FilePath); err != nil && !os.IsNotExist(err) {
			p.logger.Warn("删除性能分析文件失败", "file", result.FilePath, "error", err)
		} else {
			p.logger.Info("删除性能分析文件", "file", result.FilePath)
		}
	}

	// 更新结果列表
	p.results = newResults

	// 清理空目录
	cleanupEmptyDirs(p.outputDir)

	return nil
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	// 打开源文件
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %v", err)
	}
	defer sourceFile.Close()

	// 创建目标文件
	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %v", err)
	}
	defer destFile.Close()

	// 复制内容
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("复制文件内容失败: %v", err)
	}

	return nil
}

// cleanupEmptyDirs 清理空目录
func cleanupEmptyDirs(dir string) {
	// 遍历目录
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		// 跳过错误和非目录
		if err != nil || !info.IsDir() || path == dir {
			return nil
		}

		// 读取目录内容
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil
		}

		// 如果目录为空，删除
		if len(entries) == 0 {
			os.Remove(path)
		}

		return nil
	})
}
