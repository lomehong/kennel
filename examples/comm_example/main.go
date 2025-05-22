package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/lomehong/kennel/pkg/comm"
	"github.com/lomehong/kennel/pkg/logging"
)

func main() {
	fmt.Println("通讯模块示例")
	fmt.Println("==============================================")

	// 创建日志器
	logConfig := logging.DefaultLogConfig()
	logConfig.Level = logging.LogLevelInfo
	log, _ := logging.NewEnhancedLogger(logConfig)
	log = log.Named("comm-example")
	log.Info("初始化通讯模块")

	// 创建配置
	config := comm.DefaultConfig()
	config.ServerURL = "ws://localhost:8080/ws" // 修改为实际的服务器地址
	config.HeartbeatInterval = 15 * time.Second
	config.ReconnectInterval = 3 * time.Second

	// 创建通讯管理器
	manager := comm.NewManager(config, log)

	// 设置客户端信息
	manager.SetClientInfo(map[string]interface{}{
		"client_id":    "example-client-1",
		"version":      "1.0.0",
		"os":           runtime.GOOS,
		"arch":         runtime.GOARCH,
		"connect_time": time.Now().Format(time.RFC3339),
	})

	// 注册消息处理函数
	manager.RegisterHandler(comm.MessageTypeCommand, handleCommand)
	manager.RegisterHandler(comm.MessageTypeData, handleData)
	manager.RegisterHandler(comm.MessageTypeEvent, handleEvent)

	// 连接到服务器
	log.Info("连接到服务器", "url", config.ServerURL)
	err := manager.Connect()
	if err != nil {
		log.Error("连接服务器失败", "error", err)
		os.Exit(1)
	}

	// 启动定期发送数据的协程
	go func() {
		// 等待连接建立
		time.Sleep(2 * time.Second)

		// 发送初始事件
		log.Info("发送初始事件")
		manager.SendEvent("client_started", map[string]interface{}{
			"time": time.Now().Format(time.RFC3339),
			"info": "客户端已启动",
		})

		// 定期发送数据
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		counter := 0
		for range ticker.C {
			if manager.IsConnected() {
				counter++
				log.Info("发送周期数据", "counter", counter)

				// 发送系统信息
				manager.SendData("system_info", map[string]interface{}{
					"time":       time.Now().Format(time.RFC3339),
					"counter":    counter,
					"memory_mb":  getMemoryUsage(),
					"cpu_usage":  getCPUUsage(),
					"uptime_sec": getUptime(),
				})
			}
		}
	}()

	// 等待中断信号
	fmt.Println("客户端已启动，按Ctrl+C退出")
	fmt.Println("==============================================")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// 断开连接
	log.Info("断开连接并退出")
	manager.Disconnect()
}

// handleCommand 处理命令消息
func handleCommand(msg *comm.Message) {
	command, ok := msg.Payload["command"].(string)
	if !ok {
		fmt.Println("收到无效的命令消息")
		return
	}

	params, _ := msg.Payload["params"].(map[string]interface{})

	fmt.Printf("收到命令: %s, 参数: %v\n", command, params)

	// 处理不同类型的命令
	switch command {
	case "ping":
		fmt.Println("收到ping命令，回复pong")
		// 可以在这里回复pong
	case "welcome":
		fmt.Println("收到欢迎消息")
	case "update":
		fmt.Println("收到更新命令")
		// 可以在这里处理更新逻辑
	case "restart":
		fmt.Println("收到重启命令")
		// 可以在这里处理重启逻辑
	default:
		fmt.Printf("收到未知命令: %s\n", command)
	}
}

// handleData 处理数据消息
func handleData(msg *comm.Message) {
	dataType, ok := msg.Payload["type"].(string)
	if !ok {
		fmt.Println("收到无效的数据消息")
		return
	}

	data, _ := msg.Payload["data"]

	fmt.Printf("收到数据: 类型=%s, 内容=%v\n", dataType, data)
}

// handleEvent 处理事件消息
func handleEvent(msg *comm.Message) {
	eventType, ok := msg.Payload["event"].(string)
	if !ok {
		fmt.Println("收到无效的事件消息")
		return
	}

	details, _ := msg.Payload["details"].(map[string]interface{})

	fmt.Printf("收到事件: %s, 详情: %v\n", eventType, details)
}

// 模拟获取内存使用量
func getMemoryUsage() float64 {
	// 在实际应用中，应该使用系统API获取真实的内存使用量
	return 100.0 + float64(time.Now().Second()%10)*10.0
}

// 模拟获取CPU使用率
func getCPUUsage() float64 {
	// 在实际应用中，应该使用系统API获取真实的CPU使用率
	return 5.0 + float64(time.Now().Second()%20)
}

// 模拟获取运行时间
func getUptime() int64 {
	// 在实际应用中，应该记录程序启动时间并计算真实的运行时间
	return time.Now().Unix() % 3600
}
