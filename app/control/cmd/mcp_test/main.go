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

	"github.com/lomehong/kennel/app/control/pkg/ai/mcp"
	sdk "github.com/lomehong/kennel/pkg/sdk/go"
)

func main() {
	// 解析命令行参数
	serverAddr := flag.String("server", "http://localhost:8080", "MCP 服务器地址")
	apiKey := flag.String("api-key", "", "API 密钥")
	action := flag.String("action", "list", "操作: list, query, execute")
	toolName := flag.String("tool", "", "工具名称")
	paramsJSON := flag.String("params", "{}", "工具参数，JSON 格式")
	query := flag.String("query", "", "查询内容")
	timeout := flag.Duration("timeout", 30*time.Second, "超时时间")
	debug := flag.Bool("debug", false, "是否启用调试日志")
	flag.Parse()

	// 创建日志记录器
	logLevel := sdk.LogLevelInfo
	if *debug {
		logLevel = sdk.LogLevelDebug
	}
	logger := sdk.NewLogger("mcp-test", logLevel)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 处理信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("接收到信号，正在关闭...")
		cancel()
	}()

	// 创建 MCP 管理器配置
	config := &mcp.ManagerConfig{
		Enabled:    true,
		ServerAddr: *serverAddr,
		APIKey:     *apiKey,
		Timeout:    *timeout,
		MaxRetries: 3,
	}

	// 创建 MCP 管理器
	manager, err := mcp.NewManager(config, logger)
	if err != nil {
		logger.Error("创建 MCP 管理器失败", "error", err)
		os.Exit(1)
	}

	// 启动 MCP 管理器
	if err := manager.Start(ctx); err != nil {
		logger.Error("启动 MCP 管理器失败", "error", err)
		os.Exit(1)
	}
	defer manager.Stop()

	// 等待 MCP 管理器初始化
	time.Sleep(1 * time.Second)

	// 执行操作
	switch *action {
	case "list":
		// 获取工具列表
		tools := manager.GetTools()
		if len(tools) == 0 {
			logger.Info("没有可用的工具")
		} else {
			fmt.Printf("找到 %d 个工具:\n", len(tools))
			i := 1
			for name, tool := range tools {
				fmt.Printf("%d. %s: %s\n", i, name, tool.Description)
				fmt.Printf("   参数:\n")
				for paramName, param := range tool.Parameters {
					fmt.Printf("   - %s: %v\n", paramName, param)
				}
				fmt.Println()
				i++
			}
		}

	case "query":
		// 检查查询内容
		if *query == "" {
			logger.Error("缺少查询内容")
			os.Exit(1)
		}

		// 发送查询
		logger.Info("发送查询", "query", *query)
		response, err := manager.QueryAI(ctx, *query)
		if err != nil {
			logger.Error("查询失败", "error", err)
			os.Exit(1)
		}

		// 打印响应
		fmt.Println("响应:")
		fmt.Println(response)

	case "execute":
		// 检查工具名称
		if *toolName == "" {
			logger.Error("缺少工具名称")
			os.Exit(1)
		}

		// 解析参数
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(*paramsJSON), &params); err != nil {
			logger.Error("解析参数失败", "error", err)
			os.Exit(1)
		}

		// 执行工具
		logger.Info("执行工具", "name", *toolName, "params", params)
		result, err := manager.ExecuteTool(ctx, *toolName, params)
		if err != nil {
			logger.Error("执行工具失败", "error", err)
			os.Exit(1)
		}

		// 打印结果
		resultJSON, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			logger.Error("序列化结果失败", "error", err)
			os.Exit(1)
		}
		fmt.Printf("执行结果:\n%s\n", string(resultJSON))

	default:
		logger.Error("未知操作", "action", *action)
		os.Exit(1)
	}
}
