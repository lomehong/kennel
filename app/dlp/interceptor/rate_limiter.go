package interceptor

import (
	"sync"
	"time"

	"github.com/lomehong/kennel/pkg/logging"
)

// RateLimiter 流量限制器
type RateLimiter struct {
	maxPacketsPerSecond int64
	maxBytesPerSecond   int64
	burstSize           int64
	
	// 令牌桶
	packetTokens int64
	byteTokens   int64
	
	// 时间跟踪
	lastRefill time.Time
	
	// 统计信息
	packetsDropped uint64
	bytesDropped   uint64
	
	mu     sync.Mutex
	logger logging.Logger
}

// NewRateLimiter 创建新的流量限制器
func NewRateLimiter(maxPacketsPerSecond, maxBytesPerSecond, burstSize int64, logger logging.Logger) *RateLimiter {
	return &RateLimiter{
		maxPacketsPerSecond: maxPacketsPerSecond,
		maxBytesPerSecond:   maxBytesPerSecond,
		burstSize:           burstSize,
		packetTokens:        burstSize,
		byteTokens:          maxBytesPerSecond,
		lastRefill:          time.Now(),
		logger:              logger,
	}
}

// AllowPacket 检查是否允许处理数据包
func (rl *RateLimiter) AllowPacket(packetSize int64) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	// 补充令牌
	rl.refillTokens()
	
	// 检查是否有足够的令牌
	if rl.packetTokens > 0 && rl.byteTokens >= packetSize {
		rl.packetTokens--
		rl.byteTokens -= packetSize
		return true
	}
	
	// 记录丢弃统计
	rl.packetsDropped++
	rl.bytesDropped += uint64(packetSize)
	
	return false
}

// refillTokens 补充令牌
func (rl *RateLimiter) refillTokens() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	
	if elapsed < time.Millisecond*100 { // 最小补充间隔
		return
	}
	
	// 计算应该补充的令牌数
	seconds := elapsed.Seconds()
	
	// 补充数据包令牌
	packetTokensToAdd := int64(seconds * float64(rl.maxPacketsPerSecond))
	rl.packetTokens += packetTokensToAdd
	if rl.packetTokens > rl.burstSize {
		rl.packetTokens = rl.burstSize
	}
	
	// 补充字节令牌
	byteTokensToAdd := int64(seconds * float64(rl.maxBytesPerSecond))
	rl.byteTokens += byteTokensToAdd
	if rl.byteTokens > rl.maxBytesPerSecond {
		rl.byteTokens = rl.maxBytesPerSecond
	}
	
	rl.lastRefill = now
}

// GetStats 获取统计信息
func (rl *RateLimiter) GetStats() (packetsDropped, bytesDropped uint64) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	return rl.packetsDropped, rl.bytesDropped
}

// Reset 重置统计信息
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	rl.packetsDropped = 0
	rl.bytesDropped = 0
}

// UpdateLimits 更新限制参数
func (rl *RateLimiter) UpdateLimits(maxPacketsPerSecond, maxBytesPerSecond, burstSize int64) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	rl.maxPacketsPerSecond = maxPacketsPerSecond
	rl.maxBytesPerSecond = maxBytesPerSecond
	rl.burstSize = burstSize
	
	// 调整当前令牌数
	if rl.packetTokens > burstSize {
		rl.packetTokens = burstSize
	}
	if rl.byteTokens > maxBytesPerSecond {
		rl.byteTokens = maxBytesPerSecond
	}
	
	rl.logger.Info("更新流量限制参数",
		"max_packets_per_second", maxPacketsPerSecond,
		"max_bytes_per_second", maxBytesPerSecond,
		"burst_size", burstSize)
}

// AdaptiveLimiter 自适应流量限制器
type AdaptiveLimiter struct {
	*RateLimiter
	
	// 自适应参数
	cpuThreshold    float64
	memoryThreshold float64
	checkInterval   time.Duration
	
	// 原始限制值
	originalPacketsPerSecond int64
	originalBytesPerSecond   int64
	originalBurstSize        int64
	
	// 当前调整因子
	adjustmentFactor float64
	
	lastCheck time.Time
	logger    logging.Logger
}

// NewAdaptiveLimiter 创建自适应流量限制器
func NewAdaptiveLimiter(maxPacketsPerSecond, maxBytesPerSecond, burstSize int64, 
	cpuThreshold, memoryThreshold float64, checkInterval time.Duration, logger logging.Logger) *AdaptiveLimiter {
	
	return &AdaptiveLimiter{
		RateLimiter:              NewRateLimiter(maxPacketsPerSecond, maxBytesPerSecond, burstSize, logger),
		cpuThreshold:             cpuThreshold,
		memoryThreshold:          memoryThreshold,
		checkInterval:            checkInterval,
		originalPacketsPerSecond: maxPacketsPerSecond,
		originalBytesPerSecond:   maxBytesPerSecond,
		originalBurstSize:        burstSize,
		adjustmentFactor:         1.0,
		lastCheck:                time.Now(),
		logger:                   logger,
	}
}

// CheckAndAdjust 检查系统资源并调整限制
func (al *AdaptiveLimiter) CheckAndAdjust(cpuUsage, memoryUsage float64) {
	now := time.Now()
	if now.Sub(al.lastCheck) < al.checkInterval {
		return
	}
	
	al.lastCheck = now
	
	// 计算新的调整因子
	newFactor := 1.0
	
	if cpuUsage > al.cpuThreshold {
		// CPU使用率过高，降低限制
		newFactor *= (al.cpuThreshold / cpuUsage)
	}
	
	if memoryUsage > al.memoryThreshold {
		// 内存使用率过高，降低限制
		newFactor *= (al.memoryThreshold / memoryUsage)
	}
	
	// 限制调整范围
	if newFactor < 0.1 {
		newFactor = 0.1 // 最低10%
	} else if newFactor > 1.0 {
		newFactor = 1.0 // 最高100%
	}
	
	// 如果调整因子有显著变化，更新限制
	if abs(newFactor-al.adjustmentFactor) > 0.1 {
		al.adjustmentFactor = newFactor
		
		newPacketsPerSecond := int64(float64(al.originalPacketsPerSecond) * newFactor)
		newBytesPerSecond := int64(float64(al.originalBytesPerSecond) * newFactor)
		newBurstSize := int64(float64(al.originalBurstSize) * newFactor)
		
		al.UpdateLimits(newPacketsPerSecond, newBytesPerSecond, newBurstSize)
		
		al.logger.Info("自适应调整流量限制",
			"cpu_usage", cpuUsage,
			"memory_usage", memoryUsage,
			"adjustment_factor", newFactor,
			"new_packets_per_second", newPacketsPerSecond,
			"new_bytes_per_second", newBytesPerSecond)
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
