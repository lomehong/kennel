//go:build selfprotect
// +build selfprotect

package selfprotect

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v2"
)

// LoadProtectionConfigFromYAML 从YAML配置中加载自我防护配置
func LoadProtectionConfigFromYAML(yamlData []byte) (*ProtectionConfig, error) {
	var config struct {
		SelfProtection ProtectionConfigYAML `yaml:"self_protection"`
	}

	if err := yaml.Unmarshal(yamlData, &config); err != nil {
		return nil, fmt.Errorf("解析YAML配置失败: %w", err)
	}

	return convertYAMLToProtectionConfig(config.SelfProtection)
}

// ProtectionConfigYAML YAML配置结构
type ProtectionConfigYAML struct {
	Enabled            bool                         `yaml:"enabled"`
	Level              string                       `yaml:"level"`
	EmergencyDisable   string                       `yaml:"emergency_disable"`
	CheckInterval      string                       `yaml:"check_interval"`
	RestartDelay       string                       `yaml:"restart_delay"`
	MaxRestartAttempts int                          `yaml:"max_restart_attempts"`
	Whitelist          WhitelistConfigYAML          `yaml:"whitelist"`
	ProcessProtection  ProcessProtectionConfigYAML  `yaml:"process_protection"`
	FileProtection     FileProtectionConfigYAML     `yaml:"file_protection"`
	RegistryProtection RegistryProtectionConfigYAML `yaml:"registry_protection"`
	ServiceProtection  ServiceProtectionConfigYAML  `yaml:"service_protection"`
}

// WhitelistConfigYAML 白名单配置YAML结构
type WhitelistConfigYAML struct {
	Enabled    bool     `yaml:"enabled"`
	Processes  []string `yaml:"processes"`
	Users      []string `yaml:"users"`
	Signatures []string `yaml:"signatures"`
}

// ProcessProtectionConfigYAML 进程防护配置YAML结构
type ProcessProtectionConfigYAML struct {
	Enabled            bool     `yaml:"enabled"`
	ProtectedProcesses []string `yaml:"protected_processes"`
	MonitorChildren    bool     `yaml:"monitor_children"`
	PreventDebug       bool     `yaml:"prevent_debug"`
	PreventDump        bool     `yaml:"prevent_dump"`
}

// FileProtectionConfigYAML 文件防护配置YAML结构
type FileProtectionConfigYAML struct {
	Enabled        bool     `yaml:"enabled"`
	ProtectedFiles []string `yaml:"protected_files"`
	ProtectedDirs  []string `yaml:"protected_dirs"`
	CheckIntegrity bool     `yaml:"check_integrity"`
	BackupEnabled  bool     `yaml:"backup_enabled"`
	BackupDir      string   `yaml:"backup_dir"`
}

// RegistryProtectionConfigYAML 注册表防护配置YAML结构
type RegistryProtectionConfigYAML struct {
	Enabled        bool     `yaml:"enabled"`
	ProtectedKeys  []string `yaml:"protected_keys"`
	MonitorChanges bool     `yaml:"monitor_changes"`
}

// ServiceProtectionConfigYAML 服务防护配置YAML结构
type ServiceProtectionConfigYAML struct {
	Enabled        bool   `yaml:"enabled"`
	ServiceName    string `yaml:"service_name"`
	AutoRestart    bool   `yaml:"auto_restart"`
	PreventDisable bool   `yaml:"prevent_disable"`
}

