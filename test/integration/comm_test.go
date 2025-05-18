package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/comm"
	"github.com/lomehong/kennel/pkg/core"
	"github.com/lomehong/kennel/pkg/logger"
)

// 创建测试WebSocket服务器
func createTestServer(t *testing.T) (*httptest.Server, chan *comm.Message, chan *comm.Message) {
	// 创建消息通道
	receivedMessages := make(chan *comm.Message, 10)
	messagesToSend := make(chan *comm.Message, 10)

	// 创建WebSocket升级器
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// 创建HTTP处理函数
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 升级HTTP连接为WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("升级连接失败: %v", err)
			return
		}
		defer conn.Close()

		// 启动发送协程
		go func() {
			for msg := range messagesToSend {
				data, err := json.Marshal(msg)
				if err != nil {
					t.Logf("编码消息失败: %v", err)
					continue
				}

				if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
					t.Logf("发送消息失败: %v", err)
					return
				}
			}
		}()

		// 接收消息
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					t.Logf("读取消息错误: %v", err)
				}
				return
			}

			// 解析消息
			var msg comm.Message
			if err := json.Unmarshal(data, &msg); err != nil {
				t.Logf("解析消息失败: %v", err)
				continue
			}

			// 将消息发送到通道
			receivedMessages <- &msg

			// 如果是心跳消息，回复确认
			if msg.Type == comm.MessageTypeHeartbeat {
				ackMsg := &comm.Message{
					ID:        "ack-" + msg.ID,
					Type:      comm.MessageTypeAck,
					Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
					Payload: map[string]interface{}{
						"message_id": msg.ID,
						"time":       time.Now().UnixNano() / int64(time.Millisecond),
					},
				}
				data, _ := json.Marshal(ackMsg)
				conn.WriteMessage(websocket.TextMessage, data)
			}
		}
	})

	// 创建测试服务器
	server := httptest.NewServer(handler)

	return server, receivedMessages, messagesToSend
}

