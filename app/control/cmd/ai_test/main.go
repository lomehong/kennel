package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lomehong/kennel/app/control"
	"github.com/lomehong/kennel/pkg/core/plugin"
	sdk "github.com/lomehong/kennel/pkg/sdk/go"
)

// 导入终端管控模块

func main() {
	// 解析命令行参数
	apiKey := flag.String("api-key", "", "OpenAI API密钥")
	modelName := flag.String("model", "gpt-3.5-turbo", "OpenAI模型名称")
	streaming := flag.Bool("stream", false, "是否使用流式响应")
	flag.Parse()

	// 检查API密钥
	if *apiKey == "" {
		// 尝试从环境变量获取
		*apiKey = os.Getenv("OPENAI_API_KEY")
		if *apiKey == "" {
			fmt.Println("错误: 未提供OpenAI API密钥，请使用 -api-key 参数或设置 OPENAI_API_KEY 环境变量")
			os.Exit(1)
		}
	}

	// 创建日志记录器
	logger := sdk.NewLogger("ai-test", sdk.LogLevelInfo)
	logger.Info("初始化AI测试程序")

	// 创建上下文
	ctx := context.Background()

	// 创建模块配置
	config := &plugin.ModuleConfig{
		Settings: map[string]interface{}{
			"log_level": "info",
			"ai": map[string]interface{}{
				"enabled":    true,
				"model_type": "openai",
				"model_name": *modelName,
				"api_key":    *apiKey,
			},
		},
	}

	// 创建终端管控模块
	module := control.NewControlModule()

	// 初始化模块
	if err := module.Init(ctx, config); err != nil {
		logger.Error("初始化模块失败", "error", err)
		os.Exit(1)
	}

	// 启动模块
	if err := module.Start(); err != nil {
		logger.Error("启动模块失败", "error", err)
		os.Exit(1)
	}

	// 确保在程序退出时停止模块
	defer module.Stop()

	fmt.Println("欢迎使用终端管控AI助手!")
	fmt.Println("输入 'exit' 或 'quit' 退出程序")
	fmt.Println("输入 'stream on' 启用流式响应")
	fmt.Println("输入 'stream off' 禁用流式响应")
	fmt.Println("输入其他内容将被视为对AI助手的查询")
	fmt.Println("------------------------------------")

	// 创建输入扫描器
	scanner := bufio.NewScanner(os.Stdin)

	// 主循环
	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}

		// 获取输入
		input := scanner.Text()
		input = strings.TrimSpace(input)

		// 检查是否退出
		if input == "exit" || input == "quit" {
			break
		}

		// 检查是否切换流式响应模式
		if input == "stream on" {
			*streaming = true
			fmt.Println("已启用流式响应")
			continue
		}
		if input == "stream off" {
			*streaming = false
			fmt.Println("已禁用流式响应")
			continue
		}

		// 跳过空输入
		if input == "" {
			continue
		}

		// 创建请求
		req := &plugin.Request{
			ID:     fmt.Sprintf("req-%d", time.Now().UnixNano()),
			Action: "ai_query",
			Params: map[string]interface{}{
				"query":     input,
				"streaming": *streaming,
			},
		}

		// 发送请求
		fmt.Println("\n正在处理请求...")
		startTime := time.Now()
		resp, err := module.HandleRequest(ctx, req)
		if err != nil {
			fmt.Printf("处理请求失败: %v\n", err)
			continue
		}

		if !resp.Success {
			fmt.Printf("请求失败: %s\n", resp.Error.Message)
			continue
		}

		// 处理响应
		if *streaming {
			// 处理流式响应
			responseChan := resp.Data["channel"].(chan string)
			errorChan := resp.Data["error"].(chan error)

			// 设置超时
			timeout := time.After(5 * time.Minute)

			// 收集响应
			fmt.Println("\nAI助手: ")
			responseComplete := false

			for !responseComplete {
				select {
				case response, ok := <-responseChan:
					if !ok {
						responseComplete = true
						break
					}
					fmt.Print(response)
				case err := <-errorChan:
					fmt.Printf("\n处理请求失败: %v\n", err)
					responseComplete = true
				case <-timeout:
					fmt.Println("\n请求超时")
					responseComplete = true
				}
			}
		} else {
			// 处理非流式响应
			response := resp.Data["response"].(string)
			fmt.Println("\nAI助手: " + response)
		}

		// 显示处理时间
		duration := time.Since(startTime)
		fmt.Printf("\n处理时间: %v\n", duration)
	}

	fmt.Println("程序已退出")
}
