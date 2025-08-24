package realtimetts

import (
	"context"
	"time"
)

// AudioConfiguration 音频配置结构体
// 用来配置输出的流的属性
type AudioConfiguration struct {
	// 音频格式相关
	Format        AudioFormat // 音频格式 (WAV, MP3, etc.)
	Channels      int         // 声道数 (1=单声道, 2=立体声)
	SampleRate    int         // 采样率 (Hz)
	BitsPerSample int         // 每样本位数 (8, 16, 24, 32)

	// 设备相关
	OutputDeviceIndex int  // 输出设备索引
	Muted             bool // 是否静音

	// 缓冲相关
	FramesPerBuffer  int           // PyAudio缓冲区帧数
	PlayoutChunkSize int           // 播放块大小 (字节)
	BufferThreshold  time.Duration // 缓冲阈值 (秒)

	// 播放控制
	Volume        float64 // 音量 (0.0 - 1.0)
	PlaybackSpeed float64 // 播放速度 (1.0 = 正常速度)

	// 静音处理
	CommaSilenceDuration    time.Duration // 逗号后静音时长
	SentenceSilenceDuration time.Duration // 句子后静音时长
}

// AudioFormat 音频格式枚举
type AudioFormat int

const (
	FormatWAV AudioFormat = iota
	FormatMP3
	FormatMPEG
	FormatOGG
	FormatFLAC
)

// String 返回音频格式的字符串表示
func (f AudioFormat) String() string {
	switch f {
	case FormatWAV:
		return "WAV"
	case FormatMP3:
		return "MP3"
	case FormatMPEG:
		return "MPEG"
	case FormatOGG:
		return "OGG"
	case FormatFLAC:
		return "FLAC"
	default:
		return "Unknown"
	}
}

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
