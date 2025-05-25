package parser

import (
	"net/http"
	"time"

	"github.com/lomehong/kennel/app/dlp/interceptor"

	"github.com/lomehong/kennel/pkg/logging"
)

// ParsedData 解析后的数据
type ParsedData struct {
	Protocol    string                 `json:"protocol"`
	Headers     map[string]string      `json:"headers"`
	Body        []byte                 `json:"body"`
	Metadata    map[string]interface{} `json:"metadata"`
	Sessions    []*SessionInfo         `json:"sessions"`
	ContentType string                 `json:"content_type"`
	URL         string                 `json:"url,omitempty"`
	Method      string                 `json:"method,omitempty"`
	StatusCode  int                    `json:"status_code,omitempty"`
}

// SessionInfo 会话信息
type SessionInfo struct {
	ID          string                 `json:"id"`
	Protocol    string                 `json:"protocol"`
	SourceIP    string                 `json:"source_ip"`
	DestIP      string                 `json:"dest_ip"`
	SourcePort  uint16                 `json:"source_port"`
	DestPort    uint16                 `json:"dest_port"`
	StartTime   time.Time              `json:"start_time"`
	LastSeen    time.Time              `json:"last_seen"`
	BytesSent   uint64                 `json:"bytes_sent"`
	BytesRecv   uint64                 `json:"bytes_recv"`
	PacketCount uint64                 `json:"packet_count"`
	State       SessionState           `json:"state"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// SessionState 会话状态
type SessionState int

const (
	SessionStateNew SessionState = iota
	SessionStateEstablished
	SessionStateClosing
	SessionStateClosed
)

// ParserInfo 解析器信息
type ParserInfo struct {
	Name               string   `json:"name"`
	Version            string   `json:"version"`
	Description        string   `json:"description"`
	SupportedProtocols []string `json:"supported_protocols"`
	Author             string   `json:"author"`
	License            string   `json:"license"`
}

// ProtocolParser 协议解析器接口
type ProtocolParser interface {
	// GetParserInfo 获取解析器信息
	GetParserInfo() ParserInfo

	// CanParse 检查是否能解析指定的数据包
	CanParse(packet *interceptor.PacketInfo) bool

	// Parse 解析数据包
	Parse(packet *interceptor.PacketInfo) (*ParsedData, error)

	// GetSupportedProtocols 获取支持的协议列表
	GetSupportedProtocols() []string

	// Initialize 初始化解析器
	Initialize(config ParserConfig) error

	// Cleanup 清理资源
	Cleanup() error
}

// ParserConfig 解析器配置
type ParserConfig struct {
	MaxBodySize    int64             `yaml:"max_body_size" json:"max_body_size"`
	Timeout        time.Duration     `yaml:"timeout" json:"timeout"`
	EnableTLS      bool              `yaml:"enable_tls" json:"enable_tls"`
	TLSConfig      *TLSConfig        `yaml:"tls_config" json:"tls_config"`
	BufferSize     int               `yaml:"buffer_size" json:"buffer_size"`
	SessionTimeout time.Duration     `yaml:"session_timeout" json:"session_timeout"`
	MaxSessions    int               `yaml:"max_sessions" json:"max_sessions"`
	EnableDeepScan bool              `yaml:"enable_deep_scan" json:"enable_deep_scan"`
	CustomHeaders  map[string]string `yaml:"custom_headers" json:"custom_headers"`
	Logger         logging.Logger    `yaml:"-" json:"-"`
}

// TLSConfig TLS配置
type TLSConfig struct {
	CertFile           string   `yaml:"cert_file" json:"cert_file"`
	KeyFile            string   `yaml:"key_file" json:"key_file"`
	CAFile             string   `yaml:"ca_file" json:"ca_file"`
	ServerName         string   `yaml:"server_name" json:"server_name"`
	CipherSuites       []uint16 `yaml:"cipher_suites" json:"cipher_suites"`
	MinVersion         uint16   `yaml:"min_version" json:"min_version"`
	MaxVersion         uint16   `yaml:"max_version" json:"max_version"`
	InsecureSkipVerify bool     `yaml:"insecure_skip_verify" json:"insecure_skip_verify"`
}

// DefaultParserConfig 返回默认解析器配置
func DefaultParserConfig() ParserConfig {
	return ParserConfig{
		MaxBodySize:    10 * 1024 * 1024, // 10MB
		Timeout:        30 * time.Second,
		EnableTLS:      true,
		BufferSize:     65536,
		SessionTimeout: 5 * time.Minute,
		MaxSessions:    10000,
		EnableDeepScan: true,
		CustomHeaders:  make(map[string]string),
	}
}

// ProtocolManager 协议解析管理器接口
type ProtocolManager interface {
	// RegisterParser 注册解析器
	RegisterParser(parser ProtocolParser) error

	// GetParser 获取解析器
	GetParser(protocol string) (ProtocolParser, bool)

	// ParsePacket 解析数据包
	ParsePacket(packet *interceptor.PacketInfo) (*ParsedData, error)

	// GetSupportedProtocols 获取支持的协议列表
	GetSupportedProtocols() []string

	// GetStats 获取统计信息
	GetStats() ParserStats

	// Start 启动管理器
	Start() error

	// Stop 停止管理器
	Stop() error
}

// ParserStats 解析器统计信息
type ParserStats struct {
	TotalPackets   uint64            `json:"total_packets"`
	ParsedPackets  uint64            `json:"parsed_packets"`
	FailedPackets  uint64            `json:"failed_packets"`
	ActiveSessions uint64            `json:"active_sessions"`
	TotalSessions  uint64            `json:"total_sessions"`
	BytesProcessed uint64            `json:"bytes_processed"`
	ParserStats    map[string]uint64 `json:"parser_stats"`
	LastError      error             `json:"last_error,omitempty"`
	StartTime      time.Time         `json:"start_time"`
	Uptime         time.Duration     `json:"uptime"`
}

// SessionManager 会话管理器接口
type SessionManager interface {
	// CreateSession 创建会话
	CreateSession(packet *interceptor.PacketInfo) *SessionInfo

	// GetSession 获取会话
	GetSession(sessionID string) (*SessionInfo, bool)

	// UpdateSession 更新会话
	UpdateSession(sessionID string, packet *interceptor.PacketInfo) error

	// CloseSession 关闭会话
	CloseSession(sessionID string) error

	// GetActiveSessions 获取活跃会话
	GetActiveSessions() []*SessionInfo

	// CleanupExpiredSessions 清理过期会话
	CleanupExpiredSessions() int

	// GetStats 获取统计信息
	GetStats() SessionStats
}

// SessionStats 会话统计信息
type SessionStats struct {
	ActiveSessions  uint64 `json:"active_sessions"`
	TotalSessions   uint64 `json:"total_sessions"`
	ExpiredSessions uint64 `json:"expired_sessions"`
	ClosedSessions  uint64 `json:"closed_sessions"`
}

// ContentExtractor 内容提取器接口
type ContentExtractor interface {
	// ExtractContent 提取内容
	ExtractContent(data *ParsedData) (*ExtractedContent, error)

	// GetSupportedTypes 获取支持的内容类型
	GetSupportedTypes() []string

	// CanExtract 检查是否能提取指定类型的内容
	CanExtract(contentType string) bool
}

// ExtractedContent 提取的内容
type ExtractedContent struct {
	ContentType string                 `json:"content_type"`
	Text        string                 `json:"text"`
	Images      []ImageInfo            `json:"images"`
	Files       []FileInfo             `json:"files"`
	Links       []LinkInfo             `json:"links"`
	Metadata    map[string]interface{} `json:"metadata"`
	Encoding    string                 `json:"encoding"`
	Language    string                 `json:"language"`
	Size        int64                  `json:"size"`
}

// ImageInfo 图片信息
type ImageInfo struct {
	URL    string `json:"url"`
	Alt    string `json:"alt"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Format string `json:"format"`
	Size   int64  `json:"size"`
}

