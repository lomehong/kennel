package main

import (
	hplugin "github.com/hashicorp/go-plugin"
	pluginLib "github.com/lomehong/kennel/pkg/plugin"
)

func main() {
	// 创建插件
	auditModule := NewAuditModule()

	// 启动插件
	hplugin.Serve(&hplugin.ServeConfig{
		HandshakeConfig: hplugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "APPFRAMEWORK_PLUGIN",
			MagicCookieValue: "appframework",
		},
		Plugins: map[string]hplugin.Plugin{
			"module": &pluginLib.ModulePlugin{
				Impl: auditModule,
			},
		},
		GRPCServer: hplugin.DefaultGRPCServer,
	})
}
