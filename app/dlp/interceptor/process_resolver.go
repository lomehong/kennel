package interceptor

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/lomehong/kennel/pkg/logging"
)

// ProcessResolverImpl 统一进程信息解析器实现
type ProcessResolverImpl struct {
	logger      logging.Logger
	dataSources []ProcessDataSource
	mu          sync.RWMutex

	// 缓存
	cache           map[string]*CachedProcessInfo
	cacheMu         sync.RWMutex
	cacheExpireTime time.Duration

	// 统计信息
	stats struct {
		totalQueries  int64
		cacheHits     int64
		cacheMisses   int64
		sourceQueries map[string]int64
		lastQueryTime time.Time
		mu            sync.RWMutex
	}
}

// CachedProcessInfo 缓存的进程信息
type CachedProcessInfo struct {
	ProcessInfo *ProcessInfo
	Timestamp   time.Time
	Source      string
}

// NewProcessResolver 创建新的进程信息解析器
func NewProcessResolver(logger logging.Logger) ProcessResolver {
	pr := &ProcessResolverImpl{
		logger:          logger,
		dataSources:     make([]ProcessDataSource, 0),
		cache:           make(map[string]*CachedProcessInfo),
		cacheExpireTime: 30 * time.Second, // 缓存30秒
	}

	// 初始化统计信息
	pr.stats.sourceQueries = make(map[string]int64)

	return pr
}

// RegisterDataSource 注册数据源
func (pr *ProcessResolverImpl) RegisterDataSource(source ProcessDataSource) {
	if source == nil {
		pr.logger.Warn("尝试注册空的数据源")
		return
	}

	pr.mu.Lock()
	defer pr.mu.Unlock()

	// 检查是否已经注册
	for _, existing := range pr.dataSources {
		if existing.Name() == source.Name() {
			pr.logger.Warn("数据源已存在", "name", source.Name())
			return
		}
	}

	// 添加数据源
	pr.dataSources = append(pr.dataSources, source)

	// 按优先级排序（优先级高的在前）
	sort.Slice(pr.dataSources, func(i, j int) bool {
		return pr.dataSources[i].Priority() > pr.dataSources[j].Priority()
	})

	// 初始化统计信息
	pr.stats.mu.Lock()
	pr.stats.sourceQueries[source.Name()] = 0
	pr.stats.mu.Unlock()

	pr.logger.Info("注册进程数据源",
		"name", source.Name(),
		"priority", source.Priority(),
		"total_sources", len(pr.dataSources),
	)
}

// GetDataSources 获取所有数据源
func (pr *ProcessResolverImpl) GetDataSources() []ProcessDataSource {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	// 返回副本以避免并发修改
	sources := make([]ProcessDataSource, len(pr.dataSources))
	copy(sources, pr.dataSources)
	return sources
}

// ResolveProcess 解析进程信息
func (pr *ProcessResolverImpl) ResolveProcess(packet *PacketInfo) *ProcessInfo {
	if packet == nil {
		return nil
	}

	// 更新统计信息
	pr.stats.mu.Lock()
	pr.stats.totalQueries++
	pr.stats.lastQueryTime = time.Now()
	pr.stats.mu.Unlock()

	// 生成缓存键
	cacheKey := pr.generateCacheKey(packet)

	// 检查缓存
	if processInfo := pr.getFromCache(cacheKey); processInfo != nil {
		pr.stats.mu.Lock()
		pr.stats.cacheHits++
		pr.stats.mu.Unlock()

		pr.logger.Debug("从缓存获取进程信息",
			"cache_key", cacheKey,
			"pid", processInfo.PID,
			"process_name", processInfo.ProcessName,
		)
		return processInfo
	}

	// 缓存未命中，从数据源查询
	pr.stats.mu.Lock()
	pr.stats.cacheMisses++
	pr.stats.mu.Unlock()

	pr.mu.RLock()
	sources := make([]ProcessDataSource, len(pr.dataSources))
	copy(sources, pr.dataSources)
	pr.mu.RUnlock()

	// 按优先级顺序查询数据源
	for _, source := range sources {
		startTime := time.Now()
		processInfo := source.GetProcessInfo(packet)
		queryDuration := time.Since(startTime)

		// 更新数据源查询统计
		pr.stats.mu.Lock()
		pr.stats.sourceQueries[source.Name()]++
		pr.stats.mu.Unlock()

		if processInfo != nil && processInfo.PID > 0 {
			pr.logger.Debug("从数据源获取进程信息",
				"source", source.Name(),
				"priority", source.Priority(),
				"query_duration", queryDuration,
				"pid", processInfo.PID,
				"process_name", processInfo.ProcessName,
			)

			// 缓存结果
			pr.setCache(cacheKey, processInfo, source.Name())

			return processInfo
		}

		pr.logger.Debug("数据源未找到进程信息",
			"source", source.Name(),
			"query_duration", queryDuration,
		)
	}

	pr.logger.Debug("所有数据源都未找到进程信息", "cache_key", cacheKey)
	return nil
}

