package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lomehong/kennel/app/dlp/analyzer"
	"github.com/lomehong/kennel/app/dlp/engine"
	"github.com/lomehong/kennel/app/dlp/executor"
	"github.com/lomehong/kennel/app/dlp/interceptor"
	"github.com/lomehong/kennel/app/dlp/parser"

	"github.com/lomehong/kennel/pkg/core/plugin"
	"github.com/lomehong/kennel/pkg/logging"
	sdk "github.com/lomehong/kennel/pkg/sdk/go"
)

// DLPModule 实现了数据防泄漏模块
type DLPModule struct {
	*sdk.BaseModule

	// 日志记录器
	Logger logging.Logger

	// 传统组件（保持兼容性）
	ruleManager   *RuleManager
	alertManager  *AlertManager
	scanner       *Scanner
	monitorCtx    context.Context
	monitorCancel context.CancelFunc

	// 新的DLP核心组件
	interceptorManager interceptor.InterceptorManager
	protocolManager    parser.ProtocolManager
	analysisManager    analyzer.AnalysisManager
	policyEngine       engine.PolicyEngine
	executionManager   executor.ExecutionManager

	// 配置和状态
	dlpConfig    *DLPConfig
	running      bool
	mu           sync.RWMutex
	processingCh chan *ProcessingTask
	stopCh       chan struct{}
}

// DLPConfig DLP模块配置
type DLPConfig struct {
	EnableNetworkMonitoring   bool                          `yaml:"enable_network_monitoring" json:"enable_network_monitoring"`
	EnableFileMonitoring      bool                          `yaml:"enable_file_monitoring" json:"enable_file_monitoring"`
	EnableClipboardMonitoring bool                          `yaml:"enable_clipboard_monitoring" json:"enable_clipboard_monitoring"`
	MonitoredDirectories      []string                      `yaml:"monitored_directories" json:"monitored_directories"`
	MonitoredFileTypes        []string                      `yaml:"monitored_file_types" json:"monitored_file_types"`
	NetworkProtocols          []string                      `yaml:"network_protocols" json:"network_protocols"`
	InterceptorConfig         interceptor.InterceptorConfig `yaml:"interceptor_config" json:"interceptor_config"`
	ParserConfig              parser.ParserConfig           `yaml:"parser_config" json:"parser_config"`
	AnalyzerConfig            analyzer.AnalyzerConfig       `yaml:"analyzer_config" json:"analyzer_config"`
	EngineConfig              engine.PolicyEngineConfig     `yaml:"engine_config" json:"engine_config"`
	ExecutorConfig            executor.ExecutorConfig       `yaml:"executor_config" json:"executor_config"`
	MaxConcurrency            int                           `yaml:"max_concurrency" json:"max_concurrency"`
	BufferSize                int                           `yaml:"buffer_size" json:"buffer_size"`

	// OCR和ML相关配置
	OCRConfig            map[string]interface{} `yaml:"ocr_config" json:"ocr_config"`
	MLConfig             map[string]interface{} `yaml:"ml_config" json:"ml_config"`
	FileDetectionConfig  map[string]interface{} `yaml:"file_detection_config" json:"file_detection_config"`
	OCRPerformanceConfig map[string]interface{} `yaml:"ocr_performance_config" json:"ocr_performance_config"`
	OCRLoggingConfig     map[string]interface{} `yaml:"ocr_logging_config" json:"ocr_logging_config"`
}

// ProcessingTask 处理任务
type ProcessingTask struct {
	ID        string
	Timestamp time.Time
	Packet    *interceptor.PacketInfo
	Context   context.Context
}

// NewDLPModule 创建一个新的数据防泄漏模块
func NewDLPModule(logger logging.Logger) *DLPModule {
	// 创建基础模块
	base := sdk.NewBaseModule(
		"dlp",
		"数据防泄漏插件",
		"2.0.0",
		"数据防泄漏模块，用于检测和防止敏感数据泄漏",
	)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 创建模块
	module := &DLPModule{
		BaseModule:    base,
		monitorCtx:    ctx,
		monitorCancel: cancel,
		processingCh:  make(chan *ProcessingTask, 200), // 减少处理通道大小
		stopCh:        make(chan struct{}),
	}

	// 设置日志记录器
	if logger != nil {
		module.Logger = logger
	}

	return module
}

