package ai

import (
	"context"
	"os"
	"testing"

	"github.com/lomehong/kennel/pkg/sdk/go"
	"github.com/stretchr/testify/assert"
)

func TestAIManager_Init(t *testing.T) {
	// 跳过测试，如果没有设置API密钥
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("跳过测试：未设置OPENAI_API_KEY环境变量")
	}

	// 创建日志记录器
	logger := sdk.NewLogger("test", sdk.LogLevelInfo)

	// 创建配置
	config := map[string]interface{}{
		"ai": map[string]interface{}{
			"enabled":    true,
			"model_type": "openai",
			"model_name": "gpt-3.5-turbo",
			"api_key":    apiKey,
		},
	}

	// 创建AI管理器
	manager := NewAIManager(logger, config)

	// 初始化AI管理器
	err := manager.Init(context.Background())
	assert.NoError(t, err)
	assert.True(t, manager.initialized)
}

func TestAIManager_ProcessNaturalLanguageRequest(t *testing.T) {
	// 跳过测试，如果没有设置API密钥
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("跳过测试：未设置OPENAI_API_KEY环境变量")
	}

	// 创建日志记录器
	logger := sdk.NewLogger("test", sdk.LogLevelInfo)

	// 创建配置
	config := map[string]interface{}{
		"ai": map[string]interface{}{
			"enabled":    true,
			"model_type": "openai",
			"model_name": "gpt-3.5-turbo",
			"api_key":    apiKey,
		},
	}

	// 创建AI管理器
	manager := NewAIManager(logger, config)

	// 初始化AI管理器
	err := manager.Init(context.Background())
	assert.NoError(t, err)

	// 处理自然语言请求
	response, err := manager.ProcessNaturalLanguageRequest(context.Background(), "列出当前运行的进程")
	assert.NoError(t, err)
	assert.NotEmpty(t, response)
}
