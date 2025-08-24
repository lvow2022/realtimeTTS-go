package realtimetts

import (
	"sync"
	"time"
)

// AudioBuffer 音频缓冲管理器
// 直接使用 ttsAudioChan 接收音频流，提供时间信息缓冲
// 提供get_from_buffer/get_timing_info/get_buffered_seconds
type AudioBuffer struct {
	ttsAudioChan chan [][]byte   // TTS音频输入通道
	timings      chan TimingInfo // 时间信息缓冲区
	config       *AudioConfiguration

	// 状态管理
	mu           sync.RWMutex
	totalSamples int64 //total_samples是一个关键的计数器，用于跟踪缓冲区内音频样本的数量，是实现智能缓冲控制和时间计算的核心组件。
	bufferSize   int   // 缓冲区大小
	isClosed     bool  // 是否已关闭

}

// TimingInfo 时间信息结构体
type TimingInfo struct {
	Word      string        // 单词
	StartTime time.Duration // 开始时间
	EndTime   time.Duration // 结束时间
	Duration  time.Duration // 持续时间
}

// NewAudioBuffer 创建新的音频缓冲管理器
func NewAudioBuffer(config *AudioConfiguration, bufferSize int) *AudioBuffer {
	return &AudioBuffer{
		ttsAudioChan: make(chan [][]byte, bufferSize),
		timings:      make(chan TimingInfo, bufferSize),
		config:       config,
		bufferSize:   bufferSize,
		totalSamples: 0,
		isClosed:     false,
	}
}

// AddToBuffer 添加音频数据到缓冲区（已废弃，直接使用ttsAudioChan）
func (ab *AudioBuffer) AddToBuffer(audioData []byte) error {
	// 此方法已废弃，音频数据直接通过ttsAudioChan传递
	return nil
}

// AddTimingInfo 添加时间信息到缓冲区
func (abm *AudioBuffer) AddTimingInfo(timing TimingInfo) error {
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

// GetFromBuffer 从TTS通道获取音频数据
func (abm *AudioBuffer) GetFromBuffer(timeout time.Duration) ([]byte, error) {
	abm.mu.RLock()
	if abm.isClosed {
		abm.mu.RUnlock()
		return nil, ErrBufferEmpty
	}
	abm.mu.RUnlock()

	// 直接从ttsAudioChan读取音频数据
	select {
	case audioChunks, ok := <-abm.ttsAudioChan:
		if !ok {
			return nil, ErrBufferEmpty
		}
		// 返回第一个音频块
		if len(audioChunks) > 0 {
			audioData := audioChunks[0]
			abm.mu.Lock()
			abm.totalSamples += int64(len(audioData) / abm.config.GetBytesPerFrame())
			abm.mu.Unlock()
			return audioData, nil
		}
		return nil, ErrBufferEmpty
	case <-time.After(timeout):
		return nil, ErrBufferTimeout
	}
}

// GetTimingInfo 从缓冲区获取时间信息
func (abm *AudioBuffer) GetTimingInfo(timeout time.Duration) (TimingInfo, error) {
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
func (abm *AudioBuffer) ClearBuffer() {
	abm.mu.Lock()
	defer abm.mu.Unlock()

	// 清空时间信息缓冲区
	for len(abm.timings) > 0 {
		<-abm.timings
	}

	abm.totalSamples = 0
}

// GetBufferedSeconds 获取缓冲的音频时长（秒）
func (abm *AudioBuffer) GetBufferedSeconds() float64 {
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
func (abm *AudioBuffer) GetBufferedBytes() int64 {
	abm.mu.RLock()
	defer abm.mu.RUnlock()

	return abm.totalSamples * int64(abm.config.GetBytesPerFrame())
}

// GetBufferUsage 获取缓冲区使用率
func (abm *AudioBuffer) GetBufferUsage() float64 {
	abm.mu.RLock()
	defer abm.mu.RUnlock()

	// 由于直接使用ttsAudioChan，无法准确计算使用率
	return 0.0
}

// IsEmpty 检查缓冲区是否为空
func (abm *AudioBuffer) IsEmpty() bool {
	abm.mu.RLock()
	defer abm.mu.RUnlock()

	return len(abm.timings) == 0
}

// IsFull 检查缓冲区是否已满
func (abm *AudioBuffer) IsFull() bool {
	abm.mu.RLock()
	defer abm.mu.RUnlock()

	// 由于直接使用ttsAudioChan，无法准确判断是否已满
	return false
}

// GetStats 获取缓冲区统计信息
func (abm *AudioBuffer) GetStats() BufferStats {
	abm.mu.RLock()
	defer abm.mu.RUnlock()

	return BufferStats{
		TotalSamples:    abm.totalSamples,
		BytesProcessed:  0,   // 已移除统计字段
		ChunksProcessed: 0,   // 已移除统计字段
		BufferUsage:     0.0, // 直接使用ttsAudioChan，无法准确计算
		BufferedSeconds: float64(abm.totalSamples) / float64(abm.config.SampleRate),
		AudioQueueSize:  0, // 直接使用ttsAudioChan，无中间队列
		TimingQueueSize: len(abm.timings),
	}
}

// Start 启动音频缓冲管理器（已简化，不再需要单独的协程）
func (abm *AudioBuffer) Start() {
	// 直接使用ttsAudioChan，无需额外的处理协程
}

// Close 关闭缓冲区管理器
func (abm *AudioBuffer) Close() {
	abm.mu.Lock()
	defer abm.mu.Unlock()

	if !abm.isClosed {
		abm.isClosed = true
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
