package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lomehong/kennel/pkg/comm"
	"github.com/lomehong/kennel/pkg/logging"
)

var (
	serverMode  = flag.Bool("server", false, "运行服务器模式")
	clientMode  = flag.Bool("client", false, "运行客户端模式")
	serverAddr  = flag.String("addr", "localhost:9000", "服务器地址")
	serverPath  = flag.String("path", "/ws", "WebSocket路径")
	logLevel    = flag.String("log-level", "info", "日志级别")
	interactive = flag.Bool("interactive", false, "交互模式")
)

// 服务器模式
func runServer() {
	log.Printf("启动WebSocket服务器在 %s%s", *serverAddr, *serverPath)

	// 创建升级器
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// 创建连接管理器
	type clientConnection struct {
		conn      *websocket.Conn
		send      chan []byte
		connected bool
	}
	var (
		clients    = make(map[*websocket.Conn]*clientConnection)
		clientsMux sync.Mutex
	)

	// 处理WebSocket连接
	http.HandleFunc(*serverPath, func(w http.ResponseWriter, r *http.Request) {
		// 升级HTTP连接为WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("升级连接失败: %v", err)
			return
		}

		// 创建客户端连接
		client := &clientConnection{
			conn:      conn,
			send:      make(chan []byte, 256),
			connected: true,
		}

		// 添加到客户端列表
		clientsMux.Lock()
		clients[conn] = client
		clientsMux.Unlock()

		log.Printf("客户端已连接: %s", conn.RemoteAddr())

		// 启动发送协程
		go func() {
			defer func() {
				conn.Close()
				clientsMux.Lock()
				delete(clients, conn)
				clientsMux.Unlock()
				log.Printf("客户端已断开连接: %s", conn.RemoteAddr())
			}()

			for {
				select {
				case message, ok := <-client.send:
					if !ok {
						// 通道已关闭
						conn.WriteMessage(websocket.CloseMessage, []byte{})
						return
					}

					err := conn.WriteMessage(websocket.TextMessage, message)
					if err != nil {
						log.Printf("发送消息失败: %v", err)
						return
					}
				}
			}
		}()

		// 接收消息
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					log.Printf("读取消息错误: %v", err)
				}
				break
			}

			// 解析消息
			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Printf("解析消息失败: %v", err)
				continue
			}

			// 打印消息
			msgType, _ := msg["type"].(string)
			msgID, _ := msg["id"].(string)
			log.Printf("收到消息: 类型=%s, ID=%s", msgType, msgID)

			// 如果是心跳消息，回复确认
			if msgType == "heartbeat" {
				ackMsg := map[string]interface{}{
					"id":        fmt.Sprintf("ack-%d", time.Now().UnixNano()),
					"type":      "ack",
					"timestamp": time.Now().UnixNano() / int64(time.Millisecond),
					"payload": map[string]interface{}{
						"message_id": msgID,
						"time":       time.Now().UnixNano() / int64(time.Millisecond),
					},
				}
				ackData, _ := json.Marshal(ackMsg)
				client.send <- ackData
			}

			// 如果是连接消息，发送欢迎消息
			if msgType == "connect" {
				welcomeMsg := map[string]interface{}{
					"id":        fmt.Sprintf("welcome-%d", time.Now().UnixNano()),
					"type":      "command",
					"timestamp": time.Now().UnixNano() / int64(time.Millisecond),
					"payload": map[string]interface{}{
						"command": "welcome",
						"params": map[string]interface{}{
							"message": "欢迎连接到测试服务器",
							"time":    time.Now().Format(time.RFC3339),
						},
					},
				}
				welcomeData, _ := json.Marshal(welcomeMsg)
				client.send <- welcomeData
			}
		}
	})

	// 启动HTTP服务器
	log.Fatal(http.ListenAndServe(*serverAddr, nil))
}