// FileInfo 文件信息
type FileInfo struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Type      string `json:"type"`
	Size      int64  `json:"size"`
	Hash      string `json:"hash"`
	Extension string `json:"extension"`
}

// LinkInfo 链接信息
type LinkInfo struct {
	URL    string `json:"url"`
	Text   string `json:"text"`
	Title  string `json:"title"`
	Rel    string `json:"rel"`
	Target string `json:"target"`
}

// HTTPParser HTTP协议解析器接口
type HTTPParser interface {
	ProtocolParser

	// ParseRequest 解析HTTP请求
	ParseRequest(data []byte) (*http.Request, error)

	// ParseResponse 解析HTTP响应
	ParseResponse(data []byte) (*http.Response, error)

	// ExtractHeaders 提取HTTP头部
	ExtractHeaders(req *http.Request) map[string]string

	// ExtractBody 提取HTTP主体
	ExtractBody(req *http.Request) ([]byte, error)
}

// TLSParser TLS协议解析器接口
type TLSParser interface {
	ProtocolParser

	// ParseHandshake 解析TLS握手
	ParseHandshake(data []byte) (*TLSHandshakeInfo, error)

	// DecryptData 解密TLS数据
	DecryptData(data []byte, session *TLSSession) ([]byte, error)

	// GetCipherSuite 获取加密套件信息
	GetCipherSuite(suite uint16) *CipherSuiteInfo
}

