package comm

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/logger"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源
	},
}

// 启动一个简单的WebSocket服务器用于测试
func startTestServer(t *testing.T, port int, done chan struct{}) {
	// 创建路由
	mux := http.NewServeMux()

	// 处理WebSocket连接
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("升级连接失败: %v", err)
			return
		}
		defer conn.Close()

		t.Logf("客户端已连接: %s", conn.RemoteAddr())

		// 发送欢迎消息
		welcomeMsg := NewMessage(MessageTypeCommand, map[string]interface{}{
			"command": "welcome",
			"params": map[string]interface{}{
				"message": "欢迎连接到测试服务器",
			},
		})

		data, _ := encodeMessage(welcomeMsg)
		conn.WriteMessage(websocket.TextMessage, data)

		// 处理消息
		for {
			_, msgData, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					t.Logf("读取消息错误: %v", err)
				}
				break
			}

			// 解析消息
			msg, err := decodeMessage(msgData)
			if err != nil {
				t.Logf("解析消息失败: %v", err)
				continue
			}

			t.Logf("收到消息: %s, ID: %s", msg.Type, msg.ID)

			// 回复确认消息
			ackMsg := NewMessage(MessageTypeAck, map[string]interface{}{
				"message_id": msg.ID,
				"time":       time.Now().UnixNano() / int64(time.Millisecond),
			})

			data, _ := encodeMessage(ackMsg)
			conn.WriteMessage(websocket.TextMessage, data)

			// 如果是心跳消息，发送一个命令消息
			if msg.Type == MessageTypeHeartbeat {
				cmdMsg := NewMessage(MessageTypeCommand, map[string]interface{}{
					"command": "ping",
					"params": map[string]interface{}{
						"time": time.Now().UnixNano() / int64(time.Millisecond),
					},
				})

				data, _ := encodeMessage(cmdMsg)
				conn.WriteMessage(websocket.TextMessage, data)
			}
		}
	})

	// 处理主页请求
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "WebSocket测试服务器运行中")
	})

	// 创建服务器
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	// 启动服务器
	go func() {
		t.Logf("测试服务器启动在端口 %d", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("服务器错误: %v", err)
		}
	}()

	// 等待关闭信号
	<-done

	t.Log("关闭测试服务器")
}

// 测试通讯模块的基本功能
func TestCommBasic(t *testing.T) {
	// 启动测试服务器
	serverDone := make(chan struct{})
	port := 8080
	go startTestServer(t, port, serverDone)

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 创建日志器
	log := logger.NewLogger("comm-test", hclog.Info)

	// 创建配置
	config := DefaultConfig()
	config.ServerURL = fmt.Sprintf("ws://localhost:%d/ws", port)
	config.HeartbeatInterval = 1 * time.Second
	config.ReconnectInterval = 500 * time.Millisecond

	// 创建通讯管理器
	manager := NewManager(config, log)

	// 设置客户端信息
	manager.SetClientInfo(map[string]interface{}{
		"client_id":    "test-client",
		"version":      "1.0.0",
		"test_mode":    true,
		"connect_time": time.Now().Format(time.RFC3339),
	})

	// 注册消息处理函数
	receivedCommand := make(chan *Message, 1)
	manager.RegisterHandler(MessageTypeCommand, func(msg *Message) {
		t.Logf("收到命令消息: %v", msg.Payload)
		receivedCommand <- msg
	})

	// 连接到服务器
	err := manager.Connect()
	if err != nil {
		t.Fatalf("连接服务器失败: %v", err)
	}
	defer manager.Disconnect()

	// 等待接收欢迎消息
	select {
	case msg := <-receivedCommand:
		cmd, _ := msg.Payload["command"].(string)
		if cmd != "welcome" {
			t.Errorf("期望收到welcome命令，但收到了 %s", cmd)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("超时等待欢迎消息")
	}

	// 发送事件消息
	manager.SendEvent("test_event", map[string]interface{}{
		"time": time.Now().Format(time.RFC3339),
		"data": "这是一个测试事件",
	})

	// 等待接收ping命令（由心跳触发）
	select {
	case msg := <-receivedCommand:
		cmd, _ := msg.Payload["command"].(string)
		if cmd != "ping" {
			t.Errorf("期望收到ping命令，但收到了 %s", cmd)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("超时等待ping命令")
	}

	// 关闭测试服务器
	close(serverDone)
}
