package selfprotect

import (
	"context"
	"time"
)

// ProtectionLevel 防护级别
type ProtectionLevel string

const (
	ProtectionLevelNone     ProtectionLevel = "none"     // 无防护
	ProtectionLevelBasic    ProtectionLevel = "basic"    // 基础防护
	ProtectionLevelStandard ProtectionLevel = "standard" // 标准防护
	ProtectionLevelStrict   ProtectionLevel = "strict"   // 严格防护
)

// ProtectionType 防护类型
type ProtectionType string

const (
	ProtectionTypeProcess  ProtectionType = "process"  // 进程防护
	ProtectionTypeFile     ProtectionType = "file"     // 文件防护
	ProtectionTypeRegistry ProtectionType = "registry" // 注册表防护
	ProtectionTypeService  ProtectionType = "service"  // 服务防护
)

// ProtectionConfig 防护配置
type ProtectionConfig struct {
	Enabled            bool                     `yaml:"enabled"`
	Level              ProtectionLevel          `yaml:"level"`
	EmergencyDisable   string                   `yaml:"emergency_disable"`
	CheckInterval      time.Duration            `yaml:"check_interval"`
	RestartDelay       time.Duration            `yaml:"restart_delay"`
	MaxRestartAttempts int                      `yaml:"max_restart_attempts"`
	Whitelist          WhitelistConfig          `yaml:"whitelist"`
	ProcessProtection  ProcessProtectionConfig  `yaml:"process_protection"`
	FileProtection     FileProtectionConfig     `yaml:"file_protection"`
	RegistryProtection RegistryProtectionConfig `yaml:"registry_protection"`
	ServiceProtection  ServiceProtectionConfig  `yaml:"service_protection"`
}

// WhitelistConfig 白名单配置
type WhitelistConfig struct {
	Enabled    bool     `yaml:"enabled"`
	Processes  []string `yaml:"processes"`
	Users      []string `yaml:"users"`
	Signatures []string `yaml:"signatures"`
}

// ProcessProtectionConfig 进程防护配置
type ProcessProtectionConfig struct {
	Enabled            bool     `yaml:"enabled"`
	ProtectedProcesses []string `yaml:"protected_processes"`
	MonitorChildren    bool     `yaml:"monitor_children"`
	PreventDebug       bool     `yaml:"prevent_debug"`
	PreventDump        bool     `yaml:"prevent_dump"`
}

// FileProtectionConfig 文件防护配置
type FileProtectionConfig struct {
	Enabled        bool     `yaml:"enabled"`
	ProtectedFiles []string `yaml:"protected_files"`
	ProtectedDirs  []string `yaml:"protected_dirs"`
	CheckIntegrity bool     `yaml:"check_integrity"`
	BackupEnabled  bool     `yaml:"backup_enabled"`
	BackupDir      string   `yaml:"backup_dir"`
}

// RegistryProtectionConfig 注册表防护配置
type RegistryProtectionConfig struct {
	Enabled        bool     `yaml:"enabled"`
	ProtectedKeys  []string `yaml:"protected_keys"`
	MonitorChanges bool     `yaml:"monitor_changes"`
}

// ServiceProtectionConfig 服务防护配置
type ServiceProtectionConfig struct {
	Enabled        bool   `yaml:"enabled"`
	ServiceName    string `yaml:"service_name"`
	AutoRestart    bool   `yaml:"auto_restart"`
	PreventDisable bool   `yaml:"prevent_disable"`
}

// ProtectionEvent 防护事件
type ProtectionEvent struct {
	ID          string                 `json:"id"`
	Type        ProtectionType         `json:"type"`
	Action      string                 `json:"action"`
	Target      string                 `json:"target"`
	Source      string                 `json:"source"`
	Timestamp   time.Time              `json:"timestamp"`
	Blocked     bool                   `json:"blocked"`
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details"`
}