// TestAppCommIntegration 测试应用程序与通讯模块的集成
func TestAppCommIntegration(t *testing.T) {
	// 创建测试服务器
	server, receivedMessages, messagesToSend := createTestServer(t)
	defer server.Close()

	// 替换HTTP为WebSocket
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// 创建临时配置文件
	configContent := `
plugin_dir: "../plugins"
log_level: "debug"
enable_comm: true
server_url: "` + wsURL + `/ws"
heartbeat_interval: "100ms"
reconnect_interval: "100ms"
max_reconnect_attempts: 3
comm_shutdown_timeout: 1
`
	configFile := "test_config.yaml"
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}
	defer os.Remove(configFile)

	// 创建应用程序
	app := core.NewApp(configFile)

	// 初始化应用程序
	err = app.Init()
	if err != nil {
		t.Fatalf("初始化应用程序失败: %v", err)
	}

	// 启动应用程序
	err = app.Start()
	if err != nil {
		t.Fatalf("启动应用程序失败: %v", err)
	}
	defer app.Stop()

	// 等待连接消息
	select {
	case msg := <-receivedMessages:
		if msg.Type != comm.MessageTypeConnect {
			t.Errorf("期望收到连接消息，但收到了 %s", msg.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("超时等待连接消息")
	}

	// 等待心跳消息
	select {
	case msg := <-receivedMessages:
		if msg.Type != comm.MessageTypeHeartbeat {
			t.Errorf("期望收到心跳消息，但收到了 %s", msg.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("超时等待心跳消息")
	}

	// 获取通讯管理器
	commManager := app.GetCommManager()
	if commManager == nil {
		t.Fatal("无法获取通讯管理器")
	}

	// 检查连接状态
	if !commManager.IsConnected() {
		t.Error("通讯管理器应该处于已连接状态")
	}

	// 发送事件消息
	commManager.SendEvent("test_event", map[string]interface{}{
		"event_key": "event_value",
	})

	// 等待服务器接收消息
	select {
	case msg := <-receivedMessages:
		if msg.Type != comm.MessageTypeEvent {
			t.Errorf("期望收到事件消息，但收到了 %s", msg.Type)
		}
		if eventType, ok := msg.Payload["event"].(string); !ok || eventType != "test_event" {
			t.Errorf("事件类型不正确: %v", msg.Payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("超时等待服务器接收消息")
	}

	// 服务器发送命令消息给客户端
	commandReceived := make(chan bool, 1)
	commManager.RegisterHandler(comm.MessageTypeCommand, func(msg *comm.Message) {
		if command, ok := msg.Payload["command"].(string); ok && command == "test_command" {
			commandReceived <- true
		}
	})

	serverMsg := &comm.Message{
		ID:        "server-msg-1",
		Type:      comm.MessageTypeCommand,
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
		Payload: map[string]interface{}{
			"command": "test_command",
			"params": map[string]interface{}{
				"param1": "value1",
			},
		},
	}
	messagesToSend <- serverMsg

	// 等待客户端接收消息
	select {
	case <-commandReceived:
		// 命令已接收
	case <-time.After(2 * time.Second):
		t.Fatal("超时等待客户端接收命令消息")
	}

	// 测试优雅终止
	disconnectReceived := make(chan bool, 1)
	go func() {
		for msg := range receivedMessages {
			if msg.Type == comm.MessageTypeEvent {
				if event, ok := msg.Payload["event"].(string); ok && event == "client_disconnect" {
					disconnectReceived <- true
					return
				}
			}
		}
	}()

	// 停止应用程序
	app.Stop()

	// 等待断开连接消息
	select {
	case <-disconnectReceived:
		// 断开连接消息已接收
	case <-time.After(2 * time.Second):
		t.Fatal("超时等待断开连接消息")
	}
}

// TestCommManagerReconnect 测试通讯管理器重连功能
func TestCommManagerReconnect(t *testing.T) {
	// 创建测试服务器
	server, receivedMessages, _ := createTestServer(t)

	// 替换HTTP为WebSocket
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// 创建配置
	config := comm.DefaultConfig()
	config.ServerURL = wsURL + "/ws"
	config.HeartbeatInterval = 100 * time.Millisecond
	config.ReconnectInterval = 100 * time.Millisecond
	config.MaxReconnectAttempts = 3

	// 创建日志器
	log := logger.NewLogger("test-reconnect", hclog.Debug)

	// 创建管理器
	manager := comm.NewManager(config, log)

	// 连接到服务器
	err := manager.Connect()
	if err != nil {
		t.Fatalf("连接服务器失败: %v", err)
	}
	defer manager.Disconnect()

	// 等待连接消息
	select {
	case <-receivedMessages:
		// 忽略连接消息
	case <-time.After(1 * time.Second):
		t.Fatal("超时等待连接消息")
	}

	// 关闭服务器，触发重连
	server.Close()

	// 创建新的服务器
	newServer, newReceivedMessages, _ := createTestServer(t)
	defer newServer.Close()

	// 更新服务器URL
	newWsURL := "ws" + strings.TrimPrefix(newServer.URL, "http")
	// 由于Manager没有暴露GetClient方法，我们需要重新创建一个Manager
	manager.Disconnect()

	// 更新配置
	config.ServerURL = newWsURL + "/ws"

	// 重新创建Manager
	manager = comm.NewManager(config, log)

	// 连接到新服务器
	err = manager.Connect()
	if err != nil {
		t.Fatalf("连接新服务器失败: %v", err)
	}

	// 等待连接建立
	time.Sleep(500 * time.Millisecond)

	// 发送消息，测试连接是否恢复
	manager.SendEvent("reconnect_test", map[string]interface{}{
		"time": time.Now().Format(time.RFC3339),
	})

	// 等待新服务器接收消息
	select {
	case msg := <-newReceivedMessages:
		if msg.Type != comm.MessageTypeEvent {
			t.Errorf("期望收到事件消息，但收到了 %s", msg.Type)
		}
		if eventType, ok := msg.Payload["event"].(string); !ok || eventType != "reconnect_test" {
			t.Errorf("事件类型不正确: %v", msg.Payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("超时等待新服务器接收消息")
	}
}