// generateCacheKey 生成缓存键
func (pr *ProcessResolverImpl) generateCacheKey(packet *PacketInfo) string {
	// 使用协议、源地址、目标地址和端口生成键
	return fmt.Sprintf("%d:%s:%d:%s:%d",
		packet.Protocol,
		packet.SourceIP.String(),
		packet.SourcePort,
		packet.DestIP.String(),
		packet.DestPort,
	)
}

// getFromCache 从缓存获取进程信息
func (pr *ProcessResolverImpl) getFromCache(key string) *ProcessInfo {
	pr.cacheMu.RLock()
	defer pr.cacheMu.RUnlock()

	cached, exists := pr.cache[key]
	if !exists {
		return nil
	}

	// 检查是否过期
	if time.Since(cached.Timestamp) > pr.cacheExpireTime {
		// 异步删除过期缓存
		go func() {
			pr.cacheMu.Lock()
			delete(pr.cache, key)
			pr.cacheMu.Unlock()
		}()
		return nil
	}

	return cached.ProcessInfo
}

// setCache 设置缓存
func (pr *ProcessResolverImpl) setCache(key string, processInfo *ProcessInfo, source string) {
	pr.cacheMu.Lock()
	defer pr.cacheMu.Unlock()

	pr.cache[key] = &CachedProcessInfo{
		ProcessInfo: processInfo,
		Timestamp:   time.Now(),
		Source:      source,
	}

	// 限制缓存大小
	if len(pr.cache) > 1000 {
		pr.cleanupOldCacheEntries()
	}
}

// cleanupOldCacheEntries 清理旧的缓存条目
func (pr *ProcessResolverImpl) cleanupOldCacheEntries() {
	now := time.Now()
	keysToDelete := make([]string, 0)

	for key, cached := range pr.cache {
		if now.Sub(cached.Timestamp) > pr.cacheExpireTime {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(pr.cache, key)
	}

	pr.logger.Debug("清理过期缓存条目", "deleted_count", len(keysToDelete), "remaining_count", len(pr.cache))
}

// GetStats 获取统计信息
func (pr *ProcessResolverImpl) GetStats() map[string]interface{} {
	pr.stats.mu.RLock()
	defer pr.stats.mu.RUnlock()

	pr.cacheMu.RLock()
	cacheSize := len(pr.cache)
	pr.cacheMu.RUnlock()

	pr.mu.RLock()
	sourceCount := len(pr.dataSources)
	sourceNames := make([]string, len(pr.dataSources))
	for i, source := range pr.dataSources {
		sourceNames[i] = source.Name()
	}
	pr.mu.RUnlock()

	hitRate := float64(0)
	if pr.stats.totalQueries > 0 {
		hitRate = float64(pr.stats.cacheHits) / float64(pr.stats.totalQueries) * 100
	}

	return map[string]interface{}{
		"total_queries":     pr.stats.totalQueries,
		"cache_hits":        pr.stats.cacheHits,
		"cache_misses":      pr.stats.cacheMisses,
		"cache_hit_rate":    hitRate,
		"cache_size":        cacheSize,
		"cache_expire_time": pr.cacheExpireTime.String(),
		"source_count":      sourceCount,
		"source_names":      sourceNames,
		"source_queries":    pr.stats.sourceQueries,
		"last_query_time":   pr.stats.lastQueryTime,
	}
}

// ClearCache 清空缓存
func (pr *ProcessResolverImpl) ClearCache() {
	pr.cacheMu.Lock()
	defer pr.cacheMu.Unlock()

	oldSize := len(pr.cache)
	pr.cache = make(map[string]*CachedProcessInfo)

	pr.logger.Info("清空进程信息缓存", "cleared_entries", oldSize)
}

// SetCacheExpireTime 设置缓存过期时间
func (pr *ProcessResolverImpl) SetCacheExpireTime(duration time.Duration) {
	pr.cacheExpireTime = duration
	pr.logger.Info("设置缓存过期时间", "expire_time", duration)
}
