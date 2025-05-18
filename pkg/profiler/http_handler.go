package profiler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// HTTPHandler 性能分析HTTP处理器
type HTTPHandler struct {
	profiler Profiler
	basePath string
}

// NewHTTPHandler 创建性能分析HTTP处理器
func NewHTTPHandler(profiler Profiler, basePath string) *HTTPHandler {
	if basePath == "" {
		basePath = "/debug/pprof"
	}
	return &HTTPHandler{
		profiler: profiler,
		basePath: basePath,
	}
}

// RegisterHandlers 注册HTTP处理器
func (h *HTTPHandler) RegisterHandlers(mux *http.ServeMux) {
	// 索引页
	mux.HandleFunc(h.basePath+"/", h.indexHandler)

	// 启动性能分析
	mux.HandleFunc(h.basePath+"/start/", h.startHandler)

	// 停止性能分析
	mux.HandleFunc(h.basePath+"/stop/", h.stopHandler)

	// 获取性能分析结果
	mux.HandleFunc(h.basePath+"/results", h.resultsHandler)

	// 获取正在运行的性能分析
	mux.HandleFunc(h.basePath+"/running", h.runningHandler)

	// 获取性能分析数据
	mux.HandleFunc(h.basePath+"/profile/", h.profileHandler)

	// 分析性能分析数据
	mux.HandleFunc(h.basePath+"/analyze/", h.analyzeHandler)

	// 清理性能分析数据
	mux.HandleFunc(h.basePath+"/cleanup", h.cleanupHandler)

	// 直接访问各种性能分析类型
	mux.HandleFunc(h.basePath+"/cpu", h.cpuHandler)
	mux.HandleFunc(h.basePath+"/heap", h.heapHandler)
	mux.HandleFunc(h.basePath+"/block", h.blockHandler)
	mux.HandleFunc(h.basePath+"/goroutine", h.goroutineHandler)
	mux.HandleFunc(h.basePath+"/threadcreate", h.threadcreateHandler)
	mux.HandleFunc(h.basePath+"/mutex", h.mutexHandler)
	mux.HandleFunc(h.basePath+"/trace", h.traceHandler)
	mux.HandleFunc(h.basePath+"/allocs", h.allocsHandler)
}

