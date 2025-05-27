//go:build selfprotect && windows
// +build selfprotect,windows

package selfprotect

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"golang.org/x/sys/windows/registry"
)

// WindowsRegistryProtector Windows注册表防护器
type WindowsRegistryProtector struct {
	config        RegistryProtectionConfig
	logger        hclog.Logger
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	mu            sync.RWMutex
	
	enabled       bool
	protectedKeys map[string]*ProtectedRegistryKey
	eventCallback EventCallback
	
	// 监控状态
	monitoring    bool
	checkInterval time.Duration
}

// ProtectedRegistryKey 受保护的注册表键信息
type ProtectedRegistryKey struct {
	Path         string
	Root         registry.Key
	SubKey       string
	Protected    bool
	LastCheck    time.Time
	Backup       RegistryBackup
	Values       map[string]RegistryValue
}

// RegistryBackup 注册表备份
type RegistryBackup struct {
	Path      string
	Values    map[string]RegistryValue
	SubKeys   []string
	Timestamp time.Time
}

// RegistryValue 注册表值
type RegistryValue struct {
	Name     string
	Type     uint32
	Data     interface{}
	Size     uint32
}

// NewRegistryProtector 创建Windows注册表防护器
func NewRegistryProtector(config RegistryProtectionConfig, logger hclog.Logger) RegistryProtector {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &WindowsRegistryProtector{
		config:        config,
		logger:        logger.Named("registry-protector"),
		ctx:           ctx,
		cancel:        cancel,
		enabled:       config.Enabled,
		protectedKeys: make(map[string]*ProtectedRegistryKey),
		checkInterval: 10 * time.Second,
	}
}

// Start 启动注册表防护
func (rp *WindowsRegistryProtector) Start(ctx context.Context) error {
	if !rp.enabled {
		return nil
	}

	rp.logger.Info("启动Windows注册表防护")

	// 保护指定的注册表键
	for _, keyPath := range rp.config.ProtectedKeys {
		if err := rp.ProtectRegistryKey(keyPath); err != nil {
			rp.logger.Error("保护注册表键失败", "key", keyPath, "error", err)
		}
	}

	// 启动监控
	if rp.config.MonitorChanges {
		rp.monitoring = true
		rp.wg.Add(1)
		go rp.monitorRegistry()
	}

	return nil
}

// Stop 停止注册表防护
func (rp *WindowsRegistryProtector) Stop() error {
	rp.logger.Info("停止Windows注册表防护")
	
	rp.monitoring = false
	rp.cancel()
	rp.wg.Wait()

	return nil
}

// IsEnabled 检查是否启用
func (rp *WindowsRegistryProtector) IsEnabled() bool {
	return rp.enabled
}

// PeriodicCheck 定期检查
func (rp *WindowsRegistryProtector) PeriodicCheck() error {
	if !rp.enabled || !rp.monitoring {
		return nil
	}

	// 检查受保护注册表键的状态
	rp.mu.RLock()
	keys := make([]*ProtectedRegistryKey, 0, len(rp.protectedKeys))
	for _, key := range rp.protectedKeys {
		keys = append(keys, key)
	}
	rp.mu.RUnlock()

	for _, key := range keys {
		if err := rp.checkRegistryKeyStatus(key); err != nil {
			rp.logger.Error("检查注册表键状态失败", "key", key.Path, "error", err)
		}
	}

	return nil
}

// SetEventCallback 设置事件回调
func (rp *WindowsRegistryProtector) SetEventCallback(callback EventCallback) {
	rp.eventCallback = callback
}

