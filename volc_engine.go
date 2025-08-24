package realtimetts

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"realtimetts"
)

// VolcengineEngine 火山云TTS引擎实现
type VolcengineEngine struct {
	*BaseEngine
	client *http.Client
	config VolcengineConfig

	// 统计信息
	totalBytesSent  int64 // 总发送字节数
	totalChunksSent int64 // 总发送块数
	chunkSequence   int64 // 音频块序列号
}

// VolcengineConfig 火山云配置
type VolcengineConfig struct {
	AppID         string  `json:"app_id"`
	AccessToken   string  `json:"access_token"`
	Cluster       string  `json:"cluster"`
	Endpoint      string  `json:"endpoint"`
	VoiceType     string  `json:"voice_type"`
	Language      string  `json:"language"`
	Rate          int     `json:"rate"`
	Encoding      string  `json:"encoding"`
	SpeedRatio    float32 `json:"speed_ratio"`
	VolumeRatio   float32 `json:"volume_ratio"`
	PitchRatio    float32 `json:"pitch_ratio"`
	Channels      int     `json:"channels"`
	BitDepth      int     `json:"bit_depth"`
	FrameDuration string  `json:"frame_duration"`
	TextType      string  `json:"text_type"`
	Ssml          bool    `json:"ssml"`
}

// VolcengineResponse 火山云响应结构
type VolcengineResponse struct {
	ReqID     string       `json:"reqid"`
	Code      int          `json:"code"`
	Message   string       `json:"message"`
	Operation string       `json:"operation"`
	Sequence  int          `json:"sequence"`
	Data      string       `json:"data"`
	Addition  VolcAddition `json:"addition"`
}

// VolcAddition 火山云附加信息
type VolcAddition struct {
	Frontend string `json:"frontend"`
}

// VolcTimestamp 火山云时间戳信息
type VolcTimestamp struct {
	Begin int `json:"begin"`
	End   int `json:"end"`
}

// NewVolcengineEngine 创建新的火山云TTS引擎
func NewVolcengineEngine(appID, accessToken, cluster string) *VolcengineEngine {
	base := NewBaseEngine("Volcengine TTS", "1.0.0", "火山云语音合成服务")

	engine := &VolcengineEngine{
		BaseEngine: base,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: VolcengineConfig{
			AppID:         appID,
			AccessToken:   accessToken,
			Cluster:       cluster,
			Endpoint:      "https://openspeech.bytedance.com/api/v1/tts",
			VoiceType:     "BV700_streaming",
			Language:      "cn",
			Rate:          16000, // 改为16000
			Encoding:      "pcm", // 使用pcm格式，支持流式
			SpeedRatio:    1.0,
			VolumeRatio:   1.0,
			PitchRatio:    1.0,
			Channels:      1,
			BitDepth:      16,
			FrameDuration: "20ms",
			TextType:      "plain",
			Ssml:          false,
		},
	}

	// 设置默认配置
	config := realtimetts.DefaultEngineConfig()
	config.APIKey = accessToken
	config.Endpoint = "https://openspeech.bytedance.com/api/v1/tts"
	config.Language = "zh-CN"
	config.Voice = "BV700_streaming"
	config.Format = "pcm"
	config.SampleRate = 16000 // 改为16000，与火山云配置保持一致
	config.Channels = 1

	engine.SetConfig(*config)

	// 设置合成器接口
	engine.synthesizer = engine

	return engine
}

// doInitialize 初始化火山云引擎
func (ve *VolcengineEngine) doInitialize() error {
	// 验证配置
	if ve.config.AppID == "" {
		return fmt.Errorf("火山云引擎需要AppID")
	}
	if ve.config.AccessToken == "" {
		return fmt.Errorf("火山云引擎需要AccessToken")
	}
	if ve.config.Cluster == "" {
		return fmt.Errorf("火山云引擎需要Cluster")
	}

	return nil
}

