package main

import (
	"context"
	"fmt"
	"os"

	sdk "github.com/lomehong/kennel/pkg/sdk/go"
)

func main() {
	// 设置环境变量，确保插件使用正确的 Magic Cookie
	os.Setenv("PLUGIN_MAGIC_COOKIE", "kennel")

	// 创建资产管理模块
	module := NewAssetModule()

	// 初始化模块
	if err := module.Init(context.Background(), nil); err != nil {
		fmt.Fprintf(os.Stderr, "初始化模块失败: %v\n", err)
		os.Exit(1)
	}

	// 运行模块
	sdk.RunModule(module)
}
