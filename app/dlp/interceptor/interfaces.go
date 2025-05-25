package interceptor

import (
	"context"
	"net"
	"time"

	"github.com/lomehong/kennel/pkg/logging"
)

// PacketDirection 数据包方向
type PacketDirection int

const (
	PacketDirectionInbound PacketDirection = iota
	PacketDirectionOutbound
)

// Protocol 协议类型
type Protocol int

const (
	ProtocolTCP Protocol = 6
	ProtocolUDP Protocol = 17
)

// PacketInfo 数据包信息
type PacketInfo struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	Direction   PacketDirection        `json:"direction"`
	Protocol    Protocol               `json:"protocol"`
	SourceIP    net.IP                 `json:"source_ip"`
	DestIP      net.IP                 `json:"dest_ip"`
	SourcePort  uint16                 `json:"source_port"`
	DestPort    uint16                 `json:"dest_port"`
	Payload     []byte                 `json:"payload"`
	Size        int                    `json:"size"`
	Metadata    map[string]interface{} `json:"metadata"`
	ProcessInfo *ProcessInfo           `json:"process_info,omitempty"`
}

// ProcessInfo 进程信息
type ProcessInfo struct {
	PID         int    `json:"pid"`
	ProcessName string `json:"process_name"`
	ExecutePath string `json:"execute_path"`
	User        string `json:"user"`
	CommandLine string `json:"command_line"`
}

// InterceptorStats 拦截器统计信息
type InterceptorStats struct {
	PacketsProcessed uint64        `json:"packets_processed"`
	PacketsDropped   uint64        `json:"packets_dropped"`
	PacketsReinject  uint64        `json:"packets_reinject"`
	BytesProcessed   uint64        `json:"bytes_processed"`
	ErrorCount       uint64        `json:"error_count"`
	LastError        error         `json:"last_error,omitempty"`
	StartTime        time.Time     `json:"start_time"`
	Uptime           time.Duration `json:"uptime"`
}

// InterceptorMode 拦截器模式
type InterceptorMode int

const (
	// ModeMonitorOnly 仅监控模式 - 不阻断流量，只记录和分析
	ModeMonitorOnly InterceptorMode = iota
	// ModeInterceptAndAllow 拦截并允许模式 - 拦截分析后自动放行
	ModeInterceptAndAllow
	// ModeInterceptAndBlock 拦截并阻断模式 - 根据策略决定是否阻断
	ModeInterceptAndBlock
)

// InterceptorConfig 拦截器配置
type InterceptorConfig struct {
	Filter       string          `yaml:"filter" json:"filter"`
	BufferSize   int             `yaml:"buffer_size" json:"buffer_size"`
	ChannelSize  int             `yaml:"channel_size" json:"channel_size"`
	Priority     int16           `yaml:"priority" json:"priority"`
	Flags        uint64          `yaml:"flags" json:"flags"`
	QueueLen     uint64          `yaml:"queue_len" json:"queue_len"`
	QueueTime    uint64          `yaml:"queue_time" json:"queue_time"`
	WorkerCount  int             `yaml:"worker_count" json:"worker_count"`
	CacheSize    int             `yaml:"cache_size" json:"cache_size"`
	Interface    string          `yaml:"interface" json:"interface"`
	BypassCIDR   string          `yaml:"bypass_cidr" json:"bypass_cidr"`
	ProxyPort    int             `yaml:"proxy_port" json:"proxy_port"`
	Mode         InterceptorMode `yaml:"mode" json:"mode"`                   // 拦截器模式
	AutoReinject bool            `yaml:"auto_reinject" json:"auto_reinject"` // 自动重新注入
	Logger       logging.Logger  `yaml:"-" json:"-"`
}

// DefaultInterceptorConfig 返回默认拦截器配置（性能优化版本）
func DefaultInterceptorConfig() InterceptorConfig {
	return InterceptorConfig{
		// 优化过滤器：排除本地和私有网络流量，只监控公网流量
		Filter: "outbound and " +
			"(tcp.DstPort == 80 or tcp.DstPort == 443 or tcp.DstPort == 21 or tcp.DstPort == 25 or tcp.DstPort == 3306) and " +
			"not (ip.DstAddr >= 127.0.0.0 and ip.DstAddr <= 127.255.255.255) and " + // 排除本地回环
			"not (ip.DstAddr >= 10.0.0.0 and ip.DstAddr <= 10.255.255.255) and " + // 排除私有网络A类
			"not (ip.DstAddr >= 172.16.0.0 and ip.DstAddr <= 172.31.255.255) and " + // 排除私有网络B类
			"not (ip.DstAddr >= 192.168.0.0 and ip.DstAddr <= 192.168.255.255) and " + // 排除私有网络C类
			"not (ip.DstAddr >= 169.254.0.0 and ip.DstAddr <= 169.254.255.255)", // 排除链路本地地址
		BufferSize:   32768, // 减小缓冲区，降低内存占用
		ChannelSize:  500,   // 减小通道大小，避免积压过多数据包
		Priority:     0,
		Flags:        1,    // WINDIVERT_FLAG_SNIFF - 嗅探模式，不阻断流量
		QueueLen:     4096, // 减小队列长度，提高响应速度
		QueueTime:    1000, // 减小队列时间，快速处理数据包
		WorkerCount:  2,    // 减少工作协程数，降低CPU占用
		CacheSize:    500,  // 减小缓存大小
		Interface:    "en0",
		BypassCIDR:   "127.0.0.0/8,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,169.254.0.0/16", // 绕过本地和私有网络
		ProxyPort:    8080,
		Mode:         ModeMonitorOnly, // 默认使用监控模式
		AutoReinject: true,            // 自动重新注入数据包
	}
}