// Init 初始化模块
func (m *DLPModule) Init(ctx context.Context, config *plugin.ModuleConfig) error {
	// 调用基类初始化
	if err := m.BaseModule.Init(ctx, config); err != nil {
		return err
	}

	m.Logger.Info("初始化数据防泄漏模块v2.0")

	// 解析DLP配置
	if err := m.parseDLPConfig(config); err != nil {
		return fmt.Errorf("解析DLP配置失败: %w", err)
	}

	// 初始化核心组件
	if err := m.initializeCoreComponents(); err != nil {
		return fmt.Errorf("初始化核心组件失败: %w", err)
	}

	// 初始化传统组件（保持兼容性）
	if err := m.initializeLegacyComponents(); err != nil {
		m.Logger.Warn("初始化传统组件失败", "error", err)
		// 不返回错误，允许新架构独立运行
	}

	m.Logger.Info("数据防泄漏模块初始化完成")
	return nil
}

// parseDLPConfig 解析DLP配置
func (m *DLPModule) parseDLPConfig(config *plugin.ModuleConfig) error {
	// 获取字符串切片的辅助函数
	getStringSlice := func(key string, defaultValue []string) []string {
		if val, ok := config.Settings[key]; ok {
			if slice, ok := val.([]interface{}); ok {
				result := make([]string, len(slice))
				for i, v := range slice {
					if str, ok := v.(string); ok {
						result[i] = str
					}
				}
				return result
			}
		}
		return defaultValue
	}

	m.dlpConfig = &DLPConfig{
		EnableNetworkMonitoring:   sdk.GetConfigBool(config.Settings, "monitor_network", true),
		EnableFileMonitoring:      sdk.GetConfigBool(config.Settings, "monitor_files", true),
		EnableClipboardMonitoring: sdk.GetConfigBool(config.Settings, "monitor_clipboard", true),
		MonitoredDirectories:      getStringSlice("monitored_directories", []string{}),
		MonitoredFileTypes:        getStringSlice("monitored_file_types", []string{}),
		NetworkProtocols:          getStringSlice("network_protocols", []string{"http", "https", "ftp", "smtp"}),
		MaxConcurrency:            sdk.GetConfigInt(config.Settings, "max_concurrency", 4), // 减少并发数
		BufferSize:                sdk.GetConfigInt(config.Settings, "buffer_size", 500),   // 减少缓冲区大小
	}

	// 创建增强日志记录器用于子组件
	logConfig := logging.DefaultLogConfig()
	logConfig.Level = logging.LogLevelInfo
	enhancedLogger, err := logging.NewEnhancedLogger(logConfig)
	if err != nil {
		return fmt.Errorf("创建增强日志记录器失败: %w", err)
	}

	// 设置子组件配置
	m.dlpConfig.InterceptorConfig = interceptor.DefaultInterceptorConfig()
	m.dlpConfig.InterceptorConfig.Logger = enhancedLogger.Named("interceptor")

	m.dlpConfig.ParserConfig = parser.DefaultParserConfig()
	m.dlpConfig.ParserConfig.Logger = enhancedLogger.Named("parser")

	m.dlpConfig.AnalyzerConfig = analyzer.DefaultAnalyzerConfig()
	m.dlpConfig.AnalyzerConfig.Logger = enhancedLogger.Named("analyzer")

	m.dlpConfig.EngineConfig = engine.DefaultPolicyEngineConfig()
	m.dlpConfig.EngineConfig.Logger = enhancedLogger.Named("engine")

	m.dlpConfig.ExecutorConfig = executor.DefaultExecutorConfig()
	m.dlpConfig.ExecutorConfig.Logger = enhancedLogger.Named("executor")

	// 解析OCR和ML配置
	if err := m.parseOCRAndMLConfig(config); err != nil {
		m.Logger.Warn("解析OCR和ML配置失败", "error", err)
		// 不返回错误，允许系统继续运行
	}

	return nil
}

// parseOCRAndMLConfig 解析OCR和ML配置
func (m *DLPModule) parseOCRAndMLConfig(config *plugin.ModuleConfig) error {
	// 从主配置文件中读取OCR配置
	if ocrConfig, ok := config.Settings["ocr"].(map[string]interface{}); ok {
		m.dlpConfig.OCRConfig = ocrConfig
		m.Logger.Info("已加载OCR配置", "enabled", ocrConfig["enabled"])
	} else {
		m.Logger.Info("未找到OCR配置，使用默认设置")
		m.dlpConfig.OCRConfig = map[string]interface{}{
			"enabled": false,
		}
	}

	// 从主配置文件中读取ML配置
	if mlConfig, ok := config.Settings["ml"].(map[string]interface{}); ok {
		m.dlpConfig.MLConfig = mlConfig
		m.Logger.Info("已加载ML配置", "enabled", mlConfig["enabled"])
	} else {
		m.Logger.Info("未找到ML配置，使用默认设置")
		m.dlpConfig.MLConfig = map[string]interface{}{
			"enabled": false,
		}
	}

	// 从主配置文件中读取文件检测配置
	if fileDetectionConfig, ok := config.Settings["file_detection"].(map[string]interface{}); ok {
		m.dlpConfig.FileDetectionConfig = fileDetectionConfig
		m.Logger.Info("已加载文件检测配置", "enabled", fileDetectionConfig["enabled"])
	} else {
		m.Logger.Info("未找到文件检测配置，使用默认设置")
		m.dlpConfig.FileDetectionConfig = map[string]interface{}{
			"enabled": true,
		}
	}

	// 从主配置文件中读取OCR性能配置
	if ocrPerfConfig, ok := config.Settings["ocr_performance"].(map[string]interface{}); ok {
		m.dlpConfig.OCRPerformanceConfig = ocrPerfConfig
		m.Logger.Info("已加载OCR性能配置")
	}

	// 从主配置文件中读取OCR日志配置
	if ocrLogConfig, ok := config.Settings["ocr_logging"].(map[string]interface{}); ok {
		m.dlpConfig.OCRLoggingConfig = ocrLogConfig
		m.Logger.Info("已加载OCR日志配置")
	}

	return nil
}

