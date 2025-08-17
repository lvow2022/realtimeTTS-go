package realtimetts

import (
	"context"
	"time"
)

// AudioFormat 音频格式
type AudioFormat int

const (
	FormatPCM AudioFormat = iota + 1
	FormatWAV
	FormatMP3
)

// AudioConfig 音频配置
type AudioConfig struct {
	Format            AudioFormat // 音频格式
	Channels          int         // 声道数 (1=单声道, 2=立体声)
	SampleRate        int         // 采样率
	OutputDeviceIndex *int        // 输出设备索引
	Muted             bool        // 是否静音
	FramesPerBuffer   int         // 每缓冲区帧数
	PlayoutChunkSize  int         // 播放块大小 (-1表示自动)
}

// StreamInfo 流信息
type StreamInfo struct {
	Format     AudioFormat // 音频格式
	Channels   int         // 声道数
	SampleRate int         // 采样率
}

// TimingInfo 时间信息
type TimingInfo struct {
	Word      string        `json:"word"`
	StartTime time.Duration `json:"start_time"`
	EndTime   time.Duration `json:"end_time"`
	Duration  time.Duration `json:"duration"`
}

// Callbacks 回调函数集合
type Callbacks struct {
	OnCharacter           func(rune)                    // 字符回调
	OnWord                func(string)                  // 单词回调
	OnAudioChunk          func([]byte)                  // 音频块回调
	OnPlaybackStart       func()                        // 播放开始回调
	OnPlaybackStop        func()                        // 播放停止回调
	OnTextStreamStart     func()                        // 文本流开始回调
	OnTextStreamStop      func()                        // 文本流停止回调
	OnSentenceSynthesized func(string)                  // 句子合成回调
}

// Engine TTS引擎接口
type Engine interface {
	// Synthesize 将文本合成为音频
	Synthesize(ctx context.Context, text string) error

	// GetStreamInfo 获取流信息
	GetStreamInfo() StreamInfo

	// CanConsumeGenerators 是否可以消费生成器
	CanConsumeGenerators() bool

	// Stop 停止引擎
	Stop()

	// GetName 获取引擎名称
	GetName() string

	// SetCallbacks 设置回调函数
	SetCallbacks(callbacks *Callbacks)

	// GetAudioQueue 获取音频队列
	GetAudioQueue() <-chan []byte

	// GetTimingsQueue 获取时间信息队列
	GetTimingsQueue() <-chan TimingInfo
}

// EngineStatus 引擎状态
type EngineStatus int

const (
	EngineStatusIdle EngineStatus = iota
	EngineStatusSynthesizing
	EngineStatusError
	EngineStatusStopped
)

// EngineError 引擎错误
type EngineError struct {
	EngineName string
	Message    string
	Err        error
}

func (e *EngineError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *EngineError) Unwrap() error {
	return e.Err
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	StartTime           time.Time     // 开始时间
	FirstAudioLatency   time.Duration // 首块音频延迟
	TotalAudioChunks    int64         // 总音频块数
	TotalWordsSpoken    int64         // 总单词数
	AverageLatency      time.Duration // 平均延迟
	ProcessingTime      time.Duration // 处理时间
	AudioBufferSize     int           // 音频缓冲区大小
	EngineSwitchCount   int           // 引擎切换次数
}
