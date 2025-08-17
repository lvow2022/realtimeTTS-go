package realtimetts

import (
	"sync"
	"time"
)

// AudioChunk 音频块结构体
type AudioChunk struct {
	Data      []byte        // 音频数据
	Timestamp time.Time     // 时间戳
	Duration  time.Duration // 持续时间
}

// AudioBufferManager 音频缓冲管理器
// 从 tts chan 接收音频流
// 提供add_to_buffer/clear_buffer/get_form_buffer/get_buffered_seconds
type AudioBufferManager struct {
	ttsAudioChan chan []AudioChunk // TTS音频输入通道
	audioBuffer  chan []byte       // 音频数据缓冲区
	timings      chan TimingInfo   // 时间信息缓冲区
	config       *AudioConfiguration

	// 状态管理
	mu           sync.RWMutex
	totalSamples int64 // 总样本数
	bufferSize   int   // 缓冲区大小
	isClosed     bool  // 是否已关闭

	// 统计信息
	bytesProcessed  int64 // 已处理的字节数
	chunksProcessed int64 // 已处理的块数
}

// TimingInfo 时间信息结构体
type TimingInfo struct {
	Word      string        // 单词
	StartTime time.Duration // 开始时间
	EndTime   time.Duration // 结束时间
	Duration  time.Duration // 持续时间
}

// NewAudioBufferManager 创建新的音频缓冲管理器
func NewAudioBufferManager(ttsAudioChan chan []AudioChunk, config *AudioConfiguration, bufferSize int) *AudioBufferManager {
	return &AudioBufferManager{
		ttsAudioChan:    ttsAudioChan,
		audioBuffer:     make(chan []byte, bufferSize),
		timings:         make(chan TimingInfo, bufferSize),
		config:          config,
		bufferSize:      bufferSize,
		totalSamples:    0,
		isClosed:        false,
		bytesProcessed:  0,
		chunksProcessed: 0,
	}
}

// AddToBuffer 添加音频数据到缓冲区
func (abm *AudioBufferManager) AddToBuffer(audioData []byte) error {
	abm.mu.RLock()
	if abm.isClosed {
		abm.mu.RUnlock()
		return ErrBufferFull
	}
	abm.mu.RUnlock()

	select {
	case abm.audioBuffer <- audioData:
		abm.mu.Lock()
		abm.totalSamples += int64(len(audioData) / abm.config.GetBytesPerFrame())
		abm.bytesProcessed += int64(len(audioData))
		abm.chunksProcessed++
		abm.mu.Unlock()
		return nil
	default:
		return ErrBufferFull
	}
}

// AddTimingInfo 添加时间信息到缓冲区
func (abm *AudioBufferManager) AddTimingInfo(timing TimingInfo) error {
	abm.mu.RLock()
	if abm.isClosed {
		abm.mu.RUnlock()
		return ErrBufferFull
	}
	abm.mu.RUnlock()

	select {
	case abm.timings <- timing:
		return nil
	default:
		return ErrBufferFull
	}
}

// GetFromBuffer 从缓冲区获取音频数据
func (abm *AudioBufferManager) GetFromBuffer(timeout time.Duration) ([]byte, error) {
	abm.mu.RLock()
	if abm.isClosed {
		abm.mu.RUnlock()
		return nil, ErrBufferEmpty
	}
	abm.mu.RUnlock()

	select {
	case audioData := <-abm.audioBuffer:
		abm.mu.Lock()
		abm.totalSamples -= int64(len(audioData) / abm.config.GetBytesPerFrame())
		abm.mu.Unlock()
		return audioData, nil
	case <-time.After(timeout):
		return nil, ErrBufferTimeout
	}
}

// GetTimingInfo 从缓冲区获取时间信息
func (abm *AudioBufferManager) GetTimingInfo(timeout time.Duration) (TimingInfo, error) {
	abm.mu.RLock()
	if abm.isClosed {
		abm.mu.RUnlock()
		return TimingInfo{}, ErrBufferEmpty
	}
	abm.mu.RUnlock()

	select {
	case timing := <-abm.timings:
		return timing, nil
	case <-time.After(timeout):
		return TimingInfo{}, ErrBufferTimeout
	}
}

