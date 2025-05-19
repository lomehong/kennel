package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lomehong/kennel/app/control/ai"
	"github.com/lomehong/kennel/app/control/ai/mcp"
	sdk "github.com/lomehong/kennel/pkg/sdk/go"
)

func main() {
	// 解析命令行参数
	addr := flag.String("addr", ":8080", "服务器监听地址")
	apiKey := flag.String("api-key", "", "API密钥，用于认证")
	logLevel := flag.String("log-level", "info", "日志级别: debug, info, warn, error")
	flag.Parse()

	// 创建日志记录器
	logger := sdk.NewLogger("mcp-server", getLogLevel(*logLevel))
	logger.Info("初始化 MCP Server", "addr", *addr)

	// 创建服务器配置
	config := &mcp.ServerConfig{
		Addr:         *addr,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		APIKey:       *apiKey,
	}

	// 创建服务器
	server, err := mcp.NewServer(config, logger)
	if err != nil {
		logger.Error("创建 MCP Server 失败", "error", err)
		os.Exit(1)
	}

	// 注册工具
	registerTools(server, logger)

	// 启动服务器
	go func() {
		logger.Info("启动 MCP Server", "addr", *addr)
		if err := server.Start(); err != nil {
			logger.Error("启动 MCP Server 失败", "error", err)
			os.Exit(1)
		}
	}()

	// 等待信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// 优雅关闭
	logger.Info("关闭 MCP Server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("关闭 MCP Server 失败", "error", err)
	}
}

// 注册工具
func registerTools(server *mcp.Server, logger sdk.Logger) {
	// 创建进程列表工具
	processListTool := &mcp.BaseTool{
		Name:        "get_processes",
		Description: "获取系统进程列表，可以按名称过滤",
		Parameters: map[string]mcp.Parameter{
			"name_filter": {
				Type:        "string",
				Description: "进程名称过滤条件，可选",
				Required:    false,
			},
			"limit": {
				Type:        "number",
				Description: "返回的最大进程数量，默认为20",
				Required:    false,
				Default:     20,
			},
		},
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			// 解析参数
			nameFilter := ""
			if filter, ok := params["name_filter"].(string); ok {
				nameFilter = filter
			}

			limit := 20
			if limitVal, ok := params["limit"].(float64); ok {
				limit = int(limitVal)
			}

			// 创建工具实例
			tool := ai.NewProcessListTool(logger)

			// 构造参数
			argsJSON := fmt.Sprintf(`{"name_filter": "%s", "limit": %d}`, nameFilter, limit)

			// 执行工具
			runFunc := tool.Run()
			result, err := runFunc(ctx, argsJSON)
			if err != nil {
				return nil, err
			}

			// 解析结果
			var processes []interface{}
			if err := json.Unmarshal([]byte(result), &processes); err != nil {
				return nil, err
			}

			return processes, nil
		},
	}

	// 创建进程终止工具
	processKillTool := &mcp.BaseTool{
		Name:        "kill_process",
		Description: "终止指定的进程",
		Parameters: map[string]mcp.Parameter{
			"pid": {
				Type:        "number",
				Description: "进程ID",
				Required:    true,
			},
			"force": {
				Type:        "boolean",
				Description: "是否强制终止进程",
				Required:    false,
				Default:     false,
			},
		},
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			// 解析参数
			pidFloat, ok := params["pid"].(float64)
			if !ok {
				return nil, fmt.Errorf("无效的进程ID")
			}
			pid := int(pidFloat)

			force := false
			if forceVal, ok := params["force"].(bool); ok {
				force = forceVal
			}

			// 创建工具实例
			tool := ai.NewProcessKillTool(logger)

			// 构造参数
			argsJSON := fmt.Sprintf(`{"pid": %d, "force": %t}`, pid, force)

			// 执行工具
			runFunc := tool.Run()
			result, err := runFunc(ctx, argsJSON)
			if err != nil {
				return nil, err
			}

			// 解析结果
			var killResult map[string]interface{}
			if err := json.Unmarshal([]byte(result), &killResult); err != nil {
				return nil, err
			}

			return killResult, nil
		},
	}

	// 创建命令执行工具
	commandExecTool := &mcp.BaseTool{
		Name:        "execute_command",
		Description: "执行系统命令，返回命令的输出结果",
		Parameters: map[string]mcp.Parameter{
			"command": {
				Type:        "string",
				Description: "要执行的命令",
				Required:    true,
			},
			"args": {
				Type:        "array",
				Description: "命令参数列表",
				Required:    false,
			},
			"timeout": {
				Type:        "number",
				Description: "命令执行超时时间（秒），默认为30秒",
				Required:    false,
				Default:     30,
			},
		},
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			// 解析参数
			command, ok := params["command"].(string)
			if !ok {
				return nil, fmt.Errorf("无效的命令")
			}

			args := []string{}
			if argsVal, ok := params["args"].([]interface{}); ok {
				for _, arg := range argsVal {
					if argStr, ok := arg.(string); ok {
						args = append(args, argStr)
					}
				}
			}

			timeout := 30
			if timeoutVal, ok := params["timeout"].(float64); ok {
				timeout = int(timeoutVal)
			}

			// 创建工具实例
			tool := ai.NewCommandExecTool(logger)

			// 构造参数
			argsJSON, err := json.Marshal(map[string]interface{}{
				"command": command,
				"args":    args,
				"timeout": timeout,
			})
			if err != nil {
				return nil, err
			}

			// 执行工具
			runFunc := tool.Run()
			result, err := runFunc(ctx, string(argsJSON))
			if err != nil {
				return nil, err
			}

			// 解析结果
			var execResult map[string]interface{}
			if err := json.Unmarshal([]byte(result), &execResult); err != nil {
				return nil, err
			}

			return execResult, nil
		},
	}

	// 注册工具
	if err := server.RegisterTool(processListTool); err != nil {
		logger.Error("注册进程列表工具失败", "error", err)
	} else {
		logger.Info("注册进程列表工具成功")
	}

	if err := server.RegisterTool(processKillTool); err != nil {
		logger.Error("注册进程终止工具失败", "error", err)
	} else {
		logger.Info("注册进程终止工具成功")
	}

	if err := server.RegisterTool(commandExecTool); err != nil {
		logger.Error("注册命令执行工具失败", "error", err)
	} else {
		logger.Info("注册命令执行工具成功")
	}
}

// 获取日志级别
func getLogLevel(level string) sdk.LogLevel {
	switch level {
	case "debug":
		return sdk.LogLevelDebug
	case "info":
		return sdk.LogLevelInfo
	case "warn":
		return sdk.LogLevelWarn
	case "error":
		return sdk.LogLevelError
	default:
		return sdk.LogLevelInfo
	}
}