// TLSHandshakeInfo TLS握手信息
type TLSHandshakeInfo struct {
	Version      uint16                 `json:"version"`
	CipherSuite  uint16                 `json:"cipher_suite"`
	ServerName   string                 `json:"server_name"`
	Certificates [][]byte               `json:"certificates"`
	Extensions   map[string]interface{} `json:"extensions"`
}

// TLSSession TLS会话
type TLSSession struct {
	SessionID    []byte `json:"session_id"`
	MasterSecret []byte `json:"master_secret"`
	ClientRandom []byte `json:"client_random"`
	ServerRandom []byte `json:"server_random"`
}

// CipherSuiteInfo 加密套件信息
type CipherSuiteInfo struct {
	ID          uint16 `json:"id"`
	Name        string `json:"name"`
	KeyExchange string `json:"key_exchange"`
	Cipher      string `json:"cipher"`
	MAC         string `json:"mac"`
	Secure      bool   `json:"secure"`
}

// ParserFactory 解析器工厂接口
type ParserFactory interface {
	// CreateParser 创建解析器
	CreateParser(protocol string, config ParserConfig) (ProtocolParser, error)

	// GetSupportedProtocols 获取支持的协议
	GetSupportedProtocols() []string

	// RegisterParserType 注册解析器类型
	RegisterParserType(protocol string, creator ParserCreator) error
}

// ParserCreator 解析器创建函数
type ParserCreator func(config ParserConfig) (ProtocolParser, error)

// DatabaseParser 数据库协议解析器接口
type DatabaseParser interface {
	ProtocolParser

	// ParseQuery 解析数据库查询
	ParseQuery(data []byte) (*DatabaseQuery, error)

	// ParseResult 解析查询结果
	ParseResult(data []byte) (*DatabaseResult, error)

	// GetConnectionInfo 获取连接信息
	GetConnectionInfo() *DatabaseConnectionInfo
}

// DatabaseQuery 数据库查询
type DatabaseQuery struct {
	Type       string                 `json:"type"`       // SELECT, INSERT, UPDATE, DELETE, etc.
	SQL        string                 `json:"sql"`        // 原始SQL语句
	Tables     []string               `json:"tables"`     // 涉及的表
	Columns    []string               `json:"columns"`    // 涉及的列
	Parameters []interface{}          `json:"parameters"` // 查询参数
	Database   string                 `json:"database"`   // 数据库名
	Schema     string                 `json:"schema"`     // 模式名
	Metadata   map[string]interface{} `json:"metadata"`   // 元数据
}

// DatabaseResult 数据库查询结果
type DatabaseResult struct {
	RowCount    int64                    `json:"row_count"`    // 影响行数
	Columns     []string                 `json:"columns"`      // 列名
	Rows        []map[string]interface{} `json:"rows"`         // 结果行
	ExecuteTime time.Duration            `json:"execute_time"` // 执行时间
	Error       string                   `json:"error"`        // 错误信息
	Metadata    map[string]interface{}   `json:"metadata"`     // 元数据
}

// DatabaseConnectionInfo 数据库连接信息
type DatabaseConnectionInfo struct {
	Type     string `json:"type"`     // mysql, postgresql, sqlserver, etc.
	Host     string `json:"host"`     // 主机地址
	Port     int    `json:"port"`     // 端口
	Database string `json:"database"` // 数据库名
	Username string `json:"username"` // 用户名
	Schema   string `json:"schema"`   // 模式名
}

// MessageQueueParser 消息队列协议解析器接口
type MessageQueueParser interface {
	ProtocolParser

	// ParseMessage 解析消息
	ParseMessage(data []byte) (*QueueMessage, error)

	// ParseCommand 解析命令
	ParseCommand(data []byte) (*QueueCommand, error)

	// GetTopicInfo 获取主题信息
	GetTopicInfo() *TopicInfo
}

// QueueMessage 队列消息
type QueueMessage struct {
	ID          string                 `json:"id"`           // 消息ID
	Topic       string                 `json:"topic"`        // 主题/队列名
	Partition   int32                  `json:"partition"`    // 分区（Kafka）
	Offset      int64                  `json:"offset"`       // 偏移量（Kafka）
	Key         []byte                 `json:"key"`          // 消息键
	Value       []byte                 `json:"value"`        // 消息值
	Headers     map[string]string      `json:"headers"`      // 消息头
	Timestamp   time.Time              `json:"timestamp"`    // 时间戳
	ContentType string                 `json:"content_type"` // 内容类型
	Metadata    map[string]interface{} `json:"metadata"`     // 元数据
}

// QueueCommand 队列命令
type QueueCommand struct {
	Type       string                 `json:"type"`       // PUBLISH, SUBSCRIBE, etc.
	Topic      string                 `json:"topic"`      // 主题名
	Parameters map[string]interface{} `json:"parameters"` // 命令参数
	Metadata   map[string]interface{} `json:"metadata"`   // 元数据
}

