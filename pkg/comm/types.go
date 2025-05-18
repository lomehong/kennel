package comm

import (
	"time"
)

// MessageType 定义消息类型
type MessageType string

const (
	// 系统消息类型
	MessageTypeHeartbeat MessageType = "heartbeat" // 心跳消息
	MessageTypeConnect   MessageType = "connect"   // 连接消息
	MessageTypeAck       MessageType = "ack"       // 确认消息

	// 业务消息类型
	MessageTypeCommand  MessageType = "command"  // 命令消息
	MessageTypeData     MessageType = "data"     // 数据消息
	MessageTypeEvent    MessageType = "event"    // 事件消息
	MessageTypeResponse MessageType = "response" // 响应消息
)

// Message 定义通用消息结构
type Message struct {
	ID        string                 `json:"id"`        // 消息ID
	Type      MessageType            `json:"type"`      // 消息类型
	Timestamp int64                  `json:"timestamp"` // 时间戳
	Payload   map[string]interface{} `json:"payload"`   // 消息内容
}

// NewMessage 创建一个新消息
func NewMessage(msgType MessageType, payload map[string]interface{}) *Message {
	return &Message{
		ID:        generateID(),
		Type:      msgType,
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
		Payload:   payload,
	}
}

// ConnectionState 定义连接状态
type ConnectionState int

const (
	StateDisconnected ConnectionState = iota // 断开连接
	StateConnecting                          // 正在连接
	StateConnected                           // 已连接
	StateReconnecting                        // 正在重连
)

// String 返回连接状态的字符串表示
func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "断开连接"
	case StateConnecting:
		return "正在连接"
	case StateConnected:
		return "已连接"
	case StateReconnecting:
		return "正在重连"
	default:
		return "未知状态"
	}
}

// ConnectionConfig 定义连接配置
type ConnectionConfig struct {
	ServerURL            string         // 服务器URL
	ReconnectInterval    time.Duration  // 重连间隔
	MaxReconnectAttempts int            // 最大重连次数
	HeartbeatInterval    time.Duration  // 心跳间隔
	HandshakeTimeout     time.Duration  // 握手超时
	WriteTimeout         time.Duration  // 写超时
	ReadTimeout          time.Duration  // 读超时
	MessageBufferSize    int            // 消息缓冲区大小
	Security             SecurityConfig // 安全配置
}

// SecurityConfig 定义安全配置
type SecurityConfig struct {
	EnableTLS        bool   // 是否启用TLS
	VerifyServerCert bool   // 是否验证服务器证书
	ClientCertFile   string // 客户端证书文件路径
	ClientKeyFile    string // 客户端私钥文件路径
	CACertFile       string // CA证书文件路径

	EnableEncryption bool   // 是否启用消息加密
	EncryptionKey    string // 加密密钥

	EnableAuth bool   // 是否启用认证
	AuthToken  string // 认证令牌
	AuthType   string // 认证类型 (basic, token, jwt)
	Username   string // 用户名 (用于basic认证)
	Password   string // 密码 (用于basic认证)

	EnableCompression    bool // 是否启用消息压缩
	CompressionLevel     int  // 压缩级别 (1-9，1最快，9最高压缩率)
	CompressionThreshold int  // 压缩阈值，超过该大小才进行压缩（字节）
}

// DefaultConfig 返回默认配置
func DefaultConfig() ConnectionConfig {
	return ConnectionConfig{
		ServerURL:            "ws://localhost:8080/ws",
		ReconnectInterval:    time.Second * 5,
		MaxReconnectAttempts: 10,
		HeartbeatInterval:    time.Second * 30,
		HandshakeTimeout:     time.Second * 10,
		WriteTimeout:         time.Second * 10,
		ReadTimeout:          time.Second * 60,
		MessageBufferSize:    100,
		Security: SecurityConfig{
			EnableTLS:        false,
			VerifyServerCert: true,
			ClientCertFile:   "",
			ClientKeyFile:    "",
			CACertFile:       "",

			EnableEncryption: false,
			EncryptionKey:    "",

			EnableAuth: false,
			AuthToken:  "",
			AuthType:   "token",
			Username:   "",
			Password:   "",

			EnableCompression:    false,
			CompressionLevel:     6,
			CompressionThreshold: 1024,
		},
	}
}

// MessageHandler 定义消息处理函数类型
type MessageHandler func(msg *Message)

// ConnectionStateHandler 定义连接状态变化处理函数类型
type ConnectionStateHandler func(oldState, newState ConnectionState)

// ErrorHandler 定义错误处理函数类型
type ErrorHandler func(err error)
