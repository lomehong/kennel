package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/comm"
	"github.com/lomehong/kennel/pkg/logger"
)

var (
	serverURL = flag.String("server", "ws://localhost:8080/ws", "服务器URL")
)

func main() {
	flag.Parse()

	// 创建日志器
	log := logger.NewLogger("comm-test", hclog.Info)
	log.Info("通讯模块测试客户端启动")

	// 创建配置
	config := comm.DefaultConfig()
	config.ServerURL = *serverURL
	config.HeartbeatInterval = 15 * time.Second
	config.ReconnectInterval = 3 * time.Second

	// 创建通讯管理器
	manager := comm.NewManager(config, log)

	// 设置客户端信息
	manager.SetClientInfo(map[string]interface{}{
		"client_id":    "test-client-1",
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
	err := manager.Connect()
	if err != nil {
		log.Error("连接服务器失败", "error", err)
		os.Exit(1)
	}

	// 发送测试事件
	go func() {
		// 等待连接建立
		time.Sleep(2 * time.Second)

		// 发送事件
		log.Info("发送测试事件")
		manager.SendEvent("test_event", map[string]interface{}{
			"time": time.Now().Format(time.RFC3339),
			"data": "这是一个测试事件",
		})

		// 定期发送数据
		ticker := time.NewTicker(10 * time.Second)
		for range ticker.C {
			if manager.IsConnected() {
				log.Info("发送测试数据")
				manager.SendData("test_data", map[string]interface{}{
					"time":  time.Now().Format(time.RFC3339),
					"value": time.Now().Unix(),
				})
			}
		}
	}()

	// 等待中断信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// 断开连接
	log.Info("断开连接并退出")
	manager.Disconnect()
}

// handleCommand 处理命令消息
func handleCommand(msg *comm.Message) {
	fmt.Printf("收到命令: %v\n", msg.Payload["command"])

	// 如果是ping命令，回复pong
	if cmd, ok := msg.Payload["command"].(string); ok && cmd == "ping" {
		fmt.Println("收到ping命令，回复pong")
	}
}

// handleData 处理数据消息
func handleData(msg *comm.Message) {
	fmt.Printf("收到数据: %v\n", msg.Payload)
}

// handleEvent 处理事件消息
func handleEvent(msg *comm.Message) {
	fmt.Printf("收到事件: %v\n", msg.Payload)
}
