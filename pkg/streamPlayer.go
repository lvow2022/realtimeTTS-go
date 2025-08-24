package realtimetts

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// StreamPlayer 流播放器
// 集成 bufferManager 和 audioStream
// 负责将音频流从buffer 中取出，送入 audioStream
// 提供 start/stop/pause/resume/mute 接口
type StreamPlayer struct {
	bufferManager *AudioBuffer
	audioStream   *AudioStream

	// 播放控制
	mu             sync.RWMutex
	playbackThread *PlaybackThread
	playbackActive bool
	playbackPaused bool
	immediateStop  chan struct{}
	pauseEvent     chan struct{}
	resumeEvent    chan struct{}

	// 回调函数
	onAudioChunk     func([]byte)
	onWord           func(TimingInfo)
	onPlaybackStart  func()
	onPlaybackStop   func()
	onPlaybackPause  func()
	onPlaybackResume func()

	// 统计信息
	stats *PlaybackStats
}

// PlaybackThread 播放线程
type PlaybackThread struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// PlaybackStats 播放统计信息
type PlaybackStats struct {
	mu               sync.RWMutex
	BytesPlayed      int64         // 已播放字节数
	ChunksPlayed     int64         // 已播放块数
	WordsPlayed      int64         // 已播放单词数
	PlaybackDuration time.Duration // 播放时长
	StartTime        time.Time     // 开始时间
	LastActivityTime time.Time     // 最后活动时间
}

// NewStreamPlayer 创建新的流播放器
func NewStreamPlayer(audioBuffer *AudioBuffer, config *AudioConfiguration, bufferSize int) *StreamPlayer {
	audioStream := NewAudioStream(config)

	return &StreamPlayer{
		bufferManager:    audioBuffer,
		audioStream:      audioStream,
		playbackThread:   nil,
		playbackActive:   false,
		playbackPaused:   false,
		immediateStop:    make(chan struct{}),
		pauseEvent:       make(chan struct{}, 1),
		resumeEvent:      make(chan struct{}, 1),
		onAudioChunk:     nil,
		onWord:           nil,
		onPlaybackStart:  nil,
		onPlaybackStop:   nil,
		onPlaybackPause:  nil,
		onPlaybackResume: nil,
		stats: &PlaybackStats{
			BytesPlayed:      0,
			ChunksPlayed:     0,
			WordsPlayed:      0,
			PlaybackDuration: 0,
			StartTime:        time.Time{},
			LastActivityTime: time.Time{},
		},
	}
}

// Start 开始播放
func (sp *StreamPlayer) Start() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if sp.playbackActive {
		return ErrPlayerAlreadyPlaying
	}

	// 打开音频流
	if err := sp.audioStream.OpenStream(); err != nil {
		return fmt.Errorf("打开音频流失败: %w", err)
	}

	// 启动音频流
	if err := sp.audioStream.StartStream(); err != nil {
		return fmt.Errorf("启动音频流失败: %w", err)
	}

	// 创建播放线程
	ctx, cancel := context.WithCancel(context.Background())
	sp.playbackThread = &PlaybackThread{
		ctx:    ctx,
		cancel: cancel,
	}

	sp.playbackActive = true
	sp.playbackPaused = false

	// 重置停止信号
	sp.immediateStop = make(chan struct{})
	sp.pauseEvent = make(chan struct{})
	sp.resumeEvent = make(chan struct{})

	// 启动播放协程
	go sp.playbackWorker()

	// 更新统计信息
	sp.stats.mu.Lock()
	sp.stats.StartTime = time.Now()
	sp.stats.LastActivityTime = time.Now()
	sp.stats.mu.Unlock()

	// 触发回调
	if sp.onPlaybackStart != nil {
		sp.onPlaybackStart()
	}

	return nil
}

// Stop 停止播放
func (sp *StreamPlayer) Stop() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if !sp.playbackActive {
		return ErrPlayerNotPlaying
	}

	// 发送停止信号
	close(sp.immediateStop)

	// 取消播放线程
	if sp.playbackThread != nil {
		sp.playbackThread.cancel()
		sp.playbackThread = nil
	}

	sp.playbackActive = false
	sp.playbackPaused = false

	// 停止音频流
	if err := sp.audioStream.StopStream(); err != nil {
		return fmt.Errorf("停止音频流失败: %w", err)
	}

	// 关闭音频流
	if err := sp.audioStream.CloseStream(); err != nil {
		return fmt.Errorf("关闭音频流失败: %w", err)
	}

	// 清空缓冲区
	sp.bufferManager.ClearBuffer()

	// 更新统计信息
	sp.stats.mu.Lock()
	sp.stats.PlaybackDuration = time.Since(sp.stats.StartTime)
	sp.stats.LastActivityTime = time.Now()
	sp.stats.mu.Unlock()

	// 触发回调
	if sp.onPlaybackStop != nil {
		sp.onPlaybackStop()
	}

	return nil
}

