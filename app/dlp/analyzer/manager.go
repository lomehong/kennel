package analyzer

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lomehong/kennel/app/dlp/parser"
	"github.com/lomehong/kennel/pkg/logging"
)

// AnalysisManagerImpl 分析管理器实现
type AnalysisManagerImpl struct {
	analyzers    map[string]ContentAnalyzer
	config       AnalyzerConfig
	logger       logging.Logger
	stats        ManagerStats
	cacheManager CacheManager
	running      int32
	mu           sync.RWMutex
}

// NewAnalysisManager 创建分析管理器
func NewAnalysisManager(logger logging.Logger, config AnalyzerConfig) AnalysisManager {
	return &AnalysisManagerImpl{
		analyzers:    make(map[string]ContentAnalyzer),
		config:       config,
		logger:       logger,
		cacheManager: NewCacheManager(config.CacheSize, config.CacheTTL),
		stats: ManagerStats{
			AnalyzerStats: make(map[string]AnalyzerStats),
			StartTime:     time.Now(),
		},
	}
}

// RegisterAnalyzer 注册分析器
func (am *AnalysisManagerImpl) RegisterAnalyzer(analyzer ContentAnalyzer) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	info := analyzer.GetAnalyzerInfo()
	for _, contentType := range info.SupportedTypes {
		if _, exists := am.analyzers[contentType]; exists {
			return fmt.Errorf("内容类型分析器已存在: %s", contentType)
		}
		am.analyzers[contentType] = analyzer
		am.stats.AnalyzerStats[info.Name] = analyzer.GetStats()
	}

	am.logger.Info("注册内容分析器",
		"name", info.Name,
		"version", info.Version,
		"types", info.SupportedTypes)

	return nil
}

// GetAnalyzer 获取分析器
func (am *AnalysisManagerImpl) GetAnalyzer(contentType string) (ContentAnalyzer, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	analyzer, exists := am.analyzers[contentType]
	return analyzer, exists
}

// AnalyzeContent 分析内容
func (am *AnalysisManagerImpl) AnalyzeContent(ctx context.Context, data *parser.ParsedData) (*AnalysisResult, error) {
	startTime := time.Now()
	atomic.AddUint64(&am.stats.TotalRequests, 1)

	// 检查缓存
	cacheKey := am.generateCacheKey(data)
	if cached, found := am.cacheManager.Get(cacheKey); found {
		am.logger.Debug("从缓存获取分析结果", "cache_key", cacheKey)
		if result, ok := cached.(*AnalysisResult); ok {
			atomic.AddUint64(&am.stats.ProcessedRequests, 1)
			return result, nil
		}
	}

	// 查找合适的分析器
	analyzer, exists := am.GetAnalyzer(data.ContentType)
	if !exists {
		// 尝试通用分析器
		analyzer, exists = am.GetAnalyzer("text/plain")
		if !exists {
			atomic.AddUint64(&am.stats.FailedRequests, 1)
			return nil, fmt.Errorf("未找到合适的内容分析器: %s", data.ContentType)
		}
	}

	// 执行分析
	result, err := analyzer.Analyze(ctx, data)
	if err != nil {
		atomic.AddUint64(&am.stats.FailedRequests, 1)
		am.stats.LastError = err
		return nil, fmt.Errorf("内容分析失败: %w", err)
	}

	// 更新统计信息
	atomic.AddUint64(&am.stats.ProcessedRequests, 1)
	processingTime := time.Since(startTime)
	am.updateAverageTime(processingTime)

	// 缓存结果
	am.cacheManager.Set(cacheKey, result, am.config.CacheTTL)

	am.logger.Debug("内容分析完成",
		"content_type", data.ContentType,
		"risk_level", result.RiskLevel.String(),
		"sensitive_count", len(result.SensitiveData),
		"processing_time", processingTime)

	return result, nil
}

// GetSupportedTypes 获取支持的内容类型
func (am *AnalysisManagerImpl) GetSupportedTypes() []string {
	am.mu.RLock()
	defer am.mu.RUnlock()

	types := make([]string, 0, len(am.analyzers))
	for contentType := range am.analyzers {
		types = append(types, contentType)
	}

	return types
}

// GetStats 获取统计信息
func (am *AnalysisManagerImpl) GetStats() ManagerStats {
	am.mu.RLock()
	defer am.mu.RUnlock()

	stats := am.stats
	stats.Uptime = time.Since(am.stats.StartTime)

	// 更新分析器统计信息
	for _, analyzer := range am.analyzers {
		info := analyzer.GetAnalyzerInfo()
		stats.AnalyzerStats[info.Name] = analyzer.GetStats()
	}

	return stats
}

// Start 启动管理器
func (am *AnalysisManagerImpl) Start() error {
	if !atomic.CompareAndSwapInt32(&am.running, 0, 1) {
		return fmt.Errorf("分析管理器已在运行")
	}

	am.logger.Info("启动内容分析管理器")

	// 初始化所有分析器
	am.mu.RLock()
	for contentType, analyzer := range am.analyzers {
		if err := analyzer.Initialize(am.config); err != nil {
			am.mu.RUnlock()
			return fmt.Errorf("初始化分析器失败 [%s]: %w", contentType, err)
		}
	}
	am.mu.RUnlock()

	am.logger.Info("内容分析管理器已启动")
	return nil
}

