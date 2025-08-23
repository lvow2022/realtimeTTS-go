package engines

import (
	"context"
	"fmt"
	"sync"
	"time"

	"realtimetts"
)

// BaseEngine 基础引擎抽象类
// 提供通用的引擎功能和默认实现
type BaseEngine struct {
	mu sync.RWMutex

	// 引擎基本信息
	engineInfo   realtimetts.EngineInfo
	config       realtimetts.EngineConfig
	currentVoice realtimetts.Voice
	status       realtimetts.EngineStatus
	lastError    error

	// 音频处理
	audioChunks  chan realtimetts.AudioChunk
	stopChan     chan struct{}
	isProcessing bool

	// 统计信息
	stats *EngineStats

	// 子类合成器接口
	synthesizer Synthesizer
}

// Synthesizer 合成器接口
type Synthesizer interface {
	DoSynthesize(ctx context.Context, text string, outputChan chan<- realtimetts.AudioChunk) error
}

// EngineStats 引擎统计信息
type EngineStats struct {
	mu                  sync.RWMutex
	TotalSynthesis      int64         // 总合成次数
	SuccessfulSynthesis int64         // 成功合成次数
	FailedSynthesis     int64         // 失败合成次数
	TotalDuration       time.Duration // 总合成时长
	AverageLatency      time.Duration // 平均延迟
	LastSynthesisTime   time.Time     // 最后合成时间
}

// NewBaseEngine 创建新的基础引擎
func NewBaseEngine(name, version, description string) *BaseEngine {
	return &BaseEngine{
		engineInfo: realtimetts.EngineInfo{
			Name:         name,
			Version:      version,
			Description:  description,
			Capabilities: []string{"text-to-speech", "voice-selection"},
			Config:       make(map[string]string),
		},
		config:       *realtimetts.DefaultEngineConfig(),
		currentVoice: realtimetts.Voice{},
		status:       realtimetts.EngineStatusUninitialized,
		lastError:    nil,
		audioChunks:  make(chan realtimetts.AudioChunk, 100),
		stopChan:     make(chan struct{}),
		isProcessing: false,
		stats: &EngineStats{
			TotalSynthesis:      0,
			SuccessfulSynthesis: 0,
			FailedSynthesis:     0,
			TotalDuration:       0,
			AverageLatency:      0,
			LastSynthesisTime:   time.Time{},
		},
	}
}

// Synthesize 基础合成方法（需要子类重写）
func (be *BaseEngine) Synthesize(ctx context.Context, text string) (<-chan realtimetts.AudioChunk, error) {
	be.mu.Lock()
	defer be.mu.Unlock()

	if be.status != realtimetts.EngineStatusReady {
		return nil, fmt.Errorf("引擎未就绪，当前状态: %s", be.status.String())
	}

	if be.isProcessing {
		return nil, fmt.Errorf("引擎正在处理中")
	}

	// 更新状态
	be.status = realtimetts.EngineStatusSynthesizing
	be.isProcessing = true
	be.stats.mu.Lock()
	be.stats.TotalSynthesis++
	be.stats.LastSynthesisTime = time.Now()
	be.stats.mu.Unlock()

	// 创建新的音频块通道
	outputChan := make(chan realtimetts.AudioChunk, 50)

	// 启动合成协程
	go be.synthesisWorker(ctx, text, outputChan)

	return outputChan, nil
}

// synthesisWorker 合成工作协程
func (be *BaseEngine) synthesisWorker(ctx context.Context, text string, outputChan chan<- realtimetts.AudioChunk) {
	defer func() {
		be.mu.Lock()
		be.isProcessing = false
		be.status = realtimetts.EngineStatusReady
		be.mu.Unlock()
		close(outputChan)
	}()

	startTime := time.Now()
	fmt.Printf("   synthesisWorker开始处理文本: %s\n", text)

	// 调用子类的具体合成实现
	var err error
	if be.synthesizer != nil {
		err = be.synthesizer.DoSynthesize(ctx, text, outputChan)
	} else {
		err = be.doSynthesize(ctx, text, outputChan)
	}

	// 更新统计信息
	duration := time.Since(startTime)
	fmt.Printf("   synthesisWorker完成, 耗时: %v, 错误: %v\n", duration, err)

	be.stats.mu.Lock()
	be.stats.TotalDuration += duration
	if err != nil {
		be.stats.FailedSynthesis++
		be.lastError = err
	} else {
		be.stats.SuccessfulSynthesis++
	}
	// 计算平均延迟
	if be.stats.TotalSynthesis > 0 {
		be.stats.AverageLatency = be.stats.TotalDuration / time.Duration(be.stats.TotalSynthesis)
	}
	be.stats.mu.Unlock()

	// 检查上下文取消
	select {
	case <-ctx.Done():
		return
	case <-be.stopChan:
		return
	default:
	}
}