// ProtectRegistryKey 保护注册表键
func (rp *WindowsRegistryProtector) ProtectRegistryKey(keyPath string) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	// 解析注册表键路径
	root, subKey, err := rp.parseRegistryPath(keyPath)
	if err != nil {
		return fmt.Errorf("解析注册表路径失败: %w", err)
	}

	// 检查注册表键是否存在
	key, err := registry.OpenKey(root, subKey, registry.READ)
	if err != nil {
		if err == registry.ErrNotExist {
			rp.logger.Warn("注册表键不存在", "key", keyPath)
			// 仍然添加到保护列表，以便监控键的创建
			rp.protectedKeys[keyPath] = &ProtectedRegistryKey{
				Path:      keyPath,
				Root:      root,
				SubKey:    subKey,
				Protected: true,
				LastCheck: time.Now(),
			}
			return nil
		}
		return fmt.Errorf("打开注册表键失败: %w", err)
	}
	defer key.Close()

	// 读取注册表键的值
	values, err := rp.readRegistryValues(key)
	if err != nil {
		return fmt.Errorf("读取注册表值失败: %w", err)
	}

	// 创建备份
	backup := RegistryBackup{
		Path:      keyPath,
		Values:    values,
		Timestamp: time.Now(),
	}

	// 添加到保护列表
	rp.protectedKeys[keyPath] = &ProtectedRegistryKey{
		Path:      keyPath,
		Root:      root,
		SubKey:    subKey,
		Protected: true,
		LastCheck: time.Now(),
		Backup:    backup,
		Values:    values,
	}

	rp.logger.Info("注册表键已保护", "key", keyPath)

	// 记录事件
	if rp.eventCallback != nil {
		rp.eventCallback(ProtectionEvent{
			Type:        ProtectionTypeRegistry,
			Action:      "protect",
			Target:      keyPath,
			Description: fmt.Sprintf("注册表键 %s 已被保护", keyPath),
			Details: map[string]interface{}{
				"key_path":    keyPath,
				"value_count": len(values),
			},
		})
	}

	return nil
}

// UnprotectRegistryKey 取消保护注册表键
func (rp *WindowsRegistryProtector) UnprotectRegistryKey(keyPath string) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	_, exists := rp.protectedKeys[keyPath]
	if !exists {
		return fmt.Errorf("注册表键未受保护: %s", keyPath)
	}

	// 从保护列表移除
	delete(rp.protectedKeys, keyPath)

	rp.logger.Info("取消注册表键保护", "key", keyPath)

	// 记录事件
	if rp.eventCallback != nil {
		rp.eventCallback(ProtectionEvent{
			Type:        ProtectionTypeRegistry,
			Action:      "unprotect",
			Target:      keyPath,
			Description: fmt.Sprintf("取消保护注册表键 %s", keyPath),
		})
	}

	return nil
}

// IsRegistryKeyProtected 检查注册表键是否受保护
func (rp *WindowsRegistryProtector) IsRegistryKeyProtected(keyPath string) bool {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	
	_, exists := rp.protectedKeys[keyPath]
	return exists
}

// GetProtectedRegistryKeys 获取受保护的注册表键列表
func (rp *WindowsRegistryProtector) GetProtectedRegistryKeys() []string {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	
	keys := make([]string, 0, len(rp.protectedKeys))
	for path := range rp.protectedKeys {
		keys = append(keys, path)
	}
	return keys
}

// BackupRegistryKey 备份注册表键
func (rp *WindowsRegistryProtector) BackupRegistryKey(keyPath string) error {
	rp.mu.RLock()
	protectedKey, exists := rp.protectedKeys[keyPath]
	rp.mu.RUnlock()

	if !exists {
		return fmt.Errorf("注册表键未受保护: %s", keyPath)
	}

	// 打开注册表键
	key, err := registry.OpenKey(protectedKey.Root, protectedKey.SubKey, registry.READ)
	if err != nil {
		return fmt.Errorf("打开注册表键失败: %w", err)
	}
	defer key.Close()

	// 读取当前值
	values, err := rp.readRegistryValues(key)
	if err != nil {
		return fmt.Errorf("读取注册表值失败: %w", err)
	}

	// 更新备份
	rp.mu.Lock()
	protectedKey.Backup = RegistryBackup{
		Path:      keyPath,
		Values:    values,
		Timestamp: time.Now(),
	}
	rp.mu.Unlock()

	rp.logger.Info("注册表键已备份", "key", keyPath)
	return nil
}