// DoSynthesize 执行火山云文本合成
func (ve *VolcengineEngine) DoSynthesize(ctx context.Context, text string, outputChan chan<- []byte) error {
	fmt.Printf("   开始火山云合成: %s\n", text)

	// 构建请求参数
	params := ve.buildRequestParams(text)
	fmt.Printf("   请求参数构建完成\n")

	// 发送请求
	fmt.Printf("   发送HTTP请求到: %s\n", ve.config.Endpoint)
	resp, err := ve.sendRequest(ctx, params)
	if err != nil {
		fmt.Printf("   HTTP请求失败: %v\n", err)
		return fmt.Errorf("火山云请求失败: %w", err)
	}

	fmt.Printf("   收到响应: Code=%d, Message=%s\n", resp.Code, resp.Message)

	// 处理响应
	if resp.Code != 3000 {
		fmt.Printf("   火山云合成失败: %s\n", resp.Message)
		return fmt.Errorf("火山云合成失败: %s", resp.Message)
	}

	// 解码音频数据
	fmt.Printf("   开始解码音频数据, 长度: %d\n", len(resp.Data))
	audioData, err := base64.StdEncoding.DecodeString(resp.Data)
	if err != nil {
		fmt.Printf("   音频数据解码失败: %v\n", err)
		return fmt.Errorf("音频数据解码失败: %w", err)
	}

	fmt.Printf("   音频数据解码成功, 解码后长度: %d 字节\n", len(audioData))

	// 检查音频数据的前几个字节
	if len(audioData) >= 8 {
		fmt.Printf("   音频数据前8字节: %v\n", audioData[:8])
	}

	// 保存音频数据到本地文件
	filename := fmt.Sprintf("volcengine_audio_%d.pcm", time.Now().Unix())
	if err := os.WriteFile(filename, audioData, 0644); err != nil {
		fmt.Printf("   保存音频文件失败: %v\n", err)
	} else {
		fmt.Printf("   音频已保存到文件: %s\n", filename)
	}

	// 检查是否需要字节序转换（PCM数据通常是小端序）
	// 这里可以根据需要添加字节序转换逻辑

	// 分块发送音频数据
	return ve.sendAudioInChunks(audioData, outputChan, ctx)
}

// buildRequestParams 构建请求参数
func (ve *VolcengineEngine) buildRequestParams(text string) map[string]interface{} {
	params := make(map[string]interface{})

	// app参数
	params["app"] = map[string]interface{}{
		"appid":   ve.config.AppID,
		"token":   ve.config.AccessToken,
		"cluster": ve.config.Cluster,
	}

	// user参数
	params["user"] = map[string]interface{}{
		"uid": "uid",
	}

	// audio参数
	params["audio"] = map[string]interface{}{
		"voice_type":       ve.config.VoiceType,
		"encoding":         "pcm", // 使用pcm格式，支持流式
		"compression_rate": 1,
		"rate":             ve.config.Rate,
		"speed_ratio":      ve.config.SpeedRatio,
		"volume_ratio":     ve.config.VolumeRatio,
		"pitch_ratio":      ve.config.PitchRatio,
		"emotion":          "happy",
		"language":         "cn",
	}

	// request参数
	textType := "plain"
	if strings.HasPrefix(text, "<speak>") {
		textType = "ssml"
	}

	params["request"] = map[string]interface{}{
		"reqid":            fmt.Sprintf("req_%d_%d", time.Now().UnixNano(), time.Now().Unix()),
		"text":             text,
		"text_type":        textType,
		"operation":        "query",
		"silence_duration": "125",
		"with_frontend":    "1",
		"frontend_type":    "unitTson",
		"pure_english_opt": "1",
	}

	return params
}

