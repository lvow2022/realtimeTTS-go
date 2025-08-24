package realtimetts

import "errors"

// 音频配置相关错误
var (
	ErrInvalidChannels      = errors.New("无效的声道数")
	ErrInvalidSampleRate    = errors.New("无效的采样率")
	ErrInvalidBitsPerSample = errors.New("无效的每样本位数")
	ErrInvalidVolume        = errors.New("无效的音量值")
	ErrInvalidPlaybackSpeed = errors.New("无效的播放速度")
)

// 音频流相关错误
var (
	ErrStreamNotOpen       = errors.New("音频流未打开")
	ErrStreamAlreadyOpen   = errors.New("音频流已经打开")
	ErrStreamNotActive     = errors.New("音频流未激活")
	ErrStreamAlreadyActive = errors.New("音频流已经激活")
	ErrDeviceNotFound      = errors.New("音频设备未找到")
	ErrUnsupportedFormat   = errors.New("不支持的音频格式")
)

// 缓冲管理相关错误
var (
	ErrBufferEmpty   = errors.New("缓冲区为空")
	ErrBufferFull    = errors.New("缓冲区已满")
	ErrBufferTimeout = errors.New("缓冲区操作超时")
)

// 播放器相关错误
var (
	ErrPlayerNotInitialized = errors.New("播放器未初始化")
	ErrPlayerAlreadyPlaying = errors.New("播放器正在播放")
	ErrPlayerNotPlaying     = errors.New("播放器未在播放")
	ErrPlayerPaused         = errors.New("播放器已暂停")
)

// 引擎相关错误
var (
	ErrEngineNotInitialized  = errors.New("TTS引擎未初始化")
	ErrEngineSynthesisFailed = errors.New("TTS引擎合成失败")
	ErrNoEnginesAvailable    = errors.New("没有可用的TTS引擎")
	ErrInvalidPitch          = errors.New("无效的音调值")
)