// initializeCoreComponents 初始化核心组件
func (m *DLPModule) initializeCoreComponents() error {
	m.Logger.Info("初始化DLP核心组件")

	// 使用配置中的日志记录器
	logger := m.dlpConfig.InterceptorConfig.Logger

	// 创建拦截器管理器
	m.interceptorManager = interceptor.NewInterceptorManager(logger)

	// 创建协议解析管理器
	m.protocolManager = parser.NewProtocolManager(m.dlpConfig.ParserConfig.Logger, m.dlpConfig.ParserConfig)

	// 创建内容分析管理器
	m.analysisManager = analyzer.NewAnalysisManager(m.dlpConfig.AnalyzerConfig.Logger, m.dlpConfig.AnalyzerConfig)

	// 创建策略引擎
	m.policyEngine = engine.NewPolicyEngine(m.dlpConfig.EngineConfig.Logger, m.dlpConfig.EngineConfig)

	// 创建执行管理器
	m.executionManager = executor.NewExecutionManager(m.dlpConfig.ExecutorConfig.Logger, m.dlpConfig.ExecutorConfig)

	// 注册协议解析器
	if err := m.registerProtocolParsers(); err != nil {
		return fmt.Errorf("注册协议解析器失败: %w", err)
	}

	// 注册内容分析器
	textAnalyzer := analyzer.NewTextAnalyzer(m.dlpConfig.AnalyzerConfig.Logger)
	if err := m.analysisManager.RegisterAnalyzer(textAnalyzer); err != nil {
		return fmt.Errorf("注册文本分析器失败: %w", err)
	}

	// 配置OCR和ML功能
	if err := m.configureOCRAndML(textAnalyzer); err != nil {
		m.Logger.Warn("配置OCR和ML功能失败", "error", err)
		// 不返回错误，允许系统继续运行
	}

	m.Logger.Info("DLP核心组件初始化完成")
	return nil
}

