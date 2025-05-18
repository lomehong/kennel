package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	addr     = flag.String("addr", "localhost:8080", "服务地址")
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // 允许所有来源
		},
	}
	clients     = make(map[*websocket.Conn]bool)
	clientMutex sync.Mutex
)

// Message 消息结构
type Message struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Timestamp int64                  `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

func main() {
	flag.Parse()

	// 设置路由
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/", handleHome)

	// 启动服务器
	fmt.Println("==============================================")
	fmt.Println("WebSocket服务器启动中...")
	fmt.Printf("服务器地址: http://%s\n", *addr)
	fmt.Printf("WebSocket端点: ws://%s/ws\n", *addr)
	fmt.Println("按Ctrl+C停止服务器")
	fmt.Println("==============================================")
	log.Printf("服务器启动在 %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

// handleHome 处理主页请求
func handleHome(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "WebSocket服务器运行中")
}

// handleWebSocket 处理WebSocket连接
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 升级HTTP连接为WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("升级连接失败: %v", err)
		return
	}
	defer conn.Close()

	// 添加客户端
	clientMutex.Lock()
	clients[conn] = true
	clientMutex.Unlock()

	// 移除客户端
	defer func() {
		clientMutex.Lock()
		delete(clients, conn)
		clientMutex.Unlock()
	}()

	log.Printf("客户端已连接: %s", conn.RemoteAddr())

	// 启动心跳检测
	go sendHeartbeat(conn)

	// 处理消息
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("读取消息错误: %v", err)
			}
			break
		}

		// 解析消息
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("解析消息失败: %v", err)
			continue
		}

		// 处理消息
		handleMessage(conn, &msg)
	}
}

// handleMessage 处理接收到的消息
func handleMessage(conn *websocket.Conn, msg *Message) {
	log.Printf("收到消息: %s, ID: %s", msg.Type, msg.ID)

	switch msg.Type {
	case "connect":
		// 发送欢迎消息
		sendMessage(conn, &Message{
			ID:        generateID(),
			Type:      "command",
			Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
			Payload: map[string]interface{}{
				"command": "welcome",
				"params": map[string]interface{}{
					"message": "欢迎连接到服务器",
				},
			},
		})
	case "heartbeat":
		// 回复心跳确认
		sendMessage(conn, &Message{
			ID:        generateID(),
			Type:      "ack",
			Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
			Payload: map[string]interface{}{
				"message_id": msg.ID,
				"time":       time.Now().UnixNano() / int64(time.Millisecond),
			},
		})
	case "event":
		// 确认事件接收
		sendMessage(conn, &Message{
			ID:        generateID(),
			Type:      "ack",
			Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
			Payload: map[string]interface{}{
				"message_id": msg.ID,
				"time":       time.Now().UnixNano() / int64(time.Millisecond),
			},
		})
	case "data":
		// 确认数据接收
		sendMessage(conn, &Message{
			ID:        generateID(),
			Type:      "ack",
			Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
			Payload: map[string]interface{}{
				"message_id": msg.ID,
				"time":       time.Now().UnixNano() / int64(time.Millisecond),
			},
		})
	}
}

// sendMessage 发送消息
func sendMessage(conn *websocket.Conn, msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("编码消息失败: %v", err)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("发送消息失败: %v", err)
	}
}

// sendHeartbeat 定期发送心跳
func sendHeartbeat(conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 发送心跳
			sendMessage(conn, &Message{
				ID:        generateID(),
				Type:      "heartbeat",
				Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
				Payload: map[string]interface{}{
					"time": time.Now().UnixNano() / int64(time.Millisecond),
				},
			})

			// 随机发送命令
			if time.Now().Unix()%3 == 0 {
				sendMessage(conn, &Message{
					ID:        generateID(),
					Type:      "command",
					Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
					Payload: map[string]interface{}{
						"command": "ping",
						"params": map[string]interface{}{
							"time": time.Now().UnixNano() / int64(time.Millisecond),
						},
					},
				})
			}
		}
	}
}

// generateID 生成唯一ID
func generateID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
}
