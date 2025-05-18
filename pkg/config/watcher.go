package config

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hashicorp/go-hclog"
)

// ChangeType 表示配置变更类型
type ChangeType int

// 预定义配置变更类型
const (
	ChangeTypeCreate ChangeType = iota // 创建
	ChangeTypeUpdate               // 更新
	ChangeTypeDelete               // 删除
	ChangeTypeRename               // 重命名
	ChangeTypeChmod                // 权限变更
)

// String 返回变更类型的字符串表示
func (ct ChangeType) String() string {
	switch ct {
	case ChangeTypeCreate:
		return "Create"
	case ChangeTypeUpdate:
		return "Update"
	case ChangeTypeDelete:
		return "Delete"
	case ChangeTypeRename:
		return "Rename"
	case ChangeTypeChmod:
		return "Chmod"
	default:
		return "Unknown"
	}
}

// ChangeEvent 配置变更事件
type ChangeEvent struct {
	Type      ChangeType           // 变更类型
	Path      string               // 文件路径
	Time      time.Time            // 变更时间
	OldConfig map[string]interface{} // 旧配置
	NewConfig map[string]interface{} // 新配置
	Changes   map[string]interface{} // 变更内容
}

// ChangeHandler 配置变更处理器
type ChangeHandler func(event ChangeEvent) error

// ConfigWatcher 配置监视器
type ConfigWatcher struct {
	watcher      *fsnotify.Watcher
	handlers     map[string][]ChangeHandler
	paths        map[string]bool
	logger       hclog.Logger
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	debounceTime time.Duration
	events       chan fsnotify.Event
}

// NewConfigWatcher 创建一个新的配置监视器
func NewConfigWatcher(logger hclog.Logger) (*ConfigWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("创建文件监视器失败: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &ConfigWatcher{
		watcher:      watcher,
		handlers:     make(map[string][]ChangeHandler),
		paths:        make(map[string]bool),
		logger:       logger,
		ctx:          ctx,
		cancel:       cancel,
		debounceTime: 100 * time.Millisecond,
		events:       make(chan fsnotify.Event, 100),
	}, nil
}

// AddPath 添加监视路径
func (w *ConfigWatcher) AddPath(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 检查路径是否已存在
	if _, exists := w.paths[path]; exists {
		return nil
	}

	// 添加到监视器
	if err := w.watcher.Add(path); err != nil {
		return fmt.Errorf("添加监视路径失败: %w", err)
	}

	// 记录路径
	w.paths[path] = true
	w.logger.Debug("添加监视路径", "path", path)

	return nil
}

// RemovePath 移除监视路径
func (w *ConfigWatcher) RemovePath(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 检查路径是否存在
	if _, exists := w.paths[path]; !exists {
		return nil
	}

	// 从监视器移除
	if err := w.watcher.Remove(path); err != nil {
		return fmt.Errorf("移除监视路径失败: %w", err)
	}

	// 移除路径
	delete(w.paths, path)
	w.logger.Debug("移除监视路径", "path", path)

	return nil
}

// AddHandler 添加变更处理器
func (w *ConfigWatcher) AddHandler(path string, handler ChangeHandler) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 添加处理器
	w.handlers[path] = append(w.handlers[path], handler)
	w.logger.Debug("添加变更处理器", "path", path)
}

// RemoveHandlers 移除变更处理器
func (w *ConfigWatcher) RemoveHandlers(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 移除处理器
	delete(w.handlers, path)
	w.logger.Debug("移除变更处理器", "path", path)
}

// Start 启动监视
func (w *ConfigWatcher) Start() {
	w.logger.Info("启动配置监视")

	// 启动事件处理
	w.wg.Add(1)
	go w.processEvents()

	// 启动事件收集
	w.wg.Add(1)
	go w.collectEvents()
}

// Stop 停止监视
func (w *ConfigWatcher) Stop() {
	w.logger.Info("停止配置监视")

	// 取消上下文
	w.cancel()

	// 等待处理完成
	w.wg.Wait()

	// 关闭监视器
	w.watcher.Close()
}

// collectEvents 收集事件
func (w *ConfigWatcher) collectEvents() {
	defer w.wg.Done()

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.events <- event
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.logger.Error("监视器错误", "error", err)
		case <-w.ctx.Done():
			return
		}
	}
}

// processEvents 处理事件
func (w *ConfigWatcher) processEvents() {
	defer w.wg.Done()

	// 使用map对事件进行去重
	eventMap := make(map[string]fsnotify.Event)
	timer := time.NewTimer(w.debounceTime)
	timer.Stop()

	for {
		select {
		case event := <-w.events:
			// 记录事件
			eventMap[event.Name] = event
			// 重置定时器
			timer.Reset(w.debounceTime)
		case <-timer.C:
			// 处理事件
			for _, event := range eventMap {
				w.handleEvent(event)
			}
			// 清空事件map
			eventMap = make(map[string]fsnotify.Event)
		case <-w.ctx.Done():
			return
		}
	}
}

// handleEvent 处理单个事件
func (w *ConfigWatcher) handleEvent(event fsnotify.Event) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// 获取文件路径
	path := event.Name
	w.logger.Debug("收到文件事件", "path", path, "op", event.Op.String())

	// 获取变更类型
	var changeType ChangeType
	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		changeType = ChangeTypeCreate
	case event.Op&fsnotify.Write == fsnotify.Write:
		changeType = ChangeTypeUpdate
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		changeType = ChangeTypeDelete
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		changeType = ChangeTypeRename
	case event.Op&fsnotify.Chmod == fsnotify.Chmod:
		changeType = ChangeTypeChmod
	default:
		w.logger.Warn("未知事件类型", "op", event.Op.String())
		return
	}

	// 创建变更事件
	changeEvent := ChangeEvent{
		Type: changeType,
		Path: path,
		Time: time.Now(),
	}

	// 查找处理器
	var handlers []ChangeHandler

	// 精确匹配
	if h, ok := w.handlers[path]; ok {
		handlers = append(handlers, h...)
	}

	// 目录匹配
	dir := filepath.Dir(path)
	if h, ok := w.handlers[dir]; ok {
		handlers = append(handlers, h...)
	}

	// 通配符匹配
	for p, h := range w.handlers {
		if p == "*" {
			handlers = append(handlers, h...)
		}
	}

	// 调用处理器
	for _, handler := range handlers {
		if err := handler(changeEvent); err != nil {
			w.logger.Error("处理配置变更失败", "path", path, "error", err)
		}
	}
}

// SetDebounceTime 设置去抖时间
func (w *ConfigWatcher) SetDebounceTime(duration time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.debounceTime = duration
}

// GetWatchedPaths 获取监视的路径
func (w *ConfigWatcher) GetWatchedPaths() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	paths := make([]string, 0, len(w.paths))
	for path := range w.paths {
		paths = append(paths, path)
	}
	return paths
}