// configureOCRAndML 配置OCR和ML功能
func (m *DLPModule) configureOCRAndML(textAnalyzer analyzer.ContentAnalyzer) error {
	// 类型断言获取TextAnalyzer
	ta, ok := textAnalyzer.(*analyzer.TextAnalyzer)
	if !ok {
		return fmt.Errorf("无法转换为TextAnalyzer类型")
	}

	// 配置OCR功能
	if m.dlpConfig.OCRConfig != nil {
		if enabled, ok := m.dlpConfig.OCRConfig["enabled"].(bool); ok && enabled {
			m.Logger.Info("启用OCR功能")

			// 构建OCR配置
			ocrConfig := make(map[string]interface{})

			// 从tesseract子配置中提取参数
			if tesseractConfig, ok := m.dlpConfig.OCRConfig["tesseract"].(map[string]interface{}); ok {
				// 语言配置
				if languages, ok := tesseractConfig["languages"].([]interface{}); ok {
					langStrings := make([]string, len(languages))
					for i, lang := range languages {
						if langStr, ok := lang.(string); ok {
							langStrings[i] = langStr
						}
					}
					ocrConfig["languages"] = langStrings
				}

				// 其他配置参数
				if timeoutSec, ok := tesseractConfig["timeout_seconds"].(int); ok {
					ocrConfig["timeout_seconds"] = timeoutSec
				}
				if maxSize, ok := tesseractConfig["max_image_size"].(int); ok {
					ocrConfig["max_image_size"] = int64(maxSize)
				}
				if enablePreproc, ok := tesseractConfig["enable_preprocessing"].(bool); ok {
					ocrConfig["enable_preprocessing"] = enablePreproc
				}
				if tesseractPath, ok := tesseractConfig["tesseract_path"].(string); ok {
					ocrConfig["tesseract_path"] = tesseractPath
				}
			}

			// 启用OCR
			if err := ta.EnableOCR(ocrConfig); err != nil {
				m.Logger.Warn("启用OCR功能失败", "error", err)
				return fmt.Errorf("启用OCR功能失败: %w", err)
			}
		} else {
			m.Logger.Info("OCR功能已禁用")
		}
	}

	// 配置ML功能
	if m.dlpConfig.MLConfig != nil {
		if enabled, ok := m.dlpConfig.MLConfig["enabled"].(bool); ok && enabled {
			m.Logger.Info("启用ML功能")

			// 构建ML配置
			mlConfig := make(map[string]interface{})

			// 从simple_model子配置中提取参数
			if simpleModelConfig, ok := m.dlpConfig.MLConfig["simple_model"].(map[string]interface{}); ok {
				// 敏感关键词
				if keywords, ok := simpleModelConfig["sensitive_keywords"].([]interface{}); ok {
					keywordStrings := make([]string, len(keywords))
					for i, keyword := range keywords {
						if keywordStr, ok := keyword.(string); ok {
							keywordStrings[i] = keywordStr
						}
					}
					mlConfig["sensitive_keywords"] = keywordStrings
				}

				// 置信度阈值
				if threshold, ok := simpleModelConfig["confidence_threshold"].(float64); ok {
					mlConfig["confidence_threshold"] = threshold
				}

				// 风险评分阈值
				if riskThreshold, ok := simpleModelConfig["risk_threshold"].(float64); ok {
					mlConfig["risk_threshold"] = riskThreshold
				}
			}

			// 启用ML
			if err := ta.EnableML(mlConfig); err != nil {
				m.Logger.Warn("启用ML功能失败", "error", err)
				return fmt.Errorf("启用ML功能失败: %w", err)
			}
		} else {
			m.Logger.Info("ML功能已禁用")
		}
	}

	return nil
}

// initializeLegacyComponents 初始化传统组件
func (m *DLPModule) initializeLegacyComponents() error {
	m.Logger.Info("初始化传统组件")

	// 创建规则管理器
	m.ruleManager = NewRuleManager(m.Logger)

	// 创建警报管理器
	m.alertManager = NewAlertManager()

	// 创建扫描器
	m.scanner = NewScanner(m.Logger, m.ruleManager, m.alertManager, m.Config)

	// 加载规则
	if err := m.ruleManager.LoadRules(m.Config); err != nil {
		m.Logger.Error("加载规则失败", "error", err)
		return fmt.Errorf("加载规则失败: %w", err)
	}

	m.Logger.Info("传统组件初始化完成")
	return nil
}

// Start 启动模块
func (m *DLPModule) Start() error {
	m.Logger.Info("启动数据防泄漏模块v2.0")

	// 启动核心组件
	if err := m.startCoreComponents(); err != nil {
		return fmt.Errorf("启动核心组件失败: %w", err)
	}

	// 启动传统组件（保持兼容性）
	if err := m.startLegacyComponents(); err != nil {
		m.Logger.Warn("启动传统组件失败", "error", err)
		// 不返回错误，允许新架构独立运行
	}

	// 启动数据处理流水线
	if err := m.startProcessingPipeline(); err != nil {
		return fmt.Errorf("启动处理流水线失败: %w", err)
	}

	m.mu.Lock()
	m.running = true
	m.mu.Unlock()

	m.Logger.Info("数据防泄漏模块启动完成")
	return nil
}

// startCoreComponents 启动核心组件
func (m *DLPModule) startCoreComponents() error {
	m.Logger.Info("启动DLP核心组件")

	// 检查核心组件是否已初始化
	if m.protocolManager == nil || m.analysisManager == nil ||
		m.policyEngine == nil || m.executionManager == nil {
		m.Logger.Warn("核心组件未初始化，跳过启动")
		return nil
	}

	// 启动协议解析管理器
	if err := m.protocolManager.Start(); err != nil {
		return fmt.Errorf("启动协议解析管理器失败: %w", err)
	}

	// 启动内容分析管理器
	if err := m.analysisManager.Start(); err != nil {
		return fmt.Errorf("启动内容分析管理器失败: %w", err)
	}

	// 启动策略引擎
	if err := m.policyEngine.Start(); err != nil {
		return fmt.Errorf("启动策略引擎失败: %w", err)
	}

	// 启动执行管理器
	if err := m.executionManager.Start(); err != nil {
		return fmt.Errorf("启动执行管理器失败: %w", err)
	}

	// 如果启用网络监控，启动拦截器管理器
	if m.dlpConfig != nil && m.dlpConfig.EnableNetworkMonitoring && m.interceptorManager != nil {
		// 创建并注册流量拦截器
		trafficInterceptor, err := interceptor.NewTrafficInterceptor(m.dlpConfig.InterceptorConfig.Logger)
		if err != nil {
			m.Logger.Warn("创建流量拦截器失败", "error", err)
		} else {
			// 初始化拦截器配置
			if err := trafficInterceptor.Initialize(m.dlpConfig.InterceptorConfig); err != nil {
				m.Logger.Warn("初始化流量拦截器失败", "error", err)
			} else {
				if err := m.interceptorManager.RegisterInterceptor("traffic", trafficInterceptor); err != nil {
					m.Logger.Warn("注册流量拦截器失败，网络监控功能将被禁用", "error", err)
				} else {
					if err := m.interceptorManager.StartAll(); err != nil {
						m.Logger.Warn("启动拦截器失败，网络监控功能将被禁用", "error", err)
						m.Logger.Info("DLP系统将继续运行其他功能：文件监控、剪贴板监控等")
					} else {
						m.Logger.Info("网络流量拦截器启动成功")
					}
				}
			}
		}
	}

	m.Logger.Info("DLP核心组件启动完成")
	return nil
}

