package interceptor

import (
	"fmt"
	"sync"
	"time"

	"github.com/lomehong/kennel/pkg/logging"
)

// EnhancedProcessManager 增强的进程信息管理器
type EnhancedProcessManager struct {
	logger           logging.Logger
	processResolver  ProcessResolver
	etwMonitor       ETWNetworkMonitor
	connectionMapper ConnectionMapper
	etwDataSource    ProcessDataSource
	legacyDataSource ProcessDataSource
	
	// 组件状态
	running bool
	mu      sync.RWMutex
	
	// 配置
	config *ProcessManagerConfig
	
	// 统计信息
	stats struct {
		startTime       time.Time
		totalQueries    int64
		successQueries  int64
		failedQueries   int64
		lastQueryTime   time.Time
		mu              sync.RWMutex
	}
}

// ProcessManagerConfig 进程管理器配置
type ProcessManagerConfig struct {
	EnableETW                bool          `yaml:"enable_etw"`
	EnableLegacyFallback     bool          `yaml:"enable_legacy_fallback"`
	ETWBufferSize            int           `yaml:"etw_buffer_size"`
	ConnectionMapperMaxSize  int           `yaml:"connection_mapper_max_size"`
	ConnectionExpireTime     time.Duration `yaml:"connection_expire_time"`
	CacheExpireTime          time.Duration `yaml:"cache_expire_time"`
	LegacyUpdateInterval     time.Duration `yaml:"legacy_update_interval"`
	EnablePerformanceMonitor bool          `yaml:"enable_performance_monitor"`
}

// DefaultProcessManagerConfig 默认配置
func DefaultProcessManagerConfig() *ProcessManagerConfig {
	return &ProcessManagerConfig{
		EnableETW:                true,
		EnableLegacyFallback:     true,
		ETWBufferSize:            1000,
		ConnectionMapperMaxSize:  10000,
		ConnectionExpireTime:     5 * time.Minute,
		CacheExpireTime:          30 * time.Second,
		LegacyUpdateInterval:     10 * time.Second,
		EnablePerformanceMonitor: true,
	}
}

// NewEnhancedProcessManager 创建增强的进程信息管理器
func NewEnhancedProcessManager(logger logging.Logger, config *ProcessManagerConfig) (*EnhancedProcessManager, error) {
	if config == nil {
		config = DefaultProcessManagerConfig()
	}
	
	manager := &EnhancedProcessManager{
		logger: logger,
		config: config,
	}
	
	// 初始化组件
	if err := manager.initializeComponents(); err != nil {
		return nil, fmt.Errorf("初始化组件失败: %v", err)
	}
	
	return manager, nil
}

// initializeComponents 初始化所有组件
func (pm *EnhancedProcessManager) initializeComponents() error {
	pm.logger.Info("初始化增强进程管理器组件")
	
	// 1. 创建进程信息解析器
	pm.processResolver = NewProcessResolver(pm.logger)
	
	// 2. 创建连接映射管理器
	pm.connectionMapper = NewConnectionMapper(pm.logger)
	
	// 3. 如果启用ETW，创建ETW相关组件
	if pm.config.EnableETW {
		pm.logger.Info("启用ETW网络监听器")
		
		// 创建ETW监听器
		pm.etwMonitor = NewETWNetworkMonitor(pm.logger)
		
		// 创建ETW数据源
		pm.etwDataSource = NewETWDataSource(pm.logger, pm.etwMonitor, pm.connectionMapper)
		
		// 注册ETW数据源
		pm.processResolver.RegisterDataSource(pm.etwDataSource)
		
		pm.logger.Info("ETW组件初始化完成")
	} else {
		pm.logger.Info("ETW功能已禁用")
	}
	
	// 4. 如果启用Legacy备选方案，创建Legacy数据源
	if pm.config.EnableLegacyFallback {
		pm.logger.Info("启用Legacy备选数据源")
		
		pm.legacyDataSource = NewLegacyDataSource(pm.logger)
		
		// 注册Legacy数据源
		pm.processResolver.RegisterDataSource(pm.legacyDataSource)
		
		pm.logger.Info("Legacy数据源初始化完成")
	} else {
		pm.logger.Info("Legacy备选方案已禁用")
	}
	
	pm.logger.Info("所有组件初始化完成",
		"etw_enabled", pm.config.EnableETW,
		"legacy_enabled", pm.config.EnableLegacyFallback,
		"data_sources", len(pm.processResolver.GetDataSources()),
	)
	
	return nil
}

// Start 启动进程管理器
func (pm *EnhancedProcessManager) Start() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if pm.running {
		return fmt.Errorf("进程管理器已经在运行")
	}
	
	pm.logger.Info("启动增强进程管理器")
	
	// 启动ETW监听器
	if pm.etwMonitor != nil {
		if err := pm.etwMonitor.Start(); err != nil {
			pm.logger.Error("启动ETW监听器失败", "error", err)
			// 不返回错误，继续使用Legacy方案
		} else {
			pm.logger.Info("ETW监听器启动成功")
		}
	}
	
	pm.running = true
	pm.stats.startTime = time.Now()
	
	pm.logger.Info("增强进程管理器启动完成")
	return nil
}

