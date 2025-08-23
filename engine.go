package realtimetts

import (
	"context"
	"time"
)

// TTSEngine TTS引擎接口
// 定义所有TTS引擎必须实现的方法
type TTSEngine interface {
	// Synthesize 将文本合成为音频流
	// 返回音频数据通道和错误
	Synthesize(ctx context.Context, text string) (<-chan []byte, error)

	// GetEngineInfo 获取引擎信息
	GetEngineInfo() EngineInfo

	// GetSupportedVoices 获取支持的语音列表
	GetSupportedVoices() ([]Voice, error)

	// SetVoice 设置语音
	SetVoice(voice Voice) error

	// SetConfig 设置引擎配置
	SetConfig(config EngineConfig) error

	// IsReady 检查引擎是否就绪
	IsReady() bool

	// Initialize 初始化引擎
	Initialize() error

	// Close 关闭引擎
	Close() error
}

// EngineInfo 引擎信息结构体
type EngineInfo struct {
	Name         string            // 引擎名称
	Version      string            // 引擎版本
	Description  string            // 引擎描述
	Capabilities []string          // 支持的功能
	Config       map[string]string // 配置信息
}

// Voice 语音结构体
type Voice struct {
	ID          string            // 语音ID
	Name        string            // 语音名称
	Language    string            // 语言代码
	Gender      string            // 性别 (male/female)
	Age         string            // 年龄段
	Description string            // 描述
	Config      map[string]string // 语音特定配置
}

// EngineConfig 引擎配置结构体
type EngineConfig struct {
	APIKey        string            // API密钥
	Endpoint      string            // 服务端点
	Region        string            // 服务区域
	Language      string            // 默认语言
	Voice         string            // 默认语音
	Speed         float64           // 语速 (0.5-2.0)
	Pitch         float64           // 音调 (-20-20)
	Volume        float64           // 音量 (0.0-1.0)
	Format        string            // 音频格式
	SampleRate    int               // 采样率
	Channels      int               // 声道数
	BitsPerSample int               // 位深度
	ExtraConfig   map[string]string // 额外配置
}

// SynthesisOptions 合成选项结构体
type SynthesisOptions struct {
	Voice       Voice             // 使用的语音
	Speed       float64           // 语速
	Pitch       float64           // 音调
	Volume      float64           // 音量
	Format      string            // 音频格式
	SampleRate  int               // 采样率
	Channels    int               // 声道数
	ExtraConfig map[string]string // 额外配置
}

// SynthesisResult 合成结果结构体
type SynthesisResult struct {
	AudioData  <-chan []byte      // 音频数据通道
	Error      error              // 错误信息
	Duration   time.Duration      // 合成时长
	WordCount  int                // 单词数量
	CharCount  int                // 字符数量
}

// EngineStatus 引擎状态枚举
type EngineStatus int

const (
	EngineStatusUninitialized EngineStatus = iota
	EngineStatusInitializing
	EngineStatusReady
	EngineStatusSynthesizing
	EngineStatusError
	EngineStatusClosed
)

// String 返回引擎状态的字符串表示
func (s EngineStatus) String() string {
	switch s {
	case EngineStatusUninitialized:
		return "Uninitialized"
	case EngineStatusInitializing:
		return "Initializing"
	case EngineStatusReady:
		return "Ready"
	case EngineStatusSynthesizing:
		return "Synthesizing"
	case EngineStatusError:
		return "Error"
	case EngineStatusClosed:
		return "Closed"
	default:
		return "Unknown"
	}
}

// EngineError 引擎错误结构体
type EngineError struct {
	EngineName string    // 引擎名称
	Message    string    // 错误消息
	Code       string    // 错误代码
	Timestamp  time.Time // 错误时间
	Retryable  bool      // 是否可重试
}

// Error 实现error接口
func (e *EngineError) Error() string {
	return e.Message
}

// IsRetryable 检查错误是否可重试
func (e *EngineError) IsRetryable() bool {
	return e.Retryable
}

// DefaultEngineConfig 返回默认引擎配置
func DefaultEngineConfig() *EngineConfig {
	return &EngineConfig{
		APIKey:        "",
		Endpoint:      "",
		Region:        "",
		Language:      "en-US",
		Voice:         "",
		Speed:         1.0,
		Pitch:         0.0,
		Volume:        1.0,
		Format:        "wav",
		SampleRate:    16000,
		Channels:      1,
		BitsPerSample: 16,
		ExtraConfig:   make(map[string]string),
	}
}

// ValidateEngineConfig 验证引擎配置
func ValidateEngineConfig(config *EngineConfig) error {
	if config.Speed < 0.5 || config.Speed > 2.0 {
		return ErrInvalidPlaybackSpeed
	}
	if config.Pitch < -20 || config.Pitch > 20 {
		return ErrInvalidPitch
	}
	if config.Volume < 0.0 || config.Volume > 1.0 {
		return ErrInvalidVolume
	}
	if config.SampleRate <= 0 {
		return ErrInvalidSampleRate
	}
	if config.Channels <= 0 || config.Channels > 8 {
		return ErrInvalidChannels
	}
	return nil
}
