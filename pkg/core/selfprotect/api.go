package selfprotect

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/hashicorp/go-hclog"
)

// ProtectionAPI 自我防护API
type ProtectionAPI struct {
	service *ProtectionService
	logger  hclog.Logger
}

// NewProtectionAPI 创建自我防护API
func NewProtectionAPI(service *ProtectionService, logger hclog.Logger) *ProtectionAPI {
	return &ProtectionAPI{
		service: service,
		logger:  logger.Named("protection-api"),
	}
}

// RegisterRoutes 注册API路由
func (api *ProtectionAPI) RegisterRoutes(router *mux.Router) {
	// 防护状态相关
	router.HandleFunc("/api/protection/status", api.GetStatus).Methods("GET")
	router.HandleFunc("/api/protection/config", api.GetConfig).Methods("GET")
	router.HandleFunc("/api/protection/health", api.GetHealth).Methods("GET")
	
	// 防护事件相关
	router.HandleFunc("/api/protection/events", api.GetEvents).Methods("GET")
	router.HandleFunc("/api/protection/events/{id}", api.GetEvent).Methods("GET")
	
	// 防护统计相关
	router.HandleFunc("/api/protection/stats", api.GetStats).Methods("GET")
	router.HandleFunc("/api/protection/report", api.GetReport).Methods("GET")
	
	// 防护管理相关
	router.HandleFunc("/api/protection/enable", api.EnableProtection).Methods("POST")
	router.HandleFunc("/api/protection/disable", api.DisableProtection).Methods("POST")
	router.HandleFunc("/api/protection/restart", api.RestartProtection).Methods("POST")
	
	// 防护组件相关
	router.HandleFunc("/api/protection/processes", api.GetProtectedProcesses).Methods("GET")
	router.HandleFunc("/api/protection/files", api.GetProtectedFiles).Methods("GET")
	router.HandleFunc("/api/protection/registry", api.GetProtectedRegistryKeys).Methods("GET")
	router.HandleFunc("/api/protection/services", api.GetProtectedServices).Methods("GET")
}

// GetStatus 获取防护状态
func (api *ProtectionAPI) GetStatus(w http.ResponseWriter, r *http.Request) {
	status := api.service.GetStatus()
	
	response := map[string]interface{}{
		"success": true,
		"data":    status,
	}
	
	api.writeJSONResponse(w, http.StatusOK, response)
}

// GetConfig 获取防护配置
func (api *ProtectionAPI) GetConfig(w http.ResponseWriter, r *http.Request) {
	config := api.service.GetConfig()
	
	// 隐藏敏感信息
	safeConfig := map[string]interface{}{
		"enabled":             config.Enabled,
		"level":               config.Level,
		"check_interval":      config.CheckInterval.String(),
		"restart_delay":       config.RestartDelay.String(),
		"max_restart_attempts": config.MaxRestartAttempts,
		"process_protection":  config.ProcessProtection.Enabled,
		"file_protection":     config.FileProtection.Enabled,
		"registry_protection": config.RegistryProtection.Enabled,
		"service_protection":  config.ServiceProtection.Enabled,
		"whitelist_enabled":   config.Whitelist.Enabled,
	}
	
	response := map[string]interface{}{
		"success": true,
		"data":    safeConfig,
	}
	
	api.writeJSONResponse(w, http.StatusOK, response)
}

// GetHealth 获取健康状态
func (api *ProtectionAPI) GetHealth(w http.ResponseWriter, r *http.Request) {
	healthChecker := NewProtectionHealthChecker(api.service, api.logger)
	health := healthChecker.CheckHealth()
	
	response := map[string]interface{}{
		"success": true,
		"data":    health,
	}
	
	api.writeJSONResponse(w, http.StatusOK, response)
}

// GetEvents 获取防护事件
func (api *ProtectionAPI) GetEvents(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	query := r.URL.Query()
	
	// 获取分页参数
	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}
	
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	
	// 获取过滤参数
	eventType := query.Get("type")
	action := query.Get("action")
	
	// 获取所有事件
	allEvents := api.service.GetEvents()
	
	// 过滤事件
	var filteredEvents []ProtectionEvent
	for _, event := range allEvents {
		if eventType != "" && string(event.Type) != eventType {
			continue
		}
		if action != "" && event.Action != action {
			continue
		}
		filteredEvents = append(filteredEvents, event)
	}
	
	// 分页
	total := len(filteredEvents)
	start := (page - 1) * limit
	end := start + limit
	
	if start >= total {
		filteredEvents = []ProtectionEvent{}
	} else {
		if end > total {
			end = total
		}
		filteredEvents = filteredEvents[start:end]
	}
	
	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"events": filteredEvents,
			"pagination": map[string]interface{}{
				"page":  page,
				"limit": limit,
				"total": total,
				"pages": (total + limit - 1) / limit,
			},
		},
	}
	
	api.writeJSONResponse(w, http.StatusOK, response)
}

