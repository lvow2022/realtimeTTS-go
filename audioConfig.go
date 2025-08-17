package realtimetts

import (
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

// DefaultAudioConfig 返回默认音频配置
func DefaultAudioConfig() *AudioConfiguration {
	return &AudioConfiguration{
		Format:                  FormatWAV,
		Channels:                1,
		SampleRate:              16000,
		BitsPerSample:           16,
		OutputDeviceIndex:       0,
		Muted:                   false,
		FramesPerBuffer:         1024,
		PlayoutChunkSize:        4096,
		BufferThreshold:         2 * time.Second,
		Volume:                  1.0,
		PlaybackSpeed:           1.0,
		CommaSilenceDuration:    100 * time.Millisecond,
		SentenceSilenceDuration: 300 * time.Millisecond,
	}
}

// Validate 验证音频配置的有效性
func (c *AudioConfiguration) Validate() error {
	if c.Channels <= 0 || c.Channels > 8 {
		return ErrInvalidChannels
	}
	if c.SampleRate <= 0 {
		return ErrInvalidSampleRate
	}
	if c.BitsPerSample != 8 && c.BitsPerSample != 16 && c.BitsPerSample != 24 && c.BitsPerSample != 32 {
		return ErrInvalidBitsPerSample
	}
	if c.Volume < 0.0 || c.Volume > 1.0 {
		return ErrInvalidVolume
	}
	if c.PlaybackSpeed <= 0.0 {
		return ErrInvalidPlaybackSpeed
	}
	return nil
}

// GetBytesPerFrame 计算每帧的字节数
func (c *AudioConfiguration) GetBytesPerFrame() int {
	return c.Channels * (c.BitsPerSample / 8)
}

// GetBytesPerSecond 计算每秒的字节数
func (c *AudioConfiguration) GetBytesPerSecond() int {
	return c.SampleRate * c.GetBytesPerFrame()
}
