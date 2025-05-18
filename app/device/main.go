package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lomehong/kennel/pkg/core/plugin"
	sdk "github.com/lomehong/kennel/pkg/sdk/go"
)

func main() {
	// 创建设备管理模块
	module := NewDeviceModule()

	// 创建默认配置
	config := &plugin.ModuleConfig{
		Settings: map[string]interface{}{
			"log_level":               "info",
			"monitor_usb":             true,
			"monitor_network":         true,
			"monitor_interval":        60,
			"device_cache_expiration": 30,
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
