package main

import (
	hplugin "github.com/hashicorp/go-plugin"
	pluginLib "github.com/lomehong/kennel/pkg/plugin"
)

// main 是插件的入口点
func main() {
	// 创建插件
	assetModule := NewAssetModule()

	// 启动插件
	hplugin.Serve(&hplugin.ServeConfig{
		HandshakeConfig: hplugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "APPFRAMEWORK_PLUGIN",
			MagicCookieValue: "appframework",
		},
		Plugins: map[string]hplugin.Plugin{
			"module": &pluginLib.ModulePlugin{
				Impl: assetModule,
			},
		},
		GRPCServer: hplugin.DefaultGRPCServer,
	})
}
