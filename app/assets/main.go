package main

import (
	"github.com/lomehong/kennel/pkg/sdk/go"
)

func main() {
	// 创建资产管理模块
	module := NewAssetModule()

	// 运行模块
	sdk.RunModule(module)
}
