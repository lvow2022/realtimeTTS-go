package realtimetts

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// TextToAudioStream 主控制器
// 集成引擎管理和播放控制，提供完整的TTS流程
type TextToAudioStream struct {
	mu sync.RWMutex

	// 引擎管理
	engines       []TTSEngine
	currentEngine int
	engineFactory *EngineFactory

	// 播放控制
	player       *StreamPlayer
	playLock     sync.Mutex
	ttsAudioChan chan [][]byte

	// 文本处理
	textBuffer    chan string
	charBuffer    chan rune
	textProcessor *TextProcessor

	// 回调系统
	callbacks *Callbacks

	// 状态管理
	isPlaying bool
	isPaused  bool
	ctx       context.Context
	cancel    context.CancelFunc

	// 配置
	config *StreamConfig
}

// StreamConfig 流配置
type StreamConfig struct {
	AudioConfig             *AudioConfiguration
	BufferThresholdSeconds  float64
	MinimumSentenceLength   int
	FastSentenceFragment    bool
	CommaSilenceDuration    time.Duration
	SentenceSilenceDuration time.Duration
	OutputWavFile           string
	LogCharacters           bool
	OutputDeviceIndex       int
	Tokenizer               string
	Language                string
	Muted                   bool
}

// TextProcessor 文本处理器
type TextProcessor struct {
	mu        sync.RWMutex
	callbacks *Callbacks
	config    *StreamConfig
}