// Stop 停止管理器
func (am *AnalysisManagerImpl) Stop() error {
	if !atomic.CompareAndSwapInt32(&am.running, 1, 0) {
		return fmt.Errorf("分析管理器未在运行")
	}

	am.logger.Info("停止内容分析管理器")

	// 清理所有分析器
	am.mu.RLock()
	for contentType, analyzer := range am.analyzers {
		if err := analyzer.Cleanup(); err != nil {
			am.logger.Error("清理分析器失败", "content_type", contentType, "error", err)
		}
	}
	am.mu.RUnlock()

	// 清理缓存
	am.cacheManager.Clear()

	am.logger.Info("内容分析管理器已停止")
	return nil
}

// UpdateRules 更新规则
func (am *AnalysisManagerImpl) UpdateRules(analyzerName string, rules interface{}) error {
	am.mu.RLock()
	defer am.mu.RUnlock()

	for _, analyzer := range am.analyzers {
		info := analyzer.GetAnalyzerInfo()
		if info.Name == analyzerName {
			if err := analyzer.UpdateRules(rules); err != nil {
				return fmt.Errorf("更新分析器规则失败 [%s]: %w", analyzerName, err)
			}
			am.logger.Info("更新分析器规则", "analyzer", analyzerName)
			return nil
		}
	}

	return fmt.Errorf("未找到分析器: %s", analyzerName)
}

// generateCacheKey 生成缓存键
func (am *AnalysisManagerImpl) generateCacheKey(data *parser.ParsedData) string {
	// 简化的缓存键生成，实际应该使用更复杂的哈希算法
	return fmt.Sprintf("%s_%d_%s", data.Protocol, len(data.Body), data.ContentType)
}

// updateAverageTime 更新平均处理时间
func (am *AnalysisManagerImpl) updateAverageTime(duration time.Duration) {
	// 简化的平均时间计算，实际应该使用更精确的算法
	am.stats.AverageTime = (am.stats.AverageTime + duration) / 2
}

// CacheManagerImpl 缓存管理器实现
type CacheManagerImpl struct {
	cache   map[string]*cacheItem
	maxSize int
	stats   CacheStats
	mu      sync.RWMutex
}

// cacheItem 缓存项
type cacheItem struct {
	value    interface{}
	expireAt time.Time
	accessAt time.Time
}

// NewCacheManager 创建缓存管理器
func NewCacheManager(maxSize int, defaultTTL time.Duration) CacheManager {
	cm := &CacheManagerImpl{
		cache:   make(map[string]*cacheItem),
		maxSize: maxSize,
		stats: CacheStats{
			MaxSize: maxSize,
		},
	}

	// 启动清理协程
	go cm.cleanupWorker(defaultTTL)

	return cm
}

// Get 获取缓存
func (cm *CacheManagerImpl) Get(key string) (interface{}, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	item, exists := cm.cache[key]
	if !exists {
		cm.stats.Misses++
		return nil, false
	}

	// 检查是否过期
	if time.Now().After(item.expireAt) {
		cm.mu.RUnlock()
		cm.mu.Lock()
		delete(cm.cache, key)
		cm.mu.Unlock()
		cm.mu.RLock()
		cm.stats.Misses++
		return nil, false
	}

	// 更新访问时间
	item.accessAt = time.Now()
	cm.stats.Hits++
	cm.updateHitRate()

	return item.value, true
}

// Set 设置缓存
func (cm *CacheManagerImpl) Set(key string, value interface{}, ttl time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 检查缓存大小限制
	if len(cm.cache) >= cm.maxSize {
		cm.evictLRU()
	}

	cm.cache[key] = &cacheItem{
		value:    value,
		expireAt: time.Now().Add(ttl),
		accessAt: time.Now(),
	}

	cm.stats.Size = len(cm.cache)
}

// Delete 删除缓存
func (cm *CacheManagerImpl) Delete(key string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.cache, key)
	cm.stats.Size = len(cm.cache)
}

// Clear 清空缓存
func (cm *CacheManagerImpl) Clear() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.cache = make(map[string]*cacheItem)
	cm.stats.Size = 0
}

// Size 获取缓存大小
func (cm *CacheManagerImpl) Size() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return len(cm.cache)
}

// Stats 获取缓存统计
func (cm *CacheManagerImpl) Stats() CacheStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stats := cm.stats
	stats.Size = len(cm.cache)
	return stats
}

// evictLRU 驱逐最近最少使用的项
func (cm *CacheManagerImpl) evictLRU() {
	var oldestKey string
	var oldestTime time.Time

	for key, item := range cm.cache {
		if oldestKey == "" || item.accessAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.accessAt
		}
	}

	if oldestKey != "" {
		delete(cm.cache, oldestKey)
		cm.stats.Evictions++
	}
}

// updateHitRate 更新命中率
func (cm *CacheManagerImpl) updateHitRate() {
	total := cm.stats.Hits + cm.stats.Misses
	if total > 0 {
		cm.stats.HitRate = float64(cm.stats.Hits) / float64(total)
	}
}

// cleanupWorker 清理工作协程
func (cm *CacheManagerImpl) cleanupWorker(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		cm.cleanupExpired()
	}
}

// cleanupExpired 清理过期项
func (cm *CacheManagerImpl) cleanupExpired() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()
	expired := make([]string, 0)

	for key, item := range cm.cache {
		if now.After(item.expireAt) {
			expired = append(expired, key)
		}
	}

	for _, key := range expired {
		delete(cm.cache, key)
	}

	cm.stats.Size = len(cm.cache)
}
