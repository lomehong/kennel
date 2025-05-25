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
		// 使用嗅探模式，不阻断流量
		Filter:       "outbound and (tcp.DstPort == 80 or tcp.DstPort == 443 or tcp.DstPort == 21 or tcp.DstPort == 25 or tcp.DstPort == 3306)",
		BufferSize:   32768, // 减小缓冲区，降低内存占用
		ChannelSize:  500,   // 减小通道大小，避免积压过多数据包
		Priority:     0,
		Flags:        1,    // WINDIVERT_FLAG_SNIFF - 嗅探模式，不阻断流量
		QueueLen:     4096, // 减小队列长度，提高响应速度
		QueueTime:    1000, // 减小队列时间，快速处理数据包
		WorkerCount:  2,    // 减少工作协程数，降低CPU占用
		CacheSize:    500,  // 减小缓存大小
		Interface:    "en0",
		BypassCIDR:   "127.0.0.0/8,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16", // 绕过本地和私有网络
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