// NewTextToAudioStream 创建新的文本转音频流
func NewTextToAudioStream(engines []TTSEngine, config *StreamConfig) *TextToAudioStream {
	if len(engines) == 0 {
		panic("至少需要一个TTS引擎")
	}

	if config == nil {
		config = DefaultStreamConfig()
	}

	// 创建音频通道
	ttsAudioChan := make(chan [][]byte, 1000)

	// 创建播放器
	player := NewStreamPlayer(ttsAudioChan, config.AudioConfig, 1000)

	// 创建回调系统
	callbacks := NewCallbacks()

	// 创建文本处理器
	textProcessor := &TextProcessor{
		callbacks: callbacks,
		config:    config,
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	stream := &TextToAudioStream{
		engines:       engines,
		currentEngine: 0,
		engineFactory: NewEngineFactory(),
		player:        player,
		playLock:      sync.Mutex{},
		ttsAudioChan:  ttsAudioChan,
		textBuffer:    make(chan string, 100),
		charBuffer:    make(chan rune, 1000),
		textProcessor: textProcessor,
		callbacks:     callbacks,
		isPlaying:     false,
		isPaused:      false,
		ctx:           ctx,
		cancel:        cancel,
		config:        config,
	}

	// 设置播放器回调
	player.SetCallbacks(
		stream.onAudioChunk,
		stream.onWord,
		stream.onPlaybackStart,
		stream.onPlaybackStop,
		stream.onPlaybackPause,
		stream.onPlaybackResume,
	)

	return stream
}

// DefaultStreamConfig 返回默认流配置
func DefaultStreamConfig() *StreamConfig {
	return &StreamConfig{
		AudioConfig:             DefaultAudioConfig(),
		BufferThresholdSeconds:  2.0,
		MinimumSentenceLength:   10,
		FastSentenceFragment:    true,
		CommaSilenceDuration:    100 * time.Millisecond,
		SentenceSilenceDuration: 300 * time.Millisecond,
		OutputWavFile:           "",
		LogCharacters:           false,
		OutputDeviceIndex:       0,
		Tokenizer:               "nltk",
		Language:                "en",
		Muted:                   false,
	}
}

// Feed 输入文本
func (tts *TextToAudioStream) Feed(text string) error {
	tts.mu.Lock()
	defer tts.mu.Unlock()

	if tts.isPlaying {
		return fmt.Errorf("正在播放中，无法输入新文本")
	}

	// 发送文本到缓冲区
	select {
	case tts.textBuffer <- text:
		return nil
	default:
		return fmt.Errorf("文本缓冲区已满")
	}
}

// FeedAsync 异步输入文本
func (tts *TextToAudioStream) FeedAsync(text string) {
	go func() {
		if err := tts.Feed(text); err != nil {
			tts.callbacks.SafeCallWithArgs(tts.callbacks.OnError, err)
		}
	}()
}

// Play 同步播放
func (tts *TextToAudioStream) Play() error {
	tts.playLock.Lock()
	defer tts.playLock.Unlock()

	if tts.isPlaying {
		return fmt.Errorf("已经在播放中")
	}

	tts.isPlaying = true
	tts.isPaused = false

	// 启动播放协程
	go tts.playWorker()

	return nil
}

// PlayAsync 异步播放
func (tts *TextToAudioStream) PlayAsync() {
	go func() {
		if err := tts.Play(); err != nil {
			tts.callbacks.SafeCallWithArgs(tts.callbacks.OnError, err)
		}
	}()
}

// playWorker 播放工作协程
func (tts *TextToAudioStream) playWorker() {
	defer func() {
		tts.mu.Lock()
		tts.isPlaying = false
		tts.mu.Unlock()
	}()

	// 启动播放器
	if err := tts.player.Start(); err != nil {
		tts.callbacks.SafeCallWithArgs(tts.callbacks.OnError, err)
		return
	}

	// 处理文本流
	for {
		select {
		case text := <-tts.textBuffer:
			if err := tts.processText(text); err != nil {
				tts.callbacks.SafeCallWithArgs(tts.callbacks.OnError, err)
				return
			}
		case <-tts.ctx.Done():
			return
		default:
			// 检查是否还有文本需要处理
			if len(tts.textBuffer) == 0 {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

// processText 处理文本
func (tts *TextToAudioStream) processText(text string) error {
	// 触发文本流开始回调
	tts.callbacks.SafeCall(tts.callbacks.OnTextStreamStart)

	// 分词处理
	sentences := tts.textProcessor.splitIntoSentences(text)

	for _, sentence := range sentences {
		if err := tts.synthesizeSentence(sentence); err != nil {
			return err
		}
	}

	// 触发文本流结束回调
	tts.callbacks.SafeCall(tts.callbacks.OnTextStreamStop)

	return nil
}

// synthesizeSentence 合成句子
func (tts *TextToAudioStream) synthesizeSentence(sentence string) error {
	// 触发句子合成开始回调
	tts.callbacks.SafeCallWithArgs(tts.callbacks.OnEngineSynthesisStart, tts.getCurrentEngineName())

	startTime := time.Now()

	// 获取当前引擎
	engine := tts.getCurrentEngine()
	if engine == nil {
		return fmt.Errorf("没有可用的引擎")
	}

	// 合成音频
	audioChunks, err := engine.Synthesize(tts.ctx, sentence)
	if err != nil {
		// 尝试切换到下一个引擎
		if tts.switchToNextEngine() {
			return tts.synthesizeSentence(sentence) // 重试
		}
		return fmt.Errorf("所有引擎都失败了: %w", err)
	}

	// 发送音频数据到播放器
	for audioData := range audioChunks {
		select {
		case tts.ttsAudioChan <- [][]byte{audioData}:
		case <-tts.ctx.Done():
			return tts.ctx.Err()
		}
	}

	// 触发句子合成完成回调
	duration := time.Since(startTime)
	tts.callbacks.SafeCallWithArgs(tts.callbacks.OnSentenceSynthesized, sentence, duration)

	return nil
}

// getCurrentEngine 获取当前引擎
func (tts *TextToAudioStream) getCurrentEngine() TTSEngine {
	tts.mu.RLock()
	defer tts.mu.RUnlock()

	if len(tts.engines) == 0 {
		return nil
	}

	return tts.engines[tts.currentEngine]
}

// getCurrentEngineName 获取当前引擎名称
func (tts *TextToAudioStream) getCurrentEngineName() string {
	engine := tts.getCurrentEngine()
	if engine == nil {
		return "unknown"
	}
	return engine.GetEngineInfo().Name
}

// switchToNextEngine 切换到下一个引擎
func (tts *TextToAudioStream) switchToNextEngine() bool {
	tts.mu.Lock()
	defer tts.mu.Unlock()

	if len(tts.engines) <= 1 {
		return false // 只有一个引擎，无法切换
	}

	oldEngine := tts.engines[tts.currentEngine].GetEngineInfo().Name
	tts.currentEngine = (tts.currentEngine + 1) % len(tts.engines)
	newEngine := tts.engines[tts.currentEngine].GetEngineInfo().Name

	// 触发引擎切换回调
	tts.callbacks.SafeCallWithArgs(tts.callbacks.OnEngineSwitch, oldEngine, newEngine)

	return true
}

// Pause 暂停播放
func (tts *TextToAudioStream) Pause() error {
	tts.mu.Lock()
	defer tts.mu.Unlock()

	if !tts.isPlaying {
		return fmt.Errorf("未在播放中")
	}

	if tts.isPaused {
		return fmt.Errorf("已经暂停")
	}

	tts.isPaused = true
	return tts.player.Pause()
}

// Resume 恢复播放
func (tts *TextToAudioStream) Resume() error {
	tts.mu.Lock()
	defer tts.mu.Unlock()

	if !tts.isPlaying {
		return fmt.Errorf("未在播放中")
	}

	if !tts.isPaused {
		return fmt.Errorf("未暂停")
	}

	tts.isPaused = false
	return tts.player.Resume()
}

// Stop 停止播放
func (tts *TextToAudioStream) Stop() error {
	tts.playLock.Lock()
	defer tts.playLock.Unlock()

	if !tts.isPlaying {
		return nil
	}

	tts.isPlaying = false
	tts.isPaused = false

	// 取消上下文
	tts.cancel()

	// 停止播放器
	return tts.player.Stop()
}

// SetCallbacks 设置回调函数
func (tts *TextToAudioStream) SetCallbacks(callbacks *Callbacks) {
	tts.mu.Lock()
	defer tts.mu.Unlock()

	tts.callbacks = callbacks
	tts.textProcessor.callbacks = callbacks
}

// GetStatus 获取状态信息
func (tts *TextToAudioStream) GetStatus() map[string]interface{} {
	tts.mu.RLock()
	defer tts.mu.RUnlock()

	status := map[string]interface{}{
		"is_playing":     tts.isPlaying,
		"is_paused":      tts.isPaused,
		"current_engine": tts.getCurrentEngineName(),
		"engine_count":   len(tts.engines),
	}

	if tts.player != nil {
		status["player_stats"] = tts.player.GetStats()
	}

	return status
}

// GetBufferStats 获取缓冲管理器统计信息
func (tts *TextToAudioStream) GetBufferStats() BufferStats {
	if tts.player != nil && tts.player.bufferManager != nil {
		return tts.player.bufferManager.GetStats()
	}
	return BufferStats{}
}

// GetPlaybackStats 获取播放器统计信息
func (tts *TextToAudioStream) GetPlaybackStats() PlaybackStats {
	if tts.player != nil {
		return tts.player.GetStats()
	}
	return PlaybackStats{}
}

// WaitForPlaybackComplete 等待播放完成
func (tts *TextToAudioStream) WaitForPlaybackComplete(timeout time.Duration) error {
	if tts.player != nil {
		return tts.player.WaitForPlaybackComplete(timeout)
	}
	return fmt.Errorf("播放器未初始化")
}

// Close 关闭流
func (tts *TextToAudioStream) Close() error {
	// 停止播放
	if err := tts.Stop(); err != nil {
		return err
	}

	// 停止播放器
	if tts.player != nil {
		if err := tts.player.Stop(); err != nil {
			return err
		}
	}

	// 关闭引擎
	for _, engine := range tts.engines {
		if err := engine.Close(); err != nil {
			return err
		}
	}

	// 关闭通道
	close(tts.textBuffer)
	close(tts.charBuffer)
	close(tts.ttsAudioChan)

	return nil
}

// 回调函数实现
func (tts *TextToAudioStream) onAudioChunk(data []byte) {
	tts.callbacks.SafeCallWithArgs(tts.callbacks.OnAudioChunk, data)
}

func (tts *TextToAudioStream) onWord(timing TimingInfo) {
	tts.callbacks.SafeCallWithArgs(tts.callbacks.OnWord, timing.Word)
}

func (tts *TextToAudioStream) onPlaybackStart() {
	tts.callbacks.SafeCall(tts.callbacks.OnPlaybackStart)
}

func (tts *TextToAudioStream) onPlaybackStop() {
	tts.callbacks.SafeCall(tts.callbacks.OnPlaybackStop)
}

func (tts *TextToAudioStream) onPlaybackPause() {
	tts.callbacks.SafeCall(tts.callbacks.OnPlaybackPause)
}

func (tts *TextToAudioStream) onPlaybackResume() {
	tts.callbacks.SafeCall(tts.callbacks.OnPlaybackResume)
}

// TextProcessor 方法实现
func (tp *TextProcessor) splitIntoSentences(text string) []string {
	// 简单的句子分割逻辑
	sentences := strings.Split(text, ".")

	var result []string
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence != "" {
			result = append(result, sentence)
		}
	}

	return result
}
