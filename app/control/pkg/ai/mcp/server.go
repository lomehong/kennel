package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	sdk "github.com/lomehong/kennel/pkg/sdk/go"
)

// ServerConfig 定义了 MCP Server 的配置
type ServerConfig struct {
	Addr           string        // 监听地址，默认为 :8080
	ReadTimeout    time.Duration // 读取超时，默认为 10 秒
	WriteTimeout   time.Duration // 写入超时，默认为 10 秒
	MaxHeaderBytes int           // 最大头部字节数，默认为 1MB
	APIKey         string        // API 密钥，用于认证
}

// Server 实现了 MCP Server
type Server struct {
	config     *ServerConfig
	router     *mux.Router
	httpServer *http.Server
	tools      map[string]Tool
	logger     sdk.Logger
	mu         sync.RWMutex
}

// NewServer 创建一个新的 MCP Server
func NewServer(config *ServerConfig, logger sdk.Logger) (*Server, error) {
	if config == nil {
		config = &ServerConfig{}
	}

	// 设置默认值
	if config.Addr == "" {
		config.Addr = ":8080"
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 10 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 10 * time.Second
	}
	if config.MaxHeaderBytes == 0 {
		config.MaxHeaderBytes = 1 << 20 // 1MB
	}

	// 创建路由器
	router := mux.NewRouter()

	// 创建服务器
	server := &Server{
		config: config,
		router: router,
		tools:  make(map[string]Tool),
		logger: logger,
	}

	// 注册路由
	router.HandleFunc("/tools", server.handleListTools).Methods("GET")
	router.HandleFunc("/tools/{name}", server.handleGetTool).Methods("GET")
	router.HandleFunc("/tools/{name}/execute", server.handleExecuteTool).Methods("POST")

	// 添加中间件
	if config.APIKey != "" {
		router.Use(server.apiKeyMiddleware)
	}
	router.Use(server.loggingMiddleware)

	// 创建 HTTP 服务器
	server.httpServer = &http.Server{
		Addr:           config.Addr,
		Handler:        router,
		ReadTimeout:    config.ReadTimeout,
		WriteTimeout:   config.WriteTimeout,
		MaxHeaderBytes: config.MaxHeaderBytes,
	}

	return server, nil
}

// RegisterTool 注册工具
func (s *Server) RegisterTool(tool Tool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := tool.GetName()
	if name == "" {
		return fmt.Errorf("工具名称不能为空")
	}

	if _, exists := s.tools[name]; exists {
		return fmt.Errorf("工具 %s 已存在", name)
	}

	s.tools[name] = tool
	s.logger.Info("注册工具", "name", name, "description", tool.GetDescription())
	return nil
}

// UnregisterTool 注销工具
func (s *Server) UnregisterTool(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tools[name]; !exists {
		return fmt.Errorf("工具 %s 不存在", name)
	}

	delete(s.tools, name)
	s.logger.Info("注销工具", "name", name)
	return nil
}

// ListTools 列出所有工具
func (s *Server) ListTools() []ToolInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]ToolInfo, 0, len(s.tools))
	for _, tool := range s.tools {
		tools = append(tools, ToToolInfo(tool))
	}
	return tools
}

// Start 启动服务器
func (s *Server) Start() error {
	s.logger.Info("启动 MCP Server", "addr", s.config.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown 关闭服务器
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("关闭 MCP Server")
	return s.httpServer.Shutdown(ctx)
}

// apiKeyMiddleware 实现 API 密钥认证
func (s *Server) apiKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != s.config.APIKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware 实现请求日志记录
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Info("HTTP请求",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"duration", time.Since(start),
		)
	})
}

// handleListTools 处理列出工具请求
func (s *Server) handleListTools(w http.ResponseWriter, r *http.Request) {
	tools := s.ListTools()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(tools); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		s.logger.Error("编码工具列表失败", "error", err)
		return
	}
}

// handleGetTool 处理获取工具信息请求
func (s *Server) handleGetTool(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	s.mu.RLock()
	tool, exists := s.tools[name]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, fmt.Sprintf("工具 %s 不存在", name), http.StatusNotFound)
		return
	}

	info := ToToolInfo(tool)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(info); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		s.logger.Error("编码工具信息失败", "error", err)
		return
	}
}

// handleExecuteTool 处理执行工具请求
func (s *Server) handleExecuteTool(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	s.mu.RLock()
	tool, exists := s.tools[name]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, fmt.Sprintf("工具 %s 不存在", name), http.StatusNotFound)
		return
	}

	// 解析请求参数
	var params map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "无效的请求参数", http.StatusBadRequest)
		s.logger.Error("解析请求参数失败", "error", err)
		return
	}

	// 创建上下文，包含超时
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// 执行工具
	result, err := tool.Execute(ctx, params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		s.logger.Error("执行工具失败", "tool", name, "error", err)
		return
	}

	// 返回结果
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"result": result,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		s.logger.Error("编码执行结果失败", "error", err)
		return
	}

	s.logger.Info("执行工具成功", "tool", name)
}