// indexHandler 处理索引页请求
func (h *HTTPHandler) indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<html>
<head>
	<title>性能分析</title>
	<style>
		body { font-family: Arial, sans-serif; margin: 20px; }
		h1, h2 { color: #333; }
		table { border-collapse: collapse; width: 100%; }
		th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
		th { background-color: #f2f2f2; }
		tr:nth-child(even) { background-color: #f9f9f9; }
		.button { 
			display: inline-block; 
			padding: 8px 16px; 
			background-color: #4CAF50; 
			color: white; 
			text-decoration: none; 
			border-radius: 4px; 
			margin: 5px;
		}
		.button:hover { background-color: #45a049; }
		.button.stop { background-color: #f44336; }
		.button.stop:hover { background-color: #d32f2f; }
	</style>
</head>
<body>
	<h1>性能分析</h1>
	
	<h2>可用的性能分析类型</h2>
	<ul>
		<li><a href="%[1]s/cpu?seconds=30">CPU分析</a> - 收集CPU使用情况</li>
		<li><a href="%[1]s/heap">堆分析</a> - 收集堆内存使用情况</li>
		<li><a href="%[1]s/block">阻塞分析</a> - 收集goroutine阻塞情况</li>
		<li><a href="%[1]s/goroutine">协程分析</a> - 收集goroutine信息</li>
		<li><a href="%[1]s/threadcreate">线程创建分析</a> - 收集线程创建情况</li>
		<li><a href="%[1]s/mutex">互斥锁分析</a> - 收集互斥锁争用情况</li>
		<li><a href="%[1]s/trace?seconds=5">执行追踪</a> - 收集程序执行追踪</li>
		<li><a href="%[1]s/allocs">内存分配分析</a> - 收集内存分配情况</li>
	</ul>

	<h2>正在运行的性能分析</h2>
	<div id="running-profiles">加载中...</div>

	<h2>性能分析结果</h2>
	<div id="profile-results">加载中...</div>

	<script>
		// 加载正在运行的性能分析
		function loadRunningProfiles() {
			fetch('%[1]s/running')
				.then(response => response.json())
				.then(data => {
					const container = document.getElementById('running-profiles');
					if (Object.keys(data).length === 0) {
						container.innerHTML = '<p>没有正在运行的性能分析</p>';
						return;
					}

					let html = '<table>';
					html += '<tr><th>类型</th><th>开始时间</th><th>持续时间</th><th>操作</th></tr>';
					
					for (const [type, options] of Object.entries(data)) {
						html += '<tr>';
						html += '<td>' + type + '</td>';
						html += '<td>' + new Date().toLocaleString() + '</td>';
						html += '<td>' + options.Duration + '</td>';
						html += '<td><a href="%[1]s/stop/' + type + '" class="button stop">停止</a></td>';
						html += '</tr>';
					}
					
					html += '</table>';
					container.innerHTML = html;
				})
				.catch(error => {
					console.error('加载正在运行的性能分析失败:', error);
					document.getElementById('running-profiles').innerHTML = '<p>加载失败: ' + error.message + '</p>';
				});
		}

		// 加载性能分析结果
		function loadProfileResults() {
			fetch('%[1]s/results')
				.then(response => response.json())
				.then(data => {
					const container = document.getElementById('profile-results');
					if (data.length === 0) {
						container.innerHTML = '<p>没有性能分析结果</p>';
						return;
					}

					let html = '<table>';
					html += '<tr><th>类型</th><th>开始时间</th><th>结束时间</th><th>持续时间</th><th>大小</th><th>操作</th></tr>';
					
					for (const result of data) {
						html += '<tr>';
						html += '<td>' + result.Type + '</td>';
						html += '<td>' + new Date(result.StartTime).toLocaleString() + '</td>';
						html += '<td>' + new Date(result.EndTime).toLocaleString() + '</td>';
						html += '<td>' + result.Duration + '</td>';
						html += '<td>' + formatBytes(result.Size) + '</td>';
						html += '<td>';
						html += '<a href="%[1]s/profile/' + result.Type + '?format=pprof" class="button">下载</a> ';
						html += '<a href="%[1]s/profile/' + result.Type + '?format=svg" class="button">SVG</a> ';
						html += '<a href="%[1]s/analyze/' + result.Type + '" class="button">分析</a>';
						html += '</td>';
						html += '</tr>';
					}
					
					html += '</table>';
					container.innerHTML = html;
				})
				.catch(error => {
					console.error('加载性能分析结果失败:', error);
					document.getElementById('profile-results').innerHTML = '<p>加载失败: ' + error.message + '</p>';
				});
		}

		// 格式化字节数
		function formatBytes(bytes, decimals = 2) {
			if (bytes === 0) return '0 Bytes';
			
			const k = 1024;
			const dm = decimals < 0 ? 0 : decimals;
			const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];
			
			const i = Math.floor(Math.log(bytes) / Math.log(k));
			
			return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
		}

		// 页面加载时执行
		document.addEventListener('DOMContentLoaded', function() {
			loadRunningProfiles();
			loadProfileResults();
			
			// 定时刷新
			setInterval(loadRunningProfiles, 5000);
			setInterval(loadProfileResults, 5000);
		});
	</script>
</body>
</html>`, h.basePath)
}

// startHandler 处理启动性能分析请求
func (h *HTTPHandler) startHandler(w http.ResponseWriter, r *http.Request) {
	// 获取性能分析类型
	profileType := ProfileType(r.URL.Path[len(h.basePath+"/start/"):])
	if profileType == "" {
		http.Error(w, "未指定性能分析类型", http.StatusBadRequest)
		return
	}

	// 解析参数
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("解析参数失败: %v", err), http.StatusBadRequest)
		return
	}

	// 创建选项
	options := DefaultProfileOptions()

	// 设置持续时间
	if seconds := r.Form.Get("seconds"); seconds != "" {
		duration, err := strconv.Atoi(seconds)
		if err != nil {
			http.Error(w, fmt.Sprintf("无效的持续时间: %v", err), http.StatusBadRequest)
			return
		}
		options.Duration = time.Duration(duration) * time.Second
	}

	// 设置采样率
	if rate := r.Form.Get("rate"); rate != "" {
		rateValue, err := strconv.Atoi(rate)
		if err != nil {
			http.Error(w, fmt.Sprintf("无效的采样率: %v", err), http.StatusBadRequest)
			return
		}
		options.Rate = rateValue
	}

	// 设置输出格式
	if format := r.Form.Get("format"); format != "" {
		options.Format = ProfileFormat(format)
	}

	// 启动性能分析
	if err := h.profiler.Start(r.Context(), profileType, options); err != nil {
		http.Error(w, fmt.Sprintf("启动性能分析失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回结果
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("已启动%s性能分析", profileType),
		"type":    profileType,
		"options": options,
	})
}

// stopHandler 处理停止性能分析请求
func (h *HTTPHandler) stopHandler(w http.ResponseWriter, r *http.Request) {
	// 获取性能分析类型
	profileType := ProfileType(r.URL.Path[len(h.basePath+"/stop/"):])
	if profileType == "" {
		http.Error(w, "未指定性能分析类型", http.StatusBadRequest)
		return
	}

	// 停止性能分析
	result, err := h.profiler.Stop(profileType)
	if err != nil {
		http.Error(w, fmt.Sprintf("停止性能分析失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回结果
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("已停止%s性能分析", profileType),
		"result":  result,
	})
}

// resultsHandler 处理获取性能分析结果请求
func (h *HTTPHandler) resultsHandler(w http.ResponseWriter, r *http.Request) {
	// 获取性能分析结果
	results := h.profiler.GetResults()

	// 返回结果
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// runningHandler 处理获取正在运行的性能分析请求
func (h *HTTPHandler) runningHandler(w http.ResponseWriter, r *http.Request) {
	// 获取正在运行的性能分析
	profiles := h.profiler.GetRunningProfiles()

	// 返回结果
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profiles)
}

// profileHandler 处理获取性能分析数据请求
func (h *HTTPHandler) profileHandler(w http.ResponseWriter, r *http.Request) {
	// 获取性能分析类型
	profileType := ProfileType(r.URL.Path[len(h.basePath+"/profile/"):])
	if profileType == "" {
		http.Error(w, "未指定性能分析类型", http.StatusBadRequest)
		return
	}

	// 解析参数
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("解析参数失败: %v", err), http.StatusBadRequest)
		return
	}

	// 获取输出格式
	format := ProfileFormatPprof
	if formatStr := r.Form.Get("format"); formatStr != "" {
		format = ProfileFormat(formatStr)
	}

	// 设置内容类型
	switch format {
	case ProfileFormatPprof:
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.pprof", profileType))
	case ProfileFormatJSON:
		w.Header().Set("Content-Type", "application/json")
	case ProfileFormatText:
		w.Header().Set("Content-Type", "text/plain")
	case ProfileFormatSVG:
		w.Header().Set("Content-Type", "image/svg+xml")
	case ProfileFormatPDF:
		w.Header().Set("Content-Type", "application/pdf")
	case ProfileFormatHTML:
		w.Header().Set("Content-Type", "text/html")
	}

	// 写入性能分析数据
	if err := h.profiler.WriteProfile(profileType, format, w); err != nil {
		http.Error(w, fmt.Sprintf("获取性能分析数据失败: %v", err), http.StatusInternalServerError)
		return
	}
}

// analyzeHandler 处理分析性能分析数据请求
func (h *HTTPHandler) analyzeHandler(w http.ResponseWriter, r *http.Request) {
	// 获取性能分析类型
	profileType := ProfileType(r.URL.Path[len(h.basePath+"/analyze/"):])
	if profileType == "" {
		http.Error(w, "未指定性能分析类型", http.StatusBadRequest)
		return
	}

	// 获取最新的性能分析结果
	result := h.profiler.GetResult(profileType)
	if result == nil {
		http.Error(w, fmt.Sprintf("未找到%s性能分析结果", profileType), http.StatusNotFound)
		return
	}

	// 分析性能分析数据
	analysis, err := h.profiler.AnalyzeProfile(profileType, result.FilePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("分析性能分析数据失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回结果
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

// cleanupHandler 处理清理性能分析数据请求
func (h *HTTPHandler) cleanupHandler(w http.ResponseWriter, r *http.Request) {
	// 解析参数
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("解析参数失败: %v", err), http.StatusBadRequest)
		return
	}

	// 获取清理时间
	olderThan := 24 * time.Hour
	if days := r.Form.Get("days"); days != "" {
		daysValue, err := strconv.Atoi(days)
		if err != nil {
			http.Error(w, fmt.Sprintf("无效的天数: %v", err), http.StatusBadRequest)
			return
		}
		olderThan = time.Duration(daysValue) * 24 * time.Hour
	}

	// 清理性能分析数据
	if err := h.profiler.Cleanup(olderThan); err != nil {
		http.Error(w, fmt.Sprintf("清理性能分析数据失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回结果
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("已清理%v前的性能分析数据", olderThan),
	})
}

// cpuHandler 处理CPU分析请求
func (h *HTTPHandler) cpuHandler(w http.ResponseWriter, r *http.Request) {
	h.handleProfileRequest(w, r, ProfileTypeCPU)
}

// heapHandler 处理堆分析请求
func (h *HTTPHandler) heapHandler(w http.ResponseWriter, r *http.Request) {
	h.handleProfileRequest(w, r, ProfileTypeHeap)
}

// blockHandler 处理阻塞分析请求
func (h *HTTPHandler) blockHandler(w http.ResponseWriter, r *http.Request) {
	h.handleProfileRequest(w, r, ProfileTypeBlock)
}

// goroutineHandler 处理协程分析请求
func (h *HTTPHandler) goroutineHandler(w http.ResponseWriter, r *http.Request) {
	h.handleProfileRequest(w, r, ProfileTypeGoroutine)
}

// threadcreateHandler 处理线程创建分析请求
func (h *HTTPHandler) threadcreateHandler(w http.ResponseWriter, r *http.Request) {
	h.handleProfileRequest(w, r, ProfileTypeThreadcreate)
}

// mutexHandler 处理互斥锁分析请求
func (h *HTTPHandler) mutexHandler(w http.ResponseWriter, r *http.Request) {
	h.handleProfileRequest(w, r, ProfileTypeMutex)
}

// traceHandler 处理执行追踪请求
func (h *HTTPHandler) traceHandler(w http.ResponseWriter, r *http.Request) {
	h.handleProfileRequest(w, r, ProfileTypeTrace)
}

// allocsHandler 处理内存分配分析请求
func (h *HTTPHandler) allocsHandler(w http.ResponseWriter, r *http.Request) {
	h.handleProfileRequest(w, r, ProfileTypeAllocs)
}

// handleProfileRequest 处理性能分析请求
func (h *HTTPHandler) handleProfileRequest(w http.ResponseWriter, r *http.Request, profileType ProfileType) {
	// 解析参数
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("解析参数失败: %v", err), http.StatusBadRequest)
		return
	}

	// 获取输出格式
	format := ProfileFormatPprof
	if formatStr := r.Form.Get("format"); formatStr != "" {
		format = ProfileFormat(formatStr)
	}

	// 如果已经有结果，直接返回
	if result := h.profiler.GetResult(profileType); result != nil && !h.profiler.IsRunning(profileType) {
		// 设置内容类型
		switch format {
		case ProfileFormatPprof:
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.pprof", profileType))
		case ProfileFormatJSON:
			w.Header().Set("Content-Type", "application/json")
		case ProfileFormatText:
			w.Header().Set("Content-Type", "text/plain")
		case ProfileFormatSVG:
			w.Header().Set("Content-Type", "image/svg+xml")
		case ProfileFormatPDF:
			w.Header().Set("Content-Type", "application/pdf")
		case ProfileFormatHTML:
			w.Header().Set("Content-Type", "text/html")
		}

		// 写入性能分析数据
		if err := h.profiler.WriteProfile(profileType, format, w); err != nil {
			http.Error(w, fmt.Sprintf("获取性能分析数据失败: %v", err), http.StatusInternalServerError)
		}
		return
	}

	// 创建选项
	options := DefaultProfileOptions()
	options.Format = format

	// 设置持续时间
	if seconds := r.Form.Get("seconds"); seconds != "" {
		duration, err := strconv.Atoi(seconds)
		if err != nil {
			http.Error(w, fmt.Sprintf("无效的持续时间: %v", err), http.StatusBadRequest)
			return
		}
		options.Duration = time.Duration(duration) * time.Second
	}

	// 设置采样率
	if rate := r.Form.Get("rate"); rate != "" {
		rateValue, err := strconv.Atoi(rate)
		if err != nil {
			http.Error(w, fmt.Sprintf("无效的采样率: %v", err), http.StatusBadRequest)
			return
		}
		options.Rate = rateValue
	}

	// 创建上下文
	ctx, cancel := context.WithTimeout(r.Context(), options.Duration+5*time.Second)
	defer cancel()

	// 启动性能分析
	if err := h.profiler.Start(ctx, profileType, options); err != nil {
		http.Error(w, fmt.Sprintf("启动性能分析失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 等待性能分析完成
	time.Sleep(options.Duration)

	// 停止性能分析
	result, err := h.profiler.Stop(profileType)
	if err != nil {
		http.Error(w, fmt.Sprintf("停止性能分析失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 设置内容类型
	switch format {
	case ProfileFormatPprof:
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.pprof", profileType))
	case ProfileFormatJSON:
		w.Header().Set("Content-Type", "application/json")
	case ProfileFormatText:
		w.Header().Set("Content-Type", "text/plain")
	case ProfileFormatSVG:
		w.Header().Set("Content-Type", "image/svg+xml")
	case ProfileFormatPDF:
		w.Header().Set("Content-Type", "application/pdf")
	case ProfileFormatHTML:
		w.Header().Set("Content-Type", "text/html")
	}

	// 写入性能分析数据
	if err := h.profiler.WriteProfile(profileType, format, w); err != nil {
		http.Error(w, fmt.Sprintf("获取性能分析数据失败: %v", err), http.StatusInternalServerError)
		return
	}
}