// Pause 暂停播放
func (sp *StreamPlayer) Pause() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if !sp.playbackActive {
		return ErrPlayerNotPlaying
	}

	if sp.playbackPaused {
		return ErrPlayerPaused
	}

	sp.playbackPaused = true

	// 发送暂停信号
	select {
	case sp.pauseEvent <- struct{}{}:
	default:
		// 通道已满，忽略
	}

	// 触发回调
	if sp.onPlaybackPause != nil {
		sp.onPlaybackPause()
	}

	return nil
}

// Resume 恢复播放
func (sp *StreamPlayer) Resume() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if !sp.playbackActive {
		return ErrPlayerNotPlaying
	}

	if !sp.playbackPaused {
		return nil
	}

	sp.playbackPaused = false

	// 发送恢复信号
	select {
	case sp.resumeEvent <- struct{}{}:
	default:
		// 通道已满，忽略
	}

	// 触发回调
	if sp.onPlaybackResume != nil {
		sp.onPlaybackResume()
	}

	return nil
}

// Mute 静音
func (sp *StreamPlayer) Mute() error {
	return sp.audioStream.SetMuted(true)
}

// Unmute 取消静音
func (sp *StreamPlayer) Unmute() error {
	return sp.audioStream.SetMuted(false)
}

// SetVolume 设置音量
func (sp *StreamPlayer) SetVolume(volume float64) error {
	return sp.audioStream.SetVolume(volume)
}

// GetVolume 获取音量
func (sp *StreamPlayer) GetVolume() float64 {
	return sp.audioStream.GetVolume()
}

// IsPlaying 检查是否正在播放
func (sp *StreamPlayer) IsPlaying() bool {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	return sp.playbackActive && !sp.playbackPaused
}

// IsPaused 检查是否已暂停
func (sp *StreamPlayer) IsPaused() bool {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	return sp.playbackActive && sp.playbackPaused
}

// IsActive 检查播放器是否激活
func (sp *StreamPlayer) IsActive() bool {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	return sp.playbackActive
}

// GetBufferedSeconds 获取缓冲的音频时长
func (sp *StreamPlayer) GetBufferedSeconds() float64 {
	return sp.bufferManager.GetBufferedSeconds()
}

// GetStats 获取播放统计信息
func (sp *StreamPlayer) GetStats() PlaybackStats {
	sp.stats.mu.RLock()
	defer sp.stats.mu.RUnlock()

	return PlaybackStats{
		BytesPlayed:      sp.stats.BytesPlayed,
		ChunksPlayed:     sp.stats.ChunksPlayed,
		WordsPlayed:      sp.stats.WordsPlayed,
		PlaybackDuration: sp.stats.PlaybackDuration,
		StartTime:        sp.stats.StartTime,
		LastActivityTime: sp.stats.LastActivityTime,
	}
}