// RestoreRegistryKey 恢复注册表键
func (rp *WindowsRegistryProtector) RestoreRegistryKey(keyPath string) error {
	rp.mu.RLock()
	protectedKey, exists := rp.protectedKeys[keyPath]
	if !exists {
		rp.mu.RUnlock()
		return fmt.Errorf("注册表键未受保护: %s", keyPath)
	}

	backup := protectedKey.Backup
	rp.mu.RUnlock()

	if backup.Timestamp.IsZero() {
		return fmt.Errorf("注册表键无备份: %s", keyPath)
	}

	// 打开注册表键
	key, err := registry.OpenKey(protectedKey.Root, protectedKey.SubKey, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("打开注册表键失败: %w", err)
	}
	defer key.Close()

	// 恢复值
	for name, value := range backup.Values {
		if err := rp.setRegistryValue(key, name, value); err != nil {
			rp.logger.Error("恢复注册表值失败", "key", keyPath, "value", name, "error", err)
		}
	}

	rp.logger.Info("注册表键已恢复", "key", keyPath)

	// 记录事件
	if rp.eventCallback != nil {
		rp.eventCallback(ProtectionEvent{
			Type:        ProtectionTypeRegistry,
			Action:      "restore",
			Target:      keyPath,
			Description: fmt.Sprintf("注册表键 %s 已从备份恢复", keyPath),
			Details: map[string]interface{}{
				"key_path":      keyPath,
				"backup_time":   backup.Timestamp,
				"restored_values": len(backup.Values),
			},
		})
	}

	return nil
}

// MonitorRegistryChanges 监控注册表变更
func (rp *WindowsRegistryProtector) MonitorRegistryChanges(keyPath string) error {
	// Windows注册表监控需要使用RegNotifyChangeKeyValue API
	// 这里简化实现，实际应该使用Windows API
	rp.logger.Info("开始监控注册表变更", "key", keyPath)
	return nil
}

// parseRegistryPath 解析注册表路径
func (rp *WindowsRegistryProtector) parseRegistryPath(keyPath string) (registry.Key, string, error) {
	parts := strings.SplitN(keyPath, "\\", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("无效的注册表路径: %s", keyPath)
	}

	var root registry.Key
	switch strings.ToUpper(parts[0]) {
	case "HKEY_CLASSES_ROOT", "HKCR":
		root = registry.CLASSES_ROOT
	case "HKEY_CURRENT_USER", "HKCU":
		root = registry.CURRENT_USER
	case "HKEY_LOCAL_MACHINE", "HKLM":
		root = registry.LOCAL_MACHINE
	case "HKEY_USERS", "HKU":
		root = registry.USERS
	case "HKEY_CURRENT_CONFIG", "HKCC":
		root = registry.CURRENT_CONFIG
	default:
		return 0, "", fmt.Errorf("不支持的注册表根键: %s", parts[0])
	}

	return root, parts[1], nil
}

// readRegistryValues 读取注册表值
func (rp *WindowsRegistryProtector) readRegistryValues(key registry.Key) (map[string]RegistryValue, error) {
	values := make(map[string]RegistryValue)

	// 获取值名称列表
	valueNames, err := key.ReadValueNames(-1)
	if err != nil {
		return nil, err
	}

	// 读取每个值
	for _, name := range valueNames {
		value, valueType, err := key.GetValue(name, nil)
		if err != nil {
			rp.logger.Warn("读取注册表值失败", "name", name, "error", err)
			continue
		}

		values[name] = RegistryValue{
			Name: name,
			Type: valueType,
			Data: value,
		}
	}

	return values, nil
}