// ClearBuffer 清空缓冲区
func (abm *AudioBufferManager) ClearBuffer() {
	abm.mu.Lock()
	defer abm.mu.Unlock()

	// 清空音频缓冲区
	for len(abm.audioBuffer) > 0 {
		<-abm.audioBuffer
	}

	// 清空时间信息缓冲区
	for len(abm.timings) > 0 {
		<-abm.timings
	}

	abm.totalSamples = 0
}

// GetBufferedSeconds 获取缓冲的音频时长（秒）
func (abm *AudioBufferManager) GetBufferedSeconds() float64 {
	abm.mu.RLock()
	defer abm.mu.RUnlock()

	if abm.config.SampleRate <= 0 {
		return 0.0
	}

	bytesPerSecond := abm.config.GetBytesPerSecond()
	if bytesPerSecond <= 0 {
		return 0.0
	}

	return float64(abm.totalSamples) / float64(abm.config.SampleRate)
}

// GetBufferedBytes 获取缓冲的字节数
func (abm *AudioBufferManager) GetBufferedBytes() int64 {
	abm.mu.RLock()
	defer abm.mu.RUnlock()

	return abm.totalSamples * int64(abm.config.GetBytesPerFrame())
}

// GetBufferUsage 获取缓冲区使用率
func (abm *AudioBufferManager) GetBufferUsage() float64 {
	abm.mu.RLock()
	defer abm.mu.RUnlock()

	if abm.bufferSize <= 0 {
		return 0.0
	}

	return float64(len(abm.audioBuffer)) / float64(abm.bufferSize)
}

// IsEmpty 检查缓冲区是否为空
func (abm *AudioBufferManager) IsEmpty() bool {
	abm.mu.RLock()
	defer abm.mu.RUnlock()

	return len(abm.audioBuffer) == 0 && len(abm.timings) == 0
}

// IsFull 检查缓冲区是否已满
func (abm *AudioBufferManager) IsFull() bool {
	abm.mu.RLock()
	defer abm.mu.RUnlock()

	return len(abm.audioBuffer) >= abm.bufferSize
}

// GetStats 获取缓冲区统计信息
func (abm *AudioBufferManager) GetStats() BufferStats {
	abm.mu.RLock()
	defer abm.mu.RUnlock()

	return BufferStats{
		TotalSamples:    abm.totalSamples,
		BytesProcessed:  abm.bytesProcessed,
		ChunksProcessed: abm.chunksProcessed,
		BufferUsage:     float64(len(abm.audioBuffer)) / float64(abm.bufferSize),
		BufferedSeconds: float64(abm.totalSamples) / float64(abm.config.SampleRate),
		AudioQueueSize:  len(abm.audioBuffer),
		TimingQueueSize: len(abm.timings),
	}
}

// Start 启动从TTS通道读取数据的协程
func (abm *AudioBufferManager) Start() {
	go abm.processTTSAudio()
}

// processTTSAudio 处理TTS音频数据的协程
func (abm *AudioBufferManager) processTTSAudio() {
	for {
		abm.mu.RLock()
		if abm.isClosed {
			abm.mu.RUnlock()
			return
		}
		abm.mu.RUnlock()

		select {
		case audioChunks, ok := <-abm.ttsAudioChan:
			if !ok {
				// TTS通道已关闭
				return
			}

			// 处理每个音频块
			for _, chunk := range audioChunks {
				// 将音频块数据添加到内部缓冲区
				if err := abm.AddToBuffer(chunk.Data); err != nil {
					// 如果缓冲区满了，可以选择丢弃数据或等待
					continue
				}
			}
		}
	}
}

// Close 关闭缓冲区管理器
func (abm *AudioBufferManager) Close() {
	abm.mu.Lock()
	defer abm.mu.Unlock()

	if !abm.isClosed {
		abm.isClosed = true
		close(abm.audioBuffer)
		close(abm.timings)
	}
}

// BufferStats 缓冲区统计信息
type BufferStats struct {
	TotalSamples    int64   // 总样本数
	BytesProcessed  int64   // 已处理字节数
	ChunksProcessed int64   // 已处理块数
	BufferUsage     float64 // 缓冲区使用率
	BufferedSeconds float64 // 缓冲的音频时长（秒）
	AudioQueueSize  int     // 音频队列大小
	TimingQueueSize int     // 时间信息队列大小
}