// WaitForPlaybackComplete 等待播放完成
func (sp *StreamPlayer) WaitForPlaybackComplete(timeout time.Duration) error {
	fmt.Printf("   ⏳ 等待播放完成，超时时间: %v\n", timeout)

	startTime := time.Now()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var lastActivityTime time.Time
	var totalBytesPlayed int64
	noActivityStartTime := time.Time{}

	for {
		select {
		case <-ticker.C:
			// 检查是否还有音频数据在缓冲区中
			sp.stats.mu.RLock()
			currentLastActivity := sp.stats.LastActivityTime
			currentBytesPlayed := sp.stats.BytesPlayed
			sp.stats.mu.RUnlock()

			// 如果有新的音频活动，重置无活动开始时间
			if currentLastActivity.After(lastActivityTime) {
				lastActivityTime = currentLastActivity
				totalBytesPlayed = currentBytesPlayed
				noActivityStartTime = time.Time{} // 重置
				fmt.Printf("   📊 播放进度: %d 字节\n", currentBytesPlayed)
			} else if noActivityStartTime.IsZero() {
				// 开始记录无活动时间
				noActivityStartTime = time.Now()
				fmt.Printf("   ⏸️  检测到音频活动停止，开始等待PortAudio缓冲区播放完成...\n")
			}

			// 如果没有活动超过3秒，认为播放完成
			// 3秒足够PortAudio播放完缓冲区中的数据
			if !noActivityStartTime.IsZero() && time.Since(noActivityStartTime) > 3*time.Second {
				fmt.Printf("   ✅ 播放完成！总播放字节数: %d\n", totalBytesPlayed)

				// 额外等待1秒确保PortAudio内部缓冲区播放完成
				fmt.Printf("   ⏳ 额外等待1秒确保PortAudio缓冲区播放完成...\n")
				time.Sleep(1 * time.Second)
				fmt.Printf("   🎵 PortAudio缓冲区播放完成！\n")
				return nil
			}

			// 检查超时
			if time.Since(startTime) > timeout {
				fmt.Printf("   ⚠️  等待播放完成超时\n")
				return fmt.Errorf("等待播放完成超时")
			}
		}
	}
}

// SetCallbacks 设置回调函数
func (sp *StreamPlayer) SetCallbacks(
	onAudioChunk func([]byte),
	onWord func(TimingInfo),
	onPlaybackStart func(),
	onPlaybackStop func(),
	onPlaybackPause func(),
	onPlaybackResume func(),
) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	sp.onAudioChunk = onAudioChunk
	sp.onWord = onWord
	sp.onPlaybackStart = onPlaybackStart
	sp.onPlaybackStop = onPlaybackStop
	sp.onPlaybackPause = onPlaybackPause
	sp.onPlaybackResume = onPlaybackResume
}

// playbackWorker 播放工作协程
func (sp *StreamPlayer) playbackWorker() {
	ticker := time.NewTicker(5 * time.Millisecond) // 5ms 检查间隔，提高响应性
	defer ticker.Stop()

	fmt.Println("   🔄 playbackWorker 启动")
	loopCount := 0

	for {
		select {
		case <-sp.immediateStop:
			fmt.Println("   🛑 playbackWorker 收到停止信号")
			return

		case <-sp.pauseEvent:
			fmt.Println("   ⏸️  playbackWorker 收到暂停信号")
			// 等待恢复信号
			select {
			case <-sp.resumeEvent:
				fmt.Println("   ▶️  playbackWorker 收到恢复信号")
				continue
			case <-sp.immediateStop:
				fmt.Println("   🛑 playbackWorker 暂停时收到停止信号")
				return
			}

		case <-ticker.C:
			loopCount++

			// 处理音频数据
			if err := sp.processAudioChunk(); err != nil {
				// 如果缓冲区为空，继续等待
				if err == ErrBufferTimeout {
					continue
				}
				// 其他错误，停止播放
				fmt.Printf("   ❌ 处理音频块错误: %v，停止播放\n", err)
				sp.Stop()
				return
			}

			// 处理时间信息
			sp.processTimingInfo()
		}
	}
}

// processAudioChunk 处理音频块
func (sp *StreamPlayer) processAudioChunk() error {
	// 从缓冲区获取音频数据
	audioData, err := sp.bufferManager.GetFromBuffer(200 * time.Millisecond)
	if err != nil {
		return err
	}

	// 写入音频流
	if err := sp.audioStream.WriteAudioData(audioData); err != nil {
		return fmt.Errorf("写入音频数据失败: %w", err)
	}

	// 更新统计信息
	sp.stats.mu.Lock()
	sp.stats.BytesPlayed += int64(len(audioData))
	sp.stats.ChunksPlayed++
	sp.stats.LastActivityTime = time.Now()
	sp.stats.mu.Unlock()

	// 触发回调
	if sp.onAudioChunk != nil {
		sp.onAudioChunk(audioData)
	}

	return nil
}

// processTimingInfo 处理时间信息
func (sp *StreamPlayer) processTimingInfo() {
	// 从缓冲区获取时间信息
	timing, err := sp.bufferManager.GetTimingInfo(10 * time.Millisecond)
	if err != nil {
		return
	}

	// 更新统计信息
	sp.stats.mu.Lock()
	sp.stats.WordsPlayed++
	sp.stats.mu.Unlock()

	// 触发回调
	if sp.onWord != nil {
		sp.onWord(timing)
	}
}
