package config

import (
	"testing"
)

func TestAssetsValidator(t *testing.T) {
	validator := CreateAssetsValidator()

	// 测试有效配置
	validConfig := map[string]interface{}{
		"plugins": map[string]interface{}{
			"assets": map[string]interface{}{
				"enabled":          true,
				"collect_interval": 3600,
				"report_server":    "https://example.com/api/assets",
				"auto_report":      false,
				"log_level":        "info",
				"cache": map[string]interface{}{
					"enabled": true,
					"dir":     "data/assets/cache",
				},
			},
		},
	}

	if err := validator.Validate(validConfig); err != nil {
		t.Errorf("有效配置验证失败: %v", err)
	}

	// 测试无效的collect_interval
	invalidConfig := map[string]interface{}{
		"plugins": map[string]interface{}{
			"assets": map[string]interface{}{
				"enabled":          true,
				"collect_interval": 30, // 小于最小值60
			},
		},
	}

	if err := validator.Validate(invalidConfig); err == nil {
		t.Error("应该检测到无效的collect_interval")
	}

	// 测试无效的log_level
	invalidLogLevel := map[string]interface{}{
		"plugins": map[string]interface{}{
			"assets": map[string]interface{}{
				"enabled":          true,
				"collect_interval": 3600,
				"log_level":        "invalid", // 无效的日志级别
			},
		},
	}

	if err := validator.Validate(invalidLogLevel); err == nil {
		t.Error("应该检测到无效的log_level")
	}
}

func TestAuditValidator(t *testing.T) {
	validator := CreateAuditValidator()

	// 测试有效配置
	validConfig := map[string]interface{}{
		"plugins": map[string]interface{}{
			"audit": map[string]interface{}{
				"enabled":             true,
				"log_system_events":   true,
				"log_user_events":     true,
				"log_network_events":  true,
				"log_file_events":     true,
				"log_retention_days":  30,
				"log_level":           "info",
				"enable_alerts":       false,
				"alert_recipients":    []interface{}{"admin@example.com"},
				"storage": map[string]interface{}{
					"type": "file",
				},
			},
		},
	}

	if err := validator.Validate(validConfig); err != nil {
		t.Errorf("有效配置验证失败: %v", err)
	}

	// 测试无效的log_retention_days
	invalidConfig := map[string]interface{}{
		"plugins": map[string]interface{}{
			"audit": map[string]interface{}{
				"enabled":            true,
				"log_retention_days": 400, // 超过最大值365
			},
		},
	}

	if err := validator.Validate(invalidConfig); err == nil {
		t.Error("应该检测到无效的log_retention_days")
	}
}

func TestDeviceValidator(t *testing.T) {
	validator := CreateDeviceValidator()

	// 测试有效配置
	validConfig := map[string]interface{}{
		"plugins": map[string]interface{}{
			"device": map[string]interface{}{
				"enabled":                   true,
				"monitor_usb":               true,
				"monitor_network":           true,
				"allow_network_disable":     true,
				"device_cache_expiration":   30,
				"monitor_interval":          60,
				"log_level":                 "info",
				"protected_interfaces":      []interface{}{"lo", "eth0", "en0"},
			},
		},
	}

	if err := validator.Validate(validConfig); err != nil {
		t.Errorf("有效配置验证失败: %v", err)
	}

	// 测试无效的monitor_interval
	invalidConfig := map[string]interface{}{
		"plugins": map[string]interface{}{
			"device": map[string]interface{}{
				"enabled":          true,
				"monitor_interval": 5, // 小于最小值10
			},
		},
	}

	if err := validator.Validate(invalidConfig); err == nil {
		t.Error("应该检测到无效的monitor_interval")
	}
}

func TestControlValidator(t *testing.T) {
	validator := CreateControlValidator()

	// 测试有效配置
	validConfig := map[string]interface{}{
		"plugins": map[string]interface{}{
			"control": map[string]interface{}{
				"enabled":      true,
				"log_level":    "info",
				"auto_start":   true,
				"auto_restart": true,
				"isolation": map[string]interface{}{
					"level": "basic",
				},
				"settings": map[string]interface{}{
					"process": map[string]interface{}{
						"max_processes": 1000,
					},
				},
			},
		},
	}

	if err := validator.Validate(validConfig); err != nil {
		t.Errorf("有效配置验证失败: %v", err)
	}
}

func TestDLPValidator(t *testing.T) {
	validator := CreateDLPValidator()

	// 测试有效配置
	validConfig := map[string]interface{}{
		"plugins": map[string]interface{}{
			"dlp": map[string]interface{}{
				"enabled":           true,
				"name":              "dlp",
				"version":           "2.0.0",
				"monitor_network":   true,
				"monitor_files":     true,
				"monitor_clipboard": true,
				"max_concurrency":   4,
				"buffer_size":       500,
				"network_protocols": []interface{}{"http", "https", "ftp", "smtp"},
				"interceptor_config": map[string]interface{}{
					"filter": "outbound",
				},
			},
		},
	}

	if err := validator.Validate(validConfig); err != nil {
		t.Errorf("有效配置验证失败: %v", err)
	}

	// 测试无效的max_concurrency
	invalidConfig := map[string]interface{}{
		"plugins": map[string]interface{}{
			"dlp": map[string]interface{}{
				"enabled":         true,
				"max_concurrency": 20, // 超过最大值16
			},
		},
	}

	if err := validator.Validate(invalidConfig); err == nil {
		t.Error("应该检测到无效的max_concurrency")
	}

	// 测试无效的buffer_size
	invalidBufferSize := map[string]interface{}{
		"plugins": map[string]interface{}{
			"dlp": map[string]interface{}{
				"enabled":     true,
				"buffer_size": 50, // 小于最小值100
			},
		},
	}

	if err := validator.Validate(invalidBufferSize); err == nil {
		t.Error("应该检测到无效的buffer_size")
	}
}

func TestGetAllPluginValidators(t *testing.T) {
	validators := GetAllPluginValidators()

	expectedPlugins := []string{"assets", "audit", "device", "control", "dlp"}
	for _, plugin := range expectedPlugins {
		if _, exists := validators[plugin]; !exists {
			t.Errorf("缺少插件 %s 的验证器", plugin)
		}
	}

	if len(validators) != len(expectedPlugins) {
		t.Errorf("验证器数量不匹配，期望 %d，实际 %d", len(expectedPlugins), len(validators))
	}
}

func TestValidatorCoverage(t *testing.T) {
	// 测试验证覆盖率
	validators := GetAllPluginValidators()

	// 检查每个验证器的字段数量
	expectedMinFields := map[string]int{
		"assets":  5, // enabled, collect_interval, report_server, auto_report, log_level
		"audit":   8, // enabled, log_*_events, log_retention_days, log_level, enable_alerts, alert_recipients, storage
		"device":  7, // enabled, monitor_*, device_cache_expiration, monitor_interval, log_level, protected_interfaces
		"control": 5, // enabled, log_level, auto_start, auto_restart, isolation, settings
		"dlp":     15, // 大量配置字段
	}

	for pluginID, validator := range validators {
		fieldCount := len(validator.FieldTypes) + len(validator.RequiredFields)
		minExpected := expectedMinFields[pluginID]
		
		if fieldCount < minExpected {
			t.Errorf("插件 %s 的验证字段数量不足，期望至少 %d，实际 %d", 
				pluginID, minExpected, fieldCount)
		}
	}
}
