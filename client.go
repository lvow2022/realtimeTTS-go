package realtimetts

import (
	"context"
	"fmt"
)

// Client TTS客户端，提供简化的API
type Client struct {
	stream *TextToAudioStream
	config *AudioConfig
}

// NewClient 创建TTS客户端
func NewClient(config *AudioConfig) *Client {
	// 默认使用ESpeak引擎
	engine := NewESpeakEngine()
	stream := NewTextToAudioStream(engine, config)

	return &Client{
		stream: stream,
		config: config,
	}
}

// NewClientWithEngine 使用指定引擎创建TTS客户端
func NewClientWithEngine(engine Engine, config *AudioConfig) *Client {
	stream := NewTextToAudioStream(engine, config)

	return &Client{
		stream: stream,
		config: config,
	}
}

// Speak 同步播放文本
func (c *Client) Speak(ctx context.Context, text string) error {
	// 设置默认回调
	callbacks := &Callbacks{
		OnPlaybackStart: func() {
			fmt.Println("▶️ 开始播放")
		},
		OnPlaybackStop: func() {
			fmt.Println("⏹️ 停止播放")
		},
	}
	c.stream.SetCallbacks(callbacks)

	// 输入文本
	c.stream.Feed(text)

	// 播放
	return c.stream.Play(ctx)
}

// SpeakAsync 异步播放文本
func (c *Client) SpeakAsync(ctx context.Context, text string) {
	// 设置默认回调
	callbacks := &Callbacks{
		OnPlaybackStart: func() {
			fmt.Println("▶️ 开始播放")
		},
		OnPlaybackStop: func() {
			fmt.Println("⏹️ 停止播放")
		},
	}
	c.stream.SetCallbacks(callbacks)

	// 输入文本
	c.stream.Feed(text)

	// 异步播放
	c.stream.PlayAsync(ctx)
}

// SpeakWithCallbacks 使用自定义回调播放文本
func (c *Client) SpeakWithCallbacks(ctx context.Context, text string, callbacks *Callbacks) error {
	c.stream.SetCallbacks(callbacks)
	c.stream.Feed(text)
	return c.stream.Play(ctx)
}

// SpeakWithCallbacksAsync 使用自定义回调异步播放文本
func (c *Client) SpeakWithCallbacksAsync(ctx context.Context, text string, callbacks *Callbacks) {
	c.stream.SetCallbacks(callbacks)
	c.stream.Feed(text)
	c.stream.PlayAsync(ctx)
}

// Stop 停止播放
func (c *Client) Stop() {
	c.stream.Stop()
}

// Pause 暂停播放
func (c *Client) Pause() {
	c.stream.Pause()
}

// Resume 恢复播放
func (c *Client) Resume() {
	c.stream.Resume()
}

// IsPlaying 检查是否正在播放
func (c *Client) IsPlaying() bool {
	return c.stream.IsPlaying()
}

// GetStream 获取底层流对象（高级用法）
func (c *Client) GetStream() *TextToAudioStream {
	return c.stream
}

// SetCallbacks 设置回调函数
func (c *Client) SetCallbacks(callbacks *Callbacks) {
	c.stream.SetCallbacks(callbacks)
}

// AddEngine 添加引擎
func (c *Client) AddEngine(engine Engine) {
	c.stream.AddEngine(engine)
}

// RemoveEngine 移除引擎
func (c *Client) RemoveEngine(engineName string) {
	c.stream.RemoveEngine(engineName)
}

// GetEngines 获取引擎列表
func (c *Client) GetEngines() []Engine {
	return c.stream.GetEngines()
}

// DefaultConfig 获取默认配置
func DefaultConfig() *AudioConfig {
	return &AudioConfig{
		Format:           FormatPCM,
		Channels:         1,
		SampleRate:       22050,
		Muted:            false,
		FramesPerBuffer:  512,
		PlayoutChunkSize: -1,
	}
}

// HighQualityConfig 获取高质量配置
func HighQualityConfig() *AudioConfig {
	return &AudioConfig{
		Format:           FormatPCM,
		Channels:         2,
		SampleRate:       44100,
		Muted:            false,
		FramesPerBuffer:  1024,
		PlayoutChunkSize: -1,
	}
}

// LowLatencyConfig 获取低延迟配置
func LowLatencyConfig() *AudioConfig {
	return &AudioConfig{
		Format:           FormatPCM,
		Channels:         1,
		SampleRate:       16000,
		Muted:            false,
		FramesPerBuffer:  256,
		PlayoutChunkSize: -1,
	}
}
