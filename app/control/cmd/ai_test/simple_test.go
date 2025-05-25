package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lomehong/kennel/app/control/pkg/ai"
	"github.com/lomehong/kennel/pkg/logging"
)

func main() {
	// 解析命令行参数
	apiKey := flag.String("api-key", "", "API密钥")
	modelName := flag.String("model", "gpt-3.5-turbo", "模型名称")
	modelType := flag.String("model-type", "openai", "模型类型，可选值为openai或ark")
	baseURL := flag.String("base-url", "", "API基础URL，可选")
	streaming := flag.Bool("stream", false, "是否使用流式响应")
	flag.Parse()

	// 检查API密钥
	if *apiKey == "" {
		// 尝试从环境变量获取
		if *modelType == "openai" {
			*apiKey = os.Getenv("OPENAI_API_KEY")
			if *apiKey == "" {
				fmt.Println("错误: 未提供OpenAI API密钥，请使用 -api-key 参数或设置 OPENAI_API_KEY 环境变量")
				os.Exit(1)
			}
		} else if *modelType == "ark" {
			*apiKey = os.Getenv("ARK_API_KEY")
			if *apiKey == "" {
				fmt.Println("错误: 未提供Ark API密钥，请使用 -api-key 参数或设置 ARK_API_KEY 环境变量")
				os.Exit(1)
			}
		} else {
			fmt.Printf("错误: 不支持的模型类型: %s，支持的类型为openai和ark\n", *modelType)
			os.Exit(1)
		}
	}

	// 创建日志记录器
	logConfig := logging.DefaultLogConfig()
	logConfig.Level = logging.LogLevelInfo
	baseLogger, err := logging.NewEnhancedLogger(logConfig)
	if err != nil {
		fmt.Printf("创建日志记录器失败: %v\n", err)
		os.Exit(1)
	}
	logger := baseLogger.Named("ai-test")
	logger.Info("初始化AI测试程序")

	// 创建上下文
	ctx := context.Background()

	// 创建配置
	aiConfig := map[string]interface{}{
		"enabled":    true,
		"model_type": *modelType,
		"model_name": *modelName,
		"api_key":    *apiKey,
	}

	// 如果提供了基础URL，则添加到配置中
	if *baseURL != "" {
		aiConfig["base_url"] = *baseURL
	}

	config := map[string]interface{}{
		"ai": aiConfig,
	}

	// 创建AI管理器
	aiManager := ai.NewAIManager(logger, config)

	// 初始化AI管理器
	if err := aiManager.Init(ctx); err != nil {
		logger.Error("初始化AI管理器失败", "error", err)
		os.Exit(1)
	}

	fmt.Println("欢迎使用终端管控AI助手!")
	fmt.Printf("当前使用的模型: %s (%s)\n", *modelName, *modelType)
	if *baseURL != "" {
		fmt.Printf("API基础URL: %s\n", *baseURL)
	}
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

		// 发送请求
		fmt.Println("\n正在处理请求...")
		startTime := time.Now()

		if *streaming {
			// 流式处理
			responseChan := make(chan string)
			errorChan := make(chan error, 1)

			// 启动流式处理
			go func() {
				err := aiManager.HandleStreamRequest(ctx, input, func(content string) error {
					responseChan <- content
					return nil
				})
				if err != nil {
					errorChan <- err
					close(responseChan)
					return
				}
				close(responseChan)
			}()

			// 处理流式响应
			fmt.Println("\nAI助手: ")
			responseComplete := false
			timeout := time.After(5 * time.Minute)

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
			// 非流式处理
			response, err := aiManager.HandleRequest(ctx, input)
			if err != nil {
				fmt.Printf("处理请求失败: %v\n", err)
				continue
			}
			fmt.Println("\nAI助手: " + response)
		}

		// 显示处理时间
		duration := time.Since(startTime)
		fmt.Printf("\n处理时间: %v\n", duration)
	}

	fmt.Println("程序已退出")
}
