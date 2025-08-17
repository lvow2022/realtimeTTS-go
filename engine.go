package realtimetts

import (
	"context"
	"sync"
)

// BaseEngine 基础引擎实现
type BaseEngine struct {
	name         string
	config       *AudioConfig
	callbacks    *Callbacks
	AudioQueue   chan []byte
	TimingsQueue chan TimingInfo
	stopChan     chan struct{}
	mu           sync.RWMutex
	status       EngineStatus
}

// NewBaseEngine 创建基础引擎
func NewBaseEngine(name string, config *AudioConfig) *BaseEngine {
	return &BaseEngine{
		name:         name,
		config:       config,
		AudioQueue:   make(chan []byte, 100),    // 缓冲100个音频块
		TimingsQueue: make(chan TimingInfo, 50), // 缓冲50个时间信息
		stopChan:     make(chan struct{}),
		status:       EngineStatusIdle,
	}
}

// GetName 获取引擎名称
func (e *BaseEngine) GetName() string {
	return e.name
}

// SetCallbacks 设置回调函数
func (e *BaseEngine) SetCallbacks(callbacks *Callbacks) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.callbacks = callbacks
}

// GetAudioQueue 获取音频队列
func (e *BaseEngine) GetAudioQueue() <-chan []byte {
	return e.AudioQueue
}

// GetTimingsQueue 获取时间信息队列
func (e *BaseEngine) GetTimingsQueue() <-chan TimingInfo {
	return e.TimingsQueue
}

// Stop 停止引擎
func (e *BaseEngine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.status != EngineStatusStopped {
		close(e.stopChan)
		e.status = EngineStatusStopped

		// 清空队列
		for len(e.AudioQueue) > 0 {
			<-e.AudioQueue
		}
		for len(e.TimingsQueue) > 0 {
			<-e.TimingsQueue
		}
	}
}

// CanConsumeGenerators 是否可以消费生成器
func (e *BaseEngine) CanConsumeGenerators() bool {
	return false // 默认不支持
}

// GetStatus 获取引擎状态
func (e *BaseEngine) GetStatus() EngineStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.status
}

// SetStatus 设置引擎状态
func (e *BaseEngine) SetStatus(status EngineStatus) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status = status
}

// IsStopped 检查是否已停止
func (e *BaseEngine) IsStopped() bool {
	select {
	case <-e.stopChan:
		return true
	default:
		return false
	}
}

// SendAudioData 发送音频数据
func (e *BaseEngine) SendAudioData(ctx context.Context, data []byte) error {
	select {
	case e.AudioQueue <- data:
		// 触发音频块回调
		if e.callbacks != nil && e.callbacks.OnAudioChunk != nil {
			e.callbacks.OnAudioChunk(data)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-e.stopChan:
		return &EngineError{
			EngineName: e.name,
			Message:    "引擎已停止",
		}
	}
}

// SendTimingInfo 发送时间信息
func (e *BaseEngine) SendTimingInfo(ctx context.Context, info TimingInfo) error {
	select {
	case e.TimingsQueue <- info:
		// 触发单词回调
		if e.callbacks != nil && e.callbacks.OnWord != nil {
			e.callbacks.OnWord(info.Word)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-e.stopChan:
		return &EngineError{
			EngineName: e.name,
			Message:    "引擎已停止",
		}
	}
}

// GetConfig 获取配置
func (e *BaseEngine) GetConfig() *AudioConfig {
	return e.config
}

// GetCallbacks 获取回调函数
func (e *BaseEngine) GetCallbacks() *Callbacks {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.callbacks
}
