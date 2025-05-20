package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/hashicorp/go-hclog"
)

// DebugServer 调试服务器
// 提供插件调试功能
type DebugServer struct {
	// 插件ID
	pluginID string

	// 服务器
	server *http.Server

	// 日志记录器
	logger hclog.Logger

	// 调试端口
	port int

	// 是否启用
	enabled bool

	// 调试处理器
	handlers map[string]http.HandlerFunc
}

// DebugOption 调试选项
type DebugOption func(*DebugServer)

// WithDebugPort 设置调试端口
func WithDebugPort(port int) DebugOption {
	return func(ds *DebugServer) {
		ds.port = port
	}
}

// WithDebugEnabled 设置是否启用调试
func WithDebugEnabled(enabled bool) DebugOption {
	return func(ds *DebugServer) {
		ds.enabled = enabled
	}
}

// NewDebugServer 创建一个新的调试服务器
func NewDebugServer(pluginID string, logger hclog.Logger, options ...DebugOption) *DebugServer {
	if logger == nil {
		logger = hclog.NewNullLogger()
	}

	ds := &DebugServer{
		pluginID:  pluginID,
		logger:    logger.Named("debug-server"),
		port:      8080,
		enabled:   false,
		handlers:  make(map[string]http.HandlerFunc),
	}

	// 应用选项
	for _, option := range options {
		option(ds)
	}

	// 注册默认处理器
	ds.registerDefaultHandlers()

	return ds
}

// registerDefaultHandlers 注册默认处理器
func (ds *DebugServer) registerDefaultHandlers() {
	// 注册健康检查处理器
	ds.RegisterHandler("/health", ds.handleHealth)

	// 注册状态处理器
	ds.RegisterHandler("/status", ds.handleStatus)

	// 注册内存分析处理器
	ds.RegisterHandler("/debug/pprof/heap", ds.handleHeapProfile)

	// 注册CPU分析处理器
	ds.RegisterHandler("/debug/pprof/profile", ds.handleCPUProfile)

	// 注册goroutine分析处理器
	ds.RegisterHandler("/debug/pprof/goroutine", ds.handleGoroutineProfile)

	// 注册线程分析处理器
	ds.RegisterHandler("/debug/pprof/threadcreate", ds.handleThreadCreateProfile)

	// 注册阻塞分析处理器
	ds.RegisterHandler("/debug/pprof/block", ds.handleBlockProfile)

	// 注册内存统计处理器
	ds.RegisterHandler("/debug/memstats", ds.handleMemStats)

	// 注册GC统计处理器
	ds.RegisterHandler("/debug/gcstats", ds.handleGCStats)
}

// RegisterHandler 注册处理器
func (ds *DebugServer) RegisterHandler(path string, handler http.HandlerFunc) {
	ds.handlers[path] = handler
}

// Start 启动调试服务器
func (ds *DebugServer) Start() error {
	if !ds.enabled {
		ds.logger.Info("调试服务器未启用")
		return nil
	}

	// 创建路由器
	mux := http.NewServeMux()

	// 注册处理器
	for path, handler := range ds.handlers {
		mux.HandleFunc(path, handler)
	}

	// 创建服务器
	ds.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", ds.port),
		Handler: mux,
	}

	// 启动服务器
	ds.logger.Info("启动调试服务器", "port", ds.port)
	go func() {
		if err := ds.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ds.logger.Error("调试服务器错误", "error", err)
		}
	}()

	return nil
}

// Stop 停止调试服务器
func (ds *DebugServer) Stop() error {
	if ds.server == nil {
		return nil
	}

	ds.logger.Info("停止调试服务器")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return ds.server.Shutdown(ctx)
}

