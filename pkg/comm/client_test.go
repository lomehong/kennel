package comm

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/logger"
)

// 创建测试WebSocket服务器
func createTestServer(t *testing.T) (*httptest.Server, chan *Message, chan *Message) {
	// 创建消息通道
	receivedMessages := make(chan *Message, 10)
	messagesToSend := make(chan *Message, 10)

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
				data, err := encodeMessage(msg)
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
			msg, err := decodeMessage(data)
			if err != nil {
				t.Logf("解析消息失败: %v", err)
				continue
			}

			// 将消息发送到通道
			receivedMessages <- msg

			// 如果是心跳消息，回复确认
			if msg.Type == MessageTypeHeartbeat {
				ackMsg := createAckMessage(msg.ID)
				data, _ := encodeMessage(ackMsg)
				conn.WriteMessage(websocket.TextMessage, data)
			}
		}
	})

	// 创建测试服务器
	server := httptest.NewServer(handler)

	return server, receivedMessages, messagesToSend
}

// TestClientConnect 测试客户端连接功能
func TestClientConnect(t *testing.T) {
	// 创建测试服务器
	server, receivedMessages, _ := createTestServer(t)
	defer server.Close()

	// 替换HTTP为WebSocket
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// 创建配置
	config := DefaultConfig()
	config.ServerURL = wsURL + "/ws"
	config.HeartbeatInterval = 100 * time.Millisecond
	config.ReconnectInterval = 100 * time.Millisecond

	// 创建日志器
	log := logger.NewLogger("test-client", hclog.Debug)

	// 创建客户端
	client := NewClient(config, log)

	// 设置客户端信息
	client.SetClientInfo(map[string]interface{}{
		"client_id": "test-client",
		"test_mode": true,
	})

	// 连接到服务器
	err := client.Connect()
	if err != nil {
		t.Fatalf("连接服务器失败: %v", err)
	}
	defer client.Disconnect()

	// 等待连接消息
	select {
	case msg := <-receivedMessages:
		if msg.Type != MessageTypeConnect {
			t.Errorf("期望收到连接消息，但收到了 %s", msg.Type)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("超时等待连接消息")
	}

	// 等待心跳消息
	select {
	case msg := <-receivedMessages:
		if msg.Type != MessageTypeHeartbeat {
			t.Errorf("期望收到心跳消息，但收到了 %s", msg.Type)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("超时等待心跳消息")
	}

	// 检查连接状态
	if !client.IsConnected() {
		t.Error("客户端应该处于已连接状态")
	}
	if client.GetState() != StateConnected {
		t.Errorf("客户端状态应该是 StateConnected，但是 %v", client.GetState())
	}
}

// TestClientSendReceive 测试客户端发送和接收消息功能
func TestClientSendReceive(t *testing.T) {
	// 创建测试服务器
	server, receivedMessages, messagesToSend := createTestServer(t)
	defer server.Close()

	// 替换HTTP为WebSocket
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// 创建配置
	config := DefaultConfig()
	config.ServerURL = wsURL + "/ws"
	config.HeartbeatInterval = 1 * time.Second
	config.ReconnectInterval = 100 * time.Millisecond

	// 创建日志器
	log := logger.NewLogger("test-client", hclog.Debug)

	// 创建客户端
	client := NewClient(config, log)

	// 创建消息接收通道
	receivedClientMessages := make(chan *Message, 10)
	client.SetMessageHandler(func(msg *Message) {
		receivedClientMessages <- msg
	})

	// 连接到服务器
	err := client.Connect()
	if err != nil {
		t.Fatalf("连接服务器失败: %v", err)
	}
	defer client.Disconnect()

	// 等待连接消息
	select {
	case <-receivedMessages:
		// 忽略连接消息
	case <-time.After(1 * time.Second):
		t.Fatal("超时等待连接消息")
	}

	// 发送测试消息
	testMsg := NewMessage(MessageTypeData, map[string]interface{}{
		"test_key": "test_value",
	})
	client.Send(testMsg)

	// 等待服务器接收消息
	select {
	case msg := <-receivedMessages:
		if msg.Type != MessageTypeData {
			t.Errorf("期望收到数据消息，但收到了 %s", msg.Type)
		}
		if value, ok := msg.Payload["test_key"].(string); !ok || value != "test_value" {
			t.Errorf("消息内容不正确: %v", msg.Payload)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("超时等待服务器接收消息")
	}

	// 服务器发送消息给客户端
	serverMsg := NewMessage(MessageTypeCommand, map[string]interface{}{
		"command": "test_command",
		"params": map[string]interface{}{
			"param1": "value1",
		},
	})
	messagesToSend <- serverMsg

	// 等待客户端接收消息
	select {
	case msg := <-receivedClientMessages:
		if msg.Type != MessageTypeCommand {
			t.Errorf("期望收到命令消息，但收到了 %s", msg.Type)
		}
		if command, ok := msg.Payload["command"].(string); !ok || command != "test_command" {
			t.Errorf("命令不正确: %v", msg.Payload)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("超时等待客户端接收消息")
	}
}

// TestClientDisconnect 测试客户端断开连接功能
func TestClientDisconnect(t *testing.T) {
	// 创建测试服务器
	server, receivedMessages, _ := createTestServer(t)
	defer server.Close()

	// 替换HTTP为WebSocket
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// 创建配置
	config := DefaultConfig()
	config.ServerURL = wsURL + "/ws"
	config.HeartbeatInterval = 1 * time.Second
	config.ReconnectInterval = 100 * time.Millisecond

	// 创建日志器
	log := logger.NewLogger("test-client", hclog.Debug)

	// 创建客户端
	client := NewClient(config, log)

	// 连接到服务器
	err := client.Connect()
	if err != nil {
		t.Fatalf("连接服务器失败: %v", err)
	}

	// 等待连接消息
	select {
	case <-receivedMessages:
		// 忽略连接消息
	case <-time.After(1 * time.Second):
		t.Fatal("超时等待连接消息")
	}

	// 断开连接
	client.Disconnect()

	// 检查连接状态
	if client.IsConnected() {
		t.Error("客户端应该处于断开连接状态")
	}
	if client.GetState() != StateDisconnected {
		t.Errorf("客户端状态应该是 StateDisconnected，但是 %v", client.GetState())
	}

	// 等待关闭消息
	select {
	case msg := <-receivedMessages:
		if msg.Type != MessageTypeEvent {
			t.Errorf("期望收到事件消息，但收到了 %s", msg.Type)
		}
		if event, ok := msg.Payload["event"].(string); !ok || event != "client_disconnect" {
			t.Errorf("事件不正确: %v", msg.Payload)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("超时等待关闭消息")
	}
}
