package realtimetts

import (
	"fmt"
	"sync"
	"time"
)

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

// BaseEngine 基础引擎抽象类
// 提供通用的引擎功能和默认实现
type BaseEngine struct {
	mu sync.RWMutex

	// 基本属性
	engineName           string
	canConsumeGenerators bool

	audioBuffer *AudioBuffer

	// 回调函数
	onAudioChunk    func([]byte)
	onPlaybackStart func()

	// 控制
	stopSynthesisChan chan struct{}

	// 音频时长
	audioDuration time.Duration
}

// NewBaseEngine 创建新的基础引擎
func NewBaseEngine(name string) *BaseEngine {
	return &BaseEngine{
		engineName:           name,
		canConsumeGenerators: false,
		onAudioChunk:         nil,
		onPlaybackStart:      nil,
		stopSynthesisChan:    make(chan struct{}),
		audioDuration:        0,
	}
}

// ValidateEngineConfig 验证引擎配置
func ValidateEngineConfig(config *EngineConfig) error {
	if config.Speed < 0.5 || config.Speed > 2.0 {
		return fmt.Errorf("无效的播放速度")
	}
	if config.Pitch < -20 || config.Pitch > 20 {
		return fmt.Errorf("无效的音调")
	}
	if config.Volume < 0.0 || config.Volume > 1.0 {
		return fmt.Errorf("无效的音量")
	}
	if config.SampleRate <= 0 {
		return fmt.Errorf("无效的采样率")
	}
	if config.Channels <= 0 || config.Channels > 8 {
		return fmt.Errorf("无效的声道数")
	}
	return nil
}

// 工具方法

// GetDefaultStreamInfo 获取默认音频配置信息
func (be *BaseEngine) GetDefaultStreamInfo() *AudioConfiguration {
	return &AudioConfiguration{
		SampleRate:    16000,
		Channels:      1,
		BitsPerSample: 16,
		Volume:        1.0,
	}
}

// GetDefaultVoices 获取默认语音列表
func (be *BaseEngine) GetDefaultVoices() []Voice {
	return []Voice{}
}

// 基础方法

// GetEngineName 获取引擎名称
func (be *BaseEngine) GetEngineName() string {
	return be.engineName
}

// SetCanConsumeGenerators 设置是否支持生成器
func (be *BaseEngine) SetCanConsumeGenerators(can bool) {
	be.canConsumeGenerators = can
}

// CanConsumeGenerators 检查是否支持生成器
func (be *BaseEngine) CanConsumeGenerators() bool {
	return be.canConsumeGenerators
}

// SetOnAudioChunk 设置音频块回调
func (be *BaseEngine) SetOnAudioChunk(callback func([]byte)) {
	be.onAudioChunk = callback
}

// SetOnPlaybackStart 设置播放开始回调
func (be *BaseEngine) SetOnPlaybackStart(callback func()) {
	be.onPlaybackStart = callback
}

// SetAudioBuffer 设置音频缓冲管理器
func (be *BaseEngine) SetAudioBuffer(audioBuffer *AudioBuffer) {
	be.mu.Lock()
	defer be.mu.Unlock()
	be.audioBuffer = audioBuffer
}

// StopSynthesis 停止合成
func (be *BaseEngine) StopSynthesis() {
	close(be.stopSynthesisChan)
}

// ResetAudioDuration 重置音频时长
func (be *BaseEngine) ResetAudioDuration() {
	be.audioDuration = 0
}

// GetAudioDuration 获取音频时长
func (be *BaseEngine) GetAudioDuration() time.Duration {
	return be.audioDuration
}
