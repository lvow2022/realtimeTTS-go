package realtimetts

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/oto"
)

// TextToAudioStream 文本到音频流处理器
type TextToAudioStream struct {
	engines   []Engine
	player    *AudioPlayer
	callbacks *Callbacks

	// 控制通道
	stopChan   chan struct{}
	pauseChan  chan struct{}
	resumeChan chan struct{}

	// 状态
	isPlaying bool
	mu        sync.RWMutex

	// 文本处理
	textBuffer strings.Builder
}

// NewTextToAudioStream 创建文本到音频流
func NewTextToAudioStream(engine Engine, config *AudioConfig) *TextToAudioStream {
	return &TextToAudioStream{
		engines:    []Engine{engine},
		player:     NewAudioPlayer(config),
		stopChan:   make(chan struct{}),
		pauseChan:  make(chan struct{}),
		resumeChan: make(chan struct{}),
	}
}

// NewTextToAudioStreamWithEngines 创建多引擎文本到音频流
func NewTextToAudioStreamWithEngines(engines []Engine, config *AudioConfig) *TextToAudioStream {
	return &TextToAudioStream{
		engines:    engines,
		player:     NewAudioPlayer(config),
		stopChan:   make(chan struct{}),
		pauseChan:  make(chan struct{}),
		resumeChan: make(chan struct{}),
	}
}

// Feed 输入文本
func (s *TextToAudioStream) Feed(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 触发文本流开始回调
	if s.callbacks != nil && s.callbacks.OnTextStreamStart != nil {
		s.callbacks.OnTextStreamStart()
	}

	// 处理字符回调
	if s.callbacks != nil && s.callbacks.OnCharacter != nil {
		for _, char := range text {
			s.callbacks.OnCharacter(char)
		}
	}

	s.textBuffer.WriteString(text)
}

// FeedGenerator 输入文本生成器
func (s *TextToAudioStream) FeedGenerator(textChan <-chan string) {
	go func() {
		for text := range textChan {
			s.Feed(text)
		}
	}()
}

// Play 同步播放
func (s *TextToAudioStream) Play(ctx context.Context) error {
	s.mu.Lock()
	if s.isPlaying {
		s.mu.Unlock()
		return fmt.Errorf("已经在播放中")
	}
	s.isPlaying = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.isPlaying = false
		s.mu.Unlock()
	}()

	// 设置回调
	s.player.SetCallbacks(s.callbacks)
	for _, engine := range s.engines {
		engine.SetCallbacks(s.callbacks)
	}

	// 启动文本处理
	go s.processTextSynthesis(ctx)

	// 启动音频播放
	return s.player.PlayAudio(ctx, s.engines[0].GetAudioQueue())
}

// PlayAsync 异步播放
func (s *TextToAudioStream) PlayAsync(ctx context.Context) {
	go func() {
		if err := s.Play(ctx); err != nil {
			log.Printf("播放失败: %v", err)
		}
	}()
}

// Pause 暂停播放
func (s *TextToAudioStream) Pause() {
	s.mu.RLock()
	if !s.isPlaying {
		s.mu.RUnlock()
		return
	}
	s.mu.RUnlock()

	s.player.Pause()
	select {
	case s.pauseChan <- struct{}{}:
	default:
	}
}

// Resume 恢复播放
func (s *TextToAudioStream) Resume() {
	s.mu.RLock()
	if !s.isPlaying {
		s.mu.RUnlock()
		return
	}
	s.mu.RUnlock()

	s.player.Resume()
	select {
	case s.resumeChan <- struct{}{}:
	default:
	}
}

// Stop 停止播放
func (s *TextToAudioStream) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isPlaying {
		return
	}

	// 停止所有引擎
	for _, engine := range s.engines {
		engine.Stop()
	}

	// 停止播放器
	s.player.Stop()

	// 发送停止信号
	close(s.stopChan)

	s.isPlaying = false

	// 触发播放停止回调
	if s.callbacks != nil && s.callbacks.OnPlaybackStop != nil {
		s.callbacks.OnPlaybackStop()
	}

	// 触发文本流停止回调
	if s.callbacks != nil && s.callbacks.OnTextStreamStop != nil {
		s.callbacks.OnTextStreamStop()
	}
}

