package engines

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"realtimetts"
)

// AzureEngine Azure TTS引擎实现
type AzureEngine struct {
	*BaseEngine
	client    *http.Client
	authToken string
}

// AzureVoice Azure语音结构体
type AzureVoice struct {
	Name                string   `json:"Name"`
	DisplayName         string   `json:"DisplayName"`
	LocalName           string   `json:"LocalName"`
	ShortName           string   `json:"ShortName"`
	Gender              string   `json:"Gender"`
	Locale              string   `json:"Locale"`
	SampleRateHertz     string   `json:"SampleRateHertz"`
	VoiceType           string   `json:"VoiceType"`
	Status              string   `json:"Status"`
	WordsPerMinute      string   `json:"WordsPerMinute"`
	StyleList           []string `json:"StyleList,omitempty"`
	RolePlayList        []string `json:"RolePlayList,omitempty"`
	SecondaryLocaleList []string `json:"SecondaryLocaleList,omitempty"`
}

// AzureSynthesisRequest Azure合成请求结构体
type AzureSynthesisRequest struct {
	Text string `json:"text"`
}

// NewAzureEngine 创建新的Azure TTS引擎
func NewAzureEngine(apiKey, region string) *AzureEngine {
	base := NewBaseEngine("Azure TTS", "1.0.0", "Microsoft Azure Cognitive Services Text-to-Speech")

	engine := &AzureEngine{
		BaseEngine: base,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		authToken: "",
	}

	// 设置默认配置
	config := realtimetts.DefaultEngineConfig()
	config.APIKey = apiKey
	config.Region = region
	config.Endpoint = fmt.Sprintf("https://%s.tts.speech.microsoft.com", region)
	config.Language = "en-US"
	config.Voice = "en-US-JennyNeural"
	config.Format = "riff-16khz-16bit-mono-pcm"
	config.SampleRate = 16000
	config.Channels = 1

	engine.SetConfig(*config)
	return engine
}

// doInitialize 初始化Azure引擎
func (ae *AzureEngine) doInitialize() error {
	// 获取访问令牌
	if err := ae.getAuthToken(); err != nil {
		return fmt.Errorf("获取Azure访问令牌失败: %w", err)
	}

	// 验证连接
	if err := ae.validateConnection(); err != nil {
		return fmt.Errorf("Azure连接验证失败: %w", err)
	}

	return nil
}

// getAuthToken 获取Azure访问令牌
func (ae *AzureEngine) getAuthToken() error {
	url := fmt.Sprintf("https://%s.api.cognitive.microsoft.com/sts/v1.0/issueToken", ae.config.Region)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Ocp-Apim-Subscription-Key", ae.config.APIKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := ae.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("获取令牌失败，状态码: %d", resp.StatusCode)
	}

	token, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	ae.authToken = string(token)
	return nil
}

// validateConnection 验证Azure连接
func (ae *AzureEngine) validateConnection() error {
	voices, err := ae.GetSupportedVoices()
	if err != nil {
		return err
	}

	if len(voices) == 0 {
		return fmt.Errorf("未找到可用的Azure语音")
	}

	return nil
}

// doSynthesize 执行Azure文本合成
func (ae *AzureEngine) doSynthesize(ctx context.Context, text string, outputChan chan<- []byte) error {
	// 构建SSML
	ssml := ae.buildSSML(text)

	// 构建请求URL
	url := fmt.Sprintf("%s/cognitiveservices/v1", ae.config.Endpoint)

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(ssml))
	if err != nil {
		return err
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+ae.authToken)
	req.Header.Set("Content-Type", "application/ssml+xml")
	req.Header.Set("X-Microsoft-OutputFormat", ae.config.Format)
	req.Header.Set("User-Agent", "RealtimeTTS-Go/1.0")

	// 发送请求
	resp, err := ae.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Azure合成失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 读取音频数据
	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// 分块发送音频数据
	return ae.sendAudioInChunks(audioData, outputChan, ctx)
}

