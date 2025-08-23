package realtimetts

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// MockEngine 模拟TTS引擎用于测试
type MockEngine struct {
	isReady bool
	mu      sync.RWMutex
}

// NewMockEngine 创建模拟引擎
func NewMockEngine() *MockEngine {
	return &MockEngine{
		isReady: true,
	}
}

// Synthesize 实现TTSEngine接口
func (me *MockEngine) Synthesize(ctx context.Context, text string) (<-chan []byte, error) {
	outputChan := make(chan []byte, 10)

	go func() {
		defer close(outputChan)

		// 模拟音频数据
		audioData := []byte{1, 2, 3, 4, 5, 6, 7, 8}

		// 发送到输出通道
		select {
		case outputChan <- audioData:
		case <-ctx.Done():
		}
	}()

	return outputChan, nil
}

// GetEngineInfo 实现TTSEngine接口
func (me *MockEngine) GetEngineInfo() EngineInfo {
	return EngineInfo{
		Name:         "Mock Engine",
		Version:      "1.0.0",
		Description:  "Mock TTS Engine for Testing",
		Capabilities: []string{"text-to-speech", "voice-selection"},
		Config:       make(map[string]string),
	}
}

// GetSupportedVoices 返回模拟语音列表
func (me *MockEngine) GetSupportedVoices() ([]Voice, error) {
	return []Voice{
		{
			ID:          "mock-voice-1",
			Name:        "Mock Voice 1",
			Language:    "en",
			Gender:      "male",
			Description: "Mock voice for testing",
		},
		{
			ID:          "mock-voice-2",
			Name:        "Mock Voice 2",
			Language:    "en",
			Gender:      "female",
			Description: "Mock voice for testing",
		},
	}, nil
}

// SetVoice 实现TTSEngine接口
func (me *MockEngine) SetVoice(voice Voice) error {
	return nil
}

// SetConfig 实现TTSEngine接口
func (me *MockEngine) SetConfig(config EngineConfig) error {
	return nil
}

// IsReady 实现TTSEngine接口
func (me *MockEngine) IsReady() bool {
	me.mu.RLock()
	defer me.mu.RUnlock()
	return me.isReady
}

// Initialize 实现TTSEngine接口
func (me *MockEngine) Initialize() error {
	me.mu.Lock()
	defer me.mu.Unlock()
	me.isReady = true
	return nil
}

// Close 实现TTSEngine接口
func (me *MockEngine) Close() error {
	me.mu.Lock()
	defer me.mu.Unlock()
	me.isReady = false
	return nil
}

// TestNewTextToAudioStream 测试创建TextToAudioStream
func TestNewTextToAudioStream(t *testing.T) {
	// 创建模拟引擎
	mockEngine := NewMockEngine()

	// 创建流
	stream := NewTextToAudioStream([]TTSEngine{mockEngine}, nil)

	if stream == nil {
		t.Fatal("NewTextToAudioStream返回nil")
	}

	// 检查初始状态
	status := stream.GetStatus()
	if status["is_playing"] != false {
		t.Error("初始状态应该是未播放")
	}

	if status["engine_count"] != 1 {
		t.Error("引擎数量应该是1")
	}

	// 清理
	stream.Close()
}

// TestTextToAudioStreamFeed 测试文本输入
func TestTextToAudioStreamFeed(t *testing.T) {
	mockEngine := NewMockEngine()
	stream := NewTextToAudioStream([]TTSEngine{mockEngine}, nil)
	defer stream.Close()

	// 测试正常输入
	err := stream.Feed("Hello, this is a test.")
	if err != nil {
		t.Errorf("Feed失败: %v", err)
	}

	// 测试空文本
	err = stream.Feed("")
	if err != nil {
		t.Errorf("空文本输入失败: %v", err)
	}
}

// TestTextToAudioStreamCallbacks 测试回调系统
func TestTextToAudioStreamCallbacks(t *testing.T) {
	mockEngine := NewMockEngine()
	stream := NewTextToAudioStream([]TTSEngine{mockEngine}, nil)
	defer stream.Close()

	// 创建回调
	callbacks := NewCallbacks()

	callbacks.OnTextStreamStart = func() {
		// 回调被调用
	}

	stream.SetCallbacks(callbacks)

	// 输入文本
	stream.Feed("Test text")

	// 注意：由于是异步处理，这里只是测试回调设置是否成功
	if stream.callbacks == nil {
		t.Error("回调未设置")
	}
}