// startLegacyComponents 启动传统组件
func (m *DLPModule) startLegacyComponents() error {
	m.Logger.Info("启动传统组件")

	// 确保规则管理器已初始化
	if m.ruleManager == nil {
		m.Logger.Warn("规则管理器未初始化，尝试初始化")
		m.ruleManager = NewRuleManager(m.Logger)

		// 加载规则
		if err := m.ruleManager.LoadRules(m.Config); err != nil {
			m.Logger.Error("加载规则失败", "error", err)
		}
	}

	// 确保警报管理器已初始化
	if m.alertManager == nil {
		m.Logger.Warn("警报管理器未初始化，尝试初始化")
		m.alertManager = NewAlertManager()
	}

	// 确保扫描器已初始化
	if m.scanner == nil {
		m.Logger.Warn("扫描器未初始化，尝试初始化")
		m.scanner = NewScanner(m.Logger, m.ruleManager, m.alertManager, m.Config)
	}

	// 确保监控上下文已初始化
	if m.monitorCtx == nil {
		m.Logger.Warn("监控上下文未初始化，尝试初始化")
		m.monitorCtx, m.monitorCancel = context.WithCancel(context.Background())
	}

	// 启动剪贴板监控
	if m.scanner != nil {
		if err := m.scanner.MonitorClipboard(); err != nil {
			m.Logger.Error("启动剪贴板监控失败", "error", err)
		}

		// 启动文件监控
		if err := m.scanner.MonitorFiles(); err != nil {
			m.Logger.Error("启动文件监控失败", "error", err)
		}
	}

	m.Logger.Info("传统组件启动完成")
	return nil
}

// startProcessingPipeline 启动处理流水线
func (m *DLPModule) startProcessingPipeline() error {
	m.Logger.Info("启动数据处理流水线")

	// 启动处理工作协程
	if m.dlpConfig != nil {
		for i := 0; i < m.dlpConfig.MaxConcurrency; i++ {
			go m.processingWorker(i)
		}

		// 如果启用网络监控，启动数据包监听
		if m.dlpConfig.EnableNetworkMonitoring {
			go m.packetListener()
		}
	}

	m.Logger.Info("数据处理流水线启动完成")
	return nil
}

// processingWorker 处理工作协程
func (m *DLPModule) processingWorker(workerID int) {
	m.Logger.Debug("启动处理工作协程", "worker_id", workerID)
	defer m.Logger.Debug("处理工作协程退出", "worker_id", workerID)

	for {
		select {
		case task := <-m.processingCh:
			if err := m.processTask(task); err != nil {
				m.Logger.Error("处理任务失败", "task_id", task.ID, "error", err)
			}
		case <-m.stopCh:
			return
		}
	}
}

// packetListener 数据包监听器
func (m *DLPModule) packetListener() {
	m.Logger.Debug("启动数据包监听器")
	defer m.Logger.Debug("数据包监听器退出")

	// 检查拦截器管理器是否可用
	if m.interceptorManager == nil {
		m.Logger.Warn("拦截器管理器未初始化，跳过数据包监听")
		return
	}

	// 获取流量拦截器
	trafficInterceptor, exists := m.interceptorManager.GetInterceptor("traffic")
	if !exists {
		m.Logger.Warn("流量拦截器不存在，跳过数据包监听")
		return
	}

	// 获取数据包通道
	packetCh := trafficInterceptor.GetPacketChannel()

	for {
		select {
		case packet := <-packetCh:
			// 创建处理任务
			task := &ProcessingTask{
				ID:        fmt.Sprintf("task_%d", time.Now().UnixNano()),
				Timestamp: time.Now(),
				Packet:    packet,
				Context:   context.Background(),
			}

			// 发送到处理通道
			select {
			case m.processingCh <- task:
			case <-m.stopCh:
				return
			default:
				m.Logger.Warn("处理通道已满，丢弃任务", "task_id", task.ID)
			}
		case <-m.stopCh:
			return
		}
	}
}

