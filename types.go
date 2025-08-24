package realtimetts

import (
	"context"
)

// TTSEngine TTS引擎接口
// 定义所有TTS引擎必须实现的接口
type TTSEngine interface {
	// GetStreamInfo 返回音频配置信息
	GetStreamInfo() *AudioConfiguration

	// Synthesize 执行文本到音频的合成
	Synthesize(ctx context.Context, text string) (<-chan []byte, error)

	// GetVoices 获取可用语音列表
	GetVoices() ([]Voice, error)

	// SetVoice 设置使用的语音
	SetVoice(voice Voice) error

	// SetVoiceParameters 设置语音参数
	SetVoiceParameters(params map[string]interface{}) error

	// SetAudioBuffer 设置音频缓冲管理器
	SetAudioBuffer(audioBuffer *AudioBuffer)
}