// TopicInfo 主题信息
type TopicInfo struct {
	Name       string `json:"name"`       // 主题名
	Partitions int32  `json:"partitions"` // 分区数
	Replicas   int32  `json:"replicas"`   // 副本数
	Type       string `json:"type"`       // 主题类型
}

// EmailParser 邮件协议解析器接口
type EmailParser interface {
	ProtocolParser

	// ParseEmail 解析邮件
	ParseEmail(data []byte) (*EmailMessage, error)

	// ParseCommand 解析邮件命令
	ParseCommand(data []byte) (*EmailCommand, error)

	// GetMailboxInfo 获取邮箱信息
	GetMailboxInfo() *MailboxInfo
}

// EmailMessage 邮件消息
type EmailMessage struct {
	MessageID   string                 `json:"message_id"`  // 消息ID
	From        []EmailAddress         `json:"from"`        // 发件人
	To          []EmailAddress         `json:"to"`          // 收件人
	CC          []EmailAddress         `json:"cc"`          // 抄送
	BCC         []EmailAddress         `json:"bcc"`         // 密送
	Subject     string                 `json:"subject"`     // 主题
	Body        string                 `json:"body"`        // 正文
	HTMLBody    string                 `json:"html_body"`   // HTML正文
	Attachments []EmailAttachment      `json:"attachments"` // 附件
	Headers     map[string]string      `json:"headers"`     // 邮件头
	Date        time.Time              `json:"date"`        // 日期
	Size        int64                  `json:"size"`        // 大小
	Metadata    map[string]interface{} `json:"metadata"`    // 元数据
}

// EmailAddress 邮件地址
type EmailAddress struct {
	Name    string `json:"name"`    // 显示名
	Address string `json:"address"` // 邮件地址
}

// EmailAttachment 邮件附件
type EmailAttachment struct {
	Name        string `json:"name"`         // 文件名
	ContentType string `json:"content_type"` // 内容类型
	Size        int64  `json:"size"`         // 大小
	Data        []byte `json:"data"`         // 数据
	Hash        string `json:"hash"`         // 哈希值
}

// EmailCommand 邮件命令
type EmailCommand struct {
	Type       string                 `json:"type"`       // HELO, MAIL, RCPT, DATA, etc.
	Parameters []string               `json:"parameters"` // 命令参数
	Response   string                 `json:"response"`   // 服务器响应
	Code       int                    `json:"code"`       // 响应代码
	Metadata   map[string]interface{} `json:"metadata"`   // 元数据
}

// MailboxInfo 邮箱信息
type MailboxInfo struct {
	Server   string `json:"server"`   // 服务器地址
	Port     int    `json:"port"`     // 端口
	Protocol string `json:"protocol"` // 协议类型 (SMTP, POP3, IMAP)
	Username string `json:"username"` // 用户名
	Mailbox  string `json:"mailbox"`  // 邮箱名
}

// FileTransferParser 文件传输协议解析器接口
type FileTransferParser interface {
	ProtocolParser

	// ParseCommand 解析文件传输命令
	ParseCommand(data []byte) (*FileTransferCommand, error)

	// ParseData 解析文件数据
	ParseData(data []byte) (*FileTransferData, error)

	// GetTransferInfo 获取传输信息
	GetTransferInfo() *FileTransferInfo
}

// FileTransferCommand 文件传输命令
type FileTransferCommand struct {
	Type       string                 `json:"type"`       // GET, PUT, LIST, etc.
	Path       string                 `json:"path"`       // 文件路径
	Parameters map[string]string      `json:"parameters"` // 命令参数
	Response   string                 `json:"response"`   // 服务器响应
	Code       int                    `json:"code"`       // 响应代码
	Metadata   map[string]interface{} `json:"metadata"`   // 元数据
}

// FileTransferData 文件传输数据
type FileTransferData struct {
	FileName    string                 `json:"file_name"`    // 文件名
	FileSize    int64                  `json:"file_size"`    // 文件大小
	Data        []byte                 `json:"data"`         // 文件数据
	Offset      int64                  `json:"offset"`       // 数据偏移
	IsComplete  bool                   `json:"is_complete"`  // 是否传输完成
	Hash        string                 `json:"hash"`         // 文件哈希
	ContentType string                 `json:"content_type"` // 内容类型
	Metadata    map[string]interface{} `json:"metadata"`     // 元数据
}

// FileTransferInfo 文件传输信息
type FileTransferInfo struct {
	Protocol    string `json:"protocol"`     // FTP, SFTP, SMB, etc.
	Server      string `json:"server"`       // 服务器地址
	Port        int    `json:"port"`         // 端口
	Username    string `json:"username"`     // 用户名
	CurrentPath string `json:"current_path"` // 当前路径
	Mode        string `json:"mode"`         // 传输模式
}