// convertYAMLToProtectionConfig 将YAML配置转换为防护配置
func convertYAMLToProtectionConfig(yamlConfig ProtectionConfigYAML) (*ProtectionConfig, error) {
	// 解析时间间隔
	checkInterval, err := time.ParseDuration(yamlConfig.CheckInterval)
	if err != nil {
		checkInterval = 5 * time.Second
	}

	restartDelay, err := time.ParseDuration(yamlConfig.RestartDelay)
	if err != nil {
		restartDelay = 3 * time.Second
	}

	// 解析防护级别
	var level ProtectionLevel
	switch yamlConfig.Level {
	case "none":
		level = ProtectionLevelNone
	case "basic":
		level = ProtectionLevelBasic
	case "standard":
		level = ProtectionLevelStandard
	case "strict":
		level = ProtectionLevelStrict
	default:
		level = ProtectionLevelBasic
	}

	config := &ProtectionConfig{
		Enabled:            yamlConfig.Enabled,
		Level:              level,
		EmergencyDisable:   yamlConfig.EmergencyDisable,
		CheckInterval:      checkInterval,
		RestartDelay:       restartDelay,
		MaxRestartAttempts: yamlConfig.MaxRestartAttempts,
		Whitelist: WhitelistConfig{
			Enabled:    yamlConfig.Whitelist.Enabled,
			Processes:  yamlConfig.Whitelist.Processes,
			Users:      yamlConfig.Whitelist.Users,
			Signatures: yamlConfig.Whitelist.Signatures,
		},
		ProcessProtection: ProcessProtectionConfig{
			Enabled:            yamlConfig.ProcessProtection.Enabled,
			ProtectedProcesses: yamlConfig.ProcessProtection.ProtectedProcesses,
			MonitorChildren:    yamlConfig.ProcessProtection.MonitorChildren,
			PreventDebug:       yamlConfig.ProcessProtection.PreventDebug,
			PreventDump:        yamlConfig.ProcessProtection.PreventDump,
		},
		FileProtection: FileProtectionConfig{
			Enabled:        yamlConfig.FileProtection.Enabled,
			ProtectedFiles: yamlConfig.FileProtection.ProtectedFiles,
			ProtectedDirs:  yamlConfig.FileProtection.ProtectedDirs,
			CheckIntegrity: yamlConfig.FileProtection.CheckIntegrity,
			BackupEnabled:  yamlConfig.FileProtection.BackupEnabled,
			BackupDir:      yamlConfig.FileProtection.BackupDir,
		},
		RegistryProtection: RegistryProtectionConfig{
			Enabled:        yamlConfig.RegistryProtection.Enabled,
			ProtectedKeys:  yamlConfig.RegistryProtection.ProtectedKeys,
			MonitorChanges: yamlConfig.RegistryProtection.MonitorChanges,
		},
		ServiceProtection: ServiceProtectionConfig{
			Enabled:        yamlConfig.ServiceProtection.Enabled,
			ServiceName:    yamlConfig.ServiceProtection.ServiceName,
			AutoRestart:    yamlConfig.ServiceProtection.AutoRestart,
			PreventDisable: yamlConfig.ServiceProtection.PreventDisable,
		},
	}

	// 设置默认值
	if config.EmergencyDisable == "" {
		config.EmergencyDisable = ".emergency_disable"
	}

	if config.MaxRestartAttempts == 0 {
		config.MaxRestartAttempts = 3
	}

	if config.FileProtection.BackupDir == "" {
		config.FileProtection.BackupDir = "backup"
	}

	return config, nil
}

// ValidateProtectionConfig 验证防护配置
func ValidateProtectionConfig(config *ProtectionConfig) error {
	if config == nil {
		return fmt.Errorf("配置不能为空")
	}

	// 验证防护级别
	switch config.Level {
	case ProtectionLevelNone, ProtectionLevelBasic, ProtectionLevelStandard, ProtectionLevelStrict:
		// 有效级别
	default:
		return fmt.Errorf("无效的防护级别: %s", config.Level)
	}

	// 验证时间间隔
	if config.CheckInterval < time.Second {
		return fmt.Errorf("检查间隔不能小于1秒")
	}

	if config.RestartDelay < 0 {
		return fmt.Errorf("重启延迟不能为负数")
	}

	// 验证重启尝试次数
	if config.MaxRestartAttempts < 0 {
		return fmt.Errorf("最大重启尝试次数不能为负数")
	}

	// 验证进程防护配置
	if config.ProcessProtection.Enabled {
		if len(config.ProcessProtection.ProtectedProcesses) == 0 {
			return fmt.Errorf("启用进程防护时必须指定受保护的进程")
		}
	}

	// 验证文件防护配置
	if config.FileProtection.Enabled {
		if len(config.FileProtection.ProtectedFiles) == 0 && len(config.FileProtection.ProtectedDirs) == 0 {
			return fmt.Errorf("启用文件防护时必须指定受保护的文件或目录")
		}
	}

	// 验证注册表防护配置
	if config.RegistryProtection.Enabled {
		if len(config.RegistryProtection.ProtectedKeys) == 0 {
			return fmt.Errorf("启用注册表防护时必须指定受保护的注册表键")
		}
	}

	// 验证服务防护配置
	if config.ServiceProtection.Enabled {
		if config.ServiceProtection.ServiceName == "" {
			return fmt.Errorf("启用服务防护时必须指定服务名称")
		}
	}

	return nil
}