// processTask 处理任务
func (m *DLPModule) processTask(task *ProcessingTask) error {
	// 检查核心组件是否可用
	if m.protocolManager == nil || m.analysisManager == nil ||
		m.policyEngine == nil || m.executionManager == nil {
		return fmt.Errorf("核心组件未初始化")
	}

	// 1. 协议解析
	parsedData, err := m.protocolManager.ParsePacket(task.Packet)
	if err != nil {
		if task.Packet.ProcessInfo != nil {
			return fmt.Errorf("协议【%s】解析失败: %w", task.Packet.ProcessInfo.ProcessName, err)
		}
		return fmt.Errorf("协议解析失败: %w", err)
	}

	// 2. 内容分析
	analysisResult, err := m.analysisManager.AnalyzeContent(task.Context, parsedData)
	if err != nil {
		return fmt.Errorf("内容分析失败: %w", err)
	}

	// 3. 策略决策
	decisionContext := &engine.DecisionContext{
		PacketInfo:     task.Packet,
		ParsedData:     parsedData,
		AnalysisResult: analysisResult,
		// 其他上下文信息可以在这里添加
	}

	decision, err := m.policyEngine.EvaluatePolicy(task.Context, decisionContext)
	if err != nil {
		return fmt.Errorf("策略评估失败: %w", err)
	}

	// 4. 动作执行
	_, err = m.executionManager.ExecuteDecision(task.Context, decision)
	if err != nil {
		return fmt.Errorf("动作执行失败: %w", err)
	}

	m.Logger.Debug("任务处理完成",
		"task_id", task.ID,
		"action", decision.Action.String(),
		"risk_level", decision.RiskLevel.String())

	return nil
}

// Stop 停止模块
func (m *DLPModule) Stop() error {
	m.Logger.Info("停止数据防泄漏模块v2.0")

	// 设置停止标志
	m.mu.Lock()
	m.running = false
	m.mu.Unlock()

	// 发送停止信号
	close(m.stopCh)

	// 停止核心组件
	if err := m.stopCoreComponents(); err != nil {
		m.Logger.Error("停止核心组件失败", "error", err)
	}

	// 停止传统组件
	if err := m.stopLegacyComponents(); err != nil {
		m.Logger.Error("停止传统组件失败", "error", err)
	}

	m.Logger.Info("数据防泄漏模块已停止")
	return nil
}

// stopCoreComponents 停止核心组件
func (m *DLPModule) stopCoreComponents() error {
	m.Logger.Info("停止DLP核心组件")

	// 停止拦截器管理器
	if m.interceptorManager != nil {
		if err := m.interceptorManager.StopAll(); err != nil {
			m.Logger.Error("停止拦截器管理器失败", "error", err)
		}
	}

	// 停止执行管理器
	if m.executionManager != nil {
		if err := m.executionManager.Stop(); err != nil {
			m.Logger.Error("停止执行管理器失败", "error", err)
		}
	}

	// 停止策略引擎
	if m.policyEngine != nil {
		if err := m.policyEngine.Stop(); err != nil {
			m.Logger.Error("停止策略引擎失败", "error", err)
		}
	}

	// 停止内容分析管理器
	if m.analysisManager != nil {
		if err := m.analysisManager.Stop(); err != nil {
			m.Logger.Error("停止内容分析管理器失败", "error", err)
		}
	}

	// 停止协议解析管理器
	if m.protocolManager != nil {
		if err := m.protocolManager.Stop(); err != nil {
			m.Logger.Error("停止协议解析管理器失败", "error", err)
		}
	}

	m.Logger.Info("DLP核心组件已停止")
	return nil
}

// stopLegacyComponents 停止传统组件
func (m *DLPModule) stopLegacyComponents() error {
	m.Logger.Info("停止传统组件")

	// 停止监控
	if m.monitorCancel != nil {
		m.monitorCancel()
	} else {
		m.Logger.Warn("监控取消函数未初始化，跳过停止监控")
	}

	// 停止扫描器
	if m.scanner != nil {
		if err := m.scanner.StopMonitoring(); err != nil {
			m.Logger.Error("停止监控失败", "error", err)
		}
	} else {
		m.Logger.Warn("扫描器未初始化，跳过停止监控")
	}

	m.Logger.Info("传统组件已停止")
	return nil
}

