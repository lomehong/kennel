package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lomehong/kennel/pkg/core/plugin"
	sdk "github.com/lomehong/kennel/pkg/sdk/go"
)

func main() {
	// 创建数据防泄漏模块
	module := NewDLPModule()

	// 创建默认配置
	config := &plugin.ModuleConfig{
		Settings: map[string]interface{}{
			"log_level":         "info",
			"monitor_clipboard": true,
			"monitor_files":     true,
			"monitored_directories": []string{
				"data/dlp/monitored",
			},
			"monitored_file_types": []string{
				"*.txt", "*.doc", "*.docx", "*.xls", "*.xlsx", "*.pdf",
			},
		},
	}

	// 初始化模块
	if err := module.Init(context.Background(), config); err != nil {
		fmt.Fprintf(os.Stderr, "初始化模块失败: %v\n", err)
		os.Exit(1)
	}

	// 运行模块
	sdk.RunModule(module)
}