// TrafficInterceptor 统一的流量拦截接口
type TrafficInterceptor interface {
	// Initialize 初始化拦截器
	Initialize(config InterceptorConfig) error

	// Start 启动流量拦截
	Start() error

	// Stop 停止流量拦截
	Stop() error

	// SetFilter 设置过滤规则
	SetFilter(filter string) error

	// GetPacketChannel 获取数据包通道
	GetPacketChannel() <-chan *PacketInfo

	// Reinject 重新注入数据包
	Reinject(packet *PacketInfo) error

	// GetStats 获取统计信息
	GetStats() InterceptorStats

	// HealthCheck 健康检查
	HealthCheck() error
}

// InterceptorFactory 拦截器工厂接口
type InterceptorFactory interface {
	// CreateInterceptor 创建拦截器
	CreateInterceptor(platform string) (TrafficInterceptor, error)

	// GetSupportedPlatforms 获取支持的平台
	GetSupportedPlatforms() []string
}

// ProcessCache 进程缓存接口
type ProcessCache interface {
	// Get 获取进程信息
	Get(pid uint32) *ProcessInfo

	// Set 设置进程信息
	Set(pid uint32, info *ProcessInfo)

	// Delete 删除进程信息
	Delete(pid uint32)

	// Clear 清空缓存
	Clear()

	// Size 获取缓存大小
	Size() int
}

// PacketFilter 数据包过滤器接口
type PacketFilter interface {
	// Match 检查数据包是否匹配过滤条件
	Match(packet *PacketInfo) bool

	// SetRules 设置过滤规则
	SetRules(rules []string) error

	// GetRules 获取过滤规则
	GetRules() []string
}

// PacketProcessor 数据包处理器接口
type PacketProcessor interface {
	// Process 处理数据包
	Process(ctx context.Context, packet *PacketInfo) error

	// SetNext 设置下一个处理器
	SetNext(processor PacketProcessor)

	// GetNext 获取下一个处理器
	GetNext() PacketProcessor
}

// InterceptorManager 拦截器管理器接口
type InterceptorManager interface {
	// RegisterInterceptor 注册拦截器
	RegisterInterceptor(name string, interceptor TrafficInterceptor) error

	// GetInterceptor 获取拦截器
	GetInterceptor(name string) (TrafficInterceptor, bool)

	// StartAll 启动所有拦截器
	StartAll() error

	// StopAll 停止所有拦截器
	StopAll() error

	// GetStats 获取所有拦截器统计信息
	GetStats() map[string]InterceptorStats
}

// ===== ETW 相关类型定义 =====

// ConnectionState 连接状态
type ConnectionState int

const (
	ConnectionStateEstablished ConnectionState = iota
	ConnectionStateConnecting
	ConnectionStateClosing
	ConnectionStateClosed
)

func (s ConnectionState) String() string {
	switch s {
	case ConnectionStateEstablished:
		return "established"
	case ConnectionStateConnecting:
		return "connecting"
	case ConnectionStateClosing:
		return "closing"
	case ConnectionStateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// ConnectionInfo 连接信息
type ConnectionInfo struct {
	Protocol   Protocol
	LocalAddr  *net.TCPAddr
	RemoteAddr *net.TCPAddr
	State      ConnectionState
	Timestamp  time.Time
	ProcessID  uint32
}

// NetworkEventType 网络事件类型
type NetworkEventType int

const (
	NetworkEventTypeConnect NetworkEventType = iota
	NetworkEventTypeDisconnect
	NetworkEventTypeAccept
	NetworkEventTypeClose
)

func (t NetworkEventType) String() string {
	switch t {
	case NetworkEventTypeConnect:
		return "connect"
	case NetworkEventTypeDisconnect:
		return "disconnect"
	case NetworkEventTypeAccept:
		return "accept"
	case NetworkEventTypeClose:
		return "close"
	default:
		return "unknown"
	}
}

// ETWNetworkEvent ETW网络事件数据
type ETWNetworkEvent struct {
	EventType   NetworkEventType
	ProcessID   uint32
	ThreadID    uint32
	Connection  *ConnectionInfo
	Timestamp   time.Time
	ProcessName string
	ProcessPath string
}

// ProcessDataSource 进程数据源接口
type ProcessDataSource interface {
	GetProcessInfo(packet *PacketInfo) *ProcessInfo
	Priority() int
	Name() string
}

// ETWNetworkMonitor ETW网络事件监听器接口
type ETWNetworkMonitor interface {
	Start() error
	Stop() error
	GetConnectionMapping(localAddr, remoteAddr net.Addr) *ProcessInfo
	IsRunning() bool
	GetEventChannel() <-chan *ETWNetworkEvent
}

// ConnectionMapper 进程-连接映射管理器接口
type ConnectionMapper interface {
	AddMapping(conn *ConnectionInfo, proc *ProcessInfo)
	GetProcessByConnection(conn *ConnectionInfo) *ProcessInfo
	GetProcessByAddresses(protocol Protocol, localAddr, remoteAddr net.Addr) *ProcessInfo
	CleanExpiredMappings()
	GetMappingCount() int
}

// ProcessResolver 统一进程信息解析器接口
type ProcessResolver interface {
	ResolveProcess(packet *PacketInfo) *ProcessInfo
	RegisterDataSource(source ProcessDataSource)
	GetDataSources() []ProcessDataSource
}
