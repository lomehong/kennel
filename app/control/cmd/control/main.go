package main

import (
	"github.com/lomehong/kennel/app/control"
	sdk "github.com/lomehong/kennel/pkg/sdk/go"
)

func main() {
	// 创建终端管控模块
	module := control.NewControlModule()

	// 运行模块
	sdk.RunModule(module)
}