// HandleRequest 处理请求
func (m *DLPModule) HandleRequest(ctx context.Context, req *plugin.Request) (*plugin.Response, error) {
	m.Logger.Info("处理请求", "action", req.Action)

	// 确保规则管理器已初始化
	if m.ruleManager == nil {
		m.Logger.Warn("规则管理器未初始化，尝试初始化")
		m.ruleManager = NewRuleManager(m.Logger)

		// 加载规则
		if err := m.ruleManager.LoadRules(m.Config); err != nil {
			m.Logger.Error("加载规则失败", "error", err)
		}
	}

	// 确保警报管理器已初始化
	if m.alertManager == nil {
		m.Logger.Warn("警报管理器未初始化，尝试初始化")
		m.alertManager = NewAlertManager()
	}

	// 确保扫描器已初始化
	if m.scanner == nil {
		m.Logger.Warn("扫描器未初始化，尝试初始化")
		m.scanner = NewScanner(m.Logger, m.ruleManager, m.alertManager, m.Config)
	}

	switch req.Action {
	case "get_rules":
		// 获取规则列表
		rules := m.ruleManager.GetRules()
		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"rules": RulesToMap(rules),
				"count": len(rules),
			},
		}, nil

	case "add_rule":
		// 添加规则
		rule := &DLPRule{
			ID:          sdk.GetConfigString(req.Params, "id", ""),
			Name:        sdk.GetConfigString(req.Params, "name", ""),
			Description: sdk.GetConfigString(req.Params, "description", ""),
			Pattern:     sdk.GetConfigString(req.Params, "pattern", ""),
			Action:      sdk.GetConfigString(req.Params, "action", "alert"),
			Enabled:     sdk.GetConfigBool(req.Params, "enabled", true),
		}

		// 检查必要字段
		if rule.ID == "" || rule.Pattern == "" {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "invalid_param",
					Message: "规则ID和模式不能为空",
				},
			}, nil
		}

		// 添加规则
		if err := m.ruleManager.AddRule(rule); err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "add_error",
					Message: err.Error(),
				},
			}, nil
		}

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"rule": RuleToMap(rule),
			},
		}, nil

	case "update_rule":
		// 更新规则
		rule := &DLPRule{
			ID:          sdk.GetConfigString(req.Params, "id", ""),
			Name:        sdk.GetConfigString(req.Params, "name", ""),
			Description: sdk.GetConfigString(req.Params, "description", ""),
			Pattern:     sdk.GetConfigString(req.Params, "pattern", ""),
			Action:      sdk.GetConfigString(req.Params, "action", "alert"),
			Enabled:     sdk.GetConfigBool(req.Params, "enabled", true),
		}

		// 检查必要字段
		if rule.ID == "" || rule.Pattern == "" {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "invalid_param",
					Message: "规则ID和模式不能为空",
				},
			}, nil
		}

		// 更新规则
		if err := m.ruleManager.UpdateRule(rule); err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "update_error",
					Message: err.Error(),
				},
			}, nil
		}

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"rule": RuleToMap(rule),
			},
		}, nil

	case "delete_rule":
		// 删除规则
		id := sdk.GetConfigString(req.Params, "id", "")
		if id == "" {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "invalid_param",
					Message: "规则ID不能为空",
				},
			}, nil
		}

		// 删除规则
		if err := m.ruleManager.DeleteRule(id); err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "delete_error",
					Message: err.Error(),
				},
			}, nil
		}

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"id": id,
			},
		}, nil

	case "scan_file":
		// 扫描文件
		path := sdk.GetConfigString(req.Params, "path", "")
		if path == "" {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "invalid_param",
					Message: "文件路径不能为空",
				},
			}, nil
		}

		// 扫描文件
		alerts, err := m.scanner.ScanFile(path)
		if err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "scan_error",
					Message: err.Error(),
				},
			}, nil
		}

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"alerts": AlertsToMap(alerts),
				"count":  len(alerts),
			},
		}, nil

	case "scan_directory":
		// 扫描目录
		dir := sdk.GetConfigString(req.Params, "directory", "")
		if dir == "" {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "invalid_param",
					Message: "目录路径不能为空",
				},
			}, nil
		}

		// 扫描目录
		alerts, err := m.scanner.ScanDirectory(dir)
		if err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "scan_error",
					Message: err.Error(),
				},
			}, nil
		}

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"alerts": AlertsToMap(alerts),
				"count":  len(alerts),
			},
		}, nil

	case "scan_clipboard":
		// 扫描剪贴板
		alerts, err := m.scanner.ScanClipboard()
		if err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "scan_error",
					Message: err.Error(),
				},
			}, nil
		}

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"alerts": AlertsToMap(alerts),
				"count":  len(alerts),
			},
		}, nil

	case "get_alerts":
		// 获取警报列表
		alerts := m.alertManager.GetAlerts()
		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"alerts": AlertsToMap(alerts),
				"count":  len(alerts),
			},
		}, nil

	case "clear_alerts":
		// 清除警报
		m.alertManager.ClearAlerts()
		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"status":  "success",
				"message": "警报已清除",
			},
		}, nil

	default:
		return &plugin.Response{
			ID:      req.ID,
			Success: false,
			Error: &plugin.ErrorInfo{
				Code:    "unknown_action",
				Message: fmt.Sprintf("不支持的操作: %s", req.Action),
			},
		}, nil
	}
}

