package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/lomehong/kennel/app/control/pkg/control"
	"github.com/lomehong/kennel/pkg/core/plugin"
	"github.com/lomehong/kennel/pkg/logging"
	sdk "github.com/lomehong/kennel/pkg/sdk/go"
	"gopkg.in/yaml.v3"
)

// 配置结构体
type Config struct {
	ID       string                 `yaml:"id"`
	Name     string                 `yaml:"name"`
	Version  string                 `yaml:"version"`
	Enabled  bool                   `yaml:"enabled"`
	LogLevel string                 `yaml:"log_level"`
	Settings map[string]interface{} `yaml:"settings"`
}

var logger logging.Logger

func init() {
	// 初始化日志器
	config := logging.DefaultLogConfig()
	config.Level = logging.LogLevelInfo

	var err error
	logger, err = logging.NewEnhancedLogger(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}

	logger = logger.Named("main")
	logger.Info("日志初始化成功")
}

func main() {
	// 设置环境变量，确保插件使用正确的 Magic Cookie
	os.Setenv("APPFRAMEWORK_PLUGIN", "appframework")
	logger.Info("设置环境变量成功")

	// 输出当前工作目录
	pwd, err := os.Getwd()
	if err != nil {
		logger.Error("获取当前工作目录失败", "error", err)
	} else {
		logger.Info("当前工作目录", "path", pwd)
	}

	// 创建终端管控模块
	logger.Info("创建终端管控模块...")
	module := control.NewControlModule()
	logger.Info("终端管控模块创建成功!")

	// 加载配置文件
	logger.Info("正在加载配置文件...")
	config, err := loadConfig("config.yaml")
	if err != nil {
		logger.Error("加载配置文件失败", "error", err)
		os.Exit(1)
	}
	logger.Info("配置文件加载成功!")

	// 初始化模块
	logger.Info("正在初始化模块...")
	moduleConfig := &plugin.ModuleConfig{
		Settings: config.Settings,
	}
	logger.Info("模块配置", "config", moduleConfig)
	if err := module.Init(context.Background(), moduleConfig); err != nil {
		logger.Error("初始化模块失败", "error", err)
		os.Exit(1)
	}
	logger.Info("模块初始化成功!")

	// 运行模块
	sdk.RunModule(module)
}

// 加载配置文件
func loadConfig(configPath string) (*Config, error) {
	// 获取当前执行文件所在目录
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("获取执行文件路径失败: %w", err)
	}
	execDir := filepath.Dir(execPath)

	// 尝试从不同位置加载配置文件
	configPaths := []string{
		configPath,                               // 当前目录
		filepath.Join(execDir, configPath),       // 执行文件所在目录
		filepath.Join("app/control", configPath), // app/control 目录
	}

	logger.Info("尝试从以下位置加载配置文件")
	for _, path := range configPaths {
		logger.Info("检查配置文件", "path", path)
	}

	var configData []byte
	var loadErr error
	for _, path := range configPaths {
		logger.Info("尝试加载", "path", path)
		configData, loadErr = ioutil.ReadFile(path)
		if loadErr == nil {
			logger.Info("成功加载配置文件", "path", path)
			break
		}
		logger.Error("加载失败", "path", path, "error", loadErr)
	}

	if loadErr != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", loadErr)
	}

	// 解析配置文件
	var config Config
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 输出配置内容
	logger.Debug("配置内容", "config", config)
	logger.Debug("Settings", "settings", config.Settings)

	return &config, nil
}