// sendRequest 发送HTTP请求
func (ve *VolcengineEngine) sendRequest(ctx context.Context, params map[string]interface{}) (*VolcengineResponse, error) {
	// 序列化请求参数
	requestBody, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("序列化请求参数失败: %w", err)
	}

	// 格式化打印请求体
	var formatted map[string]interface{}
	if json.Unmarshal(requestBody, &formatted) == nil {
		prettyBytes, _ := json.MarshalIndent(formatted, "", "  ")
		fmt.Printf("   请求体:\n%s\n", string(prettyBytes))
	} else {
		fmt.Printf("   请求体: %s\n", string(requestBody))
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", ve.config.Endpoint, strings.NewReader(string(requestBody)))
	if err != nil {
		return nil, err
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer;%s", ve.config.AccessToken))

	// 发送请求
	resp, err := ve.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 解析响应
	var volcResp VolcengineResponse
	if err := json.NewDecoder(resp.Body).Decode(&volcResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &volcResp, nil
}

// sendAudioInChunks 分块发送音频数据
func (ve *VolcengineEngine) sendAudioInChunks(audioData []byte, outputChan chan<- []byte, ctx context.Context) error {
	// 调整为更小的块大小，更适合音频播放
	// 对于16kHz、16位、单声道，每个样本2字节
	// 1024个样本 = 2048字节，约0.064秒的音频
	chunkSize := 2048 // 2KB chunks
	totalSize := len(audioData)

	for i := 0; i < totalSize; i += chunkSize {
		end := i + chunkSize
		if end > totalSize {
			end = totalSize
		}

		chunk := audioData[i:end]
		// 移除未使用的变量 timestamp

		// 计算持续时间
		bytesPerSecond := ve.config.Rate * ve.config.Channels * (ve.config.BitDepth / 8)
		duration := time.Duration(len(chunk)) * time.Second / time.Duration(bytesPerSecond)

		// 直接发送音频数据
		ve.chunkSequence++

		// 流式发送：等待通道有空间再发送
		for {
			select {
			case outputChan <- chunk:
				// 发送成功，更新统计信息
				ve.totalBytesSent += int64(len(chunk))
				ve.totalChunksSent++
				fmt.Printf("   发送音频块: %d 字节, 持续时间: %v (总发送: %d 字节, %d 块)\n",
					len(chunk), duration, ve.totalBytesSent, ve.totalChunksSent)
				goto nextChunk
			case <-ctx.Done():
				return ctx.Err()
			case <-ve.stopChan:
				return fmt.Errorf("引擎已停止")
			default:
				// 通道已满，等待一段时间再尝试
				fmt.Printf("   ⏳ 输出通道已满，等待空间...\n")
				select {
				case <-time.After(50 * time.Millisecond):
					// 等待50ms后重试
					continue
				case <-ctx.Done():
					return ctx.Err()
				case <-ve.stopChan:
					return fmt.Errorf("引擎已停止")
				}
			}
		}

	nextChunk:
		// 检查上下文取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return nil
}

// GetSupportedVoices 获取火山云支持的语音列表
func (ve *VolcengineEngine) GetSupportedVoices() ([]realtimetts.Voice, error) {
	// 火山云支持的语音列表（部分示例）
	voices := []realtimetts.Voice{
		{
			ID:          "BV700_streaming",
			Name:        "BV700 流式语音",
			Language:    "zh-CN",
			Gender:      "female",
			Description: "火山云BV700流式语音合成",
			Config: map[string]string{
				"rate":      "16000",
				"encoding":  "pcm",
				"bit_depth": "16",
				"channels":  "1",
			},
		},
		{
			ID:          "BV700_V2_streaming",
			Name:        "BV700 V2 流式语音",
			Language:    "zh-CN",
			Gender:      "male",
			Description: "火山云BV700 V2流式语音合成",
			Config: map[string]string{
				"rate":      "16000",
				"encoding":  "pcm",
				"bit_depth": "16",
				"channels":  "1",
			},
		},
	}

	return voices, nil
}

// SetConfig 重写配置设置方法
func (ve *VolcengineEngine) SetConfig(config realtimetts.EngineConfig) error {
	if err := realtimetts.ValidateEngineConfig(&config); err != nil {
		return err
	}

	ve.mu.Lock()
	defer ve.mu.Unlock()

	// 更新火山云特定配置
	if config.APIKey != "" {
		ve.config.AccessToken = config.APIKey
	}
	if config.Endpoint != "" {
		ve.config.Endpoint = config.Endpoint
	}
	if config.Language != "" {
		ve.config.Language = config.Language
	}
	if config.Voice != "" {
		ve.config.VoiceType = config.Voice
	}
	if config.SampleRate > 0 {
		ve.config.Rate = config.SampleRate
	}
	if config.Channels > 0 {
		ve.config.Channels = config.Channels
	}

	ve.config = ve.config

	return nil
}

// GetEngineInfo 重写获取引擎信息方法
func (ve *VolcengineEngine) GetEngineInfo() realtimetts.EngineInfo {
	info := ve.BaseEngine.GetEngineInfo()
	info.Capabilities = append(info.Capabilities, "pcm-format", "real-time-synthesis", "timestamp-support")
	return info
}

// SetVolcengineConfig 设置火山云特定配置
func (ve *VolcengineEngine) SetVolcengineConfig(config VolcengineConfig) error {
	ve.mu.Lock()
	defer ve.mu.Unlock()

	// 确保采样率不被覆盖
	if config.Rate > 0 {
		ve.config.Rate = config.Rate
	}
	ve.config = config
	return nil
}

// GetVolcengineConfig 获取火山云配置
func (ve *VolcengineEngine) GetVolcengineConfig() VolcengineConfig {
	ve.mu.RLock()
	defer ve.mu.RUnlock()
	return ve.config
}

// GetVolcengineStats 获取火山云引擎统计信息
func (ve *VolcengineEngine) GetVolcengineStats() (int64, int64) {
	ve.mu.RLock()
	defer ve.mu.RUnlock()
	return ve.totalBytesSent, ve.totalChunksSent
}