// TestTextToAudioStreamPlaybackControl 测试播放控制
func TestTextToAudioStreamPlaybackControl(t *testing.T) {
	mockEngine := NewMockEngine()
	stream := NewTextToAudioStream([]TTSEngine{mockEngine}, nil)
	defer stream.Close()

	// 测试播放
	err := stream.Play()
	if err != nil {
		t.Errorf("Play失败: %v", err)
	}

	// 等待一小段时间
	time.Sleep(100 * time.Millisecond)

	// 测试停止
	err = stream.Stop()
	if err != nil {
		t.Errorf("Stop失败: %v", err)
	}
}

// TestTextToAudioStreamEngineSwitching 测试引擎切换
func TestTextToAudioStreamEngineSwitching(t *testing.T) {
	// 创建两个模拟引擎
	mockEngine1 := NewMockEngine()
	mockEngine2 := NewMockEngine()

	stream := NewTextToAudioStream([]TTSEngine{mockEngine1, mockEngine2}, nil)
	defer stream.Close()

	// 检查初始引擎
	status := stream.GetStatus()
	if status["current_engine"] != "Mock Engine" {
		t.Error("初始引擎不正确")
	}

	// 测试引擎切换（通过模拟引擎失败）
	// 这里只是测试切换逻辑，实际切换需要引擎失败
	if stream.switchToNextEngine() {
		status = stream.GetStatus()
		if status["current_engine"] != "Mock Engine" {
			t.Error("引擎切换后状态不正确")
		}
	}
}

// TestTextProcessor 测试文本处理器
func TestTextProcessor(t *testing.T) {
	processor := &TextProcessor{
		callbacks: NewCallbacks(),
		config:    DefaultStreamConfig(),
	}

	// 测试句子分割
	text := "Hello. This is a test. How are you?"
	sentences := processor.splitIntoSentences(text)

	expectedCount := 3
	if len(sentences) != expectedCount {
		t.Errorf("期望%d个句子，实际得到%d个", expectedCount, len(sentences))
	}

	// 检查句子内容
	expectedSentences := []string{"Hello", "This is a test", "How are you?"}
	for i, sentence := range sentences {
		if sentence != expectedSentences[i] {
			t.Errorf("句子%d不匹配，期望'%s'，实际'%s'", i, expectedSentences[i], sentence)
		}
	}
}

// TestStreamConfig 测试流配置
func TestStreamConfig(t *testing.T) {
	config := DefaultStreamConfig()

	if config.AudioConfig == nil {
		t.Error("音频配置不应该为nil")
	}

	if config.BufferThresholdSeconds != 2.0 {
		t.Error("缓冲阈值不正确")
	}

	if config.MinimumSentenceLength != 10 {
		t.Error("最小句子长度不正确")
	}

	if !config.FastSentenceFragment {
		t.Error("快速句子片段应该为true")
	}
}

// TestTextToAudioStreamConcurrency 测试并发安全性
func TestTextToAudioStreamConcurrency(t *testing.T) {
	mockEngine := NewMockEngine()
	stream := NewTextToAudioStream([]TTSEngine{mockEngine}, nil)
	defer stream.Close()

	// 并发输入文本
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			stream.FeedAsync(fmt.Sprintf("Concurrent text %d", id))
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestTextToAudioStreamErrorHandling 测试错误处理
func TestTextToAudioStreamErrorHandling(t *testing.T) {
	// 测试空引擎列表
	defer func() {
		if r := recover(); r == nil {
			t.Error("应该panic当没有引擎时")
		}
	}()

	NewTextToAudioStream([]TTSEngine{}, nil)
}

// BenchmarkTextToAudioStream 性能基准测试
func BenchmarkTextToAudioStream(b *testing.B) {
	mockEngine := NewMockEngine()
	stream := NewTextToAudioStream([]TTSEngine{mockEngine}, nil)
	defer stream.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream.Feed("Benchmark test text")
	}
}