// ProtectionStats 防护统计
type ProtectionStats struct {
	TotalEvents       int64     `json:"total_events"`
	BlockedEvents     int64     `json:"blocked_events"`
	ProcessEvents     int64     `json:"process_events"`
	FileEvents        int64     `json:"file_events"`
	RegistryEvents    int64     `json:"registry_events"`
	ServiceEvents     int64     `json:"service_events"`
	LastEvent         time.Time `json:"last_event"`
	StartTime         time.Time `json:"start_time"`
	ConfigHealthScore float64   `json:"config_health_score"`
	ConfigErrors      int64     `json:"config_errors"`
	HotReloadFailures int64     `json:"hot_reload_failures"`
	ActiveAlerts      int64     `json:"active_alerts"`
}

// ServiceStatus 服务状态
type ServiceStatus struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	State       string `json:"state"`
	StartType   string `json:"start_type"`
	ProcessID   uint32 `json:"process_id"`
}

// EventCallback 事件回调函数类型
type EventCallback func(event ProtectionEvent)

// Protector 防护器接口
type Protector interface {
	// Start 启动防护
	Start(ctx context.Context) error

	// Stop 停止防护
	Stop() error

	// IsEnabled 检查是否启用
	IsEnabled() bool

	// PeriodicCheck 定期检查
	PeriodicCheck() error

	// SetEventCallback 设置事件回调
	SetEventCallback(callback EventCallback)
}

// ProcessProtector 进程防护器接口
type ProcessProtector interface {
	Protector

	// ProtectProcess 保护进程
	ProtectProcess(processName string) error

	// UnprotectProcess 取消保护进程
	UnprotectProcess(processName string) error

	// IsProcessProtected 检查进程是否受保护
	IsProcessProtected(processName string) bool

	// GetProtectedProcesses 获取受保护的进程列表
	GetProtectedProcesses() []string

	// RestartProcess 重启进程
	RestartProcess(processName string) error

	// PreventProcessTermination 防止进程终止
	PreventProcessTermination(processID uint32) error

	// PreventProcessDebug 防止进程调试
	PreventProcessDebug(processID uint32) error
}

// FileProtector 文件防护器接口
type FileProtector interface {
	Protector

	// ProtectFile 保护文件
	ProtectFile(filePath string) error

	// UnprotectFile 取消保护文件
	UnprotectFile(filePath string) error

	// IsFileProtected 检查文件是否受保护
	IsFileProtected(filePath string) bool

	// GetProtectedFiles 获取受保护的文件列表
	GetProtectedFiles() []string

	// CheckFileIntegrity 检查文件完整性
	CheckFileIntegrity(filePath string) (bool, error)

	// BackupFile 备份文件
	BackupFile(filePath string) (string, error)

	// RestoreFile 恢复文件
	RestoreFile(filePath string) error

	// MonitorFileChanges 监控文件变更
	MonitorFileChanges(filePath string) error
}

// RegistryProtector 注册表防护器接口
type RegistryProtector interface {
	Protector

	// ProtectRegistryKey 保护注册表键
	ProtectRegistryKey(keyPath string) error

	// UnprotectRegistryKey 取消保护注册表键
	UnprotectRegistryKey(keyPath string) error

	// IsRegistryKeyProtected 检查注册表键是否受保护
	IsRegistryKeyProtected(keyPath string) bool

	// GetProtectedRegistryKeys 获取受保护的注册表键列表
	GetProtectedRegistryKeys() []string

	// BackupRegistryKey 备份注册表键
	BackupRegistryKey(keyPath string) error

	// RestoreRegistryKey 恢复注册表键
	RestoreRegistryKey(keyPath string) error

	// MonitorRegistryChanges 监控注册表变更
	MonitorRegistryChanges(keyPath string) error
}

// ServiceProtector 服务防护器接口
type ServiceProtector interface {
	Protector

	// ProtectService 保护服务
	ProtectService(serviceName string) error

	// UnprotectService 取消保护服务
	UnprotectService(serviceName string) error

	// IsServiceProtected 检查服务是否受保护
	IsServiceProtected(serviceName string) bool

	// GetProtectedServices 获取受保护的服务列表
	GetProtectedServices() []string

	// RestartService 重启服务
	RestartService(serviceName string) error

	// PreventServiceStop 防止服务停止
	PreventServiceStop(serviceName string) error

	// PreventServiceDisable 防止服务禁用
	PreventServiceDisable(serviceName string) error

	// GetServiceStatus 获取服务状态
	GetServiceStatus(serviceName string) (ServiceStatus, error)
}