// handleHealth 处理健康检查
func (ds *DebugServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// handleStatus 处理状态
func (ds *DebugServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	// 获取运行时信息
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	status := map[string]interface{}{
		"plugin_id":      ds.pluginID,
		"time":           time.Now().Format(time.RFC3339),
		"go_version":     runtime.Version(),
		"go_os":          runtime.GOOS,
		"go_arch":        runtime.GOARCH,
		"cpu_num":        runtime.NumCPU(),
		"goroutine_num":  runtime.NumGoroutine(),
		"memory_alloc":   memStats.Alloc,
		"memory_total":   memStats.TotalAlloc,
		"memory_sys":     memStats.Sys,
		"memory_heap":    memStats.HeapAlloc,
		"gc_num":         memStats.NumGC,
		"process_id":     os.Getpid(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

// handleHeapProfile 处理堆分析
func (ds *DebugServer) handleHeapProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=heap.pprof")
	pprof.Lookup("heap").WriteTo(w, 0)
}

// handleCPUProfile 处理CPU分析
func (ds *DebugServer) handleCPUProfile(w http.ResponseWriter, r *http.Request) {
	// 获取分析时间
	seconds := 30
	if s := r.URL.Query().Get("seconds"); s != "" {
		fmt.Sscanf(s, "%d", &seconds)
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=cpu.pprof")

	if err := pprof.StartCPUProfile(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	time.Sleep(time.Duration(seconds) * time.Second)
	pprof.StopCPUProfile()
}

// handleGoroutineProfile 处理goroutine分析
func (ds *DebugServer) handleGoroutineProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=goroutine.pprof")
	pprof.Lookup("goroutine").WriteTo(w, 0)
}

// handleThreadCreateProfile 处理线程分析
func (ds *DebugServer) handleThreadCreateProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=threadcreate.pprof")
	pprof.Lookup("threadcreate").WriteTo(w, 0)
}

// handleBlockProfile 处理阻塞分析
func (ds *DebugServer) handleBlockProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=block.pprof")
	pprof.Lookup("block").WriteTo(w, 0)
}

// handleMemStats 处理内存统计
func (ds *DebugServer) handleMemStats(w http.ResponseWriter, r *http.Request) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(memStats)
}

// handleGCStats 处理GC统计
func (ds *DebugServer) handleGCStats(w http.ResponseWriter, r *http.Request) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	gcStats := map[string]interface{}{
		"num_gc":         memStats.NumGC,
		"next_gc":        memStats.NextGC,
		"last_gc":        memStats.LastGC,
		"pause_total_ns": memStats.PauseTotalNs,
		"pause_ns":       memStats.PauseNs,
		"pause_end":      memStats.PauseEnd,
		"gc_cpu_fraction": memStats.GCCPUFraction,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(gcStats)
}

// EnableBlockProfile 启用阻塞分析
func EnableBlockProfile() {
	runtime.SetBlockProfileRate(1)
}

// DisableBlockProfile 禁用阻塞分析
func DisableBlockProfile() {
	runtime.SetBlockProfileRate(0)
}

// EnableMutexProfile 启用互斥锁分析
func EnableMutexProfile() {
	runtime.SetMutexProfileFraction(1)
}

// DisableMutexProfile 禁用互斥锁分析
func DisableMutexProfile() {
	runtime.SetMutexProfileFraction(0)
}

// DumpHeapProfile 导出堆分析
func DumpHeapProfile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("创建堆分析文件失败: %w", err)
	}
	defer f.Close()

	return pprof.WriteHeapProfile(f)
}

// DumpGoroutineProfile 导出goroutine分析
func DumpGoroutineProfile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("创建goroutine分析文件失败: %w", err)
	}
	defer f.Close()

	return pprof.Lookup("goroutine").WriteTo(f, 0)
}

// StartCPUProfile 开始CPU分析
func StartCPUProfile(path string) (*os.File, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("创建CPU分析文件失败: %w", err)
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		return nil, fmt.Errorf("开始CPU分析失败: %w", err)
	}

	return f, nil
}

// StopCPUProfile 停止CPU分析
func StopCPUProfile(f *os.File) {
	pprof.StopCPUProfile()
	f.Close()
}