// Stop 停止进程管理器
func (pm *EnhancedProcessManager) Stop() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if !pm.running {
		return nil
	}
	
	pm.logger.Info("停止增强进程管理器")
	
	// 停止ETW监听器
	if pm.etwMonitor != nil {
		if err := pm.etwMonitor.Stop(); err != nil {
			pm.logger.Error("停止ETW监听器失败", "error", err)
		}
	}
	
	// 停止ETW数据源
	if pm.etwDataSource != nil {
		if etwDS, ok := pm.etwDataSource.(*ETWDataSource); ok {
			if err := etwDS.Stop(); err != nil {
				pm.logger.Error("停止ETW数据源失败", "error", err)
			}
		}
	}
	
	// 停止Legacy数据源
	if pm.legacyDataSource != nil {
		if err := pm.legacyDataSource.(*LegacyDataSource).Stop(); err != nil {
			pm.logger.Error("停止Legacy数据源失败", "error", err)
		}
	}
	
	// 停止连接映射管理器
	if pm.connectionMapper != nil {
		if mapper, ok := pm.connectionMapper.(*ConnectionMapperImpl); ok {
			mapper.Stop()
		}
	}
	
	pm.running = false
	
	pm.logger.Info("增强进程管理器已停止")
	return nil
}

// GetProcessInfo 获取进程信息（主要接口）
func (pm *EnhancedProcessManager) GetProcessInfo(packet *PacketInfo) *ProcessInfo {
	pm.stats.mu.Lock()
	pm.stats.totalQueries++
	pm.stats.lastQueryTime = time.Now()
	pm.stats.mu.Unlock()
	
	if packet == nil {
		return nil
	}
	
	startTime := time.Now()
	
	// 使用进程解析器获取进程信息
	processInfo := pm.processResolver.ResolveProcess(packet)
	
	queryDuration := time.Since(startTime)
	
	if processInfo != nil {
		pm.stats.mu.Lock()
		pm.stats.successQueries++
		pm.stats.mu.Unlock()
		
		pm.logger.Debug("成功获取进程信息",
			"pid", processInfo.PID,
			"process_name", processInfo.ProcessName,
			"query_duration", queryDuration,
		)
	} else {
		pm.stats.mu.Lock()
		pm.stats.failedQueries++
		pm.stats.mu.Unlock()
		
		pm.logger.Debug("未能获取进程信息",
			"source_ip", packet.SourceIP.String(),
			"source_port", packet.SourcePort,
			"dest_ip", packet.DestIP.String(),
			"dest_port", packet.DestPort,
			"query_duration", queryDuration,
		)
	}
	
	return processInfo
}

// IsRunning 检查是否正在运行
func (pm *EnhancedProcessManager) IsRunning() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.running
}

// GetStats 获取统计信息
func (pm *EnhancedProcessManager) GetStats() map[string]interface{} {
	pm.stats.mu.RLock()
	defer pm.stats.mu.RUnlock()
	
	pm.mu.RLock()
	isRunning := pm.running
	pm.mu.RUnlock()
	
	successRate := float64(0)
	if pm.stats.totalQueries > 0 {
		successRate = float64(pm.stats.successQueries) / float64(pm.stats.totalQueries) * 100
	}
	
	uptime := time.Duration(0)
	if !pm.stats.startTime.IsZero() {
		uptime = time.Since(pm.stats.startTime)
	}
	
	stats := map[string]interface{}{
		"is_running":      isRunning,
		"start_time":      pm.stats.startTime,
		"uptime":          uptime,
		"total_queries":   pm.stats.totalQueries,
		"success_queries": pm.stats.successQueries,
		"failed_queries":  pm.stats.failedQueries,
		"success_rate":    successRate,
		"last_query_time": pm.stats.lastQueryTime,
		"config":          pm.config,
	}
	
	// 添加各组件的统计信息
	if pm.processResolver != nil {
		if resolver, ok := pm.processResolver.(*ProcessResolverImpl); ok {
			stats["process_resolver"] = resolver.GetStats()
		}
	}
	
	if pm.connectionMapper != nil {
		if mapper, ok := pm.connectionMapper.(*ConnectionMapperImpl); ok {
			stats["connection_mapper"] = mapper.GetStats()
		}
	}
	
	if pm.etwDataSource != nil {
		if etwDS, ok := pm.etwDataSource.(*ETWDataSource); ok {
			stats["etw_data_source"] = etwDS.GetStats()
		}
	}
	
	if pm.legacyDataSource != nil {
		if legacyDS, ok := pm.legacyDataSource.(*LegacyDataSource); ok {
			stats["legacy_data_source"] = legacyDS.GetStats()
		}
	}
	
	return stats
}

// GetDataSources 获取所有数据源
func (pm *EnhancedProcessManager) GetDataSources() []ProcessDataSource {
	if pm.processResolver == nil {
		return nil
	}
	return pm.processResolver.GetDataSources()
}

// ClearCache 清理所有缓存
func (pm *EnhancedProcessManager) ClearCache() {
	pm.logger.Info("清理所有缓存")
	
	if pm.processResolver != nil {
		if resolver, ok := pm.processResolver.(*ProcessResolverImpl); ok {
			resolver.ClearCache()
		}
	}
	
	if pm.legacyDataSource != nil {
		if legacyDS, ok := pm.legacyDataSource.(*LegacyDataSource); ok {
			legacyDS.ClearCache()
		}
	}
	
	pm.logger.Info("所有缓存已清理")
}
