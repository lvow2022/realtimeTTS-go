package realtimetts

import (
	"time"
)

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