// HandleEvent 处理事件
func (m *DLPModule) HandleEvent(ctx context.Context, event *plugin.Event) error {
	m.Logger.Info("处理事件", "type", event.Type, "source", event.Source)

	// 确保规则管理器已初始化
	if m.ruleManager == nil {
		m.Logger.Warn("规则管理器未初始化，尝试初始化")
		m.ruleManager = NewRuleManager(m.Logger)

		// 加载规则
		if err := m.ruleManager.LoadRules(m.Config); err != nil {
			m.Logger.Error("加载规则失败", "error", err)
		}
	}

	// 确保警报管理器已初始化
	if m.alertManager == nil {
		m.Logger.Warn("警报管理器未初始化，尝试初始化")
		m.alertManager = NewAlertManager()
	}

	// 确保扫描器已初始化
	if m.scanner == nil {
		m.Logger.Warn("扫描器未初始化，尝试初始化")
		m.scanner = NewScanner(m.Logger, m.ruleManager, m.alertManager, m.Config)
	}

	switch event.Type {
	case "system.startup":
		// 系统启动事件
		m.Logger.Info("系统启动")
		return nil

	case "system.shutdown":
		// 系统关闭事件
		m.Logger.Info("系统关闭")
		return nil

	case "dlp.scan_request":
		// 扫描请求
		m.Logger.Info("收到扫描请求")
		if path, ok := event.Data["path"].(string); ok && path != "" {
			_, err := m.scanner.ScanFile(path)
			return err
		}
		return nil

	default:
		// 忽略其他事件
		return nil
	}
}

// registerProtocolParsers 注册所有协议解析器
func (m *DLPModule) registerProtocolParsers() error {
	logger := m.dlpConfig.ParserConfig.Logger

	// HTTP 解析器（只处理明文HTTP）
	httpParser := parser.NewHTTPParser(logger)
	if err := m.protocolManager.RegisterParser(httpParser); err != nil {
		return fmt.Errorf("注册HTTP解析器失败: %w", err)
	}
	logger.Info("注册HTTP解析器成功", "protocols", httpParser.GetSupportedProtocols())

	// HTTPS 解析器（处理TLS/SSL加密的HTTP）
	httpsParser := parser.NewHTTPSParser(logger, m.dlpConfig.ParserConfig.TLSConfig)
	if err := m.protocolManager.RegisterParser(httpsParser); err != nil {
		return fmt.Errorf("注册HTTPS解析器失败: %w", err)
	}
	logger.Info("注册HTTPS解析器成功", "protocols", httpsParser.GetSupportedProtocols())

	// FTP 解析器
	ftpParser := parser.NewFTPParser(logger)
	if err := m.protocolManager.RegisterParser(ftpParser); err != nil {
		return fmt.Errorf("注册FTP解析器失败: %w", err)
	}
	logger.Info("注册FTP解析器成功", "protocols", ftpParser.GetSupportedProtocols())

	// SMTP 解析器
	smtpParser := parser.NewSMTPParser(logger)
	if err := m.protocolManager.RegisterParser(smtpParser); err != nil {
		return fmt.Errorf("注册SMTP解析器失败: %w", err)
	}
	logger.Info("注册SMTP解析器成功", "protocols", smtpParser.GetSupportedProtocols())

	// MySQL 解析器
	mysqlParser := parser.NewMySQLParser(logger)
	if err := m.protocolManager.RegisterParser(mysqlParser); err != nil {
		return fmt.Errorf("注册MySQL解析器失败: %w", err)
	}
	logger.Info("注册MySQL解析器成功", "protocols", mysqlParser.GetSupportedProtocols())

	// 添加默认解析器用于未知协议
	defaultParser := parser.NewDefaultParser(logger)
	if err := m.protocolManager.RegisterParser(defaultParser); err != nil {
		return fmt.Errorf("注册默认解析器失败: %w", err)
	}
	logger.Info("注册默认解析器成功", "protocols", defaultParser.GetSupportedProtocols())

	logger.Info("协议解析器注册完成", "count", 6)
	logger.Info("支持的协议", "protocols", []string{"http", "https", "tls", "ftp", "smtp", "mysql", "unknown", "default"})
	return nil
}
