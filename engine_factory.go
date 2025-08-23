package realtimetts

import (
	"fmt"
	"sync"
)

// EngineType 引擎类型枚举
type EngineType string

const (
	EngineTypeAzure  EngineType = "azure"
	EngineTypeOpenAI EngineType = "openai"
	EngineTypeESpeak EngineType = "espeak"
)

// EngineFactory 引擎工厂
type EngineFactory struct {
	mu      sync.RWMutex
	engines map[string]TTSEngine
	configs map[string]*EngineConfig
}

// NewEngineFactory 创建新的引擎工厂
func NewEngineFactory() *EngineFactory {
	return &EngineFactory{
		engines: make(map[string]TTSEngine),
		configs: make(map[string]*EngineConfig),
	}
}

// CreateEngine 创建指定类型的引擎
func (ef *EngineFactory) CreateEngine(engineType EngineType, config *EngineConfig) (TTSEngine, error) {
	ef.mu.Lock()
	defer ef.mu.Unlock()

	// 检查是否已存在相同配置的引擎
	configKey := ef.generateConfigKey(engineType, config)
	if existingEngine, exists := ef.engines[configKey]; exists {
		return existingEngine, nil
	}

	var engine TTSEngine
	var err error

	switch engineType {
	case EngineTypeAzure:
		engine, err = ef.createAzureEngine(config)
	case EngineTypeOpenAI:
		engine, err = ef.createOpenAIEngine(config)
	case EngineTypeESpeak:
		engine, err = ef.createESpeakEngine(config)
	default:
		return nil, fmt.Errorf("不支持的引擎类型: %s", engineType)
	}

	if err != nil {
		return nil, fmt.Errorf("创建引擎失败: %w", err)
	}

	// 初始化引擎
	if err := engine.Initialize(); err != nil {
		return nil, fmt.Errorf("初始化引擎失败: %w", err)
	}

	// 缓存引擎和配置
	ef.engines[configKey] = engine
	ef.configs[configKey] = config

	return engine, nil
}

// createAzureEngine 创建Azure引擎
func (ef *EngineFactory) createAzureEngine(config *EngineConfig) (TTSEngine, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Azure引擎需要API密钥")
	}
	if config.Region == "" {
		return nil, fmt.Errorf("Azure引擎需要区域配置")
	}

	// 使用反射或接口创建引擎，避免循环导入
	return nil, fmt.Errorf("Azure引擎创建功能暂未实现")
}

// createOpenAIEngine 创建OpenAI引擎
func (ef *EngineFactory) createOpenAIEngine(config *EngineConfig) (TTSEngine, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI引擎需要API密钥")
	}

	// 使用反射或接口创建引擎，避免循环导入
	return nil, fmt.Errorf("OpenAI引擎创建功能暂未实现")
}

// createESpeakEngine 创建ESpeak引擎
func (ef *EngineFactory) createESpeakEngine(config *EngineConfig) (TTSEngine, error) {
	// ESpeak引擎实现将在后续添加
	return nil, fmt.Errorf("ESpeak引擎暂未实现")
}

// GetEngine 获取已创建的引擎
func (ef *EngineFactory) GetEngine(engineType EngineType, config *EngineConfig) (TTSEngine, bool) {
	ef.mu.RLock()
	defer ef.mu.RUnlock()

	configKey := ef.generateConfigKey(engineType, config)
	engine, exists := ef.engines[configKey]
	return engine, exists
}

// RemoveEngine 移除引擎
func (ef *EngineFactory) RemoveEngine(engineType EngineType, config *EngineConfig) error {
	ef.mu.Lock()
	defer ef.mu.Unlock()

	configKey := ef.generateConfigKey(engineType, config)

	if engine, exists := ef.engines[configKey]; exists {
		// 关闭引擎
		if err := engine.Close(); err != nil {
			return fmt.Errorf("关闭引擎失败: %w", err)
		}

		// 从缓存中移除
		delete(ef.engines, configKey)
		delete(ef.configs, configKey)
	}

	return nil
}