// SetCallbacks 设置回调函数
func (s *TextToAudioStream) SetCallbacks(callbacks *Callbacks) {
	s.callbacks = callbacks
}

// IsPlaying 检查是否正在播放
func (s *TextToAudioStream) IsPlaying() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isPlaying
}

// processTextSynthesis 处理文本合成
func (s *TextToAudioStream) processTextSynthesis(ctx context.Context) {
	// 简单的句子分割
	sentences := s.splitIntoSentences(s.textBuffer.String())

	for _, sentence := range sentences {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		default:
			if strings.TrimSpace(sentence) != "" {
				// 尝试所有引擎
				var synthesisSuccess bool
				for _, engine := range s.engines {
					if err := engine.Synthesize(ctx, sentence); err != nil {
						log.Printf("引擎 %s 合成失败: %v", engine.GetName(), err)
						continue
					}
					synthesisSuccess = true
					break
				}

				if !synthesisSuccess {
					log.Printf("所有引擎都失败，跳过句子: %s", sentence)
					continue
				}

				// 触发句子合成回调
				if s.callbacks != nil && s.callbacks.OnSentenceSynthesized != nil {
					s.callbacks.OnSentenceSynthesized(sentence)
				}
			}
		}
	}
}

// splitIntoSentences 分割句子
func (s *TextToAudioStream) splitIntoSentences(text string) []string {
	// 简单的句子分割逻辑
	// 可以根据需要实现更复杂的句子分割算法

	// 按句号、问号、感叹号分割
	delimiters := []string{".", "!", "?", "\n"}
	sentences := []string{text}

	for _, delimiter := range delimiters {
		var newSentences []string
		for _, sentence := range sentences {
			parts := strings.Split(sentence, delimiter)
			for i, part := range parts {
				if i < len(parts)-1 {
					part = part + delimiter
				}
				if strings.TrimSpace(part) != "" {
					newSentences = append(newSentences, strings.TrimSpace(part))
				}
			}
		}
		sentences = newSentences
	}

	return sentences
}

// GetEngines 获取引擎列表
func (s *TextToAudioStream) GetEngines() []Engine {
	return s.engines
}

// AddEngine 添加引擎
func (s *TextToAudioStream) AddEngine(engine Engine) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.engines = append(s.engines, engine)
}

// RemoveEngine 移除引擎
func (s *TextToAudioStream) RemoveEngine(engineName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, engine := range s.engines {
		if engine.GetName() == engineName {
			s.engines = append(s.engines[:i], s.engines[i+1:]...)
			break
		}
	}
}

// AudioPlayer 音频播放器
type AudioPlayer struct {
	config    *AudioConfig
	callbacks *Callbacks

	// 音频播放器
	context *oto.Context
	player  *oto.Player

	// 控制通道
	stopChan   chan struct{}
	pauseChan  chan struct{}
	resumeChan chan struct{}

	// 状态
	isPlaying bool
	mu        sync.RWMutex

	// 播放线程
	playThread chan struct{}
}

// NewAudioPlayer 创建音频播放器
func NewAudioPlayer(config *AudioConfig) *AudioPlayer {
	return &AudioPlayer{
		config:     config,
		stopChan:   make(chan struct{}),
		pauseChan:  make(chan struct{}),
		resumeChan: make(chan struct{}),
		playThread: make(chan struct{}),
	}
}

// Start 启动播放器
func (p *AudioPlayer) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isPlaying {
		return fmt.Errorf("播放器已经在运行")
	}

	// 初始化音频上下文
	var err error
	p.context, err = oto.NewContext(
		p.config.SampleRate,
		p.config.Channels,
		int(p.config.Format),
		p.config.FramesPerBuffer,
	)
	if err != nil {
		// 尝试降级配置
		return p.tryFallbackConfig()
	}

	p.player = p.context.NewPlayer()
	p.isPlaying = true

	// 启动播放线程
	go p.playLoop()

	return nil
}

