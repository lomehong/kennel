package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/comm"
	"github.com/lomehong/kennel/pkg/logging"
)

var (
	serverAddr  = flag.String("addr", "localhost:8080", "服务器地址")
	serverPath  = flag.String("path", "/ws", "WebSocket路径")
	logLevel    = flag.String("log-level", "info", "日志级别")
	interval    = flag.Int("interval", 5, "监控间隔（秒）")
	jsonOutput  = flag.Bool("json", false, "输出JSON格式")
	watchMetric = flag.String("watch", "", "监控特定指标")
	duration    = flag.Int("duration", 0, "监控持续时间（秒），0表示一直运行")
)

func main() {
	flag.Parse()

	// 创建日志器
	logConfig := logging.DefaultLogConfig()

	// 设置日志级别
	level := hclog.LevelFromString(*logLevel)
	if level == hclog.NoLevel {
		level = hclog.Info
	}

	// 将 hclog 级别转换为 logging 级别
	switch level {
	case hclog.Trace:
		logConfig.Level = logging.LogLevelTrace
	case hclog.Debug:
		logConfig.Level = logging.LogLevelDebug
	case hclog.Info:
		logConfig.Level = logging.LogLevelInfo
	case hclog.Warn:
		logConfig.Level = logging.LogLevelWarn
	case hclog.Error:
		logConfig.Level = logging.LogLevelError
	default:
		logConfig.Level = logging.LogLevelInfo
	}

	// 创建日志记录器
	enhancedLogger, _ := logging.NewEnhancedLogger(logConfig)
	log := enhancedLogger.Named("comm-monitor")

	// 创建配置
	config := comm.DefaultConfig()
	config.ServerURL = fmt.Sprintf("ws://%s%s", *serverAddr, *serverPath)
	config.HeartbeatInterval = 30 * time.Second
	config.ReconnectInterval = 3 * time.Second

	// 创建管理器
	manager := comm.NewManager(config, log)

	// 设置客户端信息
	manager.SetClientInfo(map[string]interface{}{
		"client_id":    fmt.Sprintf("monitor-%d", time.Now().UnixNano()),
		"version":      "1.0.0",
		"monitor_mode": true,
	})

	// 连接到服务器
	fmt.Printf("连接到服务器 %s...\n", config.ServerURL)
	err := manager.Connect()
	if err != nil {
		log.Error("连接服务器失败", "error", err)
		os.Exit(1)
	}
	defer manager.Disconnect()

	fmt.Println("已连接到服务器")

	// 处理中断信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 创建定时器
	ticker := time.NewTicker(time.Duration(*interval) * time.Second)
	defer ticker.Stop()

	// 设置结束时间
	var endTime time.Time
	if *duration > 0 {
		endTime = time.Now().Add(time.Duration(*duration) * time.Second)
	}

	// 监控循环
	for {
		select {
		case <-ticker.C:
			// 检查是否到达结束时间
			if *duration > 0 && time.Now().After(endTime) {
				fmt.Println("监控时间到，退出...")
				return
			}

			// 获取指标
			metrics := manager.GetMetrics()

			// 输出指标
			if *jsonOutput {
				// JSON格式输出
				if *watchMetric != "" {
					// 只输出特定指标
					if value, ok := metrics[*watchMetric]; ok {
						data, _ := json.MarshalIndent(value, "", "  ")
						fmt.Printf("%s: %s\n", *watchMetric, string(data))
					} else {
						fmt.Printf("指标 %s 不存在\n", *watchMetric)
					}
				} else {
					// 输出所有指标
					data, _ := json.MarshalIndent(metrics, "", "  ")
					fmt.Println(string(data))
				}
			} else {
				// 文本格式输出
				if *watchMetric != "" {
					// 只输出特定指标
					if value, ok := metrics[*watchMetric]; ok {
						fmt.Printf("%s: %v\n", *watchMetric, value)
					} else {
						fmt.Printf("指标 %s 不存在\n", *watchMetric)
					}
				} else {
					// 输出所有指标
					fmt.Println(manager.GetMetricsReport())
				}
			}

			// 输出分隔线
			if !*jsonOutput {
				fmt.Println("----------------------------------------")
			}

		case <-sigCh:
			fmt.Println("接收到中断信号，退出...")
			return
		}
	}
}