// GetEvent 获取单个防护事件
func (api *ProtectionAPI) GetEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["id"]
	
	events := api.service.GetEvents()
	for _, event := range events {
		if event.ID == eventID {
			response := map[string]interface{}{
				"success": true,
				"data":    event,
			}
			api.writeJSONResponse(w, http.StatusOK, response)
			return
		}
	}
	
	api.writeErrorResponse(w, http.StatusNotFound, "事件未找到")
}

// GetStats 获取防护统计
func (api *ProtectionAPI) GetStats(w http.ResponseWriter, r *http.Request) {
	status := api.service.GetStatus()
	
	response := map[string]interface{}{
		"success": true,
		"data":    status.Stats,
	}
	
	api.writeJSONResponse(w, http.StatusOK, response)
}

// GetReport 获取防护报告
func (api *ProtectionAPI) GetReport(w http.ResponseWriter, r *http.Request) {
	reporter := NewProtectionReporter(api.service, api.logger)
	report := reporter.GenerateReport()
	
	response := map[string]interface{}{
		"success": true,
		"data":    report,
	}
	
	api.writeJSONResponse(w, http.StatusOK, response)
}

// EnableProtection 启用防护
func (api *ProtectionAPI) EnableProtection(w http.ResponseWriter, r *http.Request) {
	if api.service.IsEnabled() {
		api.writeErrorResponse(w, http.StatusBadRequest, "防护已经启用")
		return
	}
	
	// 注意：这里需要实现启用防护的逻辑
	// 由于当前架构限制，可能需要重启服务
	
	response := map[string]interface{}{
		"success": true,
		"message": "防护启用请求已接收，可能需要重启服务",
	}
	
	api.writeJSONResponse(w, http.StatusOK, response)
}

