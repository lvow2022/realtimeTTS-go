package engines

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"realtimetts"
)

// OpenAIEngine OpenAI TTS引擎实现
type OpenAIEngine struct {
	*BaseEngine
	client *http.Client
}

// OpenAIVoice OpenAI语音结构体
type OpenAIVoice struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Language string `json:"language"`
	Gender   string `json:"gender"`
}

// OpenAISynthesisRequest OpenAI合成请求结构体
type OpenAISynthesisRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format"`
	Speed          float64 `json:"speed"`
}

// NewOpenAIEngine 创建新的OpenAI TTS引擎
func NewOpenAIEngine(apiKey string) *OpenAIEngine {
	base := NewBaseEngine("OpenAI TTS", "1.0.0", "OpenAI Text-to-Speech API")

	engine := &OpenAIEngine{
		BaseEngine: base,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// 设置默认配置
	config := realtimetts.DefaultEngineConfig()
	config.APIKey = apiKey
	config.Endpoint = "https://api.openai.com/v1/audio/speech"
	config.Language = "en"
	config.Voice = "alloy"
	config.Format = "mp3"
	config.SampleRate = 24000
	config.Channels = 1

	engine.SetConfig(*config)
	return engine
}

// doInitialize 初始化OpenAI引擎
func (ae *OpenAIEngine) doInitialize() error {
	// 验证API密钥
	if err := ae.validateAPIKey(); err != nil {
		return fmt.Errorf("OpenAI API密钥验证失败: %w", err)
	}

	return nil
}

// validateAPIKey 验证OpenAI API密钥
func (ae *OpenAIEngine) validateAPIKey() error {
	url := "https://api.openai.com/v1/models"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+ae.config.APIKey)

	resp, err := ae.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API密钥验证失败，状态码: %d", resp.StatusCode)
	}

	return nil
}

// doSynthesize 执行OpenAI文本合成
func (ae *OpenAIEngine) doSynthesize(ctx context.Context, text string, outputChan chan<- []byte) error {
	// 构建请求
	request := OpenAISynthesisRequest{
		Model:          "tts-1",
		Input:          text,
		Voice:          ae.config.Voice,
		ResponseFormat: ae.config.Format,
		Speed:          ae.config.Speed,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", ae.config.Endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+ae.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := ae.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OpenAI合成失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 读取音频数据
	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// 分块发送音频数据
	return ae.sendAudioInChunks(audioData, outputChan, ctx)
}

// sendAudioInChunks 分块发送音频数据
func (ae *OpenAIEngine) sendAudioInChunks(audioData []byte, outputChan chan<- []byte, ctx context.Context) error {
	chunkSize := 4096 // 4KB chunks
	totalSize := len(audioData)

	for i := 0; i < totalSize; i += chunkSize {
		end := i + chunkSize
		if end > totalSize {
			end = totalSize
		}

		chunk := audioData[i:end]
		// 移除未使用的变量

		if !ae.sendAudioChunk(chunk, outputChan, ctx) {
			return fmt.Errorf("发送音频块失败")
		}

		// 检查上下文取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return nil
}

// GetSupportedVoices 获取OpenAI支持的语音列表
func (ae *OpenAIEngine) GetSupportedVoices() ([]realtimetts.Voice, error) {
	// OpenAI TTS-1模型支持的语音列表
	voices := []realtimetts.Voice{
		{
			ID:          "alloy",
			Name:        "Alloy",
			Language:    "en",
			Gender:      "neutral",
			Description: "Alloy voice - neutral tone",
			Config: map[string]string{
				"model": "tts-1",
			},
		},
		{
			ID:          "echo",
			Name:        "Echo",
			Language:    "en",
			Gender:      "male",
			Description: "Echo voice - male tone",
			Config: map[string]string{
				"model": "tts-1",
			},
		},
		{
			ID:          "fable",
			Name:        "Fable",
			Language:    "en",
			Gender:      "male",
			Description: "Fable voice - male tone",
			Config: map[string]string{
				"model": "tts-1",
			},
		},
		{
			ID:          "onyx",
			Name:        "Onyx",
			Language:    "en",
			Gender:      "male",
			Description: "Onyx voice - male tone",
			Config: map[string]string{
				"model": "tts-1",
			},
		},
		{
			ID:          "nova",
			Name:        "Nova",
			Language:    "en",
			Gender:      "female",
			Description: "Nova voice - female tone",
			Config: map[string]string{
				"model": "tts-1",
			},
		},
		{
			ID:          "shimmer",
			Name:        "Shimmer",
			Language:    "en",
			Gender:      "female",
			Description: "Shimmer voice - female tone",
			Config: map[string]string{
				"model": "tts-1",
			},
		},
	}

	return voices, nil
}

// SetConfig 重写配置设置方法
func (ae *OpenAIEngine) SetConfig(config realtimetts.EngineConfig) error {
	if err := realtimetts.ValidateEngineConfig(&config); err != nil {
		return err
	}

	ae.mu.Lock()
	defer ae.mu.Unlock()

	// 如果API密钥发生变化，需要重新验证
	if ae.config.APIKey != config.APIKey {
		ae.config = config

		// 重新初始化
		if ae.status == realtimetts.EngineStatusReady {
			ae.status = realtimetts.EngineStatusUninitialized
			return ae.Initialize()
		}
	} else {
		ae.config = config
	}

	return nil
}

// GetEngineInfo 重写获取引擎信息方法
func (ae *OpenAIEngine) GetEngineInfo() realtimetts.EngineInfo {
	info := ae.BaseEngine.GetEngineInfo()
	info.Capabilities = append(info.Capabilities, "mp3-format", "real-time-synthesis")
	return info
}