// doSynthesize 具体的合成实现（需要子类重写）
func (be *BaseEngine) doSynthesize(ctx context.Context, text string, outputChan chan<- realtimetts.AudioChunk) error {
	// 默认实现：返回错误，要求子类重写
	return fmt.Errorf("doSynthesize方法需要子类实现")
}

// GetEngineInfo 获取引擎信息
func (be *BaseEngine) GetEngineInfo() realtimetts.EngineInfo {
	be.mu.RLock()
	defer be.mu.RUnlock()
	return be.engineInfo
}

// GetSupportedVoices 获取支持的语音列表（需要子类重写）
func (be *BaseEngine) GetSupportedVoices() ([]realtimetts.Voice, error) {
	// 默认实现：返回空列表，要求子类重写
	return []realtimetts.Voice{}, fmt.Errorf("GetSupportedVoices方法需要子类实现")
}

// SetVoice 设置语音
func (be *BaseEngine) SetVoice(voice realtimetts.Voice) error {
	be.mu.Lock()
	defer be.mu.Unlock()

	// 验证语音是否支持
	voices, err := be.GetSupportedVoices()
	if err != nil {
		return fmt.Errorf("获取支持的语音列表失败: %w", err)
	}

	found := false
	for _, v := range voices {
		if v.ID == voice.ID {
			be.currentVoice = voice
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("不支持的语音: %s", voice.ID)
	}

	return nil
}

// SetConfig 设置引擎配置
func (be *BaseEngine) SetConfig(config realtimetts.EngineConfig) error {
	if err := realtimetts.ValidateEngineConfig(&config); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	be.mu.Lock()
	defer be.mu.Unlock()

	be.config = config
	return nil
}

// IsReady 检查引擎是否就绪
func (be *BaseEngine) IsReady() bool {
	be.mu.RLock()
	defer be.mu.RUnlock()
	return be.status == realtimetts.EngineStatusReady
}

// Close 关闭引擎
func (be *BaseEngine) Close() error {
	be.mu.Lock()
	defer be.mu.Unlock()

	if be.status == realtimetts.EngineStatusClosed {
		return nil
	}

	// 停止所有处理
	close(be.stopChan)
	be.status = realtimetts.EngineStatusClosed

	// 关闭音频块通道
	close(be.audioChunks)

	return nil
}

// GetStatus 获取引擎状态
func (be *BaseEngine) GetStatus() realtimetts.EngineStatus {
	be.mu.RLock()
	defer be.mu.RUnlock()
	return be.status
}

// GetLastError 获取最后的错误
func (be *BaseEngine) GetLastError() error {
	be.mu.RLock()
	defer be.mu.RUnlock()
	return be.lastError
}

// GetStats 获取统计信息
func (be *BaseEngine) GetStats() EngineStats {
	be.stats.mu.RLock()
	defer be.stats.mu.RUnlock()
	return *be.stats
}

// Initialize 初始化引擎（需要子类重写）
func (be *BaseEngine) Initialize() error {
	be.mu.Lock()
	defer be.mu.Unlock()

	if be.status == realtimetts.EngineStatusInitializing || be.status == realtimetts.EngineStatusReady {
		return nil
	}

	be.status = realtimetts.EngineStatusInitializing

	// 调用子类的具体初始化实现
	err := be.doInitialize()
	if err != nil {
		be.status = realtimetts.EngineStatusError
		be.lastError = err
		return err
	}

	be.status = realtimetts.EngineStatusReady
	return nil
}

// doInitialize 具体的初始化实现（需要子类重写）
func (be *BaseEngine) doInitialize() error {
	// 默认实现：返回成功，子类可以重写
	return nil
}

// createAudioChunk 创建音频块（简化版本，直接返回数据）
func (be *BaseEngine) createAudioChunk(data []byte, timestamp time.Time, duration time.Duration, sequence int64, isEnd bool) []byte {
	return data
}

// sendAudioChunk 发送音频数据到输出通道
func (be *BaseEngine) sendAudioChunk(audioData []byte, outputChan chan<- []byte, ctx context.Context) bool {
	select {
	case outputChan <- audioData:
		return true
	case <-ctx.Done():
		return false
	case <-be.stopChan:
		return false
	default:
		// 通道已满，丢弃数据
		return false
	}
}