// setRegistryValue 设置注册表值
func (rp *WindowsRegistryProtector) setRegistryValue(key registry.Key, name string, value RegistryValue) error {
	switch value.Type {
	case registry.SZ:
		if str, ok := value.Data.(string); ok {
			return key.SetStringValue(name, str)
		}
	case registry.EXPAND_SZ:
		if str, ok := value.Data.(string); ok {
			return key.SetExpandStringValue(name, str)
		}
	case registry.DWORD:
		if dw, ok := value.Data.(uint32); ok {
			return key.SetDWordValue(name, dw)
		}
	case registry.QWORD:
		if qw, ok := value.Data.(uint64); ok {
			return key.SetQWordValue(name, qw)
		}
	case registry.BINARY:
		if bin, ok := value.Data.([]byte); ok {
			return key.SetBinaryValue(name, bin)
		}
	case registry.MULTI_SZ:
		if strs, ok := value.Data.([]string); ok {
			return key.SetStringsValue(name, strs)
		}
	}

	return fmt.Errorf("不支持的注册表值类型: %d", value.Type)
}

// checkRegistryKeyStatus 检查注册表键状态
func (rp *WindowsRegistryProtector) checkRegistryKeyStatus(protectedKey *ProtectedRegistryKey) error {
	// 打开注册表键
	key, err := registry.OpenKey(protectedKey.Root, protectedKey.SubKey, registry.READ)
	if err != nil {
		if err == registry.ErrNotExist {
			rp.logger.Warn("受保护的注册表键不存在", "key", protectedKey.Path)
			
			// 记录事件
			if rp.eventCallback != nil {
				rp.eventCallback(ProtectionEvent{
					Type:        ProtectionTypeRegistry,
					Action:      "deleted",
					Target:      protectedKey.Path,
					Description: fmt.Sprintf("受保护的注册表键 %s 已被删除", protectedKey.Path),
				})
			}

			// 尝试恢复
			return rp.RestoreRegistryKey(protectedKey.Path)
		}
		return err
	}
	defer key.Close()

	// 读取当前值
	currentValues, err := rp.readRegistryValues(key)
	if err != nil {
		return err
	}

	// 比较值是否发生变化
	changed := rp.compareRegistryValues(protectedKey.Values, currentValues)
	if len(changed) > 0 {
		rp.logger.Warn("检测到注册表键值变更", "key", protectedKey.Path, "changed", len(changed))
		
		// 记录事件
		if rp.eventCallback != nil {
			rp.eventCallback(ProtectionEvent{
				Type:        ProtectionTypeRegistry,
				Action:      "modified",
				Target:      protectedKey.Path,
				Description: fmt.Sprintf("注册表键 %s 的值已被修改", protectedKey.Path),
				Details: map[string]interface{}{
					"key_path":      protectedKey.Path,
					"changed_values": changed,
				},
			})
		}

		// 尝试恢复
		return rp.RestoreRegistryKey(protectedKey.Path)
	}

	// 更新最后检查时间
	protectedKey.LastCheck = time.Now()
	return nil
}

// compareRegistryValues 比较注册表值
func (rp *WindowsRegistryProtector) compareRegistryValues(original, current map[string]RegistryValue) []string {
	var changed []string

	// 检查原始值是否被修改或删除
	for name, originalValue := range original {
		currentValue, exists := current[name]
		if !exists {
			changed = append(changed, name+" (deleted)")
			continue
		}

		if originalValue.Type != currentValue.Type {
			changed = append(changed, name+" (type changed)")
			continue
		}

		// 简化的值比较
		if fmt.Sprintf("%v", originalValue.Data) != fmt.Sprintf("%v", currentValue.Data) {
			changed = append(changed, name+" (value changed)")
		}
	}

	// 检查新增的值
	for name := range current {
		if _, exists := original[name]; !exists {
			changed = append(changed, name+" (added)")
		}
	}

	return changed
}

// monitorRegistry 监控注册表
func (rp *WindowsRegistryProtector) monitorRegistry() {
	defer rp.wg.Done()

	ticker := time.NewTicker(rp.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rp.ctx.Done():
			return
		case <-ticker.C:
			if rp.monitoring {
				rp.PeriodicCheck()
			}
		}
	}
}
