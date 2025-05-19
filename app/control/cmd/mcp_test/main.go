package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/lomehong/kennel/app/control/ai/mcp"
	sdk "github.com/lomehong/kennel/pkg/sdk/go"
)

func main() {
	// 解析命令行参数
	serverAddr := flag.String("server", "http://localhost:8080", "MCP Server地址")
	apiKey := flag.String("api-key", "", "API密钥")
	action := flag.String("action", "list", "操作: list, get, execute")
	toolName := flag.String("tool", "", "工具名称")
	paramsJSON := flag.String("params", "{}", "工具参数，JSON格式")
	flag.Parse()

	// 创建日志记录器
	logger := sdk.NewLogger("mcp-test", sdk.LogLevelInfo)

	// 创建客户端配置
	config := &mcp.ClientConfig{
		ServerAddr: *serverAddr,
		APIKey:     *apiKey,
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}

	// 创建客户端
	client, err := mcp.NewClient(config, logger)
	if err != nil {
		logger.Error("创建客户端失败", "error", err)
		os.Exit(1)
	}
	defer client.Close()

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 执行操作
	switch *action {
	case "list":
		// 列出工具
		tools, err := client.ListTools(ctx)
		if err != nil {
			logger.Error("列出工具失败", "error", err)
			os.Exit(1)
		}

		// 打印工具列表
		fmt.Printf("找到 %d 个工具:\n", len(tools))
		for i, tool := range tools {
			fmt.Printf("%d. %s: %s\n", i+1, tool.Name, tool.Description)
			fmt.Printf("   参数:\n")
			for name, param := range tool.Parameters {
				required := ""
				if param.Required {
					required = " (必需)"
				}
				fmt.Printf("   - %s: %s%s\n", name, param.Description, required)
			}
			fmt.Println()
		}

	case "get":
		// 检查工具名称
		if *toolName == "" {
			logger.Error("缺少工具名称")
			os.Exit(1)
		}

		// 获取工具信息
		tool, err := client.GetTool(ctx, *toolName)
		if err != nil {
			logger.Error("获取工具信息失败", "error", err)
			os.Exit(1)
		}

		// 打印工具信息
		fmt.Printf("工具: %s\n", tool.Name)
		fmt.Printf("描述: %s\n", tool.Description)
		fmt.Printf("参数:\n")
		for name, param := range tool.Parameters {
			required := ""
			if param.Required {
				required = " (必需)"
			}
			fmt.Printf("- %s: %s%s\n", name, param.Description, required)
		}

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
		result, err := client.ExecuteTool(ctx, *toolName, params)
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