// Stop 停止播放器
func (p *AudioPlayer) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isPlaying {
		return
	}

	close(p.stopChan)
	p.isPlaying = false

	// 等待播放线程结束
	select {
	case <-p.playThread:
	case <-time.After(1 * time.Second):
		log.Println("播放线程停止超时")
	}

	// 清理资源
	if p.player != nil {
		p.player.Close()
		p.player = nil
	}
	if p.context != nil {
		p.context.Close()
		p.context = nil
	}
}

// Pause 暂停播放
func (p *AudioPlayer) Pause() {
	p.mu.RLock()
	if !p.isPlaying {
		p.mu.RUnlock()
		return
	}
	p.mu.RUnlock()

	select {
	case p.pauseChan <- struct{}{}:
	default:
	}
}

// Resume 恢复播放
func (p *AudioPlayer) Resume() {
	p.mu.RLock()
	if !p.isPlaying {
		p.mu.RUnlock()
		return
	}
	p.mu.RUnlock()

	select {
	case p.resumeChan <- struct{}{}:
	default:
	}
}

// IsPlaying 检查是否正在播放
func (p *AudioPlayer) IsPlaying() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isPlaying
}

// SetCallbacks 设置回调函数
func (p *AudioPlayer) SetCallbacks(callbacks *Callbacks) {
	p.callbacks = callbacks
}

// PlayAudio 播放音频数据
func (p *AudioPlayer) PlayAudio(ctx context.Context, audioQueue <-chan []byte) error {
	if err := p.Start(); err != nil {
		return fmt.Errorf("启动播放器失败: %w", err)
	}
	defer p.Stop()

	// 触发播放开始回调
	if p.callbacks != nil && p.callbacks.OnPlaybackStart != nil {
		p.callbacks.OnPlaybackStart()
	}

	// 监听音频队列
	for {
		select {
		case audioData := <-audioQueue:
			if len(audioData) == 0 {
				// 空数据表示结束
				return nil
			}

			// 播放音频数据
			if err := p.playAudioData(audioData); err != nil {
				return fmt.Errorf("播放音频数据失败: %w", err)
			}

		case <-ctx.Done():
			return ctx.Err()
		case <-p.stopChan:
			return nil
		}
	}
}

// playLoop 播放循环
func (p *AudioPlayer) playLoop() {
	defer close(p.playThread)

	// 播放循环逻辑
	// 这里可以添加更复杂的播放控制逻辑
}

// playAudioData 播放音频数据
func (p *AudioPlayer) playAudioData(data []byte) error {
	p.mu.RLock()
	if p.player == nil {
		p.mu.RUnlock()
		return fmt.Errorf("播放器未初始化")
	}
	player := p.player
	p.mu.RUnlock()

	// 检查是否静音
	if p.config.Muted {
		return nil
	}

	// 写入音频数据
	_, err := player.Write(data)
	if err != nil {
		return fmt.Errorf("写入音频数据失败: %w", err)
	}

	return nil
}

// tryFallbackConfig 尝试降级配置
func (p *AudioPlayer) tryFallbackConfig() error {
	// 尝试不同的音频配置
	fallbackConfigs := []struct {
		sampleRate int
		channels   int
		format     int
	}{
		{16000, 1, 1},
		{8000, 1, 1},
		{44100, 1, 1},
	}

	for _, config := range fallbackConfigs {
		context, err := oto.NewContext(config.sampleRate, config.channels, config.format, 512)
		if err == nil {
			p.context = context
			p.player = context.NewPlayer()
			p.isPlaying = true
			go p.playLoop()
			log.Printf("使用降级配置: 采样率=%d, 声道=%d, 格式=%d", config.sampleRate, config.channels, config.format)
			return nil
		}
	}

	return fmt.Errorf("无法初始化音频设备")
}

// GetConfig 获取配置
func (p *AudioPlayer) GetConfig() *AudioConfig {
	return p.config
}