// 客户端模式
func runClient() {
	// 创建日志器
	logConfig := logging.DefaultLogConfig()
	switch *logLevel {
	case "debug":
		logConfig.Level = logging.LogLevelDebug
	case "info":
		logConfig.Level = logging.LogLevelInfo
	case "warn":
		logConfig.Level = logging.LogLevelWarn
	case "error":
		logConfig.Level = logging.LogLevelError
	default:
		logConfig.Level = logging.LogLevelInfo
	}
	baseLogger, err := logging.NewEnhancedLogger(logConfig)
	if err != nil {
		fmt.Printf("创建日志记录器失败: %v\n", err)
		os.Exit(1)
	}
	log := baseLogger.Named("comm-tester")

	// 创建配置
	config := comm.DefaultConfig()
	config.ServerURL = fmt.Sprintf("ws://%s%s", *serverAddr, *serverPath)
	config.HeartbeatInterval = 5 * time.Second
	config.ReconnectInterval = 3 * time.Second

	// 创建管理器
	manager := comm.NewManager(config, log)

	// 设置客户端信息
	manager.SetClientInfo(map[string]interface{}{
		"client_id":    fmt.Sprintf("tester-%d", time.Now().UnixNano()),
		"version":      "1.0.0",
		"os":           runtime.GOOS,
		"arch":         runtime.GOARCH,
		"connect_time": time.Now().Format(time.RFC3339),
	})

	// 注册消息处理函数
	manager.RegisterHandler(comm.MessageTypeCommand, func(msg *comm.Message) {
		command, _ := msg.Payload["command"].(string)
		fmt.Printf("收到命令: %s\n", command)
		fmt.Printf("消息内容: %v\n", msg.Payload)
	})

	manager.RegisterHandler(comm.MessageTypeData, func(msg *comm.Message) {
		dataType, _ := msg.Payload["type"].(string)
		fmt.Printf("收到数据: %s\n", dataType)
		fmt.Printf("消息内容: %v\n", msg.Payload)
	})

	manager.RegisterHandler(comm.MessageTypeEvent, func(msg *comm.Message) {
		eventType, _ := msg.Payload["event"].(string)
		fmt.Printf("收到事件: %s\n", eventType)
		fmt.Printf("消息内容: %v\n", msg.Payload)
	})

	// 连接到服务器
	fmt.Printf("连接到服务器 %s...\n", config.ServerURL)
	if err := manager.Connect(); err != nil {
		log.Error("连接服务器失败", "error", err)
		os.Exit(1)
	}
	defer manager.Disconnect()

	fmt.Println("已连接到服务器")

	// 如果是交互模式，启动命令处理
	if *interactive {
		fmt.Println("进入交互模式，输入命令:")
		fmt.Println("  send event <event_type> - 发送事件")
		fmt.Println("  send data <data_type> - 发送数据")
		fmt.Println("  send command <command> - 发送命令")
		fmt.Println("  exit - 退出")

		// 启动命令处理协程
		go func() {
			for {
				var cmd, cmdType, cmdValue string
				fmt.Print("> ")
				n, err := fmt.Scanf("%s %s %s", &cmd, &cmdType, &cmdValue)
				if err != nil || n < 3 {
					fmt.Println("无效的命令格式")
					continue
				}

				if cmd == "send" {
					switch cmdType {
					case "event":
						manager.SendEvent(cmdValue, map[string]interface{}{
							"time": time.Now().Format(time.RFC3339),
							"data": "测试事件数据",
						})
						fmt.Printf("已发送事件: %s\n", cmdValue)
					case "data":
						manager.SendData(cmdValue, map[string]interface{}{
							"time":  time.Now().Format(time.RFC3339),
							"value": time.Now().Unix(),
						})
						fmt.Printf("已发送数据: %s\n", cmdValue)
					case "command":
						manager.SendCommand(cmdValue, map[string]interface{}{
							"time":   time.Now().Format(time.RFC3339),
							"param1": "value1",
						})
						fmt.Printf("已发送命令: %s\n", cmdValue)
					default:
						fmt.Printf("未知的消息类型: %s\n", cmdType)
					}
				} else if cmd == "exit" {
					os.Exit(0)
				} else {
					fmt.Printf("未知的命令: %s\n", cmd)
				}
			}
		}()
	} else {
		// 非交互模式，定期发送事件
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		counter := 0
		for range ticker.C {
			if manager.IsConnected() {
				counter++
				fmt.Printf("发送测试事件 #%d\n", counter)
				manager.SendEvent("test_event", map[string]interface{}{
					"counter": counter,
					"time":    time.Now().Format(time.RFC3339),
				})
			}
		}
	}
}

func main() {
	flag.Parse()

	if !*serverMode && !*clientMode {
		fmt.Println("必须指定 -server 或 -client 模式")
		flag.Usage()
		os.Exit(1)
	}

	if *serverMode && *clientMode {
		fmt.Println("不能同时指定 -server 和 -client 模式")
		flag.Usage()
		os.Exit(1)
	}

	// 处理中断信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("接收到中断信号，退出...")
		os.Exit(0)
	}()

	if *serverMode {
		runServer()
	} else {
		runClient()
	}
}