// MergeProtectionConfigs 合并防护配置
func MergeProtectionConfigs(base, override *ProtectionConfig) *ProtectionConfig {
	if base == nil {
		return override
	}
	if override == nil {
		return base
	}

	merged := *base

	// 合并基本配置
	if override.Enabled {
		merged.Enabled = override.Enabled
	}
	if override.Level != ProtectionLevelNone {
		merged.Level = override.Level
	}
	if override.EmergencyDisable != "" {
		merged.EmergencyDisable = override.EmergencyDisable
	}
	if override.CheckInterval > 0 {
		merged.CheckInterval = override.CheckInterval
	}
	if override.RestartDelay > 0 {
		merged.RestartDelay = override.RestartDelay
	}
	if override.MaxRestartAttempts > 0 {
		merged.MaxRestartAttempts = override.MaxRestartAttempts
	}

	// 合并白名单配置
	if override.Whitelist.Enabled {
		merged.Whitelist.Enabled = override.Whitelist.Enabled
	}
	if len(override.Whitelist.Processes) > 0 {
		merged.Whitelist.Processes = append(merged.Whitelist.Processes, override.Whitelist.Processes...)
	}
	if len(override.Whitelist.Users) > 0 {
		merged.Whitelist.Users = append(merged.Whitelist.Users, override.Whitelist.Users...)
	}
	if len(override.Whitelist.Signatures) > 0 {
		merged.Whitelist.Signatures = append(merged.Whitelist.Signatures, override.Whitelist.Signatures...)
	}

	// 合并进程防护配置
	if override.ProcessProtection.Enabled {
		merged.ProcessProtection.Enabled = override.ProcessProtection.Enabled
	}
	if len(override.ProcessProtection.ProtectedProcesses) > 0 {
		merged.ProcessProtection.ProtectedProcesses = append(merged.ProcessProtection.ProtectedProcesses, override.ProcessProtection.ProtectedProcesses...)
	}
	merged.ProcessProtection.MonitorChildren = override.ProcessProtection.MonitorChildren
	merged.ProcessProtection.PreventDebug = override.ProcessProtection.PreventDebug
	merged.ProcessProtection.PreventDump = override.ProcessProtection.PreventDump

	// 合并文件防护配置
	if override.FileProtection.Enabled {
		merged.FileProtection.Enabled = override.FileProtection.Enabled
	}
	if len(override.FileProtection.ProtectedFiles) > 0 {
		merged.FileProtection.ProtectedFiles = append(merged.FileProtection.ProtectedFiles, override.FileProtection.ProtectedFiles...)
	}
	if len(override.FileProtection.ProtectedDirs) > 0 {
		merged.FileProtection.ProtectedDirs = append(merged.FileProtection.ProtectedDirs, override.FileProtection.ProtectedDirs...)
	}
	merged.FileProtection.CheckIntegrity = override.FileProtection.CheckIntegrity
	merged.FileProtection.BackupEnabled = override.FileProtection.BackupEnabled
	if override.FileProtection.BackupDir != "" {
		merged.FileProtection.BackupDir = override.FileProtection.BackupDir
	}

	// 合并注册表防护配置
	if override.RegistryProtection.Enabled {
		merged.RegistryProtection.Enabled = override.RegistryProtection.Enabled
	}
	if len(override.RegistryProtection.ProtectedKeys) > 0 {
		merged.RegistryProtection.ProtectedKeys = append(merged.RegistryProtection.ProtectedKeys, override.RegistryProtection.ProtectedKeys...)
	}
	merged.RegistryProtection.MonitorChanges = override.RegistryProtection.MonitorChanges

	// 合并服务防护配置
	if override.ServiceProtection.Enabled {
		merged.ServiceProtection.Enabled = override.ServiceProtection.Enabled
	}
	if override.ServiceProtection.ServiceName != "" {
		merged.ServiceProtection.ServiceName = override.ServiceProtection.ServiceName
	}
	merged.ServiceProtection.AutoRestart = override.ServiceProtection.AutoRestart
	merged.ServiceProtection.PreventDisable = override.ServiceProtection.PreventDisable

	return &merged
}

// GetProtectionConfigSummary 获取防护配置摘要
func GetProtectionConfigSummary(config *ProtectionConfig) map[string]interface{} {
	if config == nil {
		return map[string]interface{}{
			"enabled": false,
			"level":   "none",
		}
	}

	return map[string]interface{}{
		"enabled":                 config.Enabled,
		"level":                   config.Level,
		"emergency_disable":       config.EmergencyDisable,
		"check_interval":          config.CheckInterval.String(),
		"restart_delay":           config.RestartDelay.String(),
		"max_restart_attempts":    config.MaxRestartAttempts,
		"process_protection":      config.ProcessProtection.Enabled,
		"file_protection":         config.FileProtection.Enabled,
		"registry_protection":     config.RegistryProtection.Enabled,
		"service_protection":      config.ServiceProtection.Enabled,
		"protected_processes":     len(config.ProcessProtection.ProtectedProcesses),
		"protected_files":         len(config.FileProtection.ProtectedFiles),
		"protected_dirs":          len(config.FileProtection.ProtectedDirs),
		"protected_registry_keys": len(config.RegistryProtection.ProtectedKeys),
		"whitelist_enabled":       config.Whitelist.Enabled,
		"whitelist_processes":     len(config.Whitelist.Processes),
		"whitelist_users":         len(config.Whitelist.Users),
	}
}