// DisableProtection 禁用防护
func (api *ProtectionAPI) DisableProtection(w http.ResponseWriter, r *http.Request) {
	if !api.service.IsEnabled() {
		api.writeErrorResponse(w, http.StatusBadRequest, "防护已经禁用")
		return
	}
	
	// 解析请求体
	var req struct {
		Reason    string `json:"reason"`
		Temporary bool   `json:"temporary"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeErrorResponse(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	
	// 记录禁用原因
	api.logger.Warn("收到防护禁用请求", "reason", req.Reason, "temporary", req.Temporary)
	
	// 注意：这里需要实现禁用防护的逻辑
	// 可以通过创建紧急禁用文件或修改配置
	
	response := map[string]interface{}{
		"success": true,
		"message": "防护禁用请求已接收",
	}
	
	api.writeJSONResponse(w, http.StatusOK, response)
}

// RestartProtection 重启防护
func (api *ProtectionAPI) RestartProtection(w http.ResponseWriter, r *http.Request) {
	api.logger.Info("收到防护重启请求")
	
	// 注意：这里需要实现重启防护的逻辑
	// 可能需要停止并重新启动防护服务
	
	response := map[string]interface{}{
		"success": true,
		"message": "防护重启请求已接收",
	}
	
	api.writeJSONResponse(w, http.StatusOK, response)
}

// GetProtectedProcesses 获取受保护的进程
func (api *ProtectionAPI) GetProtectedProcesses(w http.ResponseWriter, r *http.Request) {
	config := api.service.GetConfig()
	
	processes := map[string]interface{}{
		"enabled":   config.ProcessProtection.Enabled,
		"processes": config.ProcessProtection.ProtectedProcesses,
		"settings": map[string]interface{}{
			"monitor_children": config.ProcessProtection.MonitorChildren,
			"prevent_debug":    config.ProcessProtection.PreventDebug,
			"prevent_dump":     config.ProcessProtection.PreventDump,
		},
	}
	
	response := map[string]interface{}{
		"success": true,
		"data":    processes,
	}
	
	api.writeJSONResponse(w, http.StatusOK, response)
}

// GetProtectedFiles 获取受保护的文件
func (api *ProtectionAPI) GetProtectedFiles(w http.ResponseWriter, r *http.Request) {
	config := api.service.GetConfig()
	
	files := map[string]interface{}{
		"enabled": config.FileProtection.Enabled,
		"files":   config.FileProtection.ProtectedFiles,
		"dirs":    config.FileProtection.ProtectedDirs,
		"settings": map[string]interface{}{
			"check_integrity": config.FileProtection.CheckIntegrity,
			"backup_enabled":  config.FileProtection.BackupEnabled,
			"backup_dir":      config.FileProtection.BackupDir,
		},
	}
	
	response := map[string]interface{}{
		"success": true,
		"data":    files,
	}
	
	api.writeJSONResponse(w, http.StatusOK, response)
}

// GetProtectedRegistryKeys 获取受保护的注册表键
func (api *ProtectionAPI) GetProtectedRegistryKeys(w http.ResponseWriter, r *http.Request) {
	config := api.service.GetConfig()
	
	registry := map[string]interface{}{
		"enabled": config.RegistryProtection.Enabled,
		"keys":    config.RegistryProtection.ProtectedKeys,
		"settings": map[string]interface{}{
			"monitor_changes": config.RegistryProtection.MonitorChanges,
		},
	}
	
	response := map[string]interface{}{
		"success": true,
		"data":    registry,
	}
	
	api.writeJSONResponse(w, http.StatusOK, response)
}

// GetProtectedServices 获取受保护的服务
func (api *ProtectionAPI) GetProtectedServices(w http.ResponseWriter, r *http.Request) {
	config := api.service.GetConfig()
	
	services := map[string]interface{}{
		"enabled":      config.ServiceProtection.Enabled,
		"service_name": config.ServiceProtection.ServiceName,
		"settings": map[string]interface{}{
			"auto_restart":     config.ServiceProtection.AutoRestart,
			"prevent_disable":  config.ServiceProtection.PreventDisable,
		},
	}
	
	response := map[string]interface{}{
		"success": true,
		"data":    services,
	}
	
	api.writeJSONResponse(w, http.StatusOK, response)
}

// writeJSONResponse 写入JSON响应
func (api *ProtectionAPI) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		api.logger.Error("写入JSON响应失败", "error", err)
	}
}

// writeErrorResponse 写入错误响应
func (api *ProtectionAPI) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	response := map[string]interface{}{
		"success": false,
		"error":   message,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	
	api.writeJSONResponse(w, statusCode, response)
}

// ProtectionWebUI 自我防护Web界面
type ProtectionWebUI struct {
	api    *ProtectionAPI
	logger hclog.Logger
}

// NewProtectionWebUI 创建自我防护Web界面
func NewProtectionWebUI(api *ProtectionAPI, logger hclog.Logger) *ProtectionWebUI {
	return &ProtectionWebUI{
		api:    api,
		logger: logger.Named("protection-webui"),
	}
}

// RegisterRoutes 注册Web界面路由
func (ui *ProtectionWebUI) RegisterRoutes(router *mux.Router) {
	// 静态文件服务
	router.PathPrefix("/protection/").Handler(http.StripPrefix("/protection/", http.FileServer(http.Dir("web/protection/"))))
	
	// 主页面
	router.HandleFunc("/protection", ui.IndexPage).Methods("GET")
	router.HandleFunc("/protection/dashboard", ui.DashboardPage).Methods("GET")
	router.HandleFunc("/protection/events", ui.EventsPage).Methods("GET")
	router.HandleFunc("/protection/config", ui.ConfigPage).Methods("GET")
}

// IndexPage 主页面
func (ui *ProtectionWebUI) IndexPage(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/protection/dashboard", http.StatusFound)
}

// DashboardPage 仪表板页面
func (ui *ProtectionWebUI) DashboardPage(w http.ResponseWriter, r *http.Request) {
	// 这里应该渲染仪表板HTML页面
	// 简化实现，返回基本信息
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <title>Kennel自我防护 - 仪表板</title>
    <meta charset="utf-8">
</head>
<body>
    <h1>Kennel自我防护仪表板</h1>
    <p>防护状态: <span id="status">加载中...</span></p>
    <p>防护级别: <span id="level">加载中...</span></p>
    <p>运行时间: <span id="uptime">加载中...</span></p>
    
    <script>
        fetch('/api/protection/status')
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    document.getElementById('status').textContent = data.data.enabled ? '已启用' : '已禁用';
                    document.getElementById('level').textContent = data.data.level;
                    document.getElementById('uptime').textContent = new Date(data.data.start_time).toLocaleString();
                }
            });
    </script>
</body>
</html>
	`)
}

// EventsPage 事件页面
func (ui *ProtectionWebUI) EventsPage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <title>Kennel自我防护 - 事件</title>
    <meta charset="utf-8">
</head>
<body>
    <h1>防护事件</h1>
    <div id="events">加载中...</div>
    
    <script>
        fetch('/api/protection/events')
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    const eventsDiv = document.getElementById('events');
                    eventsDiv.innerHTML = '';
                    
                    data.data.events.forEach(event => {
                        const eventDiv = document.createElement('div');
                        eventDiv.innerHTML = '<p><strong>' + event.type + '</strong> - ' + event.action + ' - ' + event.target + ' (' + new Date(event.timestamp).toLocaleString() + ')</p>';
                        eventsDiv.appendChild(eventDiv);
                    });
                }
            });
    </script>
</body>
</html>
	`)
}

// ConfigPage 配置页面
func (ui *ProtectionWebUI) ConfigPage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <title>Kennel自我防护 - 配置</title>
    <meta charset="utf-8">
</head>
<body>
    <h1>防护配置</h1>
    <div id="config">加载中...</div>
    
    <script>
        fetch('/api/protection/config')
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    const configDiv = document.getElementById('config');
                    configDiv.innerHTML = '<pre>' + JSON.stringify(data.data, null, 2) + '</pre>';
                }
            });
    </script>
</body>
</html>
	`)
}
