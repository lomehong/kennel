package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lomehong/kennel/app/control/pkg/control"
	"github.com/lomehong/kennel/pkg/core/plugin"
	sdk "github.com/lomehong/kennel/pkg/sdk/go"
)

func main() {
	// 设置环境变量，确保插件使用正确的 Magic Cookie
	os.Setenv("APPFRAMEWORK_PLUGIN", "appframework")

	// 创建终端管控模块
	module := control.NewControlModule()

	// 初始化模块
	config := &plugin.ModuleConfig{
		Settings: make(map[string]interface{}),
	}
	if err := module.Init(context.Background(), config); err != nil {
		fmt.Fprintf(os.Stderr, "初始化模块失败: %v\n", err)
		os.Exit(1)
	}

	// 运行模块
	sdk.RunModule(module)
}
