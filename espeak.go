package realtimetts

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ESpeakEngine ESpeak TTS引擎
type ESpeakEngine struct {
	*BaseEngine
	streamInfo StreamInfo
	voice      string
	rate       int
	pitch      int
	volume     int
}

// NewESpeakEngine 创建ESpeak引擎
func NewESpeakEngine() *ESpeakEngine {
	config := &AudioConfig{
		Format:           FormatPCM,
		Channels:         1,
		SampleRate:       22050,
		Muted:            false,
		FramesPerBuffer:  512,
		PlayoutChunkSize: -1,
	}

	engine := &ESpeakEngine{
		BaseEngine: NewBaseEngine("ESpeak", config),
		streamInfo: StreamInfo{
			Format:     FormatPCM,
			Channels:   1,
			SampleRate: 22050,
		},
		voice:  "en",
		rate:   150,
		pitch:  50,
		volume: 100,
	}

	return engine
}

// Synthesize 合成文本为音频
func (e *ESpeakEngine) Synthesize(ctx context.Context, text string) error {
	if e.IsStopped() {
		return &EngineError{
			EngineName: e.name,
			Message:    "引擎已停止",
		}
	}

	e.SetStatus(EngineStatusSynthesizing)
	defer e.SetStatus(EngineStatusIdle)

	// 检查espeak是否可用
	if !e.isESpeakAvailable() {
		return &EngineError{
			EngineName: e.name,
			Message:    "ESpeak未安装或不可用",
		}
	}

	// 分割文本为单词
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	// 模拟音频合成和时间信息
	go e.simulateSynthesis(ctx, words)

	return nil
}

// GetStreamInfo 获取流信息
func (e *ESpeakEngine) GetStreamInfo() StreamInfo {
	return e.streamInfo
}

// SetVoice 设置语音
func (e *ESpeakEngine) SetVoice(voice string) {
	e.voice = voice
}

// SetRate 设置语速
func (e *ESpeakEngine) SetRate(rate int) {
	e.rate = rate
}

// SetPitch 设置音调
func (e *ESpeakEngine) SetPitch(pitch int) {
	e.pitch = pitch
}

// SetVolume 设置音量
func (e *ESpeakEngine) SetVolume(volume int) {
	e.volume = volume
}

// isESpeakAvailable 检查ESpeak是否可用
func (e *ESpeakEngine) isESpeakAvailable() bool {
	cmd := exec.Command("espeak", "--version")
	return cmd.Run() == nil
}

// simulateSynthesis 模拟音频合成
func (e *ESpeakEngine) simulateSynthesis(ctx context.Context, words []string) {
	wordDuration := 200 * time.Millisecond // 每个单词200ms

	for i, word := range words {
		select {
		case <-ctx.Done():
			return
		case <-e.stopChan:
			return
		default:
			// 模拟音频数据生成
			audioData := e.generateMockAudioData(word)
			if err := e.SendAudioData(ctx, audioData); err != nil {
				fmt.Printf("发送音频数据失败: %v\n", err)
				return
			}

			// 发送时间信息
			wordStart := time.Duration(i) * wordDuration
			wordEnd := wordStart + wordDuration
			timingInfo := TimingInfo{
				Word:      word,
				StartTime: wordStart,
				EndTime:   wordEnd,
				Duration:  wordDuration,
			}

			if err := e.SendTimingInfo(ctx, timingInfo); err != nil {
				fmt.Printf("发送时间信息失败: %v\n", err)
				return
			}

			// 模拟处理延迟
			time.Sleep(50 * time.Millisecond)
		}
	}

	// 发送结束标记
	select {
	case e.AudioQueue <- []byte{}: // 空数据表示结束
	case <-ctx.Done():
	case <-e.stopChan:
	}
}

// generateMockAudioData 生成模拟音频数据
func (e *ESpeakEngine) generateMockAudioData(word string) []byte {
	// 简单的模拟音频数据生成
	// 实际实现中应该调用espeak命令生成真实的音频数据
	
	// 根据单词长度生成不同长度的音频数据
	length := len(word) * 1000 // 每个字符1000字节
	if length < 2000 {
		length = 2000 // 最小2000字节
	}
	
	audioData := make([]byte, length)
	// 填充模拟的PCM数据（静音）
	for i := range audioData {
		audioData[i] = 0
	}
	
	return audioData
}

// synthesizeWithESpeak 使用ESpeak命令合成音频
func (e *ESpeakEngine) synthesizeWithESpeak(text string) ([]byte, error) {
	// 构建espeak命令
	args := []string{
		"-v", e.voice,
		"-s", fmt.Sprintf("%d", e.rate),
		"-p", fmt.Sprintf("%d", e.pitch),
		"-a", fmt.Sprintf("%d", e.volume),
		"--stdout",
		text,
	}

	cmd := exec.Command("espeak", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("espeak命令执行失败: %w", err)
	}

	return output, nil
}