// buildSSML 构建SSML标记
func (ae *AzureEngine) buildSSML(text string) string {
	voice := ae.currentVoice
	if voice.ID == "" {
		voice.ID = ae.config.Voice
	}

	// 转义特殊字符
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	text = strings.ReplaceAll(text, "\"", "&quot;")
	text = strings.ReplaceAll(text, "'", "&apos;")

	// 构建SSML
	ssml := fmt.Sprintf(`<speak version="1.0" xmlns="http://www.w3.org/2001/10/synthesis" xml:lang="%s">
		<voice name="%s">
			<prosody rate="%s" pitch="%s" volume="%s">
				%s
			</prosody>
		</voice>
	</speak>`,
		ae.config.Language,
		voice.ID,
		ae.formatRate(ae.config.Speed),
		ae.formatPitch(ae.config.Pitch),
		ae.formatVolume(ae.config.Volume),
		text,
	)

	return ssml
}

// formatRate 格式化语速
func (ae *AzureEngine) formatRate(speed float64) string {
	if speed == 1.0 {
		return "medium"
	} else if speed < 1.0 {
		return fmt.Sprintf("%.1f%%", speed*100)
	} else {
		return fmt.Sprintf("+%.1f%%", (speed-1.0)*100)
	}
}

// formatPitch 格式化音调
func (ae *AzureEngine) formatPitch(pitch float64) string {
	if pitch == 0.0 {
		return "medium"
	} else if pitch < 0.0 {
		return fmt.Sprintf("%.1f%%", pitch)
	} else {
		return fmt.Sprintf("+%.1f%%", pitch)
	}
}

// formatVolume 格式化音量
func (ae *AzureEngine) formatVolume(volume float64) string {
	if volume == 1.0 {
		return "medium"
	} else if volume < 1.0 {
		return fmt.Sprintf("%.1f%%", volume*100)
	} else {
		return fmt.Sprintf("+%.1f%%", (volume-1.0)*100)
	}
}

// sendAudioInChunks 分块发送音频数据
func (ae *AzureEngine) sendAudioInChunks(audioData []byte, outputChan chan<- []byte, ctx context.Context) error {
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

// GetSupportedVoices 获取Azure支持的语音列表
func (ae *AzureEngine) GetSupportedVoices() ([]realtimetts.Voice, error) {
	url := fmt.Sprintf("%s/cognitiveservices/voices/list", ae.config.Endpoint)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+ae.authToken)

	resp, err := ae.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取语音列表失败，状态码: %d", resp.StatusCode)
	}

	var azureVoices []AzureVoice
	if err := json.NewDecoder(resp.Body).Decode(&azureVoices); err != nil {
		return nil, err
	}

	// 转换为通用语音格式
	voices := make([]realtimetts.Voice, 0, len(azureVoices))
	for _, av := range azureVoices {
		voice := realtimetts.Voice{
			ID:          av.ShortName,
			Name:        av.DisplayName,
			Language:    av.Locale,
			Gender:      av.Gender,
			Description: av.LocalName,
			Config: map[string]string{
				"sample_rate": av.SampleRateHertz,
				"voice_type":  av.VoiceType,
				"status":      av.Status,
			},
		}
		voices = append(voices, voice)
	}

	return voices, nil
}

// SetConfig 重写配置设置方法
func (ae *AzureEngine) SetConfig(config realtimetts.EngineConfig) error {
	if err := realtimetts.ValidateEngineConfig(&config); err != nil {
		return err
	}

	ae.mu.Lock()
	defer ae.mu.Unlock()

	// 如果API密钥或区域发生变化，需要重新获取令牌
	if ae.config.APIKey != config.APIKey || ae.config.Region != config.Region {
		ae.config = config
		ae.config.Endpoint = fmt.Sprintf("https://%s.tts.speech.microsoft.com", config.Region)

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