// ListEngines 列出所有已创建的引擎
func (ef *EngineFactory) ListEngines() map[string]EngineInfo {
	ef.mu.RLock()
	defer ef.mu.RUnlock()

	result := make(map[string]EngineInfo)
	for configKey, engine := range ef.engines {
		result[configKey] = engine.GetEngineInfo()
	}
	return result
}

// CloseAllEngines 关闭所有引擎
func (ef *EngineFactory) CloseAllEngines() error {
	ef.mu.Lock()
	defer ef.mu.Unlock()

	var lastError error
	for configKey, engine := range ef.engines {
		if err := engine.Close(); err != nil {
			lastError = fmt.Errorf("关闭引擎 %s 失败: %w", configKey, err)
		}
	}

	// 清空缓存
	ef.engines = make(map[string]TTSEngine)
	ef.configs = make(map[string]*EngineConfig)

	return lastError
}

// generateConfigKey 生成配置键
func (ef *EngineFactory) generateConfigKey(engineType EngineType, config *EngineConfig) string {
	// 使用引擎类型和关键配置生成唯一键
	key := fmt.Sprintf("%s_%s_%s", engineType, config.APIKey, config.Region)
	if config.Voice != "" {
		key += "_" + config.Voice
	}
	return key
}

// CreateEngineWithDefaults 使用默认配置创建引擎
func (ef *EngineFactory) CreateEngineWithDefaults(engineType EngineType, apiKey string) (TTSEngine, error) {
	config := DefaultEngineConfig()
	config.APIKey = apiKey

	switch engineType {
	case EngineTypeAzure:
		config.Region = "eastus" // 默认区域
		config.Voice = "en-US-JennyNeural"
	case EngineTypeOpenAI:
		config.Voice = "alloy"
	case EngineTypeESpeak:
		// ESpeak不需要API密钥
		config.APIKey = ""
	}

	return ef.CreateEngine(engineType, config)
}

// CreateMultiEngine 创建多引擎实例
func (ef *EngineFactory) CreateMultiEngine(engineConfigs []EngineConfig) ([]TTSEngine, error) {
	var engineList []TTSEngine

	for _, config := range engineConfigs {
		// 根据配置推断引擎类型
		engineType := ef.inferEngineType(&config)

		engine, err := ef.CreateEngine(engineType, &config)
		if err != nil {
			// 关闭已创建的引擎
			for _, createdEngine := range engineList {
				createdEngine.Close()
			}
			return nil, fmt.Errorf("创建引擎失败: %w", err)
		}

		engineList = append(engineList, engine)
	}

	return engineList, nil
}

// inferEngineType 根据配置推断引擎类型
func (ef *EngineFactory) inferEngineType(config *EngineConfig) EngineType {
	// 根据配置特征推断引擎类型
	if config.Region != "" {
		return EngineTypeAzure
	}
	if config.APIKey != "" && config.Endpoint == "https://api.openai.com/v1/audio/speech" {
		return EngineTypeOpenAI
	}

	// 默认返回OpenAI
	return EngineTypeOpenAI
}

// ValidateEngineConfig 验证引擎配置
func (ef *EngineFactory) ValidateEngineConfig(engineType EngineType, config *EngineConfig) error {
	switch engineType {
	case EngineTypeAzure:
		if config.APIKey == "" {
			return fmt.Errorf("Azure引擎需要API密钥")
		}
		if config.Region == "" {
			return fmt.Errorf("Azure引擎需要区域配置")
		}
	case EngineTypeOpenAI:
		if config.APIKey == "" {
			return fmt.Errorf("OpenAI引擎需要API密钥")
		}
	case EngineTypeESpeak:
		// ESpeak不需要特殊验证
	}

	return ValidateEngineConfig(config)
}
